//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testVoiceModelEngineDependency(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	const model = "synthetic-engine-dependent"
	if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,'fingerprint','nemo-titanet','synthetic/dependent','0123456789abcdef0123456789abcdef01234567')`, model); err != nil {
		t.Fatal(err)
	}
	worker := h.fixtureWorkerID()
	digest := h.sign("1.0.0")
	input := map[string]any{"model_id": model, "idempotency_key": "dependent-model-0001", "runtime_digest": digest}
	body := h.adminCall(http.MethodPost, "/api/admin/workers/"+worker+"/voice-model-orders", input, http.StatusCreated)
	var modelOrder gotRuntimeOrder
	if err := json.Unmarshal(body, &modelOrder); err != nil {
		t.Fatal(err)
	}
	engineOrders := h.delivered()
	if len(engineOrders) != 1 || engineOrders[0].Digest != digest {
		t.Fatalf("model installation must order its exact engine first: %+v", engineOrders)
	}
	assertModels := func(want int) {
		t.Helper()
		var answer struct {
			Orders []json.RawMessage `json:"orders"`
		}
		if err := json.Unmarshal(h.workerCall(http.MethodGet, "/api/worker/voice-models", nil, "", http.StatusOK), &answer); err != nil {
			t.Fatal(err)
		}
		if len(answer.Orders) != want {
			t.Fatalf("delivered %d model orders, want %d", len(answer.Orders), want)
		}
	}
	if reserved, err := reservedDisk(t.Context(), pool, worker); err != nil || reserved != testEngineManifest().DiskBytes {
		t.Fatalf("engine and model reserved the same runtime twice: %d %v", reserved, err)
	}
	assertModels(0)
	h.adminCall(http.MethodPost, "/api/admin/workers/"+worker+"/voice-model-orders", input, http.StatusOK)
	if len(h.delivered()) != 1 {
		t.Fatal("replay duplicated the engine order")
	}
	engine := engineOrders[0]
	h.ack(engine.OrderID)
	h.report(engine.OrderID, digest, "failed", "hf_network", http.StatusNoContent)
	assertModels(0)
	if h.count(`SELECT count(*) FROM worker_voice_models WHERE model_id=$1 AND state='requested' AND cause_code=''`, model) != 1 {
		t.Fatal("engine failure must leave the model waiting")
	}
	h.action(worker, digest, "retry", map[string]any{"idempotency_key": "dependent-engine-retry-01"}, http.StatusCreated)
	retried := h.delivered()
	if len(retried) != 1 {
		t.Fatalf("retry: %+v", retried)
	}
	h.ack(retried[0].OrderID)
	h.report(retried[0].OrderID, digest, "ready", "", http.StatusNoContent)
	assertModels(1)
	// Retrying failed weights reuses the now-ready engine; it never orders that runtime twice.
	h.modelAck(modelOrder.OrderID, true, "", http.StatusNoContent)
	h.modelReport(modelOrder.OrderID, model, "failed", "hf_network", http.StatusNoContent)
	input["idempotency_key"] = "dependent-model-retry-01"
	h.adminCall(http.MethodPost, "/api/admin/workers/"+worker+"/voice-model-orders", input, http.StatusCreated)
	assertModels(1)
	if h.count(`SELECT count(*) FROM worker_engine_orders WHERE worker_id=$1`, worker) != 2 {
		t.Fatal("ready engine was ordered again")
	}
	// An independently requested engine is shared by the model instead of causing a conflict or a duplicate.
	h.install(h.secondID, digest, "dependent-independent-01")
	input["idempotency_key"] = "dependent-model-second-01"
	h.adminCall(http.MethodPost, "/api/admin/workers/"+h.secondID+"/voice-model-orders", input, http.StatusCreated)
	h.asSecond(func() {
		if len(h.delivered()) != 1 {
			t.Fatal("pending engine was ordered twice")
		}
		assertModels(0)
	})

}
