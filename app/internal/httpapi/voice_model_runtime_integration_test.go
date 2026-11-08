//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type gotRuntimeOrder struct {
	OrderID       string `json:"order_id"`
	Action        string `json:"action"`
	ModelID       string `json:"model_id"`
	RuntimeIntent string `json:"runtime_intent"`
}

type gotRuntimeBinding struct {
	WorkerID        string `json:"worker_id"`
	ModelID         string `json:"model_id"`
	Digest          string `json:"digest"`
	PreviousDigest  string `json:"previous_digest"`
	PreviousRevoked bool   `json:"previous_revoked"`
	Pending         *struct {
		OrderID string `json:"order_id"`
		Intent  string `json:"intent"`
		Version string `json:"version"`
	} `json:"pending"`
	History []struct {
		Intent  string `json:"intent"`
		Version string `json:"version"`
		State   string `json:"state"`
		Cause   string `json:"cause_code"`
		Ignored bool   `json:"ignored"`
		Reason  string `json:"reason"`
	} `json:"history"`
}

func (h *engineOrderHarness) binding(workerID, modelID string) gotRuntimeBinding {
	h.t.Helper()
	var out struct {
		Bindings []gotRuntimeBinding `json:"bindings"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines/bindings", nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	for _, b := range out.Bindings {
		if b.WorkerID == workerID && b.ModelID == modelID {
			return b
		}
	}
	h.t.Fatalf("no binding for %s on %s: %+v", modelID, workerID, out)
	return gotRuntimeBinding{}
}

func (h *engineOrderHarness) runtimeOrder(workerID, modelID, digest, intent, key string, want int) gotRuntimeOrder {
	h.t.Helper()
	body := h.adminCall(http.MethodPost, "/api/admin/voice-model-runtime-orders", map[string]any{"worker_id": workerID, "model_id": modelID, "digest": digest, "intent": intent, "idempotency_key": key}, want)
	var o gotRuntimeOrder
	if want < 300 {
		if err := json.Unmarshal(body, &o); err != nil {
			h.t.Fatal(err)
		}
	}
	return o
}

// modelAck acknowledges a model order (the engine harness has its own ack for engine orders).
func (h *engineOrderHarness) modelAck(orderID string, accepted bool, cause string, want int) {
	h.t.Helper()
	h.voiceHarness.ack(orderID, accepted, cause, want)
}

func (h *engineOrderHarness) modelReport(orderID, modelID, state, cause string, want int) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": orderID, "model_id": modelID, "state": state, "cause_code": cause, "bytes_received": 0, "bytes_total": 0, "occupied_bytes": 10}, "", want)
}

func (h *engineOrderHarness) modelState(workerID, modelID string) (string, int64) {
	h.t.Helper()
	var state string
	var received int64
	if err := h.pool.QueryRow(h.t.Context(), `SELECT state,bytes_received FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, modelID).Scan(&state, &received); err != nil {
		h.t.Fatal(err)
	}
	return state, received
}

