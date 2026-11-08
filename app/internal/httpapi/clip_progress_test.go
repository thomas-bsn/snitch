//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testClipProgress(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	progress := func() (string, string) {
		t.Helper()
		var clip struct {
			Progress struct {
				Transcription string `json:"transcription"`
				Indexing      string `json:"indexing"`
			} `json:"progress"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &clip); err != nil {
			t.Fatal(err)
		}
		return clip.Progress.Transcription, clip.Progress.Indexing
	}
	if tr, ix := progress(); tr != "none" || ix != "none" {
		t.Fatalf("a clip nothing was planned for: %s %s", tr, ix)
	}
	h.legacyTranscription(2)
	if tr, _ := progress(); tr != "queued" {
		t.Fatalf("a planned transcription must be queued: %s", tr)
	}
	job, ok := h.claim()
	if !ok {
		t.Fatal("no transcription task")
	}
	if tr, _ := progress(); tr != "running" {
		t.Fatalf("a claimed transcription must be running: %s", tr)
	}
	h.complete(job, map[string]any{"provider": "fixture", "model": "deterministic-v1", "version": "deterministic-v1", "language": "fr", "audio_stream_index": 2, "segments": []map[string]any{
		{"start": 0.2, "end": 0.9, "text": "premier", "words": []map[string]any{{"word": "premier", "start": 0.2, "end": 0.9}}},
	}}, http.StatusOK)
	if tr, ix := progress(); tr != "done" || ix != "unavailable" {
		t.Fatalf("transcribed without embeddings configured: %s %s", tr, ix)
	}

	var options struct {
		Sources []struct {
			Name string `json:"name"`
		} `json:"sources"`
		Owners []struct {
			Username string `json:"username"`
		} `json:"owners"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/clip-filters", nil, http.StatusOK), &options); err != nil {
		t.Fatal(err)
	}
	if len(options.Sources) != 1 || options.Sources[0].Name != "voice-source" || len(options.Owners) != 1 || options.Owners[0].Username != "administrator" {
		t.Fatalf("unexpected filter options: %+v", options)
	}
}
