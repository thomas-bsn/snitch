package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// Eligibility of a fingerprint (ADR 0064, #390). Snitch decides it from the measures it stores and a versioned
// rule, never once and for all when the fingerprint is stored: changing the rule requalifies every production
// fingerprint in the database, without any Worker job or audio, then re-runs matching.
//
// Two uses, one column each:
//   - usable (comparable): enough net speech to be matched on its own against a Reference. Matching, the
//     measured accuracy and the voice groups read only these. The column keeps its historical name so the
//     partial indexes `WHERE usable` (voice_fingerprints_model, HNSW per group) follow the rule.
//   - contributive: enough net speech to enter a Reference once validated by a human. Comparable implies
//     contributive; an overlapped or clipped fingerprint is neither.
//
// unusable_reason explains why a fingerprint is not comparable: too_short (contributive or not, see the column),
// overlap_only, clipped, or low_quality (decided before the rule). eligibility_rule_version names the rule that
// decided; null means "decided when stored": rows stored before the rule until a requalification, and benchmark
// or pending transition fingerprints (ADR 0044), which the rule never touches.

const lockVoiceEligibility = "snitch.voice_eligibility"

// fillNetSpeechSQL derives the net speech of production fingerprints stored without Worker measures, once:
// from the stored turns, never more than the fingerprint's duration (the speech the vector was computed on),
// or, without turns (a validated speaker archived by a later diarization), that duration alone, marked as
// estimated. $1 restricts it to the fingerprints of one job.
const fillNetSpeechSQL = `UPDATE voice_fingerprints f
SET net_speech_seconds=LEAST(g.net_speech_seconds,f.duration_seconds),overlap_ratio=g.overlap_ratio,overlapped=g.overlapped,
    net_speech_source=CASE WHEN g.net_speech_seconds IS NULL THEN 'estimated' ELSE 'turns' END
FROM voice_fingerprints x
LEFT JOIN LATERAL voice_turns_net_speech(x.local_speaker_id,x.turn_id) g ON x.scope='speaker' OR x.turn_id IS NOT NULL
WHERE x.id=f.id AND f.net_speech_source IS NULL AND f.measures_version IS NULL AND NOT f.benchmark AND ($1::bigint IS NULL OR f.fingerprint_job_id=$1)`

// eligibilityDecidedSQL is the single statement of the rule: the CTEs measured, reasoned and decided, which
// decide the production fingerprints matching filter (a condition on f) under each row of a preceding CTE named
// rule (version, comparable_min_seconds, contributive_min_seconds, max_clipped_ratio). The requalification
// (applyEligibilitySQL) and the preview (#394) both read them, so a preview at given durations gives the figures
// a requalification at those durations writes.
//
// measured reads the stored net speech; for a fingerprint whose net speech was never derived, it computes what
// fillNetSpeechSQL would write, so the preview needs no write. A requalification fills first: that branch is
// then empty and calls no function.
func eligibilityDecidedSQL(filter string) string {
	return eligibilityDecidedOverSQL(`NOT f.benchmark`, filter)
}

// eligibilityDecidedOverSQL is eligibilityDecidedSQL over another population of fingerprints: the fingerprint
// benchmark (#397) decides its own benchmark fingerprints with it, read-only, under the rule of its run.
func eligibilityDecidedOverSQL(population, filter string) string {
	return `measured AS (
  SELECT f.id,f.local_speaker_id,f.clip_id,f.model,f.model_version,f.dimensions,f.usable,f.contributive,f.unusable_reason,f.eligibility_rule_version,
    f.measures_version IS NULL AS original_vector,f.overlapped,f.overlap_ratio,f.clipped_ratio,COALESCE(f.net_speech_seconds,f.duration_seconds) AS net,f.net_speech_source AS net_source
  FROM voice_fingerprints f WHERE ` + population + ` AND NOT (f.net_speech_source IS NULL AND f.measures_version IS NULL) AND (` + filter + `)
  UNION ALL
  SELECT f.id,f.local_speaker_id,f.clip_id,f.model,f.model_version,f.dimensions,f.usable,f.contributive,f.unusable_reason,f.eligibility_rule_version,
    true,g.overlapped,g.overlap_ratio,f.clipped_ratio,LEAST(g.net_speech_seconds,f.duration_seconds),CASE WHEN g.net_speech_seconds IS NULL THEN 'estimated' ELSE 'turns' END
  FROM voice_fingerprints f
  LEFT JOIN LATERAL voice_turns_net_speech(f.local_speaker_id,f.turn_id) g ON f.scope='speaker' OR f.turn_id IS NOT NULL
  WHERE ` + population + ` AND f.net_speech_source IS NULL AND f.measures_version IS NULL AND (` + filter + `)
), reasoned AS (
  SELECT m.*,r.version,r.contributive_min_seconds,
    CASE WHEN m.overlapped THEN 'overlap_only'
         WHEN m.clipped_ratio > r.max_clipped_ratio THEN 'clipped'
         WHEN m.net < r.comparable_min_seconds THEN 'too_short'
         ELSE '' END AS reason
  FROM measured m, rule r
), decided AS (
  SELECT *,reason IN ('','too_short') AND net>=contributive_min_seconds AS now_contributive FROM reasoned
)`
}

