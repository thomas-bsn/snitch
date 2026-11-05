//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"snitch/internal/auth"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A Super-administrator edits and disables a reference, an Administrator only reads. A disabled reference can
// no longer be chosen, but it stays listed with the history of the Workers.
func testVoiceCatalogEdition(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := context.Background()
	const sortformer = "/api/admin/voice-models/nvidia-sortformer-4spk-v2.1"
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `UPDATE voice_model_catalog SET label='',license='nvidia-open-model-license',vram_bytes=0,enabled=true WHERE id='nvidia-sortformer-4spk-v2.1'`)
	})
	h.reportModel("nvidia-sortformer-4spk-v2.1", "ready", "", 10, 10, http.StatusNoContent)

	var m catalogModel
	edited := h.adminCall(http.MethodPatch, sortformer, map[string]any{"label": "Sortformer 4 voix", "license": "nvidia-open-model-license-1.0", "vram_bytes": 4_000_000_000}, http.StatusOK)
	if err := json.Unmarshal(edited, &m); err != nil {
		t.Fatal(err)
	}
	if m.Label != "Sortformer 4 voix" || m.License != "nvidia-open-model-license-1.0" || m.VRAMBytes != 4_000_000_000 || m.Repo != "nvidia/diar_streaming_sortformer_4spk-v2.1" {
		t.Fatalf("edition not applied or identity changed: %+v", m)
	}
	h.adminCall(http.MethodPatch, "/api/admin/voice-models/someone-else", map[string]any{"label": "x"}, http.StatusNotFound)
	h.adminCall(http.MethodPatch, sortformer, map[string]any{"vram_bytes": -1}, http.StatusBadRequest)

	// Disabling: no more candidate, the history of the Workers stays readable.
	h.adminCall(http.MethodPatch, sortformer, map[string]any{"enabled": false}, http.StatusOK)
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": "nvidia-sortformer-4spk-v2.1"}, http.StatusBadRequest)
	h.adminCall(http.MethodPost, sortformer+"/activate", nil, http.StatusBadRequest)
	disabled := h.voiceModels().model(t, "nvidia-sortformer-4spk-v2.1")
	if disabled.Enabled || len(disabled.Workers) != 1 || disabled.Workers[0].State != "ready" || !disabled.Installed {
		t.Fatalf("a disabled reference keeps its history: %+v", disabled)
	}
	// The active fingerprint model and the candidate cannot be disabled.
	h.adminCall(http.MethodPatch, "/api/admin/voice-models/pyannote-wespeaker-resnet34", map[string]any{"enabled": false}, http.StatusConflict)
	h.adminCall(http.MethodPatch, sortformer, map[string]any{"enabled": true}, http.StatusOK)
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": "nvidia-sortformer-4spk-v2.1"}, http.StatusOK)
	h.adminCall(http.MethodPatch, sortformer, map[string]any{"enabled": false}, http.StatusConflict)
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/candidate", nil, http.StatusNoContent)

	// An Administrator reads the catalogue but cannot edit it.
	if _, err := pool.Exec(ctx, `INSERT INTO users(username,password_hash,role,approved) VALUES('operator',$1,'admin',true)`, auth.Password("synthetic-password")); err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	operator := &voiceHarness{t: t, api: h.api, pool: pool, ts: h.ts, admin: &http.Client{Jar: jar}}
	operator.adminCall(http.MethodPost, "/api/auth/login", map[string]string{"username": "operator", "password": "synthetic-password"}, http.StatusOK)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(operator.adminCall(http.MethodGet, "/api/auth/me", nil, http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	operator.csrf = me.CSRF
	operator.voiceModels()
	operator.adminCall(http.MethodPatch, sortformer, map[string]any{"label": "refused"}, http.StatusForbidden)
	operator.adminCall(http.MethodPatch, sortformer, map[string]any{"enabled": false}, http.StatusForbidden)
}
