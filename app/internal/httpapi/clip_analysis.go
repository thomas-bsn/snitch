package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/analysis"
	"snitch/internal/events"
	"snitch/internal/jobs"
)

// analysisInterval is how often active analysis runs are reconciled with their jobs, as a
// safety net beside the immediate advance after a worker result.
const analysisInterval = 5 * time.Second

// analysisHooks gives the analysis service the clip decisions that live in this package:
// the audio selection, the transcription input version and the protection of corrections.
type analysisHooks struct{ s *Server }

func (h analysisHooks) Prepare(ctx context.Context, clipID int64) (analysis.Preparation, error) {
	plan, err := h.s.planTranscription(ctx, clipID, nil, false)
	switch {
	case err != nil && err.Error() == "audio selection required":
		return analysis.Preparation{Status: analysis.StatusBlocked, Reason: "audio_track_selection_required"}, nil
	case err != nil:
		return analysis.Preparation{}, err
	case !plan.ready:
		return analysis.Preparation{Status: analysis.StatusWaiting, Reason: "clip_not_ready"}, nil
	}
	return analysis.Preparation{Status: analysis.StatusDone, Reason: "clip_ready"}, nil
}

func (h analysisHooks) Transcribe(ctx context.Context, clipID int64, since time.Time, force bool) (analysis.Transcription, error) {
	plan, err := h.s.planTranscription(ctx, clipID, nil, true)
	if err != nil {
		return analysis.Transcription{}, err
	}
	var jobID int64
	// A forced transcription replaces the valid result: only the jobs of this run count, and
	// the corrections were confirmed before the run started, so nothing is kept for them.
	err = h.s.Pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step=$2 AND input_version=$3 AND ((NOT $5 AND status IN ('pending','leased','retry_wait','done')) OR created_at>=$4) ORDER BY id DESC LIMIT 1`, clipID, plan.step, plan.version, since, force).Scan(&jobID)
	if err == nil {
		job, readErr := h.s.readJob(ctx, jobID)
		return analysis.Transcription{Job: job}, readErr
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return analysis.Transcription{}, err
	}
	// A transcription of an older input version stays when corrections would be replaced.
	if _, found, corrErr := h.s.clipCorrections(ctx, clipID); corrErr != nil {
		return analysis.Transcription{}, corrErr
	} else if found && !force {
		if visible, ok, visErr := h.s.visibleTranscription(ctx, h.s.Pool, clipID); visErr != nil {
			return analysis.Transcription{}, visErr
		} else if ok {
			job, readErr := h.s.readJob(ctx, visible.JobID)
			return analysis.Transcription{Job: job, Kept: true}, readErr
		}
	}
	job, _, err := h.s.Jobs.Schedule(ctx, jobs.ScheduleInput{ClipID: clipID, Step: plan.step, InputVersion: plan.version, MaxAttempts: 3})
	return analysis.Transcription{Job: job}, err
}

func (h analysisHooks) HasTranscription(ctx context.Context, clipID int64) (bool, error) {
	_, ok, err := h.s.visibleTranscription(ctx, h.s.Pool, clipID)
	return ok, err
}

// Index follows the semantic indexing of the visible transcription. A missing or failing
// semantic index never touches the text search, which is fed by the transcription itself.
func (h analysisHooks) Index(ctx context.Context, clipID int64) (analysis.StepResult, error) {
	visible, ok, err := h.s.visibleTranscription(ctx, h.s.Pool, clipID)
	if err != nil || !ok {
		return analysis.StepResult{Status: analysis.StatusWaiting, Reason: "awaiting_transcription"}, err
	}
	// A transcription that predates the indexing profile still needs its task.
	if _, err = h.s.Pool.Exec(ctx, `INSERT INTO embedding_tasks(profile_id,transcription_job_id) SELECT active_profile_id,$1 FROM embedding_index_state WHERE id=1 AND active_profile_id IS NOT NULL ON CONFLICT DO NOTHING`, visible.JobID); err != nil {
		return analysis.StepResult{}, err
	}
	progress, err := h.s.clipProgress(ctx, clipID, visible.JobID, true)
	if err != nil {
		return analysis.StepResult{}, err
	}
	switch progress.Indexing {
	case "done":
		return analysis.StepResult{Status: analysis.StatusDone, Reason: "completed"}, nil
	case "running":
		return analysis.StepResult{Status: analysis.StatusRunning, Reason: "running"}, nil
	case "failed":
		return analysis.StepResult{Status: analysis.StatusFailed, Reason: "semantic_index_failed"}, nil
	case "unavailable":
		return analysis.StepResult{Status: analysis.StatusNotApplicable, Reason: "semantic_search_unavailable"}, nil
	}
	return analysis.StepResult{Status: analysis.StatusWaiting, Reason: "awaiting_indexer"}, nil
}

// Voice runs the voice branch of a clip: the microphone designation is applied (a
// designated track is attributed directly and never diarized), then the diarization and
// its fingerprints are followed. An existing diarization is reused, never replaced by a
// run unless the administrator forced the diarization (and confirmed its consequences): a
// change of model does not re-diarize clips by itself.
func (h analysisHooks) Voice(ctx context.Context, clipID int64, since time.Time, force bool) (analysis.VoiceSteps, error) {
	s := h.s
	waiting := func(r analysis.StepResult) analysis.VoiceSteps {
		after := analysis.StepResult{Status: analysis.StatusWaiting, Reason: "awaiting_previous_step"}
		return analysis.VoiceSteps{Diarization: r, Fingerprints: after, Matching: &after}
	}
	tracks, err := s.clipAudioTracks(ctx, clipID)
	if err != nil {
		return analysis.VoiceSteps{}, err
	}
	selection, err := s.audioSelection(ctx, clipID, tracks)
	if err != nil {
		return analysis.VoiceSteps{}, err
	}
	if selection.AudioStreamIndex == nil {
		return waiting(analysis.StepResult{Status: analysis.StatusBlocked, Reason: "audio_track_selection_required"}), nil
	}
	index := *selection.AudioStreamIndex
	if _, err = s.applyMicrophone(ctx, clipID, authorRef{Username: "analysis"}); err != nil {
		return analysis.VoiceSteps{}, err
	}
	var microphone bool
	if err = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM local_speakers WHERE clip_id=$1 AND audio_stream_index=$2 AND kind='microphone')`, clipID, index).Scan(&microphone); err != nil {
		return analysis.VoiceSteps{}, err
	}
	if microphone {
		na := analysis.StepResult{Status: analysis.StatusNotApplicable, Reason: "microphone_track"}
		direct := analysis.StepResult{Status: analysis.StatusDone, Reason: "microphone_direct_attribution"}
		return analysis.VoiceSteps{Diarization: na, Fingerprints: na, Matching: &direct}, nil
	}
	diarStep := fmt.Sprintf("voice.diarization.%d", index)
	var diarID int64
	err = s.Pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step=$2 AND status='done' AND (NOT $3 OR created_at>=$4) ORDER BY updated_at DESC,id DESC LIMIT 1`, clipID, diarStep, force, since).Scan(&diarID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.Pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step=$2 AND (status IN ('pending','leased','retry_wait') OR created_at>=$3) ORDER BY id DESC LIMIT 1`, clipID, diarStep, since).Scan(&diarID)
	}
	var diar jobs.Job
	switch {
	case err == nil:
		diar, err = s.readJob(ctx, diarID)
	case errors.Is(err, pgx.ErrNoRows):
		var version string
		if version, _, err = s.activeVoicePipeline(ctx); err == nil {
			diar, _, err = s.Jobs.Schedule(ctx, jobs.ScheduleInput{ClipID: clipID, Step: diarStep, InputVersion: fmt.Sprintf("d5-a%d-p%s", index, version), MaxAttempts: 3})
		}
	}
	if err != nil {
		return analysis.VoiceSteps{}, err
	}
	diarResult := analysis.JobResult(diar, since)
	if diarResult.Status != analysis.StatusDone {
		return waiting(diarResult), nil
	}
	var speakers int
	if err = s.Pool.QueryRow(ctx, `SELECT CASE WHEN jsonb_typeof(result->'speakers')='array' THEN jsonb_array_length(result->'speakers') ELSE 0 END FROM jobs WHERE id=$1`, diar.ID).Scan(&speakers); err != nil {
		return analysis.VoiceSteps{}, err
	}
	if speakers == 0 {
		na := analysis.StepResult{Status: analysis.StatusNotApplicable, Reason: "no_speakers"}
		return analysis.VoiceSteps{Diarization: diarResult, Fingerprints: na, Matching: &na}, nil
	}
	pipeline, _, ok := jobs.VoiceInputVersion(diar.InputVersion)
	if !ok {
		return analysis.VoiceSteps{}, fmt.Errorf("diarization job %d has no voice input version", diar.ID)
	}
	fpStep, fpVersion := fmt.Sprintf("voice.fingerprint.%d", index), fmt.Sprintf("f5-a%d-p%s-d%d", index, pipeline, diar.ID)
	var fpID int64
	err = s.Pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step=$2 AND input_version=$3 AND NOT (status IN ('dead','failed','cancelled') AND created_at<$4 AND $4>(SELECT created_at FROM clip_analysis_runs WHERE clip_id=$1 AND status='active' ORDER BY id DESC LIMIT 1)) ORDER BY id DESC LIMIT 1`, clipID, fpStep, fpVersion, since).Scan(&fpID)
	var fp jobs.Job
	switch {
	case err == nil:
		fp, err = s.readJob(ctx, fpID)
	case errors.Is(err, pgx.ErrNoRows):
		fp, _, err = s.Jobs.Schedule(ctx, jobs.ScheduleInput{ClipID: clipID, Step: fpStep, InputVersion: fpVersion, MaxAttempts: 3})
	}
	if err != nil {
		return analysis.VoiceSteps{}, err
	}
	return analysis.VoiceSteps{Diarization: diarResult, Fingerprints: analysis.JobResult(fp, since)}, nil
}

func (h analysisHooks) Reindex(ctx context.Context, clipID int64) error {
	visible, ok, err := h.s.visibleTranscription(ctx, h.s.Pool, clipID)
	if err != nil || !ok {
		return err
	}
	_, err = h.s.Pool.Exec(ctx, `UPDATE embedding_tasks SET state='pending',lease_expires_at=NULL,error=NULL,updated_at=now() WHERE transcription_job_id=$1 AND state IN ('done','failed') AND profile_id IN (SELECT active_profile_id FROM embedding_index_state WHERE id=1)`, visible.JobID)
	return err
}

func (h analysisHooks) RetryIndex(ctx context.Context, clipID int64) error {
	visible, ok, err := h.s.visibleTranscription(ctx, h.s.Pool, clipID)
	if err != nil || !ok {
		return err
	}
	_, err = h.s.Pool.Exec(ctx, `UPDATE embedding_tasks SET state='pending',lease_expires_at=NULL,error=NULL,updated_at=now() WHERE transcription_job_id=$1 AND state='failed' AND profile_id IN (SELECT active_profile_id FROM embedding_index_state WHERE id=1)`, visible.JobID)
	return err
}

func (h analysisHooks) Match(ctx context.Context, clipID int64) error {
	_, err := h.s.rematchVoices(context.WithoutCancel(ctx), clipID)
	return err
}

func (s *Server) readJob(ctx context.Context, id int64) (jobs.Job, error) {
	var j jobs.Job
	err := s.Pool.QueryRow(ctx, `SELECT id,clip_id,step,input_version,status,attempts,created_at FROM jobs WHERE id=$1`, id).Scan(&j.ID, &j.ClipID, &j.Step, &j.InputVersion, &j.Status, &j.Attempts, &j.CreatedAt)
	return j, err
}

// reconcileAnalysis advances the analysis run of the clip a worker just reported on.
func (s *Server) reconcileAnalysis(ctx context.Context, clipID int64) {
	if err := s.Analysis.ReconcileClip(ctx, clipID); err != nil {
		s.Log.Warn("reconcile analysis after job result", "clip", clipID, "error", err)
	}
}

// settingAutoFullAnalysis is the administration setting "Analyse complète automatique". It
// is on unless it holds "false" (ADR 0035): the absence of a value means the default.
const settingAutoFullAnalysis = "analysis_auto_full"

// startAutomaticAnalysis starts the full analysis of a newly added clip when the setting is
// on. Without a Worker for the voices the run keeps the reasons on its steps; a refusal here
// (an active run of another scope, a missing transcription) never fails the addition.
func (s *Server) startAutomaticAnalysis(ctx context.Context, clipID int64) {
	enabled, err := s.Store.Get(ctx, settingAutoFullAnalysis)
	if err != nil {
		s.Log.Warn("read automatic analysis setting", "clip", clipID, "error", err)
		return
	}
	if enabled == "false" {
		return
	}
	if _, _, err := s.Analysis.Start(ctx, clipID, analysis.ScopeFull, 0); err != nil {
		s.Log.Warn("start automatic analysis", "clip", clipID, "error", err)
	}
}

func (s *Server) startClipAnalysis(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	var input struct {
		Scope                     string   `json:"scope"`
		Mode                      string   `json:"mode"`
		Steps                     []string `json:"steps"`
		ConfirmReplaceCorrections bool     `json:"confirm_replace_corrections"`
		ConfirmReplaceValidations bool     `json:"confirm_replace_validations"`
		// ConfirmedStakeAt echoes the stake_at of the refusal the administrator confirmed.
		ConfirmedStakeAt *time.Time `json:"confirmed_stake_at"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Scope == "" {
		input.Scope = analysis.ScopeTranscription
	}
	opts := analysis.Options{Mode: input.Mode, Steps: input.Steps, ConfirmReplaceCorrections: input.ConfirmReplaceCorrections, ConfirmReplaceValidations: input.ConfirmReplaceValidations}
	if input.ConfirmedStakeAt != nil {
		opts.ConfirmedStakeAt = *input.ConfirmedStakeAt
	}
	if !s.reanalysisConfirmed(w, r, clipID, input.Scope, opts) {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	run, created, err := s.Analysis.StartWith(r.Context(), clipID, input.Scope, ss.User.ID, opts)
	switch {
	case errors.Is(err, analysis.ErrInvalidScope):
		fail(w, http.StatusBadRequest, "invalid_analysis_scope")
	case errors.Is(err, analysis.ErrInvalidReanalysis):
		fail(w, http.StatusBadRequest, "invalid_reanalysis")
	case errors.Is(err, analysis.ErrTranscriptionRequired):
		fail(w, http.StatusConflict, "transcription_required")
	case errors.Is(err, analysis.ErrClipNotFound):
		fail(w, http.StatusNotFound, "clip_not_found")
	case errors.Is(err, analysis.ErrScopeConflict):
		reply(w, http.StatusConflict, map[string]any{"error": "analysis_scope_conflict", "run": run})
	case err != nil:
		s.internal(w, err)
	case created:
		reply(w, http.StatusCreated, map[string]any{"run": run})
	default:
		reply(w, http.StatusOK, map[string]any{"run": run})
	}
}