// applyEligibilitySQL applies the active rule to production fingerprints ($1 restricts it to one job) and
// counts those whose eligibility changed. Rows already decided that way are not rewritten: applying the same
// rule twice writes nothing.
var applyEligibilitySQL = `WITH rule AS (
  SELECT version,comparable_min_seconds,contributive_min_seconds,max_clipped_ratio FROM voice_eligibility_rules ORDER BY version DESC LIMIT 1
), ` + eligibilityDecidedSQL(`$1::bigint IS NULL OR f.fingerprint_job_id=$1`) + `, changed AS (
  UPDATE voice_fingerprints f
  SET usable=d.reason='',unusable_reason=d.reason,contributive=d.now_contributive,eligibility_rule_version=d.version
  FROM decided d
  WHERE f.id=d.id AND (d.usable,d.contributive,d.unusable_reason,d.eligibility_rule_version) IS DISTINCT FROM (d.reason='',d.now_contributive,d.reason,d.version)
  RETURNING (d.usable,d.contributive,d.unusable_reason) IS DISTINCT FROM (f.usable,f.contributive,f.unusable_reason) AS moved
)
SELECT count(*) FILTER (WHERE moved) FROM changed`

// qualifyFingerprints decides the eligibility of production fingerprints under the active rule: those of one
// fingerprint job, or every one (jobID nil). The caller's transaction must hold the eligibility lock, shared
// (lockEligibilityShared) or exclusive (a rule change), so no fingerprint is decided by a replaced rule.
func qualifyFingerprints(ctx context.Context, tx pgx.Tx, jobID *int64) (int, error) {
	if _, err := tx.Exec(ctx, fillNetSpeechSQL, jobID); err != nil {
		return 0, err
	}
	var changed int
	err := tx.QueryRow(ctx, applyEligibilitySQL, jobID).Scan(&changed)
	return changed, err
}

// lockEligibilityShared lets fingerprints be decided concurrently while a rule change waits for them.
func lockEligibilityShared(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock_shared(hashtext($1))`, lockVoiceEligibility)
	return err
}

type eligibilityRule struct {
	Version                int       `json:"version"`
	ComparableMinSeconds   float64   `json:"comparable_min_seconds"`
	ContributiveMinSeconds float64   `json:"contributive_min_seconds"`
	MaxClippedRatio        float64   `json:"max_clipped_ratio"`
	Author                 *string   `json:"author"`
	Provisional            bool      `json:"provisional"`
	CreatedAt              time.Time `json:"created_at"`
}

type eligibilityValues struct {
	ComparableMinSeconds   float64 `json:"comparable_min_seconds"`
	ContributiveMinSeconds float64 `json:"contributive_min_seconds"`
	MaxClippedRatio        float64 `json:"max_clipped_ratio"`
}

// eligibilitySummary counts the production speaker fingerprints by eligibility. A fingerprint that is not
// contributive is counted under its reason.
type eligibilitySummary struct {
	Comparable       int `json:"comparable"`
	ContributiveOnly int `json:"contributive_only"`
	TooShort         int `json:"too_short"`
	OverlapOnly      int `json:"overlap_only"`
	Clipped          int `json:"clipped"`
	LowQuality       int `json:"low_quality"`
	Stale            int `json:"stale"`
}

type eligibilitySettings struct {
	Rule             eligibilityRule              `json:"rule"`
	History          []eligibilityRule            `json:"history"`
	Defaults         eligibilityValues            `json:"defaults"`
	Summary          eligibilitySummary           `json:"summary"`
	Requalifications []eligibilityRequalification `json:"requalifications"`
	Requalified      *int                         `json:"requalified,omitempty"`
}

// validEligibility bounds a rule: durations between 0.5 and 60 s, the contributive one at most the comparable
// one, a clipped share between 0 and 1.
func validEligibility(v eligibilityValues) bool {
	return v.ContributiveMinSeconds >= 0.5 && v.ContributiveMinSeconds <= v.ComparableMinSeconds && v.ComparableMinSeconds <= 60 &&
		v.MaxClippedRatio >= 0 && v.MaxClippedRatio <= 1
}

func (s *Server) registerVoiceEligibilityRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/voice-eligibility", s.protected(true, http.HandlerFunc(s.getVoiceEligibility)))
	mux.Handle("PUT /api/admin/voice-eligibility", s.protected(true, http.HandlerFunc(s.putVoiceEligibility)))
	mux.Handle("GET /api/admin/voice-eligibility/preview", s.protected(true, http.HandlerFunc(s.getVoiceEligibilityPreview)))
}

func (s *Server) loadEligibility(ctx context.Context) (eligibilitySettings, error) {
	var out eligibilitySettings
	rows, err := s.Pool.Query(ctx, `SELECT version,comparable_min_seconds,contributive_min_seconds,max_clipped_ratio,author_username,author_id IS NULL,created_at FROM voice_eligibility_rules ORDER BY version DESC LIMIT 20`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.History = []eligibilityRule{}
	for rows.Next() {
		var r eligibilityRule
		if err = rows.Scan(&r.Version, &r.ComparableMinSeconds, &r.ContributiveMinSeconds, &r.MaxClippedRatio, &r.Author, &r.Provisional, &r.CreatedAt); err != nil {
			return out, err
		}
		out.History = append(out.History, r)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.History) == 0 {
		return out, errors.New("no voice eligibility rule")
	}
	out.Rule = out.History[0]
	if err = s.Pool.QueryRow(ctx, `SELECT comparable_min_seconds,contributive_min_seconds,max_clipped_ratio FROM voice_eligibility_rules WHERE version=1`).
		Scan(&out.Defaults.ComparableMinSeconds, &out.Defaults.ContributiveMinSeconds, &out.Defaults.MaxClippedRatio); err != nil {
		return out, err
	}
	sum := &out.Summary
	err = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE usable),
  count(*) FILTER (WHERE contributive AND NOT usable),
  count(*) FILTER (WHERE NOT contributive AND unusable_reason='too_short'),
  count(*) FILTER (WHERE NOT contributive AND unusable_reason='overlap_only'),
  count(*) FILTER (WHERE NOT contributive AND unusable_reason='clipped'),
  count(*) FILTER (WHERE NOT contributive AND unusable_reason='low_quality'),
  count(*) FILTER (WHERE eligibility_rule_version IS DISTINCT FROM $1)
FROM voice_fingerprints WHERE NOT benchmark AND scope='speaker'`, out.Rule.Version).
		Scan(&sum.Comparable, &sum.ContributiveOnly, &sum.TooShort, &sum.OverlapOnly, &sum.Clipped, &sum.LowQuality, &sum.Stale)
	if err != nil {
		return out, err
	}
	out.Requalifications, err = loadRequalifications(ctx, s.Pool)
	return out, err
}

