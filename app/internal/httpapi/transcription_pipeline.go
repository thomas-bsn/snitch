package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type transcriptionPipeline struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Language    string `json:"language"`
	Preparation string `json:"preparation"`
	VAD         struct {
		Mode         string  `json:"mode"`
		Version      string  `json:"version"`
		Threshold    float64 `json:"threshold,omitempty"`
		MinSpeechMs  int     `json:"min_speech_ms,omitempty"`
		MinSilenceMs int     `json:"min_silence_ms,omitempty"`
		PaddingMs    int     `json:"padding_ms,omitempty"`
	} `json:"vad"`
}

func (s *Server) pipelineConfig(ctx context.Context, version string) (transcriptionPipeline, error) {
	var raw []byte
	var config transcriptionPipeline
	err := s.Pool.QueryRow(ctx, `SELECT config FROM transcription_pipeline_versions WHERE version=$1`, version).Scan(&raw)
	if err != nil {
		return config, fmt.Errorf("read pipeline configuration: %w", err)
	}
	if err = json.Unmarshal(raw, &config); err != nil {
		return config, fmt.Errorf("decode pipeline configuration: %w", err)
	}
	return config, nil
}

func (s *Server) getTranscriptionPipeline(w http.ResponseWriter, r *http.Request) {
	var version string
	var raw []byte
	err := s.Pool.QueryRow(r.Context(), `SELECT v.version,v.config FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&version, &raw)
	if err != nil {
		s.internal(w, err)
		return
	}
	var config transcriptionPipeline
	if decodeErr := json.Unmarshal(raw, &config); decodeErr != nil {
		s.internal(w, decodeErr)
		return
	}
	reply(w, http.StatusOK, map[string]any{"version": version, "config": config})
}

func (s *Server) putTranscriptionPipeline(w http.ResponseWriter, r *http.Request) {
	var p transcriptionPipeline
	if !decode(w, r, &p) {
		return
	}
	if !validPipeline(&p) {
		fail(w, http.StatusBadRequest, "invalid_transcription_pipeline")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	version, err := activatePipeline(r.Context(), tx, p)
	if err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"version": version, "config": p})
}

func validPipeline(p *transcriptionPipeline) bool {
	p.Provider, p.Model, p.Language, p.Preparation = strings.TrimSpace(p.Provider), strings.TrimSpace(p.Model), strings.TrimSpace(p.Language), strings.TrimSpace(p.Preparation)
	p.VAD.Mode, p.VAD.Version = strings.TrimSpace(p.VAD.Mode), strings.TrimSpace(p.VAD.Version)
	if p.Provider != "openai-compatible" || p.Model == "" || len(p.Model) > 100 || !validTranscriptionLanguage(p.Language) || p.Preparation != "wav-16khz-mono-v1" || (p.VAD.Mode != "off" && p.VAD.Mode != "energy") {
		return false
	}
	if p.VAD.Mode == "energy" {
		if p.VAD.Version == "" || len(p.VAD.Version) > 64 || p.VAD.Threshold <= 0 || p.VAD.Threshold >= 1 || p.VAD.MinSpeechMs < 1 || p.VAD.MinSpeechMs > 5000 || p.VAD.MinSilenceMs < 1 || p.VAD.MinSilenceMs > 10000 || p.VAD.PaddingMs < 0 || p.VAD.PaddingMs > 2000 {
			return false
		}
	} else {
		p.VAD.Version = "vad-off-v1"
		p.VAD.Threshold, p.VAD.MinSpeechMs, p.VAD.MinSilenceMs, p.VAD.PaddingMs = 0, 0, 0, 0
	}
	return true
}

// activatePipeline stores a profile as a (possibly existing) immutable version and makes it the active one.
func activatePipeline(ctx context.Context, tx pgx.Tx, p transcriptionPipeline) (string, error) {
	raw, _ := json.Marshal(p)
	hash := sha256.Sum256(raw)
	version := "pipeline-" + hex.EncodeToString(hash[:12])
	if _, err := tx.Exec(ctx, `INSERT INTO transcription_pipeline_versions(version,config) VALUES($1,$2) ON CONFLICT(version) DO NOTHING`, version, raw); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE transcription_pipeline_active SET version=$1 WHERE id=true`, version); err != nil {
		return "", err
	}
	return version, nil
}
