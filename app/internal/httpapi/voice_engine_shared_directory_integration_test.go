//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// installOnDemandKind installs the on-demand engine of a family for one kind on a Worker and reports it ready.
func (h *engineOrderHarness) installOnDemandKind(family, kind, workerID, key string) string {
	h.t.Helper()
	var out struct {
		Results []gotEngineOrderResult `json:"results"`
	}
	body := map[string]any{"family": family, "kind": kind, "orders": []map[string]string{{"worker_id": workerID, "idempotency_key": key}}}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/voice-engines/on-demand-orders", body, http.StatusOK), &out); err != nil || len(out.Results) != 1 || out.Results[0].Status != "created" {
		h.t.Fatalf("install %s/%s: %v %+v", family, kind, err, out.Results)
	}
	order := out.Results[0].Order
	token := h.token
	if workerID == h.secondID {
		token = h.secondToken
	}
	h.asWorker(token, func() {
		h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+order.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
		h.report(order.OrderID, order.Digest, "ready", "", http.StatusNoContent)
	})
	return order.Digest
}

// The Worker keeps one environment per family and version, whatever the kind: a model bound to the engine of
// one kind keeps the directory the engine of the other kind of the same version lives in. Snitch refuses the
// removal of either identity while a model of any kind depends on that directory.
func testVoiceEngineSharedDirectory(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	fixture := h.fixtureWorkerID()
	diarization := h.installOnDemandKind("nemo", "diarization", fixture, "shared-dir-install-1")
	fingerprint := h.installOnDemandKind("nemo", "fingerprint", fixture, "shared-dir-install-2")
	if diarization == fingerprint {
		t.Fatal("each kind is its own identity")
	}
	other := h.installOnDemandKind("nemo", "fingerprint", h.secondID, "shared-dir-install-3")
	if other != fingerprint {
		t.Fatal("the same identity on another Worker")
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,family) VALUES('shared-dir-titanet','fingerprint','nemo-titanet','synthetic/shared-dir-titanet','0123456789abcdef0123456789abcdef01234567','nemo')`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM voice_model_catalog WHERE id='shared-dir-titanet'`)
	})
	// A model of the fingerprint kind is bound to the fingerprint engine.
	h.bindModel(fixture, fingerprint, "shared-dir-titanet", "ord_bind_shared_dir_00001")

	// The diarization identity shares the directory: the preview and the order both name the fingerprint model.
	ok, blockers := h.preview(fixture, diarization)
	if ok || len(blockers.Models) != 1 || blockers.Models[0].ModelID != "shared-dir-titanet" {
		t.Fatalf("preview of the sibling kind: %v %+v", ok, blockers)
	}
	if r := h.uninstall(diarization, fixture, "shared-dir-uninstall-1"); r.Status != "refused" || r.Cause != "engine_in_use" || r.Blockers == nil || len(r.Blockers.Models) != 1 {
		t.Fatalf("uninstall of the sibling kind: %+v", r)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE intent='uninstall'`); n != 0 {
		t.Fatalf("a refusal recorded %d removals", n)
	}
	// Another Worker has no such model: its identity stays removable.
	if ok, _ := h.preview(h.secondID, fingerprint); !ok {
		t.Fatal("the other Worker holds no bound model")
	}

	// Once the model is released, the directory can go through either identity.
	h.unbindModel(fixture, "shared-dir-titanet")
	if ok, _ := h.preview(fixture, diarization); !ok {
		t.Fatal("removable once the model is released")
	}
	if r := h.uninstall(diarization, fixture, "shared-dir-uninstall-2"); r.Status != "created" {
		t.Fatalf("uninstall once released: %+v", r)
	}
}
