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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/jackc/pgx/v5/pgxpool"
	"receipts/internal/auth"
	"receipts/internal/config"
	"receipts/internal/setup"
)

func testDiscordSettings(t *testing.T, pool *pgxpool.Pool) {
	var logs bytes.Buffer
	s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(&logs, nil)))
	ts := httptest.NewUnstartedServer(s.Handler())
	s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec.Servers = nil
	router, err := legacy.NewRouter(spec)
	if err != nil {
		t.Fatal(err)
	}
	call := func(t *testing.T, method, path, data, csrf string, want int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", ts.URL)
		req.Header.Set("X-CSRF-Token", csrf)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if bytes.Contains(body, []byte("fixture-discord-secret")) || bytes.Contains(body, []byte("replacement-secret")) {
			t.Fatal("secret returned to browser")
		}
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, body)
		}
		route, params, err := router.FindRoute(req)
		if err != nil {
			t.Fatal(err)
		}
		input := &openapi3filter.RequestValidationInput{Request: req, Route: route, PathParams: params, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
		if err := openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: resp.StatusCode, Header: resp.Header, Body: io.NopCloser(bytes.NewReader(body))}); err != nil {
			t.Fatal("Discord settings response contract", err)
		}
		return body
	}
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	call(t, "POST", "/api/setup", `{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`, "", 201)
	var session struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(call(t, "GET", "/api/auth/me", "", "", 200), &session); err != nil {
		t.Fatal(err)
	}
	t.Run("a legacy incomplete configuration is never advertised", func(t *testing.T) {
		if err := s.Store.Set(context.Background(), "discord_client_id", "incomplete-app"); err != nil {
			t.Fatal(err)
		}
		body := call(t, "GET", "/api/setup", "", "", 200)
		if !bytes.Contains(body, []byte(`"discord_enabled":false`)) {
			t.Fatalf("incomplete Discord enabled: %s", body)
		}
	})
	t.Run("configuration without a secret cannot be enabled", func(t *testing.T) {
		call(t, "PUT", "/api/admin/settings", `{"client_id":"synthetic-app","registration_approval":true}`, session.CSRF, 400)
		body := call(t, "GET", "/api/setup", "", "", 200)
		if !bytes.Contains(body, []byte(`"discord_enabled":false`)) {
			t.Fatalf("incomplete Discord enabled: %s", body)
		}
	})
	t.Run("invalid and blank credentials cannot replace configuration", func(t *testing.T) {
		for _, data := range []string{
			`{"registration_approval":true}`,
			`{"client_id":null,"registration_approval":true}`,
			`{"client_id":"   ","client_secret":"fixture-secret","registration_approval":true}`,
			`{"client_id":"synthetic-app","client_secret":"   ","registration_approval":true}`,
		} {
			call(t, "PUT", "/api/admin/settings", data, session.CSRF, 400)
		}
	})

	t.Run("save retain replace and disable a secret", func(t *testing.T) {
		call(t, "PUT", "/api/admin/settings", `{"client_id":"synthetic-app","client_secret":"fixture-discord-secret","registration_approval":true}`, session.CSRF, 200)
		body := call(t, "GET", "/api/admin/settings", "", "", 200)
		if !bytes.Contains(body, []byte(`"secret_configured":true`)) {
			t.Fatalf("missing configured indicator: %s", body)
		}
		if !bytes.Contains(call(t, "GET", "/api/setup", "", "", 200), []byte(`"discord_enabled":true`)) {
			t.Fatal("complete configuration not enabled")
		}
		var ciphertext []byte
		if err := pool.QueryRow(context.Background(), `SELECT value FROM settings WHERE key='discord_client_secret'`).Scan(&ciphertext); err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(ciphertext, []byte("fixture-discord-secret")) {
			t.Fatal("secret stored in plaintext")
		}
		if value, err := auth.Open(s.Config.MasterKey, "discord_client_secret", ciphertext); err != nil || value != "fixture-discord-secret" {
			t.Fatal("stored secret cannot be recovered with external master key", err)
		}
		for _, secretField := range []string{"", `,"client_secret":""`} {
			call(t, "PUT", "/api/admin/settings", `{"client_id":"updated-app","registration_approval":true`+secretField+`}`, session.CSRF, 200)
			if value, err := s.Store.Get(context.Background(), "discord_client_secret"); err != nil || value != "fixture-discord-secret" {
				t.Fatal("blank/omitted secret replaced stored value", err)
			}
		}
		call(t, "PUT", "/api/admin/settings", `{"client_id":"updated-app","client_secret":"replacement-secret","registration_approval":true}`, session.CSRF, 200)
		if value, err := s.Store.Get(context.Background(), "discord_client_secret"); err != nil || value != "replacement-secret" {
			t.Fatal("secret not replaced", err)
		}
		call(t, "PUT", "/api/admin/settings", `{"client_id":"","registration_approval":true}`, session.CSRF, 200)
		body = call(t, "GET", "/api/admin/settings", "", "", 200)
		if !bytes.Contains(body, []byte(`"secret_configured":false`)) {
			t.Fatalf("disabled secret retained: %s", body)
		}
		if !bytes.Contains(call(t, "GET", "/api/setup", "", "", 200), []byte(`"discord_enabled":false`)) {
			t.Fatal("disabled configuration advertised")
		}
		call(t, "PUT", "/api/admin/settings", `{"client_id":"updated-app","registration_approval":true}`, session.CSRF, 400)
	})
	t.Run("failed authenticated decryption is refused by the settings API", func(t *testing.T) {
		for _, damage := range []string{"tamper", "wrong key"} {
			call(t, "PUT", "/api/admin/settings", `{"client_id":"synthetic-app","client_secret":"fixture-discord-secret","registration_approval":true}`, session.CSRF, 200)
			var encrypted []byte
			if err := pool.QueryRow(context.Background(), `SELECT value FROM settings WHERE key='discord_client_secret'`).Scan(&encrypted); err != nil {
				t.Fatal(err)
			}
			if damage == "tamper" {
				encrypted[len(encrypted)-1] ^= 1
			} else {
				var err error
				encrypted, err = auth.Seal(bytes.Repeat([]byte{2}, 32), "discord_client_secret", "fixture-discord-secret")
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := pool.Exec(context.Background(), `UPDATE settings SET value=$1 WHERE key='discord_client_secret'`, encrypted); err != nil {
				t.Fatal(err)
			}
			call(t, "GET", "/api/admin/settings", "", "", 503)
			call(t, "PUT", "/api/admin/settings", `{"client_id":"updated-app","registration_approval":true}`, session.CSRF, 503)
			call(t, "PUT", "/api/admin/settings", `{"client_id":"","registration_approval":true}`, session.CSRF, 200)
		}
	})
	t.Run("settings require admin session CSRF and configured origin", func(t *testing.T) {
		data := `{"client_id":"synthetic-app","client_secret":"fixture-discord-secret","registration_approval":true}`
		call(t, "PUT", "/api/admin/settings", data, "", 403)
		call(t, "PUT", "/api/admin/settings", data, "wrong-token", 403)
		req, _ := http.NewRequest("PUT", ts.URL+"/api/admin/settings", strings.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://other.example.invalid")
		req.Header.Set("X-CSRF-Token", session.CSRF)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 403 {
			t.Fatalf("cross-origin settings accepted: %d", resp.StatusCode)
		}
		call(t, "POST", "/api/auth/logout", "", session.CSRF, 200)
		call(t, "GET", "/api/admin/settings", "", "", 401)
		call(t, "PUT", "/api/admin/settings", data, session.CSRF, 401)
		call(t, "POST", "/api/auth/register", `{"username":"ordinary-user","password":"synthetic-password"}`, "", 201)
		if _, err := pool.Exec(context.Background(), `UPDATE users SET approved=true WHERE username='ordinary-user'`); err != nil {
			t.Fatal(err)
		}
		call(t, "POST", "/api/auth/login", `{"username":"ordinary-user","password":"synthetic-password"}`, "", 200)
		if err := json.Unmarshal(call(t, "GET", "/api/auth/me", "", "", 200), &session); err != nil {
			t.Fatal(err)
		}
		call(t, "GET", "/api/admin/settings", "", "", 403)
		call(t, "PUT", "/api/admin/settings", data, session.CSRF, 403)
	})
	if strings.Contains(logs.String(), "fixture-discord-secret") || strings.Contains(logs.String(), "replacement-secret") {
		t.Fatal("Discord secret appeared in logs")
	}

}
