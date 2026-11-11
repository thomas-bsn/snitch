//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type benchScoringBody struct {
	Stage   string `json:"stage"`
	Scoring *struct {
		ScoreMode        string   `json:"score_mode"`
		ScoreModeVersion *int     `json:"score_mode_version"`
		SampleCapSeconds *float64 `json:"reference_sample_cap_seconds"`
		EligibilityRule  *struct {
			Version                int     `json:"version"`
			ComparableMinSeconds   float64 `json:"comparable_min_seconds"`
			ContributiveMinSeconds float64 `json:"contributive_min_seconds"`
			MaxClippedRatio        float64 `json:"max_clipped_ratio"`
		} `json:"eligibility_rule"`
	} `json:"scoring"`
	Results []struct {
		State       string `json:"state"`
		ItemsTotal  int    `json:"items_total"`
		ItemsScored int    `json:"items_scored"`
		Metrics     struct {
			References  int      `json:"references"`
			Tests       int      `json:"tests"`
			Persons     int      `json:"persons"`
			Ineligible  int      `json:"ineligible"`
			GenuineMean *float64 `json:"genuine_mean_score"`
			EER         *float64 `json:"eer"`
		} `json:"metrics"`
	} `json:"results"`
}

func (h *voiceHarness) benchScoring(id int64) benchScoringBody {
	h.t.Helper()
	var run benchScoringBody
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/admin/benchmarks/runs/%d", id), nil, http.StatusOK), &run); err != nil {
		h.t.Fatal(err)
	}
	return run
}

