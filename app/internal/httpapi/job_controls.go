package httpapi

import (
	"errors"
	"net/http"

	"snitch/internal/jobs"
)

func (s *Server) getJobControls(w http.ResponseWriter, r *http.Request) {
	controls, err := s.Jobs.GetControls(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, controls)
}

func (s *Server) updateJobControls(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Paused         *bool `json:"paused"`
		MaxConcurrency *int  `json:"max_concurrency"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Paused == nil || input.MaxConcurrency == nil {
		fail(w, http.StatusBadRequest, "invalid_job_controls")
		return
	}
	controls, err := s.Jobs.UpdateControls(r.Context(), jobs.ControlsInput{Paused: *input.Paused, MaxConcurrency: *input.MaxConcurrency})
	if errors.Is(err, jobs.ErrInvalidControls) {
		fail(w, http.StatusBadRequest, "invalid_job_controls")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, controls)
}
