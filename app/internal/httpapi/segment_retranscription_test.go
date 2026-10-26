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

type correctionsWarning struct {
	Error       string `json:"error"`
	Corrections struct {
		TranscriptionJobID int64    `json:"transcription_job_id"`
		TextSegments       int      `json:"text_segments"`
		SpeakerSegments    int      `json:"speaker_segments"`
		Authors            []string `json:"authors"`
		LastCorrectedAt    string   `json:"last_corrected_at"`
	} `json:"corrections"`
}

type batchCounts struct {
	Eligible          int       `json:"eligible"`
	Corrected         int       `json:"corrected"`
	CorrectedSegments int       `json:"corrected_segments"`
	Scheduled         int       `json:"scheduled"`
	AlreadyProcessed  int       `json:"already_processed"`
	Preview           *struct { // batch-schedule wraps its preview
		Eligible  int `json:"eligible"`
		Corrected int `json:"corrected"`
	} `json:"preview"`
}

func (h *voiceHarness) jobCount(clipID int64, statuses string) int {
	return h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND status IN (`+statuses+`)`, clipID)
}

func (h *voiceHarness) retranscriptionResult(texts ...string) map[string]any {
	segments := []map[string]any{}
	for i, text := range texts {
		segments = append(segments, map[string]any{"start": float64(i), "end": float64(i) + 0.8, "text": text, "words": []map[string]any{{"word": text, "start": float64(i), "end": float64(i) + 0.8}}})
	}
	return map[string]any{"provider": "fixture", "model": "deterministic-v2", "version": "deterministic-v2", "language": "fr", "audio_stream_index": 2, "segments": segments}
}

func testSegmentRetranscription(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.transcribe()
	admin := h.adminUser()
	bob := h.newUser("bob")
	clip := fmt.Sprintf("/api/admin/clips/%d", h.clipID)
	oldJob := h.clipTranscript(admin).Transcription.JobID

	// An uncorrected clip is retranscribed without confirmation (phase 3 behavior).
	// Seed the human correction afterwards.
	h.userCall(bob, http.MethodPut, segmentTextPath(h.clipID, 1), map[string]any{"text": "deuxième", "expected_version": 0, "reason": "orthographe"}, http.StatusOK)
	events := h.count(`SELECT count(*) FROM segment_correction_events`)

	// 1. Retranscribing a corrected clip needs an explicit confirmation naming what is replaced.
	var warning correctionsWarning
	if err := json.Unmarshal(h.adminCall(http.MethodPost, clip+"/transcription", map[string]any{}, http.StatusConflict), &warning); err != nil {
		t.Fatal(err)
	}
	if warning.Error != "corrections_would_be_replaced" || warning.Corrections.TextSegments != 1 || warning.Corrections.TranscriptionJobID != oldJob || len(warning.Corrections.Authors) != 1 || warning.Corrections.Authors[0] != "bob" || warning.Corrections.LastCorrectedAt == "" {
		t.Fatalf("the warning must say what is replaced: %+v", warning)
	}
	if n := h.jobCount(h.clipID, `'pending','leased','retry_wait'`); n != 0 {
		t.Fatalf("a refused retranscription must schedule nothing, got %d tasks", n)
	}
	h.adminCall(http.MethodPost, clip+"/transcription?confirm_replace_corrections=false", map[string]any{}, http.StatusConflict)

	// 3. Automatic and generic paths refuse too, with the same explicit code.
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, clip + "/audio-track", map[string]any{"audio_stream_index": 2}},
		{http.MethodDelete, clip + "/audio-track", nil},
		{http.MethodPost, "/api/admin/jobs", map[string]any{"clip_id": h.clipID, "step": "transcription.audio.2", "input_version": "manual-1"}},
	} {
		var w correctionsWarning
		if err := json.Unmarshal(h.adminCall(c.method, c.path, c.body, http.StatusConflict), &w); err != nil || w.Error != "corrections_would_be_replaced" {
			t.Fatalf("%s %s must refuse: %+v %v", c.method, c.path, w, err)
		}
	}
	if n := h.jobCount(h.clipID, `'pending','leased','retry_wait'`); n != 0 {
		t.Fatalf("refused paths must schedule nothing, got %d tasks", n)
	}
	// A job of another step is not a retranscription.
	h.adminCall(http.MethodPost, "/api/admin/jobs", map[string]any{"clip_id": h.clipID, "step": "other.step", "input_version": "manual-1"}, http.StatusCreated)

	// A source rule change reprocesses without erasing: the corrected clip is skipped and counted.
	var rule struct {
		Skipped int `json:"corrected_clips_skipped"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/sources/%d/audio-track-rule", h.sourceID()), map[string]any{"fallback_audio_stream_index": 2}, http.StatusOK), &rule); err != nil {
		t.Fatal(err)
	}
	if rule.Skipped != 1 || h.jobCount(h.clipID, `'pending','leased','retry_wait'`) != 0 {
		t.Fatalf("a rule change must skip the corrected clip: %+v", rule)
	}

	// 5. A failed retranscription keeps the current text and the corrections.
	h.adminCall(http.MethodPost, clip+"/transcription?confirm_replace_corrections=true", map[string]any{}, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "transcription.audio.2" {
		t.Fatalf("expected the retranscription task, got %+v", job)
	}
	h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", job.ID), map[string]any{"error": "provider_unavailable"}, job.Lease, http.StatusOK)
	during := h.clipTranscript(admin)
	if during.Transcription.JobID != oldJob || during.Transcription.Segments[1].Text != "deuxième" || !during.Transcription.Segments[1].Corrected {
		t.Fatalf("a failed retranscription must keep the text and the corrections: %+v", during.Transcription)
	}
	if n := h.count(`SELECT count(*) FROM segment_corrections WHERE transcription_job_id=$1 AND stale_at IS NULL AND text IS NOT NULL`, oldJob); n != 1 {
		t.Fatalf("the correction must stay active after a failure, got %d", n)
	}
	// Relaunching a dead retranscription needs the confirmation as well.
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET status='dead' WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/jobs/%d/retry", job.ID), nil, http.StatusConflict)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/jobs/%d/retry?confirm_replace_corrections=true", job.ID), nil, http.StatusOK)

	// 4. Success: the visible text changes, the old corrections stay in the history, obsolete.
	job, ok = h.claim()
	if !ok || job.ID == 0 {
		t.Fatal("expected the relaunched retranscription task")
	}
	h.complete(job, h.retranscriptionResult("premier", "second", "ailleurs"), http.StatusOK)
	after := h.clipTranscript(admin)
	if after.Transcription.JobID != job.ID || after.Transcription.Segments[1].Text != "second" || after.Transcription.Segments[1].Corrected || after.Transcription.Segments[1].Version != 0 {
		t.Fatalf("the new transcription must not carry the old correction: %+v", after.Transcription)
	}
	if n := h.count(`SELECT count(*) FROM segment_corrections WHERE transcription_job_id=$1 AND stale_at IS NOT NULL AND text='deuxième'`, oldJob); n != 1 {
		t.Fatalf("the replaced correction must be kept and marked obsolete, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM segment_correction_events`); n != events {
		t.Fatalf("the history must be kept untouched: %d -> %d", events, n)
	}
	// The obsolete correction is never reapplied, even to an identical segment text.
	h.userCall(bob, http.MethodPut, segmentTextPath(h.clipID, 1), map[string]any{"text": "deuxième", "expected_version": 0}, http.StatusOK)

	// The clip carries a fresh correction again, so the batch must protect it.
	h.adminCall(http.MethodPost, "/api/admin/vocabulary", map[string]any{"term": "Synthetic term", "variants": []string{"synthetic variant"}, "preferred_spelling": "Synthetic term", "active": true}, http.StatusCreated)
	var preview batchCounts
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/transcriptions/batch-preview", nil, http.StatusOK), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Eligible != 0 || preview.Corrected != 1 || preview.CorrectedSegments != 1 {
		t.Fatalf("the preview must exclude and report the corrected clip: %+v", preview)
	}
	var excluded batchCounts
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/transcriptions/batch-schedule", map[string]any{"confirm": true}, http.StatusOK), &excluded); err != nil {
		t.Fatal(err)
	}
	if excluded.Scheduled != 0 || h.jobCount(h.clipID, `'pending','leased','retry_wait'`) != 0 {
		t.Fatalf("the batch must not retranscribe a corrected clip by default: %+v", excluded)
	}
	var included batchCounts
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/transcriptions/batch-preview?include_corrected=true", nil, http.StatusOK), &included); err != nil {
		t.Fatal(err)
	}
	if included.Eligible != 1 || included.Corrected != 1 {
		t.Fatalf("an explicit request includes the corrected clip and still reports it: %+v", included)
	}
	var scheduled batchCounts
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/transcriptions/batch-schedule", map[string]any{"confirm": true, "include_corrected": true}, http.StatusOK), &scheduled); err != nil {
		t.Fatal(err)
	}
	if scheduled.Scheduled != 1 || h.jobCount(h.clipID, `'pending','leased','retry_wait'`) != 1 {
		t.Fatalf("an explicit request schedules the corrected clip: %+v", scheduled)
	}
}
