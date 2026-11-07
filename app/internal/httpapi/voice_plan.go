package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
	"snitch/internal/jobs"
)

// Voice analysis of the tracks a published audio plan kept (#342, ADR 0052).
//
// The plan owns one diarization and one fingerprint child per kept track, bound to their jobs in
// the transaction that creates them (audioplans.Store.BindChild). The jobs are the usual voice
// jobs of the Worker: their results project into the local speakers of their own stream, so two
// tracks never share a voice. Matching stays the clip-wide one, run after every fingerprint.
//
// Nothing here is reachable from the legacy flow: StartPlanVoice is called by the plan
// orchestration once a revision is published (the final activation, #344) or by the explicit
// administrator action POST /api/admin/clips/{id}/audio-plans/{plan_id}/voice.

var (
	errPlanNotPublished  = errors.New("audio plan is not the published revision")
	errPlanSuperseded    = errors.New("another audio plan is published")
	errVoiceStepInFlight = errors.New("a voice step of this track is running without the plan")
)

// voicePlanStake is the human work a (re)diarization of the tracks would make obsolete.
type voicePlanStake struct {
	ValidatedSpeakers  int `json:"validated_speakers"`
	VoiceSamples       int `json:"voice_samples"`
	SpeakerCorrections int `json:"speaker_corrections"`
	// AffectedPersons are the Persons whose Reference loses a Voice sample or a validation.
	AffectedPersons int `json:"affected_persons"`
	// AudioStreamIndices are the tracks that carry some of it, in order.
	AudioStreamIndices []int `json:"audio_stream_indices"`
}

func (s voicePlanStake) empty() bool {
	return s.ValidatedSpeakers+s.VoiceSamples+s.SpeakerCorrections == 0
}

type voicePlanTrack struct {
	AudioStreamIndex int `json:"audio_stream_index"`
	// Status: not_started, queued, running, done, failed, cancelled or not_applicable.
	Status string `json:"status"`
	// Cause explains a non-obvious status: voice_worker_unavailable, microphone_direct_attribution,
	// no_speakers, cancelled_by_admin, or the redacted code of the failed step.
	Cause string `json:"cause"`
	// Reused: the diarization of this pipeline already existed and was kept (only missing work is redone).
	Reused         bool           `json:"reused"`
	Microphone     bool           `json:"microphone"`
	Diarization    *voiceStep     `json:"diarization"`
	Fingerprint    *voiceStep     `json:"fingerprint"`
	SpeakerCount   int            `json:"speaker_count"`
	PipelineModel  string         `json:"model"`
	ModelVersion   string         `json:"model_version"`
	Stake          voicePlanStake `json:"stake"`
	needsDiarizing bool
	pipeline       string
}

type voicePlanState struct {
	PlanID     int64  `json:"plan_id"`
	ClipID     int64  `json:"clip_id"`
	RevisionID string `json:"revision_id"`
	// Published: the plan's revision is the visible one. Voice work only starts then.
	Published bool `json:"published"`
	// Status: not_published, not_started, queued, running, done or failed.
	Status          string           `json:"status"`
	Cause           string           `json:"cause"`
	PipelineVersion string           `json:"pipeline_version"`
	Tracks          []voicePlanTrack `json:"tracks"`
	Stake           voicePlanStake   `json:"stake"`
}

func (s *Server) registerVoicePlanRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/clips/{id}/audio-plans/{plan_id}/voice", s.protected(true, http.HandlerFunc(s.getAudioPlanVoice)))
	mux.Handle("POST /api/admin/clips/{id}/audio-plans/{plan_id}/voice", s.protected(true, http.HandlerFunc(s.startAudioPlanVoice)))
}

func planIDOf(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("plan_id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_plan_id")
		return 0, false
	}
	return id, true
}

