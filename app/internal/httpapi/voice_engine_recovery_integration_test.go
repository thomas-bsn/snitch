//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type recoveryEntry struct {
	Digest          string `json:"digest"`
	State           string `json:"state"`
	CauseCode       string `json:"cause_code"`
	Generation      int64  `json:"generation"`
	CancelRequested bool   `json:"cancel_requested"`
	Cleanup         string `json:"cleanup"`
	OrderID         string `json:"order_id"`
	History         []struct {
		Reason  string `json:"reason"`
		Ignored bool   `json:"ignored"`
	} `json:"history"`
}

func (h *engineOrderHarness) recovery(workerID, digest string) recoveryEntry {
	h.t.Helper()
	var out struct {
		Workers []struct {
			WorkerID string          `json:"worker_id"`
			Engines  []recoveryEntry `json:"engines"`
		} `json:"workers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines/workers", nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	for _, w := range out.Workers {
		if w.WorkerID != workerID {
			continue
		}
		for _, e := range w.Engines {
			if e.Digest == digest {
				return e
			}
		}
	}
	h.t.Fatalf("no entry for %s on %s", digest, workerID)
	return recoveryEntry{}
}

func (h *engineOrderHarness) reasons(workerID, digest, reason string) int {
	n := 0
	for _, ev := range h.recovery(workerID, digest).History {
		if ev.Reason == reason {
			n++
		}
	}
	return n
}

func (h *engineOrderHarness) action(workerID, digest, verb string, body any, want int) []byte {
	h.t.Helper()
	return h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-engines/"+digest+"/"+verb, body, want)
}

func (h *engineOrderHarness) status(raw []byte) string {
	var out struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Status
}

type pulledEngineOrder struct {
	OrderID         string `json:"order_id"`
	CancelRequested bool   `json:"cancel_requested"`
}

type pulledEngineCleanup struct {
	OrderID string          `json:"order_id"`
	Digest  string          `json:"digest"`
	Runtime json.RawMessage `json:"runtime"`
}

func (h *engineOrderHarness) pull() ([]pulledEngineOrder, []pulledEngineCleanup) {
	h.t.Helper()
	var out struct {
		EngineOrders   []pulledEngineOrder   `json:"engine_orders"`
		EngineCleanups []pulledEngineCleanup `json:"engine_cleanups"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, "/api/worker/voice-models", nil, "", http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.EngineOrders, out.EngineCleanups
}

func (h *engineOrderHarness) install(workerID, digest, key string) *gotEngineOrder {
	h.t.Helper()
	r := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": workerID, "idempotency_key": key})
	if r[0].Status != "created" {
		h.t.Fatalf("order refused: %+v", r[0])
	}
	return r[0].Order
}

func (h *engineOrderHarness) ack(orderID string) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+orderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
}

// asSecond runs fn with the second Worker's token.
func (h *engineOrderHarness) asSecond(fn func()) {
	token := h.token
	h.token = h.secondToken
	defer func() { h.token = token }()
	fn()
}

