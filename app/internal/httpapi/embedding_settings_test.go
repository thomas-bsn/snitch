//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
)

func testEmbeddingSettings(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	provider := newReindexProvider(t)
	input := map[string]string{"url": provider.srv.URL, "model": "fixture-v1"}
	member := h.newUser("semantic-member")
	h.userCall(member, http.MethodGet, "/api/admin/embeddings/settings", nil, http.StatusForbidden)
	h.userCall(member, http.MethodPut, "/api/admin/embeddings/settings", input, http.StatusForbidden)
	operator := h.newUser("semantic-operator")
	if _, err := pool.Exec(context.Background(), `UPDATE users SET role='admin' WHERE username=$1`, operator.name); err != nil {
		t.Fatal(err)
	}
	h.userCall(operator, http.MethodGet, "/api/admin/embeddings/settings", nil, http.StatusOK)
	h.userCall(operator, http.MethodPut, "/api/admin/embeddings/settings", input, http.StatusOK)
	h.transcribe()
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", input, http.StatusOK)
	var settings struct {
		URL   string `json:"url"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/embeddings/settings", nil, http.StatusOK), &settings); err != nil || settings.URL != provider.srv.URL || settings.Model != "fixture-v1" {
		t.Fatalf("settings: %+v %v", settings, err)
	}
	provider.setDown(true)
	var search struct {
		Degraded bool              `json:"degraded"`
		Items    []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/search?mode=hybrid&q=premier", nil, http.StatusOK), &search); err != nil || !search.Degraded || len(search.Items) == 0 {
		t.Fatalf("lexical fallback: %+v %v", search, err)
	}
	input["model"] = "fixture-v2"
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", input, http.StatusServiceUnavailable)
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/embeddings/settings", nil, http.StatusOK), &settings); err != nil || settings.Model != "fixture-v1" {
		t.Fatalf("failed test changed settings: %+v %v", settings, err)
	}
	provider.setDown(false)
	input["model"] = "fixture-v2"
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", input, http.StatusOK)
	deadline := time.Now().Add(15 * time.Second)
	for {
		var status struct {
			Model          string `json:"model"`
			CandidateModel string `json:"candidate_model"`
			Indexed        int    `json:"indexed_segments"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/embeddings", nil, http.StatusOK), &status); err != nil {
			t.Fatal(err)
		}
		if status.Model == "fixture-v2" && status.Indexed > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("reconstruction not completed: %+v", status)
		}
		time.Sleep(100 * time.Millisecond)
	}
	// A second server sees the same database but explicit environment settings take precedence.
	envServer := httptest.NewServer(New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), EmbeddingsURL: "http://environment.invalid/v1/embeddings", EmbeddingsModel: "environment-model"}, slog.New(slog.NewJSONHandler(io.Discard, nil))).Handler())
	defer envServer.Close()
	response, err := h.admin.Get(envServer.URL + "/api/admin/embeddings/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var overridden struct {
		URL                  string `json:"url"`
		Model                string `json:"model"`
		URLFromEnvironment   bool   `json:"url_from_environment"`
		ModelFromEnvironment bool   `json:"model_from_environment"`
	}
	if err := json.NewDecoder(response.Body).Decode(&overridden); err != nil || response.StatusCode != http.StatusOK || overridden.URL != "http://environment.invalid/v1/embeddings" || overridden.Model != "environment-model" || !overridden.URLFromEnvironment || !overridden.ModelFromEnvironment {
		t.Fatalf("environment precedence: status=%d %+v %v", response.StatusCode, overridden, err)
	}

	// Environment overrides must not overwrite stored settings that reappear when the override is removed.
	overrideServer := httptest.NewServer(New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), EmbeddingsURL: provider.srv.URL, EmbeddingsModel: "fixture-v2"}, slog.New(slog.NewJSONHandler(io.Discard, nil))).Handler())
	defer overrideServer.Close()
	req, _ := http.NewRequest(http.MethodPut, overrideServer.URL+"/api/admin/embeddings/settings", bytes.NewReader(mustJSON(t, map[string]string{"url": "http://127.0.0.1:9", "model": "ignored"})))
	req.Header.Set("Origin", overrideServer.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", h.csrf)
	h.do(h.admin, req, http.StatusOK)
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/embeddings/settings", nil, http.StatusOK), &settings); err != nil || settings.URL != provider.srv.URL || settings.Model != "fixture-v2" {
		t.Fatalf("stored settings overwritten: %+v %v", settings, err)
	}

}
