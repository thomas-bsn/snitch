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
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type analysisStepView struct {
	Step   string `json:"step"`
	Status string `json:"status"`
	Reason string `json:"reason"`
	JobID  *int64 `json:"job_id"`
}
type analysisRunView struct {
	ID     int64              `json:"id"`
	Scope  string             `json:"scope"`
	Status string             `json:"status"`
	Steps  []analysisStepView `json:"steps"`
	// PendingVoices is the live count of voices still to decide: the clip page and the queue agree.
	PendingVoices    int     `json:"pending_voices"`
	ConfirmedStakeAt *string `json:"confirmed_stake_at"`
}

func (r analysisRunView) step(name string) analysisStepView {
	for _, s := range r.Steps {
		if s.Step == name {
			return s
		}
	}
	return analysisStepView{}
}

func (h *voiceHarness) analysisPath() string {
	return fmt.Sprintf("/api/admin/clips/%d/analysis", h.clipID)
}

func (h *voiceHarness) startAnalysis(body map[string]any, want int) analysisRunView {
	h.t.Helper()
	var out struct {
		Run analysisRunView `json:"run"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, h.analysisPath(), body, want), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Run
}

func (h *voiceHarness) analysis() *analysisRunView {
	h.t.Helper()
	var out struct {
		Run *analysisRunView `json:"run"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, h.analysisPath(), nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Run
}

var analysisTranscript = map[string]any{"provider": "fixture", "model": "deterministic-v1", "version": "deterministic-v1", "language": "fr", "audio_stream_index": 2, "segments": []map[string]any{
	{"start": 0.2, "end": 0.9, "text": "premier", "words": []map[string]any{{"word": "premier", "start": 0.2, "end": 0.9}}},
}}

func testClipAnalysis(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	if h.analysis() != nil {
		t.Fatal("a clip never analyzed has no run")
	}
	h.adminCall(http.MethodPost, "/api/admin/clips/99999/analysis", map[string]any{}, http.StatusNotFound)
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{"scope": "everything"}, http.StatusBadRequest)

	// Voice detection never transcribes: without a valid transcription it explains the prerequisite.
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{"scope": "voice"}, http.StatusConflict)
	if h.analysis() != nil {
		t.Fatal("a refused voice analysis must not leave a run")
	}

	// The analysis waits for a track choice, and its dependent step waits behind it.
	run := h.startAnalysis(map[string]any{}, http.StatusCreated)
	if run.Status != "active" || run.Scope != "transcription" || run.step("preparation").Status != "blocked" || run.step("preparation").Reason != "audio_track_selection_required" || run.step("transcription").Status != "waiting" {
		t.Fatalf("unexpected run waiting for a track: %+v", run)
	}
	// Idempotent start: concurrent clicks give one run; another scope is refused.
	var wg sync.WaitGroup
	ids := make([]int64, 6)
	body := mustJSON(t, map[string]any{})
	for i := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodPost, h.ts.URL+h.analysisPath(), bytes.NewReader(body))
			req.Header.Set("Origin", h.ts.URL)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", h.csrf)
			resp, err := h.admin.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			data, _ := io.ReadAll(resp.Body)
			var out struct {
				Run analysisRunView `json:"run"`
			}
			_ = json.Unmarshal(data, &out)
			ids[i] = out.Run.ID
		}()
	}
	wg.Wait()
	for _, id := range ids {
		if id != run.ID {
			t.Fatalf("concurrent starts must return the active run %d: %v", run.ID, ids)
		}
	}
	if n := h.count(`SELECT count(*) FROM clip_analysis_runs WHERE clip_id=$1`, h.clipID); n != 1 {
		t.Fatalf("%d runs for one clip", n)
	}
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{"scope": "full"}, http.StatusConflict)

	// Choosing the track plans the transcription; the run follows the job.
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusOK)
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{}, http.StatusOK)
	run = *h.analysis()
	if run.step("preparation").Status != "done" || run.step("transcription").Status != "waiting" || run.step("transcription").Reason != "awaiting_compatible_worker" {
		t.Fatalf("unexpected queued run: %+v", run)
	}
	job, ok := h.claim()
	if !ok || job.Step != "transcription.audio.2" {
		t.Fatalf("expected the transcription task, got %+v", job)
	}
	if err := h.api.Analysis.ReconcileClip(context.Background(), h.clipID); err != nil {
		t.Fatal(err)
	}
	if run = *h.analysis(); run.step("transcription").Status != "running" {
		t.Fatalf("unexpected running run: %+v", run)
	}
	// A failed attempt is a retry, not a failure (reported through the worker API).
	h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", job.ID), map[string]any{"error": "synthetic"}, job.Lease, http.StatusOK)
	run = *h.analysis()
	if run.step("transcription").Status != "retry" || run.step("transcription").Reason != "retry_scheduled" || run.Status != "active" {
		t.Fatalf("unexpected retry run: %+v", run)
	}
	// The run survives a restart: the job finishes while no server reconciles, a new server catches up.
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET status='done',result=$2::jsonb,updated_at=now() WHERE id=$1`, job.ID, string(mustJSON(t, analysisTranscript))); err != nil {
		t.Fatal(err)
	}
	cfg := h.api.Config
	cfg.UploadRoot = t.TempDir()
	restarted := New(pool, cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := restarted.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	deadline := time.Now().Add(10 * time.Second)
	for h.analysis().Status == "active" && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	run = *h.analysis()
	if run.Status != "completed" || run.step("transcription").Status != "done" || run.step("transcription").Reason != "completed" {
		t.Fatalf("a restart must finish the run: %+v", run)
	}

	// A second analysis reuses the valid transcription and plans no job.
	jobs := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1`, h.clipID)
	run2 := h.startAnalysis(map[string]any{}, http.StatusCreated)
	if run2.ID == run.ID || run2.Status != "completed" || run2.step("transcription").Reason != "reused_existing" || h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1`, h.clipID) != jobs {
		t.Fatalf("valid results must be reused: %+v", run2)
	}
	// Voice scope builds on the existing transcription: it plans the diarization and waits for a Worker.
	run3 := h.startAnalysis(map[string]any{"scope": "voice"}, http.StatusCreated)
	if run3.Status != "active" || run3.step("transcription").Reason != "reused_existing" || run3.step("indexing").Status != "not_applicable" || run3.step("diarization").Reason != "awaiting_worker" || run3.step("matching").Status != "waiting" {
		t.Fatalf("unexpected voice run: %+v", run3)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE clip_analysis_runs SET status='cancelled' WHERE id=$1`, run3.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM jobs WHERE step LIKE 'voice.%'`); err != nil {
		t.Fatal(err)
	}
	// Without a configured semantic index the indexing step is not applicable, and the run still ends.
	run5 := h.startAnalysis(map[string]any{"scope": "full"}, http.StatusCreated)
	if run5.Status != "active" || run5.step("indexing").Status != "not_applicable" || run5.step("indexing").Reason != "semantic_search_unavailable" || run5.step("diarization").Reason != "awaiting_worker" {
		t.Fatalf("unexpected full run: %+v", run5)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE clip_analysis_runs SET status='cancelled' WHERE id=$1`, run5.ID); err != nil {
		t.Fatal(err)
	}

	// A job that dies fails the step and the run, and frees the clip for a new analysis.
	if _, err := pool.Exec(context.Background(), `DELETE FROM jobs WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	run4 := h.startAnalysis(map[string]any{}, http.StatusCreated)
	if run4.step("transcription").Reason != "awaiting_worker" {
		t.Fatalf("expected a queued job: %+v", run4)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET status='dead' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if err := h.api.Analysis.ReconcileClip(context.Background(), h.clipID); err != nil {
		t.Fatal(err)
	}
	if r := h.analysis(); r.Status != "failed" || r.step("transcription").Reason != "job_dead" {
		t.Fatalf("a dead job fails the run: %+v", r)
	}
	h.startAnalysis(map[string]any{}, http.StatusCreated)

	// Administrators only.
	member := h.newUser("analysis-member")
	h.userCall(member, http.MethodGet, h.analysisPath(), nil, http.StatusForbidden)
	h.userCall(member, http.MethodPost, h.analysisPath(), map[string]any{}, http.StatusForbidden)
}

func testClipAnalysisVoice(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := context.Background()
	h.transcribe()
	h.putPipeline(fakePipeline(2))
	// A semantic index is configured; its task will fail while the voices go on.
	if err := h.api.Store.Set(ctx, "embeddings_url", "http://embeddings.invalid"); err != nil {
		t.Fatal(err)
	}
	var profile int64
	if err := pool.QueryRow(ctx, `INSERT INTO embedding_profiles(model,dimensions,state) VALUES('synthetic',4,'active') RETURNING id`).Scan(&profile); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=$1 WHERE id=1`, profile); err != nil {
		t.Fatal(err)
	}

	// A source rule designates the microphone track (index 1): the run attributes it directly.
	var sourceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clip_sources ORDER BY id LIMIT 1`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	alice := h.newPerson("Alice Analysis")
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/sources/%d/microphone-track", sourceID), map[string]any{"title": "Micro", "person_id": alice}, http.StatusOK)

	run := h.startAnalysis(map[string]any{"scope": "full"}, http.StatusCreated)
	if n := h.count(`SELECT count(*) FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND ls.kind='microphone' AND a.person_id=$2 AND a.decision_level='automatic'`, h.clipID, alice); n != 1 {
		t.Fatalf("the microphone track must be attributed directly by the run: %d", n)
	}
	if run.Status != "active" || run.step("indexing").Reason != "awaiting_indexer" || run.step("diarization").Reason != "awaiting_worker" || run.step("fingerprints").Status != "waiting" || run.step("matching").Status != "waiting" {
		t.Fatalf("both branches must start after the transcription: %+v", run)
	}
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("expected the diarization task, got %+v", job)
	}
	h.complete(job, diarizationPayload("SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	run = *h.analysis()
	if run.step("diarization").Status != "done" || run.step("fingerprints").Reason != "awaiting_worker" || run.step("indexing").Status != "waiting" {
		t.Fatalf("unexpected run after the diarization: %+v", run)
	}
	// The semantic index fails: the voice branch goes on and the text search stays.
	if _, err := pool.Exec(ctx, `UPDATE embedding_tasks SET state='failed',error='synthetic' WHERE profile_id=$1`, profile); err != nil {
		t.Fatal(err)
	}
	fp, ok := h.claim()
	if !ok || fp.Step != "voice.fingerprint.2" {
		t.Fatalf("expected the fingerprint task, got %+v", fp)
	}
	vec := []float64{0.5, 0.5, 0.5, 0.5}
	h.complete(fp, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": 2, "diarization_job_id": job.ID, "dimensions": 4, "fingerprints": []map[string]any{
		{"speaker_label": "SPEAKER_00", "scope": "speaker", "start": 0, "end": 1.5, "quality": 0.9, "embedding": vec},
		{"speaker_label": "SPEAKER_01", "scope": "speaker", "start": 1, "end": 2.5, "quality": 0.9, "embedding": vec},
	}}, http.StatusOK)
	run = *h.analysis()
	if run.Status != "failed" || run.step("indexing").Status != "failed" || run.step("indexing").Reason != "semantic_index_failed" || run.step("diarization").Status != "done" || run.step("fingerprints").Status != "done" || run.step("matching").Status != "done" {
		t.Fatalf("a semantic failure must be a partial failure of the run: %+v", run)
	}
	// Unknown voices neither block the clip nor the technical end of the voice branch.
	if v := h.voice(); v.Status != "done" || len(v.Speakers) != 3 || v.Speakers[0].State != "identified" || v.Speakers[1].State != "unidentified" || v.Speakers[2].State != "unidentified" {
		t.Fatalf("unknown voices must stay unidentified: %+v", v)
	}
	if n := h.count(`SELECT count(*) FROM clips c JOIN jobs j ON j.clip_id=c.id WHERE c.id=$1 AND j.step LIKE 'transcription.audio.%' AND j.status='done'`, h.clipID); n != 1 {
		t.Fatalf("the transcription must stay: %d", n)
	}

	// Valid results are reused: the voice scope plans no job and ends at once.
	jobsBefore := h.count(`SELECT count(*) FROM jobs`)
	run2 := h.startAnalysis(map[string]any{"scope": "voice"}, http.StatusCreated)
	if run2.Status != "completed" || run2.step("diarization").Reason != "reused_existing" || run2.step("fingerprints").Reason != "reused_existing" || run2.step("matching").Status != "done" || h.count(`SELECT count(*) FROM jobs`) != jobsBefore {
		t.Fatalf("valid voice results must be reused: %+v", run2)
	}

	// A dead fingerprint job fails its branch and blocks only the steps behind it.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='dead' WHERE step='voice.fingerprint.2'`); err != nil {
		t.Fatal(err)
	}
	run3 := h.startAnalysis(map[string]any{"scope": "voice"}, http.StatusCreated)
	if run3.Status != "failed" || run3.step("diarization").Status != "done" || run3.step("fingerprints").Reason != "job_dead" || run3.step("matching").Reason != "upstream_failed" {
		t.Fatalf("unexpected run with a dead fingerprint job: %+v", run3)
	}
}
