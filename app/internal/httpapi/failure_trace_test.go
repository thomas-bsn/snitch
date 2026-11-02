//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"snitch/internal/logging"
)

// A synthetic clip fails on its diarization (GPU memory), loses a lease, retries and succeeds. The
// whole server trace, at DEBUG, carries the correlation context and none of the private content.
func testSimulatedFailureTraceIsClean(t *testing.T, pool *pgxpool.Pool) {
	logs := &lockedBuffer{}
	h := newVoiceHarnessLogged(t, pool, 1, logging.New(logs, "snitch", "DEBUG"))
	h.transcribe()
	h.putPipeline(fakePipeline(2))
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)

	first := h.mustClaim("the diarization is not claimable")
	if first.Step != "voice.diarization.2" {
		t.Fatalf("expected the diarization, got %+v", first)
	}
	h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", first.ID), map[string]any{"error": "gpu_out_of_memory", "provider": "pyannote", "model": "fixture-diarization", "duration_ms": 900}, first.Lease, http.StatusOK)
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET next_attempt_at=now() WHERE id=$1`, first.ID); err != nil {
		t.Fatal(err)
	}
	second := h.mustClaim("the retry after the GPU failure is not claimable") // this lease is left to expire
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET lease_expires_at=now()-interval '1 minute' WHERE id=$1`, first.ID); err != nil {
		t.Fatal(err)
	}
	third := h.mustClaim("the job is not claimable after its lease expired")
	h.complete(third, diarizationPayload("SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	fingerprints := h.mustClaim("the fingerprint step is missing")
	items := []map[string]any{{"speaker_label": "SPEAKER_00", "scope": "speaker", "start": 0, "end": 1.5, "quality": 0.9, "embedding": []float64{0.31415927, 0.27182818, 0.57721566, 0.16180339}}}
	h.complete(fingerprints, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": 2, "diarization_job_id": first.ID, "dimensions": 4, "fingerprints": items}, http.StatusOK)

	out := logs.String()
	jobID := float64(first.ID)
	for _, want := range []struct {
		event string
		match map[string]any
	}{
		{"job_claimed", map[string]any{"job_id": jobID, "attempt": float64(1)}},
		{"job_failed", map[string]any{"job_id": jobID, "attempt": float64(1), "error_code": "gpu_out_of_memory"}},
		{"job_retry_scheduled", map[string]any{"job_id": jobID, "attempt": float64(1)}},
		{"job_lease_expired", map[string]any{"job_id": jobID, "attempt": float64(2), "error_code": "lease_expired"}},
		{"job_completed", map[string]any{"job_id": jobID, "attempt": float64(3)}},
	} {
		if !hasLog(logEvents(t, out, want.event), want.match) {
			t.Fatalf("event %s %v missing in:\n%s", want.event, want.match, out)
		}
	}
	for _, expired := range logEvents(t, out, "job_lease_expired") {
		if expired["worker_id"] == nil || expired["worker_id"] == "" || expired["clip_id"] == nil {
			t.Fatalf("the lease expiry must name the Worker that lost it: %v", expired)
		}
	}
	for _, private := range []string{first.Lease, second.Lease, third.Lease, fingerprints.Lease, h.token, "premier", "second\"", "ailleurs", "0.31415927", "0.27182818", "0.57721566", "0.16180339", "X-Worker-Lease-Token", "Authorization", "sig="} {
		if strings.Contains(out, private) {
			t.Fatalf("the trace leaked %q:\n%s", private, out)
		}
	}
}

func (h *voiceHarness) mustClaim(message string) claimedJob {
	h.t.Helper()
	job, ok := h.claim()
	if !ok {
		h.t.Fatal(message)
	}
	return job
}
