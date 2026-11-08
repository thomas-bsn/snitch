package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/auth"
)

// Runtime update and rollback of a model that is already Ready (#352, ADR 0055). Both are explicit model orders
// carrying an intent: the new release is installed next to the old one, the weights already validated are reused
// and never downloaded, and the Worker publishes the new binding only once a synthetic inference with it has
// succeeded. Until then the model stays Ready on the previous binding, so dispatch and jobs already engaged keep
// the runtime they started with. A failure leaves the previous binding in place. A revoked release is never a
// target, neither for an update nor for a rollback.

func (s *Server) registerRuntimeUpdateRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/admin/voice-model-runtime-orders", s.protected(true, http.HandlerFunc(s.createRuntimeOrder)))
	mux.Handle("GET /api/admin/voice-engines/bindings", s.protected(true, http.HandlerFunc(s.listRuntimeBindings)))
}

// runtimeEvent appends to the history of a model's runtime. The release is the one the order carries.
func runtimeEvent(ctx context.Context, tx pgx.Tx, workerID, orderID, intent, state, cause string, ignored bool, reason string) error {
	_, err := tx.Exec(ctx, `INSERT INTO worker_model_runtime_events(worker_id,model_id,order_id,intent,digest,state,cause_code,ignored,reason)
		SELECT o.worker_id,o.model_id,o.id,$3,g.digest,$4,$5,$6,$7 FROM worker_model_orders o JOIN voice_engine_registry g ON g.manifest=o.runtime WHERE o.id=$2 AND o.worker_id=$1`,
		workerID, orderID, intent, state, cause, ignored, reason)
	return err
}

// bindInstalledRuntime records the release a model was installed with, once its install order is done.
func bindInstalledRuntime(ctx context.Context, tx pgx.Tx, workerID, modelID, orderID string) error {
	_, err := tx.Exec(ctx, `INSERT INTO worker_model_runtime_bindings(worker_id,model_id,digest,order_id)
		SELECT $1,$2,g.digest,$3 FROM worker_model_orders o JOIN voice_engine_registry g ON g.manifest=o.runtime WHERE o.id=$3
		ON CONFLICT(worker_id,model_id) DO UPDATE SET previous_digest=CASE WHEN worker_model_runtime_bindings.digest<>EXCLUDED.digest THEN worker_model_runtime_bindings.digest ELSE worker_model_runtime_bindings.previous_digest END,
			digest=EXCLUDED.digest,order_id=EXCLUDED.order_id,updated_at=now()`, workerID, modelID, orderID)
	return err
}

