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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

func testTranscriptionPipeline(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var providerMu sync.Mutex
	failModels := map[string]bool{}
	candidateStarted := make(chan struct{}, 1)
	candidateRelease := make(chan struct{})
	embeddings := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
			return
		}
		providerMu.Lock()
		shouldFail := failModels[input.Model]
		providerMu.Unlock()
		if shouldFail {
			http.Error(w, "synthetic provider failure", http.StatusServiceUnavailable)
			return
		}
		if input.Model == "fixture-embedding-v2" {
			select {
			case candidateStarted <- struct{}{}:
			default:
			}
			<-candidateRelease
		}
		data := make([]map[string]any, len(input.Input))
		for i := range input.Input {
			data[i] = map[string]any{"embedding": []float32{1, 0.25, 0.5}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer embeddings.Close()
	sourceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceRoot, "source-clip.mp4"), []byte("synthetic-source-multipiste-video"), 0o400); err != nil {
		t.Fatal(err)
	}
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	api := New(pool, config.Config{
		MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Snitch",
		ProbePath: syntheticProbeFixture(t), ClipRoots: []string{sourceRoot}, StabilityWindow: 10 * time.Millisecond, UploadRoot: t.TempDir(), UploadMaxBytes: 1024,
		UploadChunkBytes: 1024, UploadSessionTTL: time.Hour, EmbeddingsURL: embeddings.URL, EmbeddingsModel: "fixture-embedding-v1",
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
	var vocabulary struct {
		Items []struct {
			ID                int64    `json:"id"`
			Term              string   `json:"term"`
			Variants          []string `json:"variants"`
			PreferredSpelling string   `json:"preferred_spelling"`
			Active            bool     `json:"active"`
		} `json:"items"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/vocabulary", nil, "", http.StatusOK), &vocabulary); err != nil {
		t.Fatal(err)
	}
	if vocabulary.Version == "" {
		t.Fatal("empty vocabulary must still have a stable version")
	}
	var pipelineState struct {
		Version string `json:"version"`
		Config  struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
			VAD      struct {
				Mode string `json:"mode"`
			} `json:"vad"`
		} `json:"config"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/transcription-pipeline", nil, "", http.StatusOK), &pipelineState); err != nil {
		t.Fatal(err)
	}
	if pipelineState.Version != "pipeline-v1" || pipelineState.Config.VAD.Mode != "off" {
		t.Fatalf("unexpected default pipeline: %+v", pipelineState)
	}
	updatedPipeline := map[string]any{"provider": "openai-compatible", "model": "test-profile", "language": "fr", "preparation": "wav-16khz-mono-v1", "vad": map[string]any{"mode": "energy", "version": "energy-v1", "threshold": 0.02, "min_speech_ms": 100, "min_silence_ms": 300, "padding_ms": 100}}
	var newPipeline struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPut, "/api/admin/transcription-pipeline", updatedPipeline, session.CSRF, http.StatusOK), &newPipeline); err != nil {
		t.Fatal(err)
	}
	if newPipeline.Version == "" || newPipeline.Version == pipelineState.Version {
		t.Fatalf("pipeline update did not create a new version: %+v", newPipeline)
	}
	call(admin, http.MethodPut, "/api/admin/transcription-pipeline", map[string]any{"provider": "openai-compatible", "model": "bad", "language": "fr", "preparation": "wav-16khz-mono-v1", "vad": map[string]any{"mode": "energy", "version": "energy-v1", "threshold": 2, "min_speech_ms": 100, "min_silence_ms": 300, "padding_ms": 100}}, session.CSRF, http.StatusBadRequest)
	var term struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/vocabulary", map[string]any{"term": "Final Fantasy XIV", "variants": []string{"FFXIV", "Final Fantasy 14"}, "preferred_spelling": "FFXIV", "active": true}, session.CSRF, http.StatusCreated), &term); err != nil {
		t.Fatal(err)
	}
	if term.ID < 1 {
		t.Fatal("vocabulary term was not created")
	}
	call(admin, http.MethodPost, "/api/admin/vocabulary", map[string]any{"term": " ", "variants": []string{}, "preferred_spelling": "Invalid", "active": true}, session.CSRF, http.StatusBadRequest)
	call(admin, http.MethodPost, "/api/admin/vocabulary", map[string]any{"term": "final fantasy xiv", "variants": []string{}, "preferred_spelling": "FFXIV", "active": true}, session.CSRF, http.StatusConflict)
	call(admin, http.MethodPut, "/api/admin/vocabulary/"+strconv64(term.ID), map[string]any{"term": "Final Fantasy XIV", "variants": []string{"FFXIV", "Final Fantasy 14"}, "preferred_spelling": "Final Fantasy XIV", "active": false}, session.CSRF, http.StatusOK)
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/vocabulary", nil, "", http.StatusOK), &vocabulary); err != nil {
		t.Fatal(err)
	}
	if len(vocabulary.Items) != 1 || vocabulary.Items[0].Active || vocabulary.Items[0].PreferredSpelling != "Final Fantasy XIV" {
		t.Fatalf("vocabulary edit was not persisted: %+v", vocabulary)
	}
	call(admin, http.MethodDelete, "/api/admin/vocabulary/"+strconv64(term.ID), nil, session.CSRF, http.StatusNoContent)
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
	ambiguousContent := []byte("second-synthetic-multipiste-video")
	var ambiguousUpload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/uploads", map[string]any{"filename": "synthetic-ambiguous.mp4", "size_bytes": len(ambiguousContent), "visibility": "shared"}, session.CSRF, http.StatusCreated), &ambiguousUpload); err != nil {
		t.Fatal(err)
	}
	ambiguousChunk, err := http.NewRequest(http.MethodPut, ts.URL+"/api/uploads/"+ambiguousUpload.ID+"/chunks", bytes.NewReader(ambiguousContent))
	if err != nil {
		t.Fatal(err)
	}
	ambiguousChunk.Header.Set("Origin", ts.URL)
	ambiguousChunk.Header.Set("Content-Type", "application/octet-stream")
	ambiguousChunk.Header.Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(ambiguousContent)-1, len(ambiguousContent)))
	ambiguousChunk.Header.Set("X-CSRF-Token", session.CSRF)
	ambiguousResponse, err := admin.Do(ambiguousChunk)
	if err != nil {
		t.Fatal(err)
	}
	ambiguousResponse.Body.Close()
	if ambiguousResponse.StatusCode != http.StatusOK {
		t.Fatalf("ambiguous upload chunk: got %d want 200", ambiguousResponse.StatusCode)
	}
	var ambiguousCompleted struct {
		ClipID int64 `json:"clip_id"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/uploads/"+ambiguousUpload.ID+"/complete", map[string]any{}, session.CSRF, http.StatusCreated), &ambiguousCompleted); err != nil {
		t.Fatal(err)
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
	var batch struct {
		Eligible          int `json:"eligible"`
		AlreadyQueued     int `json:"already_queued"`
		Unavailable       int `json:"unavailable"`
		SelectionRequired int `json:"selection_required"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/transcriptions/batch-preview", nil, "", http.StatusOK), &batch); err != nil {
		t.Fatal(err)
	}
	if batch.AlreadyQueued < 1 {
		t.Fatalf("preview did not count the resolved clip's existing task: %+v", batch)
	}
	call(admin, http.MethodPost, "/api/admin/transcriptions/batch-schedule", map[string]bool{"confirm": false}, session.CSRF, http.StatusBadRequest)
	var batchResult struct {
		Scheduled int `json:"scheduled"`
		Preview   struct {
			AlreadyQueued     int `json:"already_queued"`
			SelectionRequired int `json:"selection_required"`
		} `json:"preview"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/transcriptions/batch-schedule", map[string]bool{"confirm": true}, session.CSRF, http.StatusOK), &batchResult); err != nil {
		t.Fatal(err)
	}
	if batchResult.Scheduled != 0 || batchResult.Preview.AlreadyQueued < 1 || batchResult.Preview.SelectionRequired < 1 {
		t.Fatalf("confirmed batch duplicated an active task: %+v", batchResult)
	}
	var queuedPage struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/jobs?clip_id="+strconv64(completed.ClipID)+"&step=transcription.audio.2&status=pending", nil, "", http.StatusOK), &queuedPage); err != nil {
		t.Fatal(err)
	}
	if len(queuedPage.Items) != 1 {
		t.Fatalf("expected one queued synthetic task before race test, got %+v", queuedPage)
	}
	call(admin, http.MethodDelete, "/api/admin/jobs/"+strconv64(queuedPage.Items[0].ID), nil, session.CSRF, http.StatusNoContent)
	var group sync.WaitGroup
	results := make(chan []byte, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			results <- call(admin, http.MethodPost, "/api/admin/transcriptions/batch-schedule", map[string]bool{"confirm": true}, session.CSRF, http.StatusOK)
		}()
	}
	group.Wait()
	close(results)
	createdInRace := 0
	for body := range results {
		var result struct {
			Scheduled int `json:"scheduled"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal(err)
		}
		createdInRace += result.Scheduled
	}
	if createdInRace > 1 {
		t.Fatalf("concurrent batch requests created duplicate work: %d tasks", createdInRace)
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/jobs?clip_id="+strconv64(completed.ClipID)+"&step=transcription.audio.2&status=pending", nil, "", http.StatusOK), &queuedPage); err != nil {
		t.Fatal(err)
	}
	if len(queuedPage.Items) != 1 {
		t.Fatalf("concurrent batches must leave one active task: %+v", queuedPage)
	}
	var libraryPage struct {
		Items []struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips?page_size=100", nil, "", http.StatusOK), &libraryPage); err != nil {
		t.Fatal(err)
	}
	for _, clip := range libraryPage.Items {
		if clip.Status != "ready" {
			continue
		}
		var detail struct {
			AudioTrackSelection struct {
				Status string `json:"status"`
			} `json:"audio_track_selection"`
		}
		if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips/"+strconv64(clip.ID), nil, "", http.StatusOK), &detail); err != nil {
			t.Fatal(err)
		}
		if detail.AudioTrackSelection.Status != "selection_required" {
			continue
		}
		var page struct {
			Total int `json:"total"`
		}
		path := "/api/admin/jobs?clip_id=" + strconv64(clip.ID) + "&step=transcription.audio.2"
		if err := json.Unmarshal(call(admin, http.MethodGet, path, nil, "", http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		if page.Total != 0 {
			t.Fatalf("batch queued an ambiguous clip %d: %d task(s)", clip.ID, page.Total)
		}
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
	var leasedInput struct {
		PipelineVersion string `json:"pipeline_version"`
		Pipeline        struct {
			Model string `json:"model"`
			VAD   struct {
				Mode string `json:"mode"`
			} `json:"vad"`
		} `json:"pipeline"`
	}
	if err := json.Unmarshal(workerCall(http.MethodGet, "/api/worker/jobs/"+strconv64(assignment.Job.ID)+"/input", nil, assignment.LeaseToken, http.StatusOK), &leasedInput); err != nil {
		t.Fatal(err)
	}
	if leasedInput.PipelineVersion != newPipeline.Version || leasedInput.Pipeline.Model != "test-profile" || leasedInput.Pipeline.VAD.Mode != "energy" {
		t.Fatalf("worker did not receive its immutable pipeline snapshot: %+v", leasedInput)
	}
	const result = `{"provider":"fixture","model":"deterministic-v1","version":"deterministic-v1","language":"fr","audio_stream_index":2,"segments":[{"start":1.25,"end":2.1,"text":"Bien joué !","words":[{"word":"Bien","start":1.25,"end":1.52},{"word":"joué","start":1.55,"end":1.9}]}]}`
	var parsedResult any
	if err := json.Unmarshal([]byte(result), &parsedResult); err != nil {
		t.Fatal(err)
	}
	workerCall(http.MethodPost, "/api/worker/jobs/"+strconv64(assignment.Job.ID)+"/complete", map[string]any{"input_version": assignment.Job.InputVersion, "result": map[string]any{}}, assignment.LeaseToken, http.StatusBadRequest)
	workerCall(http.MethodPost, "/api/worker/jobs/"+strconv64(assignment.Job.ID)+"/complete", map[string]any{"input_version": assignment.Job.InputVersion, "result": parsedResult}, assignment.LeaseToken, http.StatusOK)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM transcript_embeddings WHERE clip_id=$1`, completed.ClipID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	var indexed int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM transcript_embeddings WHERE clip_id=$1`, completed.ClipID).Scan(&indexed); err != nil || indexed != 1 {
		var taskState, taskError string
		diagnosticErr := pool.QueryRow(context.Background(), `SELECT COALESCE(t.state,'missing'),COALESCE(t.error,'') FROM jobs j LEFT JOIN embedding_tasks t ON t.transcription_job_id=j.id WHERE j.id=$1`, assignment.Job.ID).Scan(&taskState, &taskError)
		t.Fatalf("completed transcript was not indexed: count=%d err=%v task_state=%q task_error=%q diagnostic_err=%v", indexed, err, taskState, taskError, diagnosticErr)
	}
	call(admin, http.MethodPost, "/api/admin/embeddings/rebuild", map[string]string{"model": "fixture-embedding-v2"}, session.CSRF, http.StatusAccepted)
	select {
	case <-candidateStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("candidate embeddings were not requested")
	}
	var oldProfileStatus struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/embeddings", nil, "", http.StatusOK), &oldProfileStatus); err != nil || oldProfileStatus.Model != "fixture-embedding-v1" {
		t.Fatalf("active profile changed before candidate completion: %+v err=%v", oldProfileStatus, err)
	}
	var availableSearch struct {
		Items    []searchItem `json:"items"`
		Degraded bool         `json:"degraded"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=concept+inattendu&mode=hybrid&page=1&page_size=10", nil, "", http.StatusOK), &availableSearch); err != nil || availableSearch.Degraded || len(availableSearch.Items) == 0 {
		t.Fatalf("old index was unavailable during candidate rebuild: %+v err=%v", availableSearch, err)
	}
	close(candidateRelease)
	rebuildDeadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(rebuildDeadline) {
		var status struct {
			Model          string `json:"model"`
			CandidateModel string `json:"candidate_model"`
		}
		if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/embeddings", nil, "", http.StatusOK), &status); err != nil {
			t.Fatal(err)
		}
		if status.Model == "fixture-embedding-v2" && status.CandidateModel == "" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	var activeModel string
	if err := pool.QueryRow(context.Background(), `SELECT p.model FROM embedding_index_state s JOIN embedding_profiles p ON p.id=s.active_profile_id WHERE s.id=1`).Scan(&activeModel); err != nil || activeModel != "fixture-embedding-v2" {
		t.Fatalf("candidate index did not atomically replace the active profile: model=%s err=%v", activeModel, err)
	}
	var semanticSearch struct {
		Items    []searchItem `json:"items"`
		Degraded bool         `json:"degraded"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=concept+inattendu&mode=hybrid&page=1&page_size=10", nil, "", http.StatusOK), &semanticSearch); err != nil {
		t.Fatal(err)
	}
	if semanticSearch.Degraded || len(semanticSearch.Items) == 0 || semanticSearch.Items[0].MatchType != "semantic" {
		t.Fatalf("hybrid mode did not return a semantic-only hit: %+v", semanticSearch)
	}
	if semanticSearch.Items[0].Timestamp != 1.25 {
		t.Fatalf("semantic hit lost its transcript timestamp: %+v", semanticSearch.Items[0])
	}
	var fusedSearch struct {
		Items []searchItem `json:"items"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=joué&mode=hybrid&page=1&page_size=10", nil, "", http.StatusOK), &fusedSearch); err != nil {
		t.Fatal(err)
	}
	if len(fusedSearch.Items) == 0 || fusedSearch.Items[0].MatchType != "hybrid" {
		t.Fatalf("lexical and semantic ranks were not fused: %+v", fusedSearch)
	}
	var filteredSearch struct {
		Items []searchItem `json:"items"`
		Total int          `json:"total"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=concept+inattendu&mode=hybrid&visibility=private&page=1&page_size=10", nil, "", http.StatusOK), &filteredSearch); err != nil {
		t.Fatal(err)
	}
	if filteredSearch.Total != 0 || len(filteredSearch.Items) != 0 {
		t.Fatalf("hybrid visibility filter leaked shared results: %+v", filteredSearch)
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=concept+inattendu&mode=hybrid&page=2&page_size=10", nil, "", http.StatusOK), &filteredSearch); err != nil {
		t.Fatal(err)
	}
	if filteredSearch.Total != 1 || len(filteredSearch.Items) != 0 {
		t.Fatalf("hybrid pagination did not preserve total and page: %+v", filteredSearch)
	}
	providerMu.Lock()
	failModels["fixture-embedding-v2"] = true
	providerMu.Unlock()
	var fallbackSearch struct {
		Degraded bool `json:"degraded"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=concept+inattendu&mode=hybrid&page=1&page_size=10", nil, "", http.StatusOK), &fallbackSearch); err != nil || !fallbackSearch.Degraded {
		t.Fatalf("provider failure did not expose lexical fallback: %+v err=%v", fallbackSearch, err)
	}
	providerMu.Lock()
	delete(failModels, "fixture-embedding-v2")
	failModels["fixture-embedding-failed"] = true
	providerMu.Unlock()
	call(admin, http.MethodPost, "/api/admin/embeddings/rebuild", map[string]string{"model": "fixture-embedding-failed"}, session.CSRF, http.StatusAccepted)
	failureDeadline := time.Now().Add(5 * time.Second)
	var failedCandidate string
	for time.Now().Before(failureDeadline) {
		var status struct {
			CandidateState string `json:"candidate_state"`
		}
		if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/embeddings", nil, "", http.StatusOK), &status); err != nil {
			t.Fatal(err)
		}
		failedCandidate = status.CandidateState
		if failedCandidate == "failed" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if failedCandidate != "failed" {
		t.Fatalf("failed candidate state not surfaced: %q", failedCandidate)
	}
	if err := pool.QueryRow(context.Background(), `SELECT p.model FROM embedding_index_state s JOIN embedding_profiles p ON p.id=s.active_profile_id WHERE s.id=1`).Scan(&activeModel); err != nil || activeModel != "fixture-embedding-v2" {
		t.Fatalf("failed rebuild replaced the working profile: model=%s err=%v", activeModel, err)
	}
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
	var search struct {
		Total int `json:"total"`
		Items []struct {
			MatchType   string  `json:"match_type"`
			ClipID      int64   `json:"clip_id"`
			Timestamp   float64 `json:"timestamp"`
			Context     string  `json:"context"`
			MatchedText string  `json:"matched_text"`
		} `json:"items"`
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=JOUE&mode=exact&page=1&page_size=10", nil, "", http.StatusOK), &search); err != nil {
		t.Fatal(err)
	}
	if search.Total != 1 || len(search.Items) != 1 || search.Items[0].ClipID != completed.ClipID || search.Items[0].Timestamp != 1.55 || search.Items[0].MatchType != "exact" || !strings.Contains(search.Items[0].Context, "joué") {
		t.Fatalf("accent/case tolerant exact search returned unexpected results: %+v", search)
	}
	// Approximate matching is a secondary tier. It recovers a nearby French
	// transcription typo while retaining the word-level timestamp and scope.
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=jouee&mode=full_text&page=1&page_size=10", nil, "", http.StatusOK), &search); err != nil {
		t.Fatal(err)
	}
	if search.Total != 1 || len(search.Items) != 1 || search.Items[0].ClipID != completed.ClipID || search.Items[0].Timestamp != 1.55 || search.Items[0].MatchType != "approximate" || search.Items[0].MatchedText != "joué" || !strings.Contains(search.Items[0].Context, "joué") {
		t.Fatalf("approximate French search returned unexpected result or lost its excerpt/timecode: %+v", search)
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=zzzqqq&mode=full_text&page=1&page_size=10", nil, "", http.StatusOK), &search); err != nil {
		t.Fatal(err)
	}
	if search.Total != 0 || len(search.Items) != 0 {
		t.Fatalf("unrelated query produced approximate matches: %+v", search)
	}
	// Keep all three result tiers in the same SQL response and assert their
	// precedence independently of clip dates and identifiers.
	_, err = pool.Exec(context.Background(), `UPDATE jobs SET result=$1::jsonb WHERE clip_id=$2 AND status='done' AND step LIKE 'transcription.audio.%'`, `{"segments":[{"start":1,"text":"Jouee !","words":[{"word":"Jouee","start":1}]},{"start":2,"text":"On dit jouee ici","words":[{"word":"ici","start":2}]},{"start":3,"text":"Joues !","words":[{"word":"Joues","start":3}]}]}`, completed.ClipID)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/search?q=jouee&mode=full_text&page=1&page_size=10", nil, "", http.StatusOK), &search); err != nil {
		t.Fatal(err)
	}
	if search.Total != 3 || len(search.Items) != 3 || search.Items[0].MatchType != "exact" || search.Items[1].MatchType != "full_text" || search.Items[2].MatchType != "approximate" || search.Items[0].Timestamp != 1 || search.Items[1].Timestamp != 2 || search.Items[2].Timestamp != 3 {
		t.Fatalf("search results did not rank exact, full-text, then approximate matches: %+v", search)
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/vocabulary", map[string]any{"term": "Final Fantasy XIV", "variants": []string{"FFXIV", "Final Fantasy 14"}, "preferred_spelling": "FFXIV", "active": true}, session.CSRF, http.StatusCreated), &term); err != nil {
		t.Fatal(err)
	}
	var vocabTask struct {
		Job struct {
			ID           int64  `json:"id"`
			InputVersion string `json:"input_version"`
		} `json:"job"`
	}
	if err := json.Unmarshal(call(admin, http.MethodPost, "/api/admin/clips/"+strconv64(completed.ClipID)+"/transcription", map[string]int{"audio_stream_index": 2}, session.CSRF, http.StatusCreated), &vocabTask); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(vocabTask.Job.InputVersion, "-v") || vocabTask.Job.InputVersion == scheduled.Job.InputVersion {
		t.Fatalf("vocabulary version missing from transcription input: %+v", vocabTask.Job)
	}
	call(admin, http.MethodPut, "/api/admin/vocabulary/"+strconv64(term.ID), map[string]any{"term": "Final Fantasy XIV", "variants": []string{"FFXIV", "Final Fantasy 14"}, "preferred_spelling": "Final Fantasy XIV", "active": true}, session.CSRF, http.StatusOK)
	workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"transcription.audio.*"}}, "", http.StatusOK)
	if err := json.Unmarshal(workerCall(http.MethodPost, "/api/worker/jobs/claim", nil, "", http.StatusOK), &assignment); err != nil {
		t.Fatal(err)
	}
	var workerInput struct {
		Vocabulary []struct {
			Term              string   `json:"term"`
			Variants          []string `json:"variants"`
			PreferredSpelling string   `json:"preferred_spelling"`
		} `json:"vocabulary"`
	}
	if err := json.Unmarshal(workerCall(http.MethodGet, "/api/worker/jobs/"+strconv64(vocabTask.Job.ID)+"/input", nil, assignment.LeaseToken, http.StatusOK), &workerInput); err != nil {
		t.Fatal(err)
	}
	if len(workerInput.Vocabulary) != 1 || workerInput.Vocabulary[0].PreferredSpelling != "FFXIV" || len(workerInput.Vocabulary[0].Variants) != 2 {
		t.Fatalf("active French vocabulary was not sent to the worker: %+v", workerInput.Vocabulary)
	}
	if err := json.Unmarshal(call(admin, http.MethodGet, "/api/clips/"+strconv64(completed.ClipID), nil, "", http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Transcription.Provider != "fixture" {
		t.Fatalf("previous successful transcript disappeared before replacement succeeded: %+v", detail.Transcription)
	}
	workerCall(http.MethodPost, "/api/worker/jobs/"+strconv64(vocabTask.Job.ID)+"/complete", map[string]any{"input_version": assignment.Job.InputVersion, "result": map[string]any{"provider": "fixture", "model": "deterministic-v1", "version": "deterministic-v1", "audio_stream_index": 2, "segments": []any{}}}, assignment.LeaseToken, http.StatusOK)
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
	sourceDeadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(sourceDeadline) {
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
	if sourceTask.Job.Step != "transcription.audio.2" || !strings.HasPrefix(sourceTask.Job.InputVersion, "t4-a2-r1-v") || !strings.HasSuffix(sourceTask.Job.InputVersion, "-p"+newPipeline.Version) {
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
	if sourceTask.Job.Step != "transcription.audio.1" || !strings.HasPrefix(sourceTask.Job.InputVersion, "t4-a1-r2-v") || !strings.HasSuffix(sourceTask.Job.InputVersion, "-p"+newPipeline.Version) {
		t.Fatalf("updated source rule did not create a targeted transcription version: %+v", sourceTask.Job)
	}
}
