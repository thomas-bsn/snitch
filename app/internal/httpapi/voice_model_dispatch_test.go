//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// claimDeclaring pulses with the given model declaration, then tries to claim one job.
func (h *voiceHarness) claimDeclaring(models map[string]any) (claimedJob, bool) {
	h.t.Helper()
	body := map[string]any{"capacity": 2, "supported_steps": []string{"transcription.audio.*", "voice.*"}}
	if models != nil {
		body["models"] = models
	}
	h.workerCall(http.MethodPost, "/api/worker/pulse", body, "", http.StatusOK)
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

func voiceModel(name, version string) map[string]any {
	return map[string]any{"provider": "pyannote", "model": name, "version": version}
}

const (
	activeDiarization = "pyannote/speaker-diarization-community-1"
	activeEmbedding   = "pyannote/wespeaker-voxceleb-resnet34-LM"
)

func testVoiceModelDispatch(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	pyannote := map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5}
	h.putPipeline(pyannote)
	// Two jobs run at once below: the global queue limit would otherwise hide the dispatch rule.
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 3}, http.StatusOK)
	h.transcribe()
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)

	// With the real provider active, a Worker that declares no voice model takes no voice job.
	if job, ok := h.claimDeclaring(nil); ok {
		t.Fatalf("a Worker without voice models claimed %s", job.Step)
	}
	// Neither does one holding a different diarization model, or only the embedder.
	if job, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel("pyannote/speaker-diarization-3.1", "3533c8cf8e36")}); ok {
		t.Fatalf("a Worker holding another diarization model claimed %s", job.Step)
	}
	if job, ok := h.claimDeclaring(map[string]any{"fingerprint": voiceModel(activeEmbedding, "837717ddb9ff")}); ok {
		t.Fatalf("a Worker holding only the embedder claimed %s", job.Step)
	}

	// The Worker that holds the active model takes it.
	diarization, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel(activeDiarization, "3533c8cf8e36")})
	if !ok || diarization.Step != "voice.diarization.2" {
		t.Fatalf("the Worker holding the active diarization model must claim it, got %+v", diarization)
	}

	// A fingerprint job needs the active embedder, not the diarization model.
	h.adminCall(http.MethodPost, "/api/admin/jobs", map[string]any{"clip_id": h.clipID, "step": "voice.fingerprint.1", "input_version": "dispatch-1"}, http.StatusCreated)
	if job, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel(activeDiarization, "3533c8cf8e36")}); ok {
		t.Fatalf("a Worker without the embedder claimed %s", job.Step)
	}
	fingerprint, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel(activeDiarization, "3533c8cf8e36"), "fingerprint": voiceModel(activeEmbedding, "837717ddb9ff")})
	if !ok || fingerprint.Step != "voice.fingerprint.1" {
		t.Fatalf("the Worker holding the embedder must claim the fingerprint job, got %+v", fingerprint)
	}
}

type voiceOverviewWorker struct {
	Name                   string `json:"name"`
	Connected              bool   `json:"connected"`
	VoiceState             string `json:"voice_state"`
	CauseCode              string `json:"cause_code"`
	HoldsActiveDiarization bool   `json:"holds_active_diarization"`
	HoldsActiveFingerprint bool   `json:"holds_active_fingerprint"`
	Diarization            *struct {
		Model   string `json:"model"`
		Version string `json:"version"`
	} `json:"diarization"`
}

type voiceOverview struct {
	Provider     string `json:"provider"`
	ActiveModels struct {
		Diarization string `json:"diarization"`
		Fingerprint string `json:"fingerprint"`
	} `json:"active_models"`
	WorkersTotal      int                   `json:"workers_total"`
	WorkersVoiceReady int                   `json:"workers_voice_ready"`
	PendingVoiceJobs  int                   `json:"pending_voice_jobs"`
	Workers           []voiceOverviewWorker `json:"workers"`
}

func (h *voiceHarness) overview() voiceOverview {
	h.t.Helper()
	var o voiceOverview
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-overview", nil, http.StatusOK), &o); err != nil {
		h.t.Fatal(err)
	}
	return o
}

