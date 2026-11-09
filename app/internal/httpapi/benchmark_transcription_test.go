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

// claimBenchmark leases the next benchmark transcription task for the simulated Worker.
func (h *voiceHarness) claimBenchmark() (claimedJob, bool) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"benchmark.transcription.*"}}, "", http.StatusOK)
	var a struct {
		Job        *claimedJob `json:"job"`
		LeaseToken string      `json:"lease_token"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodPost, "/api/worker/jobs/claim", nil, "", http.StatusOK), &a); err != nil {
		h.t.Fatal(err)
	}
	if a.Job == nil {
		return claimedJob{}, false
	}
	a.Job.Lease = a.LeaseToken
	return *a.Job, true
}

func transcriptResult(model string, texts ...string) map[string]any {
	segments := []map[string]any{}
	for i, text := range texts {
		segments = append(segments, map[string]any{"start": float64(i), "end": float64(i) + 0.9, "text": text, "words": []any{}})
	}
	return map[string]any{"provider": "fixture", "model": model, "version": "v-" + model, "language": "fr", "audio_stream_index": 1, "segments": segments}
}

func testAIBenchmarkTranscription(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 2)
	clip := h.clipIDs[0]
	// A production transcript of two segments, the first one corrected by a human.
	var transcription int64
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,result) VALUES($1,'transcription.audio.1','prod-transcription','done',$2::jsonb) RETURNING id`, clip,
		`{"provider":"fixture","model":"prod-model","version":"v","audio_stream_index":1,"segments":[{"start":0,"end":1,"text":"le chat dors","words":[]},{"start":1,"end":2,"text":"sur le canapé","words":[]}]}`).Scan(&transcription); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO segment_corrections(transcription_job_id,segment_index,clip_id,text) VALUES($1,0,$2,'Le chat dort')`, transcription, clip); err != nil {
		t.Fatal(err)
	}
	datasets, runs := "/api/admin/benchmarks/datasets", "/api/admin/benchmarks/runs"
	corpus := func(name string, verified []int64) map[string]any {
		return map[string]any{"name": name, "test_clip_ids": []int64{clip}, "reference_clip_ids": []int64{}, "verified_clip_ids": verified}
	}
	// Without certification the clip measures nothing.
	h.adminCall(http.MethodPost, datasets, corpus("Sans certification", []int64{}), http.StatusUnprocessableEntity)
	var created struct {
		ID     int64    `json:"id"`
		Stages []string `json:"stages"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, datasets, corpus("Relu", []int64{clip}), http.StatusCreated), &created); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(created.Stages) != "[transcription]" {
		t.Fatalf("a certified clip with a transcript measures the transcription, got %v", created.Stages)
	}
	// The reference text is frozen: the correction replaced the production text.
	var refs int
	var first string
	var corrected bool
	if err := pool.QueryRow(ctx, `SELECT count(*),min(text) FILTER (WHERE ordinal=0),bool_or(corrected) FILTER (WHERE ordinal=0) FROM benchmark_reference_segments WHERE dataset_id=$1`, created.ID).Scan(&refs, &first, &corrected); err != nil {
		t.Fatal(err)
	}
	if refs != 2 || first != "Le chat dort" || !corrected {
		t.Fatalf("the reference keeps the human correction: %d %q %v", refs, first, corrected)
	}
	if _, err := pool.Exec(ctx, `UPDATE segment_corrections SET text='Modifié après coup' WHERE transcription_job_id=$1`, transcription); err != nil {
		t.Fatal(err)
	}

	// The models offered are the production one; others are named by the administrator.
	var models struct {
		Models []struct {
			ID     string `json:"id"`
			Active bool   `json:"active"`
		} `json:"models"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/benchmarks/models?stage=transcription", nil, http.StatusOK), &models); err != nil {
		t.Fatal(err)
	}
	for _, m := range models.Models {
		if !m.Active {
			t.Fatalf("only the production model is offered: %+v", models)
		}
	}

	body := func(models ...string) map[string]any {
		return map[string]any{"dataset_id": created.ID, "stage": "transcription", "model_ids": models}
	}
	h.adminCall(http.MethodPost, runs, body("model-a", ""), http.StatusBadRequest)
	h.adminCall(http.MethodPost, runs, body("model-a", string(make([]byte, 100))), http.StatusUnprocessableEntity)
	var run struct{ ID int64 }
	if err := json.Unmarshal(h.adminCall(http.MethodPost, runs, body("model-a", "model-b"), http.StatusCreated), &run); err != nil {
		t.Fatal(err)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE step='benchmark.transcription.1' AND status='pending'`); n != 2 {
		t.Fatalf("one task per model on the certified track, got %d", n)
	}
	h.adminCall(http.MethodGet, "/api/admin/jobs", nil, http.StatusOK) // the queue lists the new step

	texts := map[string][]string{"model-a": {"Le chat dort", "sur le canapé"}, "model-b": {"un chat dort", "sur le canape"}}
	for i := 0; i < 2; i++ {
		job, ok := h.claimBenchmark()
		if !ok || job.Step != "benchmark.transcription.1" {
			t.Fatalf("a benchmark transcription must be leased, got %+v", job)
		}
		var input struct {
			Model    string          `json:"transcription_model"`
			Pipeline json.RawMessage `json:"pipeline"`
		}
		if err := json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, http.StatusOK), &input); err != nil {
			t.Fatal(err)
		}
		if texts[input.Model] == nil || len(input.Pipeline) == 0 {
			t.Fatalf("the task carries the requested model and the production profile: %+v", input)
		}
		// The result of another stage shape is refused: a benchmark transcript is checked like a production one.
		h.complete(job, map[string]any{"provider": "fixture"}, http.StatusBadRequest)
		h.complete(job, transcriptResult(input.Model, texts[input.Model]...), http.StatusOK)
	}
	var detail struct {
		Status  string `json:"status"`
		Results []struct {
			ModelID     string          `json:"model_id"`
			State       string          `json:"state"`
			ItemsTotal  int             `json:"items_total"`
			ItemsScored int             `json:"items_scored"`
			Metrics     json.RawMessage `json:"metrics"`
		} `json:"results"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("%s/%d", runs, run.ID), nil, http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Status != "completed" || len(detail.Results) != 2 {
		t.Fatalf("the run settles once both tasks ended: %+v", detail)
	}
	rates := map[string]struct{ wer, cer float64 }{}
	for _, res := range detail.Results {
		var m struct {
			WER *float64 `json:"wer"`
			CER *float64 `json:"cer"`
		}
		raw := []byte(res.Metrics)
		_ = json.Unmarshal(raw, &m)
		if m.WER == nil || m.CER == nil || res.State != "complete" || res.ItemsScored != 1 || res.ItemsTotal != 1 {
			t.Fatalf("a measured model has rates and a coverage: %+v %s", res, raw)
		}
		rates[res.ModelID] = struct{ wer, cer float64 }{*m.WER, *m.CER}
	}
	if rates["model-a"].wer != 0 || rates["model-b"].wer <= 0 || rates["model-b"].cer <= 0 {
		t.Fatalf("a perfect transcript has no error, a wrong one has some: %v", rates)
	}

	// Nothing reached production: no new transcript, no embedding, no new correction, the correction is the one.
	if n := h.count(`SELECT count(*) FROM jobs WHERE step LIKE 'transcription.audio.%'`); n != 1 {
		t.Fatalf("a benchmark transcript is never a production transcript, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM embedding_tasks`); n != 0 {
		t.Fatalf("a benchmark transcript is never indexed, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM segment_corrections WHERE stale_at IS NOT NULL`); n != 0 {
		t.Fatalf("a benchmark transcript never makes a correction obsolete, got %d", n)
	}
	if first != "Le chat dort" {
		t.Fatal("the frozen reference is read once at creation")
	}
	var frozen string
	if err := pool.QueryRow(ctx, `SELECT text FROM benchmark_reference_segments WHERE dataset_id=$1 AND ordinal=0`, created.ID).Scan(&frozen); err != nil || frozen != "Le chat dort" {
		t.Fatalf("a later correction never changes a corpus: %q %v", frozen, err)
	}

	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": 999999, "stage": "transcription", "model_ids": []string{"model-a"}}, http.StatusNotFound)
}
