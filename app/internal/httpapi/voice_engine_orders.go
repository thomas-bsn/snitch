package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/auth"
)

// Explicit engine orders (#349). An Administrator orders an approved release on chosen Workers; the order is
// delivered on the Worker's usual pull (next to the model orders), acknowledged and reported by the Worker.
// An order is the only thing that installs a runtime: reconciling an inventory observes and never orders.
//
// Two identities stay apart. The signed identity is the registry digest (the hash of the signed payload) and is
// what Snitch and the order carry. The local identity is the canonical hash the Worker computes for the
// runtime it holds on its volume; it is stored beside the digest, only as the Worker's proof of presence.

var engineStates = map[string]int{"requested": 0, "downloading_runtime": 1, "installing": 2, "verifying": 3, "ready": 4, "failed": 4}
var engineFailureCauses = map[string]bool{"disk_full": true, "runtime_incompatible": true, "hf_network": true, "invalid_order": true, "runtime_missing": true, "cancelled": true}
var localIdentity = regexp.MustCompile(`^[0-9a-f]{64}$`)

type engineOrderView struct {
	OrderID        string          `json:"order_id"`
	WorkerID       string          `json:"worker_id"`
	Digest         string          `json:"digest"`
	Engine         string          `json:"engine"`
	Version        string          `json:"version"`
	Intent         string          `json:"intent"`
	Generation     int64           `json:"generation"`
	IdempotencyKey string          `json:"idempotency_key"`
	Status         string          `json:"status"`
	Runtime        json.RawMessage `json:"runtime"`
	CreatedAt      time.Time       `json:"created_at"`
	// CancelRequested asks the Worker to stop at its next safe point (#351); the Worker confirms by reporting failed/cancelled.
	CancelRequested bool `json:"cancel_requested"`
}

const engineOrderColumns = `id,worker_id,digest,engine,version,intent,generation,idempotency_key,status,manifest,created_at,cancel_requested_at IS NOT NULL`

func scanEngineOrder(row pgx.Row) (engineOrderView, error) {
	var o engineOrderView
	err := row.Scan(&o.OrderID, &o.WorkerID, &o.Digest, &o.Engine, &o.Version, &o.Intent, &o.Generation, &o.IdempotencyKey, &o.Status, &o.Runtime, &o.CreatedAt, &o.CancelRequested)
	return o, err
}

func (s *Server) registerVoiceEngineOrderRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/admin/voice-engine-orders", s.protected(true, http.HandlerFunc(s.createVoiceEngineOrders)))
	mux.Handle("GET /api/admin/voice-engines/workers", s.protected(true, http.HandlerFunc(s.listEnginesByWorker)))
	s.registerVoiceEngineRecoveryRoutes(mux)
	mux.Handle("POST /api/worker/voice-engine-orders/{id}/ack", s.workerOnly(http.HandlerFunc(s.ackWorkerEngineOrder)))
	mux.Handle("POST /api/worker/voice-engines/status", s.workerOnly(http.HandlerFunc(s.postWorkerEngineStatus)))
	mux.Handle("POST /api/worker/voice-engines/inventory", s.workerOnly(http.HandlerFunc(s.postWorkerEngineInventory)))
}

type engineOrderResult struct {
	WorkerID  string           `json:"worker_id"`
	Status    string           `json:"status"`
	CauseCode string           `json:"cause_code,omitempty"`
	Order     *engineOrderView `json:"order,omitempty"`
}