type runtimeOrderInput struct {
	WorkerID       string `json:"worker_id"`
	ModelID        string `json:"model_id"`
	Digest         string `json:"digest"`
	Intent         string `json:"intent"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (s *Server) createRuntimeOrder(w http.ResponseWriter, r *http.Request) {
	var input runtimeOrderInput
	if !decode(w, r, &input) {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	order, created, err := s.runtimeOrder(r.Context(), ss, input)
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	reply(w, status, order)
}

// runtimeOrder records an update or a rollback order, or returns the one already recorded under the same key.
// It runs under the voice model lock like every order.
func (s *Server) runtimeOrder(ctx context.Context, author session, in runtimeOrderInput) (voiceModelOrderView, bool, error) {
	if !idempotencyKeyFormat.MatchString(in.IdempotencyKey) {
		return voiceModelOrderView{}, false, orderFailure{http.StatusBadRequest, "invalid_idempotency_key"}
	}
	if !engineDigest.MatchString(in.Digest) {
		return voiceModelOrderView{}, false, orderFailure{http.StatusBadRequest, "invalid_engine_digest"}
	}
	if in.Intent != "update" && in.Intent != "rollback" {
		return voiceModelOrderView{}, false, orderFailure{http.StatusBadRequest, "invalid_runtime_order"}
	}
	var out voiceModelOrderView
	created := false
	err := s.withVoiceModelLock(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, in.WorkerID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return orderFailure{http.StatusNotFound, "worker_not_found"}
		}
		// The key first: a replay is answered with the recorded order whatever changed since.
		var prior, priorWorker, priorModel, priorIntent string
		var same bool
		err := tx.QueryRow(ctx, `SELECT o.id,o.worker_id,o.model_id,o.runtime_intent,COALESCE(g.digest=$2,false) FROM worker_model_orders o LEFT JOIN voice_engine_registry g ON g.manifest=o.runtime WHERE o.idempotency_key=$1`, in.IdempotencyKey, in.Digest).Scan(&prior, &priorWorker, &priorModel, &priorIntent, &same)
		if err == nil {
			if priorWorker != in.WorkerID || priorModel != in.ModelID || priorIntent != in.Intent || !same {
				return orderFailure{http.StatusConflict, "idempotency_key_reused"}
			}
			out, err = orderByID(ctx, tx, prior)
			return err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		entry, ok, err := voiceCatalogByID(ctx, tx, in.ModelID)
		if err != nil {
			return err
		}
		if !ok || !entry.Enabled || !entry.Supported {
			return orderFailure{http.StatusBadRequest, "invalid_voice_model"}
		}
		var state string
		switch err = tx.QueryRow(ctx, `SELECT state FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, in.WorkerID, in.ModelID).Scan(&state); {
		case errors.Is(err, pgx.ErrNoRows):
			return orderFailure{http.StatusConflict, "voice_model_not_ready"}
		case err != nil:
			return err
		case state != "ready":
			return orderFailure{http.StatusConflict, "voice_model_not_ready"}
		}
		var open bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_model_orders WHERE worker_id=$1 AND model_id=$2 AND status IN ('pending','acknowledged'))`, in.WorkerID, in.ModelID).Scan(&open); err != nil {
			return err
		}
		if open {
			return orderFailure{http.StatusConflict, "runtime_order_in_progress"}
		}
		var manifest json.RawMessage
		var engine, kind string
		var revoked bool
		err = tx.QueryRow(ctx, `SELECT manifest,engine,manifest->>'kind',revoked FROM voice_engine_registry WHERE digest=$1`, in.Digest).Scan(&manifest, &engine, &kind, &revoked)
		if errors.Is(err, pgx.ErrNoRows) {
			return orderFailure{http.StatusNotFound, "engine_not_found"}
		}
		if err != nil {
			return err
		}
		// A revoked release is never reactivated, not even by an explicit rollback.
		if revoked {
			return orderFailure{http.StatusConflict, "engine_release_unavailable"}
		}
		if engine != entry.Engine || kind != entry.Kind {
			return orderFailure{http.StatusConflict, "engine_release_mismatch"}
		}
		// A release the Administrator is removing from this Worker (#353) is never a target; one already removed
		// is absent, so it cannot be a rollback target either (an update installs it again, like any new release).
		var held string
		switch err = tx.QueryRow(ctx, `SELECT state FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2`, in.WorkerID, in.Digest).Scan(&held); {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return err
		case held == "removing":
			return orderFailure{http.StatusConflict, "engine_order_in_progress"}
		case held == "removed" && in.Intent == "rollback":
			return orderFailure{http.StatusConflict, "engine_not_installed"}
		}
		var current, previous *string
		switch err = tx.QueryRow(ctx, `SELECT digest,previous_digest FROM worker_model_runtime_bindings WHERE worker_id=$1 AND model_id=$2`, in.WorkerID, in.ModelID).Scan(&current, &previous); {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return err
		case *current == in.Digest:
			return orderFailure{http.StatusConflict, "engine_already_bound"}
		}
		if in.Intent == "rollback" {
			// Only a version this Worker already holds: the one it was bound to before, or one its inventory proves.
			held := previous != nil && *previous == in.Digest
			if !held {
				if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2 AND state='ready')`, in.WorkerID, in.Digest).Scan(&held); err != nil {
					return err
				}
			}
			if !held {
				return orderFailure{http.StatusConflict, "engine_not_installed"}
			}
		} else {
			// An engine order for this engine on this Worker is already installing a release: wait for it.
			var installing bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_engine_orders WHERE worker_id=$1 AND engine=$2 AND status IN ('pending','acknowledged'))`, in.WorkerID, engine).Scan(&installing); err != nil {
				return err
			}
			if installing {
				return orderFailure{http.StatusConflict, "engine_order_in_progress"}
			}
		}
		// The same resource check as any order that gives a runtime to a Worker (#354). A release already ready
		// on the Worker needs no new disk; the other constraints (platform, CUDA, RAM, VRAM) still apply.
		var present bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2 AND state='ready')`, in.WorkerID, in.Digest).Scan(&present); err != nil {
			return err
		}
		if err = checkWorkerEngine(ctx, tx, in.WorkerID, manifest, present || in.Intent == "rollback"); err != nil {
			return err
		}
		id := "ord_" + auth.Token()[:32]
		if _, err = tx.Exec(ctx, `INSERT INTO worker_model_orders(id,worker_id,kind,action,model_id,revision,idempotency_key,author_id,author_name,runtime,runtime_intent) VALUES($1,$2,$3,'install',$4,$5,$6,$7,$8,$9,$10)`,
			id, in.WorkerID, entry.Kind, entry.ID, entry.Revision, in.IdempotencyKey, author.User.ID, author.User.Username, manifest, in.Intent); err != nil {
			return err
		}
		if err = runtimeEvent(ctx, tx, in.WorkerID, id, in.Intent, "requested", "", false, ""); err != nil {
			return err
		}
		created = true
		out, err = orderByID(ctx, tx, id)
		return err
	})
	if created {
		s.Log.Info("voice model runtime order", "order_id", out.OrderID, "intent", in.Intent, "worker_id", in.WorkerID, "model_id", in.ModelID, "digest", in.Digest, "author_id", author.User.ID)
	}
	return out, created, err
}

