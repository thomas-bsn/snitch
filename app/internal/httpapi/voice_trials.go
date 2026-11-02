package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
)

// Diarization trials (ADR 0022). A trial runs a catalogue model, optionally with a number of speakers, on one clip
// and is shown beside the current result. It is stored in its own table and its jobs use their own step type
// (voicetrial.diarization.<trial id>), so it never reaches the local speakers, attributions, corrections,
// fingerprints, matching or search, and never touches the rules that govern the clip's real voice result.

func (s *Server) registerVoiceTrialRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/clips/{id}/voice-trials", s.protected(true, http.HandlerFunc(s.listVoiceTrials)))
	mux.Handle("POST /api/admin/clips/{id}/voice-trials", s.protected(true, http.HandlerFunc(s.createVoiceTrial)))
	mux.Handle("DELETE /api/admin/clips/{id}/voice-trials/{trial}", s.protected(true, http.HandlerFunc(s.deleteVoiceTrial)))
}

type voiceTrialSpeaker struct {
	Label           string  `json:"label"`
	SpeakingSeconds float64 `json:"speaking_seconds"`
	TurnCount       int     `json:"turn_count"`
}

type voiceTrialView struct {
	ID               int64               `json:"id"`
	ClipID           int64               `json:"clip_id"`
	AudioStreamIndex int                 `json:"audio_stream_index"`
	ModelID          string              `json:"model_id"`
	NumSpeakers      *int                `json:"num_speakers"`
	MinSpeakers      *int                `json:"min_speakers"`
	MaxSpeakers      *int                `json:"max_speakers"`
	Status           string              `json:"status"`
	Speakers         []voiceTrialSpeaker `json:"speakers"`
	CreatedAt        string              `json:"created_at"`
}

func voiceTrialStep(id int64) string { return fmt.Sprintf("voicetrial.diarization.%d", id) }

// voiceTrialJobStatus is the state of a trial without result, from its latest job: pending, running or failed.
func (s *Server) voiceTrialJobStatus(ctx context.Context, clipID, trialID int64) (string, error) {
	var status string
	err := s.Pool.QueryRow(ctx, `SELECT status FROM jobs WHERE clip_id=$1 AND step=$2 ORDER BY id DESC LIMIT 1`, clipID, voiceTrialStep(trialID)).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "failed", nil
	}
	switch status {
	case "pending", "retry_wait", "done":
		return "pending", err
	case "leased":
		return "running", err
	}
	return "failed", err
}

