//go:build integration

package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	communityID       = "pyannote-community-1"
	communityRevision = "3533c8cf8e369892e6b79ff1bf80f7b0286a54ee"
	secondID          = "test-diarization-b"
	secondRepo        = "synthetic/diarization-b"
	secondRevision    = "b00000000000000000000000000000000000000b"
)

// withSecondCatalogModel adds a synthetic diarization model to the catalogue for one test.
func withSecondCatalogModel(t *testing.T) {
	t.Helper()
	saved := voiceCatalog
	voiceCatalog = append(append([]voiceCatalogEntry{}, saved...), voiceCatalogEntry{
		ID: secondID, Kind: "diarization", Engine: "pyannote", Repo: secondRepo, Revision: secondRevision,
		License: "test", SizeBytes: 30_000_000, SupportsNumSpeakers: true,
	})
	t.Cleanup(func() { voiceCatalog = saved })
}

type catalogWorker struct {
	WorkerID      string `json:"worker_id"`
	Name          string `json:"name"`
	State         string `json:"state"`
	CauseCode     string `json:"cause_code"`
	BytesReceived int64  `json:"bytes_received"`
	BytesTotal    int64  `json:"bytes_total"`
}

type catalogModel struct {
	ID                  string          `json:"id"`
	Kind                string          `json:"kind"`
	Engine              string          `json:"engine"`
	Repo                string          `json:"repo"`
	Revision            string          `json:"revision"`
	License             string          `json:"license"`
	Gated               bool            `json:"gated"`
	MaxSpeakers         *int            `json:"max_speakers"`
	SizeBytes           int64           `json:"size_bytes"`
	SupportsNumSpeakers bool            `json:"supports_num_speakers"`
	Active              bool            `json:"active"`
	Candidate           bool            `json:"candidate"`
	Installed           bool            `json:"installed"`
	Workers             []catalogWorker `json:"workers"`
}

type voiceModelsView struct {
	Models []catalogModel `json:"models"`
}

