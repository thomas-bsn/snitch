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

type historyResponse struct {
	TranscriptionJobID int64  `json:"transcription_job_id"`
	OriginalText       string `json:"original_text"`
	Current            struct {
		Version   int    `json:"version"`
		Text      string `json:"text"`
		Corrected bool   `json:"corrected"`
	} `json:"current"`
	Items []struct {
		ID                 int64   `json:"id"`
		Kind               string  `json:"kind"`
		OldText            *string `json:"old_text"`
		NewText            *string `json:"new_text"`
		OldCorrected       bool    `json:"old_corrected"`
		NewCorrected       bool    `json:"new_corrected"`
		RestoredFromEvent  *int64  `json:"restored_from_event_id"`
		RestoredToOriginal bool    `json:"restored_to_original"`
		Reason             string  `json:"reason"`
		VersionBefore      int     `json:"version_before"`
		VersionAfter       int     `json:"version_after"`
		Author             struct {
			Username string `json:"username"`
		} `json:"author"`
		CreatedAt string `json:"created_at"`
	} `json:"items"`
}

func testSegmentHistory(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.transcribe()
	admin := h.adminUser()
	bob := h.newUser("bob")
	path := segmentTextPath(h.clipID, 1)
	historyPath := fmt.Sprintf("/api/clips/%d/segments/1/history", h.clipID)
	restorePath := fmt.Sprintf("/api/clips/%d/segments/1/restore", h.clipID)
	history := func(u *voiceUser) historyResponse {
		t.Helper()
		var v historyResponse
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, historyPath, nil, http.StatusOK), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	restore := func(u *voiceUser, body map[string]any, status int) correctionResponse {
		t.Helper()
		var v correctionResponse
		if err := json.Unmarshal(h.userCall(u, http.MethodPost, restorePath, body, status), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}

	// An uncorrected segment has an empty history.
	if v := history(admin); len(v.Items) != 0 || v.Current.Version != 0 || v.OriginalText != "second" || v.Current.Corrected {
		t.Fatalf("empty history: %+v", v)
	}
	h.userCall(admin, http.MethodPut, path, map[string]any{"text": "deuxième", "expected_version": 0, "reason": "orthographe"}, http.StatusOK)
	h.userCall(bob, http.MethodPut, path, map[string]any{"text": "deuxième !", "expected_version": 1}, http.StatusOK)
	v := history(bob)
	if len(v.Items) != 2 || v.Current.Version != 2 || v.Current.Text != "deuxième !" {
		t.Fatalf("history: %+v", v)
	}
	first := v.Items[1] // newest first
	if v.Items[0].Author.Username != "bob" || first.Author.Username != "administrator" || first.Reason != "orthographe" ||
		*first.OldText != "second" || *first.NewText != "deuxième" || first.OldCorrected || !first.NewCorrected || first.CreatedAt == "" || v.Items[0].ID < first.ID {
		t.Fatalf("history entries: %+v", v.Items)
	}

	// Visibility follows the clip; unknown segments are refused.
	anon := &voiceUser{name: "anonymous", client: &http.Client{}}
	if status, _ := h.userStatus(anon, http.MethodGet, historyPath, nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous history: %d", status)
	}
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	h.userCall(bob, http.MethodGet, historyPath, nil, http.StatusNotFound)
	h.userCall(bob, http.MethodPost, restorePath, map[string]any{"original": true, "expected_version": 2}, http.StatusNotFound)
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	h.userCall(admin, http.MethodGet, fmt.Sprintf("/api/clips/%d/segments/9/history", h.clipID), nil, http.StatusNotFound)
	h.userCall(admin, http.MethodGet, fmt.Sprintf("/api/clips/%d/segments/x/history", h.clipID), nil, http.StatusBadRequest)

	// Refused restorations write nothing.
	events := func() int { return h.count(`SELECT count(*) FROM segment_correction_events`) }
	before := events()
	h.userCall(admin, http.MethodPost, restorePath, map[string]any{"expected_version": 2}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPost, restorePath, map[string]any{"original": true, "event_id": first.ID, "expected_version": 2}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPost, restorePath, map[string]any{"original": true}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPost, restorePath, map[string]any{"original": true, "expected_version": 2, "reason": strings.Repeat("r", 501)}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPost, restorePath, map[string]any{"event_id": 999999, "expected_version": 2}, http.StatusNotFound)
	h.userCall(admin, http.MethodPost, fmt.Sprintf("/api/clips/%d/segments/0/restore", h.clipID), map[string]any{"event_id": first.ID, "expected_version": 0}, http.StatusNotFound) // entry of another segment
	if c := restore(admin, map[string]any{"original": true, "expected_version": 1}, http.StatusConflict); c.Error != "segment_conflict" || c.Current == nil || c.Current.Version != 2 {
		t.Fatalf("stale restoration: %+v", c)
	}
	if c := restore(admin, map[string]any{"original": true, "expected_version": 2, "transcription_job_id": v.TranscriptionJobID + 1000}, http.StatusConflict); c.Error != "transcription_changed" {
		t.Fatalf("stale transcription: %+v", c)
	}
	h.userCall(admin, http.MethodPost, restorePath, map[string]any{"event_id": v.Items[0].ID, "expected_version": 2}, http.StatusBadRequest) // already the current text
	if events() != before {
		t.Fatal("refused restorations must not write history")
	}

	// Restoring a past version is a new correction.
	r := restore(admin, map[string]any{"event_id": first.ID, "expected_version": 2, "reason": "retour"}, http.StatusOK)
	if r.Version != 3 || r.Text != "deuxième" {
		t.Fatalf("restoration of a past version: %+v", r)
	}
	// Restoring the original clears the correction and the reading shows the original.
	r = restore(bob, map[string]any{"original": true, "expected_version": 3}, http.StatusOK)
	if r.Version != 4 || r.Text != "second" {
		t.Fatalf("restoration of the original: %+v", r)
	}
	if got := h.clipTranscript(admin).Transcription.Segments[1]; got.Corrected || got.Text != "second" || got.Version != 4 || got.CorrectedBy != "" {
		t.Fatalf("segment after restoration: %+v", got)
	}
	// Nothing left to restore, then a new correction works from version 4.
	h.userCall(admin, http.MethodPost, restorePath, map[string]any{"original": true, "expected_version": 4}, http.StatusBadRequest)
	h.userCall(admin, http.MethodPut, path, map[string]any{"text": "encore", "expected_version": 4}, http.StatusOK)
	// An entry restoring the original can itself be restored (back to the original text).
	v = history(admin)
	if len(v.Items) != 5 {
		t.Fatalf("history after restorations: %+v", v)
	}
	toOriginal := v.Items[1]
	if toOriginal.Kind != "restore" || !toOriginal.RestoredToOriginal || toOriginal.NewCorrected || !toOriginal.OldCorrected || *toOriginal.NewText != "second" || toOriginal.VersionBefore != 3 || toOriginal.VersionAfter != 4 || toOriginal.Author.Username != "bob" {
		t.Fatalf("restore to original entry: %+v", toOriginal)
	}
	pastVersion := v.Items[2]
	if pastVersion.Kind != "restore" || pastVersion.RestoredFromEvent == nil || *pastVersion.RestoredFromEvent != first.ID || pastVersion.RestoredToOriginal || pastVersion.Reason != "retour" || *pastVersion.OldText != "deuxième !" || *pastVersion.NewText != "deuxième" {
		t.Fatalf("restore of a past version entry: %+v", pastVersion)
	}
	r = restore(admin, map[string]any{"event_id": toOriginal.ID, "expected_version": 5}, http.StatusOK)
	if r.Version != 6 || r.Text != "second" {
		t.Fatalf("restoring a restoration of the original: %+v", r)
	}

	// Past entries are never rewritten; each restoration marked the segment for reindexing.
	var pending bool
	var corrected *string
	if err := pool.QueryRow(ctx, `SELECT reindex_pending,text FROM segment_corrections WHERE transcription_job_id=$1 AND segment_index=1`, v.TranscriptionJobID).Scan(&pending, &corrected); err != nil || !pending || corrected != nil {
		t.Fatalf("restored state: %v %v %v", pending, corrected, err)
	}
	if after := history(admin); len(after.Items) != 6 || after.Items[5].ID != first.ID || *after.Items[5].NewText != "deuxième" || after.Items[5].Reason != "orthographe" {
		t.Fatalf("history must only grow: %+v", after)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM segment_correction_events WHERE id=$1`, first.ID); err == nil {
		t.Fatal("history entries must not be deletable")
	}
}