// pulled is what the Worker receives on its usual pull: its orders and the models Snitch knows as ready on it.
func (h *engineOrderHarness) pulled() ([]gotRuntimeOrder, []string) {
	h.t.Helper()
	var out struct {
		Orders []gotRuntimeOrder `json:"orders"`
		Held   []string          `json:"held"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, "/api/worker/voice-models", nil, "", http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Orders, out.Held
}

func testVoiceModelRuntimeUpdate(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	const model = "runtime-update-model"
	if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,'fingerprint','nemo-titanet','synthetic/runtime-update','0123456789abcdef0123456789abcdef01234567')`, model); err != nil {
		t.Fatal(err)
	}
	worker := h.fixtureWorkerID()
	v1, v2, v3, v4 := h.sign("1.0.0"), h.sign("2.0.0"), h.sign("3.0.0"), h.sign("4.0.0")

	// The model is installed with the first release and Ready: Snitch records what it is bound to.
	install := h.adminCall(http.MethodPost, "/api/admin/workers/"+worker+"/voice-model-orders", map[string]any{"model_id": model, "idempotency_key": "runtime-install-0001", "runtime_digest": v1}, http.StatusCreated)
	var first modelOrder
	if err := json.Unmarshal(install, &first); err != nil {
		t.Fatal(err)
	}
	h.modelAck(first.OrderID, true, "", http.StatusNoContent)
	h.modelReport(first.OrderID, model, "ready", "", http.StatusNoContent)
	if b := h.binding(worker, model); b.Digest != v1 || b.PreviousDigest != "" || b.Pending != nil {
		t.Fatalf("binding after install: %+v", b)
	}

	// Rights and refusals: nothing is recorded for a refusal.
	member := h.newUser("runtime-update-member")
	h.userCall(member, http.MethodPost, "/api/admin/voice-model-runtime-orders", map[string]any{"worker_id": worker, "model_id": model, "digest": v2, "intent": "update", "idempotency_key": "runtime-denied-0001"}, http.StatusForbidden)
	h.runtimeOrder(worker, model, v1, "update", "runtime-same-0001", http.StatusConflict)      // already bound
	h.runtimeOrder(worker, model, v2, "rollback", "runtime-norollback-1", http.StatusConflict) // never held on this Worker
	h.runtimeOrder(worker, model, v2, "other", "runtime-intent-0001", http.StatusBadRequest)
	h.runtimeOrder(worker, "unknown-model", v2, "update", "runtime-unknown-001", http.StatusBadRequest)
	revoked := h.sign("9.0.0")
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+revoked, nil, http.StatusOK)
	h.runtimeOrder(worker, model, revoked, "update", "runtime-revoked-001", http.StatusConflict)
	if n := h.count(`SELECT count(*) FROM worker_model_orders WHERE runtime_intent<>'install'`); n != 0 {
		t.Fatalf("refusals recorded %d orders", n)
	}

	// An update is an explicit order, observable in the history; the model stays Ready on its binding.
	update := h.runtimeOrder(worker, model, v2, "update", "runtime-update-0001", http.StatusCreated)
	if update.Action != "install" || update.ModelID != model {
		t.Fatalf("update order: %+v", update)
	}
	if b := h.binding(worker, model); b.Digest != v1 || b.Pending == nil || b.Pending.Intent != "update" || b.Pending.Version != "2.0.0" || len(b.History) != 1 || b.History[0].State != "requested" {
		t.Fatalf("binding with an update in progress: %+v", b)
	}
	// Duplicates: the same key replays the recorded order; another key while one is open is refused.
	if again := h.runtimeOrder(worker, model, v2, "update", "runtime-update-0001", http.StatusOK); again.OrderID != update.OrderID {
		t.Fatalf("replay created another order: %+v", again)
	}
	h.runtimeOrder(worker, model, v3, "update", "runtime-update-0001", http.StatusConflict) // key reused with other content
	h.runtimeOrder(worker, model, v3, "update", "runtime-update-0002", http.StatusConflict) // already one open
	if n := h.count(`SELECT count(*) FROM worker_model_orders WHERE runtime_intent='update'`); n != 1 {
		t.Fatalf("duplicates recorded %d update orders", n)
	}

	// Dispatch during the replacement: the model stays Ready (no download state, no weights traffic) until the
	// new binding is proven, and the Worker is still told it holds it. Orders carry the intent and the manifest.
	orders, held := h.pulled()
	if len(orders) != 1 || orders[0].OrderID != update.OrderID || orders[0].RuntimeIntent != "update" || !slices.Contains(held, model) {
		t.Fatalf("pull: %+v %v", orders, held)
	}
	h.modelAck(update.OrderID, true, "", http.StatusNoContent)
	h.modelReport(update.OrderID, model, "downloading", "", http.StatusNoContent)
	h.modelReport(update.OrderID, model, "verifying", "", http.StatusNoContent)
	if state, received := h.modelState(worker, model); state != "ready" || received != 0 {
		t.Fatalf("model row moved during the replacement: %s %d", state, received)
	}
	if _, held = h.pulled(); !slices.Contains(held, model) {
		t.Fatal("the model is no longer held while its runtime is replaced")
	}
	if b := h.binding(worker, model); b.Digest != v1 {
		t.Fatalf("binding published before validation: %+v", b)
	}

	// Failure: the previous binding stays, the cause is kept, and a late "ready" cannot publish afterwards.
	h.modelReport(update.OrderID, model, "failed", "runtime_incompatible", http.StatusNoContent)
	b := h.binding(worker, model)
	if b.Digest != v1 || b.Pending != nil || b.History[0].State != "failed" || b.History[0].Cause != "runtime_incompatible" {
		t.Fatalf("binding after a failed update: %+v", b)
	}
	if state, _ := h.modelState(worker, model); state != "ready" {
		t.Fatalf("a failed update changed the model: %s", state)
	}
	h.modelReport(update.OrderID, model, "ready", "", http.StatusNoContent)
	b = h.binding(worker, model)
	if b.Digest != v1 || !b.History[0].Ignored || b.History[0].Reason != "order_finished" {
		t.Fatalf("late report: %+v", b)
	}

	// Restart between validation and publication: the order is still open and delivered again, nothing is
	// published; only the ready report publishes, once.
	retry := h.runtimeOrder(worker, model, v2, "update", "runtime-update-0003", http.StatusCreated)
	h.modelAck(retry.OrderID, true, "", http.StatusNoContent)
	h.modelReport(retry.OrderID, model, "verifying", "", http.StatusNoContent)
	orders, _ = h.pulled()
	if len(orders) != 1 || orders[0].OrderID != retry.OrderID || orders[0].RuntimeIntent != "update" {
		t.Fatalf("redelivery after a restart: %+v", orders)
	}
	if b := h.binding(worker, model); b.Digest != v1 {
		t.Fatalf("restart published the binding: %+v", b)
	}
	h.modelReport(retry.OrderID, model, "ready", "", http.StatusNoContent)
	h.modelReport(retry.OrderID, model, "ready", "", http.StatusNoContent) // repeated outcome, no second effect
	b = h.binding(worker, model)
	if b.Digest != v2 || b.PreviousDigest != v1 || b.Pending != nil {
		t.Fatalf("binding after the update: %+v", b)
	}
	if n := h.count(`SELECT count(*) FROM worker_model_runtime_events WHERE order_id=$1 AND state='published'`, retry.OrderID); n != 1 {
		t.Fatalf("published %d times", n)
	}
	if state, received := h.modelState(worker, model); state != "ready" || received != 0 {
		t.Fatalf("model after the update: %s %d", state, received)
	}

	// Rollback: only to a version the Worker already holds and that is still approved.
	h.runtimeOrder(worker, model, v3, "rollback", "runtime-rollback-0001", http.StatusConflict) // never installed here
	back := h.runtimeOrder(worker, model, v1, "rollback", "runtime-rollback-0002", http.StatusCreated)
	if back.RuntimeIntent != "rollback" {
		t.Fatalf("rollback order: %+v", back)
	}
	h.modelAck(back.OrderID, true, "", http.StatusNoContent)
	h.modelReport(back.OrderID, model, "verifying", "", http.StatusNoContent)
	h.modelReport(back.OrderID, model, "ready", "", http.StatusNoContent)
	if b = h.binding(worker, model); b.Digest != v1 || b.PreviousDigest != v2 || b.History[0].Intent != "rollback" || b.History[0].State != "published" {
		t.Fatalf("binding after the rollback: %+v", b)
	}

	// A release revoked after it was installed keeps no way back: no silent reactivation, the binding serves on.
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+v2, nil, http.StatusOK)
	h.runtimeOrder(worker, model, v2, "rollback", "runtime-rollback-0003", http.StatusConflict)
	if b = h.binding(worker, model); b.Digest != v1 || !b.PreviousRevoked || b.Pending != nil {
		t.Fatalf("binding after a revocation: %+v", b)
	}

	// A queued update whose release is revoked before delivery is cancelled and publishes nothing, even if the
	// Worker reports late.
	queued := h.runtimeOrder(worker, model, v4, "update", "runtime-update-0004", http.StatusCreated)
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+v4, nil, http.StatusOK)
	if orders, _ = h.pulled(); len(orders) != 0 {
		t.Fatalf("a revoked queued update is delivered: %+v", orders)
	}
	h.modelReport(queued.OrderID, model, "ready", "", http.StatusNoContent)
	if b = h.binding(worker, model); b.Digest != v1 || !b.History[0].Ignored || b.History[0].Reason != "order_cancelled" {
		t.Fatalf("cancelled update: %+v", b)
	}

	// A model that is not Ready cannot be updated.
	if _, err := pool.Exec(t.Context(), `UPDATE worker_voice_models SET state='failed',cause_code='disk_full' WHERE worker_id=$1 AND model_id=$2`, worker, model); err != nil {
		t.Fatal(err)
	}
	h.runtimeOrder(worker, model, v3, "update", "runtime-update-0005", http.StatusConflict)
}
