package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// Leaving a failed engine installation (#351): retry, cooperative cancellation and explicit cleanup. Every one is
// a durable request under the voice model lock, answered the same way when repeated, and none of them touches a
// model, a binding or a ready runtime: the inventory is keyed by release, so a failed preparation of one release
// never alters what another release of the same engine already serves.
//
// Replacing is not a separate operation: a failed release never blocks an order for another compatible release of
// the same engine (installEngineOrder only refuses an open order or an already ready digest).

func (s *Server) registerVoiceEngineRecoveryRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/admin/workers/{id}/voice-engines/{digest}/retry", s.protected(true, http.HandlerFunc(s.retryEngineInstall)))
	mux.Handle("POST /api/admin/workers/{id}/voice-engines/{digest}/cancel", s.protected(true, http.HandlerFunc(s.cancelEngineInstall)))
	mux.Handle("POST /api/admin/workers/{id}/voice-engines/{digest}/cleanup", s.protected(true, http.HandlerFunc(s.requestEngineCleanup)))
	mux.Handle("POST /api/worker/voice-engines/cleanup", s.workerOnly(http.HandlerFunc(s.postWorkerEngineCleanup)))
}

type engineRecoveryResult struct {
	Status string `json:"status"`
}

// retryEngineInstall orders the same release again, only from a failed state. The idempotency key makes a double
// click or a replayed request answer with the order already recorded; an open order is refused by the order itself.
func (s *Server) retryEngineInstall(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decode(w, r, &input) {
		return
	}
	workerID, digest := r.PathValue("id"), r.PathValue("digest")
	var state string
	err := s.Pool.QueryRow(r.Context(), `SELECT state FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2`, workerID, digest).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		// A replay of a retry that already created its order finds the release requested, not failed.
		if prior, ok := s.engineOrderByKey(r.Context(), input.IdempotencyKey, workerID, digest); ok {
			reply(w, http.StatusOK, prior)
			return
		}
		fail(w, http.StatusConflict, "engine_not_failed")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if state != "failed" {
		if prior, ok := s.engineOrderByKey(r.Context(), input.IdempotencyKey, workerID, digest); ok {
			reply(w, http.StatusOK, prior)
			return
		}
		fail(w, http.StatusConflict, "engine_not_failed")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	order, created, err := s.installEngineOrder(r.Context(), ss, workerID, digest, input.IdempotencyKey)
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

func (s *Server) engineOrderByKey(ctx context.Context, key, workerID, digest string) (engineOrderView, bool) {
	o, err := scanEngineOrder(s.Pool.QueryRow(ctx, `SELECT `+engineOrderColumns+` FROM worker_engine_orders WHERE idempotency_key=$1 AND worker_id=$2 AND digest=$3`, key, workerID, digest))
	return o, err == nil
}

// currentEngineOrder is the order the inventory row of (Worker, release) stands on.
func currentEngineOrder(ctx context.Context, tx pgx.Tx, workerID, digest string) (id, status string, generation int64, cancelRequested bool, err error) {
	err = tx.QueryRow(ctx, `SELECT o.id,o.status,o.generation,o.cancel_requested_at IS NOT NULL FROM worker_engine_inventory i JOIN worker_engine_orders o ON o.id=i.order_id WHERE i.worker_id=$1 AND i.digest=$2 FOR UPDATE OF o`, workerID, digest).Scan(&id, &status, &generation, &cancelRequested)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", 0, false, orderFailure{http.StatusNotFound, "engine_order_not_found"}
	}
	return
}

// cancelEngineInstall stops an install. An order the Worker has not acknowledged is cancelled at once; one it is
// running is flagged, delivered with the next pull and confirmed by the Worker when it stopped at a safe point.
func (s *Server) cancelEngineInstall(w http.ResponseWriter, r *http.Request) {
	workerID, digest := r.PathValue("id"), r.PathValue("digest")
	result := engineRecoveryResult{}
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		id, status, generation, requested, err := currentEngineOrder(r.Context(), tx, workerID, digest)
		if err != nil {
			return err
		}
		switch status {
		case "cancelled":
			result.Status = "cancelled"
			return nil
		case "pending":
			if _, err = tx.Exec(r.Context(), `UPDATE worker_engine_orders SET status='cancelled',cause_code='cancelled',finished_at=now() WHERE id=$1`, id); err != nil {
				return err
			}
			if _, err = tx.Exec(r.Context(), `UPDATE worker_engine_inventory SET state='failed',cause_code='cancelled',updated_at=now() WHERE worker_id=$1 AND digest=$2 AND order_id=$3`, workerID, digest, id); err != nil {
				return err
			}
			result.Status = "cancelled"
			return engineEvent(r.Context(), tx, workerID, digest, id, generation, "failed", "cancelled", false, "cancel_before_delivery")
		case "acknowledged":
			result.Status = "cancelling"
			if requested {
				return nil // already recorded: repeating the request changes nothing
			}
			if _, err = tx.Exec(r.Context(), `UPDATE worker_engine_orders SET cancel_requested_at=now() WHERE id=$1`, id); err != nil {
				return err
			}
			var state string
			if err = tx.QueryRow(r.Context(), `SELECT state FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2`, workerID, digest).Scan(&state); err != nil {
				return err
			}
			return engineEvent(r.Context(), tx, workerID, digest, id, generation, state, "", false, "cancel_requested")
		default:
			return orderFailure{http.StatusConflict, "engine_order_not_cancellable"}
		}
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	if result.Status == "cancelled" {
		s.Log.Info("voice engine install cancelled", "worker_id", workerID, "digest", digest)
	}
	reply(w, http.StatusOK, result)
}

// requestEngineCleanup records the request to remove the incomplete files of a failed or cancelled install. It is
// refused while an order is open and for a release that is ready: nothing validated is ever removed this way.
func (s *Server) requestEngineCleanup(w http.ResponseWriter, r *http.Request) {
	workerID, digest := r.PathValue("id"), r.PathValue("digest")
	result := engineRecoveryResult{}
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		id, status, generation, _, err := currentEngineOrder(r.Context(), tx, workerID, digest)
		if err != nil {
			return err
		}
		var state string
		if err = tx.QueryRow(r.Context(), `SELECT state FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2`, workerID, digest).Scan(&state); err != nil {
			return err
		}
		if state != "failed" || (status != "failed" && status != "cancelled" && status != "refused") {
			return orderFailure{http.StatusConflict, "engine_not_failed"}
		}
		var cleaned, requested bool
		if err = tx.QueryRow(r.Context(), `SELECT cleaned_at IS NOT NULL,cleanup_requested_at IS NOT NULL FROM worker_engine_orders WHERE id=$1`, id).Scan(&cleaned, &requested); err != nil {
			return err
		}
		switch {
		case cleaned:
			result.Status = "done"
		case requested:
			result.Status = "requested"
		default:
			result.Status = "requested"
			if _, err = tx.Exec(r.Context(), `UPDATE worker_engine_orders SET cleanup_requested_at=now() WHERE id=$1`, id); err != nil {
				return err
			}
			return engineEvent(r.Context(), tx, workerID, digest, id, generation, "failed", "", false, "cleanup_requested")
		}
		return nil
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	reply(w, http.StatusOK, result)
}

type engineCleanupView struct {
	OrderID string          `json:"order_id"`
	Digest  string          `json:"digest"`
	Runtime json.RawMessage `json:"runtime"`
}

// pendingEngineCleanups are the cleanups a Worker still has to run. Only the order the inventory currently stands
// on is delivered: once a new order exists for the release, an older cleanup request is never run.
func (s *Server) pendingEngineCleanups(ctx context.Context, workerID string) ([]engineCleanupView, error) {
	rows, err := s.Pool.Query(ctx, `SELECT o.id,o.digest,o.manifest FROM worker_engine_orders o JOIN worker_engine_inventory i ON i.order_id=o.id
		WHERE o.worker_id=$1 AND o.cleanup_requested_at IS NOT NULL AND o.cleaned_at IS NULL AND o.status IN ('failed','cancelled','refused') AND i.state='failed' ORDER BY o.created_at,o.id`, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []engineCleanupView{}
	for rows.Next() {
		var c engineCleanupView
		if err = rows.Scan(&c.OrderID, &c.Digest, &c.Runtime); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// postWorkerEngineCleanup is the Worker's confirmation that it ran a requested cleanup (removed or nothing to
// remove). Repeating it has no second effect, and a confirmation nobody asked for is refused.
func (s *Server) postWorkerEngineCleanup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		OrderID string `json:"order_id"`
		Digest  string `json:"digest"`
		Removed bool   `json:"removed"`
	}
	if !decode(w, r, &input) {
		return
	}
	worker := currentWorker(r)
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		var digest, status string
		var generation int64
		var requested, cleaned bool
		err := tx.QueryRow(r.Context(), `SELECT digest,status,generation,cleanup_requested_at IS NOT NULL,cleaned_at IS NOT NULL FROM worker_engine_orders WHERE id=$1 AND worker_id=$2 FOR UPDATE`, input.OrderID, worker.ID).Scan(&digest, &status, &generation, &requested, &cleaned)
		if errors.Is(err, pgx.ErrNoRows) {
			return orderFailure{http.StatusNotFound, "voice_order_not_found"}
		}
		if err != nil {
			return err
		}
		if digest != input.Digest || !requested {
			return orderFailure{http.StatusBadRequest, "invalid_engine_status"}
		}
		if cleaned {
			return nil
		}
		if _, err = tx.Exec(r.Context(), `UPDATE worker_engine_orders SET cleaned_at=now() WHERE id=$1`, input.OrderID); err != nil {
			return err
		}
		reason := "cleanup_nothing"
		if input.Removed {
			reason = "cleanup_done"
		}
		return engineEvent(r.Context(), tx, worker.ID, digest, input.OrderID, generation, "failed", "", false, reason)
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