// createVoiceEngineOrders orders one release on several Workers. Each Worker is decided on its own: a refusal
// for one never hides or cancels the order of another.
func (s *Server) createVoiceEngineOrders(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Digest string `json:"digest"`
		Intent string `json:"intent"`
		Orders []struct {
			WorkerID       string `json:"worker_id"`
			IdempotencyKey string `json:"idempotency_key"`
		} `json:"orders"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Intent != "install" || len(input.Orders) == 0 || len(input.Orders) > 100 {
		fail(w, http.StatusBadRequest, "invalid_engine_order")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	results := make([]engineOrderResult, 0, len(input.Orders))
	for _, item := range input.Orders {
		order, created, err := s.installEngineOrder(r.Context(), ss, item.WorkerID, input.Digest, item.IdempotencyKey)
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

func engineEvent(ctx context.Context, tx pgx.Tx, workerID, digest, orderID string, generation int64, state, cause string, ignored bool, reason string) error {
	_, err := tx.Exec(ctx, `INSERT INTO worker_engine_events(worker_id,digest,order_id,generation,state,cause_code,ignored,reason) VALUES($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8)`, workerID, digest, orderID, generation, state, cause, ignored, reason)
	return err
}

// installEngineOrder records an order to install a release on a Worker, or returns the one already recorded
// under the same key. It runs under the voice model lock like the model orders.
func (s *Server) installEngineOrder(ctx context.Context, author session, workerID, digest, key string) (engineOrderView, bool, error) {
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
		// The key first: a replay is answered with the recorded order whatever changed since.
		prior, err := scanEngineOrder(tx.QueryRow(ctx, `SELECT `+engineOrderColumns+` FROM worker_engine_orders WHERE idempotency_key=$1`, key))
		if err == nil {
			if prior.WorkerID != workerID || prior.Digest != digest || prior.Intent != "install" {
				return orderFailure{http.StatusConflict, "idempotency_key_reused"}
			}
			out = prior
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var manifest json.RawMessage
		var engine, version string
		var revoked bool
		err = tx.QueryRow(ctx, `SELECT manifest,engine,version,revoked FROM voice_engine_registry WHERE digest=$1`, digest).Scan(&manifest, &engine, &version, &revoked)
		if errors.Is(err, pgx.ErrNoRows) {
			return orderFailure{http.StatusNotFound, "engine_not_found"}
		}
		if err != nil {
			return err
		}
		if revoked {
			return orderFailure{http.StatusConflict, "engine_release_unavailable"}
		}
		var open bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_engine_orders WHERE worker_id=$1 AND engine=$2 AND status IN ('pending','acknowledged'))`, workerID, engine).Scan(&open); err != nil {
			return err
		}
		if open {
			return orderFailure{http.StatusConflict, "engine_order_in_progress"}
		}
		var state string
		switch err = tx.QueryRow(ctx, `SELECT state FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2`, workerID, digest).Scan(&state); {
		case errors.Is(err, pgx.ErrNoRows), err == nil && state == "failed":
		case err != nil:
			return err
		case state == "ready":
			return orderFailure{http.StatusConflict, "engine_already_installed"}
		default:
			return orderFailure{http.StatusConflict, "engine_order_in_progress"}
		}
		var generation int64
		if err = tx.QueryRow(ctx, `SELECT COALESCE(max(generation),0)+1 FROM worker_engine_orders WHERE worker_id=$1 AND engine=$2`, workerID, engine).Scan(&generation); err != nil {
			return err
		}
		id := "ord_" + auth.Token()[:32]
		if _, err = tx.Exec(ctx, `INSERT INTO worker_engine_orders(id,worker_id,digest,engine,version,intent,manifest,generation,idempotency_key,author_id,author_name) VALUES($1,$2,$3,$4,$5,'install',$6,$7,$8,$9,$10)`,
			id, workerID, digest, engine, version, manifest, generation, key, author.User.ID, author.User.Username); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO worker_engine_inventory(worker_id,digest,engine,version,state,generation,order_id) VALUES($1,$2,$3,$4,'requested',$5,$6)
			ON CONFLICT(worker_id,digest) DO UPDATE SET state='requested',cause_code='',bytes_received=0,bytes_total=0,generation=EXCLUDED.generation,order_id=EXCLUDED.order_id,local_identity='',updated_at=now()`,
			workerID, digest, engine, version, generation, id); err != nil {
			return err
		}
		if err = engineEvent(ctx, tx, workerID, digest, id, generation, "requested", "", false, ""); err != nil {
			return err
		}
		created = true
		out, err = scanEngineOrder(tx.QueryRow(ctx, `SELECT `+engineOrderColumns+` FROM worker_engine_orders WHERE id=$1`, id))
		return err
	})
	if created {
		s.Log.Info("voice engine install ordered", "order_id", out.OrderID, "worker_id", workerID, "digest", digest, "generation", out.Generation, "author_id", author.User.ID)
	}
	return out, created, err
}

// pendingEngineOrders are the engine orders a Worker still has to run, in order.
func (s *Server) pendingEngineOrders(ctx context.Context, workerID string) ([]engineOrderView, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+engineOrderColumns+` FROM worker_engine_orders WHERE worker_id=$1 AND status IN ('pending','acknowledged') ORDER BY created_at,id`, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []engineOrderView{}
	for rows.Next() {
		o, err := scanEngineOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ackWorkerEngineOrder records that the Worker received the order, or refused it after its own checks.
func (s *Server) ackWorkerEngineOrder(w http.ResponseWriter, r *http.Request) {
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
		var status, digest string
		var generation int64
		err := tx.QueryRow(r.Context(), `SELECT status,digest,generation FROM worker_engine_orders WHERE id=$1 AND worker_id=$2`, r.PathValue("id"), worker.ID).Scan(&status, &digest, &generation)
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
			_, err = tx.Exec(r.Context(), `UPDATE worker_engine_orders SET status='acknowledged',acked_at=now() WHERE id=$1`, r.PathValue("id"))
			return err
		}
		if _, err = tx.Exec(r.Context(), `UPDATE worker_engine_orders SET status='refused',cause_code='invalid_order',acked_at=now(),finished_at=now() WHERE id=$1`, r.PathValue("id")); err != nil {
			return err
		}
		tag, err := tx.Exec(r.Context(), `UPDATE worker_engine_inventory SET state='failed',cause_code='invalid_order',updated_at=now() WHERE worker_id=$1 AND digest=$2 AND order_id=$3`, worker.ID, digest, r.PathValue("id"))
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		return engineEvent(r.Context(), tx, worker.ID, digest, r.PathValue("id"), generation, "failed", "invalid_order", false, "")
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// postWorkerEngineStatus is what a Worker really holds of a release, in answer to an order. A report that is
// late (older generation, cancelled or finished order, a step already passed) is ignored and journalled: it
// never moves the state back.
func (s *Server) postWorkerEngineStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		OrderID       string `json:"order_id"`
		Digest        string `json:"digest"`
		State         string `json:"state"`
		CauseCode     string `json:"cause_code"`
		BytesReceived int64  `json:"bytes_received"`
		BytesTotal    int64  `json:"bytes_total"`
		LocalIdentity string `json:"local_identity"`
	}
	if !decode(w, r, &input) {
		return
	}
	rank, known := engineStates[input.State]
	failed := input.State == "failed"
	validCause := (failed && engineFailureCauses[input.CauseCode]) || (!failed && input.CauseCode == "")
	validIdentity := localIdentity.MatchString(input.LocalIdentity) || (input.LocalIdentity == "" && input.State != "ready")
	if !known || input.State == "requested" || !validCause || !validIdentity || input.BytesReceived < 0 || input.BytesTotal < 0 || !engineDigest.MatchString(input.Digest) {
		fail(w, http.StatusBadRequest, "invalid_engine_status")
		return
	}
	worker := currentWorker(r)
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		var status, digest, engine string
		var generation int64
		var cancelRequested bool
		err := tx.QueryRow(r.Context(), `SELECT status,digest,engine,generation,cancel_requested_at IS NOT NULL FROM worker_engine_orders WHERE id=$1 AND worker_id=$2`, input.OrderID, worker.ID).Scan(&status, &digest, &engine, &generation, &cancelRequested)
		if errors.Is(err, pgx.ErrNoRows) {
			return orderFailure{http.StatusNotFound, "voice_order_not_found"}
		}
		if err != nil {
			return err
		}
		if digest != input.Digest {
			return orderFailure{http.StatusBadRequest, "invalid_engine_status"}
		}
		ignore := func(reason string) error {
			s.Log.Warn("late engine report ignored", "order_id", input.OrderID, "worker_id", worker.ID, "state", input.State, "reason", reason)
			return engineEvent(r.Context(), tx, worker.ID, digest, input.OrderID, generation, input.State, input.CauseCode, true, reason)
		}
		if status == "cancelled" {
			if failed && input.CauseCode == "cancelled" {
				return nil // the Worker repeats its confirmation of a cancellation already recorded
			}
			return ignore("order_cancelled")
		}
		if status == "done" || status == "failed" || status == "refused" {
			// Repeating the recorded outcome is accepted without effect, anything else is late.
			if (status == "done" && input.State == "ready") || (status == "failed" && failed) {
				return nil
			}
			return ignore("order_finished")
		}
		var latest int64
		var current string
		if err = tx.QueryRow(r.Context(), `SELECT COALESCE(max(generation),0) FROM worker_engine_orders WHERE worker_id=$1 AND engine=$2`, worker.ID, engine).Scan(&latest); err != nil {
			return err
		}
		if generation < latest {
			return ignore("stale_generation")
		}
		if err = tx.QueryRow(r.Context(), `SELECT state FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2 AND order_id=$3`, worker.ID, digest, input.OrderID).Scan(&current); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ignore("order_not_current")
			}
			return err
		}
		if rank < engineStates[current] {
			return ignore("stale_step")
		}
		next := "acknowledged"
		switch input.State {
		case "ready":
			next = "done"
		case "failed":
			next = "failed"
			if input.CauseCode == "cancelled" {
				// Only the Worker's confirmation of a requested cancellation ends an order as cancelled.
				if !cancelRequested {
					return ignore("cancel_not_requested")
				}
				next = "cancelled"
			}
		}
		if _, err = tx.Exec(r.Context(), `UPDATE worker_engine_orders SET status=$2,cause_code=$3,acked_at=COALESCE(acked_at,now()),finished_at=CASE WHEN $2 IN ('done','failed','cancelled') THEN now() END WHERE id=$1`, input.OrderID, next, input.CauseCode); err != nil {
			return err
		}
		if _, err = tx.Exec(r.Context(), `UPDATE worker_engine_inventory SET state=$4,cause_code=$5,bytes_received=$6,bytes_total=$7,local_identity=CASE WHEN $8<>'' THEN $8 ELSE local_identity END,updated_at=now() WHERE worker_id=$1 AND digest=$2 AND order_id=$3`,
			worker.ID, digest, input.OrderID, input.State, input.CauseCode, input.BytesReceived, input.BytesTotal, input.LocalIdentity); err != nil {
			return err
		}
		if current == input.State {
			return nil // progress inside one step is not a new history entry
		}
		return engineEvent(r.Context(), tx, worker.ID, digest, input.OrderID, generation, input.State, input.CauseCode, false, "")
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// postWorkerEngineInventory reconciles what the Worker proves it holds. It only observes: a runtime that was
// ready and is no longer on the volume loses its ready state. It never orders, downloads or installs anything,
// and a runtime that appears again must be ordered again explicitly.
func (s *Server) postWorkerEngineInventory(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Runtimes []struct {
			Engine        string `json:"engine"`
			LocalIdentity string `json:"local_identity"`
		} `json:"runtimes"`
	}
	if !decode(w, r, &input) {
		return
	}
	if len(input.Runtimes) > 512 {
		fail(w, http.StatusBadRequest, "invalid_engine_inventory")
		return
	}
	present := make([]string, 0, len(input.Runtimes))
	for _, rt := range input.Runtimes {
		if !localIdentity.MatchString(rt.LocalIdentity) {
			fail(w, http.StatusBadRequest, "invalid_engine_inventory")
			return
		}
		present = append(present, rt.LocalIdentity)
	}
	worker := currentWorker(r)
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		rows, err := tx.Query(r.Context(), `UPDATE worker_engine_inventory SET state='failed',cause_code='runtime_missing',updated_at=now() WHERE worker_id=$1 AND state='ready' AND local_identity<>'' AND NOT (local_identity = ANY($2)) RETURNING digest,order_id,generation`, worker.ID, present)
		if err != nil {
			return err
		}
		type lost struct {
			digest, order string
			generation    int64
		}
		var gone []lost
		for rows.Next() {
			var l lost
			var order *string
			if err = rows.Scan(&l.digest, &order, &l.generation); err != nil {
				rows.Close()
				return err
			}
			if order != nil {
				l.order = *order
			}
			gone = append(gone, l)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return err
		}
		for _, l := range gone {
			s.Log.Warn("engine runtime lost on Worker", "worker_id", worker.ID, "digest", l.digest)
			if err = engineEvent(r.Context(), tx, worker.ID, l.digest, l.order, l.generation, "failed", "runtime_missing", false, "inventory"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type engineHistoryEntry struct {
	State      string    `json:"state"`
	CauseCode  string    `json:"cause_code"`
	Generation int64     `json:"generation"`
	Ignored    bool      `json:"ignored"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