// clipExists answers 404 clip_not_found (or 500) and returns false when the clip is not there.
func (s *Server) clipExists(w http.ResponseWriter, r *http.Request, clipID int64) bool {
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM clips WHERE id=$1)`, clipID).Scan(&exists); err != nil {
		s.internal(w, err)
		return false
	}
	if !exists {
		fail(w, http.StatusNotFound, "clip_not_found")
	}
	return exists
}

func (s *Server) getClipAnalysis(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	if !s.clipExists(w, r, clipID) {
		return
	}
	run, err := s.Analysis.Latest(r.Context(), clipID)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"run": run})
}

func (s *Server) cancelClipAnalysis(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	run, err := s.Analysis.Cancel(r.Context(), clipID)
	switch {
	case errors.Is(err, analysis.ErrNotActive):
		fail(w, http.StatusConflict, "analysis_not_active")
	case err != nil:
		s.internal(w, err)
	default:
		reply(w, http.StatusOK, map[string]any{"run": run})
	}
}

func (s *Server) retryClipAnalysisStep(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	run, err := s.Analysis.Retry(r.Context(), clipID, r.PathValue("step"))
	switch {
	case errors.Is(err, analysis.ErrInvalidStep):
		fail(w, http.StatusBadRequest, "invalid_analysis_step")
	case errors.Is(err, analysis.ErrNoRun):
		fail(w, http.StatusNotFound, "analysis_not_found")
	case errors.Is(err, analysis.ErrNotRetryable):
		fail(w, http.StatusConflict, "analysis_not_retryable")
	case errors.Is(err, analysis.ErrStepNotFailed):
		fail(w, http.StatusConflict, "analysis_step_not_failed")
	case err != nil:
		s.internal(w, err)
	default:
		reply(w, http.StatusOK, map[string]any{"run": run})
	}
}

// clipAnalysisEvents streams the analysis of a clip: a snapshot, then one per change. The
// administrator role is rechecked against the session before every snapshot.
func (s *Server) clipAnalysisEvents(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	if !s.clipExists(w, r, clipID) {
		return
	}
	s.Events.Stream(w, r, events.Topic{
		Name:      "clip:" + strconv.FormatInt(clipID, 10),
		Authorize: s.adminAuthorizer(r),
		Snapshot: func(ctx context.Context) (any, error) {
			run, err := s.Analysis.Latest(ctx, clipID)
			return map[string]any{"run": run}, err
		},
	})
}

// reanalysisStake is what forced steps would make obsolete: the corrections of the visible
// transcription (a retranscription, ADR 0012) and the validations of the speakers (a new
// diarization, #114). Nothing of it is deleted: histories and validated samples stay.
type reanalysisStake struct {
	Corrections *correctionsAtStake      `json:"corrections"`
	Voice       *voiceRediarizationStake `json:"voice"`
}

// reanalysisConfirmed answers 409 reanalysis_confirmation_required, with the consequences,
// when the forced steps would replace human work that the request did not confirm. It
// returns true when the caller may start the run. An active run is never changed by a start,
// so nothing is asked in that case.
func (s *Server) reanalysisConfirmed(w http.ResponseWriter, r *http.Request, clipID int64, scope string, opts analysis.Options) bool {
	forced, err := analysis.ForcedSteps(scope, opts)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_reanalysis")
		return false
	}
	if len(forced) == 0 {
		return true
	}
	var active bool
	if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM clip_analysis_runs WHERE clip_id=$1 AND status='active')`, clipID).Scan(&active); err != nil {
		s.internal(w, err)
		return false
	}
	if active {
		return true
	}
	// The instant is read from the database clock before the stake, so nothing it shows is newer.
	var stakeAt time.Time
	if err = s.Pool.QueryRow(r.Context(), `SELECT clock_timestamp()`).Scan(&stakeAt); err != nil {
		s.internal(w, err)
		return false
	}
	var stake reanalysisStake
	if slices.Contains(forced, analysis.StepTranscription) && !opts.ConfirmReplaceCorrections {
		at, found, corrErr := s.clipCorrections(r.Context(), clipID)
		if corrErr != nil {
			s.internal(w, corrErr)
			return false
		}
		if found {
			stake.Corrections = &at
		}
	}
	if slices.Contains(forced, analysis.StepDiarization) && !opts.ConfirmReplaceValidations {
		tracks, trErr := s.clipAudioTracks(r.Context(), clipID)
		if trErr != nil && !errors.Is(trErr, pgx.ErrNoRows) {
			s.internal(w, trErr)
			return false
		}
		if trErr == nil {
			selection, selErr := s.audioSelection(r.Context(), clipID, tracks)
			if selErr != nil {
				s.internal(w, selErr)
				return false
			}
			if selection.AudioStreamIndex != nil {
				voice, found, voiceErr := s.voiceRediarizationStake(r.Context(), clipID, *selection.AudioStreamIndex)
				if voiceErr != nil {
					s.internal(w, voiceErr)
					return false
				}
				if found {
					stake.Voice = &voice
				}
			}
		}
	}
	if stake.Corrections == nil && stake.Voice == nil {
		return true
	}
	reply(w, http.StatusConflict, map[string]any{"error": "reanalysis_confirmation_required", "stake": stake, "stake_at": stakeAt})
	return false
}

