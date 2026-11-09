package httpapi

import (
	"context"
	"fmt"
	"net/http"

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
	// PendingVoiceJobs counts voice jobs waiting while no Worker is ready (real provider only); zero otherwise.
	PendingVoiceJobs int                    `json:"pending_voice_jobs"`
	Workers          []voiceInventoryWorker `json:"workers"`
}

func (s *Server) getVoiceOverview(w http.ResponseWriter, r *http.Request) {
	out, err := s.voiceInventory(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	// Only the steps Claim gates on a model can be blocked: the synthetic provider never waits.
	if out.real && out.WorkersVoiceReady == 0 {
		if err = s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM jobs WHERE status='pending' AND (step LIKE 'voice.diarization.%' OR step LIKE 'voice.fingerprint.%')`).Scan(&out.PendingVoiceJobs); err != nil {
			s.internal(w, err)
			return
		}
	}
	reply(w, http.StatusOK, out)
}

// voiceInventory is what the Workers report about the voice models; real tells whether the active
// provider runs a local model (the synthetic one never waits for a Worker).
func (s *Server) voiceInventory(ctx context.Context) (voiceInventoryView, error) {
	_, pipeline, err := s.activeVoicePipeline(ctx)
	if err != nil {
		return voiceInventoryView{}, err
	}
	items, err := s.Workers.List(ctx)
	if err != nil {
		return voiceInventoryView{}, err
	}
	live := items[:0]
	for _, item := range items {
		if item.RevokedAt == nil {
			live = append(live, item)
		}
	}
	if err = s.withInstallProgressCtx(ctx, live, false); err != nil {
		return voiceInventoryView{}, err
	}
	proven, err := s.workersWithCompletedVoiceJob(ctx)
	if err != nil {
		return voiceInventoryView{}, err
	}
	out := voiceInventoryView{voiceInventory: voiceInventory{Provider: pipeline.Provider, Workers: []voiceInventoryWorker{}}}
	out.ActiveModels.Diarization = pipeline.Model
	activeFingerprint, err := voiceActiveFingerprintRepo(ctx, s.Pool)
	if err != nil {
		return voiceInventoryView{}, err
	}
	out.ActiveModels.Fingerprint = activeFingerprint
	// The fake provider runs no model: nothing to hold.
	out.real = voiceRealProvider(pipeline.Provider)
	real := out.real
	for _, item := range live {
		row := voiceInventoryWorker{
			ID: item.ID, Name: item.Name, Connected: item.Connected,
			Diarization: item.Models.Diarization, Fingerprint: item.Models.Fingerprint,
			HoldsActiveDiarization: !real || (item.Models.Diarization != nil && item.Models.Diarization.Model == pipeline.Model),
			HoldsActiveFingerprint: !real || (item.Models.Fingerprint != nil && item.Models.Fingerprint.Model == activeFingerprint),
		}
		voice := workers.InstallVoice{State: "pending"}
		if item.Install != nil {
			voice = item.Install.Voice
		}
		// The installation test is only one proof: a voice job really completed by this Worker is another, so
		// a missing, trimmed or reset event does not hide a Worker that works. A failed test is never overridden.
		if voice.State == "pending" && proven[item.ID] {
			voice = workers.InstallVoice{State: "ready"}
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
	return out, nil
}

// voiceInventoryView carries the unexported provider kind next to the JSON contract.
type voiceInventoryView struct {
	voiceInventory
	real bool
}

// workersWithCompletedVoiceJob lists the Workers that completed a local voice job (diarization or fingerprint).
func (s *Server) workersWithCompletedVoiceJob(ctx context.Context) (map[string]bool, error) {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT completed_by_worker_id FROM jobs WHERE status='done' AND completed_by_worker_id IS NOT NULL AND (step LIKE 'voice.diarization.%' OR step LIKE 'voice.fingerprint.%')`)
	if err != nil {
		return nil, fmt.Errorf("read workers with a completed voice job: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan worker with a completed voice job: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}
