package httpapi

import (
	"context"
	"net/http"
	"sort"
	"time"
)

// Accuracy measure and automation switch (phase 5).
//
// The measure replays the matching on the validated samples themselves, with one rule:
// a clip that served as a reference never serves as a test. For each Person the clips
// holding an active validated sample are sorted; the first half (rounded up) are
// reference clips, the others are test candidates. A candidate whose clip also holds a
// reference of any Person is set aside ("excluded"): a validated sample is a reference
// or a test, never both, and no test shares a clip with a reference. A Person with a
// single clip only provides references.
//
// Each test is then scored against References built from the reference samples alone, with the production
// score function and mode (voice_score.go), and decided with the production decision
// (decideMatch with the automation forced on, so the measure does not depend on the
// switch). Only tests that would be automatic count: precision is the share of those
// whose Person is the one a human validated; confirmations is their number of correct
// ones; coverage is the share of tests that would be automatic. The numbers describe a
// small, synthetic or user-validated population: they are never a guarantee, and no
// quantity of samples is presented as one.
//
// The library figures come from the stored attributions. Automatic coverage there
// counts the voices the automation covers or would cover once enabled (an automatic
// attribution, or a suggestion at or above the automatic threshold). The microphone
// track is counted apart: its owner is designated by an administrator, not computed
// from a similarity, and it does not change with the switch, so it neither raises the
// automatic coverage nor lowers the verification volume.

func (s *Server) registerVoiceAutomationRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/voice-accuracy", s.protected(true, http.HandlerFunc(s.getVoiceAccuracy)))
	mux.Handle("PUT /api/admin/voice-automation/criteria", s.protected(true, http.HandlerFunc(s.putVoiceCriteria)))
	mux.Handle("PUT /api/admin/voice-automation", s.protected(true, http.HandlerFunc(s.putVoiceAutomation)))
}

type accuracyEvaluation struct {
	ReferenceSamples   int      `json:"reference_samples"`
	TestSamples        int      `json:"test_samples"`
	ExcludedSamples    int      `json:"excluded_samples"`
	EvaluatedAutomatic int      `json:"evaluated_automatic"`
	Confirmed          int      `json:"confirmed"`
	Precision          *float64 `json:"precision"`
	Coverage           *float64 `json:"coverage"`
}

type accuracyPerson struct {
	Person               personRef `json:"person"`
	ReferenceSamples     int       `json:"reference_samples"`
	TestSamples          int       `json:"test_samples"`
	AutomaticPredictions int       `json:"automatic_predictions"`
	Confirmed            int       `json:"confirmed"`
	Precision            *float64  `json:"precision"`
}

type accuracyLibrary struct {
	SpeakersTotal      int      `json:"speakers_total"`
	Validated          int      `json:"validated"`
	Automatic          int      `json:"automatic"`
	Suggestion         int      `json:"suggestion"`
	Ambiguous          int      `json:"ambiguous"`
	Unknown            int      `json:"unknown"`
	ToVerify           int      `json:"to_verify"`
	Coverage           *float64 `json:"coverage"`
	MicrophoneAssigned int      `json:"microphone_assigned"`
}

type accuracyCriteria struct {
	Met              bool `json:"met"`
	PrecisionMet     bool `json:"precision_met"`
	ConfirmationsMet bool `json:"confirmations_met"`
}

type automationEvent struct {
	Action             string    `json:"action"`
	Author             string    `json:"author"`
	CriteriaMet        bool      `json:"criteria_met"`
	Forced             bool      `json:"forced"`
	MeasuredPrecision  *float64  `json:"measured_precision"`
	Confirmations      int       `json:"confirmations"`
	PrecisionThreshold float64   `json:"precision_threshold"`
	MinConfirmations   int       `json:"min_confirmations"`
	CreatedAt          time.Time `json:"created_at"`
}

type voiceAccuracy struct {
	MeasuredAt         time.Time          `json:"measured_at"`
	AutoEnabled        bool               `json:"auto_enabled"`
	AutomaticThreshold float64            `json:"automatic_threshold"`
	PrecisionThreshold float64            `json:"precision_threshold"`
	MinConfirmations   int                `json:"min_confirmations"`
	Evaluation         accuracyEvaluation `json:"evaluation"`
	Persons            []accuracyPerson   `json:"persons"`
	Library            accuracyLibrary    `json:"library"`
	Criteria           accuracyCriteria   `json:"criteria"`
	ActivationProposed bool               `json:"activation_proposed"`
	Events             []automationEvent  `json:"events"`
	Calibration        voiceCalibration   `json:"calibration"`
}

