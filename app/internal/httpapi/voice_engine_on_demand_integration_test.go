//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// orderFamily orders an on-demand engine by family and version (#370).
func (h *engineOrderHarness) orderFamily(family, version string, want int, orders ...map[string]string) []gotEngineOrderResult {
	h.t.Helper()
	body := map[string]any{"family": family, "orders": orders}
	if version != "" {
		body["version"] = version
	}
	var out struct {
		Results []gotEngineOrderResult `json:"results"`
	}
	raw := h.adminCall(http.MethodPost, "/api/admin/voice-engines/on-demand-orders", body, want)
	if want == http.StatusOK {
		if err := json.Unmarshal(raw, &out); err != nil {
			h.t.Fatal(err)
		}
	}
	return out.Results
}

func testVoiceEngineOnDemand(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	fixture, other := h.fixtureWorkerID(), h.secondID

	// The default version of the family is used when none is given; the order carries a runtime without signature.
	results := h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "ondemand-fixture-01"})
	if len(results) != 1 || results[0].Status != "created" || results[0].Order == nil {
		t.Fatalf("order: %+v", results)
	}
	first := results[0].Order
	if first.Engine != "nemo-toolkit" || first.Version == "" || first.Intent != "install" || first.Generation != 1 || len(first.Digest) != 64 {
		t.Fatalf("order content: %+v", first)
	}
	var runtime map[string]any
	if err := json.Unmarshal(first.Runtime, &runtime); err != nil || runtime["source"] != "on_demand" || runtime["family"] != "nemo" || runtime["distribution"] != "nemo-toolkit" || runtime["version"] != first.Version {
		t.Fatalf("runtime: %s (%v)", first.Runtime, err)
	}

	// The identity is a registry row of its own source, never an envelope, and never listed with signed releases.
	if n := h.count(`SELECT count(*) FROM voice_engine_registry WHERE digest=$1 AND source='on_demand' AND signature IS NULL AND signed_payload IS NULL AND key_id IS NULL`, first.Digest); n != 1 {
		t.Fatalf("on-demand registry rows: %d", n)
	}
	var listed struct {
		Engines []engineRelease `json:"engines"`
	}
	signed := h.sign("1.0.0")
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines", nil, http.StatusOK), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Engines) != 1 || listed.Engines[0].Digest != signed || signed == first.Digest {
		t.Fatalf("signed list: %+v", listed.Engines)
	}
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+first.Digest, nil, http.StatusNotFound)

	// Same key, same result; another Worker with the same key is refused; an open order refuses a second one.
	if r := h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "ondemand-fixture-01"}); r[0].Status != "replayed" || r[0].Order.OrderID != first.OrderID {
		t.Fatalf("replay: %+v", r)
	}
	if r := h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": other, "idempotency_key": "ondemand-fixture-01"}); r[0].Status != "refused" || r[0].Cause != "idempotency_key_reused" {
		t.Fatalf("key reuse: %+v", r)
	}
	if r := h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "ondemand-fixture-02"}); r[0].Status != "refused" || r[0].Cause != "engine_order_in_progress" {
		t.Fatalf("concurrent: %+v", r)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders`); n != 1 {
		t.Fatalf("orders recorded: %d", n)
	}

	// Delivery uses the usual pull and the order appears in the Engines view with its real state.
	if got := h.delivered(); len(got) != 1 || got[0].OrderID != first.OrderID || got[0].Engine != "nemo-toolkit" {
		t.Fatalf("delivery: %+v", got)
	}
	if e, ok := h.entry(fixture, first.Digest); !ok || e.State != "requested" {
		t.Fatalf("inventory after order: %+v %v", e, ok)
	}
	h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+first.OrderID+"/ack", map[string]any{"accepted": true}, "", http.StatusNoContent)
	h.report(first.OrderID, first.Digest, "installing", "", http.StatusNoContent)
	h.report(first.OrderID, first.Digest, "failed", "package_not_found", http.StatusNoContent)
	if e, _ := h.entry(fixture, first.Digest); e.State != "failed" || e.CauseCode != "package_not_found" {
		t.Fatalf("failed install: %+v", e)
	}

	// Each new cause is recorded and distinguished; a retry under a new key creates the next generation.
	for i, cause := range []string{"resolution_failed", "build_failed", "disk_full", "hf_network"} {
		key := "ondemand-retry-0" + string(rune('1'+i))
		r := h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": key})[0]
		if r.Status != "created" || r.Order.Generation != int64(i+2) {
			t.Fatalf("retry %d: %+v", i, r)
		}
		h.report(r.Order.OrderID, r.Order.Digest, "failed", cause, http.StatusNoContent)
		if e, _ := h.entry(fixture, first.Digest); e.State != "failed" || e.CauseCode != cause {
			t.Fatalf("cause %s: %+v", cause, e)
		}
	}
	last := h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "ondemand-success-01"})[0].Order
	h.report(last.OrderID, last.Digest, "installing", "", http.StatusNoContent)
	h.report(last.OrderID, last.Digest, "verifying", "", http.StatusNoContent)
	h.report(last.OrderID, last.Digest, "ready", "", http.StatusNoContent)
	if e, _ := h.entry(fixture, first.Digest); e.State != "ready" {
		t.Fatalf("ready install: %+v", e)
	}
	if r := h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "ondemand-again-0001"}); r[0].Cause != "engine_already_installed" {
		t.Fatalf("ready again: %+v", r)
	}

	// Another version coexists under its own identity; unknown families and malformed versions create nothing.
	if r := h.orderFamily("nemo", "9.9.9", http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "ondemand-version-01"}); r[0].Status != "created" || r[0].Order.Digest == first.Digest || r[0].Order.Version != "9.9.9" {
		t.Fatalf("other version: %+v", r)
	}
	orders := h.count(`SELECT count(*) FROM worker_engine_orders`)
	h.orderFamily("unknown-family", "", http.StatusBadRequest, map[string]string{"worker_id": other, "idempotency_key": "ondemand-unknown-01"})
	h.orderFamily("nemo", "--index-url=http://evil", http.StatusBadRequest, map[string]string{"worker_id": other, "idempotency_key": "ondemand-badver-001"})
	if r := h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": "wrk_missing", "idempotency_key": "ondemand-missing-01"}); r[0].Cause != "worker_not_found" {
		t.Fatalf("missing worker: %+v", r)
	}
	if h.count(`SELECT count(*) FROM worker_engine_orders`) != orders {
		t.Fatal("a refused request created an order")
	}
}

// The SpeechBrain family (#374) is ordered like NeMo: its runtime names the family, covers fingerprint and
// separation, and is a different identity from the signed ECAPA release and from the NeMo family.
func testVoiceEngineOnDemandSpeechBrain(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	fixture := h.fixtureWorkerID()
	signed := h.sign("1.0.3")

	r := h.orderFamily("speechbrain", "", http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "speechbrain-order-01"})[0]
	if r.Status != "created" || r.Order == nil {
		t.Fatalf("order: %+v", r)
	}
	if r.Order.Engine != "speechbrain" || r.Order.Version != "1.0.3" || r.Order.Intent != "install" || r.Order.Digest == signed {
		t.Fatalf("order content: %+v", r.Order)
	}
	var runtime map[string]any
	if err := json.Unmarshal(r.Order.Runtime, &runtime); err != nil || runtime["source"] != "on_demand" || runtime["family"] != "speechbrain" || runtime["distribution"] != "speechbrain" {
		t.Fatalf("runtime: %s (%v)", r.Order.Runtime, err)
	}
	if kinds, _ := runtime["kinds"].([]any); len(kinds) != 2 || kinds[0] != "fingerprint" || kinds[1] != "separation" {
		t.Fatalf("kinds: %v", runtime["kinds"])
	}
	if e, ok := h.entry(fixture, r.Order.Digest); !ok || e.State != "requested" {
		t.Fatalf("inventory: %+v %v", e, ok)
	}
	// The engine installs and becomes ready exactly like any other; nothing activates a separation model.
	h.report(r.Order.OrderID, r.Order.Digest, "installing", "", http.StatusNoContent)
	h.report(r.Order.OrderID, r.Order.Digest, "verifying", "", http.StatusNoContent)
	h.report(r.Order.OrderID, r.Order.Digest, "ready", "", http.StatusNoContent)
	if e, _ := h.entry(fixture, r.Order.Digest); e.State != "ready" {
		t.Fatalf("ready: %+v", e)
	}
	if n := h.count(`SELECT count(*) FROM worker_voice_models WHERE model_id IN (SELECT id FROM voice_model_catalog WHERE kind='separation')`); n != 0 {
		t.Fatalf("separation model rows: %d", n)
	}
	// The NeMo family and the SpeechBrain family never share an identity.
	nemo := h.orderFamily("nemo", "1.0.3", http.StatusOK, map[string]string{"worker_id": h.secondID, "idempotency_key": "speechbrain-nemo-0001"})[0]
	if nemo.Order == nil || nemo.Order.Digest == r.Order.Digest {
		t.Fatalf("families share an identity: %+v", nemo)
	}
}

func testVoiceEngineOnDemandResources(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	worker := h.fixtureWorkerID()
	order := func(key string) gotEngineOrderResult {
		return h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": worker, "idempotency_key": key})[0]
	}
	for name, c := range map[string]struct {
		hw    workerHardware
		cause string
	}{
		"platform": {workerHardware{Python: "3.12", Platform: "darwin-arm64", RAMBytes: 64 << 30, DiskFreeBytes: 200 << 30}, "platform_incompatible"},
		"python":   {workerHardware{Python: "3.10", Platform: "linux-x86_64", RAMBytes: 64 << 30, DiskFreeBytes: 200 << 30}, "platform_incompatible"},
		"ram":      {workerHardware{Python: "3.12", Platform: "linux-x86_64", RAMBytes: 1 << 30, DiskFreeBytes: 200 << 30}, "insufficient_ram"},
		"disk":     {workerHardware{Python: "3.12", Platform: "linux-x86_64", RAMBytes: 64 << 30, DiskFreeBytes: 1 << 20}, "insufficient_disk"},
	} {
		h.setHardware(worker, c.hw)
		if r := order("ondemand-res-" + name); r.Status != "refused" || r.Cause != c.cause {
			t.Fatalf("%s: %+v", name, r)
		}
		if n := h.count(`SELECT count(*) FROM worker_engine_orders`) + h.count(`SELECT count(*) FROM worker_engine_inventory`); n != 0 {
			t.Fatalf("%s: a refusal left %d rows", name, n)
		}
	}
	// A CPU Worker without CUDA can host the family.
	h.setHardware(worker, workerHardware{Python: "3.12", Platform: "linux-x86_64", RAMBytes: 64 << 30, DiskFreeBytes: 200 << 30})
	r := order("ondemand-res-cpu0001")
	if r.Status != "created" || !strings.HasPrefix(r.Order.Engine, "nemo") {
		t.Fatalf("cpu worker: %+v", r)
	}
	if e, ok := h.entry(worker, r.Order.Digest); !ok || e.State != "requested" {
		t.Fatalf("state: %+v %v", e, ok)
	}
}
