package httpapi

import (
	"errors"
	"net/http"

	"snitch/internal/workers"
)

// postWorkerInstallEvents receives installation events from the launcher, authenticated by the Worker token.
// An event holds a phase, a status, a stable cause code and its original timestamp: no free text, no secret.
// Events queued offline arrive later with their original timestamp.
func (s *Server) postWorkerInstallEvents(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Events []workers.InstallEventInput `json:"events"`
	}
	if !decode(w, r, &input) {
		return
	}
	err := s.Workers.RecordInstallEvents(r.Context(), currentWorker(r).ID, input.Events)
	if errors.Is(err, workers.ErrInvalidInstallEvent) {
		fail(w, http.StatusBadRequest, "invalid_install_event")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// withInstallProgress attaches the installation progress to Workers.
func (s *Server) withInstallProgress(r *http.Request, items []workers.Worker, withEvents bool) error {
	progress, err := s.Workers.InstallProgress(r.Context(), items, withEvents)
	if err != nil {
		return err
	}
	for i := range items {
		p := progress[items[i].ID]
		items[i].Install = &p
	}
	return nil
}
