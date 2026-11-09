package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// On-demand engines (#370, ADR 0056). A family of library, a version and a task kind identify an engine that a
// Worker resolves with pip: nothing is signed and no manifest is frozen by a publisher. The table below is the
// only place that names families; it carries the constraints checked before any order exists.
//
// The identity of such an engine is a registry row whose digest hashes (family, version, kind) under a domain
// prefix. A signed digest is the hash of a signed payload, so the two identities never collide and never
// stand for each other (ADR 0055).

const engineSourceOnDemand = "on_demand"

type engineFamily struct {
	Key string
	// Engine is the name the registry, the orders and the inventory use. It differs from the embedded engines.
	Engine       string
	Distribution string
	DefaultVer   string
	// Kinds are the tasks the family covers; the first one is the kind of the registry row.
	Kinds    []string
	Python   string
	Platform string
	// CUDA is "" for a family that runs with or without a GPU.
	CUDA             string
	VRAMBytes        int64
	RAMBytes         int64
	DiskBytes        int64
	EstimatedSeconds int64
}

var engineFamilies = map[string]engineFamily{
	"nemo": {
		Key: "nemo", Engine: "nemo-toolkit", Distribution: "nemo-toolkit", DefaultVer: "3.0.0",
		Kinds:  []string{"diarization", "fingerprint"},
		Python: "3.12", Platform: "linux-x86_64",
		RAMBytes: 8 << 30, DiskBytes: 12 << 30, EstimatedSeconds: 600,
	},
}

// onDemandDigest is the deterministic identity of an on-demand engine.
func onDemandDigest(family, version, kind string) string {
	sum := sha256.Sum256([]byte("snitch:on-demand-engine:v1\n" + family + "\n" + version + "\n" + kind))
	return hex.EncodeToString(sum[:])
}

// onDemandManifest is the runtime an order delivers to the Worker. Its constraint fields use the names of
// the signed manifest, so the hardware check is the one that already exists.
func onDemandManifest(f engineFamily, version string) ([]byte, string, error) {
	kind := f.Kinds[0]
	raw, err := json.Marshal(map[string]any{
		"source": engineSourceOnDemand, "family": f.Key, "engine": f.Engine, "distribution": f.Distribution,
		"version": version, "kind": kind, "kinds": f.Kinds,
		"python": f.Python, "platform": f.Platform, "cuda": f.CUDA,
		"disk_bytes": f.DiskBytes, "vram_bytes": f.VRAMBytes, "ram_bytes": f.RAMBytes, "estimated_seconds": f.EstimatedSeconds,
	})
	return raw, onDemandDigest(f.Key, version, kind), err
}

// installOnDemandEngineOrder records the registry identity of the engine, then an order on a Worker through the
// same path as a signed release (idempotency, generation, hardware, inventory). A refusal rolls the identity back.
func (s *Server) installOnDemandEngineOrder(ctx context.Context, author session, workerID string, f engineFamily, version, key string) (engineOrderView, bool, error) {
	if !idempotencyKeyFormat.MatchString(key) {
		return engineOrderView{}, false, orderFailure{http.StatusBadRequest, "invalid_idempotency_key"}
	}
	manifest, digest, err := onDemandManifest(f, version)
	if err != nil {
		return engineOrderView{}, false, err
	}
	var out engineOrderView
	created := false
	err = s.withVoiceModelLock(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO voice_engine_registry(digest,engine,version,kind,source,manifest,author_id,author_name) VALUES($1,$2,$3,$4,'on_demand',$5,$6,$7) ON CONFLICT DO NOTHING`,
			digest, f.Engine, version, f.Kinds[0], manifest, author.User.ID, author.User.Username); err != nil {
			return err
		}
		var err error
		out, created, err = s.installEngineOrderIn(ctx, tx, author, workerID, digest, key)
		return err
	})
	if created {
		s.Log.Info("on-demand engine install ordered", "order_id", out.OrderID, "worker_id", workerID, "family", f.Key, "version", version, "digest", digest, "generation", out.Generation, "author_id", author.User.ID)
	}
	return out, created, err
}

// createOnDemandEngineOrders orders one family and version on several Workers; each Worker is decided on its own.
func (s *Server) createOnDemandEngineOrders(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Family  string `json:"family"`
		Version string `json:"version"`
		Orders  []struct {
			WorkerID       string `json:"worker_id"`
			IdempotencyKey string `json:"idempotency_key"`
		} `json:"orders"`
	}
	if !decode(w, r, &input) {
		return
	}
	family, ok := engineFamilies[input.Family]
	if !ok {
		fail(w, http.StatusBadRequest, "unsupported_engine_family")
		return
	}
	version := input.Version
	if version == "" {
		version = family.DefaultVer
	}
	if !engineVersion.MatchString(version) || len(input.Orders) == 0 || len(input.Orders) > 100 {
		fail(w, http.StatusBadRequest, "invalid_engine_order")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	results := make([]engineOrderResult, 0, len(input.Orders))
	for _, item := range input.Orders {
		order, created, err := s.installOnDemandEngineOrder(r.Context(), ss, item.WorkerID, family, version, item.IdempotencyKey)
		result := engineOrderResult{WorkerID: item.WorkerID}
		var f orderFailure
		switch {
		case errors.As(err, &f):
			result.Status, result.CauseCode = "refused", f.Code
		case err != nil:
			s.internal(w, err)
			return
		case created:
			result.Status, result.Order = "created", &order
		default:
			result.Status, result.Order = "replayed", &order
		}
		results = append(results, result)
	}
	reply(w, http.StatusOK, map[string]any{"results": results})
}