func (o voiceOverview) worker(t *testing.T, name string) voiceOverviewWorker {
	t.Helper()
	for _, w := range o.Workers {
		if w.Name == name {
			return w
		}
	}
	t.Fatalf("worker %q missing from %+v", name, o.Workers)
	return voiceOverviewWorker{}
}

func testVoiceOverview(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool) // its Worker is named voice-fixture
	h.putPipeline(map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5})
	h.transcribe()

	// Anonymous callers never read the inventory.
	resp, err := http.Get(h.ts.URL + "/api/admin/voice-overview")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous inventory read: %d", resp.StatusCode)
	}

	var second struct {
		Token string `json:"token"`
	}
	if err = json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/workers", map[string]any{"name": "voice-laptop", "max_concurrency": 2}, http.StatusCreated), &second); err != nil {
		t.Fatal(err)
	}
	fixture := h.token
	as := func(token string) { h.token = token }
	voiceEvents := func(status, cause string) {
		now := time.Now().UTC().Format(time.RFC3339)
		h.workerCall(http.MethodPost, "/api/worker/install-events", map[string]any{"events": []map[string]any{
			{"phase": "voice-models", "status": "succeeded", "occurred_at": now},
			{"phase": "voice-test", "status": status, "cause_code": cause, "occurred_at": now},
		}}, "", http.StatusNoContent)
	}

	// The fixture Worker holds both active models and passed its synthetic inference.
	as(fixture)
	h.claimDeclaring(map[string]any{"diarization": voiceModel(activeDiarization, "3533c8cf8e36"), "fingerprint": voiceModel(activeEmbedding, "837717ddb9ff")})
	voiceEvents("succeeded", "")
	// The laptop holds another diarization model.
	as(second.Token)
	h.claimDeclaring(map[string]any{"diarization": voiceModel("pyannote/speaker-diarization-3.1", "abcdef012345")})
	as(fixture)

	o := h.overview()
	if o.Provider != "pyannote" || o.ActiveModels.Diarization != activeDiarization || o.ActiveModels.Fingerprint != activeEmbedding {
		t.Fatalf("active models: %+v", o)
	}
	if o.WorkersTotal != 2 || o.WorkersVoiceReady != 1 || o.PendingVoiceJobs != 0 {
		t.Fatalf("counts: %+v", o)
	}
	ready := o.worker(t, "voice-fixture")
	if ready.VoiceState != "ready" || !ready.Connected || !ready.HoldsActiveDiarization || !ready.HoldsActiveFingerprint || ready.Diarization == nil || ready.Diarization.Version != "3533c8cf8e36" {
		t.Fatalf("ready worker: %+v", ready)
	}
	other := o.worker(t, "voice-laptop")
	if other.VoiceState != "models_missing" || other.HoldsActiveDiarization || other.HoldsActiveFingerprint || other.Diarization == nil || other.Diarization.Model != "pyannote/speaker-diarization-3.1" {
		t.Fatalf("worker holding another model: %+v", other)
	}

	// A pending voice job counts as waiting only while no Worker is ready.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	if got := h.overview().PendingVoiceJobs; got != 0 {
		t.Fatalf("a ready Worker exists: %d jobs reported waiting", got)
	}
	voiceEvents("failed", "gpu_unavailable")
	o = h.overview()
	failed := o.worker(t, "voice-fixture")
	if failed.VoiceState != "unavailable" || failed.CauseCode != "gpu_unavailable" || o.WorkersVoiceReady != 0 || o.PendingVoiceJobs != 1 {
		t.Fatalf("after a failed test: %+v / %+v", o, failed)
	}

	// The synthetic provider needs no model: its pending jobs are never blocked on a Worker.
	h.putPipeline(fakePipeline(2))
	if got := h.overview().PendingVoiceJobs; got != 0 {
		t.Fatalf("with the synthetic provider %d jobs reported waiting", got)
	}
}
