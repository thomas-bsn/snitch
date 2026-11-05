//go:build integration

package httpapi

import (
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The target model is installed next to the source on a Worker by the replacement of #308, which no longer
// decides the switch for a fingerprint model: the transition does, and the source goes only afterwards.
func testVoiceFingerprintTransitionWorkers(t *testing.T, pool *pgxpool.Pool) {
	lib := newTransitionLibrary(t, pool)
	h := lib.h
	workerID := h.fixtureWorkerID()
	h.reportModel(fingerprintID, "ready", "", 1, 1, http.StatusNoContent)

	// Without a confirmed transition that shows its impact, the replacement is refused with a clear code.
	h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-model-orders", map[string]any{"model_id": benchModelID, "idempotency_key": "replace-fp-0001", "replaces_model_id": fingerprintID}, http.StatusConflict)
	h.startTransition(benchModelID, 1, http.StatusCreated)
	o := h.replace(workerID, benchModelID, fingerprintID, "replace-fp-0002", http.StatusCreated)
	h.ack(o.OrderID, true, "", http.StatusNoContent)
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": o.OrderID, "model_id": benchModelID, "revision": "0123456789abcdef0123456789abcdef01234567", "state": "ready", "bytes_received": 30, "bytes_total": 30, "occupied_bytes": 30}, "", http.StatusNoContent)
	for _, order := range h.workerOrders() {
		t.Fatalf("the source model is not removed while the references are recomputed: %+v", order)
	}
	if st := h.transitionState(); st.Transition.Phase != "recomputing" || st.Transition.WorkersReady != 1 {
		t.Fatalf("the Worker holds both models and the references are recomputed: %+v", st.Transition)
	}

	h.completeNext(lib.reference, "REF", benchModelRepo, 0)
	if h.activeFingerprintModel() != benchModelID {
		t.Fatal("the transition switched the active model")
	}
	// The catch-up job of the other voice is still pending: the source is not in use by it (the new model is).
	orders := h.workerOrders()
	if len(orders) != 1 || orders[0].Action != "remove" || orders[0].ModelID != fingerprintID {
		t.Fatalf("after the switch the source model is removed on order: %+v", orders)
	}
}

// A production fingerprint computed by a catalogue model that is not the active one is never filed in the
// production space: it would otherwise sit next to the references of another model.
func testVoiceFingerprintStaleModel(t *testing.T, pool *pgxpool.Pool) {
	lib := newTransitionLibrary(t, pool)
	h := lib.h
	h.reportModel(benchModelID, "ready", "", 1, 1, http.StatusNoContent)
	var version string
	if err := pool.QueryRow(t.Context(), `SELECT j.input_version FROM voice_fingerprints f JOIN jobs j ON j.id=f.fingerprint_job_id WHERE f.local_speaker_id=$1`, lib.reference.ID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	h.scheduleRequested(lib.reference.ClipID, "voice.fingerprint.9", version, "", http.StatusCreated)
	job, ok := h.claim()
	if !ok {
		t.Fatal("the production job must be leased")
	}
	h.complete(job, benchmarkResult(h.diarizationOf(lib.reference), "REF", benchModelRepo, 0), http.StatusConflict)
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE model=$1`, benchModelRepo); n != 0 {
		t.Fatalf("a result of another model stores nothing, got %d", n)
	}
}
