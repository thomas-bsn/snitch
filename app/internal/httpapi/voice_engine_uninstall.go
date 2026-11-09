package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"snitch/internal/auth"
)

// Removing an approved engine release from one Worker (#353, ADR 0055). It is the Administrator's explicit
// decision, designating one release on one Worker: nothing else is deleted, no model in particular. A release
// a model of the Worker is bound to is never removed: the Administrator first changes or removes the model.
//
// The removal is an order like the installation. Snitch records it with its author, delivers it on the Worker's
// pull only once nothing depends on the release any more (the drain), and the Worker removes the runtime and
// reports. Everything the decision rests on is in the database, so a restart of Snitch or of the Worker resumes
// the same order without a second effect.

// engineBlockerModel is a model of the Worker bound to the release, with what keeps it in use.
type engineBlockerModel struct {
	ModelID string `json:"model_id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	State   string `json:"state"`
	// Role is "bound" when the model runs on this release (or an update to it is open), "previous" when the release
	// is the version the model can go back to (#352): removing it would make the rollback impossible.
	Role         string `json:"role"`
	Busy         bool   `json:"busy"`
	InTransition bool   `json:"in_transition"`
}

// engineBlockers is what prevents (or delays) the removal of a release on a Worker.
type engineBlockers struct {
	Models []engineBlockerModel `json:"models"`
	// OpenOrder: an installation of this release is still running on the Worker.
	OpenOrder bool `json:"open_order"`
}

func (b engineBlockers) any() bool { return len(b.Models) > 0 || b.OpenOrder }

// sharedDirectoryDigests lists the identities that live in the same Worker directory as this one. A signed release
// has a directory of its own. An on-demand engine is one directory per family and version, whatever its kind, so
// the identities of every kind of that family and version stand for the same environment (ADR 0056).
func sharedDirectoryDigests(ctx context.Context, db pgQuerier, digest string) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT s.digest FROM voice_engine_registry r JOIN voice_engine_registry s ON s.source='on_demand' AND s.engine=r.engine AND s.version=r.version
		WHERE r.digest=$1 AND r.source='on_demand'`, digest)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	digests := []string{digest}
	for rows.Next() {
		var other string
		if err = rows.Scan(&other); err != nil {
			return nil, err
		}
		if other != digest {
			digests = append(digests, other)
		}
	}
	return digests, rows.Err()
}

