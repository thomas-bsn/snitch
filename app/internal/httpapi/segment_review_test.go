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

// testSegmentReviewFixes covers the findings of the phase 6 code review.
func testSegmentReviewFixes(t *testing.T, pool *pgxpool.Pool, reset func(*testing.T)) {
	t.Run("a speaker correction during the initial indexing does not lose the vector", func(t *testing.T) {
		ctx := context.Background()
		reset(t)
		h := newVoiceHarness(t, pool)
		_, second := h.diarizeClip()
		provider := newReindexProvider(t)
		h.api.Config.EmbeddingsURL, h.api.Config.EmbeddingsModel = provider.srv.URL, "fixture-v1"
		if err := h.api.ensureEmbeddingProfile(ctx); err != nil {
			t.Fatal(err)
		}
		h.adminCall(http.MethodPost, "/api/admin/embeddings/backfill", map[string]any{}, http.StatusAccepted)
		admin := h.adminUser()
		status := 0
		provider.mu.Lock()
		provider.hook = func() {
			status, _ = h.userStatus(admin, http.MethodPut, fmt.Sprintf("/api/clips/%d/segments/0/speaker", h.clipID), map[string]any{"local_speaker_id": second, "expected_version": 0})
		}
		provider.mu.Unlock()
		if err := h.api.indexNext(ctx); err != nil {
			t.Fatal(err)
		}
		if status != http.StatusOK {
			t.Fatalf("the speaker correction made during the indexing must succeed: %d", status)
		}
		var active, job int64
		if err := pool.QueryRow(ctx, `SELECT active_profile_id FROM embedding_index_state WHERE id=1`).Scan(&active); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND status='done'`, h.clipID).Scan(&job); err != nil {
			t.Fatal(err)
		}
		if rows := embeddingRows(t, pool, active, job); len(rows) != 3 {
			t.Fatalf("a speaker correction does not change the text: every segment must be indexed, got %+v", rows)
		}
		if h.count(`SELECT count(*) FROM segment_corrections WHERE reindex_pending`) != 0 {
			t.Fatal("a speaker correction must not request a reindexing")
		}
	})

	t.Run("an original text with surrounding spaces is compared like the cleaned text", func(t *testing.T) {
		reset(t)
		h := newVoiceHarness(t, pool)
		h.transcribe()
		if _, err := pool.Exec(context.Background(), `UPDATE jobs SET result=jsonb_set(result,'{segments,0,text}','" premier "') WHERE clip_id=$1 AND step LIKE 'transcription.audio.%'`, h.clipID); err != nil {
			t.Fatal(err)
		}
		admin := h.adminUser()
		path := segmentTextPath(h.clipID, 0)
		var c correctionResponse
		if err := json.Unmarshal(h.userCall(admin, http.MethodPut, path, map[string]any{"text": "premier", "expected_version": 0}, http.StatusBadRequest), &c); err != nil || c.Error != "text_unchanged" {
			t.Fatalf("a correction that only changes the spaces is refused: %+v %v", c, err)
		}
		if h.count(`SELECT count(*) FROM segment_corrections`) != 0 || h.count(`SELECT count(*) FROM segment_correction_events`) != 0 {
			t.Fatal("a refused correction writes nothing")
		}
		h.userCall(admin, http.MethodPut, path, map[string]any{"text": "premier mot", "expected_version": 0}, http.StatusOK)
	})

	t.Run("corrections replaced by a retranscription stay readable and obsolete", func(t *testing.T) {
		reset(t)
		h := newVoiceHarness(t, pool)
		h.transcribe()
		admin := h.adminUser()
		bob := h.newUser("bob")
		outsider := h.newUser("outsider")
		obsolete := func(u *voiceUser, status int) (out struct {
			Items []struct {
				JobID        int64   `json:"transcription_job_id"`
				Index        int     `json:"index"`
				OriginalText string  `json:"original_text"`
				Text         *string `json:"text"`
				ObsoleteAt   string  `json:"obsolete_at"`
				History      []struct {
					Kind    string  `json:"kind"`
					NewText *string `json:"new_text"`
					Reason  string  `json:"reason"`
				} `json:"history"`
			} `json:"items"`
		}) {
			t.Helper()
			if err := json.Unmarshal(h.userCall(u, http.MethodGet, fmt.Sprintf("/api/clips/%d/obsolete-corrections", h.clipID), nil, status), &out); err != nil && status == http.StatusOK {
				t.Fatal(err)
			}
			return out
		}
		oldJob := h.clipTranscript(admin).Transcription.JobID
		if got := obsolete(admin, http.StatusOK); len(got.Items) != 0 {
			t.Fatalf("nothing is obsolete before a retranscription: %+v", got)
		}
		h.userCall(bob, http.MethodPut, segmentTextPath(h.clipID, 1), map[string]any{"text": "deuxième", "expected_version": 0, "reason": "orthographe"}, http.StatusOK)
		h.userCall(bob, http.MethodPut, segmentTextPath(h.clipID, 1), map[string]any{"text": "deuxième !", "expected_version": 1}, http.StatusOK)
		if got := obsolete(admin, http.StatusOK); len(got.Items) != 0 {
			t.Fatalf("an active correction is not obsolete: %+v", got)
		}
		h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/transcription?confirm_replace_corrections=true", h.clipID), map[string]any{}, http.StatusCreated)
		job, ok := h.claim()
		if !ok {
			t.Fatal("expected the retranscription task")
		}
		h.complete(job, h.retranscriptionResult("premier", "second", "ailleurs"), http.StatusOK)
		got := obsolete(bob, http.StatusOK)
		if len(got.Items) != 1 || got.Items[0].JobID != oldJob || got.Items[0].Index != 1 || got.Items[0].OriginalText != "second" || got.Items[0].Text == nil || *got.Items[0].Text != "deuxième !" || got.Items[0].ObsoleteAt == "" {
			t.Fatalf("the replaced correction must stay readable and marked obsolete: %+v", got)
		}
		if h := got.Items[0].History; len(h) != 2 || h[0].Kind != "text" || *h[0].NewText != "deuxième !" || h[1].Reason != "orthographe" {
			t.Fatalf("its history is complete, newest first: %+v", h)
		}
		var visible historyResponse
		if err := json.Unmarshal(h.userCall(bob, http.MethodGet, fmt.Sprintf("/api/clips/%d/segments/1/history", h.clipID), nil, http.StatusOK), &visible); err != nil || len(visible.Items) != 0 || visible.Current.Corrected {
			t.Fatalf("the visible segment starts clean: %+v %v", visible, err)
		}
		if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='private' WHERE id=$1`, h.clipID); err != nil {
			t.Fatal(err)
		}
		obsolete(outsider, http.StatusNotFound)
	})

	t.Run("deleting a Person keeps the speaker override and the reading", func(t *testing.T) {
		reset(t)
		h := newVoiceHarness(t, pool)
		first, second := h.diarizeClip()
		admin := h.adminUser()
		var bob struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(h.userCall(admin, http.MethodPost, "/api/persons", map[string]any{"display_name": "Bob"}, http.StatusCreated), &bob); err != nil {
			t.Fatal(err)
		}
		h.userCall(admin, http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", second), map[string]any{"person_id": bob.ID}, http.StatusCreated)
		h.userCall(admin, http.MethodPut, fmt.Sprintf("/api/clips/%d/segments/0/speaker", h.clipID), map[string]any{"local_speaker_id": second, "expected_version": 0}, http.StatusOK)
		h.userCall(admin, http.MethodDelete, fmt.Sprintf("/api/persons/%d", bob.ID), nil, http.StatusOK)
		read := func() speakerSegment {
			var v speakerTranscript
			if err := json.Unmarshal(h.userCall(admin, http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &v); err != nil {
				t.Fatal(err)
			}
			return v.Transcription.Segments[0]
		}
		if s := read(); s.SpeakerID != second || !s.Corrected || s.PersonName != "" {
			t.Fatalf("the override stays, the voice is simply no longer named: %+v", s)
		}
		// The Person is gone, so the diarization is no longer locked: a new diarization
		// replaces the Local speakers. The lost override is recorded in the history.
		h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2, "confirm_replace_validations": true}, http.StatusCreated)
		job, ok := h.claim()
		if !ok || job.Step != "voice.diarization.2" {
			t.Fatalf("expected a new diarization, got %+v", job)
		}
		h.complete(job, diarizationPayload("SPEAKER_00", "SPEAKER_01"), http.StatusOK)
		if s := read(); s.Corrected || s.SpeakerID == second || s.SpeakerID == first {
			t.Fatalf("the segment returns to its deduced speaker: %+v", s)
		}
		var history struct {
			Items []struct {
				Kind     string `json:"kind"`
				OldLabel string `json:"old_speaker_label"`
				Reason   string `json:"reason"`
				Author   struct {
					Username string `json:"username"`
				} `json:"author"`
			} `json:"items"`
		}
		if err := json.Unmarshal(h.userCall(admin, http.MethodGet, fmt.Sprintf("/api/clips/%d/segments/0/history", h.clipID), nil, http.StatusOK), &history); err != nil {
			t.Fatal(err)
		}
		if len(history.Items) != 2 || history.Items[0].Kind != "speaker" || history.Items[0].OldLabel != "SPEAKER_01" || history.Items[0].Reason == "" || history.Items[0].Author.Username != "system" {
			t.Fatalf("the removal of the override must leave a trace: %+v", history.Items)
		}
	})
}
