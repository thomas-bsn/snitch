package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"snitch/internal/workers"
)

// The voice model manager (ADR 0022): the administrator picks a candidate in the catalogue, every Worker
// pulls that desired state (like the transcription configuration, ADR 0020), prepares the model and reports
// what it really holds. Snitch never trusts a typed string to claim a model is available.

func (s *Server) registerVoiceModelRoutes(mux *http.ServeMux) {
	s.registerVoiceModelAddRoutes(mux)
	mux.Handle("GET /api/admin/voice-models", s.protected(true, http.HandlerFunc(s.listVoiceModels)))
	mux.Handle("PATCH /api/admin/voice-models/{id}", s.protectedFor(isSuperAdministrator, http.HandlerFunc(s.patchVoiceModel)))
	mux.Handle("PUT /api/admin/voice-models/candidate", s.protected(true, http.HandlerFunc(s.chooseVoiceModelCandidate)))
	mux.Handle("DELETE /api/admin/voice-models/candidate", s.protected(true, http.HandlerFunc(s.cancelVoiceModelCandidate)))
	mux.Handle("POST /api/admin/voice-models/{id}/activate", s.protected(true, http.HandlerFunc(s.activateVoiceModel)))
	mux.Handle("DELETE /api/admin/voice-models/{id}/installation", s.protected(true, http.HandlerFunc(s.removeVoiceModel)))
	mux.Handle("POST /api/admin/workers/{id}/voice-models/retry", s.protected(true, http.HandlerFunc(s.retryWorkerVoiceModel)))
	mux.Handle("GET /api/worker/voice-models", s.workerOnly(http.HandlerFunc(s.getWorkerVoiceModels)))
	mux.Handle("POST /api/worker/voice-models/status", s.workerOnly(http.HandlerFunc(s.postWorkerVoiceModelStatus)))
}

// activeVoiceModelID is the catalogue id of the active diarization model, empty for the synthetic provider
// or a repository outside the catalogue.
func (s *Server) activeVoiceModelID(r *http.Request) (string, error) {
	_, pipeline, err := s.activeVoicePipeline(r.Context())
	if err != nil {
		return "", err
	}
	if !voiceRealProvider(pipeline.Provider) {
		return "", nil
	}
	entry, _, err := voiceCatalogByRepo(r.Context(), s.Pool, "diarization", pipeline.Model)
	return entry.ID, err
}