// applyRuntimeOrderStatus is the report of a Worker answering an update or a rollback order. It never touches the
// model row. A late report (cancelled order, finished order) is ignored and journalled. The binding is published
// only by a ready report, which the Worker sends after a synthetic inference with the new runtime.
func (s *Server) applyRuntimeOrderStatus(ctx context.Context, tx pgx.Tx, workerID, orderID, intent, status, state, cause string) error {
	ignore := func(reason string) error {
		s.Log.Warn("late runtime report ignored", "order_id", orderID, "worker_id", workerID, "state", state, "reason", reason)
		eventState := state
		if state == "ready" {
			eventState = "published"
		}
		return runtimeEvent(ctx, tx, workerID, orderID, intent, eventState, cause, true, reason)
	}
	switch status {
	case "cancelled":
		return ignore("order_cancelled")
	case "done", "failed", "refused":
		if (status == "done" && state == "ready") || (status == "failed" && state == "failed") {
			return nil
		}
		return ignore("order_finished")
	}
	var modelID, digest, modelState string
	err := tx.QueryRow(ctx, `SELECT o.model_id,g.digest,COALESCE(m.state,'') FROM worker_model_orders o JOIN voice_engine_registry g ON g.manifest=o.runtime
		LEFT JOIN worker_voice_models m ON m.worker_id=o.worker_id AND m.model_id=o.model_id WHERE o.id=$1`, orderID).Scan(&modelID, &digest, &modelState)
	if err != nil {
		return err
	}
	next, eventState := "acknowledged", state
	switch state {
	case "ready":
		next, eventState = "done", "published"
		if modelState != "ready" {
			// The model was removed while the order ran: there is nothing to bind.
			return ignore("model_not_ready")
		}
	case "failed":
		next = "failed"
	}
	if _, err = tx.Exec(ctx, `UPDATE worker_model_orders SET status=$2,cause_code=$3,acked_at=COALESCE(acked_at,now()),finished_at=CASE WHEN $2 IN ('done','failed') THEN now() END WHERE id=$1`, orderID, next, cause); err != nil {
		return err
	}
	var last string
	switch err = tx.QueryRow(ctx, `SELECT state FROM worker_model_runtime_events WHERE order_id=$1 AND NOT ignored ORDER BY id DESC LIMIT 1`, orderID).Scan(&last); {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return err
	}
	if state == "ready" {
		if _, err = tx.Exec(ctx, `INSERT INTO worker_model_runtime_bindings(worker_id,model_id,digest,order_id) VALUES($1,$2,$3,$4)
			ON CONFLICT(worker_id,model_id) DO UPDATE SET previous_digest=worker_model_runtime_bindings.digest,digest=EXCLUDED.digest,order_id=EXCLUDED.order_id,updated_at=now()`, workerID, modelID, digest, orderID); err != nil {
			return err
		}
	}
	if last == eventState {
		return nil // progress inside one step is not a new history entry
	}
	return runtimeEvent(ctx, tx, workerID, orderID, intent, eventState, cause, false, "")
}