func (s *Server) getVoiceEligibility(w http.ResponseWriter, r *http.Request) {
	out, err := s.loadEligibility(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}

// putVoiceEligibility records a new version of the rule (nothing when unchanged), requalifies every production
// fingerprint under the active rule, traces the requalification (author, rule, before and after, #394), then
// re-runs matching when an eligibility changed. Requalifying under an unchanged rule writes no fingerprint again.
func (s *Server) putVoiceEligibility(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ComparableMinSeconds   *float64 `json:"comparable_min_seconds"`
		ContributiveMinSeconds *float64 `json:"contributive_min_seconds"`
		MaxClippedRatio        *float64 `json:"max_clipped_ratio"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.ComparableMinSeconds == nil || input.ContributiveMinSeconds == nil || input.MaxClippedRatio == nil ||
		!validEligibility(eligibilityValues{*input.ComparableMinSeconds, *input.ContributiveMinSeconds, *input.MaxClippedRatio}) {
		fail(w, http.StatusBadRequest, "invalid_voice_eligibility")
		return
	}
	ctx := context.WithoutCancel(r.Context())
	author := s.authorOf(r)
	tx, err := s.beginLocked(ctx, lockVoiceEligibility)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var version int
	var current eligibilityValues
	if err = tx.QueryRow(ctx, `SELECT version,comparable_min_seconds,contributive_min_seconds,max_clipped_ratio FROM voice_eligibility_rules ORDER BY version DESC LIMIT 1`).
		Scan(&version, &current.ComparableMinSeconds, &current.ContributiveMinSeconds, &current.MaxClippedRatio); err != nil {
		s.internal(w, err)
		return
	}
	wanted := eligibilityValues{*input.ComparableMinSeconds, *input.ContributiveMinSeconds, *input.MaxClippedRatio}
	report := eligibilityRequalification{RuleVersion: version}
	if report.Before, err = storedEligibility(ctx, tx, nil); err != nil {
		s.internal(w, err)
		return
	}
	if wanted != current {
		report.RuleVersion = version + 1
		if _, err = tx.Exec(ctx, `INSERT INTO voice_eligibility_rules(version,comparable_min_seconds,contributive_min_seconds,max_clipped_ratio,author_id,author_username) VALUES($1,$2,$3,$4,$5,$6)`,
			version+1, wanted.ComparableMinSeconds, wanted.ContributiveMinSeconds, wanted.MaxClippedRatio, author.ID, author.Username); err != nil {
			s.internal(w, err)
			return
		}
	}
	changed, err := qualifyFingerprints(ctx, tx, nil)
	if err != nil {
		s.internal(w, err)
		return
	}
	report.Changed = changed
	if report.After, err = storedEligibility(ctx, tx, &report); err != nil {
		s.internal(w, err)
		return
	}
	if err = traceRequalification(ctx, tx, report, author); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	if changed > 0 {
		if _, err = s.rematchVoices(ctx, 0); err != nil {
			s.internal(w, err)
			return
		}
	}
	out, err := s.loadEligibility(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	out.Requalified = &changed
	reply(w, http.StatusOK, out)
}
