package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"snitch/internal/library"
)

// « Scanner toute la racine » (#404, ADR 0038 amended): an Administrator asks once; the walk of the disk
// and the queuing happen in the background (library.StartRootScan) and survive a restart. Dossiers
// privés are left out for every role: a Super-administrator scans one by selecting it explicitly.

// previewRootScan counts what a scan of the whole Racine would queue and leave out, without the paths.
func (s *Server) previewRootScan(w http.ResponseWriter, r *http.Request) {
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		fail(w, http.StatusConflict, "video_root_unconfigured")
		return
	}
	tally, err := s.Library.PreviewRootScan(r.Context(), root)
	if r.Context().Err() != nil {
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, tally)
}

func (s *Server) startRootScan(w http.ResponseWriter, r *http.Request) {
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		fail(w, http.StatusConflict, "video_root_unconfigured")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	id, err := s.Library.StartRootScan(r.Context(), root.ID, ss.User.ID)
	if errors.Is(err, library.ErrRootScanActive) {
		fail(w, http.StatusConflict, "index_run_active")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	s.replyRootScan(w, r, http.StatusAccepted, root.ID, id)
}

func (s *Server) currentRootScan(w http.ResponseWriter, r *http.Request) {
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		reply(w, http.StatusOK, map[string]any{"run": nil})
		return
	}
	hidden, err := s.hiddenPaths(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	run, err := s.Library.CurrentRootScan(r.Context(), root.ID, hidden)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"run": run})
}

func (s *Server) cancelRootScan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	err = s.Library.CancelRootScan(r.Context(), id)
	switch {
	case errors.Is(err, library.ErrNotFound):
		fail(w, http.StatusNotFound, "index_run_not_found")
		return
	case errors.Is(err, library.ErrRootScanState):
		fail(w, http.StatusConflict, "index_run_state")
		return
	case err != nil:
		s.internal(w, err)
		return
	}
	var sourceID int64
	if err = s.Pool.QueryRow(r.Context(), `SELECT source_id FROM library_index_runs WHERE id=$1`, id).Scan(&sourceID); err != nil {
		s.internal(w, err)
		return
	}
	s.replyRootScan(w, r, http.StatusOK, sourceID, id)
}

func (s *Server) replyRootScan(w http.ResponseWriter, r *http.Request, status int, sourceID, id int64) {
	hidden, err := s.hiddenPaths(r, sourceID)
	if err != nil {
		s.internal(w, err)
		return
	}
	run, err := s.Library.RootScan(r.Context(), id, hidden)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, status, map[string]any{"run": run})
}
