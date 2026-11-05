package httpapi

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/auth"
)

// Orders to install a model on one Worker (ADR 0038). Snitch owns the desired state: an Administrator orders,
// Snitch records the order (idempotency key, author, time), the Worker pulls it on its usual outgoing request,
// acknowledges it and reports the real state, which lives in worker_voice_models. The Worker never decides
// on its own what to download.

var idempotencyKeyFormat = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,100}$`)

var voiceModelKinds = []string{"diarization", "fingerprint", "separation"}

// orderFailure is a refusal with its HTTP status and stable error code.
type orderFailure struct {
	Status int
	Code   string
}

func (f orderFailure) Error() string { return f.Code }

type voiceModelOrderView struct {
	OrderID        string `json:"order_id"`
	WorkerID       string `json:"worker_id"`
	Kind           string `json:"kind"`
	Action         string `json:"action"`
	ModelID        string `json:"model_id"`
	Engine         string `json:"engine"`
	Repo           string `json:"repo"`
	Revision       string `json:"revision"`
	IdempotencyKey string `json:"idempotency_key"`
	Status         string `json:"status"`
	// HFToken is set only on the first delivery of an order that downloads a gated model (ADR 0027).
	HFToken   string    `json:"hf_token,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// installOrder records an order to install modelID on a Worker, or returns the one already recorded under the