type accuracySample struct {
	id, personID, clipID int64
}

func ratio(num, den int) *float64 {
	if den == 0 {
		return nil
	}
	v := float64(num) / float64(den)
	return &v
}

// splitSamples assigns each sample a role. Samples are ordered by Person, clip, id.
func splitSamples(samples []accuracySample) (references, tests []accuracySample, excluded int) {
	byPerson := map[int64][]accuracySample{}
	var order []int64
	for _, sample := range samples {
		if _, seen := byPerson[sample.personID]; !seen {
			order = append(order, sample.personID)
		}
		byPerson[sample.personID] = append(byPerson[sample.personID], sample)
	}
	referenceClips := map[int64]bool{}
	var candidates []accuracySample
	for _, person := range order {
		list := byPerson[person]
		var clips []int64
		for _, sample := range list {
			if len(clips) == 0 || clips[len(clips)-1] != sample.clipID {
				clips = append(clips, sample.clipID)
			}
		}
		firstTest := (len(clips) + 1) / 2
		testClips := map[int64]bool{}
		for _, clip := range clips[firstTest:] {
			testClips[clip] = true
		}
		for _, sample := range list {
			if testClips[sample.clipID] {
				candidates = append(candidates, sample)
				continue
			}
			references = append(references, sample)
			referenceClips[sample.clipID] = true
		}
	}
	for _, sample := range candidates {
		if referenceClips[sample.clipID] {
			excluded++
			continue
		}
		tests = append(tests, sample)
	}
	return references, tests, excluded
}