type workerEngineEntry struct {
	Digest        string `json:"digest"`
	Engine        string `json:"engine"`
	Version       string `json:"version"`
	Kind          string `json:"kind"`
	State         string `json:"state"`
	CauseCode     string `json:"cause_code"`
	BytesReceived int64  `json:"bytes_received"`
	BytesTotal    int64  `json:"bytes_total"`
	// CancelRequested: a cooperative cancellation is recorded and the Worker has not confirmed it yet.
	CancelRequested bool `json:"cancel_requested"`
	// Cleanup is "" (none requested), "requested" or "done": the explicit removal of the incomplete files of a failed install.
	Cleanup    string               `json:"cleanup"`
	OrderID    string               `json:"order_id"`
	Generation int64                `json:"generation"`
	Revoked    bool                 `json:"revoked"`
	AuthorName string               `json:"author_name"`
	UpdatedAt  time.Time            `json:"updated_at"`
	History    []engineHistoryEntry `json:"history"`
}

type workerEnginesRow struct {
	WorkerID  string              `json:"worker_id"`
	Name      string              `json:"name"`
	Connected bool                `json:"connected"`
	Engines   []workerEngineEntry `json:"engines"`
}

const engineHistoryLimit = 20

// listEnginesByWorker is the Engines view: for each Worker, only the releases it has been ordered or holds,
// with their real state, their cause and their history.
func (s *Server) listEnginesByWorker(w http.ResponseWriter, r *http.Request) {
	items, err := s.Workers.List(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT i.worker_id,i.digest,i.engine,i.version,COALESCE(g.manifest->>'kind',''),i.state,i.cause_code,i.bytes_received,i.bytes_total,COALESCE(o.cancel_requested_at IS NOT NULL AND o.status='acknowledged',false),CASE WHEN o.cleaned_at IS NOT NULL THEN 'done' WHEN o.cleanup_requested_at IS NOT NULL THEN 'requested' ELSE '' END,COALESCE(i.order_id,''),i.generation,g.revoked,COALESCE(o.author_name,''),i.updated_at
		FROM worker_engine_inventory i JOIN voice_engine_registry g ON g.digest=i.digest LEFT JOIN worker_engine_orders o ON o.id=i.order_id ORDER BY i.engine,i.version`)
	if err != nil {
		s.internal(w, err)
		return
	}
	byWorker := map[string][]workerEngineEntry{}
	for rows.Next() {
		var workerID string
		var e workerEngineEntry
		if err = rows.Scan(&workerID, &e.Digest, &e.Engine, &e.Version, &e.Kind, &e.State, &e.CauseCode, &e.BytesReceived, &e.BytesTotal, &e.CancelRequested, &e.Cleanup, &e.OrderID, &e.Generation, &e.Revoked, &e.AuthorName, &e.UpdatedAt); err != nil {
			rows.Close()
			s.internal(w, err)
			return
		}
		e.History = []engineHistoryEntry{}
		byWorker[workerID] = append(byWorker[workerID], e)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	out := struct {
		Workers []workerEnginesRow `json:"workers"`
	}{Workers: []workerEnginesRow{}}
	for _, item := range items {
		if item.RevokedAt != nil {
			continue
		}
		row := workerEnginesRow{WorkerID: item.ID, Name: item.Name, Connected: item.Connected, Engines: []workerEngineEntry{}}
		for _, e := range byWorker[item.ID] {
			hist, err := s.Pool.Query(r.Context(), `SELECT state,cause_code,generation,ignored,reason,created_at FROM worker_engine_events WHERE worker_id=$1 AND digest=$2 ORDER BY id DESC LIMIT $3`, item.ID, e.Digest, engineHistoryLimit)
			if err != nil {
				s.internal(w, err)
				return
			}
			for hist.Next() {
				var h engineHistoryEntry
				if err = hist.Scan(&h.State, &h.CauseCode, &h.Generation, &h.Ignored, &h.Reason, &h.CreatedAt); err != nil {
					hist.Close()
					s.internal(w, err)
					return
				}
				e.History = append(e.History, h)
			}
			hist.Close()
			row.Engines = append(row.Engines, e)
		}
		out.Workers = append(out.Workers, row)
	}
	reply(w, http.StatusOK, out)
}
