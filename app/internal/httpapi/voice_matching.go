package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"snitch/internal/logging"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// Voice matching (phase 5). Matching compares the usable fingerprints of
// local speakers with the references of every Person, entirely in PostgreSQL: no worker
// and no media. References are the active validated voice samples and nothing else;
// matching only writes machine attributions (origin "matching") and their candidates,
// never a voice sample, so an automatic attribution can never feed its own references.
//
// Fingerprints are only compared inside one (model, model_version, dimensions) group,
// and only when they are usable (long and clean enough, decided when they were stored).
// A group without any reference is not evaluated: the machine attributions of its
// speakers are removed instead of kept stale.
//
// Decision for one speaker, from the cosine similarity to each Person (best reference
// of the Person):
//   - below suggestion_threshold for every Person: unknown;
//   - two or more Persons within ambiguity_margin of the best: ambiguous, with those
//     candidates (two close voices are never merged into one identity);
//   - best at or above automatic_threshold: automatic when auto_enabled, otherwise a
//     suggestion;
//   - otherwise: a suggestion.
//
// The automation switch, Person corrections and voice profile deletion reuse rematchVoices (after
// merging or deleting a Person) and extend the writers of machine attributions through
// the origin column.

const matchingOrigin = "matching"

type matchingSettings struct {
	AutoEnabled         bool       `json:"auto_enabled"`
	AutomaticThreshold  float64    `json:"automatic_threshold"`
	SuggestionThreshold float64    `json:"suggestion_threshold"`
	AmbiguityMargin     float64    `json:"ambiguity_margin"`
	MaxCandidates       int        `json:"max_candidates"`
	PrecisionThreshold  float64    `json:"precision_threshold"`
	MinConfirmations    int        `json:"min_confirmations"`
	AutoChangedAt       *time.Time `json:"auto_changed_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type voiceDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Closed reasons a voice stays unknown or ambiguous in the per-clip matching log.
const (
	reasonNoReference      = "no_reference"
	reasonTiedCandidates   = "tied_candidates"
	reasonBelowThreshold   = "below_threshold"
	reasonMicroTrackDirect = "micro_track_direct"
)

func loadMatchingSettings(ctx context.Context, db voiceDB) (matchingSettings, error) {
	var v matchingSettings
	err := db.QueryRow(ctx, `SELECT auto_enabled,automatic_threshold,suggestion_threshold,ambiguity_margin,max_candidates,precision_threshold,min_confirmations,auto_changed_at,updated_at FROM voice_matching_settings WHERE id=true`).Scan(&v.AutoEnabled, &v.AutomaticThreshold, &v.SuggestionThreshold, &v.AmbiguityMargin, &v.MaxCandidates, &v.PrecisionThreshold, &v.MinConfirmations, &v.AutoChangedAt, &v.UpdatedAt)
	return v, err
}

// registerVoiceMatchingRoutes declares the matching routes. Reading the settings is open
// to every signed-in user (they must see whether the automation is active); changing
// the thresholds and forcing a rematch are administrator actions.
func (s *Server) registerVoiceMatchingRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/voice-matching", s.protected(false, http.HandlerFunc(s.getVoiceMatching)))
	mux.Handle("PUT /api/admin/voice-matching", s.protected(true, http.HandlerFunc(s.putVoiceMatching)))
	mux.Handle("POST /api/admin/voice-matching/rematch", s.protected(true, http.HandlerFunc(s.postVoiceRematch)))
}

func (s *Server) getVoiceMatching(w http.ResponseWriter, r *http.Request) {
	v, err := loadMatchingSettings(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

// putVoiceMatching changes the thresholds. The automation switch is deliberately not
// part of this input: its activation is a separate administrator decision (automation switch).
func (s *Server) putVoiceMatching(w http.ResponseWriter, r *http.Request) {
	var input struct {
		AutomaticThreshold  *float64 `json:"automatic_threshold"`
		SuggestionThreshold *float64 `json:"suggestion_threshold"`
		AmbiguityMargin     *float64 `json:"ambiguity_margin"`
		MaxCandidates       *int     `json:"max_candidates"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.AutomaticThreshold == nil || input.SuggestionThreshold == nil || input.AmbiguityMargin == nil || input.MaxCandidates == nil ||
		*input.SuggestionThreshold < 0 || *input.SuggestionThreshold > *input.AutomaticThreshold || *input.AutomaticThreshold > 1 ||
		*input.AmbiguityMargin < 0 || *input.AmbiguityMargin > 1 || *input.MaxCandidates < 2 || *input.MaxCandidates > 20 {
		fail(w, http.StatusBadRequest, "invalid_voice_matching")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	if _, err := s.Pool.Exec(r.Context(), `UPDATE voice_matching_settings SET automatic_threshold=$1,suggestion_threshold=$2,ambiguity_margin=$3,max_candidates=$4,updated_by=$5,updated_at=now() WHERE id=true`, *input.AutomaticThreshold, *input.SuggestionThreshold, *input.AmbiguityMargin, *input.MaxCandidates, ss.User.ID); err != nil {
		s.internal(w, err)
		return
	}
	// New thresholds re-examine every stored fingerprint, never the videos.
	if _, err := s.rematchVoices(context.WithoutCancel(r.Context()), 0); err != nil {
		s.internal(w, err)
		return
	}
	v, err := loadMatchingSettings(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

func (s *Server) postVoiceRematch(w http.ResponseWriter, r *http.Request) {
	stats, err := s.rematchVoices(context.WithoutCancel(r.Context()), 0)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, stats)
}

// matchingChanged re-runs matching after a change of references or fingerprints. The
// triggering change is already committed: a failure is logged and never undoes it, and
// the next rematch (or the administrator's explicit one) repairs the attributions.
func (s *Server) matchingChanged(ctx context.Context, clipID int64) {
	if _, err := s.rematchVoices(context.WithoutCancel(ctx), clipID); err != nil {
		s.Log.Error("voice matching failed", append([]any{"clip_id", clipID}, logging.Failure(err)...)...)
	}
}

type matchStats struct {
	Evaluated  int `json:"evaluated"`
	Automatic  int `json:"automatic"`
	Suggestion int `json:"suggestion"`
	Ambiguous  int `json:"ambiguous"`
	Unknown    int `json:"unknown"`
	Removed    int `json:"removed"`
}

type matchScore struct {
	personID int64
	score    float64
}

type matchTarget struct {
	speakerID, clipID int64
	scores            []matchScore // best first, already at or above the suggestion threshold
	model, version    string
	noReference       bool // fingerprint group without any reference: nothing to compare with
}

type existingMatch struct {
	attributionID int64
	personID      *int64
	level         DecisionLevel
	confidence    *float64
	signature     string
}

type matchGroup struct {
	model, version string
	dimensions     int
}

var voiceMatchingLock = sync.Mutex{}

// rematchVoices recomputes the machine attributions of local speakers. clipID 0 covers
// the whole library; otherwise only the speakers of that clip (new fingerprints).
// Runs are serialized (process mutex and database advisory lock) so concurrent changes
// of references cannot interleave their writes; each run reads the committed state.
func (s *Server) rematchVoices(ctx context.Context, clipID int64) (matchStats, error) {
	voiceMatchingLock.Lock()
	defer voiceMatchingLock.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var stats matchStats
	tx, err := s.beginLocked(ctx, lockVoiceMatching)
	if err != nil {
		return stats, err
	}
	defer tx.Rollback(ctx)
	settings, err := loadMatchingSettings(ctx, tx)
	if err != nil {
		return stats, err
	}
	var scope *int64
	if clipID > 0 {
		scope = &clipID
	}
	groupRows, err := tx.Query(ctx, `SELECT DISTINCT model,model_version,dimensions FROM voice_fingerprints WHERE usable AND scope='speaker' AND ($1::bigint IS NULL OR clip_id=$1)`, scope)
	if err != nil {
		return stats, err
	}
	var groups []matchGroup
	for groupRows.Next() {
		var g matchGroup
		if err = groupRows.Scan(&g.model, &g.version, &g.dimensions); err != nil {
			groupRows.Close()
			return stats, err
		}
		groups = append(groups, g)
	}
	groupRows.Close()
	if err = groupRows.Err(); err != nil {
		return stats, err
	}
	var targets []matchTarget
	for _, g := range groups {
		found, groupErr := s.matchGroupScores(ctx, tx, g, settings, scope)
		if groupErr != nil {
			return stats, groupErr
		}
		targets = append(targets, found...)
	}
	existing, err := loadExistingMatches(ctx, tx, scope)
	if err != nil {
		return stats, err
	}
	names, err := personNames(ctx, tx, targets, existing)
	if err != nil {
		return stats, err
	}
	evaluated := make(map[int64]bool, len(targets))
	// Per clip, why voices stay unknown or ambiguous: closed reasons and counts only.
	reasons := map[int64]map[string]int{}
	note := func(clip int64, reason string) {
		if reasons[clip] == nil {
			reasons[clip] = map[string]int{}
		}
		reasons[clip][reason]++
	}
	for _, t := range targets {
		if t.noReference {
			note(t.clipID, reasonNoReference)
			continue
		}
		evaluated[t.speakerID] = true
		level, person, confidence, candidates := decideMatch(t.scores, settings)
		stats.Evaluated++
		switch level {
		case LevelAutomatic:
			stats.Automatic++
		case LevelSuggestion:
			stats.Suggestion++
		case LevelAmbiguous:
			stats.Ambiguous++
		default:
			stats.Unknown++
		}
		switch level {
		case LevelAmbiguous:
			note(t.clipID, reasonTiedCandidates)
		case LevelUnknown:
			note(t.clipID, reasonBelowThreshold)
		}
		if err = writeMatch(ctx, tx, t, level, person, confidence, candidates, existing[t.speakerID], names); err != nil {
			return stats, err
		}
	}
	// Machine rows of speakers that are no longer evaluated (validated since, unusable
	// fingerprint, no reference left for their model) must not linger.
	for speakerID, old := range existing {
		if evaluated[speakerID] {
			continue
		}
		if err = removeMatch(ctx, tx, speakerID, old, names); err != nil {
			return stats, err
		}
		stats.Removed++
	}
	summaries, err := clipMatchSummaries(ctx, tx, reasons, scope)
	if err != nil {
		return stats, err
	}
	if err = tx.Commit(ctx); err != nil {
		return stats, err
	}
	s.logClipSummaries(summaries, scope != nil)
	for _, g := range groups {
		s.ensureVoiceIndex(ctx, g)
	}
	// Proposed proximity between unknown voices follows the references and thresholds.
	if err = s.refreshVoiceGroupLinks(ctx); err != nil {
		s.Log.Error("voice group links failed", logging.Failure(err)...)
	}
	// The voices that still need a human decision are announced (or the action resolved).
	if clipID > 0 {
		s.syncSpeakerReview(ctx, clipID)
	} else {
		s.syncAllSpeakerReviews(ctx)
	}
	return stats, nil
}

// matchGroupScores returns every evaluable speaker of one fingerprint group with its
// per-Person best similarity at or above the suggestion threshold. The cast to a typed
// vector is written into the statement (the dimension is a validated integer), and is
// only evaluated on rows already restricted to that dimension.
func (s *Server) matchGroupScores(ctx context.Context, tx pgx.Tx, g matchGroup, settings matchingSettings, scope *int64) ([]matchTarget, error) {
	query := fmt.Sprintf(`WITH refs AS (
  SELECT vs.person_id,f.embedding::vector(%[1]d) AS e
  FROM voice_samples vs JOIN voice_fingerprints f ON f.local_speaker_id=vs.local_speaker_id
  WHERE vs.retired_at IS NULL AND NOT EXISTS (SELECT 1 FROM persons op WHERE op.id=vs.person_id AND op.voice_profile_deleted_at IS NOT NULL) AND f.usable AND f.scope='speaker' AND f.model=$1 AND f.model_version=$2 AND f.dimensions=%[1]d
), targets AS (
  SELECT f.local_speaker_id AS sid,f.clip_id,f.embedding::vector(%[1]d) AS e
  FROM voice_fingerprints f
  WHERE f.usable AND f.scope='speaker' AND f.model=$1 AND f.model_version=$2 AND f.dimensions=%[1]d AND ($3::bigint IS NULL OR f.clip_id=$3)
    AND NOT EXISTS (SELECT 1 FROM local_speakers ls WHERE ls.id=f.local_speaker_id AND ls.kind='microphone')
    AND NOT EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=f.local_speaker_id AND (a.validation_state='validated' OR a.origin<>'%[2]s'))
    AND NOT EXISTS (SELECT 1 FROM local_speaker_dispositions d WHERE d.local_speaker_id=f.local_speaker_id)
), scored AS (
  SELECT t.sid,r.person_id,max(1-(t.e <=> r.e)) AS score FROM targets t CROSS JOIN refs r
  WHERE NOT EXISTS (SELECT 1 FROM voice_refusals x WHERE x.local_speaker_id=t.sid AND x.person_id=r.person_id)
  GROUP BY t.sid,r.person_id HAVING max(1-(t.e <=> r.e)) >= $4
)
SELECT t.sid,t.clip_id,s.person_id,s.score,NOT EXISTS (SELECT 1 FROM refs) FROM targets t LEFT JOIN scored s ON s.sid=t.sid
ORDER BY t.sid,s.score DESC NULLS LAST,s.person_id`, g.dimensions, matchingOrigin)
	rows, err := tx.Query(ctx, query, g.model, g.version, scope, settings.SuggestionThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []matchTarget
	for rows.Next() {
		var sid, clip int64
		var person *int64
		var score *float64
		var noReference bool
		if err = rows.Scan(&sid, &clip, &person, &score, &noReference); err != nil {
			return nil, err
		}
		if len(out) == 0 || out[len(out)-1].speakerID != sid {
			out = append(out, matchTarget{speakerID: sid, clipID: clip, model: g.model, version: g.version, noReference: noReference})
		}
		if person != nil && score != nil {
			last := &out[len(out)-1]
			last.scores = append(last.scores, matchScore{personID: *person, score: *score})
		}
	}
	return out, rows.Err()
}

// decideMatch turns the per-Person scores of one speaker into a decision level.
func decideMatch(scores []matchScore, settings matchingSettings) (level DecisionLevel, person *int64, confidence *float64, candidates []matchScore) {
	if len(scores) == 0 {
		return LevelUnknown, nil, nil, nil
	}
	best := scores[0]
	conf := best.score
	if conf > 1 {
		conf = 1
	}
	var close []matchScore
	for _, c := range scores {
		if best.score-c.score <= settings.AmbiguityMargin {
			close = append(close, c)
		}
	}
	if len(close) >= 2 {
		if len(close) > settings.MaxCandidates {
			close = close[:settings.MaxCandidates]
		}
		return LevelAmbiguous, nil, &conf, close
	}
	alternatives := scores
	if len(alternatives) > settings.MaxCandidates {
		alternatives = alternatives[:settings.MaxCandidates]
	}
	level = LevelSuggestion
	if settings.AutoEnabled && best.score >= settings.AutomaticThreshold {
		level = LevelAutomatic
	}
	id := best.personID
	return level, &id, &conf, alternatives
}

func matchSignature(candidates []matchScore) string {
	parts := make([]string, len(candidates))
	for i, c := range candidates {
		parts[i] = fmt.Sprintf("%d:%.6f", c.personID, c.score)
	}
	return strings.Join(parts, ",")
}

func loadExistingMatches(ctx context.Context, tx pgx.Tx, scope *int64) (map[int64]existingMatch, error) {
	rows, err := tx.Query(ctx, `SELECT a.local_speaker_id,a.id,a.person_id,a.decision_level,a.confidence,COALESCE((SELECT string_agg(c.person_id||':'||round(c.score::numeric,6)::text,',' ORDER BY c.rank) FROM speaker_attribution_candidates c WHERE c.attribution_id=a.id),'')
FROM speaker_attributions a WHERE a.validation_state='unvalidated' AND a.origin=$2 AND ($1::bigint IS NULL OR a.local_speaker_id IN (SELECT id FROM local_speakers WHERE clip_id=$1))`, scope, matchingOrigin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]existingMatch{}
	for rows.Next() {
		var speakerID int64
		var e existingMatch
		if err = rows.Scan(&speakerID, &e.attributionID, &e.personID, &e.level, &e.confidence, &e.signature); err != nil {
			return nil, err
		}
		out[speakerID] = e
	}
	return out, rows.Err()
}

func personNames(ctx context.Context, tx pgx.Tx, targets []matchTarget, existing map[int64]existingMatch) (map[int64]string, error) {
	ids := map[int64]bool{}
	for _, t := range targets {
		for _, c := range t.scores {
			ids[c.personID] = true
		}
	}
	for _, e := range existing {
		if e.personID != nil {
			ids[*e.personID] = true
		}
	}
	list := make([]int64, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
	names := map[int64]string{}
	rows, err := tx.Query(ctx, `SELECT id,display_name FROM persons WHERE id=ANY($1)`, list)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err = rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}

func eventRow(ctx context.Context, tx pgx.Tx, speakerID, clipID int64, oldPerson *int64, oldLevel DecisionLevel, newPerson *int64, newLevel DecisionLevel, names map[int64]string) error {
	name := func(id *int64) string {
		if id == nil {
			return ""
		}
		return names[*id]
	}
	_, err := tx.Exec(ctx, `INSERT INTO attribution_events(local_speaker_id,clip_id,action,old_person_id,old_person_name,old_decision_level,new_person_id,new_person_name,new_decision_level) VALUES($1,$2,'match',$3,$4,$5,$6,$7,$8)`, speakerID, clipID, oldPerson, name(oldPerson), oldLevel, newPerson, name(newPerson), newLevel)
	return err
}

// writeMatch stores the decision of one speaker, and a history event when the Person
// or the decision level changed (a plain confidence drift is not an event).
func writeMatch(ctx context.Context, tx pgx.Tx, t matchTarget, level DecisionLevel, person *int64, confidence *float64, candidates []matchScore, old existingMatch, names map[int64]string) error {
	signature := matchSignature(candidates)
	hadOld := old.attributionID != 0
	samePerson := hadOld && ((old.personID == nil) == (person == nil)) && (person == nil || *old.personID == *person)
	if hadOld && samePerson && old.level == level && old.signature == signature && ((old.confidence == nil) == (confidence == nil)) {
		return nil
	}
	var id int64
	err := tx.QueryRow(ctx, `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,confidence,origin,match_model,match_model_version)
SELECT $1,$2,$3,'unvalidated',$4,$7,$5,$6 WHERE NOT EXISTS (SELECT 1 FROM speaker_attributions v WHERE v.local_speaker_id=$1 AND v.validation_state='validated')
ON CONFLICT (local_speaker_id) WHERE validation_state='unvalidated' DO UPDATE SET person_id=EXCLUDED.person_id,decision_level=EXCLUDED.decision_level,confidence=EXCLUDED.confidence,match_model=EXCLUDED.match_model,match_model_version=EXCLUDED.match_model_version,
 revision=CASE WHEN speaker_attributions.person_id IS NOT DISTINCT FROM EXCLUDED.person_id AND speaker_attributions.decision_level=EXCLUDED.decision_level THEN speaker_attributions.revision ELSE speaker_attributions.revision+1 END,decided_at=now()
RETURNING id`, t.speakerID, person, level, confidence, t.model, t.version, matchingOrigin).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // validated by a human in the meantime: the human decision stands
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM speaker_attribution_candidates WHERE attribution_id=$1`, id); err != nil {
		return err
	}
	for rank, c := range candidates {
		if _, err = tx.Exec(ctx, `INSERT INTO speaker_attribution_candidates(attribution_id,person_id,rank,score) VALUES($1,$2,$3,$4)`, id, c.personID, rank+1, c.score); err != nil {
			return err
		}
	}
	if hadOld && samePerson && old.level == level {
		return nil
	}
	oldLevel := LevelUnknown
	if hadOld {
		oldLevel = old.level
	}
	if !hadOld && level == LevelUnknown {
		return nil // first evaluation with no match: nothing changed for readers
	}
	return eventRow(ctx, tx, t.speakerID, t.clipID, old.personID, oldLevel, person, level, names)
}

func removeMatch(ctx context.Context, tx pgx.Tx, speakerID int64, old existingMatch, names map[int64]string) error {
	var clipID int64
	if err := tx.QueryRow(ctx, `SELECT clip_id FROM local_speakers WHERE id=$1`, speakerID).Scan(&clipID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM speaker_attributions WHERE id=$1 AND validation_state='unvalidated' AND origin=$2`, old.attributionID, matchingOrigin); err != nil {
		return err
	}
	if old.level == LevelUnknown {
		return nil
	}
	return eventRow(ctx, tx, speakerID, clipID, old.personID, old.level, nil, LevelUnknown, names)
}

const (
	lockVoiceMatching = "snitch.voice_matching"
	lockVoiceGroups   = "snitch.voice_groups"
)

// beginLocked opens a transaction holding the named transaction-level advisory lock.
func (s *Server) beginLocked(ctx context.Context, lockName string) (pgx.Tx, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockName); err != nil {
		tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

var voiceIndexes sync.Map

// ensureVoiceIndex creates the HNSW index of one fingerprint group: vectors of
// different models, versions or dimensions never share an index. The matching above
// compares against the (small) reference set exactly; the index serves the nearest
// neighbor searches over all usable fingerprints (groups of unknown voices, unknown voice groups).
func (s *Server) ensureVoiceIndex(ctx context.Context, g matchGroup) {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", g.model, g.version, g.dimensions)))
	name := "voice_fingerprints_hnsw_" + hex.EncodeToString(sum[:6])
	if _, done := voiceIndexes.Load(name); done {
		return
	}
	quote := func(v string) string { return "'" + strings.ReplaceAll(v, "'", "''") + "'" }
	statement := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON voice_fingerprints USING hnsw ((embedding::vector(%d)) vector_cosine_ops) WHERE usable AND scope='speaker' AND model=%s AND model_version=%s AND dimensions=%d`, name, g.dimensions, quote(g.model), quote(g.version), g.dimensions)
	if _, err := s.Pool.Exec(ctx, statement); err != nil {
		s.Log.Error("voice index creation failed", append([]any{"index", name}, logging.Failure(err)...)...)
		return
	}
	voiceIndexes.Store(name, true)
}

type clipMatchSummary struct {
	clipID  int64
	levels  map[string]int
	reasons map[string]int
}

// clipMatchSummaries counts, per clip touched by this run, the voices by decision level and the
// closed reasons of the unknown and ambiguous ones. Only counts: no Person, fingerprint or score.
func clipMatchSummaries(ctx context.Context, tx pgx.Tx, reasons map[int64]map[string]int, scope *int64) ([]clipMatchSummary, error) {
	clips := map[int64]bool{}
	for clip := range reasons {
		clips[clip] = true
	}
	if scope != nil {
		clips[*scope] = true
	}
	var out []clipMatchSummary
	for clip := range clips {
		rows, err := tx.Query(ctx, `SELECT a.decision_level,a.origin=$2,count(*) FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 GROUP BY 1,2`, clip, microphoneOrigin)
		if err != nil {
			return nil, err
		}
		summary := clipMatchSummary{clipID: clip, levels: map[string]int{}, reasons: map[string]int{}}
		for reason, n := range reasons[clip] {
			summary.reasons[reason] = n
		}
		for rows.Next() {
			var level string
			var direct bool
			var n int
			if err = rows.Scan(&level, &direct, &n); err != nil {
				rows.Close()
				return nil, err
			}
			summary.levels[level] += n
			if direct {
				summary.reasons[reasonMicroTrackDirect] += n
			}
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return nil, err
		}
		// A voice without reference is unknown but holds no attribution row.
		summary.levels[string(LevelUnknown)] += reasons[clip][reasonNoReference]
		out = append(out, summary)
	}
	return out, nil
}

// logClipSummaries writes one line per clip: INFO for a clip-scoped run, DEBUG for a library-wide one.
func (s *Server) logClipSummaries(summaries []clipMatchSummary, scoped bool) {
	level := slog.LevelDebug
	if scoped {
		level = slog.LevelInfo
	}
	for _, summary := range summaries {
		s.Log.Log(context.Background(), level, "voice_matching_summary", "clip_id", summary.clipID, "levels", summary.levels, "reasons", summary.reasons)
	}
}