func (s *Server) candidateVoiceModelID(r *http.Request) (string, error) {
	var id string
	err := s.Pool.QueryRow(r.Context(), `SELECT model_id FROM voice_model_candidate WHERE id=true`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

type voiceModelWorkerState struct {
	WorkerID      string `json:"worker_id"`
	Name          string `json:"name"`
	Connected     bool   `json:"connected"`
	State         string `json:"state"`
	CauseCode     string `json:"cause_code"`
	BytesReceived int64  `json:"bytes_received"`
	BytesTotal    int64  `json:"bytes_total"`
}

// voiceModelViewOf is the reference itself, without the state of the Workers.
func voiceModelViewOf(entry voiceCatalogEntry) voiceModelView {
	view := voiceModelView{
		ID: entry.ID, Kind: entry.Kind, Engine: entry.Engine, Repo: entry.Repo, Revision: entry.Revision, License: entry.License,
		Gated: entry.Gated, SizeBytes: entry.SizeBytes, SupportsNumSpeakers: entry.SupportsNumSpeakers, Label: entry.Label, VRAMBytes: entry.VRAMBytes,
		Enabled: entry.Enabled, Supported: entry.Supported, UnsupportedReason: entry.UnsupportedReason, Workers: []voiceModelWorkerState{},
	}
	if entry.MaxSpeakers > 0 {
		limit := entry.MaxSpeakers
		view.MaxSpeakers = &limit
	}
	return view
}

type voiceModelView struct {
	ID                  string                  `json:"id"`
	Kind                string                  `json:"kind"`
	Engine              string                  `json:"engine"`
	Repo                string                  `json:"repo"`
	Revision            string                  `json:"revision"`
	License             string                  `json:"license"`
	Gated               bool                    `json:"gated"`
	MaxSpeakers         *int                    `json:"max_speakers"`
	SizeBytes           int64                   `json:"size_bytes"`
	SupportsNumSpeakers bool                    `json:"supports_num_speakers"`
	Active              bool                    `json:"active"`
	Candidate           bool                    `json:"candidate"`
	Installed           bool                    `json:"installed"`
	Label               string                  `json:"label"`
	VRAMBytes           int64                   `json:"vram_bytes"`
	Enabled             bool                    `json:"enabled"`
	Supported           bool                    `json:"supported"`
	UnsupportedReason   string                  `json:"unsupported_reason"`
	Workers             []voiceModelWorkerState `json:"workers"`
}

func (s *Server) listVoiceModels(w http.ResponseWriter, r *http.Request) {
	active, err := s.activeVoiceModelID(r)
	if err != nil {
		s.internal(w, err)
		return
	}
	candidate, err := s.candidateVoiceModelID(r)
	if err != nil {
		s.internal(w, err)
		return
	}
	items, err := s.Workers.List(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	type row struct {
		State, Cause    string
		Received, Total int64
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT worker_id,model_id,state,cause_code,bytes_received,bytes_total FROM worker_voice_models`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	reported := map[[2]string]row{}
	for rows.Next() {
		var workerID, modelID string
		var v row
		if err = rows.Scan(&workerID, &modelID, &v.State, &v.Cause, &v.Received, &v.Total); err != nil {
			s.internal(w, err)
			return
		}
		reported[[2]string{workerID, modelID}] = v
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	out := struct {
		Models []voiceModelView `json:"models"`
	}{Models: []voiceModelView{}}
	catalog, err := voiceCatalogList(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	for _, entry := range catalog {
		view := voiceModelViewOf(entry)
		view.Active, view.Candidate = entry.ID == active, entry.ID == candidate
		for _, item := range items {
			if item.RevokedAt != nil {
				continue
			}
			state := voiceModelWorkerState{WorkerID: item.ID, Name: item.Name, Connected: item.Connected, State: "not_installed"}
			if v, ok := reported[[2]string{item.ID, entry.ID}]; ok {
				state.State, state.CauseCode, state.BytesReceived, state.BytesTotal = v.State, v.Cause, v.Received, v.Total
				view.Installed = view.Installed || v.State == "ready"
			}
			view.Workers = append(view.Workers, state)
		}
		out.Models = append(out.Models, view)
	}
	reply(w, http.StatusOK, out)
}

// pgQuerier is what a query helper needs: the pool or a transaction.
type pgQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

var errVoiceModelNotReady = errors.New("voice model not ready")

// withVoiceModelLock runs one change of the voice model state in a transaction that holds the row of the
// active pipeline: choosing, cancelling, activating, removing and retrying are serialised, so what an
// operation read (active, candidate, ready) is still true when it writes.
func (s *Server) withVoiceModelLock(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var version string
	if err = tx.QueryRow(ctx, `SELECT version FROM voice_pipeline_active WHERE id=true FOR UPDATE`).Scan(&version); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func voiceActiveModelIDIn(ctx context.Context, db pgQuerier) (string, error) {
	var raw []byte
	if err := db.QueryRow(ctx, `SELECT v.config FROM voice_pipeline_active a JOIN voice_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&raw); err != nil {
		return "", err
	}
	var pipeline voicePipeline
	if err := json.Unmarshal(raw, &pipeline); err != nil {
		return "", err
	}
	if !voiceRealProvider(pipeline.Provider) {
		return "", nil
	}
	entry, _, err := voiceCatalogByRepo(ctx, db, "diarization", pipeline.Model)
	return entry.ID, err
}

func voiceCandidateIDIn(ctx context.Context, db pgQuerier) (string, error) {
	var id string
	err := db.QueryRow(ctx, `SELECT model_id FROM voice_model_candidate WHERE id=true`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// voiceModelReadyIn tells whether a Worker that is connected right now holds the model ready.
func voiceModelReadyIn(ctx context.Context, db pgQuerier, modelID string) (bool, error) {
	var ready bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_voice_models m JOIN workers w ON w.id=m.worker_id WHERE m.model_id=$1 AND m.state='ready' AND m.removal_requested=false AND w.revoked_at IS NULL AND w.last_seen_at >= now() - ($2 * interval '1 second'))`,
		modelID, int64(workers.ConnectedWindow.Seconds())).Scan(&ready)
	return ready, err
}

func (s *Server) chooseVoiceModelCandidate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ModelID string `json:"model_id"`
	}
	if !decode(w, r, &input) {
		return
	}
	entry, ok, err := voiceCatalogByID(r.Context(), s.Pool, input.ModelID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !ok || !entry.choosable() {
		fail(w, http.StatusBadRequest, "invalid_voice_model")
		return
	}
	errActive := errors.New("already active")
	err = s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		active, err := voiceActiveModelIDIn(r.Context(), tx)
		if err != nil {
			return err
		}
		if active == entry.ID {
			return errActive
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO voice_model_candidate(id,model_id) VALUES(true,$1) ON CONFLICT(id) DO UPDATE SET model_id=EXCLUDED.model_id,chosen_at=now()`, entry.ID); err != nil {
			return err
		}
		// Choosing a model again withdraws a pending removal request.
		_, err = tx.Exec(r.Context(), `UPDATE worker_voice_models SET removal_requested=false WHERE model_id=$1`, entry.ID)
		return err
	})
	if errors.Is(err, errActive) {
		fail(w, http.StatusConflict, "voice_model_already_active")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]string{"candidate": entry.ID})
}

