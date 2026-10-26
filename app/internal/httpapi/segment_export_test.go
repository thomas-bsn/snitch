//go:build integration

package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const segmentExportPath = "/api/admin/segment-corrections/export"

type exportItem struct {
	ClipID             int64   `json:"clip_id"`
	TranscriptionJobID int64   `json:"transcription_job_id"`
	AudioStreamIndex   int     `json:"audio_stream_index"`
	Index              int     `json:"index"`
	Start              float64 `json:"start"`
	End                float64 `json:"end"`
	OriginalText       string  `json:"original_text"`
	CorrectedText      string  `json:"corrected_text"`
	Author             struct {
		Username string `json:"username"`
	} `json:"author"`
	CorrectedAt string `json:"corrected_at"`
	Version     int    `json:"version"`
}

type exportPage struct {
	Items    []exportItem `json:"items"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Total    int          `json:"total"`
}

func testSegmentExport(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.transcribe()
	admin := h.adminUser()
	bob := h.newUser("bob")
	anon := &voiceUser{name: "anonymous", client: &http.Client{}}
	export := func(query string) exportPage {
		t.Helper()
		var v exportPage
		if err := json.Unmarshal(h.userCall(admin, http.MethodGet, segmentExportPath+query, nil, http.StatusOK), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}

	// Reserved to administrators; nothing corrected yet means an empty export.
	if status, _ := h.userStatus(anon, http.MethodGet, segmentExportPath, nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous export: %d", status)
	}
	h.userCall(bob, http.MethodGet, segmentExportPath, nil, http.StatusForbidden)
	h.userCall(bob, http.MethodGet, segmentExportPath+"?format=ndjson", nil, http.StatusForbidden)
	if v := export(""); v.Total != 0 || len(v.Items) != 0 || v.Items == nil {
		t.Fatalf("an export without corrections is empty: %+v", v)
	}
	for _, bad := range []string{"?format=xml", "?page=0", "?page_size=0", "?page_size=501", "?page=x"} {
		h.userCall(admin, http.MethodGet, segmentExportPath+bad, nil, http.StatusBadRequest)
	}

	// Segment 1 is corrected twice, segment 0 once; segment 2 is never corrected.
	h.userCall(bob, http.MethodPut, segmentTextPath(h.clipID, 1), map[string]any{"text": "deuxième", "expected_version": 0}, http.StatusOK)
	h.userCall(admin, http.MethodPut, segmentTextPath(h.clipID, 1), map[string]any{"text": "deuxième bis", "expected_version": 1}, http.StatusOK)
	h.userCall(admin, http.MethodPut, segmentTextPath(h.clipID, 0), map[string]any{"text": "premier bis", "expected_version": 0}, http.StatusOK)
	v := export("")
	if v.Total != 2 || len(v.Items) != 2 || v.Page != 1 {
		t.Fatalf("two corrected segments expected: %+v", v)
	}
	first, second := v.Items[0], v.Items[1]
	if first.Index != 0 || first.ClipID != h.clipID || first.AudioStreamIndex != 2 || first.OriginalText != "premier" || first.CorrectedText != "premier bis" || first.Author.Username != "administrator" || first.Version != 1 || first.CorrectedAt == "" || first.TranscriptionJobID < 1 || first.Start != 0.2 || first.End != 0.9 {
		t.Fatalf("export item 0: %+v", first)
	}
	if second.Index != 1 || second.OriginalText != "second" || second.CorrectedText != "deuxième bis" || second.Author.Username != "administrator" || second.Version != 2 {
		t.Fatalf("export item 1 carries the original, the current text, author and version: %+v", second)
	}
	// A segment restored to the original is no longer corrected, hence not exported.
	h.userCall(admin, http.MethodPost, fmt.Sprintf("/api/clips/%d/segments/0/restore", h.clipID), map[string]any{"original": true, "expected_version": 1}, http.StatusOK)
	if v = export(""); v.Total != 1 || v.Items[0].Index != 1 {
		t.Fatalf("a segment restored to the original is not exported: %+v", v)
	}
	h.userCall(admin, http.MethodPut, segmentTextPath(h.clipID, 0), map[string]any{"text": "premier ter", "expected_version": 2}, http.StatusOK)

	// Pagination and the streamed NDJSON download.
	if v = export("?page_size=1&page=2"); v.Total != 2 || len(v.Items) != 1 || v.Items[0].Index != 1 || v.Page != 2 || v.PageSize != 1 {
		t.Fatalf("second page: %+v", v)
	}
	if v = export("?page_size=1&page=3"); v.Total != 2 || len(v.Items) != 0 {
		t.Fatalf("page past the end: %+v", v)
	}
	req, _ := http.NewRequest(http.MethodGet, h.ts.URL+segmentExportPath+"?format=ndjson", nil)
	resp, err := admin.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/x-ndjson") || !strings.Contains(resp.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("ndjson headers: %d %v", resp.StatusCode, resp.Header)
	}
	var lines []exportItem
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var item exportItem
		if err = json.Unmarshal(scanner.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, item)
	}
	if len(lines) != 2 || lines[0].Index != 0 || lines[0].CorrectedText != "premier ter" || lines[1].CorrectedText != "deuxième bis" {
		t.Fatalf("ndjson lines: %+v", lines)
	}
	if bytes.Contains(h.userCall(admin, http.MethodGet, segmentExportPath, nil, http.StatusOK), []byte("error_rate")) {
		t.Fatal("Snitch computes no error rate")
	}

	// A private clip of another user is still exported for an administrator.
	if _, err = pool.Exec(ctx, `UPDATE clips SET visibility='private',owner_id=(SELECT id FROM users WHERE username='bob') WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if v = export(""); v.Total != 2 {
		t.Fatalf("administrators see every clip: %+v", v)
	}

	// Deleting a Person or a voice profile never touches text corrections.
	events := h.count(`SELECT count(*) FROM segment_correction_events`)
	person := h.newPerson("Eve")
	h.userCall(admin, http.MethodDelete, fmt.Sprintf("/api/persons/%d/voice-profile", person), nil, http.StatusOK)
	h.userCall(admin, http.MethodDelete, personPath(person, ""), nil, http.StatusOK)
	if h.personExists(person) {
		t.Fatal("the Person must be gone")
	}
	if v = export(""); v.Total != 2 || v.Items[0].CorrectedText != "premier ter" || h.count(`SELECT count(*) FROM segment_corrections`) != 2 || h.count(`SELECT count(*) FROM segment_correction_events`) != events {
		t.Fatalf("person and profile deletions must keep the corrections and their history: %+v", v)
	}

	// Deleting the clip erases its corrections and history; the original media file
	// stays on disk.
	if _, err = pool.Exec(ctx, `DELETE FROM source_files WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM clips WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if v = export(""); v.Total != 0 || len(v.Items) != 0 {
		t.Fatalf("a deleted clip leaves no corrected segment in the export: %+v", v)
	}
	if n := h.count(`SELECT (SELECT count(*) FROM segment_corrections)+(SELECT count(*) FROM segment_correction_events)`); n != 0 {
		t.Fatalf("clip deletion must erase corrections and history, got %d", n)
	}
	if _, err = os.Stat(filepath.Join(h.api.Config.ClipRoots[0], "voice-clip.mp4")); err != nil {
		t.Fatalf("the original media must stay on disk: %v", err)
	}
}
