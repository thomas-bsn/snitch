package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"receipts/internal/jobs"
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
	pageResult, err := s.Jobs.List(r.Context(), jobs.Filter{Status: jobs.Status(query.Get("status")), ClipID: clipID, Step: query.Get("step"), Page: page, PageSize: pageSize})
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
	}
	if !decode(w, r, &input) {
		return
	}
	job, created, err := s.Jobs.Schedule(r.Context(), jobs.ScheduleInput{ClipID: input.ClipID, Step: input.Step, InputVersion: input.InputVersion, Priority: input.Priority, MaxAttempts: input.MaxAttempts})
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
		AudioStreamIndex int `json:"audio_stream_index"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.AudioStreamIndex < 0 {
		fail(w, http.StatusBadRequest, "invalid_audio_stream")
		return
	}
	var ready bool
	var rawTracks []byte
	err = s.Pool.QueryRow(r.Context(), `SELECT c.tracks, (EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready')) FROM clips c WHERE c.id=$1`, clipID).Scan(&rawTracks, &ready)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "clip_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if !ready {
		fail(w, http.StatusConflict, "clip_not_ready")
		return
	}
	var tracks []struct {
		Index int    `json:"index"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(rawTracks, &tracks); err != nil {
		s.internal(w, err)
		return
	}
	found := false
	for _, track := range tracks {
		if track.Index == input.AudioStreamIndex && track.Type == "audio" {
			found = true
			break
		}
	}
	if !found {
		fail(w, http.StatusBadRequest, "audio_stream_not_found")
		return
	}
	version := fmt.Sprintf("transcription-v1-audio-%d", input.AudioStreamIndex)
	job, created, err := s.Jobs.Schedule(r.Context(), jobs.ScheduleInput{ClipID: clipID, Step: fmt.Sprintf("transcription.audio.%d", input.AudioStreamIndex), InputVersion: version, MaxAttempts: 3})
	if errors.Is(err, jobs.ErrClipNotFound) {
		fail(w, http.StatusNotFound, "clip_not_found")
	} else if errors.Is(err, jobs.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_transcription")
	} else if err != nil {
		s.internal(w, err)
	} else {
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		reply(w, status, map[string]any{"job": job})
	}
}
