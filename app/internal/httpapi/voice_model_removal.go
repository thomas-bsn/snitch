package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/auth"
)

// Explicit removal of a model on one Worker, and the history of a Worker's orders (ADR 0038, #312).
//
// Removing is the Administrator's own decision: Snitch never deletes a model to make room on its own. It is
// refused while a job uses the model, and the screen says beforehand that no active model will remain.

type voiceModelHistoryItem struct {
	OrderID    string     `json:"order_id"`
	Kind       string     `json:"kind"`
	Action     string     `json:"action"`
	ModelID    string     `json:"model_id"`
	Label      string     `json:"label"`
	Repo       string     `json:"repo"`
	Revision   string     `json:"revision"`
	Status     string     `json:"status"`
	CauseCode  string     `json:"cause_code"`
	AuthorName string     `json:"author_name"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// removalOrder records a remove order for a model that is ready or failed on the Worker, or returns the one
// already recorded under the same key.
func (s *Server) removalOrder(ctx context.Context, author session, workerID, modelID, key string) (voiceModelOrderView, bool, error) {
	if !idempotencyKeyFormat.MatchString(key) {
		return voiceModelOrderView{}, false, orderFailure{http.StatusBadRequest, "invalid_idempotency_key"}
	}
	var out voiceModelOrderView
	created := false
	err := s.withVoiceModelLock(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, workerID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return orderFailure{http.StatusNotFound, "worker_not_found"}
		}
		var prior, priorWorker, priorModel string
		err := tx.QueryRow(ctx, `SELECT id,worker_id,model_id FROM worker_model_orders WHERE idempotency_key=$1`, key).Scan(&prior, &priorWorker, &priorModel)
		if err == nil {
			if priorWorker != workerID || priorModel != modelID {
				return orderFailure{http.StatusConflict, "idempotency_key_reused"}
			}
			out, err = orderByID(ctx, tx, prior)
			return err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		entry, ok, err := voiceCatalogByID(ctx, tx, modelID)
		if err != nil {
			return err
		}
		if !ok {
			return orderFailure{http.StatusBadRequest, "invalid_voice_model"}
		}
		var state, revision string
		switch err = tx.QueryRow(ctx, `SELECT state,COALESCE(NULLIF(revision,''),$3) FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, modelID, entry.Revision).Scan(&state, &revision); {
		case errors.Is(err, pgx.ErrNoRows):
			return orderFailure{http.StatusConflict, "voice_model_not_removable"}
		case err != nil:
			return err
		case state != "ready" && state != "failed":
			return orderFailure{http.StatusConflict, "voice_model_not_removable"}
		}
		// A job that still uses the model keeps it; the Administrator removes it once the job has ended.
		busy, err := voiceModelInUseIn(ctx, tx, entry)
		if err != nil {
			return err
		}
		if busy {
			return orderFailure{http.StatusConflict, "voice_model_in_use"}
		}
		// A replacement in progress owns its two models. The only way out is the explicit one: the new model
		// failed (typically for lack of disk), so the Administrator gives up the old one to make room.
		reps, err := replacementsIn(ctx, tx, `WHERE worker_id=$1 AND kind=$2`, workerID, entry.Kind)
		if err != nil {
			return err
		}
		if len(reps) == 1 {
			rep := reps[0]
			var next string
			err = tx.QueryRow(ctx, `SELECT state FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, rep.New).Scan(&next)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			abandon := rep.Old == modelID && rep.Phase == "preparing" && next == "failed"
			if !abandon {
				return orderFailure{http.StatusConflict, "voice_replacement_in_progress"}
			}
			if _, err = tx.Exec(ctx, `DELETE FROM worker_model_replacements WHERE worker_id=$1 AND kind=$2`, workerID, entry.Kind); err != nil {
				return err
			}
		}
		id := "ord_" + auth.Token()[:32]
		if _, err = tx.Exec(ctx, `INSERT INTO worker_model_orders(id,worker_id,kind,action,model_id,revision,idempotency_key,author_id,author_name) VALUES($1,$2,$3,'remove',$4,$5,$6,$7,$8)`,
			id, workerID, entry.Kind, entry.ID, revision, key, author.User.ID, author.User.Username); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE worker_voice_models SET state='removing',order_id=$3,updated_at=now() WHERE worker_id=$1 AND model_id=$2`, workerID, entry.ID, id); err != nil {
			return err
		}
		created = true
		out, err = orderByID(ctx, tx, id)
		return err
	})
	if created {
		s.Log.Info("voice model removal ordered", "order_id", out.OrderID, "worker_id", workerID, "model_id", modelID, "author_id", author.User.ID)
	}
	return out, created, err
}

func (s *Server) createVoiceModelRemoval(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ModelID        string `json:"model_id"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decode(w, r, &input) {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	order, created, err := s.removalOrder(r.Context(), ss, r.PathValue("id"), input.ModelID, input.IdempotencyKey)
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

// listVoiceModelOrders is the history of the orders of one Worker, newest first.
func (s *Server) listVoiceModelOrders(w http.ResponseWriter, r *http.Request) {
	workerID := r.PathValue("id")
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1)`, workerID).Scan(&exists); err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT o.id,o.kind,o.action,o.model_id,c.label,c.repo,o.revision,o.status,o.cause_code,o.author_name,o.created_at,o.finished_at
		FROM worker_model_orders o JOIN voice_model_catalog c ON c.id=o.model_id WHERE o.worker_id=$1 ORDER BY o.created_at DESC,o.id DESC LIMIT 100`, workerID)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	out := struct {
		Orders []voiceModelHistoryItem `json:"orders"`
	}{Orders: []voiceModelHistoryItem{}}
	for rows.Next() {
		var h voiceModelHistoryItem
		if err = rows.Scan(&h.OrderID, &h.Kind, &h.Action, &h.ModelID, &h.Label, &h.Repo, &h.Revision, &h.Status, &h.CauseCode, &h.AuthorName, &h.CreatedAt, &h.FinishedAt); err != nil {
			s.internal(w, err)
			return
		}
		out.Orders = append(out.Orders, h)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}
