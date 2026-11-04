//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// #273 end to end: the diarization detects three voices, the transcript (three segments, the
// last one with no voice) attributes words to two. The third voice must stay visible with the
// reason, and the trace must expose the raw steps without any transcript text.
func testVoiceCoverage(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.transcribe() // segments 0.2-0.9, 1.25-2.1, 3.9-4.1 on track 2
	h.putPipeline(fakePipeline(3))
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("expected the diarization task, got %+v", job)
	}
	turn := func(start, end float64) map[string]any { return map[string]any{"start": start, "end": end} }
	h.complete(job, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "audio_stream_index": 2, "speakers": []map[string]any{
		{"label": "SPEAKER_00", "turns": []map[string]any{turn(0, 1)}},
		{"label": "SPEAKER_01", "turns": []map[string]any{turn(1.2, 2.2)}},
		// Talks over SPEAKER_00 (outweighed), then again after the transcript ended.
		{"label": "SPEAKER_02", "turns": []map[string]any{turn(0.3, 0.5), turn(4.5, 5)}},
	}}, http.StatusOK)

	var detail struct {
		Transcription struct {
			Segments []struct {
				Label *string `json:"local_speaker_label"`
			} `json:"segments"`
			Coverage []struct {
				ID       int64   `json:"local_speaker_id"`
				Status   string  `json:"coverage_status"`
				Overlap  float64 `json:"overlapped_seconds"`
				Uncover  float64 `json:"uncovered_seconds"`
				Segments int     `json:"attributed_segments"`
			} `json:"speaker_coverage"`
		} `json:"transcription"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	labels := map[int64]string{}
	for _, s := range h.voice().Speakers {
		labels[s.ID] = s.Label
	}
	got := map[string]int{}
	for _, c := range detail.Transcription.Coverage {
		got[labels[c.ID]] = c.Segments
		switch labels[c.ID] {
		case "SPEAKER_00", "SPEAKER_01":
			if c.Status != "words" || c.Segments != 1 {
				t.Errorf("%s: %+v", labels[c.ID], c)
			}
		case "SPEAKER_02":
			if c.Status != "speech_without_words" || math.Abs(c.Overlap-0.2) > 1e-6 || math.Abs(c.Uncover-0.5) > 1e-6 {
				t.Errorf("the outweighed voice must stay visible with its reason: %+v", c)
			}
		}
	}
	if len(got) != 3 {
		t.Fatalf("three voices were detected, coverage lists %v", got)
	}
	segs := detail.Transcription.Segments
	if segs[0].Label == nil || *segs[0].Label != "SPEAKER_00" || segs[1].Label == nil || *segs[1].Label != "SPEAKER_01" || segs[2].Label != nil {
		t.Fatalf("segment attribution changed: %+v", segs)
	}

	// The trace: raw steps before and after alignment, model versions, no transcript text.
	body := h.adminCall(http.MethodGet, fmt.Sprintf("/api/admin/clips/%d/voice-trace", h.clipID), nil, http.StatusOK)
	for _, secret := range []string{`"premier"`, `"second"`, `"ailleurs"`} {
		if strings.Contains(string(body), secret) {
			t.Fatalf("the trace leaked transcript text %q", secret)
		}
	}
	var trace struct {
		AudioStreamIndex int `json:"audio_stream_index"`
		Transcription    struct {
			Model string `json:"model"`
		} `json:"transcription"`
		Diarization struct {
			Model        string `json:"model"`
			ModelVersion string `json:"model_version"`
		} `json:"diarization"`
		Turns    []struct{ Label string } `json:"turns"`
		Segments []struct {
			Index      int `json:"index"`
			Candidates []struct {
				ID      int64   `json:"local_speaker_id"`
				Overlap float64 `json:"overlap_seconds"`
			} `json:"candidates"`
			Speaker *int64 `json:"local_speaker_id"`
		} `json:"segments"`
	}
	if err := json.Unmarshal(body, &trace); err != nil {
		t.Fatal(err)
	}
	if trace.AudioStreamIndex != 2 || trace.Transcription.Model != "deterministic-v1" || trace.Diarization.Model != "fixture-diarization" || trace.Diarization.ModelVersion != "fixture-v1" {
		t.Errorf("trace lost the track or the model versions: %+v", trace)
	}
	if len(trace.Turns) != 4 || len(trace.Segments) != 3 {
		t.Fatalf("raw turns and segments expected (4, 3): %d, %d", len(trace.Turns), len(trace.Segments))
	}
	if len(trace.Segments[0].Candidates) != 2 || trace.Segments[0].Speaker == nil || len(trace.Segments[2].Candidates) != 0 || trace.Segments[2].Speaker != nil {
		t.Errorf("candidates before alignment and winner after: %+v", trace.Segments)
	}
}