type runtimeHistoryEntry struct {
	Intent    string    `json:"intent"`
	Digest    string    `json:"digest"`
	Version   string    `json:"version"`
	State     string    `json:"state"`
	CauseCode string    `json:"cause_code"`
	Ignored   bool      `json:"ignored"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type runtimePending struct {
	OrderID string `json:"order_id"`
	Intent  string `json:"intent"`
	Digest  string `json:"digest"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type runtimeBinding struct {
	WorkerID        string                `json:"worker_id"`
	ModelID         string                `json:"model_id"`
	ModelLabel      string                `json:"model_label"`
	Engine          string                `json:"engine"`
	Kind            string                `json:"kind"`
	Digest          string                `json:"digest"`
	Version         string                `json:"version"`
	Revoked         bool                  `json:"revoked"`
	PreviousDigest  string                `json:"previous_digest"`
	PreviousVersion string                `json:"previous_version"`
	PreviousRevoked bool                  `json:"previous_revoked"`
	Pending         *runtimePending       `json:"pending,omitempty"`
	History         []runtimeHistoryEntry `json:"history"`
}

const runtimeHistoryLimit = 20

// listRuntimeBindings lists, for each Ready model of a Worker that runs on an approved engine, the runtime it is
// bound to, the one it can go back to, the order in progress and the history of its updates.
func (s *Server) listRuntimeBindings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT m.worker_id,m.model_id,c.label,c.engine,c.kind,
			COALESCE(b.digest,''),COALESCE(g.manifest->>'version',''),COALESCE(g.revoked,false),
			COALESCE(b.previous_digest,''),COALESCE(p.manifest->>'version',''),COALESCE(p.revoked,false)
		FROM worker_voice_models m JOIN voice_model_catalog c ON c.id=m.model_id
		JOIN workers k ON k.id=m.worker_id AND k.revoked_at IS NULL
		LEFT JOIN worker_model_runtime_bindings b ON b.worker_id=m.worker_id AND b.model_id=m.model_id
		LEFT JOIN voice_engine_registry g ON g.digest=b.digest LEFT JOIN voice_engine_registry p ON p.digest=b.previous_digest
		WHERE m.state='ready' AND EXISTS(SELECT 1 FROM voice_engine_registry e WHERE e.engine=c.engine AND e.manifest->>'kind'=c.kind)
		ORDER BY m.worker_id,m.model_id`)
	if err != nil {
		s.internal(w, err)
		return
	}
	out := struct {
		Bindings []runtimeBinding `json:"bindings"`
	}{Bindings: []runtimeBinding{}}
	for rows.Next() {
		var b runtimeBinding
		if err = rows.Scan(&b.WorkerID, &b.ModelID, &b.ModelLabel, &b.Engine, &b.Kind, &b.Digest, &b.Version, &b.Revoked, &b.PreviousDigest, &b.PreviousVersion, &b.PreviousRevoked); err != nil {
			rows.Close()
			s.internal(w, err)
			return
		}
		b.History = []runtimeHistoryEntry{}
		out.Bindings = append(out.Bindings, b)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	for i := range out.Bindings {
		b := &out.Bindings[i]
		var p runtimePending
		switch err = s.Pool.QueryRow(r.Context(), `SELECT o.id,o.runtime_intent,g.digest,g.manifest->>'version',o.status FROM worker_model_orders o JOIN voice_engine_registry g ON g.manifest=o.runtime
			WHERE o.worker_id=$1 AND o.model_id=$2 AND o.runtime_intent<>'install' AND o.status IN ('pending','acknowledged') ORDER BY o.created_at DESC LIMIT 1`, b.WorkerID, b.ModelID).Scan(&p.OrderID, &p.Intent, &p.Digest, &p.Version, &p.Status); {
		case err == nil:
			b.Pending = &p
		case !errors.Is(err, pgx.ErrNoRows):
			s.internal(w, err)
			return
		}
		hist, err := s.Pool.Query(r.Context(), `SELECT e.intent,e.digest,COALESCE(g.manifest->>'version',''),e.state,e.cause_code,e.ignored,e.reason,e.created_at FROM worker_model_runtime_events e
			JOIN voice_engine_registry g ON g.digest=e.digest WHERE e.worker_id=$1 AND e.model_id=$2 ORDER BY e.id DESC LIMIT $3`, b.WorkerID, b.ModelID, runtimeHistoryLimit)
		if err != nil {
			s.internal(w, err)
			return
		}
		for hist.Next() {
			var h runtimeHistoryEntry
			if err = hist.Scan(&h.Intent, &h.Digest, &h.Version, &h.State, &h.CauseCode, &h.Ignored, &h.Reason, &h.CreatedAt); err != nil {
				hist.Close()
				s.internal(w, err)
				return
			}
			b.History = append(b.History, h)
		}
		hist.Close()
	}
	reply(w, http.StatusOK, out)
}
