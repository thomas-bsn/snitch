//go:build integration

package httpapi

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A failure of the new model (download, test, disk) keeps the old one in service and active, shows the cause and
// offers Retry; the replacement can be cancelled; nothing is removed.
func testVoiceModelReplacementFailure(t *testing.T, pool *pgxpool.Pool) {
	h, workerID, _ := startActiveCommunity(t, pool)
	before := h.pipelineVersion()

	o := h.replace(workerID, secondID, communityID, "replace-b-0001", http.StatusCreated)
	h.ack(o.OrderID, true, "", http.StatusNoContent)
	h.reportOrder(o.OrderID, secondID, "downloading", "", 10, 30, 10, http.StatusNoContent)
	// Cancelling while a download runs is refused: it cannot be interrupted.
	h.cancelReplacement(workerID, "diarization", http.StatusConflict)
	h.reportOrder(o.OrderID, secondID, "failed", "disk_full", 0, 0, 10, http.StatusNoContent)
	if c := h.cells(workerID)["diarization"]; c.State != "failed" || c.CauseCode != "disk_full" || c.ModelID != secondID || c.ReplacesModelID != communityID {
		t.Fatalf("a disk too small for both models is reported on the replacement: %+v", c)
	}
	if h.pipelineVersion() != before || h.modelRowState(workerID, communityID) != "ready" {
		t.Fatal("the old model stays active and installed after a failure")
	}
	if orders := h.workerOrders(); len(orders) != 0 {
		t.Fatalf("nothing is removed or retried silently: %+v", orders)
	}

	// Retry orders the installation again, still as the replacement of the old model.
	again := h.retry(workerID, secondID, "retry-key-0001", http.StatusCreated)
	if again.OrderID == o.OrderID || again.Action != "install" {
		t.Fatalf("Retry is a new install order: %+v", again)
	}
	if c := h.cells(workerID)["diarization"]; c.State != "requested" || c.ReplacesModelID != communityID {
		t.Fatalf("Retry keeps the replacement: %+v", c)
	}
	h.ack(again.OrderID, true, "", http.StatusNoContent)
	h.reportOrder(again.OrderID, secondID, "failed", "inference_failed", 30, 30, 30, http.StatusNoContent)
	if c := h.cells(workerID)["diarization"]; c.State != "failed" || c.CauseCode != "inference_failed" {
		t.Fatalf("a failed test is reported: %+v", c)
	}

	// Cancelling the replacement drops the failed attempt and leaves the old model alone.
	h.cancelReplacement(workerID, "diarization", http.StatusNoContent)
	h.cancelReplacement(workerID, "diarization", http.StatusNotFound)
	if c := h.cells(workerID)["diarization"]; c.State != "ready" || c.ModelID != communityID || c.ReplacesModelID != "" {
		t.Fatalf("the old model is back on its own: %+v", c)
	}
	if h.modelRowState(workerID, secondID) != "absent" || h.pipelineVersion() != before {
		t.Fatal("cancelling leaves the old model active and the failed attempt forgotten")
	}
	// A requested replacement that the Worker has not received yet is cancelled with its order.
	pending := h.replace(workerID, secondID, communityID, "replace-b-0002", http.StatusCreated)
	h.cancelReplacement(workerID, "diarization", http.StatusNoContent)
	if orders := h.workerOrders(); len(orders) != 0 {
		t.Fatalf("a cancelled order %s must not be delivered: %+v", pending.OrderID, orders)
	}
	// Without the replacement flag, the occupied type is still refused; only a replacement lifts it.
	h.order(workerID, secondID, "install-b-0003", http.StatusConflict)
	// Only another model of the same type, and the one installed and ready, can be replaced.
	h.replace(workerID, secondID, "someone/else", "replace-b-0004", http.StatusBadRequest)
	h.replace(workerID, secondID, secondID, "replace-b-0005", http.StatusBadRequest)
	h.replace(workerID, secondID, fingerprintID, "replace-b-0006", http.StatusBadRequest)

	// The fingerprint model is never switched here: its replacement waits for the versioned transition.
	h.reportModel(fingerprintID, "ready", "", 1, 1, http.StatusNoContent)
	if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,license,size_bytes) VALUES('test-fingerprint-b','fingerprint','pyannote','synthetic/fingerprint-b','d00000000000000000000000000000000000000d','test',1000)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(t.Context(), `DELETE FROM voice_model_catalog WHERE id='test-fingerprint-b'`) })
	body := h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-model-orders", map[string]any{"model_id": "test-fingerprint-b", "idempotency_key": "replace-fp-0001", "replaces_model_id": fingerprintID}, http.StatusConflict)
	if !bytes.Contains(body, []byte("voice_replacement_kind_unsupported")) {
		t.Fatalf("a fingerprint replacement is refused with a clear code: %s", body)
	}
	// Different types coexist on a Worker.
	cells := h.cells(workerID)
	if cells["diarization"].ModelID != communityID || cells["fingerprint"].ModelID != fingerprintID || cells["fingerprint"].State != "ready" {
		t.Fatalf("diarization and fingerprint models coexist: %+v", cells)
	}
}

// A removal that fails on the Worker keeps the old model and offers Retry, which orders it again.
func testVoiceModelReplacementRemovalFailure(t *testing.T, pool *pgxpool.Pool) {
	h, workerID, job := startActiveCommunity(t, pool)
	o := h.replace(workerID, secondID, communityID, "replace-b-0001", http.StatusCreated)
	h.ack(o.OrderID, true, "", http.StatusNoContent)
	h.reportOrder(o.OrderID, secondID, "ready", "", 30, 30, 30, http.StatusNoContent)
	if _, err := pool.Exec(t.Context(), `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	orders := h.workerOrders()
	if len(orders) != 1 || orders[0].Action != "remove" {
		t.Fatalf("remove order expected: %+v", orders)
	}
	h.ack(orders[0].OrderID, true, "", http.StatusNoContent)
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": orders[0].OrderID, "model_id": communityID, "revision": communityRevision, "state": "failed", "cause_code": "models_volume_unwritable"}, "", http.StatusNoContent)
	if c := h.cells(workerID)["diarization"]; c.State != "failed" || c.CauseCode != "models_volume_unwritable" || c.ReplacesModelID != communityID {
		t.Fatalf("a failed removal is reported: %+v", c)
	}
	if h.modelRowState(workerID, communityID) != "ready" {
		t.Fatal("the old model is still on the Worker")
	}
	if orders = h.workerOrders(); len(orders) != 0 {
		t.Fatalf("no automatic second attempt: %+v", orders)
	}
	h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-models/retry", map[string]any{"model_id": secondID, "idempotency_key": "retry-rm-0001"}, http.StatusOK)
	if orders = h.workerOrders(); len(orders) != 1 || orders[0].Action != "remove" || orders[0].ModelID != communityID {
		t.Fatalf("Retry orders the removal again: %+v", orders)
	}
}
