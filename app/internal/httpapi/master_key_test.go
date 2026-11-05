//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/secrets"
	"snitch/internal/setup"
)

// testMasterKey covers the key a deployment generates for itself: what the settings API says about it, who can
// export it, and what happens to the encrypted secrets when the key is absent, different or restored.
func testMasterKey(t *testing.T, pool *pgxpool.Pool) {
	var logs bytes.Buffer
	spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec.Servers = nil
	router, err := legacy.NewRouter(spec)
	if err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// start runs one instance of the application with the key a start would have resolved.
	type instance struct {
		url  string
		call func(t *testing.T, method, path, data, csrf string, want int) []byte
	}
	start := func(t *testing.T, secretsDir, explicit string) (instance, config.Config) {
		t.Helper()
		key, origin, err := secrets.ResolveMasterKey(secretsDir, explicit)
		if err != nil {
			t.Fatal(err)
		}
		s := New(pool, config.Config{MasterKey: key, MasterKeyOrigin: origin}, slog.New(slog.NewJSONHandler(&logs, nil)))
		ts := httptest.NewUnstartedServer(s.Handler())
		ts.Start()
		t.Cleanup(ts.Close)
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
			if resp.StatusCode != want {
				t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, body)
			}
			route, params, err := router.FindRoute(req)
			if err != nil {
				t.Fatal(err)
			}
			input := &openapi3filter.RequestValidationInput{Request: req, Route: route, PathParams: params, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
			if err := openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: resp.StatusCode, Header: resp.Header, Body: io.NopCloser(bytes.NewReader(body))}); err != nil {
				t.Fatal("response contract", method, path, err)
			}
			return body
		}
		return instance{url: ts.URL, call: call}, s.Config
	}
	type settingsView struct {
		ClientID          string   `json:"client_id"`
		SecretConfigured  bool     `json:"secret_configured"`
		MasterKeyOrigin   string   `json:"master_key_origin"`
		UnreadableSecrets []string `json:"unreadable_secrets"`
	}
	readSettings := func(t *testing.T, app instance) settingsView {
		t.Helper()
		var view settingsView
		if err := json.Unmarshal(app.call(t, "GET", "/api/admin/settings", "", "", 200), &view); err != nil {
			t.Fatal(err)
		}
		return view
	}
	csrfOf := func(t *testing.T, app instance) string {
		t.Helper()
		var me struct {
			CSRF string `json:"csrf_token"`
		}
		if err := json.Unmarshal(app.call(t, "GET", "/api/auth/me", "", "", 200), &me); err != nil {
			t.Fatal(err)
		}
		return me.CSRF
	}

	dir := t.TempDir()
	app, cfg := start(t, dir, "")
	if cfg.MasterKeyOrigin != secrets.MasterKeyGenerated {
		t.Fatal("the deployment must start without any key configured")
	}
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	app.call(t, "POST", "/api/setup", `{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`, "", 201)
	csrf := csrfOf(t, app)
	app.call(t, "PUT", "/api/admin/settings", `{"client_id":"synthetic-app","client_secret":"fixture-discord-secret","registration_approval":true}`, csrf, 200)

	t.Run("secrets are encrypted with the generated key, which is not in the database", func(t *testing.T) {
		view := readSettings(t, app)
		if view.MasterKeyOrigin != "generated" || len(view.UnreadableSecrets) != 0 || !view.SecretConfigured {
			t.Fatalf("unexpected settings: %+v", view)
		}
		// A dump of the whole database, as a backup would take it, must not contain the key in any column.
		raw := base64.StdEncoding.EncodeToString(cfg.MasterKey)
		rows, err := pool.Query(context.Background(), `SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'`)
		if err != nil {
			t.Fatal(err)
		}
		var tables []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatal(err)
			}
			tables = append(tables, name)
		}
		rows.Close()
		for _, table := range tables {
			var found bool
			query := `SELECT EXISTS (SELECT 1 FROM "` + table + `" t WHERE t::text LIKE '%' || $1 || '%' OR t::text LIKE '%' || $2 || '%')`
			if err := pool.QueryRow(context.Background(), query, raw, "\\x"+hex.EncodeToString(cfg.MasterKey)).Scan(&found); err != nil || found {
				t.Fatalf("the key is stored in the database (%s): %v", table, err)
			}
		}
	})

	t.Run("the administrator exports the key with its warning", func(t *testing.T) {
		body := app.call(t, "POST", "/api/admin/master-key/export", "", csrf, 200)
		var export struct {
			MasterKey string `json:"master_key"`
			Origin    string `json:"origin"`
			Warning   string `json:"warning"`
		}
		if err := json.Unmarshal(body, &export); err != nil {
			t.Fatal(err)
		}
		if export.MasterKey != base64.StdEncoding.EncodeToString(cfg.MasterKey) || export.Origin != "generated" || export.Warning != "master_key_not_replayable" {
			t.Fatalf("unexpected export: %+v", export)
		}
		app.call(t, "POST", "/api/admin/master-key/export", "", "", 403) // CSRF is required
		if strings.Contains(logs.String(), export.MasterKey) {
			t.Fatal("the key appeared in the logs")
		}
		if !strings.Contains(logs.String(), "master key exported") {
			t.Fatal("the export is not traced")
		}
	})

	t.Run("a key that is absent or different leaves the application running and lists the secrets to re-enter", func(t *testing.T) {
		exported := base64.StdEncoding.EncodeToString(cfg.MasterKey)
		for name, replacement := range map[string]struct{ dir, explicit string }{
			"volume lost":  {t.TempDir(), ""},
			"other key":    {t.TempDir(), base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))},
			"damaged file": {t.TempDir(), ""},
		} {
			t.Log("scenario:", name)
			if name == "damaged file" {
				if err := os.WriteFile(filepath.Join(replacement.dir, secrets.MasterKeyFile), []byte("truncated"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			other, _ := start(t, replacement.dir, replacement.explicit)
			view := readSettings(t, other)
			if view.ClientID != "" || view.SecretConfigured || len(view.UnreadableSecrets) != 2 || view.UnreadableSecrets[0] != "discord_client_id" || view.UnreadableSecrets[1] != "discord_client_secret" {
				t.Fatalf("%s: unreadable secrets not reported: %+v", name, view)
			}
			if !bytes.Contains(other.call(t, "GET", "/api/setup", "", "", 200), []byte(`"discord_enabled":false`)) {
				t.Fatal("an unreadable Discord configuration is advertised")
			}
			other.call(t, "POST", "/api/auth/discord/start", `{}`, "", 409)
			// Entering only the application identifier cannot rebuild a configuration without its secret.
			otherCSRF := csrfOf(t, other)
			other.call(t, "PUT", "/api/admin/settings", `{"client_id":"synthetic-app","registration_approval":true}`, otherCSRF, 400)
			other.call(t, "PUT", "/api/admin/settings", `{"client_id":"re-entered-app","client_secret":"re-entered-secret","registration_approval":true}`, otherCSRF, 200)
			view = readSettings(t, other)
			if view.ClientID != "re-entered-app" || !view.SecretConfigured || len(view.UnreadableSecrets) != 0 {
				t.Fatalf("%s: re-entering did not work: %+v", name, view)
			}
			// The re-entered values now belong to the replacement key: going back to the original key lists them as
			// unreadable again, and re-entering them there restores the state for the next scenario.
			original, _ := start(t, dir, exported)
			if view = readSettings(t, original); len(view.UnreadableSecrets) != 2 || view.MasterKeyOrigin != "provided" {
				t.Fatalf("%s: unexpected state after restoring the exported key: %+v", name, view)
			}
			original.call(t, "PUT", "/api/admin/settings", `{"client_id":"synthetic-app","client_secret":"fixture-discord-secret","registration_approval":true}`, csrfOf(t, original), 200)
		}
	})

	t.Run("the exported key restores the secrets it protected", func(t *testing.T) {
		exported := app.call(t, "POST", "/api/admin/master-key/export", "", csrf, 200)
		var export struct {
			MasterKey string `json:"master_key"`
		}
		if err := json.Unmarshal(exported, &export); err != nil {
			t.Fatal(err)
		}
		lost, _ := start(t, t.TempDir(), "")
		if view := readSettings(t, lost); len(view.UnreadableSecrets) != 2 {
			t.Fatalf("a lost volume must report the secrets: %+v", view)
		}
		restored, _ := start(t, t.TempDir(), export.MasterKey)
		if view := readSettings(t, restored); len(view.UnreadableSecrets) != 0 || !view.SecretConfigured || view.ClientID != "synthetic-app" {
			t.Fatalf("the exported key did not restore the secrets: %+v", view)
		}
	})

	t.Run("an explicit key is announced as provided", func(t *testing.T) {
		explicit, _ := start(t, t.TempDir(), base64.StdEncoding.EncodeToString(cfg.MasterKey))
		if view := readSettings(t, explicit); view.MasterKeyOrigin != "provided" || len(view.UnreadableSecrets) != 0 {
			t.Fatalf("unexpected settings: %+v", view)
		}
	})

	t.Run("export is refused to anyone who is not an administrator", func(t *testing.T) {
		app.call(t, "POST", "/api/auth/logout", "", csrf, 200)
		app.call(t, "POST", "/api/admin/master-key/export", "", "", 401)
		app.call(t, "POST", "/api/auth/register", `{"username":"ordinary-user","password":"synthetic-password"}`, "", 201)
		if _, err := pool.Exec(context.Background(), `UPDATE users SET approved=true WHERE username='ordinary-user'`); err != nil {
			t.Fatal(err)
		}
		app.call(t, "POST", "/api/auth/login", `{"username":"ordinary-user","password":"synthetic-password"}`, "", 200)
		ordinary := csrfOf(t, app)
		body := app.call(t, "POST", "/api/admin/master-key/export", "", ordinary, 403)
		if bytes.Contains(body, []byte(base64.StdEncoding.EncodeToString(cfg.MasterKey))) {
			t.Fatal("the key leaked to an ordinary user")
		}
		app.call(t, "GET", "/api/admin/settings", "", "", 403)
	})
}