// testBenchmarkScoring checks that the fingerprint benchmark scores like production (#397): for the same vectors
// and net speech, the score of a benchmark trial is the confidence production matching gives, in each score
// mode; the run records and shows the mode and the eligibility rule it used; a run recorded before them reads as
// the legacy centroid; and a run writes nothing in the library.
func testBenchmarkScoring(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 4)
	h.addBenchmarkCatalogModel()
	h.putPipeline(fakePipeline(4))
	h.reportModel(benchModelID, "ready", "", 1, 1, http.StatusNoContent)
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 4}, http.StatusOK)
	if _, err := pool.Exec(ctx, `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.99,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clips := h.clipIDs

	// Alice speaks 2 s on one clip (contributive, not comparable under the 3 s / 1 s rule of shortVoice) and 8 s on
	// another; Bob 5 s; the test voice of Alice 4 s. The weighted Reference, the unweighted centroid and the best
	// sample give three different scores.
	voices := map[string]shortVoice{
		"SHORT": {vector: []float64{1, 0, 0, 0}, net: 2},
		"LONG":  {vector: []float64{0, 1, 0, 0}, net: 8},
		"BOB":   {vector: []float64{0, 0, 1, 0}, net: 5},
		"TEST":  {vector: []float64{0.3, 1, 0, 0}, net: 4},
	}
	labels := []string{"SHORT", "LONG", "BOB", "TEST"}
	speakers := map[string]fixtureSpeaker{}
	for i, label := range labels {
		speakers[label] = h.shortVoice(clips[i], label, voices[label])
	}
	h.validate(speakers["SHORT"], "Alice")
	alice := h.personOf(speakers["SHORT"])
	h.validate(speakers["LONG"], alice)
	h.validate(speakers["BOB"], "Bob")

	production := func() float64 {
		t.Helper()
		got := h.matched(speakers["TEST"])
		if got.Person == nil || got.Person.ID != alice || got.Attribution == nil || got.Attribution.Confidence == nil {
			t.Fatalf("production must propose Alice for the test voice: %+v", got)
		}
		return *got.Attribution.Confidence
	}
	cumulative := production()
	if want := cosineOf(voices["TEST"].vector, []float64{2, 8, 0, 0}); math.Abs(cumulative-want) > 1e-6 {
		t.Fatalf("production scores against the weighted Reference %.6f, got %.6f", want, cumulative)
	}
	h.scoreMode(http.MethodPut, map[string]any{"mode": "best_sample"}, http.StatusOK)
	bestSample := production()
	h.scoreMode(http.MethodPut, map[string]any{"mode": "cumulative"}, http.StatusOK)
	centroid := cosineOf(voices["TEST"].vector, []float64{1, 1, 0, 0})
	if math.Abs(cumulative-bestSample) < 1e-3 || math.Abs(cumulative-centroid) < 1e-3 {
		t.Fatalf("the fixture must tell the modes apart: cumulative %.4f best sample %.4f centroid %.4f", cumulative, bestSample, centroid)
	}

	// The test voice is now validated to enter the corpus; its production score is already read.
	h.validate(speakers["TEST"], alice)
	pipeline := h.pipelineVersion()
	byDiarization := map[int64]string{}
	for _, label := range labels {
		d := h.diarizationOf(speakers[label])
		byDiarization[d] = label
		if _, err := pool.Exec(ctx, `UPDATE jobs SET input_version=$2 WHERE id=$1`, d, fmt.Sprintf("diar-p%s", pipeline)); err != nil {
			t.Fatal(err)
		}
	}
	var dataset struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/benchmarks/datasets", map[string]any{
		"name": "Voix brèves", "test_clip_ids": []int64{clips[3]}, "reference_clip_ids": []int64{clips[0], clips[1], clips[2]}, "verified_clip_ids": []int64{},
	}, http.StatusCreated), &dataset); err != nil {
		t.Fatal(err)
	}

	// The library as production reads it: samples, attributions and production fingerprints with their decision.
	library := func() string {
		t.Helper()
		var out string
		if err := pool.QueryRow(ctx, `SELECT concat_ws('|',
  (SELECT string_agg(concat_ws(',',id,person_id,local_speaker_id,retired_at),';' ORDER BY id) FROM voice_samples),
  (SELECT string_agg(concat_ws(',',local_speaker_id,person_id,validation_state,origin,confidence),';' ORDER BY local_speaker_id) FROM speaker_attributions),
  (SELECT string_agg(concat_ws(',',id,usable,contributive,unusable_reason,eligibility_rule_version),';' ORDER BY id) FROM voice_fingerprints WHERE NOT benchmark),
  (SELECT string_agg(concat_ws(',',id,display_name),';' ORDER BY id) FROM persons))`).Scan(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	// A run computes the same vectors, with the Worker's measures, through the benchmark model.
	run := func() int64 {
		t.Helper()
		id := h.benchID(h.adminCall(http.MethodPost, "/api/admin/benchmarks/runs", map[string]any{"dataset_id": dataset.ID, "stage": "fingerprint", "model_ids": []string{benchModelID}}, http.StatusCreated))
		for i := 0; i < len(labels); i++ {
			job, ok := h.claim()
			if !ok {
				t.Fatalf("job %d of %d must be leased", i, len(labels))
			}
			var label string
			var diarization int64
			for d, l := range byDiarization {
				if job.InputVersion == fmt.Sprintf("f5-a9-p%s-d%d", pipeline, d) {
					label, diarization = l, d
				}
			}
			if label == "" {
				t.Fatalf("unexpected job %+v", job)
			}
			v := voices[label]
			h.complete(job, map[string]any{
				"embedding_model": benchModelRepo, "embedding_model_version": "bench-v1", "audio_stream_index": 9, "diarization_job_id": diarization, "dimensions": 4,
				"fingerprints": []map[string]any{{"speaker_label": label, "scope": "speaker", "start": 0.0, "end": v.net, "quality": 0.9, "embedding": v.vector,
					"measures": map[string]any{"version": 1, "net_speech_seconds": v.net, "overlap_ratio": 0.0, "overlapped": false, "clipped_ratio": 0.0}}},
			}, http.StatusOK)
			h.release(job)
		}
		return id
	}
	checkRun := func(id int64, mode string, modeVersion int, genuine float64, references, ineligible int) {
		t.Helper()
		got := h.benchScoring(id)
		s := got.Scoring
		if s == nil || s.ScoreMode != mode || s.ScoreModeVersion == nil || *s.ScoreModeVersion != modeVersion || s.SampleCapSeconds == nil || *s.SampleCapSeconds != 10 ||
			s.EligibilityRule == nil || s.EligibilityRule.Version != 2 || s.EligibilityRule.ComparableMinSeconds != referenceComparable || s.EligibilityRule.ContributiveMinSeconds != referenceContributive || s.EligibilityRule.MaxClippedRatio != 0.05 {
			t.Fatalf("the run shows the mode %s (v%d) and the rule it used: %+v", mode, modeVersion, s)
		}
		if len(got.Results) != 1 {
			t.Fatalf("one result is expected: %+v", got)
		}
		r := got.Results[0]
		if r.Metrics.GenuineMean == nil || *r.Metrics.GenuineMean != genuine {
			t.Fatalf("for the same vectors the benchmark must give the production score %.17g, got %v", genuine, r.Metrics.GenuineMean)
		}
		if r.Metrics.References != references || r.Metrics.Ineligible != ineligible || r.Metrics.Tests != 1 || r.Metrics.Persons != 2 || r.ItemsTotal != 4 || r.ItemsScored != 4-ineligible {
			t.Fatalf("the rule of the run decides the fingerprints as production: %+v", r)
		}
	}

	before := library()
	cumulativeRun := run()
	checkRun(cumulativeRun, "cumulative", 3, cumulative, 3, 0)
	if after := library(); after != before {
		t.Fatalf("a run writes nothing in the library:\nbefore %s\nafter  %s", before, after)
	}
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE benchmark AND eligibility_rule_version IS NOT NULL`); n != 0 {
		t.Fatalf("the rule decides benchmark fingerprints without requalifying them, %d were", n)
	}

	// Under the best-sample mode, the short sample of Alice is no reference on its own: the score is the one of
	// her long sample, as in production. The first run keeps showing the mode it was scored under.
	h.scoreMode(http.MethodPut, map[string]any{"mode": "best_sample"}, http.StatusOK)
	before = library()
	bestRun := run()
	checkRun(bestRun, "best_sample", 4, bestSample, 2, 1)
	if after := library(); after != before {
		t.Fatalf("a run writes nothing in the library:\nbefore %s\nafter  %s", before, after)
	}
	checkRun(cumulativeRun, "cumulative", 3, cumulative, 3, 0)

	// A run recorded before #397 stays readable, its stored figures untouched, and reads as the legacy centroid.
	var legacy int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,params,author_name,finished_at)
  VALUES($1,'Ancien','fingerprint','completed',ARRAY[$2],'{"split":"reference/test","scope":"speaker"}','admin',now()) RETURNING id`, dataset.ID, benchModelID).Scan(&legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO benchmark_results(run_id,model_id,model,model_version,dimensions,state,items_total,items_scored,metrics)
  VALUES($1,$2,$3,'bench-v1',4,'complete',4,4,'{"references":3,"tests":1,"persons":2,"genuine_mean_score":0.88}')`, legacy, benchModelID, benchModelRepo); err != nil {
		t.Fatal(err)
	}
	old := h.benchScoring(legacy)
	if old.Scoring == nil || old.Scoring.ScoreMode != "centroid" || old.Scoring.EligibilityRule != nil || old.Scoring.ScoreModeVersion != nil {
		t.Fatalf("an older run reads as the legacy centroid, without a rule: %+v", old.Scoring)
	}
	if len(old.Results) != 1 || old.Results[0].Metrics.GenuineMean == nil || *old.Results[0].Metrics.GenuineMean != 0.88 {
		t.Fatalf("an older run keeps its figures: %+v", old.Results)
	}

	// A run still open when the mode was recorded is scored under the mode and rule active when it settles, and
	// records them.
	h.scoreMode(http.MethodPut, map[string]any{"mode": "cumulative"}, http.StatusOK)
	open := run()
	if _, err := pool.Exec(ctx, `UPDATE benchmark_runs SET params=params-'scoring',status='running',finished_at=NULL WHERE id=$1`, open); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM benchmark_results WHERE run_id=$1`, open); err != nil {
		t.Fatal(err)
	}
	checkRun(open, "cumulative", 5, cumulative, 3, 0)

	// Other stages carry no fingerprint scoring.
	var other int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,author_name,finished_at) VALUES($1,'Autre','transcription','completed',ARRAY['m'],'admin',now()) RETURNING id`, dataset.ID).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if got := h.benchScoring(other); got.Scoring != nil {
		t.Fatalf("a transcription run has no fingerprint scoring: %+v", got.Scoring)
	}
}
