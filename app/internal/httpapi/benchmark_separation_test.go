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
	// The certified clip contains a Chevauchement: Bob starts before Alice ends.
	if _, err := pool.Exec(ctx, `UPDATE benchmark_reference_turns SET start_seconds=4 WHERE dataset_id=$1 AND start_seconds=5`, datasetID); err != nil {
		t.Fatal(err)
	}

	upstream := func() {
		t.Helper()
		for i := 0; i < 2; i++ {
			job, ok := h.claimBenchmarkTask()
			if !ok {
				t.Fatalf("upstream task %d of 2 must be leased", i)
			}
			switch job.Step {
			case "benchmark.transcription.9":
				// Bob's "salut alice" is lost under Alice; "oui" falls in a turn of S1 only, between two words of S0. A
				// Worker reporting its cost (inference time, no GPU for an external transcription).
				word := func(start, end float64, w string) map[string]any {
					return map[string]any{"start": start, "end": end, "word": w}
				}
				h.complete(job, map[string]any{"provider": "fixture", "model": "asr-x", "version": "v", "language": "fr", "audio_stream_index": 9, "inference_seconds": 1.5, "segments": []map[string]any{
					{"start": 0.0, "end": 15.0, "text": "bonjour tout le monde oui on commence", "words": []map[string]any{
						word(0.5, 1, "bonjour"), word(1.5, 2, "tout"), word(2, 2.5, "le"), word(3, 3.5, "monde"), word(4.55, 4.75, "oui"), word(10.5, 11, "on"), word(11, 12, "commence")}}}}, http.StatusOK)
			case "benchmark.diarization.9":
				// An old Worker: no cost reported.
				h.complete(job, map[string]any{"speakers": []map[string]any{
					{"label": "S0", "turns": []map[string]any{{"start": 0.0, "end": 4.5}, {"start": 4.8, "end": 15.0}}},
					{"label": "S1", "turns": []map[string]any{{"start": 4.0, "end": 10.0}}}}}, http.StatusOK)
			default:
				t.Fatalf("no separation before the chain's own tasks are done, got %s", job.Step)
			}
		}
	}
	var resources json.RawMessage
	detail := func(id int64) (status string, metrics json.RawMessage) {
		var v struct {
			Status string `json:"status"`
			Chains []struct {
				SeparationModelID string `json:"separation_model_id"`
			} `json:"chains"`
			Results []struct {
				Metrics   json.RawMessage `json:"metrics"`
				Resources json.RawMessage `json:"resources"`
			} `json:"results"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("%s/%d", runs, id), nil, http.StatusOK), &v); err != nil {
			t.Fatal(err)
		}
		if len(v.Chains) != 1 || v.Chains[0].SeparationModelID != "bench-sep" {
			t.Fatalf("the chain keeps its separation model: %+v", v.Chains)
		}
		if len(v.Results) == 1 {
			metrics, resources = v.Results[0].Metrics, v.Results[0].Resources
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
			Model           string   `json:"model"`
			Revision        string   `json:"revision"`
			AmbiguityMargin float64  `json:"ambiguity_margin"`
			SilentSpeakers  []string `json:"silent_speakers"`
			WordSpans       []struct {
				Speaker    string
				Start, End float64
			} `json:"word_spans"`
			Segments []struct {
				Words []struct{ Word string } `json:"words"`
			} `json:"segments"`
		} `json:"separation"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, http.StatusOK), &input); err != nil {
		t.Fatal(err)
	}
	if input.TranscriptionModel != "asr-x" || input.Separation.Model != "fixture/sep" || input.Separation.Revision == "" || input.Separation.AmbiguityMargin <= 0 ||
		len(input.Separation.Segments) != 1 || input.Separation.Segments[0].Words[0].Word != "bonjour" ||
		len(input.Voice.Diarization.Speakers) != 2 || input.Voice.Diarization.Speakers[1].Turns[0].Start != 4 {
		t.Fatalf("the task reads its chain's transcript and turns: %+v", input)
	}
	// Parity with production (#403): S1 covers "oui" most, but a single word between two words of S0 is S0's under the
	// word rule (segmentparts), so S1 is without words; a rule of the Worker's own gave "oui" to S1.
	if fmt.Sprint(input.Separation.SilentSpeakers) != "[S1]" {
		t.Fatalf("the speakers without words follow the production rule: %v", input.Separation.SilentSpeakers)
	}
	// The same rule gives the Worker the times of each speaker's words, to merge a source zone by zone: S0 holds all 7.
	if spans := input.Separation.WordSpans; len(spans) != 7 || spans[0].Speaker != "S0" || spans[4].Speaker != "S0" || spans[4].Start != 4.55 || spans[6].End != 12 {
		t.Fatalf("the word spans follow the production rule: %+v", spans)
	}
	h.complete(job, map[string]any{"audio_stream_index": 7, "zones": []any{}, "sources": []any{}, "added": []any{}}, http.StatusBadRequest) // another track
	word := func(start, end float64, w string) map[string]any {
		return map[string]any{"start": start, "end": end, "word": w}
	}
	added := func(start, end float64, w string) map[string]any {
		return map[string]any{"zone": 0, "source": 0, "speaker": "S1", "start": start, "end": end, "word": w}
	}
	// S1 gets Bob's words back and one word of nobody; the other source repeats "salut" at the same time.
	h.complete(job, map[string]any{"audio_stream_index": 9, "inference_seconds": 2.5, "gpu_peak_bytes": 1000,
		"silent_speakers": []string{"S1"},
		"zones":           []map[string]any{{"start": 4.8, "end": 10.0, "silent_speakers": []string{"S1"}}},
		"sources": []map[string]any{
			{"zone": 0, "index": 0, "speaker": "S1", "words": []map[string]any{word(5.5, 6, "salut"), word(6, 6.5, "alice"), word(7, 7.5, "bravo")}},
			{"zone": 0, "index": 1, "speaker": nil, "words": []map[string]any{word(5.6, 6, "salut"), word(8, 8.5, "monde")}}},
		"added": []map[string]any{added(5.5, 6, "salut"), added(6, 6.5, "alice"), added(7, 7.5, "bravo")}}, http.StatusOK)
	status, metrics := detail(run.ID)
	var sep struct {
		CPWER      *float64                   `json:"cpwer"`
		Separation map[string]json.RawMessage `json:"separation"`
	}
	if err := json.Unmarshal(metrics, &sep); err != nil {
		t.Fatal(err)
	}
	if status != "completed" || sep.CPWER == nil || sep.Separation == nil {
		t.Fatalf("the run settles with its separation measures: %s %s", status, metrics)
	}
	counts := map[string]string{}
	for k, v := range sep.Separation {
		counts[k] = string(v)
	}
	rates := func(key string) string {
		var r struct {
			Clips          int      `json:"clips"`
			ReferenceWords int      `json:"reference_words"`
			CPWordErrors   int      `json:"cp_word_errors"`
			WordErrors     int      `json:"word_errors"`
			CPWER          *float64 `json:"cpwer"`
			WER            *float64 `json:"wer"`
		}
		if json.Unmarshal(sep.Separation[key], &r) != nil || r.CPWER == nil || r.WER == nil {
			return "unreadable: " + string(sep.Separation[key])
		}
		return fmt.Sprintf("clips=%d ref=%d cp=%d w=%d cpwer=%v wer=%v", r.Clips, r.ReferenceWords, r.CPWordErrors, r.WordErrors, *r.CPWER, *r.WER)
	}
	// Without: Bob's two words lost and "oui" added (cpWER 3/8); with: Bob's words back, "bravo" added (2/8).
	if got := rates("without_separation"); got != "clips=1 ref=8 cp=3 w=2 cpwer=0.375 wer=0.25" {
		t.Fatalf("without separation: %s", got)
	}
	if got := rates("with_separation"); got != "clips=1 ref=8 cp=2 w=2 cpwer=0.25 wer=0.25" {
		t.Fatalf("with separation, on the same clips: %s", got)
	}
	delete(counts, "without_separation")
	delete(counts, "with_separation")
	if got := fmt.Sprint(counts); got != "map[min_clips:10 sources:2 sources_without_speaker:1 speakers_without_words_after:0 speakers_without_words_before:1 tracks:1 words_added:3 words_added_absent_from_reference:1 words_duplicated_between_sources:1 zones:1]" {
		t.Fatalf("separation counts: %s", got)
	}
	// Cost per model of the chain: an old Worker reported nothing (absent, not zero); no GPU for the transcription.
	if got := string(resources); got != `[{"model_id":"bench-diar-a","tasks":1,"inference_seconds":null,"gpu_peak_bytes":null},{"model_id":"asr-x","tasks":1,"inference_seconds":1.5,"gpu_peak_bytes":null},{"model_id":"bench-sep","tasks":1,"inference_seconds":2.5,"gpu_peak_bytes":1000}]` {
		t.Fatalf("resources per model: %s", got)
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
