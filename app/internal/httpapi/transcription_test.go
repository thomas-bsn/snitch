//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"receipts/internal/config"
	"receipts/internal/setup"
)

func testTranscriptionPipeline(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	sourceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceRoot, "source-clip.mp4"), []byte("synthetic-source-multipiste-video"), 0o400); err != nil {
		t.Fatal(err)
	}
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	api := New(pool, config.Config{
		MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts",
		ProbePath: syntheticProbeFixture(t), ClipRoots: []string{sourceRoot}, StabilityWindow: 10 * time.Millisecond, UploadRoot: t.TempDir(), UploadMaxBytes: 1024,
		UploadChunkBytes: 1024, UploadSessionTTL: time.Hour,
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := api.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer api.Close()
	ts := httptest.NewUnstartedServer(api.Handler())
	api.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	admin := &http.Client{Jar: jar}
	call := func(client *http.Client, method, path string, body any, csrf string, want int) []byte {
		t.Helper()
		var payload []byte
		if body != nil {
			payload = mustJSON(t, body)
		}
		req, err := http.NewRequest(method, ts.URL+path, bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", ts.URL)
		req.Header.Set("Content-Type", "application/json")
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return data
	}
	call(admin, http.MethodPost, "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", http.StatusCreated)
	var session struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/auth/me", nil, "", http.StatusOK), &session); err != nil {
		t.Fatal(err)
	}
	content := []byte("synthetic-multipiste-video")
	var upload struct {
		ID string `json:"id"`
	}
	created := call(admin, http.MethodPost, "/api/uploads", map[string]any{"filename": "synthetic-multipiste.mp4", "size_bytes": len(content), "visibility": "shared"}, session.CSRF, http.StatusCreated)
	if err := json.Unmarshal(created, &upload); err != nil {
		t.Fatal(err)
	}
	chunkReq, err := http.NewRequest(http.MethodPut, ts.URL+"/api/uploads/"+upload.ID+"/chunks", bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	chunkReq.Header.Set("Origin", ts.URL)
	chunkReq.Header.Set("Content-Type", "application/octet-stream")
	chunkReq.Header.Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(content)-1, len(content)))
	chunkReq.Header.Set("X-CSRF-Token", session.CSRF)
	chunkResp, err := admin.Do(chunkReq)
	if err != nil {
		t.Fatal(err)
	}
	chunkResp.Body.Close()
	if chunkResp.StatusCode != http.StatusOK {
		t.Fatalf("upload chunk: got %d want %d", chunkResp.StatusCode, http.StatusOK)
	}
	var completed struct {
		ClipID int64 `json:"clip_id"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/uploads/"+upload.ID+"/complete", map[string]any{}, session.CSRF, http.StatusCreated), &completed); err != nil {
		t.Fatal(err)
	}
	if completed.ClipID < 1 {
		t.Fatal("upload did not create a clip")
	}
	var clip struct {
		Tracks []struct {
			Index         int    `json:"index"`
			Type          string `json:"type"`
			Title         string `json:"title"`
			Language      string `json:"language"`
			Channels      int    `json:"channels"`
			ChannelLayout string `json:"channel_layout"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips/"+strconv64(completed.ClipID), nil, "", http.StatusOK), &clip); err != nil {
		t.Fatal(err)
	}
	if len(clip.Tracks) != 3 || clip.Tracks[2].Index != 2 || clip.Tracks[2].Type != "audio" || clip.Tracks[2].Title != "Voix" || clip.Tracks[2].Language != "fra" || clip.Tracks[2].Channels != 2 || clip.Tracks[2].ChannelLayout != "stereo" {
		t.Fatalf("clip track indexes and metadata are not selectable: %+v", clip.Tracks)
	}
	var selection struct {
		AudioTrackSelection struct {
			Status           string `json:"status"`
			AudioStreamIndex *int   `json:"audio_stream_index"`
		} `json:"audio_track_selection"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips/"+strconv64(completed.ClipID), nil, "", http.StatusOK), &selection); err != nil {
		t.Fatal(err)
	}
	if selection.AudioTrackSelection.Status != "selection_required" {
		t.Fatalf("ambiguous multitrack clip must require selection: %+v", selection.AudioTrackSelection)
	}
	call(admin, http.MethodPost, "/api/admin/clips/"+strconv64(completed.ClipID)+"/transcription", map[string]any{}, session.CSRF, http.StatusConflict)
	call(admin, http.MethodPut, "/api/admin/clips/"+strconv64(completed.ClipID)+"/audio-track", map[string]int{"audio_stream_index": 2}, session.CSRF, http.StatusOK)
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips/"+strconv64(completed.ClipID), nil, "", http.StatusOK), &selection); err != nil {
		t.Fatal(err)
	}
	if selection.AudioTrackSelection.Status != "selected" || selection.AudioTrackSelection.AudioStreamIndex == nil || *selection.AudioTrackSelection.AudioStreamIndex != 2 {
		t.Fatalf("clip override was not exposed: %+v", selection.AudioTrackSelection)
	}
	var scheduled struct {
		Job struct {
			ID           int64  `json:"id"`
			Step         string `json:"step"`
			InputVersion string `json:"input_version"`
		} `json:"job"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/clips/"+strconv64(completed.ClipID)+"/transcription", map[string]int{"audio_stream_index": 2}, session.CSRF, http.StatusOK), &scheduled); err != nil {
		t.Fatal(err)
	}
	if scheduled.Job.ID < 1 || scheduled.Job.Step != "transcription.audio.2" || scheduled.Job.InputVersion == "" {
		t.Fatalf("selected audio track was not scheduled as a versioned transcription task: %+v", scheduled.Job)
	}
	call(admin, http.MethodPost, "/api/admin/workers", map[string]any{"name": "fixture-transcriber", "max_concurrency": 1}, session.CSRF, http.StatusCreated)
	var worker struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/workers", map[string]any{"name": "fixture-transcriber-2", "max_concurrency": 1}, session.CSRF, http.StatusCreated), &worker); err != nil {
		t.Fatal(err)
	}
	workerCall := func(method, path string, body any, lease string, want int) []byte {
		t.Helper()
		var payload []byte
		if body != nil {
			payload = mustJSON(t, body)
		}
		req, err := http.NewRequest(method, ts.URL+path, bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+worker.Token)
		req.Header.Set("Content-Type", "application/json")
		if lease != "" {
			req.Header.Set("X-Worker-Lease-Token", lease)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != want {
			t.Fatalf("worker %s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return data
	}
	workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"transcription.audio.*"}}, "", http.StatusOK)
	var assignment struct {
		Job struct {
			ID           int64  `json:"id"`
			InputVersion string `json:"input_version"`
		} `json:"job"`
		LeaseToken string `json:"lease_token"`
	}
	if err := json.Unmarshal(workerCall(http.MethodPost, "/api/worker/jobs/claim", nil, "", http.StatusOK), &assignment); err != nil {
		t.Fatal(err)
	}
	if assignment.Job.ID != scheduled.Job.ID || assignment.LeaseToken == "" {
		t.Fatalf("transcription was not leased to a capable worker: %+v", assignment)
	}
	workerCall(http.MethodGet, "/api/worker/jobs/"+strconv64(assignment.Job.ID)+"/input", nil, assignment.LeaseToken, http.StatusOK)
	const result = `{"provider":"fixture","model":"deterministic-v1","version":"deterministic-v1","language":"fr","audio_stream_index":2,"segments":[{"start":1.25,"end":2.1,"text":"Bien joué !","words":[{"word":"Bien","start":1.25,"end":1.52},{"word":"joué","start":1.55,"end":1.9}]}]}`
	var parsedResult any
	if err := json.Unmarshal([]byte(result), &parsedResult); err != nil {
		t.Fatal(err)
	}
	workerCall(http.MethodPost, "/api/worker/jobs/"+strconv64(assignment.Job.ID)+"/complete", map[string]any{"input_version": assignment.Job.InputVersion, "result": map[string]any{}}, assignment.LeaseToken, http.StatusBadRequest)
	workerCall(http.MethodPost, "/api/worker/jobs/"+strconv64(assignment.Job.ID)+"/complete", map[string]any{"input_version": assignment.Job.InputVersion, "result": parsedResult}, assignment.LeaseToken, http.StatusOK)
	var detail struct {
		Transcription struct {
			Provider         string `json:"provider"`
			Model            string `json:"model"`
			Version          string `json:"version"`
			AudioStreamIndex int    `json:"audio_stream_index"`
			Segments         []struct {
				Start float64 `json:"start"`
				Words []struct {
					Word  string  `json:"word"`
					Start float64 `json:"start"`
				} `json:"words"`
			} `json:"segments"`
		} `json:"transcription"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips/"+strconv64(completed.ClipID), nil, "", http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Transcription.Provider != "fixture" || detail.Transcription.Model != "deterministic-v1" || detail.Transcription.Version != "deterministic-v1" || detail.Transcription.AudioStreamIndex != 2 || len(detail.Transcription.Segments) != 1 || detail.Transcription.Segments[0].Start != 1.25 || detail.Transcription.Segments[0].Words[1].Word != "joué" || detail.Transcription.Segments[0].Words[1].Start != 1.55 {
		t.Fatalf("completed transcript was not available on the clip with word-level timecodes: %+v", detail.Transcription)
	}
	var emptyScheduled struct {
		Job struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/clips/"+strconv64(completed.ClipID)+"/transcription", map[string]int{"audio_stream_index": 2}, session.CSRF, http.StatusCreated), &emptyScheduled); err != nil {
		t.Fatal(err)
	}
	workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"transcription.audio.*"}}, "", http.StatusOK)
	if err := json.Unmarshal(workerCall(http.MethodPost, "/api/worker/jobs/claim", nil, "", http.StatusOK), &assignment); err != nil {
		t.Fatal(err)
	}
	if assignment.Job.ID != emptyScheduled.Job.ID {
		t.Fatalf("empty transcription task was not leased: %+v", assignment)
	}
	emptyResult := map[string]any{"provider": "fixture", "model": "deterministic-v1", "version": "deterministic-v1", "audio_stream_index": 2, "segments": []any{}}
	workerCall(http.MethodPost, "/api/worker/jobs/"+strconv64(assignment.Job.ID)+"/complete", map[string]any{"input_version": assignment.Job.InputVersion, "result": emptyResult}, assignment.LeaseToken, http.StatusOK)
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips/"+strconv64(completed.ClipID), nil, "", http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Transcription.Segments == nil || len(detail.Transcription.Segments) != 0 {
		t.Fatalf("empty transcript was not persisted as success: %+v", detail.Transcription)
	}
	var createdSource struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/sources", map[string]string{"name": "synthetic-multitrack-source", "path": sourceRoot}, session.CSRF, http.StatusCreated), &createdSource); err != nil {
		t.Fatal(err)
	}
	var sourceClipID int64
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var files struct {
			Items []struct {
				ClipID *int64 `json:"clip_id"`
				Status string `json:"status"`
			} `json:"items"`
		}
		if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/sources/"+strconv64(createdSource.ID)+"/files", nil, "", http.StatusOK), &files); err != nil {
			t.Fatal(err)
		}
		for _, file := range files.Items {
			if file.Status == "ready" && file.ClipID != nil {
				sourceClipID = *file.ClipID
			}
		}
		if sourceClipID > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if sourceClipID == 0 {
		t.Fatal("synthetic source clip was not discovered")
	}
	title := "Voix"
	var ruleResult struct {
		Version int64 `json:"version"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPut, "/api/admin/sources/"+strconv64(createdSource.ID)+"/audio-track-rule", map[string]string{"title": title}, session.CSRF, http.StatusOK), &ruleResult); err != nil {
		t.Fatal(err)
	}
	var sourceSelection struct {
		AudioTrackSelection struct {
			Status           string `json:"status"`
			AudioStreamIndex *int   `json:"audio_stream_index"`
			SelectionSource  string `json:"selection_source"`
			RuleVersion      int64  `json:"rule_version"`
		} `json:"audio_track_selection"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips/"+strconv64(sourceClipID), nil, "", http.StatusOK), &sourceSelection); err != nil {
		t.Fatal(err)
	}
	if ruleResult.Version != 1 || sourceSelection.AudioTrackSelection.Status != "selected" || sourceSelection.AudioTrackSelection.AudioStreamIndex == nil || *sourceSelection.AudioTrackSelection.AudioStreamIndex != 2 || sourceSelection.AudioTrackSelection.SelectionSource != "source_rule" || sourceSelection.AudioTrackSelection.RuleVersion != 1 {
		t.Fatalf("source rule did not resolve the synthetic track: %+v, version %d", sourceSelection.AudioTrackSelection, ruleResult.Version)
	}
	var sourceTask struct {
		Job struct {
			Step         string `json:"step"`
			InputVersion string `json:"input_version"`
		} `json:"job"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/clips/"+strconv64(sourceClipID)+"/transcription", map[string]any{}, session.CSRF, http.StatusOK), &sourceTask); err != nil {
		t.Fatal(err)
	}
	if sourceTask.Job.Step != "transcription.audio.2" || sourceTask.Job.InputVersion != "transcription-v2-audio-2-rule-1" {
		t.Fatalf("source policy version not included in targeted task: %+v", sourceTask.Job)
	}
	if err := json.Unmarshal(call(admin, http.MethodPut, "/api/admin/sources/"+strconv64(createdSource.ID)+"/audio-track-rule", map[string]string{"title": "Micro"}, session.CSRF, http.StatusOK), &ruleResult); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips/"+strconv64(sourceClipID), nil, "", http.StatusOK), &sourceSelection); err != nil {
		t.Fatal(err)
	}
	if sourceSelection.AudioTrackSelection.AudioStreamIndex == nil || *sourceSelection.AudioTrackSelection.AudioStreamIndex != 1 || sourceSelection.AudioTrackSelection.RuleVersion != 2 {
		t.Fatalf("updated source rule did not resolve its new track/version: %+v", sourceSelection.AudioTrackSelection)
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/clips/"+strconv64(sourceClipID)+"/transcription", map[string]any{}, session.CSRF, http.StatusOK), &sourceTask); err != nil {
		t.Fatal(err)
	}
	if sourceTask.Job.Step != "transcription.audio.1" || sourceTask.Job.InputVersion != "transcription-v2-audio-1-rule-2" {
		t.Fatalf("updated source rule did not create a targeted transcription version: %+v", sourceTask.Job)
	}
}
