//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// runtimeVersion orders an update or a rollback to a version of the family of the model (no digest known).
func (h *engineOrderHarness) runtimeVersion(workerID, modelID, version, intent, key string, want int) gotRuntimeOrder {
	h.t.Helper()
	body := h.adminCall(http.MethodPost, "/api/admin/voice-model-runtime-orders", map[string]any{"worker_id": workerID, "model_id": modelID, "version": version, "intent": intent, "idempotency_key": key}, want)
	var o gotRuntimeOrder
	if want < 300 {
		if err := json.Unmarshal(body, &o); err != nil {
			h.t.Fatal(err)
		}
	}
	return o
}

// installFamilyModel installs a model of the nemo family on the Worker and runs it to Ready.
func (h *engineOrderHarness) installFamilyModel(worker, model, key string) {
	h.t.Helper()
	var order modelOrder
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/workers/"+worker+"/voice-model-orders", map[string]any{"model_id": model, "idempotency_key": key}, http.StatusCreated), &order); err != nil {
		h.t.Fatal(err)
	}
	for _, e := range h.delivered() {
		h.ack(e.OrderID)
		h.report(e.OrderID, e.Digest, "ready", "", http.StatusNoContent)
	}
	h.modelAck(order.OrderID, true, "", http.StatusNoContent)
	h.modelReport(order.OrderID, model, "ready", "", http.StatusNoContent)
}

