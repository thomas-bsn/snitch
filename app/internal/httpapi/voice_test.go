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
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/jobs"
	"snitch/internal/setup"
)

type voiceHarness struct {
	engineContract routers.Router
	t              *testing.T
	api            *Server
	pool           *pgxpool.Pool
	ts             *httptest.Server
	admin          *http.Client
	csrf           string
	token          string
	clipID         int64
	// clipIDs lists every synthetic clip (clipID is the first); see newVoiceHarnessClips.
	clipIDs []int64
	// ffmpegLog records the arguments of each synthetic excerpt cut.
	ffmpegLog string
}

func (h *voiceHarness) adminCall(method, path string, body any, want int) []byte {
	h.t.Helper()
	var payload []byte
	if body != nil {
		payload = mustJSON(h.t, body)
	}
	req, _ := http.NewRequest(method, h.ts.URL+path, bytes.NewReader(payload))
	req.Header.Set("Origin", h.ts.URL)
	req.Header.Set("Content-Type", "application/json")
	if h.csrf != "" {
		req.Header.Set("X-CSRF-Token", h.csrf)
	}
	return h.do(h.admin, req, want)
}

func (h *voiceHarness) workerCall(method, path string, body any, lease string, want int) []byte {
	h.t.Helper()
	var payload []byte
	if body != nil {
		payload = mustJSON(h.t, body)
	}
	req, _ := http.NewRequest(method, h.ts.URL+path, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Content-Type", "application/json")
	if lease != "" {
		req.Header.Set("X-Worker-Lease-Token", lease)
	}
	return h.do(http.DefaultClient, req, want)
}

func (h *voiceHarness) do(client *http.Client, req *http.Request, want int) []byte {
	h.t.Helper()
	resp, err := client.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		h.t.Fatalf("%s %s: got %d want %d: %s", req.Method, req.URL.Path, resp.StatusCode, want, data)
	}
	if strings.Contains(req.URL.Path, "/voice-engine") || req.URL.Path == "/api/admin/voice-model-runtime-orders" {
		if h.engineContract == nil {
			spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
			if err != nil {
				h.t.Fatal(err)
			}
			if err = spec.Validate(context.Background()); err != nil {
				h.t.Fatal(err)
			}
			spec.Servers = nil
			h.engineContract, err = legacy.NewRouter(spec)
			if err != nil {
				h.t.Fatal(err)
			}
		}
		route, params, err := h.engineContract.FindRoute(req)
		if err != nil {
			h.t.Fatal(err)
		}
		input := &openapi3filter.RequestValidationInput{Request: req, Route: route, PathParams: params, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
		if want < 400 && req.GetBody != nil {
			req.Body, err = req.GetBody()
			if err != nil {
				h.t.Fatal(err)
			}
			if err = openapi3filter.ValidateRequest(context.Background(), input); err != nil {
				h.t.Fatalf("engine request contract %s %s: %v", req.Method, req.URL.Path, err)
			}
		}
		if err = openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: resp.StatusCode, Header: resp.Header, Body: io.NopCloser(bytes.NewReader(data))}); err != nil {
			h.t.Fatalf("engine response contract %s %s: %v", req.Method, req.URL.Path, err)
		}
	}
	return data
}

type claimedJob struct {
	ID           int64  `json:"id"`
	Step         string `json:"step"`
	InputVersion string `json:"input_version"`
	Lease        string `json:"-"`
}

