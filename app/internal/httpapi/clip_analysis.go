package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/analysis"
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

func (h analysisHooks) Transcribe(ctx context.Context, clipID int64, since time.Time) (analysis.Transcription, error) {
	plan, err := h.s.planTranscription(ctx, clipID, nil, true)
	if err != nil {
		return analysis.Transcription{}, err
	}
	var jobID int64
	err = h.s.Pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step=$2 AND input_version=$3 AND (status IN ('pending','leased','retry_wait','done') OR created_at>=$4) ORDER BY id DESC LIMIT 1`, clipID, plan.step, plan.version, since).Scan(&jobID)
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
	} else if found {
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

func (s *Server) startClipAnalysis(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	var input struct {
		Scope string `json:"scope"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Scope == "" {
		input.Scope = analysis.ScopeTranscription
	}
	ss := r.Context().Value(sessionKey{}).(session)
	run, created, err := s.Analysis.Start(r.Context(), clipID, input.Scope, ss.User.ID)
	switch {
	case errors.Is(err, analysis.ErrInvalidScope):
		fail(w, http.StatusBadRequest, "invalid_analysis_scope")
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

func (s *Server) getClipAnalysis(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	var exists bool
	if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM clips WHERE id=$1)`, clipID).Scan(&exists); err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "clip_not_found")
		return
	}
	run, err := s.Analysis.Latest(r.Context(), clipID)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"run": run})
}
