package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"snitch/internal/analysis"
	"snitch/internal/logging"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// Voice matching (phase 5, ADR 0064). Matching compares the comparable fingerprints of
// local speakers with the Reference of every Person, from the stored vectors: no worker
// and no media. References are the active validated voice samples and nothing else;
// matching only writes machine attributions (origin "matching") and their candidates,
// never a voice sample, so an automatic attribution can never feed its own references.
//
// Fingerprints are only compared inside one (model, model_version, dimensions) group,
// and only when they are comparable (column usable, decided by the eligibility rule).
// A group without any reference is not evaluated: the machine attributions of its
// speakers are removed instead of kept stale.
//
// Scores come from the one score function of Snitch (voice_score.go): the cosine with the cumulative
// Reference of each Person (contributive validated samples weighted by net speech), or with her best comparable
// sample when the administrator chose that score mode. References are derived from the samples at every run,
// never stored: any change of samples (validation, correction, merge, separation, voice profile or clip
// deletion) is seen by the next run.
//
// Decision for one speaker, from its score against each Person:
//   - below suggestion_threshold for every Person: unknown;
//   - two or more Persons within ambiguity_margin of the best: ambiguous, with those
//     candidates (two close voices are never merged into one identity);
//   - best at or above automatic_threshold: automatic when auto_enabled, otherwise a
//     suggestion;
//   - otherwise: a suggestion.
//
// A change of references asks for a background pass (matchingChanged, voice_matching_queue.go, #392); the
// analysis step matches its clip at once (rematchVoices). Writers of machine attributions are told apart
// through the origin column.

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
	// ScoreMode is the active score mode (voice_score_modes), read with the thresholds so one run uses one
	// consistent setting. Exposed by its own administrator route, not with the thresholds.
	ScoreMode scoreMode `json:"-"`
	// Group is the active fingerprint group whose thresholds these are (effectiveMatchingSettings); nil when
	// there is none, the thresholds then being the provisional ones.
	Group *fingerprintGroupView `json:"group"`
	// ThresholdsApplied tells whether the group has thresholds applied by an administrator. Without them it only
	// produces suggestions (voice_group_thresholds.go).
	ThresholdsApplied bool `json:"thresholds_applied"`
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
	err := db.QueryRow(ctx, `SELECT auto_enabled,automatic_threshold,suggestion_threshold,ambiguity_margin,max_candidates,precision_threshold,min_confirmations,auto_changed_at,updated_at,
  (SELECT mode FROM voice_score_modes ORDER BY version DESC LIMIT 1) FROM voice_matching_settings WHERE id=true`).Scan(&v.AutoEnabled, &v.AutomaticThreshold, &v.SuggestionThreshold, &v.AmbiguityMargin, &v.MaxCandidates, &v.PrecisionThreshold, &v.MinConfirmations, &v.AutoChangedAt, &v.UpdatedAt, &v.ScoreMode)
	return v, err
}

// registerVoiceMatchingRoutes declares the matching routes. Reading the settings is open
// to every signed-in user (they must see whether the automation is active); changing
// the thresholds and forcing a rematch are administrator actions.
func (s *Server) registerVoiceMatchingRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/voice-matching", s.protected(false, http.HandlerFunc(s.getVoiceMatching)))
	mux.Handle("GET /api/voice-matching/status", s.protected(false, http.HandlerFunc(s.getVoiceMatchingStatus)))
	mux.Handle("GET /api/voice-matching/status/events", s.protected(false, http.HandlerFunc(s.voiceMatchingStatusEvents)))
	mux.Handle("PUT /api/admin/voice-matching", s.protected(true, http.HandlerFunc(s.putVoiceMatching)))
	mux.Handle("POST /api/admin/voice-matching/rematch", s.protected(true, http.HandlerFunc(s.postVoiceRematch)))
}

