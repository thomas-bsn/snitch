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

func (h *voiceHarness) claimBenchmarkDiarization() (claimedJob, bool) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"benchmark.diarization.*"}}, "", http.StatusOK)
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

func testAIBenchmarkDiarization(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 1)
	clip := h.clipIDs[0]
	for _, id := range []string{"bench-diar-a", "bench-diar-b"} {
		if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,'diarization','pyannote',$2,'0123456789abcdef0123456789abcdef01234567') ON CONFLICT DO NOTHING`, id, "fixture/"+id); err != nil {
			t.Fatal(err)
		}
		h.reportModel(id, "ready", "", 1, 1, http.StatusNoContent)
	}
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent) // a real provider is only active once a Worker holds its model
	h.putPipeline(map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5})

	// Two voices of the clip with their turns, validated as Alice and Bob: the reference of the corpus.
	alice := h.fixtureSpeaker(clip, "A", usableAt(0))
	bob := h.fixtureSpeaker(clip, "B", usableAt(90))
	turns := func(speaker int64, spans ...[2]float64) {
		for i, s := range spans {
			if _, err := pool.Exec(ctx, `INSERT INTO speaker_turns(local_speaker_id,clip_id,turn_index,start_seconds,end_seconds) VALUES($1,$2,$3,$4,$5)`, speaker, clip, i, s[0], s[1]); err != nil {
				t.Fatal(err)
			}
		}
	}
	turns(alice.ID, [2]float64{0, 10}, [2]float64{20, 30})
	turns(bob.ID, [2]float64{10, 20})
	h.validate(alice, "Alice")
	h.validate(bob, "Bob")

	datasets, runs := "/api/admin/benchmarks/datasets", "/api/admin/benchmarks/runs"
	h.adminCall(http.MethodPost, datasets, map[string]any{"name": "Sans certification", "test_clip_ids": []int64{clip}, "reference_clip_ids": []int64{}, "verified_clip_ids": []int64{}}, http.StatusUnprocessableEntity)
	var created struct {
		ID     int64    `json:"id"`
		Stages []string `json:"stages"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, datasets, map[string]any{"name": "Voix relues", "test_clip_ids": []int64{clip}, "reference_clip_ids": []int64{}, "verified_clip_ids": []int64{clip}}, http.StatusCreated), &created); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(created.Stages) != "[diarization]" {
		t.Fatalf("a certified clip with validated voices measures the diarization, got %v", created.Stages)
	}
	if n := h.count(`SELECT count(*) FROM benchmark_reference_turns WHERE dataset_id=$1 AND audio_stream_index=9`, created.ID); n != 3 {
		t.Fatalf("the reference keeps the three turns, got %d", n)
	}
	if n := h.count(`SELECT count(DISTINCT speaker) FROM benchmark_reference_turns WHERE dataset_id=$1 AND speaker LIKE 'p%'`, created.ID); n != 2 {
		t.Fatalf("a speaker is the validated Person, got %d", n)
	}

	var models struct {
		Models []struct {
			ID           string `json:"id"`
			ReadyWorkers int    `json:"ready_workers"`
		} `json:"models"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/benchmarks/models?stage=diarization", nil, http.StatusOK), &models); err != nil {
		t.Fatal(err)
	}
	ready := map[string]int{}
	for _, m := range models.Models {
		ready[m.ID] = m.ReadyWorkers
	}
	if ready["bench-diar-a"] != 1 || ready["bench-diar-b"] != 1 {
		t.Fatalf("the diarization models of the catalogue are offered with their Workers: %v", ready)
	}

	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": created.ID, "stage": "diarization", "model_ids": []string{benchModelID}}, http.StatusUnprocessableEntity) // a fingerprint model
	var run struct{ ID int64 }
	if err := json.Unmarshal(h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": created.ID, "stage": "diarization", "model_ids": []string{"bench-diar-a", "bench-diar-b"}}, http.StatusCreated), &run); err != nil {
		t.Fatal(err)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE step='benchmark.diarization.9' AND status='pending'`); n != 2 {
		t.Fatalf("one task per model and track, got %d", n)
	}

	results := map[string]map[string]any{
		"fixture/bench-diarization-a": {"speakers": []map[string]any{
			{"label": "S0", "turns": []map[string]any{{"start": 0.0, "end": 10.1}, {"start": 20.0, "end": 30.0}}},
			{"label": "S1", "turns": []map[string]any{{"start": 10.1, "end": 20.0}}}}},
		"fixture/bench-diarization-b": {"speakers": []map[string]any{
			{"label": "S0", "turns": []map[string]any{{"start": 0.0, "end": 30.0}}}}},
	}
	_ = results
	for i := 0; i < 2; i++ {
		job, ok := h.claimBenchmarkDiarization()
		if !ok || job.Step != "benchmark.diarization.9" {
			t.Fatalf("a benchmark diarization must be leased, got %+v", job)
		}
		var input struct {
			Voice struct {
				Config struct {
					Provider string `json:"provider"`
					Model    string `json:"model"`
				} `json:"config"`
			} `json:"voice"`
		}
		if err := json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, http.StatusOK), &input); err != nil {
			t.Fatal(err)
		}
		model := input.Voice.Config.Model
		if input.Voice.Config.Provider != "pyannote" || (model != "fixture/bench-diar-a" && model != "fixture/bench-diar-b") {
			t.Fatalf("the task carries the requested model in the production configuration: %+v", input)
		}
		h.complete(job, map[string]any{"provider": "pyannote"}, http.StatusBadRequest) // no speakers
		if model == "fixture/bench-diar-a" {
			h.complete(job, results["fixture/bench-diarization-a"], http.StatusOK)
		} else {
			h.complete(job, results["fixture/bench-diarization-b"], http.StatusOK)
		}
	}

	var detail struct {
		Status  string `json:"status"`
		Results []struct {
			ModelID string          `json:"model_id"`
			State   string          `json:"state"`
			Metrics json.RawMessage `json:"metrics"`
		} `json:"results"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("%s/%d", runs, run.ID), nil, http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Status != "completed" || len(detail.Results) != 2 {
		t.Fatalf("the run settles once both tasks ended: %+v", detail)
	}
	der := map[string]float64{}
	for _, res := range detail.Results {
		var m struct {
			DER *float64 `json:"der"`
		}
		_ = json.Unmarshal(res.Metrics, &m)
		if m.DER == nil || res.State != "complete" {
			t.Fatalf("a measured model has a DER: %+v %s", res, res.Metrics)
		}
		der[res.ModelID] = *m.DER
	}
	if der["bench-diar-a"] > 0.02 || der["bench-diar-b"] < 0.25 {
		t.Fatalf("a good diarization has little error, one merging the speakers has much: %v", der)
	}

	// Nothing reached production: the local speakers and the diarization jobs are the ones of the fixture.
	if n := h.count(`SELECT count(*) FROM local_speakers WHERE clip_id=$1`, clip); n != 2 {
		t.Fatalf("a benchmark never changes the local speakers, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE step LIKE 'voice.diarization.%'`); n != 2 {
		t.Fatalf("a benchmark diarization is never a production one, got %d", n)
	}
}
