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

// testAIBenchmarkSeparation: a chain may carry a separation model (#402). Its separation task comes once the chain's
// diarization and transcription of a clip are done, asks for the model, reads their results, and its result is only
// stored: nothing reaches the transcripts, the fingerprints or the References of the library.
func testAIBenchmarkSeparation(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h, _, datasetID := e2eCorpus(t, pool)
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,separated_voices,sample_rate_hz) VALUES('bench-sep','separation','pyannote','fixture/sep','0123456789abcdef0123456789abcdef01234567',2,16000)`); err != nil {
		t.Fatal(err)
	}
	h.reportModel("bench-sep", "ready", "", 1, 1, http.StatusNoContent)
	runs := "/api/admin/benchmarks/runs"
	body := func(separation string) map[string]any {
		return map[string]any{"dataset_id": datasetID, "stage": "e2e", "chains": []map[string]any{{"diarization_model_id": "bench-diar-a", "transcription_model": "asr-x", "separation_model_id": separation}}}
	}
	h.adminCall(http.MethodPost, runs, body("bench-diar-b"), http.StatusUnprocessableEntity) // not a separation model

	library := func() string {
		return fmt.Sprint(h.count(`SELECT count(*) FROM voice_fingerprints`), h.count(`SELECT count(*) FROM local_speakers`), h.count(`SELECT count(*) FROM segment_corrections`),
			h.count(`SELECT count(*) FROM transcript_revisions`), h.count(`SELECT count(*) FROM embedding_tasks`), h.count(`SELECT count(*) FROM jobs WHERE step NOT LIKE 'benchmark.%'`))
	}
	before := library()

	upstream := func() {
		t.Helper()
		for i := 0; i < 2; i++ {
			job, ok := h.claimBenchmarkTask()
			if !ok {
				t.Fatalf("upstream task %d of 2 must be leased", i)
			}
			switch job.Step {
			case "benchmark.transcription.9":
				h.complete(job, map[string]any{"provider": "fixture", "model": "asr-x", "version": "v", "language": "fr", "audio_stream_index": 9, "segments": []map[string]any{
					{"start": 0.0, "end": 5.0, "text": "bonjour", "words": []map[string]any{{"start": 0.5, "end": 1.0, "word": "bonjour"}}}}}, http.StatusOK)
			case "benchmark.diarization.9":
				h.complete(job, map[string]any{"speakers": []map[string]any{
					{"label": "S0", "turns": []map[string]any{{"start": 0.0, "end": 5.0}}},
					{"label": "S1", "turns": []map[string]any{{"start": 3.0, "end": 8.0}}}}}, http.StatusOK)
			default:
				t.Fatalf("no separation before the chain's own tasks are done, got %s", job.Step)
			}
		}
	}
	detail := func(id int64) (status string, metrics json.RawMessage) {
		var v struct {
			Status string `json:"status"`
			Chains []struct {
				SeparationModelID string `json:"separation_model_id"`
			} `json:"chains"`
			Results []struct {
				Metrics json.RawMessage `json:"metrics"`
			} `json:"results"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("%s/%d", runs, id), nil, http.StatusOK), &v); err != nil {
			t.Fatal(err)
		}
		if len(v.Chains) != 1 || v.Chains[0].SeparationModelID != "bench-sep" {
			t.Fatalf("the chain keeps its separation model: %+v", v.Chains)
		}
		if len(v.Results) == 1 {
			metrics = v.Results[0].Metrics
		}
		return v.Status, metrics
	}

	var run struct{ ID int64 }
	if err := json.Unmarshal(h.adminCall(http.MethodPost, runs, body("bench-sep"), http.StatusCreated), &run); err != nil {
		t.Fatal(err)
	}
	upstream()
	if status, _ := detail(run.ID); status != "running" {
		t.Fatalf("the run waits for its separation task, got %s", status)
	}
	job, ok := h.claimBenchmarkTask()
	if !ok || job.Step != "benchmark.separation.9" {
		t.Fatalf("the separation task follows the chain's tasks: %+v %v", job, ok)
	}
	if model := h.count(`SELECT count(*) FROM jobs WHERE id=$1 AND requested_model_id='bench-sep'`, job.ID); model != 1 {
		t.Fatal("the separation task asks for its model")
	}
	var input struct {
		TranscriptionModel string `json:"transcription_model"`
		Voice              struct {
			Diarization struct {
				Speakers []struct {
					Label string `json:"label"`
					Turns []struct {
						Index int     `json:"index"`
						Start float64 `json:"start"`
					} `json:"turns"`
				} `json:"speakers"`
			} `json:"diarization"`
		} `json:"voice"`
		Separation struct {
			Model           string  `json:"model"`
			Revision        string  `json:"revision"`
			AmbiguityMargin float64 `json:"ambiguity_margin"`
			Segments        []struct {
				Words []struct{ Word string } `json:"words"`
			} `json:"segments"`
		} `json:"separation"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, http.StatusOK), &input); err != nil {
		t.Fatal(err)
	}
	if input.TranscriptionModel != "asr-x" || input.Separation.Model != "fixture/sep" || input.Separation.Revision == "" || input.Separation.AmbiguityMargin <= 0 ||
		len(input.Separation.Segments) != 1 || input.Separation.Segments[0].Words[0].Word != "bonjour" ||
		len(input.Voice.Diarization.Speakers) != 2 || input.Voice.Diarization.Speakers[1].Turns[0].Start != 3 {
		t.Fatalf("the task reads its chain's transcript and turns: %+v", input)
	}
	h.complete(job, map[string]any{"audio_stream_index": 7, "zones": []any{}, "sources": []any{}, "added": []any{}}, http.StatusBadRequest) // another track
	h.complete(job, map[string]any{"audio_stream_index": 9,
		"zones":   []map[string]any{{"start": 3.0, "end": 5.0, "silent_speakers": []string{"S1"}}},
		"sources": []map[string]any{{"zone": 0, "index": 0, "speaker": "S1", "words": []map[string]any{{"start": 3.5, "end": 4.0, "word": "salut"}}}, {"zone": 0, "index": 1, "speaker": nil, "words": []any{}}},
		"added":   []map[string]any{{"zone": 0, "source": 0, "speaker": "S1", "start": 3.5, "end": 4.0, "word": "salut"}}}, http.StatusOK)
	status, metrics := detail(run.ID)
	var m struct {
		CPWER      *float64        `json:"cpwer"`
		Separation json.RawMessage `json:"separation"`
	}
	if err := json.Unmarshal(metrics, &m); err != nil {
		t.Fatal(err)
	}
	if status != "completed" || m.CPWER == nil || m.Separation == nil {
		t.Fatalf("the run settles with its separation counts: %s %s", status, metrics)
	}
	var raw map[string]map[string]int
	_ = json.Unmarshal(metrics, &raw)
	if got := fmt.Sprint(raw["separation"]); got != "map[sources:2 sources_without_speaker:1 speakers_without_words_after:0 speakers_without_words_before:1 tracks:1 words_added:1 zones:1]" {
		t.Fatalf("raw separation counts: %s", got)
	}
	if after := library(); after != before {
		t.Fatalf("a run with separation writes nothing to the library: %s then %s", before, after)
	}

	// A lack of GPU memory ends the task at once with its stable cause: the same window would fail again.
	if err := json.Unmarshal(h.adminCall(http.MethodPost, runs, body("bench-sep"), http.StatusCreated), &run); err != nil {
		t.Fatal(err)
	}
	upstream()
	detail(run.ID)
	job, ok = h.claimBenchmarkTask()
	if !ok || job.Step != "benchmark.separation.9" {
		t.Fatalf("second separation task: %+v %v", job, ok)
	}
	h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", job.ID), map[string]any{"error": "gpu_out_of_memory"}, job.Lease, http.StatusOK)
	if n := h.count(`SELECT count(*) FROM jobs WHERE id=$1 AND status='dead' AND last_error='gpu_out_of_memory' AND attempts=1`, job.ID); n != 1 {
		t.Fatal("a separation task out of GPU memory is not retried")
	}
	if status, _ = detail(run.ID); status != "completed" {
		t.Fatalf("the chain is still measured without separation, got %s", status)
	}
}
