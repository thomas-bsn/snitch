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
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
)

// validated tests the settings and returns the saved-ready input carrying the validation token.
func (h *voiceHarness) validatedEmbedding(input map[string]string) map[string]string {
	h.t.Helper()
	var proof struct {
		Token      string `json:"validation_token"`
		Dimensions int    `json:"dimensions"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/embeddings/settings/test", input, http.StatusOK), &proof); err != nil || proof.Token == "" || proof.Dimensions < 1 {
		h.t.Fatalf("test proof: %+v %v", proof, err)
	}
	return map[string]string{"url": input["url"], "model": input["model"], "validation_token": proof.Token}
}

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
	h.userCall(member, http.MethodPost, "/api/admin/embeddings/settings/test", input, http.StatusForbidden)
	h.transcribe()
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", h.validatedEmbedding(input), http.StatusOK)
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
	h.adminCall(http.MethodPost, "/api/admin/embeddings/settings/test", input, http.StatusServiceUnavailable)
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/embeddings/settings", nil, http.StatusOK), &settings); err != nil || settings.Model != "fixture-v1" {
		t.Fatalf("failed test changed settings: %+v %v", settings, err)
	}
	provider.setDown(false)
	input["model"] = "fixture-v2"
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", h.validatedEmbedding(input), http.StatusOK)
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
	var proof struct {
		Token string `json:"validation_token"`
	}
	testReq, _ := http.NewRequest(http.MethodPost, overrideServer.URL+"/api/admin/embeddings/settings/test", bytes.NewReader(mustJSON(t, map[string]string{"url": "http://127.0.0.1:9", "model": "ignored"})))
	testReq.Header.Set("Origin", overrideServer.URL)
	testReq.Header.Set("Content-Type", "application/json")
	testReq.Header.Set("X-CSRF-Token", h.csrf)
	if err := json.Unmarshal(h.do(h.admin, testReq, http.StatusOK), &proof); err != nil || proof.Token == "" {
		t.Fatalf("environment test proof: %+v %v", proof, err)
	}
	req, _ := http.NewRequest(http.MethodPut, overrideServer.URL+"/api/admin/embeddings/settings", bytes.NewReader(mustJSON(t, map[string]string{"url": "http://127.0.0.1:9", "model": "ignored", "validation_token": proof.Token})))
	req.Header.Set("Origin", overrideServer.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", h.csrf)
	h.do(h.admin, req, http.StatusOK)
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/embeddings/settings", nil, http.StatusOK), &settings); err != nil || settings.URL != provider.srv.URL || settings.Model != "fixture-v2" {
		t.Fatalf("stored settings overwritten: %+v %v", settings, err)
	}

}

// testEmbeddingValidationGate: Test never writes; Save needs the single-use, short-lived proof of this exact
// configuration, however the API is called.
func testEmbeddingValidationGate(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	provider := newReindexProvider(t)
	input := map[string]string{"url": provider.srv.URL, "model": "gate-v1"}
	count := func(query string) (n int) {
		t.Helper()
		if err := pool.QueryRow(ctx, query).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Test writes nothing: no profile, no stored setting, no index change.
	h.validatedEmbedding(input)
	if count(`SELECT count(*) FROM embedding_profiles`) != 0 || count(`SELECT count(*) FROM settings WHERE key LIKE 'embeddings_%'`) != 0 {
		t.Fatal("Test must not persist anything")
	}
	// A failing provider yields no proof, and a malformed request is refused.
	provider.setDown(true)
	h.adminCall(http.MethodPost, "/api/admin/embeddings/settings/test", input, http.StatusServiceUnavailable)
	provider.setDown(false)
	h.adminCall(http.MethodPost, "/api/admin/embeddings/settings/test", map[string]string{"url": "ftp://nope", "model": "m"}, http.StatusBadRequest)

	// What failed is told: unknown model and invalid answer have their own codes.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.Error(w, "model not found", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer bad.Close()
	for path, want := range map[string]string{"/missing": "embedding_model_not_found", "/garbage": "embedding_response_invalid"} {
		if got := string(h.adminCall(http.MethodPost, "/api/admin/embeddings/settings/test", map[string]string{"url": bad.URL + path, "model": "m"}, http.StatusServiceUnavailable)); !strings.Contains(got, want) {
			t.Fatalf("%s: %s", path, got)
		}
	}

	// Direct API calls cannot skip the proof.
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", input, http.StatusConflict)
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", map[string]string{"url": input["url"], "model": input["model"], "validation_token": "forged"}, http.StatusConflict)
	if count(`SELECT count(*) FROM embedding_profiles`) != 0 {
		t.Fatal("a refused Save must not change anything")
	}
	// The proof is bound to the exact tested values.
	proven := h.validatedEmbedding(input)
	changed := map[string]string{"url": proven["url"], "model": "gate-other", "validation_token": proven["validation_token"]}
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", changed, http.StatusConflict)
	// ... and to the user who tested.
	other := h.newUser("gate-other-admin")
	if _, err := pool.Exec(ctx, `UPDATE users SET role='admin' WHERE username=$1`, other.name); err != nil {
		t.Fatal(err)
	}
	h.userCall(other, http.MethodPut, "/api/admin/embeddings/settings", proven, http.StatusConflict)
	// Expired proofs are refused with their own code.
	if _, err := pool.Exec(ctx, `UPDATE embedding_validations SET expires_at=now()-interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	if got := string(h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", proven, http.StatusConflict)); !strings.Contains(got, "embedding_validation_expired") {
		t.Fatalf("expired proof: %s", got)
	}
	// Valid proof: saved once; reusing it (double click, replay) is refused.
	proven = h.validatedEmbedding(input)
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", proven, http.StatusOK)
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", proven, http.StatusConflict)
	var settings struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/embeddings/settings", nil, http.StatusOK), &settings); err != nil || settings.Model != "gate-v1" {
		t.Fatalf("saved settings: %+v %v", settings, err)
	}
	// A refused Save (rebuild in progress) keeps the proof usable.
	next := map[string]string{"url": input["url"], "model": "gate-v2"}
	proof2 := h.validatedEmbedding(next)
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", proof2, http.StatusOK)
	third := h.validatedEmbedding(map[string]string{"url": input["url"], "model": "gate-v3"})
	h.adminCall(http.MethodPut, "/api/admin/embeddings/settings", third, http.StatusConflict) // candidate still building
	if count(`SELECT count(*) FROM embedding_validations`) < 1 {
		t.Fatal("a Save refused for another reason must not burn the proof")
	}
}
