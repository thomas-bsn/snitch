package httpapi

import (
	"net/http"

	"snitch/internal/jobs"
	"snitch/internal/workers"
)

// voiceInventoryWorker is what one Worker really reports about the voice models, never a typed value.
type voiceInventoryWorker struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	// VoiceState: ready (holds the active models and passed its synthetic inference), untested
	// (holds them, no inference proof yet), unavailable (a test or download failed, see
	// cause_code) or models_missing (declares no active model).
	VoiceState             string         `json:"voice_state"`
	CauseCode              string         `json:"cause_code"`
	Diarization            *workers.Model `json:"diarization"`
	Fingerprint            *workers.Model `json:"fingerprint"`
	HoldsActiveDiarization bool           `json:"holds_active_diarization"`
	HoldsActiveFingerprint bool           `json:"holds_active_fingerprint"`
}

type voiceInventory struct {
	Provider     string `json:"provider"`
	ActiveModels struct {
		Diarization string `json:"diarization"`
		Fingerprint string `json:"fingerprint"`
	} `json:"active_models"`
	WorkersTotal      int `json:"workers_total"`
	WorkersVoiceReady int `json:"workers_voice_ready"`
	// PendingVoiceJobs counts voice jobs waiting while no Worker is ready; zero otherwise.
	PendingVoiceJobs int                    `json:"pending_voice_jobs"`
	Workers          []voiceInventoryWorker `json:"workers"`
}

func (s *Server) getVoiceOverview(w http.ResponseWriter, r *http.Request) {
	_, pipeline, err := s.activeVoicePipeline(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	items, err := s.Workers.List(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	live := items[:0]
	for _, item := range items {
		if item.RevokedAt == nil {
			live = append(live, item)
		}
	}
	if err = s.withInstallProgress(r, live, false); err != nil {
		s.internal(w, err)
		return
	}
	out := voiceInventory{Provider: pipeline.Provider, Workers: []voiceInventoryWorker{}}
	out.ActiveModels.Diarization = pipeline.Model
	out.ActiveModels.Fingerprint = jobs.ActiveEmbeddingModel
	// The fake provider runs no model: nothing to hold.
	real := pipeline.Provider == "pyannote"
	for _, item := range live {
		row := voiceInventoryWorker{
			ID: item.ID, Name: item.Name, Connected: item.Connected,
			Diarization: item.Models.Diarization, Fingerprint: item.Models.Fingerprint,
			HoldsActiveDiarization: !real || (item.Models.Diarization != nil && item.Models.Diarization.Model == pipeline.Model),
			HoldsActiveFingerprint: !real || (item.Models.Fingerprint != nil && item.Models.Fingerprint.Model == jobs.ActiveEmbeddingModel),
		}
		voice := workers.InstallVoice{State: "pending"}
		if item.Install != nil {
			voice = item.Install.Voice
		}
		switch {
		case !row.HoldsActiveDiarization || !row.HoldsActiveFingerprint:
			row.VoiceState = "models_missing"
		case voice.State == "unavailable":
			row.VoiceState, row.CauseCode = "unavailable", voice.CauseCode
		case voice.State == "ready":
			row.VoiceState = "ready"
		default:
			row.VoiceState = "untested"
		}
		if row.VoiceState == "ready" && row.Connected {
			out.WorkersVoiceReady++
		}
		out.Workers = append(out.Workers, row)
	}
	out.WorkersTotal = len(live)
	if out.WorkersVoiceReady == 0 {
		if err = s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM jobs WHERE status='pending' AND step LIKE 'voice.%'`).Scan(&out.PendingVoiceJobs); err != nil {
			s.internal(w, err)
			return
		}
	}
	reply(w, http.StatusOK, out)
}
