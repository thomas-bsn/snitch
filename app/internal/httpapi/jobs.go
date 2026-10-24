package httpapi

import (
	"errors"
	"net/http"
	"strconv"

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
	}
	if !decode(w, r, &input) {
		return
	}
	job, created, err := s.Jobs.Schedule(r.Context(), jobs.ScheduleInput{ClipID: input.ClipID, Step: input.Step, InputVersion: input.InputVersion, Priority: input.Priority})
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
