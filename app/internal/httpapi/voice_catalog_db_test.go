//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The catalogue lives in the database (ADR 0038): the three historical references are migrated as they were,
// a reference added to the table is served, and the separation type exists without any adapter.
func testVoiceCatalogInDatabase(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	models := h.voiceModels()

	sortformer := models.model(t, "nvidia-sortformer-4spk-v2.1")
	if sortformer.Kind != "diarization" || sortformer.Engine != "nemo" || sortformer.Repo != "nvidia/diar_streaming_sortformer_4spk-v2.1" ||
		sortformer.Revision != "cd03eee90fbec18297ac31b8c21546e596b7f71c" || sortformer.MaxSpeakers == nil || *sortformer.MaxSpeakers != 4 || sortformer.Gated {
		t.Fatalf("historical Sortformer reference changed: %+v", sortformer)
	}
	wespeaker := models.model(t, "pyannote-wespeaker-resnet34")
	if wespeaker.Kind != "fingerprint" || wespeaker.Repo != "pyannote/wespeaker-voxceleb-resnet34-LM" || wespeaker.Revision != "837717ddb9ff5507820346191109dc79c958d614" || !wespeaker.Enabled || !wespeaker.Supported {
		t.Fatalf("historical WeSpeaker reference changed: %+v", wespeaker)
	}
	// The active fingerprint model is read from the database, no longer from a constant of the code.
	var overview struct {
		ActiveModels struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"active_models"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-overview", nil, http.StatusOK), &overview); err != nil {
		t.Fatal(err)
	}
	if overview.ActiveModels.Fingerprint != wespeaker.Repo {
		t.Fatalf("the migrated active fingerprint model is %q, want %q", overview.ActiveModels.Fingerprint, wespeaker.Repo)
	}

	// Separation has no adapter: a reference of that type can never be marked supported, only listed as unsupported.
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM voice_model_catalog WHERE id IN ('sep-a','sep-b','table-only')`)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,repo,revision,supported) VALUES('sep-a','separation','synthetic/sep','d000000000000000000000000000000000000000',true)`); err == nil {
		t.Fatal("a supported separation reference must be rejected by the database")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,repo,revision,supported,unsupported_reason) VALUES('sep-b','separation','synthetic/sep','d000000000000000000000000000000000000000',false,'no_adapter')`); err != nil {
		t.Fatal(err)
	}
	if m := h.voiceModels().model(t, "sep-b"); m.Kind != "separation" || m.Supported || m.UnsupportedReason != "no_adapter" {
		t.Fatalf("a separation reference is listed as unsupported: %+v", m)
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": "sep-b"}, http.StatusBadRequest)

	// A reference added to the table is served: the code holds no list.
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,license,supported) VALUES('table-only','diarization','pyannote','synthetic/table-only','c000000000000000000000000000000000000000','test',true)`); err != nil {
		t.Fatal(err)
	}
	if m := h.voiceModels().model(t, "table-only"); m.Repo != "synthetic/table-only" {
		t.Fatalf("a table row must appear in the catalogue: %+v", m)
	}
}