// accuracySamples reads the active validated samples the measure splits: those with a contributive
// production fingerprint, ordered by Person, clip and id.
func (s *Server) accuracySamples(ctx context.Context) ([]accuracySample, error) {
	rows, err := s.Pool.Query(ctx, `SELECT vs.id,vs.person_id,vs.clip_id FROM voice_samples vs
WHERE vs.retired_at IS NULL AND EXISTS (SELECT 1 FROM voice_fingerprints f WHERE f.clip_id=vs.clip_id AND f.local_speaker_id=vs.local_speaker_id AND f.contributive AND NOT f.benchmark AND f.scope='speaker')
ORDER BY vs.person_id,vs.clip_id,vs.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var samples []accuracySample
	for rows.Next() {
		var sample accuracySample
		if err = rows.Scan(&sample.id, &sample.personID, &sample.clipID); err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}
	return samples, rows.Err()
}

// measureAccuracy computes the evaluation and the per-Person figures from the database.
func (s *Server) measureAccuracy(ctx context.Context, settings matchingSettings) (accuracyEvaluation, []accuracyPerson, accuracySplit, error) {
	var eval accuracyEvaluation
	var split accuracySplit
	samples, err := s.accuracySamples(ctx)
	if err != nil {
		return eval, nil, split, err
	}
	references, tests, excluded := splitSamples(samples)
	eval.ReferenceSamples, eval.TestSamples, eval.ExcludedSamples = len(references), len(tests), excluded

	people := map[int64]*accuracyPerson{}
	for _, sample := range samples {
		if people[sample.personID] == nil {
			people[sample.personID] = &accuracyPerson{Person: personRef{ID: sample.personID}}
		}
	}
	for _, sample := range references {
		people[sample.personID].ReferenceSamples++
	}
	truth := map[int64]int64{}
	for _, sample := range tests {
		people[sample.personID].TestSamples++
		truth[sample.id] = sample.personID
	}

	split.references, split.tests = references, tests
	thresholds, err := loadGroupThresholds(ctx, s.Pool, settings.ScoreMode)
	if err != nil {
		return eval, nil, split, err
	}
	scores, err := s.testScores(ctx, references, tests, settings, thresholds)
	if err != nil {
		return eval, nil, split, err
	}
	for _, sample := range tests {
		level, predicted, _, _ := decideMatch(scores[sample.id].scores, scores[sample.id].settings)
		if level != LevelAutomatic || predicted == nil {
			continue
		}
		eval.EvaluatedAutomatic++
		if people[*predicted] == nil {
			continue
		}
		people[*predicted].AutomaticPredictions++
		if *predicted == truth[sample.id] {
			eval.Confirmed++
			people[*predicted].Confirmed++
		}
	}
	eval.Precision = ratio(eval.Confirmed, eval.EvaluatedAutomatic)
	eval.Coverage = ratio(eval.EvaluatedAutomatic, eval.TestSamples)

	ids := make([]int64, 0, len(people))
	for id := range people {
		ids = append(ids, id)
	}
	names := map[int64]string{}
	nameRows, err := s.Pool.Query(ctx, `SELECT id,display_name FROM persons WHERE id=ANY($1)`, ids)
	if err != nil {
		return eval, nil, split, err
	}
	for nameRows.Next() {
		var id int64
		var name string
		if err = nameRows.Scan(&id, &name); err != nil {
			nameRows.Close()
			return eval, nil, split, err
		}
		names[id] = name
	}
	nameRows.Close()
	if err = nameRows.Err(); err != nil {
		return eval, nil, split, err
	}
	out := make([]accuracyPerson, 0, len(people))
	for id, p := range people {
		p.Person.DisplayName = names[id]
		p.Precision = ratio(p.Confirmed, p.AutomaticPredictions)
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Person.DisplayName != out[j].Person.DisplayName {
			return out[i].Person.DisplayName < out[j].Person.DisplayName
		}
		return out[i].Person.ID < out[j].Person.ID
	})
	return eval, out, split, nil
}

// testScore is the score of a test sample against each Person, and the setting its group decides with.
type testScore struct {
	scores   []matchScore
	settings matchingSettings
}

// accuracySplit keeps the reference and test samples of the measure for the calibration.
type accuracySplit struct {
	references, tests []accuracySample
}

// measureSettings is the setting a group is measured with: its applied thresholds with the automation forced on,
// so that the measure does not depend on the switch; a group without applied thresholds is never automatic, as
// in production.
func measureSettings(settings matchingSettings, t *groupThresholds) matchingSettings {
	out := settings.forGroup(t)
	out.AutoEnabled = t != nil
	return out
}

// testScores gives, for each test sample, its score against each Person with the production score function and
// mode, the References being built from the reference samples only, one fingerprint group at a time (vectors of
// different models, versions or dimensions are never compared), filtered with the suggestion threshold of the
// group like the production matching.
func (s *Server) testScores(ctx context.Context, references, tests []accuracySample, settings matchingSettings, thresholds map[matchGroup]groupThresholds) (map[int64]testScore, error) {
	out := map[int64]testScore{}
	if len(references) == 0 || len(tests) == 0 {
		return out, nil
	}
	refIDs, testIDs := make([]int64, len(references)), make([]int64, len(tests))
	for i, sample := range references {
		refIDs[i] = sample.id
	}
	for i, sample := range tests {
		testIDs[i] = sample.id
	}
	// A test is compared like a production voice: through its comparable fingerprint, in the first group found.
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT ON (vs.id) vs.id,f.model,f.model_version,f.dimensions,f.embedding::real[]
  FROM voice_samples vs JOIN voice_fingerprints f ON f.clip_id=vs.clip_id AND f.local_speaker_id=vs.local_speaker_id
  WHERE vs.id=ANY($1) AND f.usable AND NOT f.benchmark AND f.scope='speaker'
  ORDER BY vs.id,f.model,f.model_version,f.dimensions`, testIDs)
	if err != nil {
		return nil, err
	}
	type testVector struct {
		id     int64
		vector []float32
	}
	byGroup := map[matchGroup][]testVector{}
	var groups []matchGroup
	for rows.Next() {
		var g matchGroup
		var t testVector
		if err = rows.Scan(&t.id, &g.model, &g.version, &g.dimensions, &t.vector); err != nil {
			rows.Close()
			return nil, err
		}
		if _, seen := byGroup[g]; !seen {
			groups = append(groups, g)
		}
		byGroup[g] = append(byGroup[g], t)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for _, g := range groups {
		samples, loadErr := loadReferenceSamples(ctx, s.Pool, g, refIDs)
		if loadErr != nil {
			return nil, loadErr
		}
		refs := newVoiceReferences(settings.ScoreMode, g.dimensions, samples)
		gs := measureSettings(settings, thresholdsOf(thresholds, g))
		for _, t := range byGroup[g] {
			out[t.id] = testScore{scores: refs.rank(t.vector, nil, gs.SuggestionThreshold), settings: gs}
		}
	}
	return out, nil
}

// libraryFigures counts the stored attributions. A suggestion "would be automatic once enabled" when its score
// reaches the automatic threshold applied to its fingerprint group for the active score mode; a group without
// thresholds for that mode never attributes automatically, so its suggestions stay suggestions.
func (s *Server) libraryFigures(ctx context.Context, mode scoreMode) (accuracyLibrary, error) {
	var l accuracyLibrary
	err := s.Pool.QueryRow(ctx, `SELECT
  count(*) FILTER (WHERE ls.kind='diarized'),
  count(*) FILTER (WHERE ls.kind='diarized' AND v.id IS NOT NULL),
  count(*) FILTER (WHERE ls.kind='diarized' AND v.id IS NULL AND m.person_id IS NOT NULL AND (m.decision_level='automatic' OR (m.decision_level='suggestion' AND m.confidence >= gt.automatic_threshold))),
  count(*) FILTER (WHERE ls.kind='diarized' AND v.id IS NULL AND m.person_id IS NOT NULL AND m.decision_level='suggestion' AND m.confidence < COALESCE(gt.automatic_threshold,2)),
  count(*) FILTER (WHERE ls.kind='diarized' AND v.id IS NULL AND m.decision_level='ambiguous'),
  count(*) FILTER (WHERE ls.kind='microphone' AND EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.person_id IS NOT NULL))
FROM local_speakers ls
LEFT JOIN speaker_attributions v ON v.local_speaker_id=ls.id AND v.validation_state='validated'
LEFT JOIN speaker_attributions m ON m.local_speaker_id=ls.id AND m.validation_state='unvalidated' AND m.origin=$1
LEFT JOIN LATERAL (SELECT t.automatic_threshold FROM voice_group_thresholds t
  WHERE t.model=m.match_model AND t.model_version=m.match_model_version AND t.score_mode=$2
    AND t.dimensions IN (SELECT f.dimensions FROM voice_fingerprints f WHERE f.local_speaker_id=ls.id AND f.model=m.match_model AND f.model_version=m.match_model_version AND NOT f.benchmark AND f.scope='speaker')
  ORDER BY t.version DESC LIMIT 1) gt ON true
WHERE ls.kind='microphone' OR (ls.kind='diarized' AND `+notIgnoredVoice+`)`, matchingOrigin, mode).Scan(&l.SpeakersTotal, &l.Validated, &l.Automatic, &l.Suggestion, &l.Ambiguous, &l.MicrophoneAssigned)
	if err != nil {
		return l, err
	}
	l.Unknown = l.SpeakersTotal - l.Validated - l.Automatic - l.Suggestion - l.Ambiguous
	l.ToVerify = l.Suggestion + l.Ambiguous + l.Unknown
	l.Coverage = ratio(l.Automatic, l.SpeakersTotal-l.Validated)
	return l, nil
}

func criteriaOf(eval accuracyEvaluation, settings matchingSettings) accuracyCriteria {
	c := accuracyCriteria{
		PrecisionMet:     eval.EvaluatedAutomatic > 0 && float64(eval.Confirmed) >= settings.PrecisionThreshold*float64(eval.EvaluatedAutomatic)-1e-9,
		ConfirmationsMet: eval.Confirmed >= settings.MinConfirmations,
	}
	c.Met = c.PrecisionMet && c.ConfirmationsMet
	return c
}

func (s *Server) loadAutomationEvents(ctx context.Context) ([]automationEvent, error) {
	rows, err := s.Pool.Query(ctx, `SELECT action,author_username,criteria_met,forced,measured_precision,confirmations,precision_threshold,min_confirmations,created_at FROM voice_automation_events ORDER BY id DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []automationEvent{}
	for rows.Next() {
		var e automationEvent
		if err = rows.Scan(&e.Action, &e.Author, &e.CriteriaMet, &e.Forced, &e.MeasuredPrecision, &e.Confirmations, &e.PrecisionThreshold, &e.MinConfirmations, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Server) buildAccuracy(ctx context.Context) (voiceAccuracy, error) {
	var report voiceAccuracy
	settings, err := loadMatchingSettings(ctx, s.Pool)
	if err != nil {
		return report, err
	}
	effective, err := effectiveMatchingSettings(ctx, s.Pool)
	if err != nil {
		return report, err
	}
	report.MeasuredAt = time.Now().UTC()
	report.AutoEnabled, report.AutomaticThreshold = settings.AutoEnabled, effective.AutomaticThreshold
	report.PrecisionThreshold, report.MinConfirmations = settings.PrecisionThreshold, settings.MinConfirmations
	var split accuracySplit
	if report.Evaluation, report.Persons, split, err = s.measureAccuracy(ctx, settings); err != nil {
		return report, err
	}
	if report.Calibration, err = s.calibration(ctx, settings, split); err != nil {
		return report, err
	}
	if report.Library, err = s.libraryFigures(ctx, settings.ScoreMode); err != nil {
		return report, err
	}
	report.Criteria = criteriaOf(report.Evaluation, settings)
	report.ActivationProposed = report.Criteria.Met && !settings.AutoEnabled
	report.Events, err = s.loadAutomationEvents(ctx)
	return report, err
}

func (s *Server) getVoiceAccuracy(w http.ResponseWriter, r *http.Request) {
	report, err := s.buildAccuracy(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, report)
}

// putVoiceCriteria changes the activation criteria. They only decide when Snitch
// proposes the activation: they never switch anything by themselves.
func (s *Server) putVoiceCriteria(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PrecisionThreshold *float64 `json:"precision_threshold"`
		MinConfirmations   *int     `json:"min_confirmations"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.PrecisionThreshold == nil || input.MinConfirmations == nil || *input.PrecisionThreshold < 0 || *input.PrecisionThreshold > 1 || *input.MinConfirmations < 1 || *input.MinConfirmations > 1000000 {
		fail(w, http.StatusBadRequest, "invalid_voice_criteria")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	if _, err := s.Pool.Exec(r.Context(), `UPDATE voice_matching_settings SET precision_threshold=$1,min_confirmations=$2,updated_by=$3,updated_at=now() WHERE id=true`, *input.PrecisionThreshold, *input.MinConfirmations, ss.User.ID); err != nil {
		s.internal(w, err)
		return
	}
	settings, err := effectiveMatchingSettings(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, settings)
}

// putVoiceAutomation is the only place that changes auto_enabled. Enabling while the
// criteria are not met needs the explicit confirmation of the administrator; every
// effective change is recorded with the measure at that moment, then the stored
// fingerprints are matched again (disabling brings automatic attributions back to
// suggestions, enabling promotes the strong ones). References are never touched.
func (s *Server) putVoiceAutomation(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Enabled              *bool `json:"enabled"`
		ConfirmUnmetCriteria bool  `json:"confirm_unmet_criteria"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Enabled == nil {
		fail(w, http.StatusBadRequest, "invalid_voice_automation")
		return
	}
	ctx := context.WithoutCancel(r.Context())
	ss := r.Context().Value(sessionKey{}).(session)
	settings, err := loadMatchingSettings(ctx, s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	eval, _, _, err := s.measureAccuracy(ctx, settings)
	if err != nil {
		s.internal(w, err)
		return
	}
	criteria := criteriaOf(eval, settings)
	if *input.Enabled && !settings.AutoEnabled && !criteria.Met && !input.ConfirmUnmetCriteria {
		fail(w, http.StatusConflict, "criteria_not_met")
		return
	}
	if err = s.recordAutomationSwitch(ctx, *input.Enabled, input.ConfirmUnmetCriteria, ss, eval, criteria, settings); err != nil {
		s.internal(w, err)
		return
	}
	// The switch answers what it changed: one full pass at once (it also takes the requests waiting).
	stats, err := s.rematchNow(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	updated, err := effectiveMatchingSettings(ctx, s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"settings": updated, "summary": stats})
}

func (s *Server) recordAutomationSwitch(ctx context.Context, enabled, confirmed bool, ss session, eval accuracyEvaluation, criteria accuracyCriteria, settings matchingSettings) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var current bool
	if err = tx.QueryRow(ctx, `SELECT auto_enabled FROM voice_matching_settings WHERE id=true FOR UPDATE`).Scan(&current); err != nil {
		return err
	}
	if current == enabled {
		return tx.Commit(ctx) // already in that state: nothing to record
	}
	if _, err = tx.Exec(ctx, `UPDATE voice_matching_settings SET auto_enabled=$1,auto_changed_at=now(),auto_changed_by=$2,updated_by=$2,updated_at=now() WHERE id=true`, enabled, ss.User.ID); err != nil {
		return err
	}
	action := "disable"
	if enabled {
		action = "enable"
	}
	forced := enabled && confirmed && !criteria.Met
	if _, err = tx.Exec(ctx, `INSERT INTO voice_automation_events(action,author_id,author_username,criteria_met,forced,measured_precision,confirmations,precision_threshold,min_confirmations) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, action, ss.User.ID, ss.User.Username, criteria.Met, forced, eval.Precision, eval.Confirmed, settings.PrecisionThreshold, settings.MinConfirmations); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
