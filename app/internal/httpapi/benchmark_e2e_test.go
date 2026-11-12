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

// claimBenchmarkTask leases the next benchmark task of either stage.
func (h *voiceHarness) claimBenchmarkTask() (claimedJob, bool) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"benchmark.diarization.*", "benchmark.transcription.*", "benchmark.separation.*"}}, "", http.StatusOK)
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

// e2eCorpus builds a corpus whose one clip, transcribed, has Alice and Bob speaking in turn with their voices validated.
func e2eCorpus(t *testing.T, pool *pgxpool.Pool) (*voiceHarness, int64, int64) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 1)
	clip := h.clipIDs[0]
	for _, id := range []string{"bench-diar-a", "bench-diar-b"} {
		if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,'diarization','pyannote',$2,'0123456789abcdef0123456789abcdef01234567') ON CONFLICT DO NOTHING`, id, "fixture/"+id); err != nil {
			t.Fatal(err)
		}
		h.reportModel(id, "ready", "", 1, 1, http.StatusNoContent)
	}
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent)
	h.putPipeline(map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5})

	// A clip where Alice and Bob speak in turn, transcribed, with its two voices validated.
	alice := h.fixtureSpeaker(clip, "A", usableAt(0))
	bob := h.fixtureSpeaker(clip, "B", usableAt(90))
	for speaker, spans := range map[int64][][2]float64{alice.ID: {{0, 5}, {10, 15}}, bob.ID: {{5, 10}}} {
		for i, s := range spans {
			if _, err := pool.Exec(ctx, `INSERT INTO speaker_turns(local_speaker_id,clip_id,turn_index,start_seconds,end_seconds) VALUES($1,$2,$3,$4,$5)`, speaker, clip, i, s[0], s[1]); err != nil {
				t.Fatal(err)
			}
		}
	}
	h.validate(alice, "Alice")
	h.validate(bob, "Bob")
	if _, err := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,result) VALUES($1,'transcription.audio.9','prod-transcription','done',$2::jsonb)`, clip,
		`{"provider":"fixture","model":"m","version":"v","audio_stream_index":9,"segments":[{"start":0,"end":5,"text":"bonjour tout le monde","words":[]},{"start":5,"end":10,"text":"salut alice","words":[]},{"start":10,"end":15,"text":"on commence","words":[]}]}`); err != nil {
		t.Fatal(err)
	}

	datasets := "/api/admin/benchmarks/datasets"
	var created struct {
		ID     int64    `json:"id"`
		Stages []string `json:"stages"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, datasets, map[string]any{"name": "Dialogue", "test_clip_ids": []int64{clip}, "reference_clip_ids": []int64{}, "verified_clip_ids": []int64{clip}}, http.StatusCreated), &created); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(created.Stages) != "[transcription diarization e2e]" {
		t.Fatalf("a certified, transcribed clip with validated voices measures the three stages, got %v", created.Stages)
	}
	return h, clip, created.ID
}

func testAIBenchmarkEndToEnd(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h, clip, datasetID := e2eCorpus(t, pool)
	runs := "/api/admin/benchmarks/runs"
	created := struct{ ID int64 }{datasetID}
	// Each reference segment keeps who spoke it.
	var speakers int
	if err := pool.QueryRow(ctx, `SELECT count(DISTINCT speaker) FROM benchmark_reference_segments WHERE dataset_id=$1 AND speaker IS NOT NULL`, created.ID).Scan(&speakers); err != nil || speakers != 2 {
		t.Fatalf("the reference segments carry two speakers: %d %v", speakers, err)
	}

	chain := func(diarization, transcription string) map[string]any {
		return map[string]any{"diarization_model_id": diarization, "transcription_model": transcription}
	}
	body := func(chains ...map[string]any) map[string]any {
		return map[string]any{"dataset_id": created.ID, "stage": "e2e", "chains": chains}
	}
	h.adminCall(http.MethodPost, runs, body(), http.StatusBadRequest)
	h.adminCall(http.MethodPost, runs, body(chain("bench-diar-a", "asr-x"), chain("bench-diar-a", "asr-x")), http.StatusBadRequest) // the same chain twice
	h.adminCall(http.MethodPost, runs, body(chain(benchModelID, "asr-x")), http.StatusUnprocessableEntity)                          // no diarization model
	h.adminCall(http.MethodPost, runs, body(chain("bench-diar-a", "")), http.StatusUnprocessableEntity)
	h.adminCall(http.MethodPost, runs, map[string]any{"dataset_id": created.ID, "stage": "e2e", "model_ids": []string{"x"}, "chains": []any{chain("bench-diar-a", "asr-x")}}, http.StatusBadRequest)

	// Two chains share the transcription model: its task runs once. Chain 1 diarizes well, chain 2 merges the speakers.
	var run struct{ ID int64 }
	if err := json.Unmarshal(h.adminCall(http.MethodPost, runs, body(chain("bench-diar-a", "asr-x"), chain("bench-diar-b", "asr-x")), http.StatusCreated), &run); err != nil {
		t.Fatal(err)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE step='benchmark.transcription.9' AND status='pending'`); n != 1 {
		t.Fatalf("a model shared by two chains runs once, got %d transcription tasks", n)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE step='benchmark.diarization.9' AND status='pending'`); n != 2 {
		t.Fatalf("each diarization model runs once, got %d", n)
	}
	for i := 0; i < 3; i++ {
		job, ok := h.claimBenchmarkTask()
		if !ok {
			t.Fatalf("task %d of 3 must be leased", i)
		}
		switch job.Step {
		case "benchmark.transcription.9":
			// Bob's answer falls in Alice's segment: only the word rule gives it back to him.
			word := func(start, end float64, text string) map[string]any {
				return map[string]any{"start": start, "end": end, "word": text}
			}
			h.complete(job, map[string]any{"provider": "fixture", "model": "asr-x", "version": "v", "language": "fr", "audio_stream_index": 9, "segments": []map[string]any{
				{"start": 0.0, "end": 10.0, "text": "Bonjour tout le monde salut Alice", "words": []any{
					word(0, 1, "Bonjour"), word(1, 2, "tout"), word(2, 3, "le"), word(3, 4, "monde"), word(5, 7, "salut"), word(7, 9, "Alice")}},
				{"start": 10.0, "end": 15.0, "text": "on commence", "words": []any{}}}}, http.StatusOK)
		case "benchmark.diarization.9":
			var input struct {
				Voice struct {
					Config struct {
						Model string `json:"model"`
					} `json:"config"`
				} `json:"voice"`
			}
			if err := json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, http.StatusOK), &input); err != nil {
				t.Fatal(err)
			}
			if input.Voice.Config.Model == "fixture/bench-diar-a" {
				h.complete(job, map[string]any{"speakers": []map[string]any{
					{"label": "S0", "turns": []map[string]any{{"start": 0.0, "end": 5.0}, {"start": 10.0, "end": 15.0}}},
					{"label": "S1", "turns": []map[string]any{{"start": 5.0, "end": 10.0}}}}}, http.StatusOK)
			} else {
				h.complete(job, map[string]any{"speakers": []map[string]any{{"label": "S0", "turns": []map[string]any{{"start": 0.0, "end": 15.0}}}}}, http.StatusOK)
			}
		default:
			t.Fatalf("unexpected task %s", job.Step)
		}
	}
	var detail struct {
		Status string `json:"status"`
		Chains []struct {
			ID                 string `json:"id"`
			DiarizationModelID string `json:"diarization_model_id"`
			TranscriptionModel string `json:"transcription_model"`
		} `json:"chains"`
		Results []struct {
			ModelID string          `json:"model_id"`
			Model   string          `json:"model"`
			State   string          `json:"state"`
			Metrics json.RawMessage `json:"metrics"`
		} `json:"results"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("%s/%d", runs, run.ID), nil, http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Status != "completed" || len(detail.Results) != 2 || len(detail.Chains) != 2 || detail.Chains[0].DiarizationModelID != "bench-diar-a" {
		t.Fatalf("the run settles with its two chains: %+v", detail)
	}
	rates := map[string]struct{ cpwer, wordCPWER, wer, der float64 }{}
	for _, res := range detail.Results {
		var m struct {
			CPWER    *float64 `json:"cpwer"`
			WER      *float64 `json:"wer"`
			DER      *float64 `json:"der"`
			WordRule struct {
				CPWER *float64 `json:"cpwer"`
			} `json:"word_rule"`
		}
		_ = json.Unmarshal(res.Metrics, &m)
		if m.CPWER == nil || m.WER == nil || m.DER == nil || m.WordRule.CPWER == nil || res.State != "complete" {
			t.Fatalf("a chain has its four rates: %+v %s", res, res.Metrics)
		}
		rates[res.ModelID] = struct{ cpwer, wordCPWER, wer, der float64 }{*m.CPWER, *m.WordRule.CPWER, *m.WER, *m.DER}
	}
	// The text is right in both chains; only the first attributes it to the right speakers, and only by the word rule.
	if rates["c1"].wer != 0 || rates["c2"].wer != 0 || rates["c1"].cpwer != 0.5 || rates["c1"].wordCPWER != 0 || rates["c2"].cpwer < 0.4 || rates["c2"].wordCPWER < 0.4 || rates["c1"].der > 0.02 || rates["c2"].der < 0.25 {
		t.Fatalf("the speaker-attributed rates see what the plain WER cannot: %v", rates)
	}

	// Both rules are measured on the same three tasks: the word rule asks the Worker for nothing more (#400).
	if n := h.count(`SELECT count(*) FROM jobs WHERE step LIKE 'benchmark.%'`); n != 3 {
		t.Fatalf("a chain is measured from its two stages' tasks only, got %d tasks", n)
	}

	// Nothing reached production, as for each stage on its own.
	if n := h.count(`SELECT count(*) FROM local_speakers WHERE clip_id=$1`, clip); n != 2 {
		t.Fatalf("local speakers are untouched, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE step LIKE 'transcription.audio.%' OR step LIKE 'voice.diarization.%'`); n != 3 {
		t.Fatalf("production jobs are untouched, got %d", n)
	}
}