func (s *Server) getAudioPlanVoice(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	planID, ok := planIDOf(w, r)
	if !ok {
		return
	}
	state, err := s.PlanVoiceState(r.Context(), planID, clip.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "audio_plan_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, state)
}

func (s *Server) startAudioPlanVoice(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	planID, ok := planIDOf(w, r)
	if !ok {
		return
	}
	var input struct {
		ConfirmReplaceValidations bool `json:"confirm_replace_validations"`
	}
	if !decode(w, r, &input) {
		return
	}
	state, err := s.StartPlanVoice(r.Context(), planID, clip.ID, s.authorOf(r), input.ConfirmReplaceValidations)
	var confirmation *voiceConfirmationError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		fail(w, http.StatusNotFound, "audio_plan_not_found")
	case errors.Is(err, errPlanNotPublished):
		fail(w, http.StatusConflict, "audio_plan_not_published")
	case errors.Is(err, errPlanSuperseded):
		fail(w, http.StatusConflict, "audio_plan_superseded")
	case errors.Is(err, errVoiceStepInFlight):
		fail(w, http.StatusConflict, "voice_step_in_progress")
	case errors.As(err, &confirmation):
		reply(w, http.StatusConflict, map[string]any{"error": "voice_rediarization_confirmation_required", "stake": confirmation.Stake})
	case err != nil:
		s.internal(w, err)
	default:
		reply(w, http.StatusOK, state)
	}
}

// voiceConfirmationError carries the stake the administrator has to confirm.
type voiceConfirmationError struct{ Stake voicePlanStake }

func (e *voiceConfirmationError) Error() string { return "voice reanalysis needs confirmation" }

// planVoiceTrackIndices lists the kept tracks of the plan, ordered.
func planVoiceTrackIndices(ctx context.Context, q querier, planID int64) ([]int, error) {
	rows, err := q.Query(ctx, `SELECT track_index FROM audio_plan_tracks WHERE plan_id=$1 ORDER BY track_index`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var index int
		if err = rows.Scan(&index); err != nil {
			return nil, err
		}
		out = append(out, index)
	}
	return out, rows.Err()
}

const voiceStepColumns = `j.id,j.step,j.status,j.input_version,j.attempts,j.max_attempts,j.last_error`

func scanVoiceStep(row pgx.Row) (*voiceStep, error) {
	var step voiceStep
	var status string
	err := row.Scan(&step.JobID, &step.Step, &status, &step.InputVersion, &step.Attempts, &step.MaxAttempts, &step.LastError)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	step.Status = status
	return &step, err
}

func voiceStepOf(ctx context.Context, q querier, clipID int64, step, version string) (*voiceStep, error) {
	return scanVoiceStep(q.QueryRow(ctx, `SELECT `+voiceStepColumns+` FROM jobs j WHERE j.clip_id=$1 AND j.step=$2 AND j.input_version=$3 AND j.requested_model_id IS NULL ORDER BY j.id DESC LIMIT 1`, clipID, step, version))
}

func voiceChildOf(ctx context.Context, q querier, planID int64, track int, stage string) (*voiceStep, error) {
	return scanVoiceStep(q.QueryRow(ctx, `SELECT `+voiceStepColumns+` FROM audio_plan_children c JOIN jobs j ON j.id=c.job_id WHERE c.plan_id=$1 AND c.track_index=$2 AND c.stage=$3`, planID, track, stage))
}

// planVoicePin is the pipeline version the plan's voice work is pinned to: the one of its first
// start, otherwise the active one. Every track of a plan uses the same models.
func (s *Server) planVoicePin(ctx context.Context, q querier, planID int64) (string, bool, error) {
	var version string
	err := q.QueryRow(ctx, `SELECT pipeline_version FROM audio_plan_voice_starts WHERE plan_id=$1`, planID).Scan(&version)
	if err == nil {
		return version, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, err
	}
	active, _, err := s.activeVoicePipeline(ctx)
	return active, false, err
}

