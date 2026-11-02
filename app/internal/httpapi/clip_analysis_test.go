//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type analysisStepView struct {
	Step   string `json:"step"`
	Status string `json:"status"`
	Reason string `json:"reason"`
	JobID  *int64 `json:"job_id"`
}
type analysisRunView struct {
	ID     int64              `json:"id"`
	Scope  string             `json:"scope"`
	Status string             `json:"status"`
	Steps  []analysisStepView `json:"steps"`
}

func (r analysisRunView) step(name string) analysisStepView {
	for _, s := range r.Steps {
		if s.Step == name {
			return s
		}
	}
	return analysisStepView{}
}

func (h *voiceHarness) analysisPath() string {
	return fmt.Sprintf("/api/admin/clips/%d/analysis", h.clipID)
}

func (h *voiceHarness) startAnalysis(body map[string]any, want int) analysisRunView {
	h.t.Helper()
	var out struct {
		Run analysisRunView `json:"run"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, h.analysisPath(), body, want), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Run
}

func (h *voiceHarness) analysis() *analysisRunView {
	h.t.Helper()
	var out struct {
		Run *analysisRunView `json:"run"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, h.analysisPath(), nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Run
}

var analysisTranscript = map[string]any{"provider": "fixture", "model": "deterministic-v1", "version": "deterministic-v1", "language": "fr", "audio_stream_index": 2, "segments": []map[string]any{
	{"start": 0.2, "end": 0.9, "text": "premier", "words": []map[string]any{{"word": "premier", "start": 0.2, "end": 0.9}}},
}}

func testClipAnalysis(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	if h.analysis() != nil {
		t.Fatal("a clip never analyzed has no run")
	}
	h.adminCall(http.MethodPost, "/api/admin/clips/99999/analysis", map[string]any{}, http.StatusNotFound)
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{"scope": "everything"}, http.StatusBadRequest)

	// The analysis waits for a track choice, and its dependent step waits behind it.
	run := h.startAnalysis(map[string]any{}, http.StatusCreated)
	if run.Status != "active" || run.Scope != "transcription" || run.step("preparation").Status != "blocked" || run.step("preparation").Reason != "audio_track_selection_required" || run.step("transcription").Status != "waiting" {
		t.Fatalf("unexpected run waiting for a track: %+v", run)
	}
	// Idempotent start: concurrent clicks give one run; another scope is refused.
	var wg sync.WaitGroup
	ids := make([]int64, 6)
	body := mustJSON(t, map[string]any{})
	for i := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodPost, h.ts.URL+h.analysisPath(), bytes.NewReader(body))
			req.Header.Set("Origin", h.ts.URL)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", h.csrf)
			resp, err := h.admin.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			data, _ := io.ReadAll(resp.Body)
			var out struct {
				Run analysisRunView `json:"run"`
			}
			_ = json.Unmarshal(data, &out)
			ids[i] = out.Run.ID
		}()
	}
	wg.Wait()
	for _, id := range ids {
		if id != run.ID {
			t.Fatalf("concurrent starts must return the active run %d: %v", run.ID, ids)
		}
	}
	if n := h.count(`SELECT count(*) FROM clip_analysis_runs WHERE clip_id=$1`, h.clipID); n != 1 {
		t.Fatalf("%d runs for one clip", n)
	}
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{"scope": "preparation"}, http.StatusConflict)

	// Choosing the track plans the transcription; the run follows the job.
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusOK)
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{}, http.StatusOK)
	run = *h.analysis()
	if run.step("preparation").Status != "done" || run.step("transcription").Status != "waiting" || run.step("transcription").Reason != "awaiting_worker" {
		t.Fatalf("unexpected queued run: %+v", run)
	}
	job, ok := h.claim()
	if !ok || job.Step != "transcription.audio.2" {
		t.Fatalf("expected the transcription task, got %+v", job)
	}
	if err := h.api.Analysis.ReconcileClip(context.Background(), h.clipID); err != nil {
		t.Fatal(err)
	}
	if run = *h.analysis(); run.step("transcription").Status != "running" {
		t.Fatalf("unexpected running run: %+v", run)
	}
	// A failed attempt is a retry, not a failure (reported through the worker API).
	h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", job.ID), map[string]any{"error": "synthetic"}, job.Lease, http.StatusOK)
	run = *h.analysis()
	if run.step("transcription").Status != "retry" || run.step("transcription").Reason != "retry_scheduled" || run.Status != "active" {
		t.Fatalf("unexpected retry run: %+v", run)
	}
	// The run survives a restart: the job finishes while no server reconciles, a new server catches up.
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET status='done',result=$2::jsonb,updated_at=now() WHERE id=$1`, job.ID, string(mustJSON(t, analysisTranscript))); err != nil {
		t.Fatal(err)
	}
	cfg := h.api.Config
	cfg.UploadRoot = t.TempDir()
	restarted := New(pool, cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := restarted.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	deadline := time.Now().Add(10 * time.Second)
	for h.analysis().Status == "active" && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	run = *h.analysis()
	if run.Status != "completed" || run.step("transcription").Status != "done" || run.step("transcription").Reason != "completed" {
		t.Fatalf("a restart must finish the run: %+v", run)
	}

	// A second analysis reuses the valid transcription and plans no job.
	jobs := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1`, h.clipID)
	run2 := h.startAnalysis(map[string]any{}, http.StatusCreated)
	if run2.ID == run.ID || run2.Status != "completed" || run2.step("transcription").Reason != "reused_existing" || h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1`, h.clipID) != jobs {
		t.Fatalf("valid results must be reused: %+v", run2)
	}
	// A narrower scope shows the later step as not applicable.
	run3 := h.startAnalysis(map[string]any{"scope": "preparation"}, http.StatusCreated)
	if run3.Status != "completed" || run3.step("transcription").Status != "not_applicable" || run3.step("transcription").Reason != "outside_scope" {
		t.Fatalf("unexpected preparation run: %+v", run3)
	}

	// A job that dies fails the step and the run, and frees the clip for a new analysis.
	if _, err := pool.Exec(context.Background(), `DELETE FROM jobs WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	run4 := h.startAnalysis(map[string]any{}, http.StatusCreated)
	if run4.step("transcription").Reason != "awaiting_worker" {
		t.Fatalf("expected a queued job: %+v", run4)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET status='dead' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if err := h.api.Analysis.ReconcileClip(context.Background(), h.clipID); err != nil {
		t.Fatal(err)
	}
	if r := h.analysis(); r.Status != "failed" || r.step("transcription").Reason != "job_dead" {
		t.Fatalf("a dead job fails the run: %+v", r)
	}
	h.startAnalysis(map[string]any{}, http.StatusCreated)

	// Administrators only.
	member := h.newUser("analysis-member")
	h.userCall(member, http.MethodGet, h.analysisPath(), nil, http.StatusForbidden)
	h.userCall(member, http.MethodPost, h.analysisPath(), map[string]any{}, http.StatusForbidden)
}
