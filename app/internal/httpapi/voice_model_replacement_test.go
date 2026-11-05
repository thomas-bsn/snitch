//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// replace orders the installation of newID on a Worker as the replacement of oldID (same type).
func (h *voiceHarness) replace(workerID, newID, oldID, key string, want int) modelOrder {
	h.t.Helper()
	var o modelOrder
	body := h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-model-orders", map[string]any{"model_id": newID, "idempotency_key": key, "replaces_model_id": oldID}, want)
	if want < 300 {
		if err := json.Unmarshal(body, &o); err != nil {
			h.t.Fatal(err)
		}
	}
	return o
}

func (h *voiceHarness) cancelReplacement(workerID, kind string, want int) {
	h.t.Helper()
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+workerID+"/voice-model-replacements/"+kind, nil, want)
}

func (h *voiceHarness) modelRowState(workerID, modelID string) string {
	h.t.Helper()
	var state string
	if err := h.pool.QueryRow(h.t.Context(), `SELECT state FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, modelID).Scan(&state); err != nil {
		return "absent"
	}
	return state
}

func (h *voiceHarness) activeModelRepo() string {
	h.t.Helper()
	var repo string
	if err := h.pool.QueryRow(h.t.Context(), `SELECT v.config->>'model' FROM voice_pipeline_active a JOIN voice_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&repo); err != nil {
		h.t.Fatal(err)
	}
	return repo
}

// startActiveCommunity makes the community model active and ready on the fixture Worker, with one diarization
// job leased on it, and the second catalogue model available.
func startActiveCommunity(t *testing.T, pool *pgxpool.Pool) (*voiceHarness, string, claimedJob) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t, pool)
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent)
	h.putPipeline(map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5})
	h.transcribe()
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	job, ok := h.claimDeclaring(nil)
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("the active model must run the job, got %+v", job)
	}
	return h, h.fixtureWorkerID(), job
}

// The new model is downloaded and verified while the old one stays active; Snitch decides the switch once it is
// ready; the old one is removed only by an order, and only once no job uses it.
func testVoiceModelReplacement(t *testing.T, pool *pgxpool.Pool) {
	h, workerID, job := startActiveCommunity(t, pool)
	before := h.pipelineVersion()

	o := h.replace(workerID, secondID, communityID, "replace-b-0001", http.StatusCreated)
	if o.Action != "install" || o.ModelID != secondID {
		t.Fatalf("a replacement starts with an install order: %+v", o)
	}
	if c := h.cells(workerID)["diarization"]; c.ModelID != secondID || c.State != "requested" || c.ReplacesModelID != communityID {
		t.Fatalf("the cell shows the replacement: %+v", c)
	}
	h.ack(o.OrderID, true, "", http.StatusNoContent)
	h.reportOrder(o.OrderID, secondID, "downloading", "", 10, 30, 10, http.StatusNoContent)
	h.reportOrder(o.OrderID, secondID, "verifying", "", 30, 30, 30, http.StatusNoContent)
	// Until the new one is ready, nothing changes: the old one stays installed, active and in service.
	if h.pipelineVersion() != before || h.modelRowState(workerID, communityID) != "ready" {
		t.Fatal("the old model must stay active while the new one is prepared")
	}
	if orders := h.workerOrders(); len(orders) != 1 || orders[0].Action != "install" {
		t.Fatalf("no removal before the switch: %+v", orders)
	}

	// Ready: Snitch switches the active model; a job still uses the old one, so the removal waits.
	h.reportOrder(o.OrderID, secondID, "ready", "", 30, 30, 30, http.StatusNoContent)
	if h.pipelineVersion() == before || h.activeModelRepo() != secondRepo {
		t.Fatalf("Snitch must switch the active model once the new one is ready (repo %s)", h.activeModelRepo())
	}
	if c := h.cells(workerID)["diarization"]; c.State != "switching" || c.ModelID != secondID || !c.WaitingForJobs || c.ReplacesModelID != communityID {
		t.Fatalf("a removal deferred by a running job is switching: %+v", c)
	}
	if h.modelRowState(workerID, communityID) != "ready" {
		t.Fatal("the old model is not removed while a job uses it")
	}
	if orders := h.workerOrders(); len(orders) != 0 {
		t.Fatalf("no remove order while a job uses the old model: %+v", orders)
	}

	// The job ends; the next pull of the Worker carries the remove order.
	if _, err := pool.Exec(t.Context(), `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	orders := h.workerOrders()
	if len(orders) != 1 || orders[0].Action != "remove" || orders[0].ModelID != communityID || orders[0].Repo != activeDiarization || orders[0].Revision != communityRevision {
		t.Fatalf("a remove order for the old model: %+v", orders)
	}
	if c := h.cells(workerID)["diarization"]; c.State != "removing" || c.ModelID != secondID {
		t.Fatalf("removing expected: %+v", c)
	}
	if h.modelRowState(workerID, communityID) != "removing" {
		t.Fatal("the old model is being removed")
	}
	// An install order cannot report a removal.
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": o.OrderID, "model_id": secondID, "revision": secondRevision, "state": "removed"}, "", http.StatusBadRequest)
	h.ack(orders[0].OrderID, true, "", http.StatusNoContent)
	removed := map[string]any{"order_id": orders[0].OrderID, "model_id": communityID, "revision": communityRevision, "state": "removed"}
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", removed, "", http.StatusNoContent)
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", removed, "", http.StatusNoContent) // replay
	if h.modelRowState(workerID, communityID) != "absent" {
		t.Fatal("the old model is gone once the Worker reports its removal")
	}
	if c := h.cells(workerID)["diarization"]; c.State != "ready" || c.ModelID != secondID || c.ReplacesModelID != "" || c.WaitingForJobs {
		t.Fatalf("one model per type after the replacement: %+v", c)
	}
	var kept int
	_ = pool.QueryRow(t.Context(), `SELECT count(*) FROM worker_voice_models m JOIN voice_model_catalog c ON c.id=m.model_id WHERE m.worker_id=$1 AND c.kind='diarization'`, workerID).Scan(&kept)
	if kept != 1 {
		t.Fatalf("a single diarization model per Worker, got %d", kept)
	}
	if orders = h.workerOrders(); len(orders) != 0 {
		t.Fatalf("nothing left to run: %+v", orders)
	}
}