// loadPlanVoiceTrack reads what exists for one kept track. Steps bound to the plan win; failing
// that, a diarization of the pinned pipeline that is already the track's current one is kept.
func (s *Server) loadPlanVoiceTrack(ctx context.Context, q querier, planID, clipID int64, track int, pipeline string, microphone bool) (voicePlanTrack, error) {
	out := voicePlanTrack{AudioStreamIndex: track, Microphone: microphone, pipeline: pipeline, Status: "not_started", Stake: voicePlanStake{AudioStreamIndices: []int{}}}
	if microphone {
		out.Status, out.Cause = "not_applicable", "microphone_direct_attribution"
		return out, nil
	}
	var err error
	if out.Diarization, err = voiceChildOf(ctx, q, planID, track, "diarization"); err != nil {
		return out, err
	}
	diarStep := fmt.Sprintf("voice.diarization.%d", track)
	if out.Diarization == nil {
		expected := fmt.Sprintf("d5-a%d-p%s", track, pipeline)
		current, err := scanVoiceStep(q.QueryRow(ctx, `SELECT `+voiceStepColumns+` FROM jobs j WHERE j.clip_id=$1 AND j.step=$2 AND j.status='done' AND j.requested_model_id IS NULL ORDER BY j.updated_at DESC,j.id DESC LIMIT 1`, clipID, diarStep))
		if err != nil {
			return out, err
		}
		if current != nil && current.InputVersion == expected {
			out.Diarization, out.Reused = current, true
		}
	}
	out.needsDiarizing = out.Diarization == nil || out.Diarization.Status != "done"
	if out.Fingerprint, err = voiceChildOf(ctx, q, planID, track, "fingerprint"); err != nil {
		return out, err
	}
	if out.Fingerprint == nil && out.Diarization != nil && out.Diarization.Status == "done" {
		if version, _, ok := diarizationFingerprintVersion(out.Diarization, track); ok {
			if out.Fingerprint, err = voiceStepOf(ctx, q, clipID, fmt.Sprintf("voice.fingerprint.%d", track), version); err != nil {
				return out, err
			}
		}
	}
	if err = q.QueryRow(ctx, `SELECT count(*) FROM local_speakers WHERE clip_id=$1 AND audio_stream_index=$2 AND kind<>'archived'`, clipID, track).Scan(&out.SpeakerCount); err != nil {
		return out, err
	}
	return out, nil
}

// diarizationFingerprintVersion is the input version of the fingerprint step of a diarization.
func diarizationFingerprintVersion(diarization *voiceStep, track int) (string, string, bool) {
	pipeline, _, ok := jobs.VoiceInputVersion(diarization.InputVersion)
	if !ok {
		return "", "", false
	}
	return fmt.Sprintf("f5-a%d-p%s-d%d", track, pipeline, diarization.JobID), pipeline, true
}

// settle derives the status of a track from its steps.
func (t *voicePlanTrack) settle(ctx context.Context, q querier, workersMissing bool) error {
	describe := func(step *voiceStep) (string, string) {
		switch step.Status {
		case "pending", "retry_wait":
			if workersMissing {
				return "queued", "voice_worker_unavailable"
			}
			return "queued", ""
		case "leased":
			return "running", ""
		case "done":
			return "done", ""
		case "cancelled":
			return "cancelled", "cancelled_by_admin"
		}
		cause := step.LastError
		if cause == "" {
			cause = "step_failed"
		}
		return "failed", cause
	}
	switch {
	case t.Microphone:
		return nil
	case t.Diarization == nil:
		t.Status, t.Cause = "not_started", ""
	default:
		t.Status, t.Cause = describe(t.Diarization)
		if t.Status == "done" {
			var speakers int
			if err := q.QueryRow(ctx, `SELECT CASE WHEN jsonb_typeof(result->'speakers')='array' THEN jsonb_array_length(result->'speakers') ELSE 0 END FROM jobs WHERE id=$1`, t.Diarization.JobID).Scan(&speakers); err != nil {
				return err
			}
			switch {
			case speakers == 0:
				t.Status, t.Cause = "not_applicable", "no_speakers"
			case t.Fingerprint == nil:
				t.Status, t.Cause = "not_started", "fingerprint_missing"
			default:
				t.Status, t.Cause = describe(t.Fingerprint)
			}
		}
	}
	return nil
}

func overallVoiceStatus(tracks []voicePlanTrack) (string, string) {
	counts := map[string]int{}
	cause := ""
	for _, t := range tracks {
		counts[t.Status]++
		if cause == "" && (t.Status == "queued" || t.Status == "failed" || t.Status == "cancelled") {
			cause = t.Cause
		}
	}
	switch {
	case counts["running"] > 0:
		return "running", ""
	case counts["queued"] > 0:
		return "queued", cause
	case counts["failed"] > 0 || counts["cancelled"] > 0:
		return "failed", cause
	case counts["not_started"] > 0:
		return "not_started", ""
	}
	return "done", ""
}