// claim leases the next voice or transcription task for the simulated worker.
func (h *voiceHarness) claim() (claimedJob, bool) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"transcription.audio.*", "voice.*"}}, "", http.StatusOK)
	var a struct {
		Job        *claimedJob `json:"job"`
		LeaseToken string      `json:"lease_token"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodPost, "/api/worker/jobs/claim", nil, "", http.StatusOK), &a); err != nil {
		h.t.Fatal(err)
	}
	if a.Job == nil {
		return claimedJob{}, false
	}
	a.Job.Lease = a.LeaseToken
	return *a.Job, true
}

func (h *voiceHarness) complete(job claimedJob, result any, want int) []byte {
	h.t.Helper()
	return h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/complete", job.ID), map[string]any{"input_version": job.InputVersion, "result": result}, job.Lease, want)
}

type voiceView struct {
	Status string `json:"status"`
	Steps  []struct {
		Step      string `json:"step"`
		Status    string `json:"status"`
		LastError string `json:"last_error"`
	} `json:"steps"`
	History  []voiceHistoryEntry `json:"history"`
	Speakers []struct {
		ID            int64  `json:"id"`
		Label         string `json:"label"`
		State         string `json:"state"`
		DecisionLevel string `json:"decision_level"`
		Person        *struct {
			ID          int64  `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"person"`
		Attribution *struct {
			ValidationState string `json:"validation_state"`
			Author          *struct {
				Username string `json:"username"`
			} `json:"author"`
		} `json:"attribution"`
		SpeakingSecond float64 `json:"speaking_seconds"`
		Fingerprints   []struct {
			Scope         string  `json:"scope"`
			Duration      float64 `json:"duration_seconds"`
			Quality       float64 `json:"quality"`
			Usable        bool    `json:"usable"`
			UnusableCause string  `json:"unusable_reason"`
			Model         string  `json:"model"`
			ModelVersion  string  `json:"model_version"`
		} `json:"fingerprints"`
	} `json:"speakers"`
}

func (h *voiceHarness) voice() voiceView {
	h.t.Helper()
	var v voiceView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", h.clipID), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

func newVoiceHarness(t *testing.T, pool *pgxpool.Pool) *voiceHarness {
	t.Helper()
	return newVoiceHarnessClips(t, pool, 1)
}

// newVoiceHarnessClips discovers several synthetic clips so cross-clip behavior
// (matching) can be exercised; clipID is the first one found.
func newVoiceHarnessClips(t *testing.T, pool *pgxpool.Pool, clips int) *voiceHarness {
	t.Helper()
	return newVoiceHarnessLogged(t, pool, clips, slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

// newVoiceHarnessLogged is newVoiceHarnessClips with a logger the test can capture.
func newVoiceHarnessLogged(t *testing.T, pool *pgxpool.Pool, clips int, logger *slog.Logger) *voiceHarness {
	t.Helper()
	sourceRoot := t.TempDir()
	for i := 0; i < clips; i++ {
		name := "voice-clip.mp4"
		if i > 0 {
			name = fmt.Sprintf("voice-clip-%d.mp4", i+1)
		}
		if err := os.WriteFile(filepath.Join(sourceRoot, name), []byte(fmt.Sprintf("synthetic-voice-video-%d", i)), 0o400); err != nil {
			t.Fatal(err)
		}
	}
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	ffmpegPath, ffmpegLog := syntheticFFmpegFixture(t)
	api := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), ProbePath: syntheticProbeFixture(t), FFmpegPath: ffmpegPath, ClipRoots: []string{sourceRoot}, StabilityWindow: 10 * time.Millisecond, UploadRoot: t.TempDir(), UploadMaxBytes: 1024, UploadChunkBytes: 1024, UploadSessionTTL: time.Hour}, logger)
	if err = api.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(api.Close)
	ts := httptest.NewUnstartedServer(api.Handler())
	ts.Start()
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	h := &voiceHarness{t: t, api: api, pool: pool, ts: ts, admin: &http.Client{Jar: jar}, ffmpegLog: ffmpegLog}
	h.adminCall(http.MethodPost, "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, http.StatusCreated)
	var session struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(h.adminCall(http.MethodGet, "/api/auth/me", nil, http.StatusOK), &session); err != nil {
		t.Fatal(err)
	}
	h.csrf = session.CSRF
	var worker struct {
		Token string `json:"token"`
	}
	if err = json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/workers", map[string]any{"name": "voice-fixture", "max_concurrency": 2}, http.StatusCreated), &worker); err != nil {
		t.Fatal(err)
	}
	h.token = worker.Token
	var source struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/sources", map[string]string{"name": "voice-source", "path": sourceRoot}, http.StatusCreated), &source); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for len(h.clipIDs) < clips && time.Now().Before(deadline) {
		h.clipIDs = h.clipIDs[:0]
		var files struct {
			Items []struct {
				ClipID *int64 `json:"clip_id"`
				Status string `json:"status"`
			} `json:"items"`
		}
		if err = json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/sources/"+strconv64(source.ID)+"/files", nil, http.StatusOK), &files); err != nil {
			t.Fatal(err)
		}
		for _, f := range files.Items {
			if f.Status == "ready" && f.ClipID != nil {
				h.clipIDs = append(h.clipIDs, *f.ClipID)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(h.clipIDs) < clips {
		t.Fatal("synthetic clips were not discovered")
	}
	sort.Slice(h.clipIDs, func(i, j int) bool { return h.clipIDs[i] < h.clipIDs[j] })
	h.clipID = h.clipIDs[0]
	return h
}

// useSingleAudioTrack keeps only audio track 2 on every synthetic clip: a clip with one audio track is planned by
// the legacy single-track scheduler, which the tests of that scheduler exercise (a clip with several audio tracks
// is planned by an audio plan, ADR 0054).
func (h *voiceHarness) useSingleAudioTrack() {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), `UPDATE clips SET tracks=COALESCE((SELECT jsonb_agg(t ORDER BY ord) FROM jsonb_array_elements(tracks) WITH ORDINALITY e(t,ord) WHERE t->>'type'<>'audio' OR (t->>'index')::int=2),'[]'::jsonb) WHERE id=ANY($1)`, h.clipIDs); err != nil {
		h.t.Fatal(err)
	}
}

// legacyTranscription is a clip whose single-track transcription was made before the audio plans existed (the
// state of every clip transcribed with a track rule or an override): the override is recorded and the legacy job is
// scheduled directly. The audio plan pipeline never produces it again for a clip with several audio tracks.
func (h *voiceHarness) legacyTranscription(index int) {
	h.t.Helper()
	ctx := context.Background()
	if _, err := h.pool.Exec(ctx, `INSERT INTO clip_audio_track_overrides(clip_id,audio_stream_index) VALUES($1,$2) ON CONFLICT(clip_id) DO UPDATE SET audio_stream_index=EXCLUDED.audio_stream_index,version=clip_audio_track_overrides.version+1,updated_at=now()`, h.clipID, index); err != nil {
		h.t.Fatal(err)
	}
	plan, err := h.api.planTranscription(ctx, h.clipID, &index, true)
	if err != nil {
		h.t.Fatal(err)
	}
	if _, _, err = h.api.Jobs.Schedule(ctx, jobs.ScheduleInput{ClipID: h.clipID, Step: plan.step, InputVersion: plan.version, MaxAttempts: 3}); err != nil {
		h.t.Fatal(err)
	}
}

func (h *voiceHarness) transcribe() {
	h.t.Helper()
	h.legacyTranscription(2)
	job, ok := h.claim()
	if !ok || job.Step != "transcription.audio.2" {
		h.t.Fatalf("expected the transcription task, got %+v", job)
	}
	h.complete(job, map[string]any{"provider": "fixture", "model": "deterministic-v1", "version": "deterministic-v1", "language": "fr", "audio_stream_index": 2, "segments": []map[string]any{
		{"start": 0.2, "end": 0.9, "text": "premier", "words": []map[string]any{{"word": "premier", "start": 0.2, "end": 0.9}}},
		{"start": 1.25, "end": 2.1, "text": "second", "words": []map[string]any{{"word": "second", "start": 1.25, "end": 2.1}}},
		{"start": 3.9, "end": 4.1, "text": "ailleurs", "words": []map[string]any{{"word": "ailleurs", "start": 3.9, "end": 4.1}}},
	}}, http.StatusOK)
}

func (h *voiceHarness) putPipeline(config map[string]any) string {
	h.t.Helper()
	var state struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPut, "/api/admin/voice-pipeline", config, http.StatusOK), &state); err != nil {
		h.t.Fatal(err)
	}
	return state.Version
}

func fakePipeline(speakers int) map[string]any {
	return map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "min_fingerprint_seconds": 1.0, "min_quality": 0.5, "fake": map[string]any{"speakers": speakers, "dimensions": 4}}
}

func diarizationPayload(speakers ...string) map[string]any {
	items := []map[string]any{}
	for i, label := range speakers {
		items = append(items, map[string]any{"label": label, "turns": []map[string]any{{"start": float64(i), "end": float64(i) + 1.5}}})
	}
	return map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "audio_stream_index": 2, "speakers": items}
}

func testVoiceDiarization(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	voicePath := fmt.Sprintf("/api/clips/%d/voice", h.clipID)

	// Anonymous callers never read voice data.
	resp, err := http.Get(h.ts.URL + voicePath)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous voice read: %d", resp.StatusCode)
	}
	if v := h.voice(); v.Status != "none" || len(v.Speakers) != 0 {
		t.Fatalf("a clip never scheduled must have no voice state: %+v", v)
	}

	// Voice processing is explicit, needs a transcription, and never downloads a model implicitly.
	scheduleBody := map[string]any{"audio_stream_index": 2}
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), scheduleBody, http.StatusConflict)
	h.transcribe()
	var jobsBefore int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM jobs WHERE step LIKE 'voice.%'`).Scan(&jobsBefore)
	if jobsBefore != 0 {
		t.Fatal("voice tasks must never be scheduled implicitly")
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-pipeline", map[string]any{"provider": "unknown"}, http.StatusBadRequest)
	pipelineV1 := h.putPipeline(fakePipeline(2))
	var scheduled struct {
		Job struct {
			ID           int64  `json:"id"`
			Step         string `json:"step"`
			InputVersion string `json:"input_version"`
		} `json:"job"`
	}
	if err = json.Unmarshal(h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), scheduleBody, http.StatusCreated), &scheduled); err != nil {
		t.Fatal(err)
	}
	if scheduled.Job.Step != "voice.diarization.2" || scheduled.Job.InputVersion == "" {
		t.Fatalf("diarization must be a versioned queue step: %+v", scheduled.Job)
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), scheduleBody, http.StatusOK)
	if v := h.voice(); v.Status != "queued" {
		t.Fatalf("scheduled voice step must be visible as queued: %+v", v)
	}

	// The worker receives the model configuration through its leased input only.
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("worker did not receive the diarization task: %+v", job)
	}
	var input struct {
		Voice struct {
			PipelineVersion string `json:"pipeline_version"`
			Config          struct {
				Provider string `json:"provider"`
			} `json:"config"`
		} `json:"voice"`
	}
	if err = json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, http.StatusOK), &input); err != nil {
		t.Fatal(err)
	}
	if input.Voice.PipelineVersion != pipelineV1 || input.Voice.Config.Provider != "fake" {
		t.Fatalf("voice input did not carry the pinned configuration: %+v", input.Voice)
	}
	// Invalid or foreign results are refused without leaving partial voices; the lease stays usable.
	h.complete(job, map[string]any{"provider": "fake", "model": "other-model", "model_version": "fixture-v1", "audio_stream_index": 2, "speakers": []any{}}, http.StatusBadRequest)
	h.complete(job, diarizationPayload("A", "A"), http.StatusBadRequest)
	var speakerRows int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM local_speakers`).Scan(&speakerRows)
	if speakerRows != 0 {
		t.Fatal("a rejected result left partial speakers")
	}
	h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/complete", job.ID), map[string]any{"input_version": "d5-stale", "result": diarizationPayload("A")}, job.Lease, http.StatusConflict)
	withPeaks := diarizationPayload("SPEAKER_00", "SPEAKER_01")
	withPeaks["peaks"] = []int{0, 128, 255}
	h.complete(job, withPeaks, http.StatusOK)
	h.complete(job, withPeaks, http.StatusOK) // lost response retried
	var storedPeaks bool
	var storedWaveforms int
	if err := pool.QueryRow(context.Background(), `SELECT result ? 'peaks' FROM jobs WHERE id=$1`, job.ID).Scan(&storedPeaks); err != nil || storedPeaks {
		t.Fatalf("diarization result kept peaks in jobs.result: %v err=%v", storedPeaks, err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM clip_waveforms WHERE audio_stream_index=2 AND job_id=$1`, job.ID).Scan(&storedWaveforms); err != nil || storedWaveforms != 1 {
		t.Fatalf("diarization peaks were not stored for the track: %d err=%v", storedWaveforms, err)
	}

	view := h.voice()
	if len(view.Speakers) != 2 || view.Speakers[0].State != "unidentified" || view.Status != "queued" {
		t.Fatalf("diarization must expose unidentified local speakers while fingerprints are queued: %+v", view)
	}
	// Each transcript segment shows the local speaker that overlaps it most.
	var detail struct {
		Transcription struct {
			Segments []struct {
				Text         string  `json:"text"`
				SpeakerID    *int64  `json:"local_speaker_id"`
				SpeakerLabel *string `json:"local_speaker_label"`
			} `json:"segments"`
		} `json:"transcription"`
	}
	if err = json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	segments := detail.Transcription.Segments
	if len(segments) != 3 || segments[0].SpeakerLabel == nil || *segments[0].SpeakerLabel != "SPEAKER_00" || segments[1].SpeakerLabel == nil || *segments[1].SpeakerLabel != "SPEAKER_01" || segments[2].SpeakerLabel != nil {
		t.Fatalf("segments do not show their local speaker (unmatched segments stay unlabeled): %+v", segments)
	}

	// Fingerprints are queued automatically for this explicit run only.
	fp, ok := h.claim()
	if !ok || fp.Step != "voice.fingerprint.2" {
		t.Fatalf("fingerprint step was not scheduled by the diarization result: %+v", fp)
	}
	if err = json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", fp.ID), nil, fp.Lease, http.StatusOK), &struct{}{}); err != nil {
		t.Fatal(err)
	}
	fingerprints := func(diarizationJobID int64, items ...map[string]any) map[string]any {
		return map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": 2, "diarization_job_id": diarizationJobID, "dimensions": 4, "fingerprints": items}
	}
	vec := []float64{0.5, 0.5, 0.5, 0.5}
	item := func(label, scope string, start, end, quality float64) map[string]any {
		return map[string]any{"speaker_label": label, "scope": scope, "start": start, "end": end, "quality": quality, "embedding": vec}
	}
	h.complete(fp, fingerprints(job.ID+999, item("SPEAKER_00", "speaker", 0, 1.5, 0.9)), http.StatusBadRequest)
	h.complete(fp, fingerprints(job.ID, map[string]any{"speaker_label": "SPEAKER_00", "scope": "speaker", "start": 0, "end": 1.5, "quality": 0.9, "embedding": []float64{1, 2}}), http.StatusBadRequest)
	h.complete(fp, fingerprints(job.ID, item("SPEAKER_00", "speaker", 0, 1.5, 0.9), item("SPEAKER_00", "turn", 0, 0.4, 0.9), item("SPEAKER_01", "speaker", 1, 2.5, 0.2)), http.StatusOK)
	view = h.voice()
	if view.Status != "done" || len(view.Speakers) != 2 {
		t.Fatalf("clip with unknown voices must complete: %+v", view)
	}
	first, second := view.Speakers[0].Fingerprints, view.Speakers[1].Fingerprints
	if len(first) != 2 || !first[0].Usable || first[0].Model != "fixture-embedding" || first[0].ModelVersion != "emb-v1" || first[0].Duration != 1.5 || first[0].Quality != 0.9 {
		t.Fatalf("usable fingerprint did not keep duration/quality/model/version: %+v", first)
	}
	if first[1].Usable || first[1].UnusableCause != "too_short" || len(second) != 1 || second[0].Usable || second[0].UnusableCause != "low_quality" {
		t.Fatalf("short or poor fingerprints must be unusable as references: %+v %+v", first, second)
	}
	h.complete(fp, fingerprints(job.ID, item("SPEAKER_00", "speaker", 0, 1.5, 0.9), item("SPEAKER_00", "turn", 0, 0.4, 0.9), item("SPEAKER_01", "speaker", 1, 2.5, 0.2)), http.StatusOK)
	var fingerprintRows int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM voice_fingerprints`).Scan(&fingerprintRows)
	if fingerprintRows != 3 {
		t.Fatalf("retried result duplicated fingerprints: %d", fingerprintRows)
	}

	// Re-running replaces the previous result and drops its fingerprints; an older fingerprint lease is refused.
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 2}, http.StatusOK)
	h.putPipeline(fakePipeline(1))
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), scheduleBody, http.StatusCreated)
	if v := h.voice(); v.Status != "queued" || len(v.Speakers) != 2 {
		t.Fatalf("a replanned clip must show its new step while keeping the previous result: %+v", v)
	}
	rerun, ok := h.claim()
	if !ok || rerun.Step != "voice.diarization.2" {
		t.Fatalf("expected the replanned diarization: %+v", rerun)
	}
	h.complete(rerun, diarizationPayload("SOLO"), http.StatusOK)
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM voice_fingerprints`).Scan(&fingerprintRows)
	if view = h.voice(); len(view.Speakers) != 1 || view.Speakers[0].Label != "SOLO" || fingerprintRows != 0 {
		t.Fatalf("replacement left stale speakers or fingerprints: %+v / %d", view, fingerprintRows)
	}
	older, ok := h.claim()
	if !ok || older.Step != "voice.fingerprint.2" {
		t.Fatalf("replacement fingerprint step missing: %+v", older)
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), scheduleBody, http.StatusCreated)
	rerun2, ok := h.claim()
	if !ok || rerun2.Step != "voice.diarization.2" {
		t.Fatalf("expected the second replanned diarization: %+v", rerun2)
	}
	h.complete(rerun2, diarizationPayload("DUO_A", "DUO_B"), http.StatusOK)
	h.complete(older, fingerprints(rerun.ID, item("SOLO", "speaker", 0, 1.5, 0.9)), http.StatusConflict)
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM voice_fingerprints`).Scan(&fingerprintRows)
	if view = h.voice(); len(view.Speakers) != 2 || view.Speakers[0].Label != "DUO_A" || fingerprintRows != 0 {
		t.Fatalf("a superseded fingerprint result must not be stored: %+v / %d", view, fingerprintRows)
	}

	// A worker failure is shown with its redacted code and an unknown voice never blocks anything.
	stale, ok := h.claim()
	if !ok || stale.Step != "voice.fingerprint.2" {
		t.Fatalf("latest fingerprint step missing: %+v", stale)
	}
	h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", stale.ID), map[string]string{"error": "provider_unavailable"}, stale.Lease, http.StatusOK)
	view = h.voice()
	failed := false
	for _, step := range view.Steps {
		failed = failed || (step.Step == "voice.fingerprint.2" && step.LastError == "provider_unavailable")
	}
	if !failed || len(view.Speakers) != 2 || view.Speakers[0].State != "unidentified" {
		t.Fatalf("failed step error not visible or speakers lost: %+v", view)
	}
	// Reading the clip and its voices needs no worker and no media processing.
	h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK)
	// The queue lists voice steps without exposing fingerprint vectors.
	if body := h.adminCall(http.MethodGet, fmt.Sprintf("/api/admin/jobs?clip_id=%d&step=voice.fingerprint.2", h.clipID), nil, http.StatusOK); bytes.Contains(body, []byte("embedding")) {
		t.Fatalf("job list leaked fingerprint vectors: %s", body)
	}
	// Deleting a clip would cascade; at minimum the schema forbids mixing dimensions.
	if _, err = pool.Exec(context.Background(), `INSERT INTO voice_fingerprints(local_speaker_id,clip_id,fingerprint_job_id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,model,model_version,dimensions,embedding) SELECT id,clip_id,diarization_job_id,'speaker',0,1,1,1,true,'m','v',3,'[1,2,3,4]' FROM local_speakers LIMIT 1`); err == nil {
		t.Fatal("fingerprint dimensions must match their declared dimensions")
	}
}
