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

func (h *voiceHarness) analysisAction(path string, want int) analysisRunView {
	h.t.Helper()
	var out struct {
		Run analysisRunView `json:"run"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, h.analysisPath()+path, nil, want), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Run
}

// testClipAnalysisControl covers the waits with their exact cause, the automatic resume
// after the track or Worker is resolved, the cancellation and the retry of a failed step.
func testClipAnalysisControl(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := context.Background()
	h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusConflict)
	h.adminCall(http.MethodPost, h.analysisPath()+"/steps/transcription/retry", nil, http.StatusNotFound)

	run := h.startAnalysis(map[string]any{}, http.StatusCreated)
	if run.step("preparation").Reason != "audio_track_selection_required" {
		t.Fatalf("expected a track choice: %+v", run)
	}
	// Choosing the track resumes the run by itself; no Worker supports the task yet.
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusOK)
	if run = *h.analysis(); run.step("preparation").Status != "done" || run.step("transcription").Reason != "awaiting_compatible_worker" {
		t.Fatalf("the choice of the track must resume the run and name the missing Worker: %+v", run)
	}
	// A compatible Worker comes back: the cause changes by itself.
	h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"transcription.audio.*"}}, "", http.StatusOK)
	if err := h.api.Analysis.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if run = *h.analysis(); run.step("transcription").Reason != "awaiting_worker" {
		t.Fatalf("a connected compatible Worker is a different wait: %+v", run)
	}

	// Cancelling keeps the clip free, stops the steps behind and forgets the pending job.
	h.analysisAction("/steps/transcription/retry", http.StatusConflict)
	cancelled := h.analysisAction("/cancel", http.StatusOK)
	if cancelled.ID != run.ID || cancelled.Status != "cancelled" || cancelled.step("transcription").Status != "blocked" || cancelled.step("transcription").Reason != "run_cancelled" || cancelled.step("preparation").Status != "done" {
		t.Fatalf("unexpected cancelled run: %+v", cancelled)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND status='cancelled'`, h.clipID); n != 1 {
		t.Fatalf("the pending job must be cancelled: %d", n)
	}
	h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusConflict)
	h.adminCall(http.MethodPost, h.analysisPath()+"/steps/transcription/retry", nil, http.StatusConflict)

	// A new run starts afresh with a new job.
	run2 := h.startAnalysis(map[string]any{}, http.StatusCreated)
	if run2.ID == run.ID || run2.step("transcription").Reason != "awaiting_worker" || *run2.step("transcription").JobID == *run.step("transcription").JobID {
		t.Fatalf("a new run must plan a new job: %+v", run2)
	}

	// A running step that ends validly after the cancellation keeps its result.
	job, ok := h.claim()
	if !ok {
		t.Fatal("expected the transcription task")
	}
	if err := h.api.Analysis.ReconcileClip(ctx, h.clipID); err != nil {
		t.Fatal(err)
	}
	cancelled = h.analysisAction("/cancel", http.StatusOK)
	if cancelled.step("transcription").Status != "running" {
		t.Fatalf("a running step is left to finish: %+v", cancelled)
	}
	h.complete(job, analysisTranscript, http.StatusOK)
	if r := *h.analysis(); r.Status != "cancelled" || r.step("transcription").Status != "done" || r.step("transcription").Reason != "completed" {
		t.Fatalf("the valid result must be kept: %+v", r)
	}

	// A step that failed for good is retried alone; the retry follows the job policy.
	if _, err := pool.Exec(ctx, `DELETE FROM jobs WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	run3 := h.startAnalysis(map[string]any{}, http.StatusCreated)
	h.analysisAction("/steps/transcription/retry", http.StatusConflict)
	h.analysisAction("/steps/everything/retry", http.StatusBadRequest)
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='dead' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if err := h.api.Analysis.ReconcileClip(ctx, h.clipID); err != nil {
		t.Fatal(err)
	}
	if r := *h.analysis(); r.Status != "failed" || r.step("transcription").Reason != "job_dead" {
		t.Fatalf("expected a failed run: %+v", r)
	}
	retried := h.analysisAction("/steps/transcription/retry", http.StatusOK)
	if retried.ID != run3.ID || retried.Status != "active" || retried.step("transcription").Status != "waiting" || retried.step("preparation").Status != "done" {
		t.Fatalf("unexpected retried run: %+v", retried)
	}
	job, ok = h.claim()
	if !ok {
		t.Fatal("expected a fresh transcription task")
	}
	h.complete(job, analysisTranscript, http.StatusOK)
	if r := *h.analysis(); r.Status != "completed" || r.step("transcription").Status != "done" {
		t.Fatalf("the retried run must complete: %+v", r)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.%'`, h.clipID); n != 2 {
		t.Fatalf("the retry must add one job: %d", n)
	}

	// Administrators only.
	member := h.newUser("analysis-control-member")
	h.userCall(member, http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusForbidden)
	h.userCall(member, http.MethodPost, h.analysisPath()+"/steps/transcription/retry", nil, http.StatusForbidden)
}