// voiceStakeOf counts the human work attached to the diarized voices of the given streams.
func voiceStakeOf(ctx context.Context, q querier, clipID int64, tracks []int) (voicePlanStake, error) {
	stake := voicePlanStake{AudioStreamIndices: []int{}}
	if len(tracks) == 0 {
		return stake, nil
	}
	err := q.QueryRow(ctx, `SELECT
 (SELECT count(DISTINCT a.local_speaker_id) FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND ls.audio_stream_index=ANY($2) AND ls.kind='diarized' AND a.validation_state='validated'),
 (SELECT count(*) FROM voice_samples vs JOIN local_speakers ls ON ls.id=vs.local_speaker_id WHERE vs.clip_id=$1 AND vs.retired_at IS NULL AND ls.audio_stream_index=ANY($2) AND ls.kind='diarized'),
 (SELECT count(*) FROM segment_corrections sc JOIN local_speakers ls ON ls.id=sc.speaker_override_id WHERE sc.clip_id=$1 AND sc.stale_at IS NULL AND ls.audio_stream_index=ANY($2) AND ls.kind='diarized'),
 (SELECT count(DISTINCT person_id) FROM (
   SELECT a.person_id FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND ls.audio_stream_index=ANY($2) AND ls.kind='diarized' AND a.validation_state='validated' AND a.person_id IS NOT NULL
   UNION SELECT vs.person_id FROM voice_samples vs JOIN local_speakers ls ON ls.id=vs.local_speaker_id WHERE vs.clip_id=$1 AND vs.retired_at IS NULL AND ls.audio_stream_index=ANY($2) AND ls.kind='diarized') p),
 COALESCE((SELECT jsonb_agg(DISTINCT ls.audio_stream_index ORDER BY ls.audio_stream_index) FROM local_speakers ls WHERE ls.clip_id=$1 AND ls.audio_stream_index=ANY($2) AND ls.kind='diarized' AND (
   EXISTS(SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.validation_state='validated')
   OR EXISTS(SELECT 1 FROM voice_samples vs WHERE vs.local_speaker_id=ls.id AND vs.retired_at IS NULL)
   OR EXISTS(SELECT 1 FROM segment_corrections sc WHERE sc.speaker_override_id=ls.id AND sc.stale_at IS NULL))),'[]'::jsonb)`, clipID, tracks).Scan(&stake.ValidatedSpeakers, &stake.VoiceSamples, &stake.SpeakerCorrections, &stake.AffectedPersons, &jsonInto{&stake.AudioStreamIndices})
	return stake, err
}

// jsonInto scans a jsonb column into its target.
type jsonInto struct{ target any }

func (j *jsonInto) Scan(src any) error {
	switch value := src.(type) {
	case []byte:
		return json.Unmarshal(value, j.target)
	case string:
		return json.Unmarshal([]byte(value), j.target)
	}
	return fmt.Errorf("unexpected jsonb value %T", src)
}

// voiceStepSettled: the job stopped without a result and can be resumed.
func voiceStepSettled(status string) bool {
	return status == "failed" || status == "dead" || status == "cancelled"
}

// planPublication tells whether the plan's revision is the visible one.
func planPublication(ctx context.Context, q querier, planID, clipID int64) (revision string, published bool, superseded bool, err error) {
	var plan *int64
	err = q.QueryRow(ctx, `SELECT p.revision_id,r.plan_id FROM audio_publications p JOIN transcript_revisions r ON r.id=p.revision_id WHERE p.clip_id=$1`, clipID).Scan(&revision, &plan)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, false, nil
	}
	if err != nil {
		return "", false, false, err
	}
	if plan != nil && *plan == planID {
		return revision, true, false, nil
	}
	return "", false, true, nil
}