// getClipAnalysisHistory lists the recent runs of a clip with the versions each step followed.
func (s *Server) getClipAnalysisHistory(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	if !s.clipExists(w, r, clipID) {
		return
	}
	runs, err := s.Analysis.History(r.Context(), clipID, 20)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"runs": runs})
}

// adminQueueEvents streams the Task Queue: one snapshot of every analysis run and loose
// task, then one per change. The administrator role is rechecked before every snapshot.
func (s *Server) adminQueueEvents(w http.ResponseWriter, r *http.Request) {
	s.Events.Stream(w, r, events.Topic{
		Name:      "admin:queue",
		Authorize: s.adminAuthorizer(r),
		Snapshot: func(ctx context.Context) (any, error) {
			return s.Analysis.Queue(ctx)
		},
		// A client that asks for it (delta=1) is sent what changed, not 3000 runs at each change.
		Delta: func(previous, next any) (any, bool) {
			before, ok := previous.(analysis.Queue)
			after, isQueue := next.(analysis.Queue)
			if !ok || !isQueue {
				return nil, false
			}
			return after.DeltaFrom(before), true
		},
	})
}

func (s *Server) adminAnalysisQueue(w http.ResponseWriter, r *http.Request) {
	q, err := s.Analysis.Queue(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, q)
}