// engineBlockersIn reads the dependencies of a release on a Worker. A model is bound to the release when its
// current (latest accepted) installation order carried exactly the manifest of the release. Busy tells that a
// job of this Worker still uses the model, InTransition that it belongs to an open fingerprint transition.
func engineBlockersIn(ctx context.Context, db pgQuerier, workerID, digest string) (engineBlockers, error) {
	out := engineBlockers{Models: []engineBlockerModel{}}
	shared, err := sharedDirectoryDigests(ctx, db, digest)
	if err != nil {
		return out, err
	}
	rows, err := db.Query(ctx, `SELECT m.model_id,m.state,CASE WHEN b.previous_digest=ANY($2) AND NOT b.digest=ANY($2) AND NOT EXISTS (
				SELECT 1 FROM worker_model_orders o WHERE o.worker_id=m.worker_id AND o.model_id=m.model_id AND o.action='install' AND o.status IN ('pending','acknowledged','done') AND o.runtime IN (SELECT manifest FROM voice_engine_registry WHERE digest=ANY($2))
				AND o.id=(SELECT x.id FROM worker_model_orders x WHERE x.worker_id=m.worker_id AND x.model_id=m.model_id AND x.action='install' AND x.status IN ('pending','acknowledged','done') ORDER BY x.created_at DESC,x.id DESC LIMIT 1)) THEN 'previous' ELSE 'bound' END
		FROM worker_voice_models m LEFT JOIN worker_model_runtime_bindings b ON b.worker_id=m.worker_id AND b.model_id=m.model_id
		WHERE m.worker_id=$1 AND m.state<>'failed' AND (b.digest=ANY($2) OR b.previous_digest=ANY($2) OR EXISTS (
			SELECT 1 FROM worker_model_orders o
			WHERE o.worker_id=m.worker_id AND o.model_id=m.model_id AND o.action='install' AND o.status IN ('pending','acknowledged','done')
			AND o.runtime IN (SELECT manifest FROM voice_engine_registry WHERE digest=ANY($2))
			AND o.id=(SELECT x.id FROM worker_model_orders x WHERE x.worker_id=m.worker_id AND x.model_id=m.model_id AND x.action='install' AND x.status IN ('pending','acknowledged','done') ORDER BY x.created_at DESC,x.id DESC LIMIT 1)))
		ORDER BY m.model_id`, workerID, shared)
	if err != nil {
		return out, err
	}
	var bound []engineBlockerModel
	for rows.Next() {
		var m engineBlockerModel
		if err = rows.Scan(&m.ModelID, &m.State, &m.Role); err != nil {
			rows.Close()
			return out, err
		}
		bound = append(bound, m)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, err
	}
	for _, m := range bound {
		entry, ok, err := voiceCatalogByID(ctx, db, m.ModelID)
		if err != nil {
			return out, err
		}
		m.Label = m.ModelID
		if ok {
			m.Label, m.Kind = entry.Label, entry.Kind
			if m.Busy, err = voiceModelInUseIn(ctx, db, entry, workerID); err != nil {
				return out, err
			}
		}
		if m.InTransition, err = transitionBlocks(ctx, db, m.ModelID); err != nil {
			return out, err
		}
		out.Models = append(out.Models, m)
	}
	err = db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_engine_orders WHERE worker_id=$1 AND digest=ANY($2) AND intent='install' AND status IN ('pending','acknowledged'))`, workerID, shared).Scan(&out.OpenOrder)
	return out, err
}

// uninstallEngineOrder records the removal of a release on a Worker, or returns the one already recorded under
// the same key. It runs under the voice model lock, like every order that changes what a Worker holds.
func (s *Server) uninstallEngineOrder(ctx context.Context, author session, workerID, digest, key string) (engineOrderView, bool, error) {
	if !idempotencyKeyFormat.MatchString(key) {
		return engineOrderView{}, false, orderFailure{http.StatusBadRequest, "invalid_idempotency_key"}
	}
	if !engineDigest.MatchString(digest) {
		return engineOrderView{}, false, orderFailure{http.StatusBadRequest, "invalid_engine_digest"}
	}
	var out engineOrderView
	created := false
	err := s.withVoiceModelLock(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, workerID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return orderFailure{http.StatusNotFound, "worker_not_found"}
		}
		prior, err := scanEngineOrder(tx.QueryRow(ctx, `SELECT `+engineOrderColumns+` FROM worker_engine_orders WHERE idempotency_key=$1`, key))
		if err == nil {
			if prior.WorkerID != workerID || prior.Digest != digest || prior.Intent != "uninstall" {
				return orderFailure{http.StatusConflict, "idempotency_key_reused"}
			}
			out = prior
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var manifest []byte
		var engine, version string
		err = tx.QueryRow(ctx, `SELECT manifest,engine,version FROM voice_engine_registry WHERE digest=$1`, digest).Scan(&manifest, &engine, &version)
		if errors.Is(err, pgx.ErrNoRows) {
			return orderFailure{http.StatusNotFound, "engine_not_found"}
		}
		if err != nil {
			return err
		}
		// A revoked release can be removed: revocation forbids installing it, it never forbids getting rid of it.
		var open bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_engine_orders WHERE worker_id=$1 AND engine=$2 AND status IN ('pending','acknowledged'))`, workerID, engine).Scan(&open); err != nil {
			return err
		}
		if open {
			return orderFailure{http.StatusConflict, "engine_order_in_progress"}
		}
		var state string
		switch err = tx.QueryRow(ctx, `SELECT state FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2`, workerID, digest).Scan(&state); {
		case errors.Is(err, pgx.ErrNoRows), err == nil && state == "removed":
			return orderFailure{http.StatusConflict, "engine_not_installed"}
		case err != nil:
			return err
		case state != "ready" && state != "failed":
			return orderFailure{http.StatusConflict, "engine_order_in_progress"}
		}
		blockers, err := engineBlockersIn(ctx, tx, workerID, digest)
		if err != nil {
			return err
		}
		if blockers.any() {
			return orderFailure{http.StatusConflict, "engine_in_use"}
		}
		var generation int64
		if err = tx.QueryRow(ctx, `SELECT COALESCE(max(generation),0)+1 FROM worker_engine_orders WHERE worker_id=$1 AND engine=$2`, workerID, engine).Scan(&generation); err != nil {
			return err
		}
		id := "ord_" + auth.Token()[:32]
		if _, err = tx.Exec(ctx, `INSERT INTO worker_engine_orders(id,worker_id,digest,engine,version,intent,manifest,generation,idempotency_key,author_id,author_name) VALUES($1,$2,$3,$4,$5,'uninstall',$6,$7,$8,$9,$10)`,
			id, workerID, digest, engine, version, manifest, generation, key, author.User.ID, author.User.Username); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE worker_engine_inventory SET state='removing',cause_code='',generation=$3,order_id=$4,updated_at=now() WHERE worker_id=$1 AND digest=$2`, workerID, digest, generation, id); err != nil {
			return err
		}
		if err = engineEvent(ctx, tx, workerID, digest, id, generation, "removing", "", false, ""); err != nil {
			return err
		}
		created = true
		out, err = scanEngineOrder(tx.QueryRow(ctx, `SELECT `+engineOrderColumns+` FROM worker_engine_orders WHERE id=$1`, id))
		return err
	})
	if created {
		s.Log.Info("voice engine uninstall ordered", "order_id", out.OrderID, "worker_id", workerID, "digest", digest, "generation", out.Generation, "author_id", author.User.ID)
	}
	return out, created, err
}

// restoreAfterFailedUninstall puts the inventory back where the Worker proves the runtime still is: the
// removal did not happen, so the release is ready again (or failed if it never was proven present).
func restoreAfterFailedUninstall(ctx context.Context, tx pgx.Tx, workerID, digest, orderID string) error {
	_, err := tx.Exec(ctx, `UPDATE worker_engine_inventory SET state=CASE WHEN local_identity<>'' THEN 'ready' ELSE 'failed' END,cause_code=CASE WHEN local_identity<>'' THEN '' ELSE 'runtime_missing' END,updated_at=now() WHERE worker_id=$1 AND digest=$2 AND order_id=$3`, workerID, digest, orderID)
	return err
}

// uninstallPreview is what the Uninstall action shows before the Administrator confirms.
func (s *Server) uninstallPreview(w http.ResponseWriter, r *http.Request) {
	workerID, digest := r.URL.Query().Get("worker_id"), r.URL.Query().Get("digest")
	if !engineDigest.MatchString(digest) || workerID == "" {
		fail(w, http.StatusBadRequest, "invalid_engine_order")
		return
	}
	var state string
	err := s.Pool.QueryRow(r.Context(), `SELECT state FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2`, workerID, digest).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "engine_not_installed")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	blockers, err := engineBlockersIn(r.Context(), s.Pool, workerID, digest)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"state": state, "removable": !blockers.any() && (state == "ready" || state == "failed"), "blockers": blockers})
}