// microphoneIndex is the designated microphone stream of the clip, when one is applied.
func microphoneIndex(ctx context.Context, q querier, clipID int64) (int, bool, error) {
	var index int
	err := q.QueryRow(ctx, `SELECT audio_stream_index FROM local_speakers WHERE clip_id=$1 AND kind='microphone'`, clipID).Scan(&index)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return index, err == nil, err
}

// PlanVoiceState is the read-only per-track state of the voice analysis of a plan, with the human
// work a start would replace. pgx.ErrNoRows: no such plan on this clip.
func (s *Server) PlanVoiceState(ctx context.Context, planID, clipID int64) (voicePlanState, error) {
	var owner int64
	if err := s.Pool.QueryRow(ctx, `SELECT clip_id FROM audio_plans WHERE id=$1`, planID).Scan(&owner); err != nil {
		return voicePlanState{}, err
	}
	if owner != clipID {
		return voicePlanState{}, pgx.ErrNoRows
	}
	return s.planVoiceState(ctx, s.Pool, planID, clipID)
}

func (s *Server) planVoiceState(ctx context.Context, q querier, planID, clipID int64) (voicePlanState, error) {
	state := voicePlanState{PlanID: planID, ClipID: clipID, Tracks: []voicePlanTrack{}, Stake: voicePlanStake{AudioStreamIndices: []int{}}}
	var superseded bool
	var err error
	if state.RevisionID, state.Published, superseded, err = planPublication(ctx, q, planID, clipID); err != nil {
		return state, err
	}
	if state.PipelineVersion, _, err = s.planVoicePin(ctx, q, planID); err != nil {
		return state, err
	}
	config, err := s.voiceConfig(ctx, state.PipelineVersion)
	if err != nil {
		return state, err
	}
	inventory, err := s.voiceInventory(ctx)
	if err != nil {
		return state, err
	}
	workersMissing := inventory.real && inventory.WorkersVoiceReady == 0
	indices, err := planVoiceTrackIndices(ctx, q, planID)
	if err != nil {
		return state, err
	}
	mic, hasMic, err := microphoneIndex(ctx, q, clipID)
	if err != nil {
		return state, err
	}
	atStake := []int{}
	for _, track := range indices {
		t, err := s.loadPlanVoiceTrack(ctx, q, planID, clipID, track, state.PipelineVersion, hasMic && mic == track)
		if err != nil {
			return state, err
		}
		t.PipelineModel, t.ModelVersion = config.Model, config.ModelVersion
		if err = t.settle(ctx, q, workersMissing); err != nil {
			return state, err
		}
		if t.needsDiarizing && !t.Microphone {
			if t.Stake, err = voiceStakeOf(ctx, q, clipID, []int{track}); err != nil {
				return state, err
			}
			atStake = append(atStake, track)
		}
		state.Tracks = append(state.Tracks, t)
	}
	if state.Stake, err = voiceStakeOf(ctx, q, clipID, atStake); err != nil {
		return state, err
	}
	switch {
	case superseded:
		state.Status, state.Cause = "not_published", "audio_plan_superseded"
	case !state.Published:
		state.Status, state.Cause = "not_published", "audio_plan_not_published"
	default:
		state.Status, state.Cause = overallVoiceStatus(state.Tracks)
	}
	return state, nil
}

