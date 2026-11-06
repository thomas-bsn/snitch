package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"snitch/internal/jobs"
)

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, pageSize := 1, 24
	var err error
	if raw := query.Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 || page > 100000 {
			fail(w, http.StatusBadRequest, "invalid_pagination")
			return
		}
	}
	if raw := query.Get("page_size"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil || pageSize < 1 || pageSize > 100 {
			fail(w, http.StatusBadRequest, "invalid_pagination")
			return
		}
	}
	clipID := int64(0)
	if raw := query.Get("clip_id"); raw != "" {
		clipID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || clipID < 1 {
			fail(w, http.StatusBadRequest, "invalid_job_filter")
			return
		}
	}
	pageResult, err := s.Jobs.List(r.Context(), jobs.Filter{Workflow: query.Get("workflow"), Status: jobs.Status(query.Get("status")), ClipID: clipID, Step: query.Get("step"), Page: page, PageSize: pageSize})
	if errors.Is(err, jobs.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_job_filter")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, pageResult)
}

func (s *Server) scheduleJob(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ClipID       int64  `json:"clip_id"`
		Step         string `json:"step"`
		InputVersion string `json:"input_version"`
		Priority     int    `json:"priority"`
		MaxAttempts  int    `json:"max_attempts"`
		// RequestedModelID asks for one catalogue fingerprint model (benchmark); the active model otherwise.
		RequestedModelID string `json:"requested_model_id"`
	}
	if !decode(w, r, &input) {
		return
	}
	if s.refuseWhileAnalyzing(w, r, input.ClipID) {
		return
	}
	if isTranscriptionStep(input.Step) && input.ClipID > 0 && !s.correctionsGuard(w, r, input.ClipID) {
		return
	}
	job, created, err := s.Jobs.Schedule(r.Context(), jobs.ScheduleInput{ClipID: input.ClipID, Step: input.Step, InputVersion: input.InputVersion, Priority: input.Priority, MaxAttempts: input.MaxAttempts, RequestedModelID: input.RequestedModelID})
	switch {
	case errors.Is(err, jobs.ErrInvalidInput):
		fail(w, http.StatusBadRequest, "invalid_job")
	case errors.Is(err, jobs.ErrClipNotFound):
		fail(w, http.StatusNotFound, "clip_not_found")
	case err != nil:
		s.internal(w, err)
	default:
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		reply(w, status, job)
	}
}

// scheduleTranscription schedules the selected global FFmpeg audio stream for a ready clip.
func (s *Server) scheduleTranscription(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	var input struct {
		AudioStreamIndex *int `json:"audio_stream_index"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.AudioStreamIndex != nil && *input.AudioStreamIndex < 0 {
		fail(w, 400, "invalid_audio_stream")
		return
	}
	if s.refuseWhileAnalyzing(w, r, clipID) || !s.correctionsGuard(w, r, clipID) {
		return
	}
	job, created, err := s.scheduleResolvedTranscription(r.Context(), clipID, input.AudioStreamIndex, true)
	if err != nil {
		if err.Error() == "audio selection required" {
			fail(w, http.StatusConflict, "audio_track_selection_required")
			return
		}
		if err.Error() == "audio stream not found" {
			fail(w, 400, "audio_stream_not_found")
			return
		}
		if err.Error() == "clip not ready" {
			fail(w, http.StatusConflict, "clip_not_ready")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 404, "clip_not_found")
			return
		}
		if strings.Contains(err.Error(), "audio stream") {
			fail(w, 400, "audio_stream_not_found")
			return
		}
		s.internal(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	reply(w, status, map[string]any{"status": "scheduled", "job": job})
}
