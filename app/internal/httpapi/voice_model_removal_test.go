//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (h *voiceHarness) remove(workerID, modelID, key string, want int) modelOrder {
	h.t.Helper()
	var o modelOrder
	body := h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-model-removals", map[string]any{"model_id": modelID, "idempotency_key": key}, want)
	if want < 300 {
		if err := json.Unmarshal(body, &o); err != nil {
			h.t.Fatal(err)
		}
	}
	return o
}

type historyItem struct {
	OrderID    string `json:"order_id"`
	Action     string `json:"action"`
	ModelID    string `json:"model_id"`
	Status     string `json:"status"`
	CauseCode  string `json:"cause_code"`
	AuthorName string `json:"author_name"`
}

func (h *voiceHarness) history(workerID string) []historyItem {
	h.t.Helper()
	var out struct {
		Orders []historyItem `json:"orders"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/workers/"+workerID+"/voice-model-orders", nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Orders
}

// The Administrator removes a model on a Worker explicitly: refused while a job uses it, never silent, and the
// history of the Worker keeps every order.
func testVoiceModelExplicitRemoval(t *testing.T, pool *pgxpool.Pool) {
	h, workerID, job := startActiveCommunity(t, pool)

	// A job uses the model: refused.
	h.remove(workerID, communityID, "remove-a-0001", http.StatusConflict)
	if h.modelRowState(workerID, communityID) != "ready" {
		t.Fatal("a refused removal changes nothing")
	}
	if _, err := pool.Exec(t.Context(), `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	// A model the Worker does not hold cannot be removed.
	h.remove(workerID, secondID, "remove-a-0002", http.StatusConflict)

	o := h.remove(workerID, communityID, "remove-a-0003", http.StatusCreated)
	if o.Action != "remove" || o.ModelID != communityID {
		t.Fatalf("a remove order: %+v", o)
	}
	if replay := h.remove(workerID, communityID, "remove-a-0003", http.StatusOK); replay.OrderID != o.OrderID {
		t.Fatal("the same key replays the recorded order")
	}
	h.order(workerID, communityID, "remove-a-0003", http.StatusConflict)  // same key, different action
	h.remove(workerID, communityID, "remove-a-0004", http.StatusConflict) // already being removed
	if c := h.cells(workerID)["diarization"]; c.State != "removing" || c.ModelID != communityID {
		t.Fatalf("removing expected: %+v", c)
	}
	orders := h.workerOrders()
	if len(orders) == 1 && orders[0].HFToken != "" {
		t.Fatalf("a removal never carries the token: %+v", orders)
	}
	if len(orders) != 1 || orders[0].Action != "remove" {
		t.Fatalf("the Worker receives the order: %+v", orders)
	}
	h.ack(orders[0].OrderID, true, "", http.StatusNoContent)
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": o.OrderID, "model_id": communityID, "revision": communityRevision, "state": "removed"}, "", http.StatusNoContent)
	if c := h.cells(workerID)["diarization"]; c.State != "not_installed" {
		t.Fatalf("not installed after the removal: %+v", c)
	}
	hist := h.history(workerID)
	if len(hist) != 1 || hist[0].Action != "remove" || hist[0].Status != "done" || hist[0].AuthorName == "" {
		t.Fatalf("history: %+v", hist)
	}
	h.adminCall(http.MethodGet, "/api/admin/workers/wrk_unknown/voice-model-orders", nil, http.StatusNotFound)
}

// When the disk cannot hold both models, the Administrator gives up the old one explicitly; the failed new model
// is then retried as a plain installation. Before the failure, the old model of a replacement cannot be removed.
func testVoiceModelRemovalForSpace(t *testing.T, pool *pgxpool.Pool) {
	h, workerID, job := startActiveCommunity(t, pool)
	if _, err := pool.Exec(t.Context(), `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	o := h.replace(workerID, secondID, communityID, "replace-c-0001", http.StatusCreated)
	h.remove(workerID, communityID, "remove-c-0001", http.StatusConflict) // the replacement is under way
	h.ack(o.OrderID, true, "", http.StatusNoContent)
	h.reportOrder(o.OrderID, secondID, "failed", "disk_full", 0, 0, 0, http.StatusNoContent)

	rm := h.remove(workerID, communityID, "remove-c-0002", http.StatusCreated)
	if c := h.cells(workerID)["diarization"]; c.State != "removing" || c.ModelID != communityID || c.ReplacesModelID != "" {
		t.Fatalf("the old model is being removed, the replacement is abandoned: %+v", c)
	}
	h.ack(rm.OrderID, true, "", http.StatusNoContent)
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": rm.OrderID, "model_id": communityID, "revision": communityRevision, "state": "removed"}, "", http.StatusNoContent)
	if c := h.cells(workerID)["diarization"]; c.State != "failed" || c.ModelID != secondID || c.CauseCode != "disk_full" {
		t.Fatalf("the failed new model stays, ready to be retried: %+v", c)
	}
	again := h.retry(workerID, secondID, "retry-c-0001", http.StatusCreated)
	if again.Action != "install" {
		t.Fatalf("Retry installs the new model alone: %+v", again)
	}
}
