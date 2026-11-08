//go:build integration

package httpapi

import (
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testVoiceEngineRuntimeUninstall covers the meeting of #352 (runtime update, rollback) and #353 (removal): the
// version a model is bound to and the one it can go back to cannot be removed, with the reason; a release being
// removed (or removed) is never an update or rollback target.
func testVoiceEngineRuntimeUninstall(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	const model = "interaction-runtime-model"
	if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,'fingerprint','nemo-titanet','synthetic/interaction','0123456789abcdef0123456789abcdef01234567')`, model); err != nil {
		t.Fatal(err)
	}
	worker := h.fixtureWorkerID()
	v1, v2, v3 := h.sign("1.0.0"), h.sign("2.0.0"), h.sign("3.0.0")
	h.installReady(v1, worker, "interaction-rt-engine-1")
	h.installReady(v2, worker, "interaction-rt-engine-2")
	h.installReady(v3, worker, "interaction-rt-engine-3")

	h.adminCall(http.MethodPost, "/api/admin/workers/"+worker+"/voice-model-orders", map[string]any{"model_id": model, "idempotency_key": "interaction-rt-model-01", "runtime_digest": v1}, http.StatusCreated)
	var orderID string
	if err := pool.QueryRow(t.Context(), `SELECT id FROM worker_model_orders WHERE model_id=$1`, model).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	h.modelAck(orderID, true, "", http.StatusNoContent)
	h.modelReport(orderID, model, "ready", "", http.StatusNoContent)
	update := h.runtimeOrder(worker, model, v2, "update", "interaction-rt-update-01", http.StatusCreated)
	h.modelAck(update.OrderID, true, "", http.StatusNoContent)
	h.modelReport(update.OrderID, model, "ready", "", http.StatusNoContent)
	if b := h.binding(worker, model); b.Digest != v2 || b.PreviousDigest != v1 {
		t.Fatalf("binding: %+v", b)
	}

	// The bound version and the rollback version are both refused, each with its reason.
	roles := map[string]string{v2: "bound", v1: "previous"}
	for digest, role := range roles {
		r := h.uninstall(digest, worker, "interaction-rt-uninstall-"+role)
		if r.Status != "refused" || r.Cause != "engine_in_use" || r.Blockers == nil || len(r.Blockers.Models) != 1 || r.Blockers.Models[0].ModelID != model || r.Blockers.Models[0].Role != role {
			t.Fatalf("removal of the %s version: %+v", role, r)
		}
		if ok, b := h.preview(worker, digest); ok || len(b.Models) != 1 {
			t.Fatalf("preview of the %s version: %v %+v", role, ok, b)
		}
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE intent='uninstall'`); n != 0 {
		t.Fatalf("refusals recorded %d removals", n)
	}

	// An unrelated version is removable; while it is being removed it is no update or rollback target.
	if r := h.uninstall(v3, worker, "interaction-rt-uninstall-v3"); r.Status != "created" {
		t.Fatalf("removal of an unrelated version: %+v", r)
	}
	h.runtimeOrder(worker, model, v3, "update", "interaction-rt-update-v3", http.StatusConflict)
	h.runtimeOrder(worker, model, v3, "rollback", "interaction-rt-rollback-v3", http.StatusConflict)
	if n := h.count(`SELECT count(*) FROM worker_model_orders WHERE runtime_intent<>'install' AND idempotency_key LIKE 'interaction-rt-%v3'`); n != 0 {
		t.Fatalf("targets of a removal recorded %d orders", n)
	}
}
