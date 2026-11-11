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
// a reference added to the table is served, and a separation reference is supported only with its limits.
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

	// Separation (#401, ADR 0065): the two candidates are in the catalogue with the limits of their documentation,
	// visible through the API, and nothing is installed by their presence.
	whamr := models.model(t, "speechbrain-sepformer-whamr16k")
	if whamr.Kind != "separation" || whamr.Engine != "speechbrain-sepformer" || whamr.Family != "speechbrain" || !whamr.Supported || !whamr.Enabled ||
		whamr.Repo != "speechbrain/sepformer-whamr16k" || whamr.Revision != "21a5b500c6f52fddc387c5d9e5fb13ffd6f039c5" || whamr.License != "apache-2.0" || whamr.Gated ||
		whamr.SeparatedVoices != 2 || whamr.SampleRateHz != 16000 || whamr.SizeBytes == 0 || whamr.SignedRelease {
		t.Fatalf("WHAMR! 16 kHz candidate: %+v", whamr)
	}
	wsj := models.model(t, "speechbrain-sepformer-wsj03mix")
	if wsj.Kind != "separation" || !wsj.Supported || wsj.Repo != "speechbrain/sepformer-wsj03mix" || wsj.Revision != "21e78b44a378092c34fff32ba8e788a73584f474" ||
		wsj.SeparatedVoices != 3 || wsj.SampleRateHz != 8000 {
		t.Fatalf("WSJ0-3Mix candidate: %+v", wsj)
	}
	for _, m := range []catalogModel{whamr, wsj} {
		if m.Installed || m.Active {
			t.Fatalf("a catalogue reference installs nothing: %+v", m)
		}
		for _, w := range m.Workers {
			if w.State != "not_installed" {
				t.Fatalf("a catalogue reference installs nothing: %+v", m)
			}
		}
		if separationModels[m.Repo] != (separationLimits{Voices: m.SeparatedVoices, SampleRateHz: m.SampleRateHz}) {
			t.Fatalf("the seeded limits of %s differ from the documented table", m.Repo)
		}
	}
	// Another type shows no separation limit.
	if sortformer.SeparatedVoices != 0 || sortformer.SampleRateHz != 0 {
		t.Fatalf("a diarization model has no separation limit: %+v", sortformer)
	}
	// A supported separation reference must declare both limits; an unsupported one is still listed as such.
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM voice_model_catalog WHERE id IN ('sep-a','sep-b','sep-c','table-only')`)
	})
	for _, limits := range []string{"0,0", "2,0", "0,16000", "1,16000"} {
		if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,supported,family,separated_voices,sample_rate_hz) VALUES('sep-a','separation','speechbrain-sepformer','synthetic/sep','d000000000000000000000000000000000000000',true,'speechbrain',`+limits+`)`); err == nil {
			t.Fatalf("a supported separation reference without its limits (%s) must be rejected by the database", limits)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,supported,family,separated_voices,sample_rate_hz) VALUES('sep-c','separation','speechbrain-sepformer','synthetic/sep-c','d000000000000000000000000000000000000000',true,'speechbrain',2,8000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,repo,revision,supported,unsupported_reason) VALUES('sep-b','separation','synthetic/sep','d000000000000000000000000000000000000000',false,'no_adapter')`); err != nil {
		t.Fatal(err)
	}
	if m := h.voiceModels().model(t, "sep-b"); m.Kind != "separation" || m.Supported || m.UnsupportedReason != "no_adapter" {
		t.Fatalf("a separation reference is listed as unsupported: %+v", m)
	}
	// A separation model is never the diarization candidate, nor the active model of anything.
	for _, id := range []string{"sep-b", "sep-c", "speechbrain-sepformer-whamr16k"} {
		h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": id}, http.StatusBadRequest)
		h.adminCall(http.MethodPost, "/api/admin/voice-models/"+id+"/activate", nil, http.StatusBadRequest)
	}

	// A reference added to the table is served: the code holds no list.
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,license,supported) VALUES('table-only','diarization','pyannote','synthetic/table-only','c000000000000000000000000000000000000000','test',true)`); err != nil {
		t.Fatal(err)
	}
	if m := h.voiceModels().model(t, "table-only"); m.Repo != "synthetic/table-only" {
		t.Fatalf("a table row must appear in the catalogue: %+v", m)
	}
}
