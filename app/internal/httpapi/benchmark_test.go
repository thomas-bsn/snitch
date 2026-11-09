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

type benchRunBody struct {
	ID       int64  `json:"id"`
	Status   string `json:"status"`
	Progress struct {
		Jobs, Done, Failed, Active int
	} `json:"progress"`
	Results []struct {
		ModelID      string `json:"model_id"`
		Model        string `json:"model"`
		ModelVersion string `json:"model_version"`
		State        string `json:"state"`
		ItemsTotal   int    `json:"items_total"`
		ItemsScored  int    `json:"items_scored"`
		Metrics      struct {
			EER          *float64 `json:"eer"`
			Top1Accuracy *float64 `json:"top1_accuracy"`
			Precision    *float64 `json:"precision"`
			Recall       *float64 `json:"recall"`
		} `json:"metrics"`
	} `json:"results"`
}

func (h *voiceHarness) benchRun(id int64) benchRunBody {
	h.t.Helper()
	var run benchRunBody
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/admin/benchmarks/runs/%d", id), nil, http.StatusOK), &run); err != nil {
		h.t.Fatal(err)
	}
	return run
}

func (h *voiceHarness) benchID(body []byte) int64 {
	h.t.Helper()
	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.ID < 1 {
		h.t.Fatalf("an id is expected, got %s", body)
	}
	return out.ID
}

