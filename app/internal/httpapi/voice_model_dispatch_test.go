//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

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
	pyannote := map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": "community-1", "access_confirmed": true, "min_fingerprint_seconds": 1.0, "min_quality": 0.5}
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
