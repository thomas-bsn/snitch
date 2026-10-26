//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

type declaredModel struct {
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Version  *string `json:"version"`
	Endpoint *string `json:"endpoint"`
}

type declaredModels struct {
	Transcription *declaredModel `json:"transcription"`
	Diarization   *declaredModel `json:"diarization"`
	Fingerprint   *declaredModel `json:"fingerprint"`
}

func (m declaredModels) empty() bool {
	return m.Transcription == nil && m.Diarization == nil && m.Fingerprint == nil
}

// testWorkerModels covers the models a worker declares on its pulse: stored and
// listed to administrators, replaced on each pulse, and never accepting secrets.
func testWorkerModels(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewTLSServer(server.Handler())
	defer ts.Close()
	server.Config.PublicURL = ts.URL
	jar, _ := cookiejar.New(nil)
	admin := ts.Client()
	admin.Jar = jar
	anon := &http.Client{Transport: ts.Client().Transport}
	call := func(client *http.Client, method, path string, body any, csrf, token string, want int) []byte {
		t.Helper()
		var payload []byte
		if body != nil {
			payload = mustJSON(t, body)
		}
		req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(payload))
		req.Header.Set("Origin", ts.URL)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, requestErr := client.Do(req)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return data
	}
	call(admin, http.MethodPost, "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", "", http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/auth/me", nil, "", "", http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	var registration struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err = json.Unmarshal(call(admin, http.MethodPost, "/api/admin/workers", map[string]any{"name": "synthetic-models", "max_concurrency": 1}, me.CSRF, "", http.StatusCreated), &registration); err != nil {
		t.Fatal(err)
	}
	listModels := func() declaredModels {
		t.Helper()
		var page struct {
			Items []struct {
				ID     string         `json:"id"`
				Models declaredModels `json:"models"`
			} `json:"items"`
		}
		if err := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/workers", nil, "", "", http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		for _, candidate := range page.Items {
			if candidate.ID == registration.ID {
				return candidate.Models
			}
		}
		t.Fatal("worker missing")
		return declaredModels{}
	}
	pulse := func(models any, want int) []byte {
		t.Helper()
		body := map[string]any{"capacity": 1, "supported_steps": []string{"voice.*"}}
		if models != nil {
			body["models"] = models
		}
		return call(anon, http.MethodPost, "/api/worker/pulse", body, "", registration.Token, want)
	}

	// A worker that never declared anything lists no model.
	if m := listModels(); !m.empty() {
		t.Fatalf("undeclared worker lists models: %+v", m)
	}

	pulse(map[string]any{
		"transcription": map[string]any{"provider": "openai-compatible", "model": "large-v3", "endpoint": "host.docker.internal:8000"},
		"diarization":   map[string]any{"provider": "pyannote", "model": "pyannote/speaker-diarization-community-1", "version": "4.0.1"},
		"fingerprint":   map[string]any{"provider": "pyannote", "model": "wespeaker-voxceleb-resnet34-LM", "version": "1"},
	}, http.StatusOK)
	got := listModels()
	if got.Transcription == nil || got.Transcription.Model != "large-v3" || got.Transcription.Endpoint == nil || *got.Transcription.Endpoint != "host.docker.internal:8000" || got.Transcription.Version != nil {
		t.Fatalf("transcription model: %+v", got.Transcription)
	}
	if got.Diarization == nil || got.Diarization.Provider != "pyannote" || got.Diarization.Version == nil || *got.Diarization.Version != "4.0.1" {
		t.Fatalf("diarization model: %+v", got.Diarization)
	}
	if got.Fingerprint == nil || got.Fingerprint.Model != "wespeaker-voxceleb-resnet34-LM" {
		t.Fatalf("fingerprint model: %+v", got.Fingerprint)
	}

	// Roles are optional and every pulse replaces the previous declaration.
	pulse(map[string]any{"fingerprint": map[string]any{"provider": "fake", "model": "fake-voice-embedding", "version": "fake-v1"}}, http.StatusOK)
	got = listModels()
	if got.Transcription != nil || got.Diarization != nil || got.Fingerprint == nil || got.Fingerprint.Provider != "fake" {
		t.Fatalf("declaration was not replaced: %+v", got)
	}

	// Invalid and secret-looking values are refused and leave the stored declaration intact.
	one := func(role string, fields map[string]any) map[string]any { return map[string]any{role: fields} }
	for name, models := range map[string]any{
		"unknown role":           one("summary", map[string]any{"provider": "x", "model": "y"}),
		"missing model":          one("transcription", map[string]any{"provider": "x"}),
		"empty provider":         one("transcription", map[string]any{"provider": "", "model": "y"}),
		"oversized model":        one("transcription", map[string]any{"provider": "x", "model": strings.Repeat("a", 129)}),
		"control characters":     one("transcription", map[string]any{"provider": "x", "model": "a\nb"}),
		"markup":                 one("transcription", map[string]any{"provider": "x", "model": "<b>m</b>"}),
		"unknown field":          one("transcription", map[string]any{"provider": "x", "model": "y", "api_key": "k"}),
		"endpoint with userinfo": one("transcription", map[string]any{"provider": "x", "model": "y", "endpoint": "user:pass@host:8000"}),
		"endpoint is a URL":      one("transcription", map[string]any{"provider": "x", "model": "y", "endpoint": "https://host:8000/v1"}),
		"endpoint with query":    one("transcription", map[string]any{"provider": "x", "model": "y", "endpoint": "host:8000?key=abc"}),
		"hugging face token":     one("diarization", map[string]any{"provider": "pyannote", "model": "hf_abcdefghijklmnopqrstuvwxyz"}),
		"api key as version":     one("diarization", map[string]any{"provider": "pyannote", "model": "m", "version": "sk-abcdefghijklmnop"}),
		"worker token":           one("transcription", map[string]any{"provider": "rw_abcdefghijklmnopqrstuv", "model": "m"}),
		"long opaque secret":     one("transcription", map[string]any{"provider": "x", "model": "0123456789abcdef0123456789abcdef0123"}),
		"bearer value":           one("transcription", map[string]any{"provider": "x", "model": "Bearer abcdef"}),
		"not an object":          []string{"x"},
	} {
		body := pulse(models, http.StatusBadRequest)
		if !bytes.Contains(body, []byte("invalid_worker_models")) && !bytes.Contains(body, []byte("invalid_request")) {
			t.Fatalf("%s: unexpected error body %s", name, body)
		}
		for _, secret := range []string{"hf_abcdefghijklmnopqrstuvwxyz", "sk-abcdefghijklmnop", "user:pass"} {
			if bytes.Contains(body, []byte(secret)) {
				t.Fatalf("%s: error echoed a secret: %s", name, body)
			}
		}
		if after := listModels(); after.Fingerprint == nil || after.Fingerprint.Provider != "fake" {
			t.Fatalf("%s: rejected pulse changed the declaration: %+v", name, after)
		}
	}
	var stored string
	if err = pool.QueryRow(ctx, `SELECT models::text FROM workers WHERE id=$1`, registration.ID).Scan(&stored); err != nil || strings.Contains(stored, "hf_") || strings.Contains(stored, "sk-") {
		t.Fatalf("stored models: %q %v", stored, err)
	}

	// Only administrators read the declaration; a pulse without models clears it.
	call(anon, http.MethodGet, "/api/admin/workers", nil, "", registration.Token, http.StatusUnauthorized)
	pulse(nil, http.StatusOK)
	if m := listModels(); !m.empty() {
		t.Fatalf("pulse without models kept a declaration: %+v", m)
	}
}
