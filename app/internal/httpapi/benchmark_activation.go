package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
)

// Activating a model from a benchmark (#287, ADR 0062). It only changes the model new work will use. It never
// schedules a reanalysis, never touches a human correction or validation, and needs an explicit confirmation of the
// model it replaces. Reanalysing the library stays a separate, explicit action of the screens that already own it.
// The fingerprint model changes through its own transition (ADR 0044) and is refused here.

type activationPreview struct {
	Stage          string `json:"stage"`
	CurrentModel   string `json:"current_model"`
	TargetModel    string `json:"target_model"`
	TargetLabel    string `json:"target_label"`
	AlreadyActive  bool   `json:"already_active"`
	WorkersReady   int    `json:"workers_ready"`
	Clips          int    `json:"clips_with_analysis"`
	CorrectedClips int    `json:"clips_with_human_work"`
	CorrectedItems int    `json:"human_items"`
	Reanalysis     string `json:"reanalysis"`
}

func (s *Server) registerBenchmarkActivationRoutes(mux *http.ServeMux) {
	const base = "/api/admin/benchmarks/activation"
	mux.Handle("GET "+base, s.protected(true, http.HandlerFunc(s.previewBenchmarkActivation)))
	mux.Handle("POST "+base, s.protected(true, http.HandlerFunc(s.confirmBenchmarkActivation)))
}

// activeBenchmarkModel is the model production uses for the stage, in the id the benchmark results use.
func (s *Server) activeBenchmarkModel(ctx context.Context, stage string) (string, error) {
	var model string
	var err error
	switch stage {
	case "transcription":
		err = s.Pool.QueryRow(ctx, `SELECT COALESCE(v.config->>'model','') FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&model)
	case "diarization":
		// The catalogue id of the model the voice pipeline runs; empty when it is not a catalogue model.
		err = s.Pool.QueryRow(ctx, `SELECT COALESCE((SELECT c.id FROM voice_model_catalog c WHERE c.kind='diarization' AND c.repo=v.config->>'model'),'')
			FROM voice_pipeline_active a JOIN voice_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&model)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return model, err
}

func (s *Server) previewBenchmarkActivation(w http.ResponseWriter, r *http.Request) {
	stage, target := r.URL.Query().Get("stage"), r.URL.Query().Get("model")
	preview, status, code := s.buildActivationPreview(r, stage, target)
	if code != "" {
		fail(w, status, code)
		return
	}
	reply(w, http.StatusOK, preview)
}

func (s *Server) buildActivationPreview(r *http.Request, stage, target string) (activationPreview, int, string) {
	ctx := r.Context()
	p := activationPreview{Stage: stage, TargetModel: target, TargetLabel: target}
	switch stage {
	case "transcription":
		if !jobs.ValidTranscriptionModelName(target) {
			return p, http.StatusUnprocessableEntity, "benchmark_model_invalid"
		}
		p.Reanalysis = "/admin/transcription"
		// The existing batch preview says how many clips a reanalysis would redo and how many carry human corrections
		// (excluded by default): the numbers the administrator reads before deciding anything.
		batch, err := s.previewTranscriptionBatch(r.Clone(ctx), false)
		if err != nil && !errors.Is(err, errBatchFilter) {
			return p, http.StatusServiceUnavailable, "unavailable"
		}
		p.Clips, p.CorrectedClips, p.CorrectedItems = batch.Eligible+batch.AlreadyProcessed+batch.Corrected, batch.Corrected, batch.CorrectedSegments
	case "diarization":
		entry, ok, err := voiceCatalogByID(ctx, s.Pool, target)
		if err != nil {
			return p, http.StatusServiceUnavailable, "unavailable"
		}
		if !ok || !entry.choosable() {
			return p, http.StatusUnprocessableEntity, "benchmark_model_invalid"
		}
		p.TargetLabel = entry.Repo
		p.Reanalysis = "/admin/voice"
		if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM worker_voice_models wm JOIN voice_model_catalog c ON c.id=wm.model_id JOIN workers w ON w.id=wm.worker_id AND w.revoked_at IS NULL
			WHERE wm.model_id=$1 AND wm.state='ready' AND NOT wm.removal_requested AND `+jobs.RuntimeReadyClause("wm", "c"), target).Scan(&p.WorkersReady); err != nil {
			return p, http.StatusServiceUnavailable, "unavailable"
		}
		if err = s.Pool.QueryRow(ctx, `SELECT count(DISTINCT ls.clip_id),count(DISTINCT ls.clip_id) FILTER (WHERE EXISTS(SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.validation_state='validated'))
			FROM local_speakers ls WHERE ls.kind='diarized'`).Scan(&p.Clips, &p.CorrectedClips); err != nil {
			return p, http.StatusServiceUnavailable, "unavailable"
		}
		if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM speaker_attributions WHERE validation_state='validated'`).Scan(&p.CorrectedItems); err != nil {
			return p, http.StatusServiceUnavailable, "unavailable"
		}
	default:
		// The fingerprint changes through its transition; a chain is not a model.
		return p, http.StatusUnprocessableEntity, "benchmark_activation_unsupported"
	}
	current, err := s.activeBenchmarkModel(ctx, stage)
	if err != nil {
		return p, http.StatusServiceUnavailable, "unavailable"
	}
	p.CurrentModel = current
	p.AlreadyActive = current == target
	return p, http.StatusOK, ""
}