func (s *Server) listVoiceTrials(w http.ResponseWriter, r *http.Request) {
	clipID, ok := pathID(w, r, "invalid_clip_id")
	if !ok {
		return
	}
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM clips WHERE id=$1)`, clipID).Scan(&exists); err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "clip_not_found")
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT id,audio_stream_index,model_id,num_speakers,min_speakers,max_speakers,result,to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM voice_diarization_trials WHERE clip_id=$1 ORDER BY id DESC`, clipID)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	trials := []voiceTrialView{}
	for rows.Next() {
		v := voiceTrialView{ClipID: clipID, Speakers: []voiceTrialSpeaker{}}
		var raw []byte
		if err = rows.Scan(&v.ID, &v.AudioStreamIndex, &v.ModelID, &v.NumSpeakers, &v.MinSpeakers, &v.MaxSpeakers, &raw, &v.CreatedAt); err != nil {
			s.internal(w, err)
			return
		}
		if raw != nil {
			v.Status = "done"
			var stored struct {
				Speakers []voiceTrialSpeaker `json:"speakers"`
			}
			if err = json.Unmarshal(raw, &stored); err == nil && stored.Speakers != nil {
				v.Speakers = stored.Speakers
			}
		}
		trials = append(trials, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	rows.Close()
	for i := range trials {
		if trials[i].Status == "done" {
			continue
		}
		if trials[i].Status, err = s.voiceTrialJobStatus(r.Context(), clipID, trials[i].ID); err != nil {
			s.internal(w, err)
			return
		}
	}
	reply(w, http.StatusOK, map[string]any{"trials": trials})
}

type voiceTrialInput struct {
	ModelID          string `json:"model_id"`
	AudioStreamIndex *int   `json:"audio_stream_index"`
	NumSpeakers      *int   `json:"num_speakers"`
	MinSpeakers      *int   `json:"min_speakers"`
	MaxSpeakers      *int   `json:"max_speakers"`
}

func (s *Server) createVoiceTrial(w http.ResponseWriter, r *http.Request) {
	clipID, ok := pathID(w, r, "invalid_clip_id")
	if !ok {
		return
	}
	var input voiceTrialInput
	if !decode(w, r, &input) {
		return
	}
	entry, found := voiceCatalogByID(input.ModelID)
	if !found || entry.Kind != "diarization" {
		fail(w, http.StatusBadRequest, "invalid_voice_model")
		return
	}
	active, err := s.activeVoiceModelID(r)
	if err != nil {
		s.internal(w, err)
		return
	}
	candidate, err := s.candidateVoiceModelID(r)
	if err != nil {
		s.internal(w, err)
		return
	}
	// A trial is for the model being evaluated (the candidate) or the one in use, never any catalogue entry.
	if entry.ID != active && entry.ID != candidate {
		fail(w, http.StatusBadRequest, "invalid_voice_model")
		return
	}
	limit := maxVoiceSpeakers
	if entry.MaxSpeakers > 0 {
		limit = entry.MaxSpeakers
	}
	for _, n := range []*int{input.NumSpeakers, input.MinSpeakers, input.MaxSpeakers} {
		if n != nil && !entry.SupportsNumSpeakers {
			fail(w, http.StatusBadRequest, "voice_trial_option_unsupported")
			return
		}
		if n != nil && (*n < 1 || *n > limit) {
			fail(w, http.StatusBadRequest, "invalid_voice_trial_options")
			return
		}
	}
	if input.NumSpeakers != nil && (input.MinSpeakers != nil || input.MaxSpeakers != nil) || (input.MinSpeakers != nil && input.MaxSpeakers != nil && *input.MinSpeakers > *input.MaxSpeakers) {
		fail(w, http.StatusBadRequest, "invalid_voice_trial_options")
		return
	}
	tracks, err := s.clipAudioTracks(r.Context(), clipID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "clip_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	index := input.AudioStreamIndex
	if index != nil && !hasAudioTrack(tracks, *index) {
		fail(w, http.StatusBadRequest, "invalid_audio_stream")
		return
	}
	if index == nil {
		selection, selErr := s.audioSelection(r.Context(), clipID, tracks)
		if selErr != nil {
			s.internal(w, selErr)
			return
		}
		if selection.AudioStreamIndex == nil {
			fail(w, http.StatusConflict, "audio_selection_required")
			return
		}
		index = selection.AudioStreamIndex
	}
	var ready bool
	if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM worker_voice_models m JOIN workers w ON w.id=m.worker_id WHERE m.model_id=$1 AND m.state='ready' AND m.removal_requested=false AND w.revoked_at IS NULL)`, entry.ID).Scan(&ready); err != nil {
		s.internal(w, err)
		return
	}
	if !ready {
		fail(w, http.StatusConflict, "voice_model_not_ready")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var id int64
	var run int
	key := `clip_id=$1 AND audio_stream_index=$2 AND model_id=$3 AND COALESCE(num_speakers,0)=COALESCE($4::int,0) AND COALESCE(min_speakers,0)=COALESCE($5::int,0) AND COALESCE(max_speakers,0)=COALESCE($6::int,0)`
	err = tx.QueryRow(r.Context(), `SELECT id,run FROM voice_diarization_trials WHERE `+key+` FOR UPDATE`, clipID, *index, entry.ID, input.NumSpeakers, input.MinSpeakers, input.MaxSpeakers).Scan(&id, &run)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		run = 1
		err = tx.QueryRow(r.Context(), `INSERT INTO voice_diarization_trials(clip_id,audio_stream_index,model_id,model_repo,model_revision,num_speakers,min_speakers,max_speakers,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, clipID, *index, entry.ID, entry.Repo, entry.Revision, input.NumSpeakers, input.MinSpeakers, input.MaxSpeakers, s.authorOf(r).Username).Scan(&id)
	case err == nil:
		// The same (clip, model, options) is kept once: running it again replaces its result.
		var running bool
		if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM jobs WHERE clip_id=$1 AND step=$2 AND status IN ('pending','leased','retry_wait'))`, clipID, voiceTrialStep(id)).Scan(&running); err == nil && running {
			fail(w, http.StatusConflict, "voice_trial_running")
			return
		}
		run++
		if err == nil {
			_, err = tx.Exec(r.Context(), `UPDATE voice_diarization_trials SET run=$2,result=NULL,completed_at=NULL,model_repo=$3,model_revision=$4,created_at=now() WHERE id=$1`, id, run, entry.Repo, entry.Revision)
		}
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	_, _, err = s.Jobs.Schedule(r.Context(), jobs.ScheduleInput{ClipID: clipID, Step: voiceTrialStep(id), InputVersion: fmt.Sprintf("x1-t%d-r%d", id, run), MaxAttempts: 2})
	switch {
	case errors.Is(err, jobs.ErrClipNotFound):
		fail(w, http.StatusNotFound, "clip_not_found")
	case err != nil:
		s.internal(w, err)
	default:
		reply(w, http.StatusCreated, voiceTrialView{ID: id, ClipID: clipID, AudioStreamIndex: *index, ModelID: entry.ID, NumSpeakers: input.NumSpeakers, MinSpeakers: input.MinSpeakers, MaxSpeakers: input.MaxSpeakers, Status: "pending", Speakers: []voiceTrialSpeaker{}})
	}
}

func (s *Server) deleteVoiceTrial(w http.ResponseWriter, r *http.Request) {
	clipID, ok := pathID(w, r, "invalid_clip_id")
	if !ok {
		return
	}
	trialID, err := strconv.ParseInt(r.PathValue("trial"), 10, 64)
	if err != nil || trialID < 1 {
		fail(w, http.StatusBadRequest, "invalid_voice_trial_id")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `DELETE FROM voice_diarization_trials WHERE id=$1 AND clip_id=$2`, trialID, clipID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, http.StatusNotFound, "voice_trial_not_found")
		return
	}
	// A job still waiting for a Worker would have nothing to report to.
	if _, err = tx.Exec(r.Context(), `UPDATE jobs SET status='cancelled',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,updated_at=now() WHERE clip_id=$1 AND step=$2 AND status IN ('pending','leased','retry_wait')`, clipID, voiceTrialStep(trialID)); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// completeVoiceTrial records a trial result in the trial's own table: no local speaker, attribution,
// fingerprint step or match is ever created from it.
func (s *Server) completeVoiceTrial(w http.ResponseWriter, r *http.Request, id int64, step, status, inputVersion string, result json.RawMessage) {
	var project func(ctx context.Context, tx pgx.Tx, job jobs.Job) error
	if status != "done" {
		leased, err := s.Jobs.Input(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r))
		if errors.Is(err, jobs.ErrLeaseNotOwned) {
			fail(w, http.StatusConflict, "lease_not_owned")
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
		trialID, convErr := strconv.ParseInt(strings.TrimPrefix(step, "voicetrial.diarization."), 10, 64)
		if convErr != nil {
			fail(w, http.StatusBadRequest, "invalid_voice_result")
			return
		}
		var repo, revision string
		var index int
		if err = s.Pool.QueryRow(r.Context(), `SELECT model_repo,model_revision,audio_stream_index FROM voice_diarization_trials WHERE id=$1 AND clip_id=$2`, trialID, leased.Job.ClipID).Scan(&repo, &revision, &index); err != nil {
			// The trial was deleted while the Worker ran: its result has nowhere to go.
			fail(w, http.StatusConflict, "voice_trial_not_found")
			return
		}
		parsed, valid := parseDiarization(result, voicePipeline{Provider: "pyannote", Model: repo, ModelVersion: revision}, index)
		if !valid {
			fail(w, http.StatusBadRequest, "invalid_voice_result")
			return
		}
		speakers := make([]voiceTrialSpeaker, 0, len(*parsed.Speakers))
		for _, speaker := range *parsed.Speakers {
			item := voiceTrialSpeaker{Label: speaker.Label, TurnCount: len(*speaker.Turns)}
			for _, t := range *speaker.Turns {
				item.SpeakingSeconds += *t.End - *t.Start
			}
			speakers = append(speakers, item)
		}
		stored, _ := json.Marshal(map[string]any{"speakers": speakers})
		project = func(ctx context.Context, tx pgx.Tx, job jobs.Job) error {
			_, err := tx.Exec(ctx, `UPDATE voice_diarization_trials SET result=$2::jsonb,completed_at=now() WHERE id=$1`, trialID, string(stored))
			return err
		}
	}
	summarize := func(json.RawMessage) json.RawMessage { return json.RawMessage(`{}`) }
	_, err := s.Jobs.CompleteWith(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r), inputVersion, result, jobs.CompleteOptions{MaxResultBytes: maxVoiceResultBytes, Summarize: summarize, Project: project})
	switch {
	case errors.Is(err, jobs.ErrInvalidInput):
		fail(w, http.StatusBadRequest, "invalid_voice_result")
	case errors.Is(err, jobs.ErrLeaseNotOwned):
		fail(w, http.StatusConflict, "lease_not_owned")
	case err != nil:
		s.internal(w, err)
	default:
		reply(w, http.StatusOK, map[string]bool{"recorded": true})
	}
}
