//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type historyEvent struct {
	Event             string `json:"event"`
	Cycle             int    `json:"cycle"`
	Attempt           *int   `json:"attempt"`
	Actor             string `json:"actor"`
	WorkerName        string `json:"worker_name"`
	ErrorCode         string `json:"error_code"`
	Model             string `json:"model"`
	RecommendedAction string `json:"recommended_action"`
	CreatedAt         string `json:"created_at"`
}

type historyStep struct {
	Step              string         `json:"step"`
	Status            string         `json:"status"`
	FailureCode       string         `json:"failure_code"`
	RecommendedAction string         `json:"recommended_action"`
	Events            []historyEvent `json:"events"`
}

type historyEventsPage struct {
	Items []struct {
		ID     int64         `json:"id"`
		ClipID int64         `json:"clip_id"`
		Status string        `json:"status"`
		Steps  []historyStep `json:"steps"`
	} `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// The history of a run carries, for every step, the events of the job the step followed: the
// attempts with their Worker, the model and the persisted cause with its advised action; search
// and pagination keep working, only administrators read it, and no raw error text is exposed.
func testAdminQueueHistoryEvents(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := context.Background()
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusOK)
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{}, http.StatusOK)
	codes := []string{"whisper_network", "whisper_timeout", "whisper_model_missing"}
	for i, code := range codes {
		job := h.mustClaim("the transcription is not claimable")
		h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", job.ID), map[string]any{"error": code, "provider": "whisper", "model": "whisper-large-v3"}, job.Lease, http.StatusOK)
		if i < len(codes)-1 {
			if _, err := pool.Exec(ctx, `UPDATE jobs SET next_attempt_at=now() WHERE id=$1`, job.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := h.api.Analysis.ReconcileClip(ctx, h.clipID); err != nil {
		t.Fatal(err)
	}
	// Ten more finished runs push ours onto a second page.
	if _, err := pool.Exec(ctx, `WITH c AS (INSERT INTO clips(owner_id, original_name, size_bytes, modified_at, recorded_at)
 SELECT owner_id, 'frise-' || n, 1, now(), now() FROM clips, generate_series(1, 10) n WHERE id=$1 RETURNING id)
 INSERT INTO clip_analysis_runs(clip_id, scope, status, finished_at) SELECT id, 'full', 'completed', now() FROM c`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='failed', finished_at=now()-interval '1 hour' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	var clipName string
	if err := pool.QueryRow(ctx, `SELECT original_name FROM clips WHERE id=$1`, h.clipID).Scan(&clipName); err != nil {
		t.Fatal(err)
	}

	read := func(query string) (historyEventsPage, []byte) {
		t.Helper()
		raw := h.adminCall(http.MethodGet, "/api/admin/analysis/history"+query, nil, http.StatusOK)
		var page historyEventsPage
		if err := json.Unmarshal(raw, &page); err != nil {
			t.Fatal(err)
		}
		return page, raw
	}
	page, raw := read("?page=2")
	if page.Total != 11 || len(page.Items) != 1 || page.Items[0].ClipID != h.clipID || page.Page != 2 || page.PageSize != 10 {
		t.Fatalf("pagination keeps ten per page: %+v", page)
	}
	first, _ := read("")
	if len(first.Items) != 10 || first.Total != 11 {
		t.Fatalf("first page: %+v", first)
	}
	var failed historyStep
	for _, st := range page.Items[0].Steps {
		if st.Step == "transcription" {
			failed = st
		}
	}
	if failed.FailureCode != "whisper_model_missing" || failed.RecommendedAction != "install_transcription_model" {
		t.Fatalf("the failed step keeps its cause and action: %+v", failed)
	}
	var names []string
	for _, e := range failed.Events {
		names = append(names, e.Event)
		if e.CreatedAt == "" || e.Actor == "" {
			t.Fatalf("timestamped events with their actor: %+v", e)
		}
	}
	if got := strings.Join(names, ","); got != "attempt_started,retry_scheduled,attempt_started,retry_scheduled,attempt_started,marked_dead" {
		t.Fatalf("every event of the followed job, oldest first: %s", got)
	}
	last := failed.Events[len(failed.Events)-1]
	if last.ErrorCode != "whisper_model_missing" || last.Model != "whisper-large-v3" || last.WorkerName != "voice-fixture" || last.Attempt == nil || *last.Attempt != 3 || last.RecommendedAction != "install_transcription_model" {
		t.Fatalf("the last event carries cause, model, Worker and attempt: %+v", last)
	}
	if strings.Contains(string(raw), "last_error") {
		t.Fatalf("no raw error text: %s", raw)
	}

	// Search still narrows by clip name and keeps the events.
	searched, _ := read("?q=" + url.QueryEscape(clipName))
	if searched.Total != 1 || len(searched.Items[0].Steps) == 0 {
		t.Fatalf("search by clip name: %+v", searched)
	}

	// Administrators only.
	h.userCall(h.newUser("frise-member"), http.MethodGet, "/api/admin/analysis/history", nil, http.StatusForbidden)
	anon, err := http.Get(h.ts.URL + "/api/admin/analysis/history")
	if err != nil {
		t.Fatal(err)
	}
	anon.Body.Close()
	if anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous history: %d", anon.StatusCode)
	}
}