func testAIBenchmarkFingerprints(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 5)
	h.addBenchmarkCatalogModel()
	h.putPipeline(fakePipeline(5))
	h.reportModel(benchModelID, "ready", "", 1, 1, http.StatusNoContent)
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 4}, http.StatusOK)

	refA := h.fixtureSpeaker(h.clipIDs[0], "REFA", usableAt(0))
	refB := h.fixtureSpeaker(h.clipIDs[1], "REFB", usableAt(90))
	testA := h.fixtureSpeaker(h.clipIDs[2], "TESTA", usableAt(5))
	testB := h.fixtureSpeaker(h.clipIDs[3], "TESTB", usableAt(85))
	pipeline := h.pipelineVersion()
	for _, sp := range []fixtureSpeaker{refA, refB, testA, testB} {
		// The benchmark rebuilds the fingerprint version of a real diarization, which carries its pipeline.
		if _, err := pool.Exec(ctx, `UPDATE jobs SET input_version=$2 WHERE id=$1`, h.diarizationOf(sp), fmt.Sprintf("diar-p%s", pipeline)); err != nil {
			t.Fatal(err)
		}
	}
	h.validate(refA, "Alice")
	h.validate(refB, "Bob")
	alice, bob := h.personOf(refA), h.personOf(refB)
	h.validate(testA, alice)
	h.validate(testB, bob)
	activeModel := func() string {
		var id string
		if err := pool.QueryRow(ctx, `SELECT model_id FROM voice_active_model WHERE kind='fingerprint'`).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	activeBefore := activeModel()

	// Only administrators reach the benchmarks.
	member := h.newUser("benchmark-member")
	for _, call := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/benchmarks/datasets"}, {http.MethodPost, "/api/admin/benchmarks/runs"},
		{http.MethodGet, "/api/admin/benchmarks/runs/1"}, {http.MethodPost, "/api/admin/benchmarks/runs/1/cancel"},
	} {
		if status, _ := h.userStatus(member, call.method, call.path, map[string]any{}); status != http.StatusForbidden {
			t.Fatalf("%s %s must be refused to a member, got %d", call.method, call.path, status)
		}
	}

	// Dataset rules: validated ground truth, no voice on both sides, no test without reference, two persons at least.
	unvalidated := h.fixtureSpeaker(h.clipIDs[4], "NOBODY", usableAt(40))
	datasets := "/api/admin/benchmarks/datasets"
	bad := func(body map[string]any, want int) {
		t.Helper()
		h.adminCall(http.MethodPost, datasets, body, want)
	}
	bad(map[string]any{"name": "x", "reference_speaker_ids": []int64{refA.ID, refB.ID}, "test_speaker_ids": []int64{unvalidated.ID}}, http.StatusUnprocessableEntity)
	bad(map[string]any{"name": "x", "reference_speaker_ids": []int64{refA.ID, refB.ID}, "test_speaker_ids": []int64{refA.ID}}, http.StatusUnprocessableEntity)
	bad(map[string]any{"name": "x", "reference_speaker_ids": []int64{refA.ID}, "test_speaker_ids": []int64{testA.ID, testB.ID}}, http.StatusUnprocessableEntity)
	bad(map[string]any{"name": "x", "reference_speaker_ids": []int64{refA.ID}, "test_speaker_ids": []int64{testA.ID}}, http.StatusUnprocessableEntity)
	bad(map[string]any{"name": " ", "reference_speaker_ids": []int64{refA.ID, refB.ID}, "test_speaker_ids": []int64{testA.ID}}, http.StatusBadRequest)
	dataset := h.benchID(h.adminCall(http.MethodPost, datasets, map[string]any{"name": "Voix validées", "reference_speaker_ids": []int64{refA.ID, refB.ID}, "test_speaker_ids": []int64{testA.ID, testB.ID}}, http.StatusCreated))
	bad(map[string]any{"name": "voix VALIDÉES", "reference_speaker_ids": []int64{refA.ID, refB.ID}, "test_speaker_ids": []int64{testA.ID}}, http.StatusConflict)

	runs := "/api/admin/benchmarks/runs"
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{"unknown-model"}}, http.StatusUnprocessableEntity)
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "diarization", "model_ids": []string{benchModelID}}, http.StatusBadRequest)

	// A run cancelled before any Worker took a job leaves no job behind, and frees the next run.
	first := h.benchID(h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusCreated))
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusConflict)
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", datasets, dataset), nil, http.StatusConflict)
	if got := h.benchRun(first); got.Status != "running" || got.Progress.Jobs != 4 || got.Progress.Active != 4 || len(got.Results) != 0 {
		t.Fatalf("a fresh run waits for its four jobs: %+v", got)
	}
	cancelled := func() benchRunBody {
		var run benchRunBody
		if err := json.Unmarshal(h.adminCall(http.MethodPost, fmt.Sprintf("%s/%d/cancel", runs, first), nil, http.StatusOK), &run); err != nil {
			t.Fatal(err)
		}
		return run
	}()
	if cancelled.Status != "cancelled" || cancelled.Progress.Active != 0 || len(cancelled.Results) != 0 {
		t.Fatalf("a cancelled run keeps no active job and no result: %+v", cancelled)
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("%s/%d/cancel", runs, first), nil, http.StatusConflict)
	if n := h.count(`SELECT count(*) FROM jobs WHERE requested_model_id IS NOT NULL AND status IN ('pending','leased','retry_wait')`); n != 0 {
		t.Fatalf("cancelling must stop the benchmark jobs, %d remain", n)
	}

	// The real run: the Worker completes the four benchmark jobs with the requested model.
	second := h.benchID(h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusCreated))
	degrees := map[int64]float64{h.diarizationOf(refA): 0, h.diarizationOf(refB): 90, h.diarizationOf(testA): 8, h.diarizationOf(testB): 80}
	labels := map[int64]string{h.diarizationOf(refA): "REFA", h.diarizationOf(refB): "REFB", h.diarizationOf(testA): "TESTA", h.diarizationOf(testB): "TESTB"}
	for i := 0; i < 4; i++ {
		job, ok := h.claim()
		if !ok {
			t.Fatalf("job %d of 4 must be leased", i)
		}
		var diarization int64
		for d := range degrees {
			if job.InputVersion == fmt.Sprintf("f5-a9-p%s-d%d", pipeline, d) {
				diarization = d
			}
		}
		if _, known := degrees[diarization]; !known {
			t.Fatalf("unexpected job %+v", job)
		}
		h.complete(job, benchmarkResult(diarization, labels[diarization], benchModelRepo, degrees[diarization]), http.StatusOK)
		h.release(job)
	}
	run := h.benchRun(second)
	if run.Status != "completed" || len(run.Results) != 1 {
		t.Fatalf("the run settles once its jobs have ended: %+v", run)
	}
	result := run.Results[0]
	if result.ModelID != benchModelID || result.Model != benchModelRepo || result.ModelVersion != "bench-v1" || result.State != "complete" || result.ItemsTotal != 4 || result.ItemsScored != 4 {
		t.Fatalf("the result names the model, its version and the coverage: %+v", result)
	}
	if result.Metrics.EER == nil || *result.Metrics.EER != 0 || result.Metrics.Top1Accuracy == nil || *result.Metrics.Top1Accuracy != 1 || result.Metrics.Precision == nil || *result.Metrics.Precision != 1 || result.Metrics.Recall == nil || *result.Metrics.Recall != 1 {
		t.Fatalf("two well separated persons must be perfectly identified: %+v", result.Metrics)
	}
	// Reading again changes nothing: the settlement is idempotent.
	if again := h.benchRun(second); len(again.Results) != 1 || again.Status != "completed" {
		t.Fatalf("settling twice must not duplicate results: %+v", again)
	}

	// The benchmark never touches production: fingerprints, validations, active model.
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE NOT benchmark AND model='fixture-embedding'`); n != 5 {
		t.Fatalf("the production fingerprints are untouched, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE benchmark AND model=$1`, benchModelRepo); n != 4 {
		t.Fatalf("the benchmark fingerprints stay in their own group, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM speaker_attributions WHERE validation_state='validated'`); n != 4 {
		t.Fatalf("the human validations are untouched, got %d", n)
	}
	if after := activeModel(); after != activeBefore {
		t.Fatalf("a benchmark must not change the active model: %s then %s", activeBefore, after)
	}

	// A run whose jobs all fail has no result to show and ends failed, with the error of each job.
	third := h.benchID(h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusCreated))
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='dead',last_error='worker_error' WHERE id IN (SELECT job_id FROM benchmark_run_jobs WHERE run_id=$1)`, third); err != nil {
		t.Fatal(err)
	}
	if got := h.benchRun(third); got.Status != "failed" || len(got.Results) != 1 || got.Results[0].State != "failed" || got.Results[0].ItemsScored != 0 {
		t.Fatalf("a run without any fingerprint fails: %+v", got)
	}

	// A run where one voice has no fingerprint is partial: the result says how much it covers.
	fourth := h.benchID(h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusCreated))
	for i := 0; i < 3; i++ {
		job, ok := h.claim()
		if !ok {
			t.Fatalf("job %d of 3 must be leased", i)
		}
		for d, deg := range degrees {
			if job.InputVersion == fmt.Sprintf("f5-a9-p%s-d%d", pipeline, d) {
				h.complete(job, benchmarkResult(d, labels[d], benchModelRepo, deg), http.StatusOK)
			}
		}
		h.release(job)
	}
	if got := h.benchRun(fourth); got.Status != "running" || got.Progress.Done != 3 || got.Progress.Active != 1 || len(got.Results) != 0 {
		t.Fatalf("a run waits for its last job before settling: %+v", got)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='dead',last_error='worker_error' WHERE id IN (SELECT job_id FROM benchmark_run_jobs WHERE run_id=$1) AND status IN ('pending','leased','retry_wait')`, fourth); err != nil {
		t.Fatal(err)
	}
	partial := h.benchRun(fourth)
	if partial.Status != "completed" || len(partial.Results) != 1 || partial.Results[0].State != "partial" || partial.Results[0].ItemsScored != 3 || partial.Results[0].ItemsTotal != 4 {
		t.Fatalf("a missing voice makes the result partial: %+v", partial)
	}

	// Plumbing: unknown or malformed ids, the dataset list and the deletion of an unused dataset.
	h.adminCall(http.MethodGet, runs+"/999999", nil, http.StatusNotFound)
	h.adminCall(http.MethodGet, runs+"/abc", nil, http.StatusBadRequest)
	h.adminCall(http.MethodPost, runs+"/999999/cancel", nil, http.StatusNotFound)
	h.adminCall(http.MethodDelete, datasets+"/999999", nil, http.StatusNotFound)
	h.adminCall(http.MethodDelete, datasets+"/0", nil, http.StatusBadRequest)
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": 999999, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusNotFound)
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{benchModelID, benchModelID}}, http.StatusBadRequest)
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{}}, http.StatusBadRequest)
	spare := h.benchID(h.adminCall(http.MethodPost, datasets, map[string]any{"name": "Jetable", "reference_speaker_ids": []int64{refA.ID, refB.ID}, "test_speaker_ids": []int64{testA.ID}}, http.StatusCreated))
	var listed struct {
		Datasets []struct {
			ID                                int64
			References, Tests, Persons, Clips int
		} `json:"datasets"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, datasets, nil, http.StatusOK), &listed); err != nil || len(listed.Datasets) != 2 {
		t.Fatalf("two datasets are listed, got %d (%v)", len(listed.Datasets), err)
	}
	if d := listed.Datasets[1]; d.ID != dataset || d.References != 2 || d.Tests != 2 || d.Persons != 2 || d.Clips != 4 {
		t.Fatalf("the counts describe the dataset: %+v", d)
	}
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", datasets, spare), nil, http.StatusNoContent)
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", datasets, spare), nil, http.StatusNotFound)

	// The dataset stays while a run refers to it, and the list shows every run.
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", datasets, dataset), nil, http.StatusConflict)
	var list struct {
		Runs []benchRunBody `json:"runs"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, runs, nil, http.StatusOK), &list); err != nil || len(list.Runs) != 4 {
		t.Fatalf("the history lists the four runs, got %d (%v)", len(list.Runs), err)
	}
}
