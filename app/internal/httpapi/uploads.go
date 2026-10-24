package httpapi

import (
	"errors"
	"io"
	"net/http"

	"snitch/internal/uploads"
)

type createUploadRequest struct {
	Filename   string `json:"filename"`
	SizeBytes  int64  `json:"size_bytes"`
	Visibility string `json:"visibility"`
}

func (s *Server) listUploads(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	items, err := s.Uploads.List(r.Context(), ss.User.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createUpload(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var request createUploadRequest
	if !decode(w, r, &request) {
		return
	}
	if request.Visibility == "" {
		request.Visibility = "shared"
	}
	item, err := s.Uploads.Create(r.Context(), ss.User.ID, request.Filename, request.Visibility, request.SizeBytes)
	if err != nil {
		s.uploadFailure(w, err)
		return
	}
	reply(w, http.StatusCreated, item)
}

func (s *Server) uploadChunk(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	start, end, total, err := uploads.ParseContentRange(r.Header.Get("Content-Range"))
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_content_range")
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, http.StatusRequestEntityTooLarge, "chunk_too_large")
		return
	}
	offset, err := s.Uploads.Chunk(r.Context(), ss.User.ID, r.PathValue("id"), start, end, total, data)
	if err != nil {
		s.uploadFailure(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]int64{"offset_bytes": offset})
}

func (s *Server) completeUpload(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	item, err := s.Uploads.Complete(r.Context(), ss.User.ID, r.PathValue("id"))
	if err != nil {
		s.uploadFailure(w, err)
		return
	}
	reply(w, http.StatusCreated, item)
}

func (s *Server) cancelUpload(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	if err := s.Uploads.Cancel(r.Context(), ss.User.ID, r.PathValue("id")); err != nil {
		s.uploadFailure(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) uploadFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, uploads.ErrNotFound):
		fail(w, http.StatusNotFound, "not_found")
	case errors.Is(err, uploads.ErrTooLarge):
		fail(w, http.StatusRequestEntityTooLarge, "upload_too_large")
	case errors.Is(err, uploads.ErrInvalid):
		fail(w, http.StatusBadRequest, "invalid_upload")
	case errors.Is(err, uploads.ErrConflict):
		fail(w, http.StatusConflict, "upload_conflict")
	case errors.Is(err, uploads.ErrExpired):
		fail(w, http.StatusGone, "upload_expired")
	case errors.Is(err, uploads.ErrInvalidVideo):
		fail(w, http.StatusUnprocessableEntity, "invalid_video")
	default:
		s.internal(w, err)
	}
}