// same key. It runs under the voice model lock so that what it read is still true when it writes.
func (s *Server) installOrder(ctx context.Context, author session, workerID, modelID, key string) (voiceModelOrderView, bool, error) {
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
		// The key first: a replay is answered with the recorded order whatever changed since.
		var prior string
		var priorWorker, priorModel string
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
		if !ok || !entry.Enabled || !entry.Supported {
			return orderFailure{http.StatusBadRequest, "invalid_voice_model"}
		}
		var state string
		switch err = tx.QueryRow(ctx, `SELECT state FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, modelID).Scan(&state); {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return err
		case state == "ready":
			return orderFailure{http.StatusConflict, "voice_model_already_installed"}
		case state != "failed":
			return orderFailure{http.StatusConflict, "voice_model_order_in_progress"}
		}
		// At most one model per type and Worker: replacing one is a separate decision.
		var occupied bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_voice_models m JOIN voice_model_catalog c ON c.id=m.model_id WHERE m.worker_id=$1 AND c.kind=$2 AND m.model_id<>$3 AND m.state<>'failed')`, workerID, entry.Kind, modelID).Scan(&occupied); err != nil {
			return err
		}
		if occupied {
			return orderFailure{http.StatusConflict, "voice_model_type_occupied"}
		}
		id := "ord_" + auth.Token()[:32]
		if _, err = tx.Exec(ctx, `INSERT INTO worker_model_orders(id,worker_id,kind,action,model_id,revision,idempotency_key,author_id,author_name) VALUES($1,$2,$3,'install',$4,$5,$6,$7,$8)`,
			id, workerID, entry.Kind, entry.ID, entry.Revision, key, author.User.ID, author.User.Username); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO worker_voice_models(worker_id,model_id,state,cause_code,bytes_received,bytes_total,occupied_bytes,revision,order_id) VALUES($1,$2,'requested','',0,0,0,$3,$4)
			ON CONFLICT(worker_id,model_id) DO UPDATE SET state='requested',cause_code='',bytes_received=0,bytes_total=0,occupied_bytes=0,revision=EXCLUDED.revision,order_id=EXCLUDED.order_id,retry_requested=false,removal_requested=false,updated_at=now()`,
			workerID, entry.ID, entry.Revision, id); err != nil {
			return err
		}
		created = true
		out, err = orderByID(ctx, tx, id)
		return err
	})
	if created {
		s.Log.Info("voice model install ordered", "order_id", out.OrderID, "worker_id", workerID, "model_id", modelID, "revision", out.Revision, "author_id", author.User.ID)
	}
	return out, created, err
}

const orderColumns = `o.id,o.worker_id,o.kind,o.action,o.model_id,c.engine,c.repo,o.revision,o.idempotency_key,o.status,o.created_at`

func scanOrder(row pgx.Row) (voiceModelOrderView, error) {
	var o voiceModelOrderView
	err := row.Scan(&o.OrderID, &o.WorkerID, &o.Kind, &o.Action, &o.ModelID, &o.Engine, &o.Repo, &o.Revision, &o.IdempotencyKey, &o.Status, &o.CreatedAt)
	return o, err
}

func orderByID(ctx context.Context, db pgQuerier, id string) (voiceModelOrderView, error) {
	return scanOrder(db.QueryRow(ctx, `SELECT `+orderColumns+` FROM worker_model_orders o JOIN voice_model_catalog c ON c.id=o.model_id WHERE o.id=$1`, id))
}

func (s *Server) writeOrderFailure(w http.ResponseWriter, err error) bool {
	var f orderFailure
	if errors.As(err, &f) {
		fail(w, f.Status, f.Code)
		return true
	}
	return false
}

func (s *Server) createVoiceModelOrder(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ModelID        string `json:"model_id"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decode(w, r, &input) {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	order, created, err := s.installOrder(r.Context(), ss, r.PathValue("id"), input.ModelID, input.IdempotencyKey)
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

// pendingOrders are the orders a Worker still has to run. The central Hugging Face token travels once, with
// the first delivery of an order that downloads a gated model, and with no other order.
func (s *Server) pendingOrders(ctx context.Context, workerID string) ([]voiceModelOrderView, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+orderColumns+`,c.gated FROM worker_model_orders o JOIN voice_model_catalog c ON c.id=o.model_id WHERE o.worker_id=$1 AND o.status IN ('pending','acknowledged') ORDER BY o.created_at,o.id`, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []voiceModelOrderView{}
	var gated []bool
	for rows.Next() {
		var o voiceModelOrderView
		var g bool
		if err = rows.Scan(&o.OrderID, &o.WorkerID, &o.Kind, &o.Action, &o.ModelID, &o.Engine, &o.Repo, &o.Revision, &o.IdempotencyKey, &o.Status, &o.CreatedAt, &g); err != nil {
			return nil, err
		}
		out = append(out, o)
		gated = append(gated, g)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	var token string
	for i, o := range out {
		if o.Status == "pending" && gated[i] {
			if token == "" {
				state, _, err := s.hfRead(ctx)
				if err != nil {
					return nil, err
				}
				token = state.Token
			}
			out[i].HFToken = token
		}
	}
	return out, nil
}

// ackWorkerModelOrder records that the Worker received the order, or refused it after its own checks.
func (s *Server) ackWorkerModelOrder(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Accepted  bool   `json:"accepted"`
		CauseCode string `json:"cause_code"`
	}
	if !decode(w, r, &input) {
		return
	}
	if (input.Accepted && input.CauseCode != "") || (!input.Accepted && input.CauseCode != "invalid_order") {
		fail(w, http.StatusBadRequest, "invalid_voice_order_ack")
		return
	}
	worker := currentWorker(r)
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		var status, modelID, orderID string
		err := tx.QueryRow(r.Context(), `SELECT id,status,model_id FROM worker_model_orders WHERE id=$1 AND worker_id=$2`, r.PathValue("id"), worker.ID).Scan(&orderID, &status, &modelID)
		if errors.Is(err, pgx.ErrNoRows) {
			return orderFailure{http.StatusNotFound, "voice_order_not_found"}
		}
		if err != nil {
			return err
		}
		if status != "pending" {
			return nil // a replayed acknowledgement has no second effect
		}
		if input.Accepted {
			_, err = tx.Exec(r.Context(), `UPDATE worker_model_orders SET status='acknowledged',acked_at=now() WHERE id=$1`, orderID)
			return err
		}
		if _, err = tx.Exec(r.Context(), `UPDATE worker_model_orders SET status='refused',cause_code='invalid_order',acked_at=now(),finished_at=now() WHERE id=$1`, orderID); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `UPDATE worker_voice_models SET state='failed',cause_code='invalid_order',updated_at=now() WHERE worker_id=$1 AND model_id=$2 AND order_id=$3`, worker.ID, modelID, orderID)
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

type workerModelCell struct {
	Kind          string `json:"kind"`
	Supported     bool   `json:"supported"`
	ModelID       string `json:"model_id"`
	Label         string `json:"label"`
	Repo          string `json:"repo"`
	Revision      string `json:"revision"`
	State         string `json:"state"`
	CauseCode     string `json:"cause_code"`
	BytesReceived int64  `json:"bytes_received"`
	BytesTotal    int64  `json:"bytes_total"`
	OccupiedBytes int64  `json:"occupied_bytes"`
	OrderID       string `json:"order_id"`
}

type workerModelsRow struct {
	WorkerID  string            `json:"worker_id"`
	Name      string            `json:"name"`
	Connected bool              `json:"connected"`
	Models    []workerModelCell `json:"models"`
}

// listModelsByWorker is the minimal "Models per Worker" view: for each Worker, the model of each type.
func (s *Server) listModelsByWorker(w http.ResponseWriter, r *http.Request) {
	items, err := s.Workers.List(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	supported := map[string]bool{}
	catalog, err := voiceCatalogList(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	for _, e := range catalog {
		supported[e.Kind] = supported[e.Kind] || (e.Enabled && e.Supported)
	}
	// One cell per Worker and type: a model that is not failed wins over a failed attempt, then the latest.
	rows, err := s.Pool.Query(r.Context(), `SELECT DISTINCT ON (m.worker_id, c.kind) m.worker_id,c.kind,m.model_id,c.label,c.repo,m.revision,m.state,m.cause_code,m.bytes_received,m.bytes_total,m.occupied_bytes,COALESCE(m.order_id,'')
		FROM worker_voice_models m JOIN voice_model_catalog c ON c.id=m.model_id ORDER BY m.worker_id, c.kind, (m.state='failed'), m.updated_at DESC`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	held := map[[2]string]workerModelCell{}
	for rows.Next() {
		var workerID string
		var c workerModelCell
		if err = rows.Scan(&workerID, &c.Kind, &c.ModelID, &c.Label, &c.Repo, &c.Revision, &c.State, &c.CauseCode, &c.BytesReceived, &c.BytesTotal, &c.OccupiedBytes, &c.OrderID); err != nil {
			s.internal(w, err)
			return
		}
		held[[2]string{workerID, c.Kind}] = c
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	out := struct {
		Workers []workerModelsRow `json:"workers"`
	}{Workers: []workerModelsRow{}}
	for _, item := range items {
		if item.RevokedAt != nil {
			continue
		}
		row := workerModelsRow{WorkerID: item.ID, Name: item.Name, Connected: item.Connected, Models: []workerModelCell{}}
		for _, kind := range voiceModelKinds {
			c, ok := held[[2]string{item.ID, kind}]
			if !ok {
				c = workerModelCell{Kind: kind, State: "not_installed"}
			}
			c.Supported = supported[kind]
			row.Models = append(row.Models, c)
		}
		out.Workers = append(out.Workers, row)
	}
	reply(w, http.StatusOK, out)
}
