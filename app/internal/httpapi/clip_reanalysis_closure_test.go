//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testClipReanalysisClosure covers the closure of chosen steps on the server: a missing
// prerequisite and the dependent steps join the run, and the scope widens to hold them,
// whatever scope the request named.
func testClipReanalysisClosure(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.useSingleAudioTrack()
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusOK)
	type closedRun struct {
		analysisRunView
		Mode        string   `json:"mode"`
		ForcedSteps []string `json:"forced_steps"`
	}
	start := func(body map[string]any) closedRun {
		t.Helper()
		var out struct {
			Run closedRun `json:"run"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodPost, h.analysisPath(), body, http.StatusCreated), &out); err != nil {
			t.Fatal(err)
		}
		return out.Run
	}

	// Voice detection on a clip never transcribed: the transcription is added, not refused.
	run := start(map[string]any{"scope": "voice", "mode": "steps", "steps": []string{"diarization"}})
	if run.Scope != "full" || fmt.Sprint(run.ForcedSteps) != "[diarization]" || run.step("transcription").Status == "not_applicable" ||
		run.step("indexing").Status == "not_applicable" || run.step("fingerprints").Status == "not_applicable" || run.step("matching").Status == "not_applicable" {
		t.Fatalf("the prerequisite and the dependents must join the run: %+v", run)
	}
	job, ok := h.claim()
	if !ok || job.Step != "transcription.audio.2" {
		t.Fatalf("the missing transcription must be planned first, got %+v", job)
	}
	h.complete(job, analysisTranscript, http.StatusOK)
	h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusOK)

	// A retranscription asked within the transcription scope brings its indexing along.
	run = start(map[string]any{"scope": "transcription", "mode": "steps", "steps": []string{"transcription"}})
	if run.Scope != "full" || fmt.Sprint(run.ForcedSteps) != "[transcription]" || run.step("indexing").Status == "not_applicable" {
		t.Fatalf("the indexing must follow the retranscription: %+v", run)
	}
	h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusOK)

	// With a valid transcription, voices alone stay a voice run: nothing else is added.
	run = start(map[string]any{"scope": "voice", "mode": "steps", "steps": []string{"diarization"}})
	if run.Scope != "voice" || run.step("transcription").Reason != "reused_existing" || run.step("indexing").Status != "not_applicable" {
		t.Fatalf("a transcribed clip needs no prerequisite: %+v", run)
	}
}
