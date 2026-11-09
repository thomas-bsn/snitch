package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// Models that ask for an on-demand engine (#372, ADR 0056). Choosing such a model orders the engine of its family
// and kind, then the model, in one request; the model waits for its engine through the dependency that signed
// releases already use. Nothing here orders anything without the Administrator having confirmed the order.

const (
	engineStateMissing    = "missing"
	engineStateInProgress = "in_progress"
	engineStateReady      = "ready"
)

// onDemandEngineState looks for an engine of a family and kind that a Worker already has, ready or being
// installed. A ready one wins; among equals the newest registered version does. version is empty and state
// "missing" when the Worker has none.
func onDemandEngineState(ctx context.Context, db pgQuerier, workerID, family, kind string) (version, state string, err error) {
	var ready bool
	err = db.QueryRow(ctx, `SELECT r.version, EXISTS (SELECT 1 FROM worker_engine_inventory i WHERE i.worker_id=$1 AND i.digest=r.digest AND i.state='ready')
		FROM voice_engine_registry r
		WHERE r.source='on_demand' AND r.manifest->>'family'=$2 AND r.kind=$3 AND (
			EXISTS (SELECT 1 FROM worker_engine_inventory i WHERE i.worker_id=$1 AND i.digest=r.digest AND i.state='ready')
			OR EXISTS (SELECT 1 FROM worker_engine_orders o WHERE o.worker_id=$1 AND o.digest=r.digest AND o.intent='install' AND o.status IN ('pending','acknowledged')))
		ORDER BY 2 DESC, r.created_at DESC, r.digest LIMIT 1`, workerID, family, kind).Scan(&version, &ready)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return "", engineStateMissing, nil
	case err != nil:
		return "", "", err
	case ready:
		return version, engineStateReady, nil
	}
	return version, engineStateInProgress, nil
}

// onDemandRuntime registers the engine a model needs and returns the runtime its order freezes. An engine the
// Worker already has (ready or in progress) is the one used, whatever its version; otherwise the default version
// of the family. A family unknown to the table, or one that does not cover the kind of the model, is refused
// here, before any row is written.
func onDemandRuntime(ctx context.Context, tx pgx.Tx, author session, workerID string, entry voiceCatalogEntry) (json.RawMessage, error) {
	f, _, supported := entry.engineNeed()
	if !supported {
		return nil, orderFailure{http.StatusBadRequest, "unsupported_engine_family"}
	}
	version, _, err := onDemandEngineState(ctx, tx, workerID, f.Key, entry.Kind)
	if err != nil {
		return nil, err
	}
	if version == "" {
		version = f.DefaultVer
	}
	if err := registerOnDemandEngine(ctx, tx, author, f, version, entry.Kind); err != nil {
		return nil, err
	}
	manifest, _, err := onDemandManifest(f, version, entry.Kind)
	return manifest, err
}

// enginePreview is what the confirmation of a model order shows before anything is ordered.
type enginePreview struct {
	ModelID        string `json:"model_id"`
	EngineRequired bool   `json:"engine_required"`
	Family         string `json:"family,omitempty"`
	Engine         string `json:"engine,omitempty"`
	Version        string `json:"version,omitempty"`
	Kind           string `json:"kind,omitempty"`
	State          string `json:"state,omitempty"`
	// Active is false when the family installs for this kind but no adapter runs it (SpeechBrain separation): the
	// engine is ordered with the model and the model stays unavailable.
	Active           bool  `json:"active"`
	DiskBytes        int64 `json:"disk_bytes"`
	EstimatedSeconds int64 `json:"estimated_seconds"`
	// Blocker is the stable cause that would refuse the engine order on this Worker (hardware), empty otherwise.
	Blocker string `json:"blocker,omitempty"`
}

func (s *Server) previewVoiceModelOrder(w http.ResponseWriter, r *http.Request) {
	workerID, modelID := r.PathValue("id"), r.URL.Query().Get("model_id")
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, workerID).Scan(&exists); err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	entry, ok, err := voiceCatalogByID(r.Context(), s.Pool, modelID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !ok || !entry.Enabled || !entry.Supported {
		fail(w, http.StatusBadRequest, "invalid_voice_model")
		return
	}
	out := enginePreview{ModelID: modelID}
	f, needed, supported := entry.engineNeed()
	switch {
	case !needed:
		reply(w, http.StatusOK, out)
		return
	case !supported:
		fail(w, http.StatusBadRequest, "unsupported_engine_family")
		return
	}
	version, state, err := onDemandEngineState(r.Context(), s.Pool, workerID, f.Key, entry.Kind)
	if err != nil {
		s.internal(w, err)
		return
	}
	if version == "" {
		version = f.DefaultVer
	}
	out.EngineRequired, out.Family, out.Engine, out.Version, out.Kind, out.State, out.Active = true, f.Key, f.Engine, version, entry.Kind, state, f.runs(entry.Kind)
	if state != engineStateReady {
		out.DiskBytes, out.EstimatedSeconds = f.DiskBytes, f.EstimatedSeconds
	}
	if state == engineStateMissing {
		manifest, _, err := onDemandManifest(f, version, entry.Kind)
		if err == nil {
			err = checkWorkerEngine(r.Context(), s.Pool, workerID, manifest, false)
		}
		var failure orderFailure
		switch {
		case errors.As(err, &failure):
			out.Blocker = failure.Code
		case err != nil:
			s.internal(w, err)
			return
		}
	}
	reply(w, http.StatusOK, out)
}
