package httpapi

import (
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
		var status, modelID, revision string
		err := tx.QueryRow(r.Context(), `SELECT status,model_id,revision FROM worker_model_orders WHERE id=$1 AND worker_id=$2`, input.OrderID, worker.ID).Scan(&status, &modelID, &revision)
		if errors.Is(err, pgx.ErrNoRows) {
			return orderFailure{http.StatusNotFound, "voice_order_not_found"}
		}
		if err != nil {
			return err
		}
		if modelID != input.ModelID || input.State == "removed" || (input.Revision != "" && input.Revision != revision) {
			return orderFailure{http.StatusBadRequest, "invalid_voice_model_status"}
		}
		if status == "done" || status == "failed" || status == "refused" {
			if (status == "done" && input.State == "ready") || (status == "failed" && failed) {
				return nil
			}
			return orderFailure{http.StatusConflict, "stale_voice_order"}
		}
		next := "acknowledged"
		switch input.State {
		case "ready":
			next = "done"
		case "failed":
			next = "failed"
		}
		if _, err = tx.Exec(r.Context(), `UPDATE worker_model_orders SET status=$2,cause_code=$3,acked_at=COALESCE(acked_at,now()),finished_at=CASE WHEN $2 IN ('done','failed') THEN now() END WHERE id=$1`, input.OrderID, next, input.CauseCode); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO worker_voice_models(worker_id,model_id,state,cause_code,bytes_received,bytes_total,occupied_bytes,revision,order_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT(worker_id,model_id) DO UPDATE SET state=EXCLUDED.state,cause_code=EXCLUDED.cause_code,bytes_received=EXCLUDED.bytes_received,bytes_total=EXCLUDED.bytes_total,occupied_bytes=EXCLUDED.occupied_bytes,revision=EXCLUDED.revision,order_id=EXCLUDED.order_id,retry_requested=false,updated_at=now()`,
			worker.ID, input.ModelID, input.State, input.CauseCode, input.BytesReceived, input.BytesTotal, input.OccupiedBytes, revision, input.OrderID)
		return err
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
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
	var state string
	err := s.Pool.QueryRow(r.Context(), `SELECT state FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, input.ModelID).Scan(&state)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	if state != "failed" {
		fail(w, http.StatusConflict, "voice_model_not_failed")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	order, created, err := s.installOrder(r.Context(), ss, workerID, input.ModelID, input.IdempotencyKey)
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