func (s *Server) confirmBenchmarkActivation(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var input struct {
		Stage           string `json:"stage"`
		Model           string `json:"model"`
		RunID           int64  `json:"run_id"`
		ExpectedCurrent string `json:"expected_current"`
		Confirm         bool   `json:"confirm"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !input.Confirm {
		fail(w, http.StatusBadRequest, "benchmark_activation_confirm_required")
		return
	}
	preview, status, code := s.buildActivationPreview(r, input.Stage, input.Model)
	if code != "" {
		fail(w, status, code)
		return
	}
	// The administrator confirms the model he saw replaced: another change since then is shown again, not overwritten.
	if preview.CurrentModel != input.ExpectedCurrent || preview.AlreadyActive {
		fail(w, http.StatusConflict, "benchmark_activation_stale")
		return
	}
	var version string
	switch input.Stage {
	case "transcription":
		tx, err := s.Pool.Begin(r.Context())
		if err != nil {
			s.internal(w, err)
			return
		}
		defer tx.Rollback(r.Context())
		var raw []byte
		if err = tx.QueryRow(r.Context(), `SELECT v.config FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true FOR UPDATE OF a`).Scan(&raw); err != nil {
			s.internal(w, err)
			return
		}
		var p transcriptionPipeline
		if err = json.Unmarshal(raw, &p); err != nil {
			s.internal(w, err)
			return
		}
		p.Model = input.Model // the language, the preparation and the speech detection stay
		if !validPipeline(&p) {
			fail(w, http.StatusBadRequest, "invalid_transcription_pipeline")
			return
		}
		if version, err = activatePipeline(r.Context(), tx, p); err != nil {
			s.internal(w, err)
			return
		}
		if err = recordActivation(r.Context(), tx, input.Stage, preview.CurrentModel, input.Model, input.RunID, ss.User.ID, ss.User.Username); err != nil {
			s.internal(w, err)
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			s.internal(w, err)
			return
		}
	case "diarization":
		entry, _, err := voiceCatalogByID(r.Context(), s.Pool, input.Model)
		if err != nil {
			s.internal(w, err)
			return
		}
		// The thresholds stay; only the model changes, and the model version is the pinned revision.
		version, _, err = s.switchVoicePipeline(r.Context(), &entry, func(current voicePipeline) voicePipeline {
			current.Provider, current.Model, current.ModelVersion, current.Fake = entry.Engine, entry.Repo, entry.Revision, nil
			return current
		})
		if errors.Is(err, errVoiceModelNotReady) {
			fail(w, http.StatusConflict, "voice_model_not_ready")
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
		if err = recordActivation(r.Context(), s.Pool, input.Stage, preview.CurrentModel, input.Model, input.RunID, ss.User.ID, ss.User.Username); err != nil {
			s.internal(w, err)
			return
		}
	}
	reply(w, http.StatusOK, map[string]any{"version": version, "stage": input.Stage, "model": input.Model, "reanalysis": preview.Reanalysis, "clips_with_human_work": preview.CorrectedClips})
}

func recordActivation(ctx context.Context, db pgQuerier, stage, from, to string, runID, authorID int64, author string) error {
	var run any
	if runID > 0 {
		run = runID
	}
	_, err := db.Exec(ctx, `INSERT INTO benchmark_activations(stage,from_model,to_model,run_id,author_id,author_name) VALUES($1,$2,$3,(SELECT id FROM benchmark_runs WHERE id=$4),$5,$6)`, stage, from, to, run, authorID, author)
	return err
}