// testVoiceEngineRecovery covers #351: retry, replacement, cooperative cancellation and bounded cleanup.
func testVoiceEngineRecovery(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	worker, second := h.fixtureWorkerID(), h.secondID
	a, b, c := h.sign("1.0.0"), h.sign("2.0.0"), h.sign("3.0.0")

	// Retry: only from Failed, a new generation, idempotent under its key, never while an order is open.
	first := h.install(worker, a, "recovery-first-0001")
	h.action(worker, a, "retry", map[string]string{"idempotency_key": "recovery-retry-0001"}, http.StatusConflict)
	h.ack(first.OrderID)
	h.report(first.OrderID, a, "downloading_runtime", "", http.StatusNoContent)
	h.report(first.OrderID, a, "failed", "disk_full", http.StatusNoContent)
	if e := h.recovery(worker, a); e.State != "failed" || e.CauseCode != "disk_full" {
		t.Fatalf("failed: %+v", e)
	}
	var retried gotEngineOrder
	if err := json.Unmarshal(h.action(worker, a, "retry", map[string]string{"idempotency_key": "recovery-retry-0001"}, http.StatusCreated), &retried); err != nil || retried.Generation != 2 {
		t.Fatalf("retry: %+v %v", retried, err)
	}
	var replay gotEngineOrder
	if err := json.Unmarshal(h.action(worker, a, "retry", map[string]string{"idempotency_key": "recovery-retry-0001"}, http.StatusOK), &replay); err != nil || replay.OrderID != retried.OrderID {
		t.Fatalf("replayed retry: %+v %v", replay, err)
	}
	h.action(worker, a, "retry", map[string]string{"idempotency_key": "recovery-retry-0002"}, http.StatusConflict)
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE worker_id=$1 AND digest=$2`, worker, a); n != 2 {
		t.Fatalf("retry recorded %d orders", n)
	}
	// The failed generation's late report never reaches the retry.
	h.report(first.OrderID, a, "ready", "", http.StatusNoContent)
	if e := h.recovery(worker, a); e.State != "requested" || e.Generation != 2 {
		t.Fatalf("late report of the failed generation: %+v", e)
	}

	// Replace: a failed replacement leaves the working release untouched, and a Failed release never blocks the
	// choice of another compatible release.
	h.ack(retried.OrderID)
	h.report(retried.OrderID, a, "ready", "", http.StatusNoContent)
	replacement := h.install(worker, b, "recovery-replace-001")
	h.ack(replacement.OrderID)
	h.report(replacement.OrderID, b, "installing", "", http.StatusNoContent)
	h.report(replacement.OrderID, b, "failed", "runtime_incompatible", http.StatusNoContent)
	if e := h.recovery(worker, a); e.State != "ready" || e.CauseCode != "" {
		t.Fatalf("a failed replacement changed the working release: %+v", e)
	}
	if e := h.recovery(worker, b); e.State != "failed" || e.CauseCode != "runtime_incompatible" {
		t.Fatalf("replacement: %+v", e)
	}
	replaced := h.install(worker, c, "recovery-replace-002")
	h.ack(replaced.OrderID)
	h.report(replaced.OrderID, c, "failed", "hf_network", http.StatusNoContent)
	if e := h.recovery(worker, b); e.State != "failed" {
		t.Fatalf("former failure: %+v", e)
	}

	// Cancel before delivery: immediate, durable, idempotent; a late report is ignored.
	cancelled := h.install(second, a, "recovery-cancel-0001")
	if h.status(h.action(second, a, "cancel", nil, http.StatusOK)) != "cancelled" || h.status(h.action(second, a, "cancel", nil, http.StatusOK)) != "cancelled" {
		t.Fatal("cancel of a pending order")
	}
	if e := h.recovery(second, a); e.State != "failed" || e.CauseCode != "cancelled" {
		t.Fatalf("cancelled: %+v", e)
	}
	h.report(cancelled.OrderID, a, "ready", "", http.StatusNotFound) // another Worker's token cannot report it
	h.asSecond(func() { h.report(cancelled.OrderID, a, "ready", "", http.StatusNoContent) })
	if e := h.recovery(second, a); e.State != "failed" || h.reasons(second, a, "order_cancelled") != 1 {
		t.Fatalf("late report after cancellation: %+v", e)
	}

	// Cooperative cancellation of a running install.
	running := h.install(second, b, "recovery-cancel-0002")
	h.asSecond(func() {
		h.ack(running.OrderID)
		h.report(running.OrderID, b, "downloading_runtime", "", http.StatusNoContent)
		// A Worker cannot claim a cancellation nobody asked for.
		h.report(running.OrderID, b, "failed", "cancelled", http.StatusNoContent)
	})
	if e := h.recovery(second, b); e.State != "downloading_runtime" || h.reasons(second, b, "cancel_not_requested") != 1 {
		t.Fatalf("unrequested cancellation: %+v", e)
	}
	if h.status(h.action(second, b, "cancel", nil, http.StatusOK)) != "cancelling" || h.status(h.action(second, b, "cancel", nil, http.StatusOK)) != "cancelling" {
		t.Fatal("cancel of a running order")
	}
	if e := h.recovery(second, b); !e.CancelRequested || e.State != "downloading_runtime" || h.reasons(second, b, "cancel_requested") != 1 {
		t.Fatalf("cancel requested: %+v", e)
	}
	h.asSecond(func() {
		for i := 0; i < 2; i++ { // a reconnection delivers the same request again
			orders, _ := h.pull()
			if len(orders) != 1 || orders[0].OrderID != running.OrderID || !orders[0].CancelRequested {
				t.Fatalf("the request is not delivered: %+v", orders)
			}
		}
	})
	// A new order for the same engine cannot start while the cancellation is not confirmed.
	if r := h.orderOn(c, http.StatusOK, map[string]string{"worker_id": second, "idempotency_key": "recovery-cancel-0003"}); r[0].Cause != "engine_order_in_progress" {
		t.Fatalf("order during cancellation: %+v", r)
	}
	h.asSecond(func() {
		h.report(running.OrderID, b, "failed", "cancelled", http.StatusNoContent)
		h.report(running.OrderID, b, "failed", "cancelled", http.StatusNoContent) // repeated confirmation
		h.report(running.OrderID, b, "installing", "", http.StatusNoContent)      // late progress
	})
	if e := h.recovery(second, b); e.State != "failed" || e.CauseCode != "cancelled" || e.CancelRequested {
		t.Fatalf("confirmed cancellation: %+v", e)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE id=$1 AND status='cancelled' AND finished_at IS NOT NULL`, running.OrderID); n != 1 {
		t.Fatal("the order is not cancelled")
	}
	if h.status(h.action(second, b, "cancel", nil, http.StatusOK)) != "cancelled" {
		t.Fatal("repeated cancel after confirmation")
	}
	// A finished install cannot be cancelled; one that finished before the cancellation was seen stays ready.
	h.action(worker, a, "cancel", nil, http.StatusConflict)
	raced := h.install(second, c, "recovery-race-00001")
	h.asSecond(func() { h.ack(raced.OrderID) })
	h.action(second, c, "cancel", nil, http.StatusOK)
	h.asSecond(func() { h.report(raced.OrderID, c, "ready", "", http.StatusNoContent) })
	if e := h.recovery(second, c); e.State != "ready" || e.CancelRequested {
		t.Fatalf("install finished before the cancellation was seen: %+v", e)
	}
	h.action(second, strings.Repeat("f", 64), "cancel", nil, http.StatusNotFound)

	// Cleanup: explicit, only for a failed or cancelled install, delivered with the manifest, confirmed once.
	h.action(worker, a, "cleanup", nil, http.StatusConflict) // ready
	h.action(second, c, "cleanup", nil, http.StatusConflict) // ready
	if h.status(h.action(worker, c, "cleanup", nil, http.StatusOK)) != "requested" || h.status(h.action(worker, c, "cleanup", nil, http.StatusOK)) != "requested" {
		t.Fatal("cleanup request")
	}
	if e := h.recovery(worker, c); e.Cleanup != "requested" || e.State != "failed" || h.reasons(worker, c, "cleanup_requested") != 1 {
		t.Fatalf("cleanup requested: %+v", e)
	}
	_, cleanups := h.pull()
	if len(cleanups) != 1 || cleanups[0].OrderID != replaced.OrderID || cleanups[0].Digest != c || len(cleanups[0].Runtime) == 0 {
		t.Fatalf("cleanups delivered: %+v", cleanups)
	}
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/cleanup", map[string]any{"order_id": replaced.OrderID, "digest": b, "removed": true}, "", http.StatusBadRequest)
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/cleanup", map[string]any{"order_id": replaced.OrderID, "digest": c, "removed": true}, "", http.StatusNoContent)
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/cleanup", map[string]any{"order_id": replaced.OrderID, "digest": c, "removed": true}, "", http.StatusNoContent)
	if _, cleanups = h.pull(); len(cleanups) != 0 {
		t.Fatalf("cleanup delivered again: %+v", cleanups)
	}
	if e := h.recovery(worker, c); e.Cleanup != "done" || h.reasons(worker, c, "cleanup_done") != 1 {
		t.Fatalf("cleanup done: %+v", e)
	}
	if h.status(h.action(worker, c, "cleanup", nil, http.StatusOK)) != "done" {
		t.Fatal("cleanup repeated after completion")
	}
	// A confirmation nobody asked for is refused, and the working release is never part of a cleanup.
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/cleanup", map[string]any{"order_id": retried.OrderID, "digest": a, "removed": true}, "", http.StatusBadRequest)
	if e := h.recovery(worker, a); e.State != "ready" || e.Cleanup != "" {
		t.Fatalf("the ready release was touched: %+v", e)
	}
	// A retry after a cleanup request that has not run: the old request is never delivered over the new order.
	h.action(worker, b, "cleanup", nil, http.StatusOK)
	h.action(worker, b, "retry", map[string]string{"idempotency_key": "recovery-retry-0003"}, http.StatusCreated)
	if _, cleanups = h.pull(); len(cleanups) != 0 {
		t.Fatalf("a stale cleanup is delivered over a new order: %+v", cleanups)
	}
	h.action(worker, b, "cleanup", nil, http.StatusConflict) // open order
}
