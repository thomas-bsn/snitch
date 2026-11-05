//go:build integration

package httpapi

import (
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (h *voiceHarness) holdsReady(workerID, modelID, revision string) {
	h.t.Helper()
	if _, err := h.pool.Exec(h.t.Context(), `INSERT INTO worker_voice_models(worker_id,model_id,state,revision) VALUES($1,$2,'ready',$3) ON CONFLICT(worker_id,model_id) DO UPDATE SET state='ready'`, workerID, modelID, revision); err != nil {
		h.t.Fatal(err)
	}
}

// A job running on one Worker never blocks the removal of the same model on another Worker, but still blocks it
// on the Worker that holds the job (review of #308).
func testVoiceModelInUsePerWorker(t *testing.T, pool *pgxpool.Pool) {
	h, workerID, _ := startActiveCommunity(t, pool)
	other := h.otherWorker()
	h.holdsReady(other, communityID, communityRevision)

	h.remove(workerID, communityID, "remove-w-0001", http.StatusConflict) // its own job protects it
	o := h.remove(other, communityID, "remove-w-0002", http.StatusCreated)
	if o.WorkerID != other || o.Action != "remove" {
		t.Fatalf("the other Worker is not held by the job of the first: %+v", o)
	}
	if h.modelRowState(workerID, communityID) != "ready" {
		t.Fatal("the Worker running the job keeps its model")
	}
}

// The legacy uninstall route names one Worker and orders the removal there only (US27).
func testVoiceModelUninstallOneWorker(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t, pool)
	workerID, other := h.fixtureWorkerID(), h.otherWorker()
	h.holdsReady(workerID, secondID, secondRevision)
	h.holdsReady(other, secondID, secondRevision)

	h.adminCall(http.MethodDelete, "/api/admin/voice-models/"+secondID+"/installation", nil, http.StatusBadRequest)
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/"+secondID+"/installation?worker_id="+other, nil, http.StatusNoContent)
	if h.modelRowState(other, secondID) != "removing" || h.modelRowState(workerID, secondID) != "ready" {
		t.Fatalf("only the named Worker is ordered: %s / %s", h.modelRowState(other, secondID), h.modelRowState(workerID, secondID))
	}
	if n := h.count(`SELECT count(*) FROM worker_model_orders WHERE action='remove' AND model_id='` + secondID + `'`); n != 1 {
		t.Fatalf("exactly one remove order, got %d", n)
	}
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/"+secondID+"/installation?worker_id="+other, nil, http.StatusConflict) // not held any more
}
