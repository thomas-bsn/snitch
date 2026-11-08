package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// postWorkerVoiceModelStatus is what a Worker really holds, optionally in answer to an order. Snitch never
// trusts a typed string: ready is whatever the Worker reports after loading and a synthetic inference.
func (s *Server) postWorkerVoiceModelStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ModelID       string `json:"model_id"`
		OrderID       string `json:"order_id"`
		Revision      string `json:"revision"`
		State         string `json:"state"`
		CauseCode     string `json:"cause_code"`
		BytesReceived int64  `json:"bytes_received"`
		BytesTotal    int64  `json:"bytes_total"`
		OccupiedBytes int64  `json:"occupied_bytes"`
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
	validCause := (failed && (voiceFailureCauses[input.CauseCode] || input.CauseCode == "invalid_order")) || (!failed && input.CauseCode == "")
	if !known || !validState || !validCause || input.BytesReceived < 0 || input.BytesTotal < 0 || input.OccupiedBytes < 0 || len(input.Revision) > 64 {
		fail(w, http.StatusBadRequest, "invalid_voice_model_status")
		return
	}
	worker := currentWorker(r)
	if input.OrderID == "" {
		s.applyUnorderedVoiceModelStatus(w, r, worker.ID, input.ModelID, input.State, input.CauseCode, input.Revision, input.BytesReceived, input.BytesTotal, input.OccupiedBytes)
		return
	}
	// A report that answers an order: the order must be this Worker's, for this model and revision, and still
	// open. A report that repeats the outcome already recorded is accepted without effect.
	err = s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		var status, modelID, revision, action, intent string
		err := tx.QueryRow(r.Context(), `SELECT status,model_id,revision,action,runtime_intent FROM worker_model_orders WHERE id=$1 AND worker_id=$2`, input.OrderID, worker.ID).Scan(&status, &modelID, &revision, &action, &intent)
		if errors.Is(err, pgx.ErrNoRows) {
			return orderFailure{http.StatusNotFound, "voice_order_not_found"}
		}
		if err != nil {
			return err
		}
		// An install order reports the progress of the model; a remove order reports the removal and nothing else.
		removal := action == "remove"
		if modelID != input.ModelID || (input.Revision != "" && input.Revision != revision) || removal != (input.State == "removed" || (removal && failed)) {
			return orderFailure{http.StatusBadRequest, "invalid_voice_model_status"}
		}
		if intent != "install" {
			// An update or a rollback never moves the model row: the model stays Ready on its binding.
			return s.applyRuntimeOrderStatus(r.Context(), tx, worker.ID, input.OrderID, intent, status, input.State, input.CauseCode)
		}
		if status == "done" || status == "failed" || status == "refused" || status == "cancelled" {
			if (status == "done" && (input.State == "ready" || input.State == "removed")) || (status == "failed" && failed) {
				return nil
			}
			return orderFailure{http.StatusConflict, "stale_voice_order"}
		}
		next := "acknowledged"
		switch input.State {
		case "ready", "removed":
			next = "done"
		case "failed":
			next = "failed"
		}
		if _, err = tx.Exec(r.Context(), `UPDATE worker_model_orders SET status=$2,cause_code=$3,acked_at=COALESCE(acked_at,now()),finished_at=CASE WHEN $2 IN ('done','failed') THEN now() END WHERE id=$1`, input.OrderID, next, input.CauseCode); err != nil {
			return err
		}
		if removal {
			if input.State == "removed" {
				return removalDone(r.Context(), tx, worker.ID, modelID)
			}
			return removalFailed(r.Context(), tx, worker.ID, modelID, input.CauseCode)
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO worker_voice_models(worker_id,model_id,state,cause_code,bytes_received,bytes_total,occupied_bytes,revision,order_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT(worker_id,model_id) DO UPDATE SET state=EXCLUDED.state,cause_code=EXCLUDED.cause_code,bytes_received=EXCLUDED.bytes_received,bytes_total=EXCLUDED.bytes_total,occupied_bytes=EXCLUDED.occupied_bytes,revision=EXCLUDED.revision,order_id=EXCLUDED.order_id,retry_requested=false,updated_at=now()`,
			worker.ID, input.ModelID, input.State, input.CauseCode, input.BytesReceived, input.BytesTotal, input.OccupiedBytes, revision, input.OrderID); err != nil {
			return err
		}
		if input.State == "ready" {
			if err = bindInstalledRuntime(r.Context(), tx, worker.ID, modelID, input.OrderID); err != nil {
				return err
			}
		}
		return s.advanceReplacementsIn(r.Context(), tx, worker.ID)
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	s.stepFingerprintTransition(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

// applyUnorderedVoiceModelStatus is the report of a Worker that prepares the active or candidate model by
// itself (the former behaviour, until the Worker only runs orders): no order, same rows.
func (s *Server) applyUnorderedVoiceModelStatus(w http.ResponseWriter, r *http.Request, workerID, modelID, state, cause, revision string, received, total, occupied int64) {
	var err error
	if state == "removed" {
		_, err = s.Pool.Exec(r.Context(), `DELETE FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, modelID)
	} else {
		_, err = s.Pool.Exec(r.Context(), `INSERT INTO worker_voice_models(worker_id,model_id,state,cause_code,bytes_received,bytes_total,occupied_bytes,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT(worker_id,model_id) DO UPDATE SET state=EXCLUDED.state,cause_code=EXCLUDED.cause_code,bytes_received=EXCLUDED.bytes_received,bytes_total=EXCLUDED.bytes_total,occupied_bytes=EXCLUDED.occupied_bytes,revision=CASE WHEN EXCLUDED.revision<>'' THEN EXCLUDED.revision ELSE worker_voice_models.revision END,retry_requested=false,updated_at=now()`,
			workerID, modelID, state, cause, received, total, occupied, revision)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	s.stepFingerprintTransition(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

// retryVoiceModelOrder orders again the installation of a model whose last attempt failed on this Worker: a
// new order with its own key, so the history keeps both attempts.
func (s *Server) retryVoiceModelOrder(w http.ResponseWriter, r *http.Request, workerID string) {
	var input struct {
		ModelID        string `json:"model_id"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decode(w, r, &input) {
		return
	}
	// A replacement whose removal of the old model failed: Retry orders the removal again.
	if retried, err := s.retryReplacementRemoval(r.Context(), workerID, input.ModelID); err != nil {
		s.internal(w, err)
		return
	} else if retried {
		reply(w, http.StatusOK, map[string]bool{"retry_requested": true})
		return
	}
	var state string
	err := s.Pool.QueryRow(r.Context(), `SELECT state FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, input.ModelID).Scan(&state)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	if state != "failed" {
		// A stalled order (Worker gone, restarted or silent) is abandoned and ordered again, with its own key.
		abandoned, err := s.abandonStalledOrder(r.Context(), workerID, input.ModelID)
		if err != nil {
			s.internal(w, err)
			return
		}
		if !abandoned {
			fail(w, http.StatusConflict, "voice_model_not_failed")
			return
		}
	}
	ss := r.Context().Value(sessionKey{}).(session)
	order, created, err := s.installOrder(r.Context(), ss, workerID, input.ModelID, input.IdempotencyKey, "")
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

// abandonStalledOrder cancels the open install order of a model that stopped moving, and forgets its row, so that a
// new order can replace it. A late report of the old order is then refused as stale. It reports whether it did.
func (s *Server) abandonStalledOrder(ctx context.Context, workerID, modelID string) (bool, error) {
	abandoned := false
	err := s.withVoiceModelLock(ctx, func(tx pgx.Tx) error {
		var orderID string
		err := tx.QueryRow(ctx, `SELECT order_id FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2 AND order_id IS NOT NULL`, workerID, modelID).Scan(&orderID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		stalled, err := orderStalled(ctx, tx, orderID)
		if err != nil || !stalled {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE worker_model_orders SET status='cancelled',finished_at=now() WHERE id=$1 AND status IN ('pending','acknowledged')`, orderID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, modelID)
		abandoned = err == nil
		return err
	})
	return abandoned, err
}