// StartPlanVoice schedules, for the published plan, the voice steps that are missing, on every kept
// track that is not the microphone: diarization, then (queued by its projection) fingerprints.
// Work already done or running is left alone; failed or cancelled steps are resumed in place.
// When the work it would replace includes human validations, References or speaker corrections it
// returns a *voiceConfirmationError unless confirm is true; old voices stay until a new result
// succeeds, and a validation made after the confirmed instant still locks them.
func (s *Server) StartPlanVoice(ctx context.Context, planID, clipID int64, author authorRef, confirm bool) (voicePlanState, error) {
	var owner int64
	if err := s.Pool.QueryRow(ctx, `SELECT clip_id FROM audio_plans WHERE id=$1`, planID).Scan(&owner); err != nil {
		return voicePlanState{}, err
	}
	if owner != clipID {
		return voicePlanState{}, pgx.ErrNoRows
	}
	// The microphone designation of the clip is (re)applied first, as in the single-track flow.
	if _, err := s.applyMicrophone(ctx, clipID, author); err != nil {
		return voicePlanState{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return voicePlanState{}, err
	}
	defer tx.Rollback(ctx)
	var locked int64
	if err = tx.QueryRow(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, clipID).Scan(&locked); err != nil {
		return voicePlanState{}, err
	}
	_, published, superseded, err := planPublication(ctx, tx, planID, clipID)
	if err != nil {
		return voicePlanState{}, err
	}
	if superseded {
		return voicePlanState{}, errPlanSuperseded
	}
	if !published {
		return voicePlanState{}, errPlanNotPublished
	}
	state, err := s.planVoiceState(ctx, tx, planID, clipID)
	if err != nil {
		return voicePlanState{}, err
	}
	// Only a track with no live diarization gets new work; a live one is already covered by the
	// confirmation of the start that created it, whose instant keeps protecting its voices.
	fresh := false
	for _, track := range state.Tracks {
		if track.needsDiarizing && !track.Microphone && (track.Diarization == nil || voiceStepSettled(track.Diarization.Status)) {
			fresh = true
		}
	}
	if fresh && !state.Stake.empty() && !confirm {
		return voicePlanState{}, &voiceConfirmationError{Stake: state.Stake}
	}
	stakeJSON, _ := json.Marshal(state.Stake)
	if fresh {
		if _, err = tx.Exec(ctx, `INSERT INTO audio_plan_voice_starts(plan_id,pipeline_version,stake_at,stake,confirmed,started_by) VALUES($1,$2,clock_timestamp(),$3::jsonb,$4,$5)
ON CONFLICT(plan_id) DO UPDATE SET stake_at=excluded.stake_at,stake=excluded.stake,confirmed=excluded.confirmed OR audio_plan_voice_starts.confirmed,started_by=excluded.started_by`, planID, state.PipelineVersion, stakeJSON, !state.Stake.empty(), author.ID); err != nil {
			return voicePlanState{}, err
		}
	}
	store := audioplans.New(tx)
	for _, track := range state.Tracks {
		if track.Microphone {
			continue
		}
		diarStep := fmt.Sprintf("voice.diarization.%d", track.AudioStreamIndex)
		if track.needsDiarizing {
			if track.Diarization != nil && !voiceStepSettled(track.Diarization.Status) {
				continue // queued or running: nothing to do, its start already bound it
			}
			version := fmt.Sprintf("d5-a%d-p%s", track.AudioStreamIndex, state.PipelineVersion)
			if track.Diarization != nil {
				version = track.Diarization.InputVersion
			}
			if err = s.ensurePlanVoiceJob(ctx, tx, store, planID, clipID, track.AudioStreamIndex, "diarization", diarStep, version, author); err != nil {
				return voicePlanState{}, err
			}
			continue
		}
		// The diarization is done (by this plan, or kept from the same pipeline): only the
		// fingerprints can be missing.
		version, _, ok := diarizationFingerprintVersion(track.Diarization, track.AudioStreamIndex)
		if !ok || track.Cause == "no_speakers" {
			continue
		}
		if err = s.ensurePlanVoiceJob(ctx, tx, store, planID, clipID, track.AudioStreamIndex, "fingerprint", fmt.Sprintf("voice.fingerprint.%d", track.AudioStreamIndex), version, author); err != nil {
			return voicePlanState{}, err
		}
	}
	state, err = s.planVoiceState(ctx, tx, planID, clipID)
	if err != nil {
		return voicePlanState{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return voicePlanState{}, err
	}
	return state, nil
}

// ensurePlanVoiceJob guarantees one live job for the step, bound to the plan before any Worker can
// claim it: it creates it, or resumes a failed, dead or cancelled one in place (same job, new cycle).
func (s *Server) ensurePlanVoiceJob(ctx context.Context, tx pgx.Tx, store *audioplans.Store, planID, clipID int64, track int, stage, step, version string, author authorRef) error {
	if _, err := tx.Exec(ctx, `UPDATE jobs SET status='failed',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,last_error='input_version_superseded',updated_at=now() WHERE clip_id=$1 AND step=$2 AND input_version<>$3 AND requested_model_id IS NULL AND status IN ('pending','leased','retry_wait') AND id NOT IN (SELECT job_id FROM audio_plan_children WHERE plan_id=$4)`, clipID, step, version, planID); err != nil {
		return err
	}
	var jobID int64
	var status string
	var bound bool
	err := tx.QueryRow(ctx, `SELECT j.id,j.status,EXISTS(SELECT 1 FROM audio_plan_children c WHERE c.job_id=j.id AND c.plan_id=$4) FROM jobs j WHERE j.clip_id=$1 AND j.step=$2 AND j.input_version=$3 AND j.requested_model_id IS NULL ORDER BY j.id DESC LIMIT 1`, clipID, step, version, planID).Scan(&jobID, &status, &bound)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if err = tx.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,priority,max_attempts) VALUES($1,$2,$3,0,3) RETURNING id`, clipID, step, version).Scan(&jobID); err != nil {
			return err
		}
		status = "pending"
	case err != nil:
		return err
	}
	switch status {
	case "done":
		return nil
	case "dead", "cancelled", "failed":
		var cycle int
		if err = tx.QueryRow(ctx, `UPDATE jobs SET status='pending',attempts=0,cycle=cycle+1,next_attempt_at=now(),last_error='',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,updated_at=now() WHERE id=$1 RETURNING cycle`, jobID).Scan(&cycle); err != nil {
			return err
		}
		actor := ""
		if author.ID != nil {
			actor = strconv.FormatInt(*author.ID, 10)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,actor,actor_id) VALUES($1,'admin_relaunched',$2,'admin',$3)`, jobID, cycle, actor); err != nil {
			return err
		}
		status = "pending"
	}
	if bound {
		return nil
	}
	if status != "pending" {
		// Running without the plan (another flow leased it): never adopt it half way.
		return errVoiceStepInFlight
	}
	if err = store.BindChild(ctx, audioplans.Child{PlanId: planID, TrackIndex: track, Stage: contract.AudioPlanChildStage(stage), JobId: jobID, InputVersion: version}); err != nil {
		if errors.Is(err, audioplans.ErrConflict) {
			return errVoiceStepInFlight
		}
		return err
	}
	return nil
}

