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
		{http.MethodGet, "/api/admin/benchmarks/datasets"}, {http.MethodGet, "/api/admin/benchmarks/clips"}, {http.MethodGet, "/api/admin/benchmarks/models?stage=fingerprint"}, {http.MethodPost, "/api/admin/benchmarks/runs"},
		{http.MethodGet, "/api/admin/benchmarks/runs/1"}, {http.MethodPost, "/api/admin/benchmarks/runs/1/cancel"},
	} {
		if status, _ := h.userStatus(member, call.method, call.path, map[string]any{}); status != http.StatusForbidden {
			t.Fatalf("%s %s must be refused to a member, got %d", call.method, call.path, status)
		}
	}

	// Corpus rules: shared clips only, one role per clip, certification for test clips only, one stage at least.
	c0, c1, c2, c3, c4 := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2], h.clipIDs[3], h.clipIDs[4]
	if _, err := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,result) VALUES($1,'transcription.audio.1','bench-transcription','done',$2::jsonb)`, c2, `{"provider":"fixture","model":"m","audio_stream_index":1,"segments":[{"start":0,"end":1,"text":"bonjour","words":[]}]}`); err != nil {
		t.Fatal(err)
	}
	datasets := "/api/admin/benchmarks/datasets"
	bad := func(body map[string]any, want int) {
		t.Helper()
		h.adminCall(http.MethodPost, datasets, body, want)
	}
	corpus := func(name string, test, reference, verified []int64) map[string]any {
		return map[string]any{"name": name, "test_clip_ids": test, "reference_clip_ids": reference, "verified_clip_ids": verified}
	}
	bad(corpus("x", []int64{c4}, []int64{}, []int64{}), http.StatusUnprocessableEntity)           // no validated voice, nothing certified: no stage to measure
	bad(corpus("x", []int64{c2}, []int64{c2}, []int64{}), http.StatusUnprocessableEntity)         // a clip is a test or a reference, once
	bad(corpus("x", []int64{c2}, []int64{c0, c1}, []int64{c0}), http.StatusUnprocessableEntity)   // only a test clip is certified
	bad(corpus("x", []int64{999999}, []int64{c0, c1}, []int64{}), http.StatusUnprocessableEntity) // unknown clip
	bad(corpus(" ", []int64{c2}, []int64{c0, c1}, []int64{}), http.StatusBadRequest)              // blank name
	bad(corpus("x", []int64{}, []int64{c0, c1}, []int64{}), http.StatusBadRequest)                // nothing to measure
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private' WHERE id=$1`, c1); err != nil {
		t.Fatal(err)
	}
	bad(corpus("x", []int64{c2}, []int64{c0, c1}, []int64{}), http.StatusUnprocessableEntity) // a private clip never enters a corpus
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, c1); err != nil {
		t.Fatal(err)
	}
	var created struct {
		ID     int64    `json:"id"`
		Stages []string `json:"stages"`
	}
	create := func(body map[string]any) (int64, []string) {
		t.Helper()
		if err := json.Unmarshal(h.adminCall(http.MethodPost, datasets, body, http.StatusCreated), &created); err != nil || created.ID < 1 {
			t.Fatalf("a corpus id is expected: %v", err)
		}
		return created.ID, created.Stages
	}
	dataset, stages := create(corpus("Voix validées", []int64{c2, c3}, []int64{c0, c1}, []int64{}))
	if fmt.Sprint(stages) != "[fingerprint]" {
		t.Fatalf("a corpus of validated voices alone measures the fingerprint only, got %v", stages)
	}
	bad(corpus("voix VALIDÉES", []int64{c2}, []int64{c0, c1}, []int64{}), http.StatusConflict)
	certified, stages := create(corpus("Certifié", []int64{c2, c3}, []int64{c0, c1}, []int64{c2}))
	if fmt.Sprint(stages) != "[fingerprint transcription diarization e2e]" {
		t.Fatalf("a certified clip opens every stage, got %v", stages)
	}
	noFingerprint, stages := create(corpus("Sans empreinte", []int64{c2}, []int64{}, []int64{c2}))
	if fmt.Sprint(stages) != "[transcription diarization e2e]" {
		t.Fatalf("without references the fingerprint is out, the other three stages are measurable, got %v", stages)
	}

	// The clips and the models a screen offers.
	var offered struct {
		Clips []struct {
			ID                int64 `json:"id"`
			SegmentsTotal     int   `json:"segments_total"`
			SegmentsCorrected int   `json:"segments_corrected"`
			ValidatedPersons  []struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"validated_persons"`
		} `json:"clips"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/benchmarks/clips", nil, http.StatusOK), &offered); err != nil {
		t.Fatal(err)
	}
	found := map[int64]bool{}
	for _, c := range offered.Clips {
		found[c.ID] = true
		if c.ID == c2 && (c.SegmentsTotal != 1 || c.SegmentsCorrected != 0 || len(c.ValidatedPersons) != 1 || c.ValidatedPersons[0].Name != "Alice" || c.ValidatedPersons[0].ID < 1) {
			t.Fatalf("the clip describes its transcript and validated voices: %+v", c)
		}
	}
	if !found[c0] || !found[c2] || found[c4] {
		t.Fatalf("clips with a validated voice or a transcript are offered, the others not: %v", found)
	}
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private' WHERE id=$1`, c0); err != nil {
		t.Fatal(err)
	}
	offered.Clips = nil
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/benchmarks/clips", nil, http.StatusOK), &offered); err != nil {
		t.Fatal(err)
	}
	for _, c := range offered.Clips {
		if c.ID == c0 {
			t.Fatal("a private clip is never offered")
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, c0); err != nil {
		t.Fatal(err)
	}
	// A clip only found in a private folder is never offered nor accepted, even when it is shared. A clip with
	// another live provenance stays open (ADR 0026), so this one has only the private file.
	var admin, source, secret int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'secret.mp4',1,now(),now()) RETURNING id`, admin).Scan(&secret); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,result) VALUES($1,'transcription.audio.1','secret-transcription','done',$2::jsonb)`, secret, `{"provider":"fixture","model":"m","audio_stream_index":1,"segments":[{"start":0,"end":1,"text":"secret","words":[]}]}`); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO clip_sources(name,root_path,owner_id) VALUES('bench-private','/bench-private',$1) RETURNING id`, admin).Scan(&source); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,clip_id,size_bytes,modified_at,last_seen_generation,status) VALUES($1,'secret/a.mp4',$2,1,now(),1,'ready')`, source, secret); err != nil {
		t.Fatal(err)
	}
	offeredIDs := func() map[int64]bool {
		t.Helper()
		var list struct {
			Clips []struct {
				ID int64 `json:"id"`
			} `json:"clips"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/benchmarks/clips", nil, http.StatusOK), &list); err != nil {
			t.Fatal(err)
		}
		out := map[int64]bool{}
		for _, c := range list.Clips {
			out[c.ID] = true
		}
		return out
	}
	if !offeredIDs()[secret] {
		t.Fatal("a shared clip outside any private folder is offered")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO private_folders(source_id,folder) VALUES($1,'secret')`, source); err != nil {
		t.Fatal(err)
	}
	if offeredIDs()[secret] {
		t.Fatal("a clip in a private folder is never offered")
	}
	bad(corpus("x", []int64{c2, secret}, []int64{c0, c1}, []int64{}), http.StatusUnprocessableEntity)
	if _, err := pool.Exec(ctx, `DELETE FROM private_folders WHERE source_id=$1`, source); err != nil {
		t.Fatal(err)
	}
	// Certifying a clip that has neither a transcript nor a validated voice vouches for nothing.
	bad(corpus("x", []int64{c2, c4}, []int64{c0, c1}, []int64{c4}), http.StatusUnprocessableEntity)

	var models struct {
		Models []struct {
			ID           string `json:"id"`
			Active       bool   `json:"active"`
			ReadyWorkers int    `json:"ready_workers"`
			TotalWorkers int    `json:"total_workers"`
		} `json:"models"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/benchmarks/models?stage=fingerprint", nil, http.StatusOK), &models); err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, m := range models.Models {
		seen[m.ID] = m.ReadyWorkers
		if m.TotalWorkers < 1 {
			t.Fatalf("the Worker total is reported: %+v", m)
		}
		if m.ID == benchModelID && m.Active {
			t.Fatal("the benchmark candidate is not the active model")
		}
	}
	if seen[benchModelID] != 1 {
		t.Fatalf("the model the Worker reported ready counts one Worker, got %v", seen)
	}
	if n, ok := seen["pyannote-wespeaker-resnet34"]; !ok || n != 0 {
		t.Fatalf("the active model is listed, ready on no Worker here: %v", seen)
	}
	h.adminCall(http.MethodGet, "/api/admin/benchmarks/models?stage=e2e", nil, http.StatusUnprocessableEntity) // not an executable stage
	h.adminCall(http.MethodGet, "/api/admin/benchmarks/models", nil, http.StatusUnprocessableEntity)

	runs := "/api/admin/benchmarks/runs"
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{"unknown-model"}}, http.StatusUnprocessableEntity)
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "transcription", "model_ids": []string{benchModelID}}, http.StatusUnprocessableEntity) // this corpus has no certified clip
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "e2e", "model_ids": []string{benchModelID}}, http.StatusBadRequest)                    // not executable yet
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "diarization", "model_ids": []string{benchModelID}}, http.StatusUnprocessableEntity)   // a fingerprint model is no diarization model
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": noFingerprint, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusUnprocessableEntity)
	_ = certified

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
	// The fingerprints of a run are not the staging space of a model transition: they never count as a
	// recomputed reference and would never be promoted to production by its switch.
	refs, err := loadReferences(ctx, pool, "fixture-embedding", benchModelRepo)
	if err != nil || len(refs) == 0 {
		t.Fatalf("the production references are listed: %d (%v)", len(refs), err)
	}
	for _, r := range refs {
		if r.Done {
			t.Fatalf("a benchmark run fingerprint must not stand for a transition recomputation: %+v", r)
		}
	}
	// Reading again changes nothing: the settlement is idempotent.
	if again := h.benchRun(second); len(again.Results) != 1 || again.Status != "completed" {
		t.Fatalf("settling twice must not duplicate results: %+v", again)
	}

	// The benchmark never touches production: fingerprints, validations, active model.
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE NOT benchmark AND model='fixture-embedding'`); n != 4 {
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
	spare, _ := create(corpus("Jetable", []int64{c2}, []int64{c0, c1}, []int64{}))
	var listed struct {
		Datasets []struct {
			ID             int64    `json:"id"`
			TestClips      int      `json:"test_clips"`
			ReferenceClips int      `json:"reference_clips"`
			VerifiedClips  int      `json:"verified_clips"`
			Persons        int      `json:"persons"`
			Stages         []string `json:"stages"`
		} `json:"datasets"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, datasets, nil, http.StatusOK), &listed); err != nil || len(listed.Datasets) != 4 {
		t.Fatalf("four corpora are listed, got %d (%v)", len(listed.Datasets), err)
	}
	for _, d := range listed.Datasets {
		if d.ID == dataset && (d.TestClips != 2 || d.ReferenceClips != 2 || d.VerifiedClips != 0 || d.Persons != 2 || fmt.Sprint(d.Stages) != "[fingerprint]") {
			t.Fatalf("the counts describe the corpus: %+v", d)
		}
		if d.ID == certified && d.VerifiedClips != 1 {
			t.Fatalf("the certification is counted: %+v", d)
		}
	}
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", datasets, spare), nil, http.StatusNoContent)
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", datasets, spare), nil, http.StatusNotFound)

	// A run nobody read since its jobs ended does not block the next one.
	fifth := h.benchID(h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusCreated))
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='dead',last_error='worker_error' WHERE id IN (SELECT job_id FROM benchmark_run_jobs WHERE run_id=$1)`, fifth); err != nil {
		t.Fatal(err)
	}
	// An open fingerprint transition recomputes the same groups: no benchmark starts meanwhile.
	if _, err := pool.Exec(ctx, `INSERT INTO voice_fingerprint_transitions(from_model_id,to_model_id,announced_references,author_name) VALUES($1,$2,1,'test')`, activeBefore, benchModelID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusConflict)
	if _, err := pool.Exec(ctx, `DELETE FROM voice_fingerprint_transitions`); err != nil {
		t.Fatal(err)
	}
	sixth := h.benchID(h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": dataset, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusCreated))
	if got := h.benchRun(fifth); got.Status != "failed" {
		t.Fatalf("the unread run was settled when the next one started: %+v", got)
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("%s/%d/cancel", runs, sixth), nil, http.StatusOK)

	// The dataset stays while a run refers to it, and the list shows every run.
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", datasets, dataset), nil, http.StatusConflict)
	var list struct {
		Runs []benchRunBody `json:"runs"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, runs, nil, http.StatusOK), &list); err != nil || len(list.Runs) != 6 {
		t.Fatalf("the history lists the six runs, got %d (%v)", len(list.Runs), err)
	}
}
