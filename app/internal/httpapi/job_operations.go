package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"snitch/internal/jobs"
)

func parseJobID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_job_id")
		return 0, false
	}
	return id, true
}

func (s *Server) jobHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := parseJobID(w, r)
	if !ok {
		return
	}
	events, err := s.Jobs.History(r.Context(), id)
	if errors.Is(err, jobs.ErrJobNotFound) {
		fail(w, http.StatusNotFound, "job_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": events})
}

func (s *Server) relaunchJob(w http.ResponseWriter, r *http.Request) {
	id, ok := parseJobID(w, r)
	if !ok {
		return
	}
	var clipID int64
	var step, status string
	if s.Pool.QueryRow(r.Context(), `SELECT clip_id,step,status FROM jobs WHERE id=$1`, id).Scan(&clipID, &step, &status) == nil && status == "dead" && isTranscriptionStep(step) && !s.correctionsGuard(w, r, clipID) {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	job, err := s.Jobs.Relaunch(r.Context(), id, ss.User.ID)
	if errors.Is(err, jobs.ErrJobNotFound) {
		fail(w, http.StatusNotFound, "job_not_found")
		return
	}
	if errors.Is(err, jobs.ErrNotRelaunchable) {
		fail(w, http.StatusConflict, "job_not_dead")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	s.syncIncidents(r.Context())
	reply(w, http.StatusOK, job)
}

func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) {
	id, ok := parseJobID(w, r)
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	err := s.Jobs.Cancel(r.Context(), id, ss.User.ID)
	if errors.Is(err, jobs.ErrJobNotFound) {
		fail(w, http.StatusNotFound, "job_not_found")
		return
	}
	if errors.Is(err, jobs.ErrNotCancellable) {
		fail(w, http.StatusConflict, "job_not_cancellable")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	s.syncIncidents(r.Context())
	w.WriteHeader(http.StatusNoContent)
}