// visiblePlanOf returns the plan whose revision is the visible transcript of the clip, if any.
func (s *Server) visiblePlanOf(ctx context.Context, clipID int64) (int64, bool, error) {
	var plan int64
	err := s.Pool.QueryRow(ctx, `SELECT r.plan_id FROM audio_publications p JOIN transcript_revisions r ON r.id=p.revision_id WHERE p.clip_id=$1 AND r.plan_id IS NOT NULL`, clipID).Scan(&plan)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return plan, err == nil, err
}

// planClipVoice is the voice view of a clip whose visible transcript comes from a plan: the same
// shape as the single-track view, with every kept track's steps and voices (each voice carries its
// stream) and the per-track state in tracks.
func (s *Server) planClipVoice(ctx context.Context, clipID, planID int64) (map[string]any, error) {
	state, err := s.planVoiceState(ctx, s.Pool, planID, clipID)
	if err != nil {
		return nil, err
	}
	indices := make([]int, 0, len(state.Tracks))
	steps := []voiceStep{}
	history := []voiceHistoryEntry{}
	for _, track := range state.Tracks {
		indices = append(indices, track.AudioStreamIndex)
		if track.Diarization != nil {
			steps = append(steps, *track.Diarization)
		}
		if track.Fingerprint != nil {
			steps = append(steps, *track.Fingerprint)
		}
		entries, err := s.voiceHistory(ctx, clipID, track.AudioStreamIndex)
		if err != nil {
			return nil, err
		}
		history = append(history, entries...)
	}
	speakers, err := s.voiceSpeakers(ctx, clipID, indices)
	if err != nil {
		return nil, err
	}
	status := state.Status
	switch status {
	case "not_started", "not_published":
		status = "none"
	}
	return map[string]any{"clip_id": clipID, "status": status, "audio_stream_index": nil, "steps": steps, "speakers": speakers, "history": history, "plan_id": planID, "tracks": state.Tracks}, nil
}