// Several versions of an on-demand engine side by side, update and rollback of a model's runtime (#376).
func testVoiceModelRuntimeOnDemand(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	const rev = "0123456789abcdef0123456789abcdef01234567"
	// Model A is a fingerprint model (TitaNet-like), model B a diarization one (Nemotron-like): one family, two tasks.
	for _, m := range [][]string{{"family-model-a", "fingerprint", "nemo-titanet"}, {"family-model-b", "diarization", "nemo"}} {
		if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,family) VALUES($1,$3,$4,'synthetic/'||$1,$2,'nemo')`, m[0], rev, m[1], m[2]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES('signed-model','fingerprint','nemo-titanet','synthetic/signed',$1)`, rev); err != nil {
		t.Fatal(err)
	}
	worker := h.fixtureWorkerID()
	def := engineFamilies["nemo"].DefaultVer
	v1, v2 := onDemandDigest("nemo", def, "fingerprint"), onDemandDigest("nemo", "3.1.0", "fingerprint")

	// Model A installs on the default version; the model list of the Engines tab knows it with its family.
	h.installFamilyModel(worker, "family-model-a", "family-install-0001")
	b := h.binding(worker, "family-model-a")
	if b.Digest != v1 || b.Family != "nemo" || b.Version != def || !slices.Contains(b.Versions, def) {
		t.Fatalf("binding of a family model: %+v", b)
	}

	// Refusals: the version is validated, a signed release and an on-demand one never stand for each other.
	h.runtimeVersion(worker, "family-model-a", "--index-url=http://evil", "update", "family-badver-0001", http.StatusBadRequest)
	h.runtimeVersion(worker, "family-model-a", def, "update", "family-same-000001", http.StatusConflict) // already bound
	h.runtimeVersion(worker, "signed-model", "3.1.0", "update", "family-signed-00001", http.StatusBadRequest)
	signed := h.sign("1.0.0")
	h.runtimeOrder(worker, "family-model-a", signed, "update", "family-mismatch-001", http.StatusConflict)
	h.runtimeOrder(worker, "signed-model", v1, "update", "family-mismatch-002", http.StatusConflict)
	if h.count(`SELECT count(*) FROM worker_model_orders WHERE runtime_intent<>'install'`)+h.count(`SELECT count(*) FROM worker_engine_orders WHERE version='3.1.0'`) != 0 {
		t.Fatal("a refusal recorded something")
	}

	// A rollback never installs anything and needs a version the Worker holds.
	h.runtimeVersion(worker, "family-model-a", "3.1.0", "rollback", "family-norollback-01", http.StatusConflict)

	// Update: the new version is installed as an engine order first; the model order waits for it and the model
	// keeps its binding until the test inference succeeds.
	update := h.runtimeVersion(worker, "family-model-a", "3.1.0", "update", "family-update-00001", http.StatusCreated)
	if update.RuntimeIntent != "update" {
		t.Fatalf("update order: %+v", update)
	}
	if h.runtimeVersion(worker, "family-model-a", "3.1.0", "update", "family-update-00001", http.StatusOK).OrderID != update.OrderID {
		t.Fatal("replay created another order")
	}
	h.runtimeVersion(worker, "family-model-a", "3.2.0", "update", "family-update-00002", http.StatusConflict) // one open per model
	if orders, _ := h.pulled(); len(orders) != 0 {
		t.Fatalf("update delivered before its engine: %+v", orders)
	}
	engines := h.delivered()
	if len(engines) != 1 || engines[0].Digest != v2 || engines[0].Version != "3.1.0" {
		t.Fatalf("engine order of the update: %+v", engines)
	}
	if b = h.binding(worker, "family-model-a"); b.Digest != v1 || b.Pending == nil || b.Pending.Version != "3.1.0" {
		t.Fatalf("binding while the engine installs: %+v", b)
	}
	// Heavy installs stay serialised: another version of the family waits for the open order of the Worker.
	if r := h.orderFamily("nemo", "3.2.0", http.StatusOK, map[string]string{"worker_id": worker, "idempotency_key": "family-heavy-000001"}); r[0].Status != "refused" || r[0].Cause != "engine_order_in_progress" {
		t.Fatalf("second heavy install: %+v", r)
	}
	h.ack(engines[0].OrderID)
	h.report(engines[0].OrderID, v2, "ready", "", http.StatusNoContent)
	orders, held := h.pulled()
	if len(orders) != 1 || orders[0].OrderID != update.OrderID || !slices.Contains(held, "family-model-a") {
		t.Fatalf("pull once the engine is ready: %+v %v", orders, held)
	}
	h.modelAck(update.OrderID, true, "", http.StatusNoContent)
	h.modelReport(update.OrderID, "family-model-a", "verifying", "", http.StatusNoContent)
	if b = h.binding(worker, "family-model-a"); b.Digest != v1 || !h.runtimeReady(worker, "family-model-a") {
		t.Fatalf("model stopped serving during the test inference: %+v", b)
	}
	// The test inference fails: the previous binding is intact.
	h.modelReport(update.OrderID, "family-model-a", "failed", "inference_failed", http.StatusNoContent)
	if b = h.binding(worker, "family-model-a"); b.Digest != v1 || b.Pending != nil || b.History[0].State != "failed" || !h.runtimeReady(worker, "family-model-a") {
		t.Fatalf("binding after a failed update: %+v", b)
	}
	// A new update reuses the installed engine: no second engine order, the weights are not fetched again.
	before := h.count(`SELECT count(*) FROM worker_engine_orders`)
	retry := h.runtimeVersion(worker, "family-model-a", "3.1.0", "update", "family-update-00003", http.StatusCreated)
	if h.count(`SELECT count(*) FROM worker_engine_orders`) != before {
		t.Fatal("an engine already installed was ordered again")
	}
	if orders, _ = h.pulled(); len(orders) != 1 || orders[0].OrderID != retry.OrderID {
		t.Fatalf("retry delivery: %+v", orders)
	}
	h.modelAck(retry.OrderID, true, "", http.StatusNoContent)
	h.modelReport(retry.OrderID, "family-model-a", "ready", "", http.StatusNoContent)
	if b = h.binding(worker, "family-model-a"); b.Digest != v2 || b.PreviousDigest != v1 || b.Pending != nil || b.Version != "3.1.0" {
		t.Fatalf("binding after the update: %+v", b)
	}

	// Side by side: model B (diarization) installs on the default version and moves to 3.1.0 while A goes back to
	// the default one. Both versions of the family stay installed and each model stays bound to its own.
	h.installFamilyModel(worker, "family-model-b", "family-install-0002")
	if b = h.binding(worker, "family-model-b"); b.Digest != onDemandDigest("nemo", def, "diarization") {
		t.Fatalf("model B: %+v", b)
	}
	updateB := h.runtimeVersion(worker, "family-model-b", "3.1.0", "update", "family-update-b-001", http.StatusCreated)
	for _, e := range h.delivered() {
		h.ack(e.OrderID)
		h.report(e.OrderID, e.Digest, "ready", "", http.StatusNoContent)
	}
	h.modelAck(updateB.OrderID, true, "", http.StatusNoContent)
	h.modelReport(updateB.OrderID, "family-model-b", "ready", "", http.StatusNoContent)
	engineOrders := h.count(`SELECT count(*) FROM worker_engine_orders`)
	back := h.runtimeVersion(worker, "family-model-a", def, "rollback", "family-rollback-0001", http.StatusCreated)
	if back.RuntimeIntent != "rollback" || h.count(`SELECT count(*) FROM worker_engine_orders`) != engineOrders {
		t.Fatalf("rollback order %+v installed something", back)
	}
	if orders, _ = h.pulled(); len(orders) != 1 || orders[0].OrderID != back.OrderID {
		t.Fatalf("rollback delivery: %+v", orders)
	}
	h.modelAck(back.OrderID, true, "", http.StatusNoContent)
	h.modelReport(back.OrderID, "family-model-a", "ready", "", http.StatusNoContent)
	a, bb := h.binding(worker, "family-model-a"), h.binding(worker, "family-model-b")
	if a.Digest != v1 || a.PreviousDigest != v2 || bb.Version != "3.1.0" || bb.Digest != onDemandDigest("nemo", "3.1.0", "diarization") {
		t.Fatalf("side by side: A=%+v B=%+v", a, bb)
	}
	for _, d := range []string{v1, v2, bb.Digest} {
		if e, ok := h.entry(worker, d); !ok || e.State != "ready" {
			t.Fatalf("version %s: %+v %v", d, e, ok)
		}
	}
	if !h.runtimeReady(worker, "family-model-a") || !h.runtimeReady(worker, "family-model-b") {
		t.Fatal("each model must be eligible on its own version")
	}

	// Uninstall is refused while a model is bound, and the models are listed.
	if removable, blockers := h.preview(worker, v2); removable || len(blockers.Models) != 1 || blockers.Models[0].Role != "previous" {
		t.Fatalf("preview of a bound version: %v %+v", removable, blockers)
	}
	if r := h.uninstall(v1, worker, "family-uninstall-001"); r.Status != "refused" || r.Cause != "engine_in_use" || r.Blockers == nil || len(r.Blockers.Models) != 1 || r.Blockers.Models[0].ModelID != "family-model-a" {
		t.Fatalf("uninstall of a bound version: %+v", r)
	}
	if h.count(`SELECT count(*) FROM worker_engine_orders WHERE intent='uninstall'`) != 0 {
		t.Fatal("a refused uninstall was recorded")
	}

	// A version that disappeared cannot be a rollback target; the model keeps its binding.
	if _, err := pool.Exec(t.Context(), `UPDATE worker_engine_inventory SET state='removed' WHERE worker_id=$1 AND digest=$2`, worker, v2); err != nil {
		t.Fatal(err)
	}
	h.runtimeVersion(worker, "family-model-a", "3.1.0", "rollback", "family-lost-0000001", http.StatusConflict)
	if b = h.binding(worker, "family-model-a"); b.Digest != v1 {
		t.Fatalf("binding after a refused rollback: %+v", b)
	}
}
