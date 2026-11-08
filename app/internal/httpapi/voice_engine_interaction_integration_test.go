//go:build integration

package httpapi

import (
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testVoiceEngineCancelUninstall covers the meeting of #351 (cancellation, cleanup) and #353 (removal): a
// removal is refused while a cancellation is not yet confirmed, and a removal is never cancelled or cleaned up.
func testVoiceEngineCancelUninstall(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	worker := h.fixtureWorkerID()
	a, b := h.sign("1.0.0"), h.sign("2.0.0")
	h.installReady(a, worker, "interaction-install-a1")

	// b is being installed and its cancellation was requested but not confirmed (cancelling).
	running := h.install(worker, b, "interaction-install-b1")
	h.ack(running.OrderID)
	h.report(running.OrderID, b, "downloading_runtime", "", http.StatusNoContent)
	if h.status(h.action(worker, b, "cancel", nil, http.StatusOK)) != "cancelling" {
		t.Fatal("cancel of a running install")
	}
	// Removing the ready release of the same engine waits for the Worker's confirmation.
	if r := h.uninstall(a, worker, "interaction-uninstall-01"); r.Status != "refused" || r.Cause != "engine_order_in_progress" {
		t.Fatalf("removal during a cancellation: %+v", r)
	}
	h.report(running.OrderID, b, "failed", "cancelled", http.StatusNoContent)
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE intent='uninstall'`); n != 0 {
		t.Fatalf("a refusal recorded %d removals", n)
	}

	// Once confirmed, the removal is accepted, and it can neither be cancelled nor cleaned up.
	r := h.uninstall(a, worker, "interaction-uninstall-02")
	if r.Status != "created" {
		t.Fatalf("removal after the cancellation: %+v", r)
	}
	h.action(worker, a, "cancel", nil, http.StatusConflict)
	h.action(worker, a, "cleanup", nil, http.StatusConflict)
	if e := h.uninstallEntry(worker, a); e.State != "removing" || e.OrderStatus != "pending" {
		t.Fatalf("a cancel or cleanup touched the removal: %+v", e)
	}
	// The Worker cannot report a cancellation of a removal.
	h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+r.Order.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
	h.report(r.Order.OrderID, a, "failed", "cancelled", http.StatusBadRequest)
	h.report(r.Order.OrderID, a, "removed", "", http.StatusNoContent)
	if e := h.uninstallEntry(worker, a); e.State != "removed" {
		t.Fatalf("removal: %+v", e)
	}
	// A cancelled install's cause stays storable next to a refused removal's cause.
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE cause_code='cancelled'`); n != 1 {
		t.Fatalf("cancelled install: %d", n)
	}
}
