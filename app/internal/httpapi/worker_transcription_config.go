package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Server) getWorkerTranscriptionConfig(w http.ResponseWriter, r *http.Request) {
	worker := currentWorker(r)
	var mode, endpoint, version string
	err := s.Pool.QueryRow(r.Context(), `SELECT mode,endpoint,version FROM transcription_settings WHERE id=true`).Scan(&mode, &endpoint, &version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		mode = "unconfigured"
	}
	var model, language string
	err = s.Pool.QueryRow(r.Context(), `SELECT v.config->>'model',v.config->>'language' FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&model, &language)
	if err != nil {
		s.internal(w, err)
		return
	}
	if version != "" {
		_, err = s.Pool.Exec(r.Context(), `INSERT INTO worker_transcription_config(worker_id,received_version) VALUES($1,$2) ON CONFLICT(worker_id) DO UPDATE SET received_version=EXCLUDED.received_version,updated_at=now()`, worker.ID, version)
		if err != nil {
			s.internal(w, err)
			return
		}
	}
	var testRequested bool
	var endpointOverride string
	err = s.Pool.QueryRow(r.Context(), `SELECT COALESCE(test_requested_at>tested_at,test_requested_at IS NOT NULL), COALESCE(endpoint_override,'') FROM worker_transcription_config WHERE worker_id=$1`, worker.ID).Scan(&testRequested, &endpointOverride)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	// Determine applicable endpoint: override takes priority
	applicableEndpoint := endpoint
	if endpointOverride != "" {
		applicableEndpoint = endpointOverride
	}
	reply(w, http.StatusOK, map[string]any{"mode": mode, "endpoint": applicableEndpoint, "model": model, "language": language, "version": version, "test_requested": testRequested, "endpoint_override": endpointOverride})
}

func (s *Server) postWorkerTranscriptionStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Version   string `json:"version"`
		Active    bool   `json:"active"`
		ErrorCode string `json:"error_code"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Version == "" || (input.Active && input.ErrorCode != "") || (!input.Active && input.ErrorCode != "connection_failed" && input.ErrorCode != "model_missing") {
		fail(w, http.StatusBadRequest, "invalid_transcription_config_status")
		return
	}
	worker := currentWorker(r)
	var desired string
	err := s.Pool.QueryRow(r.Context(), `SELECT version FROM transcription_settings WHERE id=true AND mode='enabled'`).Scan(&desired)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && desired != input.Version) {
		fail(w, http.StatusConflict, "stale_transcription_config")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	tag, err := s.Pool.Exec(r.Context(), `UPDATE worker_transcription_config SET tested_version=$2,error_code=$3,active_version=CASE WHEN $4 THEN $2 ELSE active_version END,tested_at=now(),updated_at=now() WHERE worker_id=$1 AND received_version=$2`, worker.ID, input.Version, input.ErrorCode, input.Active)
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() != 1 {
		fail(w, http.StatusConflict, "stale_transcription_config")
		return
	}
	reply(w, http.StatusOK, map[string]any{"received_version": input.Version, "tested_version": input.Version, "active_version": func() string {
		if input.Active {
			return input.Version
		}
		return ""
	}(), "error_code": input.ErrorCode})
}

func (s *Server) requestWorkerTranscriptionTest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var exists bool
	err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, id).Scan(&exists)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	_, err = s.Pool.Exec(r.Context(), `INSERT INTO worker_transcription_config(worker_id,test_requested_at) VALUES($1,now()) ON CONFLICT(worker_id) DO UPDATE SET test_requested_at=now(),updated_at=now()`, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"test_requested": true})
}

// Admin endpoints for worker-specific transcription endpoint override

func (s *Server) putWorkerTranscriptionOverride(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var exists bool
	err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, id).Scan(&exists)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	var input struct {
		EndpointOverride string `json:"endpoint_override"`
	}
	if !decode(w, r, &input) {
		return
	}
	input.EndpointOverride = strings.TrimSpace(input.EndpointOverride)
	if input.EndpointOverride != "" && !validWhisperURL(input.EndpointOverride) {
		fail(w, http.StatusBadRequest, "invalid_transcription_endpoint")
		return
	}
	_, err = s.Pool.Exec(r.Context(), `INSERT INTO worker_transcription_config(worker_id,endpoint_override) VALUES($1,$2) ON CONFLICT(worker_id) DO UPDATE SET endpoint_override=EXCLUDED.endpoint_override,updated_at=now()`, id, input.EndpointOverride)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"endpoint_override": input.EndpointOverride})
}

func (s *Server) deleteWorkerTranscriptionOverride(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var exists bool
	err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, id).Scan(&exists)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	_, err = s.Pool.Exec(r.Context(), `UPDATE worker_transcription_config SET endpoint_override='',updated_at=now() WHERE worker_id=$1`, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"endpoint_override": ""})
}