func (h *voiceHarness) voiceModels() voiceModelsView {
	h.t.Helper()
	var v voiceModelsView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-models", nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

func (v voiceModelsView) model(t *testing.T, id string) catalogModel {
	t.Helper()
	for _, m := range v.Models {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("model %q missing from the catalogue", id)
	return catalogModel{}
}

type wantedModel struct {
	ModelID  string `json:"model_id"`
	Repo     string `json:"repo"`
	Revision string `json:"revision"`
	Role     string `json:"role"`
	Retry    bool   `json:"retry"`
}

func (h *voiceHarness) workerWants() ([]wantedModel, []wantedModel) {
	h.t.Helper()
	var out struct {
		Models []wantedModel `json:"models"`
		Remove []wantedModel `json:"remove"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, "/api/worker/voice-models", nil, "", http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Models, out.Remove
}

func (h *voiceHarness) reportModel(id, state, cause string, received, total int64, want int) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"model_id": id, "state": state, "cause_code": cause, "bytes_received": received, "bytes_total": total}, "", want)
}

func (h *voiceHarness) pipelineVersion() string {
	h.t.Helper()
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-pipeline", nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v.Version
}

func testVoiceModelCandidate(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t)
	h.putPipeline(map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5})
	h.transcribe()

	// The catalogue is data in the code: pinned revision, licence, gating, limits.
	community := h.voiceModels().model(t, communityID)
	if community.Repo != activeDiarization || community.Revision != communityRevision || community.Engine != "pyannote" || community.License == "" || !community.Gated || !community.SupportsNumSpeakers || community.SizeBytes <= 0 || !community.Active || community.Candidate {
		t.Fatalf("unexpected catalogue entry %+v", community)
	}
	// Free text is never a model; the fingerprint model is shown but cannot be a candidate.
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": "someone/else"}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": "pyannote-wespeaker-resnet34"}, http.StatusBadRequest)
	// The active model cannot be its own candidate.
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": communityID}, http.StatusConflict)

	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": secondID}, http.StatusOK)
	if m := h.voiceModels().model(t, secondID); !m.Candidate || m.Installed || len(m.Workers) != 1 || m.Workers[0].State != "not_installed" {
		t.Fatalf("a fresh candidate is not installed anywhere: %+v", m)
	}

	// The Worker pulls the desired state (candidate and active), never a repository it chose.
	wanted, _ := h.workerWants()
	if len(wanted) != 2 || wanted[0].Role != "candidate" || wanted[0].Repo != secondRepo || wanted[0].Revision != secondRevision || wanted[1].Role != "active" || wanted[1].ModelID != communityID {
		t.Fatalf("unexpected desired state %+v", wanted)
	}

	// Activation is refused until a Worker reports the candidate ready.
	if body := h.adminCall(http.MethodPost, "/api/admin/voice-models/"+secondID+"/activate", nil, http.StatusConflict); !bytes.Contains(body, []byte("voice_model_not_ready")) {
		t.Fatalf("expected voice_model_not_ready: %s", body)
	}
	h.reportModel(secondID, "downloading", "", 10_000_000, 30_000_000, http.StatusNoContent)
	if w := h.voiceModels().model(t, secondID).Workers[0]; w.State != "downloading" || w.BytesReceived != 10_000_000 || w.BytesTotal != 30_000_000 {
		t.Fatalf("progress not reported: %+v", w)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-models/"+secondID+"/activate", nil, http.StatusConflict)
	h.reportModel(secondID, "verifying", "", 30_000_000, 30_000_000, http.StatusNoContent)

	// A failure carries a stable cause; unknown causes and a failure without cause are refused.
	h.reportModel(secondID, "failed", "hf_token_missing", 0, 0, http.StatusNoContent)
	h.reportModel(secondID, "failed", "free text with hf_secret", 0, 0, http.StatusBadRequest)
	h.reportModel(secondID, "failed", "", 0, 0, http.StatusBadRequest)
	h.reportModel(secondID, "ready", "disk_full", 0, 0, http.StatusBadRequest)
	if w := h.voiceModels().model(t, secondID).Workers[0]; w.State != "failed" || w.CauseCode != "hf_token_missing" {
		t.Fatalf("failure cause not kept: %+v", w)
	}

	// Retry asks that Worker to prepare again; the flag is cleared by its next report.
	h.adminCall(http.MethodPost, "/api/admin/workers/wk_unknown_worker_identifier_000/voice-models/retry", nil, http.StatusNotFound)
	var workerID string
	if err := pool.QueryRow(t.Context(), `SELECT id FROM workers WHERE name='voice-fixture'`).Scan(&workerID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-models/retry", nil, http.StatusOK)
	if wanted, _ = h.workerWants(); !wanted[0].Retry || wanted[1].Retry {
		t.Fatalf("retry must be requested for the failed candidate only: %+v", wanted)
	}
	h.reportModel(secondID, "downloading", "", 0, 30_000_000, http.StatusNoContent)
	if wanted, _ = h.workerWants(); wanted[0].Retry {
		t.Fatal("the retry request must be consumed by the next report")
	}
	h.reportModel(secondID, "ready", "", 30_000_000, 30_000_000, http.StatusNoContent)
	if m := h.voiceModels().model(t, secondID); !m.Installed || m.Workers[0].State != "ready" {
		t.Fatalf("a ready Worker makes the model installed: %+v", m)
	}

	// Activation creates a new pipeline version; claims follow the active model's repository.
	var activated struct {
		Version string `json:"version"`
		Config  struct {
			Provider     string `json:"provider"`
			Model        string `json:"model"`
			ModelVersion string `json:"model_version"`
		} `json:"config"`
	}
	before := h.pipelineVersion()
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/voice-models/"+secondID+"/activate", nil, http.StatusOK), &activated); err != nil {
		t.Fatal(err)
	}
	if activated.Version == before || activated.Config.Provider != "pyannote" || activated.Config.Model != secondRepo || activated.Config.ModelVersion != secondRevision {
		t.Fatalf("activation must create a new pipeline version: %+v (was %s)", activated, before)
	}
	if m := h.voiceModels().model(t, secondID); !m.Active || m.Candidate {
		t.Fatalf("an activated candidate stops being a candidate: %+v", m)
	}
	h.adminCall(http.MethodPost, "/api/admin/clips/"+strconv64(h.clipID)+"/voice", map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	if job, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel(activeDiarization, "3533c8cf8e36")}); ok {
		t.Fatalf("a Worker holding only the previous model claimed %s", job.Step)
	}
	if _, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel(secondRepo, "b00000000000")}); !ok {
		t.Fatal("the Worker holding the new active model must claim the job")
	}

	// Rollback: the previous model stays installed, and reactivating it needs no new preparation.
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent)
	h.adminCall(http.MethodPost, "/api/admin/voice-models/"+communityID+"/activate", nil, http.StatusOK)
	if m := h.voiceModels().model(t, communityID); !m.Active || !m.Installed {
		t.Fatalf("rollback failed: %+v", m)
	}

	// Cancelling a candidate leaves the active model alone; delete only targets a model that is neither active nor candidate.
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": secondID}, http.StatusOK)
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/"+secondID+"/installation", nil, http.StatusConflict)
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/"+communityID+"/installation", nil, http.StatusConflict)
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/candidate", nil, http.StatusNoContent)
	if m := h.voiceModels().model(t, secondID); m.Candidate || !m.Installed {
		t.Fatalf("cancel must keep the installed model: %+v", m)
	}
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/"+secondID+"/installation", nil, http.StatusNoContent)
	if _, remove := h.workerWants(); len(remove) != 1 || remove[0].ModelID != secondID {
		t.Fatalf("the Worker must be told to remove the model: %+v", remove)
	}
	h.reportModel(secondID, "removed", "", 0, 0, http.StatusNoContent)
	if m := h.voiceModels().model(t, secondID); m.Installed || m.Workers[0].State != "not_installed" {
		t.Fatalf("a removed model is not installed: %+v", m)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-models/unknown/activate", nil, http.StatusNotFound)
	h.reportModel("unknown", "ready", "", 0, 0, http.StatusBadRequest)
}
