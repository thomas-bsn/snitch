//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type segmentView struct {
	Index        int     `json:"index"`
	Start        float64 `json:"start"`
	End          float64 `json:"end"`
	Text         string  `json:"text"`
	OriginalText string  `json:"original_text"`
	Corrected    bool    `json:"corrected"`
	Version      int     `json:"correction_version"`
	CorrectedBy  string  `json:"corrected_by"`
	CorrectedAt  string  `json:"corrected_at"`
	Words        []struct {
		Word  string  `json:"word"`
		Start float64 `json:"start"`
	} `json:"words"`
}

type clipTranscript struct {
	Transcription struct {
		JobID    int64         `json:"transcription_job_id"`
		Segments []segmentView `json:"segments"`
	} `json:"transcription"`
}

func (h *voiceHarness) clipTranscript(u *voiceUser) clipTranscript {
	h.t.Helper()
	var v clipTranscript
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

func segmentTextPath(clipID int64, index int) string {
	return fmt.Sprintf("/api/clips/%d/segments/%d/text", clipID, index)
}

type correctionResponse struct {
	Error   string `json:"error"`
	Version int    `json:"version"`
	Text    string `json:"text"`
	Current *struct {
		Version   int    `json:"version"`
		Text      string `json:"text"`
		Corrected bool   `json:"corrected"`
		Author    *struct {
			Username string `json:"username"`
		} `json:"author"`
	} `json:"current"`
}

func testSegmentCorrections(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.transcribe()
	admin := h.adminUser()
	before := h.clipTranscript(admin)
	if len(before.Transcription.Segments) != 3 || before.Transcription.JobID < 1 {
		t.Fatalf("the clip must expose its transcription job and indexed segments: %+v", before)
	}
	for i, s := range before.Transcription.Segments {
		if s.Index != i || s.Corrected || s.Version != 0 || s.Text != s.OriginalText || s.OriginalText == "" {
			t.Fatalf("an uncorrected segment shows the original text at version 0: %+v", s)
		}
	}
	path := segmentTextPath(h.clipID, 1)
	body := map[string]any{"text": "deuxième", "expected_version": 0}

	// Anonymous callers and invisible clips are refused.
	anon := &voiceUser{name: "anonymous", client: &http.Client{}}
	if status, _ := h.userStatus(anon, http.MethodPut, path, body); status != http.StatusUnauthorized {
		t.Fatalf("anonymous correction: %d", status)
	}
	bob := h.newUser("bob")
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	h.userCall(bob, http.MethodPut, path, body, http.StatusNotFound)
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}

	// Refused inputs change nothing.
	h.userCall(admin, http.MethodPut, path, map[string]any{"text": "  ", "expected_version": 0}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPut, path, map[string]any{"text": strings.Repeat("a", 4001), "expected_version": 0}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPut, path, map[string]any{"text": "ok"}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPut, path, map[string]any{"text": "ok", "expected_version": -1}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPut, path, map[string]any{"text": "ok", "expected_version": 0, "reason": strings.Repeat("r", 501)}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPut, path, map[string]any{"text": "second", "expected_version": 0}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPut, segmentTextPath(h.clipID, 9), body, http.StatusNotFound)
	h.userCall(admin, http.MethodPut, "/api/clips/999999/segments/0/text", body, http.StatusNotFound)
	h.userCall(admin, http.MethodPut, "/api/clips/"+strconv64(h.clipID)+"/segments/x/text", body, http.StatusBadRequest)
	if n := h.count(`SELECT count(*) FROM segment_corrections`); n != 0 {
		t.Fatalf("refused corrections must write nothing, got %d rows", n)
	}

	// A correction by a regular user who sees the clip overlays the text only.
	var first correctionResponse
	if err := json.Unmarshal(h.userCall(bob, http.MethodPut, path, map[string]any{"text": " deuxième ", "expected_version": 0, "reason": "orthographe"}, http.StatusOK), &first); err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 || first.Text != "deuxième" {
		t.Fatalf("first correction: %+v", first)
	}
	after := h.clipTranscript(admin)
	got, orig := after.Transcription.Segments[1], before.Transcription.Segments[1]
	if !got.Corrected || got.Text != "deuxième" || got.OriginalText != "second" || got.Version != 1 || got.CorrectedBy != "bob" || got.CorrectedAt == "" {
		t.Fatalf("corrected segment: %+v", got)
	}
	if got.Start != orig.Start || got.End != orig.End || len(got.Words) != 1 || got.Words[0].Word != "second" || got.Words[0].Start != orig.Words[0].Start {
		t.Fatalf("bounds and original words must not change: %+v", got)
	}
	if after.Transcription.Segments[0].Corrected || after.Transcription.Segments[2].Corrected {
		t.Fatal("other segments must stay uncorrected")
	}
	var stored string
	if err := pool.QueryRow(ctx, `SELECT result->'segments'->1->>'text' FROM jobs WHERE id=$1`, after.Transcription.JobID).Scan(&stored); err != nil || stored != "second" {
		t.Fatalf("the worker transcription must stay intact: %q %v", stored, err)
	}

	// Optimistic version check: the stale writer is refused with the current version.
	var conflict correctionResponse
	if err := json.Unmarshal(h.userCall(admin, http.MethodPut, path, map[string]any{"text": "autre", "expected_version": 0}, http.StatusConflict), &conflict); err != nil {
		t.Fatal(err)
	}
	if conflict.Error != "segment_conflict" || conflict.Current == nil || conflict.Current.Version != 1 || conflict.Current.Text != "deuxième" || conflict.Current.Author == nil || conflict.Current.Author.Username != "bob" {
		t.Fatalf("conflict must carry the current version and author: %+v", conflict)
	}
	var second correctionResponse
	if err := json.Unmarshal(h.userCall(admin, http.MethodPut, path, map[string]any{"text": "autre", "expected_version": 1}, http.StatusOK), &second); err != nil || second.Version != 2 {
		t.Fatalf("second correction: %+v %v", second, err)
	}
	h.userCall(admin, http.MethodPut, path, map[string]any{"text": "autre", "expected_version": 2}, http.StatusBadRequest)
	// Another segment is independent and a stale transcription id is refused.
	h.userCall(admin, http.MethodPut, segmentTextPath(h.clipID, 0), map[string]any{"text": "premier bis", "expected_version": 0}, http.StatusOK)
	h.userCall(admin, http.MethodPut, segmentTextPath(h.clipID, 2), map[string]any{"text": "ailleurs bis", "expected_version": 0, "transcription_job_id": after.Transcription.JobID + 1000}, http.StatusConflict)
	if after = h.clipTranscript(admin); after.Transcription.Segments[1].Text != "autre" || after.Transcription.Segments[0].Text != "premier bis" || after.Transcription.Segments[2].Corrected {
		t.Fatalf("segments after corrections: %+v", after.Transcription.Segments)
	}

	// The history keeps old and new values, author and reason; it is immutable.
	rows, err := pool.Query(ctx, `SELECT kind,old_text,new_text,old_corrected,new_corrected,reason,version_before,version_after,author_username FROM segment_correction_events WHERE transcription_job_id=$1 AND segment_index=1 ORDER BY id`, after.Transcription.JobID)
	if err != nil {
		t.Fatal(err)
	}
	var trail []string
	for rows.Next() {
		var kind, oldText, newText, reason, author string
		var oldC, newC bool
		var vb, va int
		if err = rows.Scan(&kind, &oldText, &newText, &oldC, &newC, &reason, &vb, &va, &author); err != nil {
			t.Fatal(err)
		}
		trail = append(trail, fmt.Sprint(kind, "|", oldText, "|", newText, "|", oldC, "|", newC, "|", reason, "|", vb, "|", va, "|", author))
	}
	rows.Close()
	want := []string{"text|second|deuxième|false|true|orthographe|0|1|bob", "text|deuxième|autre|true|true||1|2|administrator"}
	if strings.Join(trail, "\n") != strings.Join(want, "\n") {
		t.Fatalf("history: %q", trail)
	}
	if _, err = pool.Exec(ctx, `UPDATE segment_correction_events SET new_text='x'`); err == nil {
		t.Fatal("history entries must not be updatable")
	}
	if _, err = pool.Exec(ctx, `DELETE FROM segment_correction_events`); err == nil {
		t.Fatal("history entries must not be deletable")
	}

	// The shared extension point yields the same effective text as the clip reading.
	effective, err := pool.Query(ctx, `WITH `+visibleTranscriptsCTE("")+`, `+effectiveSegmentsCTE()+` SELECT idx,text,original_text,corrected FROM eff_segments WHERE clip_id=$1 ORDER BY idx`, h.clipID)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for effective.Next() {
		var idx int
		var text, original string
		var corrected bool
		if err = effective.Scan(&idx, &text, &original, &corrected); err != nil {
			t.Fatal(err)
		}
		if s := after.Transcription.Segments[idx]; s.Text != text || s.OriginalText != original || s.Corrected != corrected {
			t.Fatalf("effective segment %d differs from the clip reading: %q %q %v vs %+v", idx, text, original, corrected, s)
		}
		seen++
	}
	effective.Close()
	if seen != 3 {
		t.Fatalf("effective segments: %d", seen)
	}

	// Deleting the clip erases corrections and history through the cascade.
	if _, err = pool.Exec(ctx, `DELETE FROM source_files WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM clips WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if n := h.count(`SELECT (SELECT count(*) FROM segment_corrections)+(SELECT count(*) FROM segment_correction_events)`); n != 0 {
		t.Fatalf("clip deletion must erase corrections and history, got %d", n)
	}
}
