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

// benchmarkResult is the fingerprint result of a requested job: one speaker, one vector.
func benchmarkResult(diarizationJobID int64, label, model string, degrees float64) map[string]any {
	return map[string]any{
		"embedding_model": model, "embedding_model_version": "bench-v1", "audio_stream_index": 9, "diarization_job_id": diarizationJobID, "dimensions": 4,
		"fingerprints": []map[string]any{{"speaker_label": label, "scope": "speaker", "start": 0.0, "end": 2.0, "quality": 0.9, "embedding": angle(degrees)}},
	}
}

// runBenchmark schedules, leases and completes a benchmark fingerprint job for the speaker through the APIs.
func (h *voiceHarness) runBenchmark(sp fixtureSpeaker, label, model string, degrees float64, want int) {
	h.t.Helper()
	var diarization int64
	if err := h.pool.QueryRow(context.Background(), `SELECT diarization_job_id FROM local_speakers WHERE id=$1`, sp.ID).Scan(&diarization); err != nil {
		h.t.Fatal(err)
	}
	version := fmt.Sprintf("f5-a9-p%s-d%d", h.pipelineVersion(), diarization)
	h.scheduleRequested(sp.ClipID, "voice.fingerprint.9", version, benchModelID, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "voice.fingerprint.9" {
		h.t.Fatalf("the benchmark job must be leased, got %+v", job)
	}
	h.complete(job, benchmarkResult(diarization, label, model, degrees), want)
	if want != http.StatusOK {
		h.release(job)
		if _, err := h.pool.Exec(context.Background(), `UPDATE jobs SET status='cancelled' WHERE id=$1`, job.ID); err != nil {
			h.t.Fatal(err)
		}
	}
}

func testVoiceBenchmarkIsolation(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	h.addBenchmarkCatalogModel()
	h.putPipeline(fakePipeline(2))
	h.reportModel(benchModelID, "ready", "", 1, 1, http.StatusNoContent)
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=true,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	reference := h.fixtureSpeaker(h.clipIDs[0], "REF", usableAt(0))
	target := h.fixtureSpeaker(h.clipIDs[1], "TARGET", usableAt(90)) // unrelated to the reference in production
	h.validate(reference, "Alice")
	if got := h.matched(target); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("the production voices are unrelated: %+v", got)
	}

	// A result that does not carry the requested model is refused: it would be filed under another group.
	h.runBenchmark(reference, "REF", "fixture/other-model", 0, http.StatusBadRequest)

	// In the benchmark model both voices would be identical (angle 0): had they met the production
	// references, TARGET would be matched with Alice.
	h.runBenchmark(reference, "REF", benchModelRepo, 0, http.StatusOK)
	h.runBenchmark(target, "TARGET", benchModelRepo, 0, http.StatusOK)
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)

	if got := h.matched(target); got.DecisionLevel != "unknown" || got.Person != nil || (got.Attribution != nil && len(got.Attribution.Candidates) > 0) {
		t.Fatalf("benchmark fingerprints must never feed identification: %+v", got)
	}
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE benchmark AND model=$1`, benchModelRepo); n != 2 {
		t.Fatalf("the benchmark fingerprints stay in their own group, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE NOT benchmark AND model='fixture-embedding'`); n != 2 {
		t.Fatalf("a benchmark result must not replace the production fingerprints, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM speaker_attributions WHERE match_model=$1`, benchModelRepo); n != 0 {
		t.Fatalf("no attribution may come from a benchmark group, got %d", n)
	}
	var view struct {
		Fingerprints []struct {
			Model string `json:"model"`
		} `json:"fingerprints"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", reference.ClipID), nil, http.StatusOK), &view); err != nil {
		t.Fatal(err)
	}
	for _, fp := range view.Fingerprints {
		if fp.Model == benchModelRepo {
			t.Fatalf("the clip voice view lists production fingerprints only: %+v", view.Fingerprints)
		}
	}
}