func (s *Server) adminAuthorizer(r *http.Request) func(context.Context) error {
	return func(ctx context.Context) error {
		ss, err := s.findSession(r.WithContext(ctx))
		switch {
		case errors.Is(err, pgx.ErrNoRows), errors.Is(err, http.ErrNoCookie):
			return events.ErrForbidden
		case err != nil:
			return err
		case !isAdministrator(ss.User.Role):
			return events.ErrForbidden
		}
		return nil
	}
}

// adminAnalysisHistory pages the finished runs of every clip: the durable record behind the
// Task Queue's Clip History, with the steps and failures of each run.
func (s *Server) adminAnalysisHistory(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
		page = n
	}
	h, err := s.Analysis.FinishedRuns(r.Context(), strings.TrimSpace(r.URL.Query().Get("q")), page)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, h)
}

// refuseWhileAnalyzing answers 409 analysis_active and returns true when an analysis run is
// in progress on the clip: an isolated step must not race the orchestrator, which owns the
// order of the steps while it runs.
func (s *Server) refuseWhileAnalyzing(w http.ResponseWriter, r *http.Request, clipID int64) bool {
	if clipID < 1 {
		return false
	}
	var active bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM clip_analysis_runs WHERE clip_id=$1 AND status='active')`, clipID).Scan(&active); err != nil {
		s.internal(w, err)
		return true
	}
	if active {
		fail(w, http.StatusConflict, "analysis_active")
	}
	return active
}

// clipReadable tells whether the session of the request may open the clip: shared, its own, or
// any clip for an administrator (the rule of authorizedClip).
func (s *Server) clipReadable(ctx context.Context, r *http.Request, clipID int64) error {
	ss, err := s.findSession(r.WithContext(ctx))
	switch {
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, http.ErrNoCookie):
		return events.ErrForbidden
	case err != nil:
		return err
	}
	var ok bool
	if err = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clips c WHERE c.id=$1 AND clip_readable(c.id,c.owner_id,c.visibility,$2,$3))`, clipID, ss.User.ID, isAdministrator(ss.User.Role)).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return events.ErrForbidden
	}
	return nil
}

// getClipAnalysisView is the read-only analysis of a clip for every user who can open it: the
// status and the stable reason of each step, no job, Worker or error detail.
func (s *Server) getClipAnalysisView(w http.ResponseWriter, r *http.Request) {
	v, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	run, err := s.Analysis.Latest(r.Context(), v.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"run": run.Public()})
}

// clipAnalysisViewEvents streams the read-only analysis: access to the clip is rechecked
// against the session before every snapshot, so a clip made private stops its stream.
func (s *Server) clipAnalysisViewEvents(w http.ResponseWriter, r *http.Request) {
	v, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	clipID := v.ID
	s.Events.Stream(w, r, events.Topic{
		Name:      "clip:" + strconv.FormatInt(clipID, 10),
		Authorize: func(ctx context.Context) error { return s.clipReadable(ctx, r, clipID) },
		Snapshot: func(ctx context.Context) (any, error) {
			run, err := s.Analysis.Latest(ctx, clipID)
			return map[string]any{"run": run.Public()}, err
		},
	})
}
