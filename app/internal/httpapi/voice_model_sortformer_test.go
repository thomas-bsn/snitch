//go:build integration

package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sortformerID       = "nvidia-sortformer-4spk-v2.1"
	sortformerRepo     = "nvidia/diar_streaming_sortformer_4spk-v2.1"
	sortformerRevision = "cd03eee90fbec18297ac31b8c21546e596b7f71c"
)

// A second catalogue engine (NVIDIA Sortformer through NeMo) is chosen, prepared and activated
// through the same mechanism as pyannote; the previous model stays installed and can come back.
func testVoiceModelSortformer(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent) // a real model is only activated once a connected Worker holds it
	h.putPipeline(map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5})
	h.transcribe()

	// The catalogue says what the model is: engine, pinned revision, licence, 4 voices at most, no speaker count.
	m := h.voiceModels().model(t, sortformerID)
	if m.Engine != "nemo" || m.Repo != sortformerRepo || m.Revision != sortformerRevision || m.License != "nvidia-open-model-license" || m.Gated ||
		m.MaxSpeakers == nil || *m.MaxSpeakers != 4 || m.SupportsNumSpeakers || m.SizeBytes <= 0 || m.Active || m.Candidate {
		t.Fatalf("unexpected catalogue entry %+v", m)
	}
	// Its engine is the provider: a typed pipeline cannot pair it with another one.
	h.adminCall(http.MethodPut, "/api/admin/voice-pipeline", map[string]any{"provider": "pyannote", "model": sortformerRepo, "model_version": sortformerRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5}, http.StatusBadRequest)

	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": sortformerID}, http.StatusOK)
	wanted, _ := h.workerWants()
	if len(wanted) != 2 || wanted[0].Role != "candidate" || wanted[0].Repo != sortformerRepo || wanted[0].Revision != sortformerRevision || wanted[0].Engine != "nemo" {
		t.Fatalf("unexpected desired state %+v", wanted)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-models/"+sortformerID+"/activate", nil, http.StatusConflict)

	// Without NeMo the Worker reports runtime_incompatible and the model stays unusable.
	h.reportModel(sortformerID, "failed", "runtime_incompatible", 0, 0, http.StatusNoContent)
	h.adminCall(http.MethodPost, "/api/admin/voice-models/"+sortformerID+"/activate", nil, http.StatusConflict)
	if w := h.voiceModels().model(t, sortformerID).Workers[0]; w.State != "failed" || w.CauseCode != "runtime_incompatible" {
		t.Fatalf("failure cause not kept: %+v", w)
	}
	h.reportModel(sortformerID, "ready", "", 1, 1, http.StatusNoContent)

	// Activation switches the provider to the engine; only a Worker holding the model claims voice jobs.
	body := string(h.adminCall(http.MethodPost, "/api/admin/voice-models/"+sortformerID+"/activate", nil, http.StatusOK))
	if !strings.Contains(body, `"provider":"nemo"`) || !strings.Contains(body, sortformerRepo) {
		t.Fatalf("activation must select the nemo provider: %s", body)
	}
	h.adminCall(http.MethodPost, "/api/admin/clips/"+strconv64(h.clipID)+"/voice", map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	if job, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel(activeDiarization, "3533c8cf8e36")}); ok {
		t.Fatalf("a Worker holding only the previous model claimed %s", job.Step)
	}
	if _, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel(sortformerRepo, "cd03eee90fbe")}); !ok {
		t.Fatal("the Worker holding Sortformer must claim the job")
	}
	if o := h.overview(); o.Provider != "nemo" {
		t.Fatalf("overview provider %q", o.Provider)
	}

	// Rollback: the previous model was never removed.
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent)
	h.adminCall(http.MethodPost, "/api/admin/voice-models/"+communityID+"/activate", nil, http.StatusOK)
	if c := h.voiceModels().model(t, communityID); !c.Active || !c.Installed {
		t.Fatalf("rollback failed: %+v", c)
	}
	if s := h.voiceModels().model(t, sortformerID); s.Active || !s.Installed {
		t.Fatalf("Sortformer must stay installed after rollback: %+v", s)
	}
}
