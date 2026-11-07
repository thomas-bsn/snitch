package httpapi

import (
	"context"
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
	return s.withInstallProgressCtx(r.Context(), items, withEvents)
}

func (s *Server) withInstallProgressCtx(ctx context.Context, items []workers.Worker, withEvents bool) error {
	// A deleted Worker keeps no installation detail, only its minimal audit.
	live := make([]workers.Worker, 0, len(items))
	for _, item := range items {
		if item.RevokedAt == nil {
			live = append(live, item)
		}
	}
	progress, err := s.Workers.InstallProgress(ctx, live, withEvents)
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].RevokedAt != nil {
			continue
		}
		p := progress[items[i].ID]
		items[i].Install = &p
	}
	return nil
}
