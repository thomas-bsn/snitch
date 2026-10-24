package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"receipts/internal/library"
)

func (s *Server) listSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.Library.ListSources(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"allowed_roots": s.Library.AllowedRoots(), "items": sources})
}

func (s *Server) createSource(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if !decode(w, r, &input) {
		return
	}
	if len(s.Library.AllowedRoots()) == 0 {
		fail(w, http.StatusServiceUnavailable, "source_roots_unconfigured")
		return
	}
	current, ok := r.Context().Value(sessionKey{}).(session)
	if !ok {
		fail(w, http.StatusUnauthorized, "session_required")
		return
	}
	source, err := s.Library.CreateSource(r.Context(), current.User.ID, input.Name, input.Path)
	switch {
	case errors.Is(err, library.ErrInvalidSource):
		fail(w, http.StatusBadRequest, "source_path_outside_allowed_roots")
	case errors.Is(err, library.ErrUnavailable):
		fail(w, http.StatusUnprocessableEntity, "source_path_unavailable")
	case errors.Is(err, library.ErrSourceExists):
		fail(w, http.StatusConflict, "source_already_exists")
	case err != nil:
		s.internal(w, err)
	default:
		reply(w, http.StatusCreated, source)
	}
}

func (s *Server) disableSource(w http.ResponseWriter, r *http.Request) {
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	err := s.Library.DisableSource(r.Context(), id)
	if errors.Is(err, library.ErrNotFound) {
		fail(w, http.StatusNotFound, "source_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]string{"status": "disabled"})
}

func (s *Server) sourceFiles(w http.ResponseWriter, r *http.Request) {
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	if _, err := s.Library.GetSource(r.Context(), id); errors.Is(err, library.ErrNotFound) {
		fail(w, http.StatusNotFound, "source_not_found")
		return
	} else if err != nil {
		s.internal(w, err)
		return
	}
	files, err := s.Library.ListFiles(r.Context(), id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": files})
}

func (s *Server) scanSource(w http.ResponseWriter, r *http.Request) {
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	if _, err := s.Library.GetSource(r.Context(), id); errors.Is(err, library.ErrNotFound) {
		fail(w, http.StatusNotFound, "source_not_found")
		return
	} else if err != nil {
		s.internal(w, err)
		return
	}
	var empty struct{}
	if !decode(w, r, &empty) {
		return
	}
	s.Library.ScheduleScan(id, 0)
	reply(w, http.StatusAccepted, map[string]string{"status": "scheduled"})
}

func sourceID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || id <= 0 {
		fail(w, http.StatusBadRequest, "invalid_source_id")
		return 0, false
	}
	return id, true
}
