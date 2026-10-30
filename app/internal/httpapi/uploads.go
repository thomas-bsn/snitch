package httpapi

import (
	"errors"
	"io"
	"net/http"
	"os"

	"snitch/internal/library"
	"snitch/internal/uploads"
)

type createUploadRequest struct {
	Filename         string  `json:"filename"`
	SizeBytes        int64   `json:"size_bytes"`
	Visibility       string  `json:"visibility"`
	Destination      *string `json:"destination"`
	OverwriteVersion string  `json:"overwrite_version"`
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
	var source int64
	var relative, version string
	if request.Destination != nil {
		var ok bool
		source, relative, version, ok = s.uploadDestination(w, r, request.Filename, *request.Destination, request.OverwriteVersion)
		if !ok {
			return
		}
	}
	item, err := s.Uploads.Create(r.Context(), ss.User.ID, request.Filename, request.Visibility, request.SizeBytes)
	if err != nil {
		s.uploadFailure(w, err)
		return
	}
	if request.Destination != nil {
		if _, err = s.Pool.Exec(r.Context(), `INSERT INTO upload_destinations(upload_id,source_id,relative_path,expected_version) VALUES($1,$2,$3,$4)`, item.ID, source, relative, version); err != nil {
			_ = s.Uploads.Cancel(r.Context(), ss.User.ID, item.ID)
			s.internal(w, err)
			return
		}
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
	if item.ClipID != nil {
		if err := s.publishUpload(r.Context(), item.ID, *item.ClipID); err != nil {
			s.recordPublicationFailure(r.Context(), item.ID, err)
			s.uploadFailure(w, err)
			return
		}
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
	case errors.Is(err, os.ErrPermission):
		fail(w, http.StatusUnprocessableEntity, "video_root_permission")
	case errors.Is(err, os.ErrNotExist):
		fail(w, http.StatusUnprocessableEntity, "video_root_unavailable")
	case errors.Is(err, library.ErrInvalidSource):
		fail(w, http.StatusUnprocessableEntity, "video_path_unavailable")
	default:
		s.internal(w, err)
	}
}
