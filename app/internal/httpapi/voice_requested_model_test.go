//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	benchModelID   = "bench-embedding"
	benchModelRepo = "fixture/bench-embedding"
)

// addBenchmarkCatalogModel adds a supported fingerprint model that is not the active one.
func (h *voiceHarness) addBenchmarkCatalogModel() {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,'fingerprint','pyannote',$2,'0123456789abcdef0123456789abcdef01234567') ON CONFLICT DO NOTHING`, benchModelID, benchModelRepo); err != nil {
		h.t.Fatal(err)
	}
}

// scheduleRequested creates a job through the administrator API, optionally asking for a model.
func (h *voiceHarness) scheduleRequested(clipID int64, step, version, model string, want int) []byte {
	h.t.Helper()
	body := map[string]any{"clip_id": clipID, "step": step, "input_version": version}
	if model != "" {
		body["requested_model_id"] = model
	}
	return h.adminCall(http.MethodPost, "/api/admin/jobs", body, want)
}

// release frees the Worker's capacity without going through a result.
func (h *voiceHarness) release(job claimedJob) {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE id=$1`, job.ID); err != nil {
		h.t.Fatal(err)
	}
}

func testVoiceRequestedModelDispatch(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.addBenchmarkCatalogModel()
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent)
	h.putPipeline(map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5})
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 3}, http.StatusOK)
	holdsActive := map[string]any{"fingerprint": voiceModel(activeEmbedding, "837717ddb9ff")}
	holdsBench := map[string]any{"fingerprint": voiceModel(benchModelRepo, "0123456789ab")}

	// A request names a catalogue fingerprint model and nothing else.
	h.scheduleRequested(h.clipID, "voice.fingerprint.1", "bench-x", "no-such-model", http.StatusBadRequest)
	h.scheduleRequested(h.clipID, "voice.diarization.1", "bench-x", communityID, http.StatusBadRequest)
	h.scheduleRequested(h.clipID, "transcription.audio.1", "bench-x", benchModelID, http.StatusBadRequest)
	var created struct {
		RequestedModelID string `json:"requested_model_id"`
	}
	if err := json.Unmarshal(h.scheduleRequested(h.clipID, "voice.fingerprint.1", "bench-1", benchModelID, http.StatusCreated), &created); err != nil || created.RequestedModelID != benchModelID {
		t.Fatalf("the created job must echo its requested model: %q %v", created.RequestedModelID, err)
	}
	// Asking again for the same job is idempotent.
	h.scheduleRequested(h.clipID, "voice.fingerprint.1", "bench-1", benchModelID, http.StatusOK)

	// A production job of the same identity is another work: it neither replaces nor is replaced by the benchmark.
	h.scheduleRequested(h.clipID, "voice.fingerprint.1", "bench-1", "", http.StatusCreated)
	h.scheduleRequested(h.clipID, "voice.fingerprint.1", "bench-1b", "", http.StatusCreated)
	if n := h.count(`SELECT count(*) FROM jobs WHERE step='voice.fingerprint.1' AND requested_model_id IS NOT NULL AND status='pending'`); n != 1 {
		t.Fatalf("scheduling production work must not supersede a requested job, got %d pending", n)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET status='cancelled' WHERE step='voice.fingerprint.1' AND requested_model_id IS NULL`); err != nil {
		t.Fatal(err)
	}

	// No Worker has the requested model ready: the job waits, whatever the Worker declares.
	if job, ok := h.claimDeclaring(holdsBench); ok {
		t.Fatalf("a requested job was claimed without a ready model: %+v", job)
	}
	// The real state decides, not the declaration: a downloading model is not ready.
	h.reportModel(benchModelID, "downloading", "", 1, 2, http.StatusNoContent)
	if job, ok := h.claimDeclaring(holdsBench); ok {
		t.Fatalf("a requested job was claimed while the model downloads: %+v", job)
	}

	// A model that is ready but not active holds no production work.
	h.reportModel(benchModelID, "ready", "", 1, 1, http.StatusNoContent)
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET status='cancelled' WHERE requested_model_id IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	h.scheduleRequested(h.clipID, "voice.fingerprint.3", "prod-1", "", http.StatusCreated)
	if job, ok := h.claimDeclaring(holdsBench); ok {
		t.Fatalf("a ready but inactive model took production work: %+v", job)
	}
	// Without a requested model the active model applies, as in production: the Worker must have reported it
	// ready (what it declares never opens a local job, ADR 0038).
	h.reportModel("pyannote-wespeaker-resnet34", "ready", "", 1, 1, http.StatusNoContent)
	prod, ok := h.claimDeclaring(holdsActive)
	if !ok || prod.Step != "voice.fingerprint.3" {
		t.Fatalf("the Worker holding the active model must take the production job, got %+v", prod)
	}
	h.assertFingerprintInput(prod, activeEmbedding, "837717ddb9ff5507820346191109dc79c958d614")
	h.release(prod)

	// A Worker without the requested model ready does not get the job; with it, it does.
	h.scheduleRequested(h.clipID, "voice.fingerprint.4", "bench-2", benchModelID, http.StatusCreated)
	if _, err := pool.Exec(context.Background(), `DELETE FROM worker_voice_models WHERE model_id=$1`, benchModelID); err != nil {
		t.Fatal(err)
	}
	if job, ok := h.claimDeclaring(holdsActive); ok {
		t.Fatalf("a Worker without the requested model took the job: %+v", job)
	}
	h.reportModel(benchModelID, "ready", "", 1, 1, http.StatusNoContent)
	bench, ok := h.claimDeclaring(nil)
	if !ok || bench.Step != "voice.fingerprint.4" {
		t.Fatalf("the Worker holding the requested model must take the job, got %+v", bench)
	}
	h.assertFingerprintInput(bench, benchModelRepo, "0123456789abcdef0123456789abcdef01234567")
	var requested string
	if err := pool.QueryRow(context.Background(), `SELECT requested_model_id FROM jobs WHERE id=$1`, bench.ID).Scan(&requested); err != nil || requested != benchModelID {
		t.Fatalf("the leased job keeps its requested model: %q %v", requested, err)
	}
}

// Verify the actual leased input, independent of the Worker's legacy environment.
func (h *voiceHarness) assertFingerprintInput(job claimedJob, repo, revision string) {
	h.t.Helper()
	version := fmt.Sprintf("f5-a3-p%s-d%d", h.pipelineVersion(), job.ID)
	if _, err := h.pool.Exec(context.Background(), `UPDATE jobs SET input_version=$2 WHERE id=$1`, job.ID, version); err != nil {
		h.t.Fatal(err)
	}
	var input struct {
		Voice struct {
			FingerprintModel struct {
				Repo     string `json:"repo"`
				Revision string `json:"revision"`
				Engine   string `json:"engine"`
			} `json:"fingerprint_model"`
		} `json:"voice"`
	}
	raw := h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, http.StatusOK)
	if err := json.Unmarshal(raw, &input); err != nil {
		h.t.Fatal(err)
	}
	ref := input.Voice.FingerprintModel
	if ref.Repo != repo || ref.Revision != revision || ref.Engine != "pyannote" {
		h.t.Fatalf("incorrect fingerprint reference: %+v", ref)
	}
}
