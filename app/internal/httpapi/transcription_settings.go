package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type transcriptionSettingsInput struct {
	Mode     string                 `json:"mode"`
	Endpoint string                 `json:"endpoint"`
	Model    string                 `json:"model"`
	Language string                 `json:"language"`
	Profile  *transcriptionPipeline `json:"profile,omitempty"`
}

func (s *Server) getTranscriptionSettings(w http.ResponseWriter, r *http.Request) {
	var mode, endpoint, version string
	err := s.Pool.QueryRow(r.Context(), `SELECT mode,endpoint,version FROM transcription_settings WHERE id=true`).Scan(&mode, &endpoint, &version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		mode = "unconfigured"
	}
	var profile transcriptionPipeline
	var raw []byte
	err = s.Pool.QueryRow(r.Context(), `SELECT v.config FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&raw)
	if err != nil {
		s.internal(w, err)
		return
	}
	if err = json.Unmarshal(raw, &profile); err != nil {
		s.internal(w, err)
		return
	}
	status := mode
	if mode == "enabled" {
		status = "untested"
	}
	reply(w, http.StatusOK, map[string]any{"mode": mode, "status": status, "endpoint": endpoint, "model": profile.Model, "language": profile.Language, "version": version})
}

func (s *Server) putTranscriptionSettings(w http.ResponseWriter, r *http.Request) {
	var input transcriptionSettingsInput
	if !decode(w, r, &input) {
		return
	}
	input.Mode = strings.TrimSpace(input.Mode)
	input.Endpoint = strings.TrimSpace(input.Endpoint)
	input.Model = strings.TrimSpace(input.Model)
	input.Language = strings.TrimSpace(input.Language)
	if input.Mode != "enabled" && input.Mode != "disabled" {
		fail(w, http.StatusBadRequest, "invalid_transcription_settings")
		return
	}
	if input.Mode == "enabled" && (!validWhisperURL(input.Endpoint) || !transcriptionModelPattern.MatchString(input.Model) || !transcriptionLanguagePattern.MatchString(input.Language)) {
		fail(w, http.StatusBadRequest, "invalid_transcription_settings")
		return
	}
	if input.Mode == "disabled" {
		input.Endpoint = ""
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var raw []byte
	err = tx.QueryRow(r.Context(), `SELECT v.config FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true FOR UPDATE OF a`).Scan(&raw)
	if err != nil {
		s.internal(w, err)
		return
	}
	var profile transcriptionPipeline
	if err = json.Unmarshal(raw, &profile); err != nil {
		s.internal(w, err)
		return
	}
	if input.Profile != nil {
		profile = *input.Profile
	}
	if input.Mode == "enabled" {
		profile.Model = input.Model
		profile.Language = input.Language
	}
	if input.Profile != nil && !validPipeline(&profile) {
		fail(w, http.StatusBadRequest, "invalid_transcription_pipeline")
		return
	}
	profileVersion, err := activatePipeline(r.Context(), tx, profile)
	if err != nil {
		s.internal(w, err)
		return
	}
	hash := sha256.Sum256([]byte(input.Mode + "\x00" + input.Endpoint + "\x00" + profileVersion))
	version := "transcription-" + hex.EncodeToString(hash[:12])
	_, err = tx.Exec(r.Context(), `INSERT INTO transcription_settings(id,mode,endpoint,version) VALUES(true,$1,$2,$3) ON CONFLICT(id) DO UPDATE SET mode=EXCLUDED.mode,endpoint=EXCLUDED.endpoint,version=EXCLUDED.version,updated_at=now()`, input.Mode, input.Endpoint, version)
	if err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	status := input.Mode
	if input.Mode == "enabled" {
		status = "untested"
	}
	reply(w, http.StatusOK, map[string]any{"mode": input.Mode, "status": status, "endpoint": input.Endpoint, "model": profile.Model, "language": profile.Language, "version": version})
}