func (s *Server) getVoiceMatching(w http.ResponseWriter, r *http.Request) {
	v, err := effectiveMatchingSettings(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

// putVoiceMatching applies thresholds: to the given fingerprint group, or to the active one, as a new traced
// version of its thresholds (voice_group_thresholds.go). Without any group (blank installation) they are the
// provisional values a group uses for its suggestions until thresholds are applied to it. The number of
// candidates is global. The automation switch is deliberately not part of this input: its activation is a
// separate administrator decision (automation switch).
func (s *Server) putVoiceMatching(w http.ResponseWriter, r *http.Request) {
	var input struct {
		AutomaticThreshold  *float64              `json:"automatic_threshold"`
		SuggestionThreshold *float64              `json:"suggestion_threshold"`
		AmbiguityMargin     *float64              `json:"ambiguity_margin"`
		MaxCandidates       *int                  `json:"max_candidates"`
		Group               *fingerprintGroupView `json:"group"`
		Source              *string               `json:"source"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.AutomaticThreshold == nil || input.SuggestionThreshold == nil || input.AmbiguityMargin == nil || input.MaxCandidates == nil ||
		*input.SuggestionThreshold < 0 || *input.SuggestionThreshold > *input.AutomaticThreshold || *input.AutomaticThreshold > 1 ||
		*input.AmbiguityMargin < 0 || *input.AmbiguityMargin > 1 || *input.MaxCandidates < 2 || *input.MaxCandidates > 20 ||
		(input.Source != nil && *input.Source != thresholdsFromAdministrator && *input.Source != thresholdsFromProposal) ||
		(input.Group != nil && (input.Group.Model == "" || input.Group.ModelVersion == "" || input.Group.Dimensions < 1)) {
		fail(w, http.StatusBadRequest, "invalid_voice_matching")
		return
	}
	ctx := context.WithoutCancel(r.Context())
	author := s.authorOf(r)
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var group matchGroup
	found := false
	if input.Group != nil {
		group = input.Group.group()
		if found, err = productionGroupExists(ctx, tx, group); err == nil && !found {
			fail(w, http.StatusConflict, "voice_group_unknown")
			return
		}
	} else {
		group, found, err = activeFingerprintGroup(ctx, tx)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if found {
		source := thresholdsFromAdministrator
		if input.Source != nil {
			source = *input.Source
		}
		_, err = applyGroupThresholds(ctx, tx, group, groupThresholds{SuggestionThreshold: *input.SuggestionThreshold, AutomaticThreshold: *input.AutomaticThreshold, AmbiguityMargin: *input.AmbiguityMargin, Source: source}, author)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE voice_matching_settings SET max_candidates=$1,updated_by=$2,updated_at=now() WHERE id=true`, *input.MaxCandidates, author.ID)
		}
	} else {
		_, err = tx.Exec(ctx, `UPDATE voice_matching_settings SET automatic_threshold=$1,suggestion_threshold=$2,ambiguity_margin=$3,max_candidates=$4,updated_by=$5,updated_at=now() WHERE id=true`, *input.AutomaticThreshold, *input.SuggestionThreshold, *input.AmbiguityMargin, *input.MaxCandidates, author.ID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	// New thresholds re-examine every stored fingerprint, never the videos, in the background.
	s.matchingChanged(ctx, 0)
	v, err := effectiveMatchingSettings(ctx, s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

// postVoiceRematch is the explicit rematch of an administrator: one full pass at once, whose figures it answers.
func (s *Server) postVoiceRematch(w http.ResponseWriter, r *http.Request) {
	stats, err := s.rematchNow(context.WithoutCancel(r.Context()))
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, stats)
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
	noReference       bool             // fingerprint group without any reference: nothing to compare with
	settings          matchingSettings // the setting of its fingerprint group
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

// rematchVoices recomputes the machine attributions of local speakers at once, in the caller. clipID 0 covers
// the whole library; otherwise only the speakers of that clip (new fingerprints). The analysis step "matching"
// uses it; a change of references asks for a background pass instead (matchingChanged, voice_matching_queue.go).
func (s *Server) rematchVoices(ctx context.Context, clipID int64) (matchStats, error) {
	voiceMatchingLock.Lock()
	defer voiceMatchingLock.Unlock()
	return s.rematchLocked(ctx, clipID)
}

// rematchLocked is rematchVoices for a caller holding voiceMatchingLock. Runs are serialized (process mutex and
// database advisory lock) so concurrent changes of references cannot interleave their writes; each run reads the
// committed state. Scores are computed before any write, so the rows a run locks are held only while it writes.
func (s *Server) rematchLocked(ctx context.Context, clipID int64) (matchStats, error) {
	outer := ctx
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var stats matchStats
	began := time.Now()
	phases := map[string]int64{}
	phase := func(name string) { phases[name] = time.Since(began).Milliseconds() }
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
	groupRows, err := tx.Query(ctx, `SELECT DISTINCT model,model_version,dimensions FROM voice_fingerprints WHERE usable AND NOT benchmark AND scope='speaker' AND ($1::bigint IS NULL OR clip_id=$1)`, scope)
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
	refused, err := loadRefusals(ctx, tx, scope)
	if err != nil {
		return stats, err
	}
	thresholds, err := loadGroupThresholds(ctx, tx)
	if err != nil {
		return stats, err
	}
	var targets []matchTarget
	for _, g := range groups {
		// Each group is decided with its own thresholds; without applied ones it only suggests.
		found, groupErr := s.matchGroupScores(ctx, tx, g, settings.forGroup(thresholdsOf(thresholds, g)), scope, refused)
		if groupErr != nil {
			return stats, groupErr
		}
		targets = append(targets, found...)
	}
	phase("ranked_ms")
	existing, err := loadExistingMatches(ctx, tx, scope)
	if err != nil {
		return stats, err
	}
	names, err := personNames(ctx, tx, targets, existing)
	if err != nil {
		return stats, err
	}
	evaluated := make(map[int64]bool, len(targets))
	churn := 0 // machine rows inserted or deleted by this run
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
		if existing[t.speakerID].attributionID == 0 {
			churn++ // a new machine row
		}
		level, person, confidence, candidates := decideMatch(t.scores, t.settings)
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
		churn++
	}
	phase("written_ms")
	summaries, err := clipMatchSummaries(ctx, tx, reasons, scope)
	if err != nil {
		return stats, err
	}
	if err = tx.Commit(ctx); err != nil {
		return stats, err
	}
	phase("committed_ms")
	s.logClipSummaries(summaries, scope != nil)
	// The steps below follow the committed matching, each with its own time budget: a slow one (the proximity of
	// unknown voices on a large library) never leaves the reviews unannounced. Measured in
	// docs/performance-rapprochement.md.
	after := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(outer, 5*time.Minute)
	}
	ctx, cancel = after()
	defer cancel()
	// Many machine rows inserted or deleted at once (a first pass, a new reference group) leave the planner
	// statistics stale; the queries below on attributions would pick a poor plan until autovacuum catches up.
	if churn >= analyzeChurn {
		if _, err = s.Pool.Exec(ctx, `ANALYZE speaker_attributions, speaker_attribution_candidates`); err != nil {
			s.Log.Error("voice matching statistics failed", logging.Failure(err)...)
		}
	}
	for _, g := range groups {
		s.ensureVoiceIndex(ctx, g)
	}
	phase("indexed_ms")
	// Proposed proximity between unknown voices follows the references and thresholds.
	ctx, cancel = after()
	defer cancel()
	if err = s.refreshVoiceGroupLinks(ctx); err != nil {
		s.Log.Error("voice group links failed", logging.Failure(err)...)
	}
	phase("linked_ms")
	// The voices that still need a human decision are announced (or the action resolved).
	ctx, cancel = after()
	defer cancel()
	if clipID > 0 {
		s.syncSpeakerReview(ctx, clipID)
	} else {
		s.syncAllSpeakerReviews(ctx)
	}
	phase("reviewed_ms")
	// Elapsed time at the end of each phase, for a measurement or a slow library (#392).
	s.Log.Debug("voice_matching_phases", "clip_id", clipID, "targets", len(targets), "phases", phases)
	return stats, nil
}

// matchGroupScores returns every evaluable speaker of one fingerprint group with its score against each Person
// at or above the suggestion threshold, best first. Refused Persons (refused: speaker -> Persons) are never
// scored for that speaker.
func (s *Server) matchGroupScores(ctx context.Context, tx pgx.Tx, g matchGroup, settings matchingSettings, scope *int64, refused map[int64]map[int64]bool) ([]matchTarget, error) {
	samples, err := loadReferenceSamples(ctx, tx, g, nil)
	if err != nil {
		return nil, err
	}
	refs := newVoiceReferences(settings.ScoreMode, g.dimensions, samples)
	rows, err := tx.Query(ctx, `SELECT f.local_speaker_id,f.clip_id,f.embedding::real[]
  FROM voice_fingerprints f
  WHERE f.usable AND NOT f.benchmark AND f.scope='speaker' AND f.model=$1 AND f.model_version=$2 AND f.dimensions=$3 AND ($4::bigint IS NULL OR f.clip_id=$4)
    AND EXISTS (SELECT 1 FROM local_speakers ls WHERE ls.id=f.local_speaker_id AND ls.kind<>'microphone' AND `+analysis.NoDispositionVoice+`)
    AND NOT EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=f.local_speaker_id AND (a.validation_state='validated' OR a.origin<>$5))
  ORDER BY f.local_speaker_id`, g.model, g.version, g.dimensions, scope, matchingOrigin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []matchTarget
	for rows.Next() {
		var sid, clip int64
		var vector []float32
		if err = rows.Scan(&sid, &clip, &vector); err != nil {
			return nil, err
		}
		if len(out) > 0 && out[len(out)-1].speakerID == sid {
			continue // one speaker fingerprint per speaker and group
		}
		t := matchTarget{speakerID: sid, clipID: clip, model: g.model, version: g.version, noReference: refs.empty(), settings: settings}
		if !t.noReference {
			t.scores = refs.rank(vector, func(person int64) bool { return !refused[sid][person] }, settings.SuggestionThreshold)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// loadReferenceSamples reads the samples the References of one fingerprint group are built from: the active
// validated samples of Persons who kept their voice profile, with the contributive production fingerprint of
// their local speaker in that group. Only a human validation makes a sample: an automatic attribution never
// enters a Reference. The weight is the net speech the vector was computed on (its duration for a row stored
// before net speech was measured). sampleIDs, when not nil, keeps those samples only (the reference half of the
// measured accuracy).
func loadReferenceSamples(ctx context.Context, q pgQuerier, g matchGroup, sampleIDs []int64) ([]referenceSample, error) {
	rows, err := q.Query(ctx, `SELECT vs.person_id,vs.clip_id,f.embedding::real[],COALESCE(f.net_speech_seconds,f.duration_seconds),f.usable
  `+referenceSamplesFrom+`
    AND ($4::bigint[] IS NULL OR vs.id=ANY($4))
  ORDER BY vs.person_id,vs.id`, g.model, g.version, g.dimensions, sampleIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []referenceSample
	for rows.Next() {
		var sample referenceSample
		if err = rows.Scan(&sample.Person, &sample.Clip, &sample.Vector, &sample.NetSpeech, &sample.Comparable); err != nil {
			return nil, err
		}
		out = append(out, sample)
	}
	return out, rows.Err()
}

// loadRefusals lists, per local speaker, the Persons refused for it (ADR 0031), within the clip when scoped.
func loadRefusals(ctx context.Context, tx pgx.Tx, scope *int64) (map[int64]map[int64]bool, error) {
	rows, err := tx.Query(ctx, `SELECT x.local_speaker_id,x.person_id FROM voice_refusals x WHERE $1::bigint IS NULL OR x.local_speaker_id IN (SELECT id FROM local_speakers WHERE clip_id=$1)`, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]map[int64]bool{}
	for rows.Next() {
		var speaker, person int64
		if err = rows.Scan(&speaker, &person); err != nil {
			return nil, err
		}
		if out[speaker] == nil {
			out[speaker] = map[int64]bool{}
		}
		out[speaker][person] = true
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

// analyzeChurn is the number of machine rows a run inserts or deletes from which it refreshes the planner
// statistics of the attributions (#392 measurement: 22 000 new suggestions made the proximity of unknown voices
// take minutes instead of seconds).
const analyzeChurn = 1000

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