func (s *Server) cancelVoiceModelCandidate(w http.ResponseWriter, r *http.Request) {
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		_, err := tx.Exec(r.Context(), `DELETE FROM voice_model_candidate WHERE id=true`)
		return err
	})
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// switchVoicePipeline is the single way a catalogue model becomes the active one: the administrator's
// "activate" and a typed pipeline apply the same rule. A real model needs a connected Worker that holds it
// ready; the synthetic provider (entry nil) has no model to hold. build receives the active configuration
// read under the lock, so thresholds are never lost to a concurrent change.
func (s *Server) switchVoicePipeline(ctx context.Context, entry *voiceCatalogEntry, build func(current voicePipeline) voicePipeline) (string, voicePipeline, error) {
	var version string
	var config voicePipeline
	err := s.withVoiceModelLock(ctx, func(tx pgx.Tx) error {
		if entry != nil {
			ready, err := voiceModelReadyIn(ctx, tx, entry.ID)
			if err != nil {
				return err
			}
			if !ready {
				return errVoiceModelNotReady
			}
		}
		var raw []byte
		if err := tx.QueryRow(ctx, `SELECT v.config FROM voice_pipeline_active a JOIN voice_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&raw); err != nil {
			return err
		}
		var current voicePipeline
		if err := json.Unmarshal(raw, &current); err != nil {
			return err
		}
		config = build(current)
		var err error
		if version, err = saveVoicePipelineIn(ctx, tx, config); err != nil {
			return err
		}
		if entry != nil {
			_, err = tx.Exec(ctx, `DELETE FROM voice_model_candidate WHERE model_id=$1`, entry.ID)
		}
		return err
	})
	return version, config, err
}

func (s *Server) activateVoiceModel(w http.ResponseWriter, r *http.Request) {
	entry, ok, err := voiceCatalogByID(r.Context(), s.Pool, r.PathValue("id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	if !ok {
		fail(w, http.StatusNotFound, "voice_model_not_found")
		return
	}
	if !entry.choosable() {
		fail(w, http.StatusBadRequest, "invalid_voice_model")
		return
	}
	// The thresholds stay; only the model changes, and the model version is the pinned revision.
	version, config, err := s.switchVoicePipeline(r.Context(), &entry, func(current voicePipeline) voicePipeline {
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
	reply(w, http.StatusOK, map[string]any{"version": version, "config": config})
}

func (s *Server) removeVoiceModel(w http.ResponseWriter, r *http.Request) {
	entry, ok, err := voiceCatalogByID(r.Context(), s.Pool, r.PathValue("id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	if !ok {
		fail(w, http.StatusNotFound, "voice_model_not_found")
		return
	}
	errInUse := errors.New("in use")
	err = s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		active, err := voiceActiveModelIDIn(r.Context(), tx)
		if err != nil {
			return err
		}
		candidate, err := voiceCandidateIDIn(r.Context(), tx)
		if err != nil {
			return err
		}
		if entry.ID == active || entry.ID == candidate {
			return errInUse
		}
		_, err = tx.Exec(r.Context(), `UPDATE worker_voice_models SET removal_requested=true WHERE model_id=$1`, entry.ID)
		return err
	})
	if errors.Is(err, errInUse) {
		fail(w, http.StatusConflict, "voice_model_in_use")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) retryWorkerVoiceModel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, id).Scan(&exists); err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	errNone := errors.New("no candidate")
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		candidate, err := voiceCandidateIDIn(r.Context(), tx)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(r.Context(), `UPDATE worker_voice_models SET retry_requested=true WHERE worker_id=$1 AND model_id=$2 AND state='failed'`, id, candidate)
		if err != nil {
			return err
		}
		if candidate == "" || tag.RowsAffected() != 1 {
			return errNone
		}
		return nil
	})
	if errors.Is(err, errNone) {
		fail(w, http.StatusConflict, "no_voice_candidate")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]bool{"retry_requested": true})
}

type workerVoiceModel struct {
	ModelID  string `json:"model_id"`
	Engine   string `json:"engine"`
	Repo     string `json:"repo"`
	Revision string `json:"revision"`
	Role     string `json:"role"`
	Retry    bool   `json:"retry"`
}

func (s *Server) getWorkerVoiceModels(w http.ResponseWriter, r *http.Request) {
	worker := currentWorker(r)
	active, err := s.activeVoiceModelID(r)
	if err != nil {
		s.internal(w, err)
		return
	}
	candidate, err := s.candidateVoiceModelID(r)
	if err != nil {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT model_id,retry_requested,removal_requested FROM worker_voice_models WHERE worker_id=$1`, worker.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	retry, removal := map[string]bool{}, []string{}
	for rows.Next() {
		var id string
		var wantRetry, wantRemoval bool
		if err = rows.Scan(&id, &wantRetry, &wantRemoval); err != nil {
			s.internal(w, err)
			return
		}
		retry[id] = wantRetry
		if wantRemoval {
			removal = append(removal, id)
		}
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	out := struct {
		Models []workerVoiceModel `json:"models"`
		Remove []workerVoiceModel `json:"remove"`
	}{Models: []workerVoiceModel{}, Remove: []workerVoiceModel{}}
	add := func(list *[]workerVoiceModel, id, role string) {
		if entry, ok, lookupErr := voiceCatalogByID(r.Context(), s.Pool, id); lookupErr != nil {
			err = lookupErr
		} else if ok {
			*list = append(*list, workerVoiceModel{ModelID: id, Engine: entry.Engine, Repo: entry.Repo, Revision: entry.Revision, Role: role, Retry: retry[id]})
		}
	}
	if candidate != "" {
		add(&out.Models, candidate, "candidate")
	}
	if active != "" && active != candidate {
		add(&out.Models, active, "active")
	}
	for _, id := range removal {
		add(&out.Remove, id, "removal")
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}

func (s *Server) postWorkerVoiceModelStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ModelID       string `json:"model_id"`
		State         string `json:"state"`
		CauseCode     string `json:"cause_code"`
		BytesReceived int64  `json:"bytes_received"`
		BytesTotal    int64  `json:"bytes_total"`
	}
	if !decode(w, r, &input) {
		return
	}
	_, known, err := voiceCatalogByID(r.Context(), s.Pool, input.ModelID)
	if err != nil {
		s.internal(w, err)
		return
	}
	failed := input.State == "failed"
	validState := input.State == "downloading" || input.State == "verifying" || input.State == "ready" || input.State == "removed" || failed
	validCause := (failed && voiceFailureCauses[input.CauseCode]) || (!failed && input.CauseCode == "")
	if !known || !validState || !validCause || input.BytesReceived < 0 || input.BytesTotal < 0 {
		fail(w, http.StatusBadRequest, "invalid_voice_model_status")
		return
	}
	worker := currentWorker(r)
	if input.State == "removed" {
		_, err = s.Pool.Exec(r.Context(), `DELETE FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, worker.ID, input.ModelID)
	} else {
		_, err = s.Pool.Exec(r.Context(), `INSERT INTO worker_voice_models(worker_id,model_id,state,cause_code,bytes_received,bytes_total) VALUES($1,$2,$3,$4,$5,$6)
			ON CONFLICT(worker_id,model_id) DO UPDATE SET state=EXCLUDED.state,cause_code=EXCLUDED.cause_code,bytes_received=EXCLUDED.bytes_received,bytes_total=EXCLUDED.bytes_total,retry_requested=false,updated_at=now()`,
			worker.ID, input.ModelID, input.State, input.CauseCode, input.BytesReceived, input.BytesTotal)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// patchVoiceModel edits the metadata of a reference and enables or disables it. The identity of a reference
// (type, repository, revision, adapter) is never edited: a different model is a new reference. A reference
// that is active or the candidate cannot be disabled; the Super-administrator replaces it first.
func (s *Server) patchVoiceModel(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Label     *string `json:"label"`
		License   *string `json:"license"`
		SizeBytes *int64  `json:"size_bytes"`
		VRAMBytes *int64  `json:"vram_bytes"`
		Enabled   *bool   `json:"enabled"`
	}
	if !decode(w, r, &input) {
		return
	}
	if (input.Label != nil && len(*input.Label) > 100) || (input.License != nil && len(*input.License) > 100) ||
		(input.SizeBytes != nil && *input.SizeBytes < 0) || (input.VRAMBytes != nil && *input.VRAMBytes < 0) {
		fail(w, http.StatusBadRequest, "invalid_voice_model")
		return
	}
	id := r.PathValue("id")
	errNotFound, errInUse := errors.New("not found"), errors.New("in use")
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		entry, ok, err := voiceCatalogByID(r.Context(), tx, id)
		if err != nil {
			return err
		}
		if !ok {
			return errNotFound
		}
		if input.Enabled != nil && !*input.Enabled && entry.Enabled {
			active, err := voiceActiveModelIDIn(r.Context(), tx)
			if err != nil {
				return err
			}
			candidate, err := voiceCandidateIDIn(r.Context(), tx)
			if err != nil {
				return err
			}
			var activeOfType bool
			if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM voice_active_model WHERE model_id=$1)`, id).Scan(&activeOfType); err != nil {
				return err
			}
			if id == active || id == candidate || activeOfType {
				return errInUse
			}
		}
		_, err = tx.Exec(r.Context(), `UPDATE voice_model_catalog SET label=COALESCE($2,label),license=COALESCE($3,license),size_bytes=COALESCE($4,size_bytes),vram_bytes=COALESCE($5,vram_bytes),enabled=COALESCE($6,enabled),updated_at=now() WHERE id=$1`,
			id, input.Label, input.License, input.SizeBytes, input.VRAMBytes, input.Enabled)
		return err
	})
	if errors.Is(err, errNotFound) {
		fail(w, http.StatusNotFound, "voice_model_not_found")
		return
	}
	if errors.Is(err, errInUse) {
		fail(w, http.StatusConflict, "voice_model_in_use")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	entry, _, err := voiceCatalogByID(r.Context(), s.Pool, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, voiceModelViewOf(entry))
}
