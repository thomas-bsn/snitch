//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"receipts/internal/config"
	"receipts/internal/setup"
	"receipts/migrations"
)

const pgImage = "pgvector/pgvector:0.8.7-pg18-bookworm@sha256:2358fcba361ed2233a5ed81b5fe4ca779ccb304120ce531a3bf51c0ed7e2bc11"

func TestFoundation(t *testing.T) {
	ctx := context.Background()
	pg, err := postgres.Run(ctx, pgImage, postgres.WithDatabase("receipts"), postgres.WithUsername("receipts"), postgres.WithPassword("synthetic-only"), postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	provider, err := goose.NewProvider(database.DialectPostgres, db, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	replay, err := provider.Up(ctx)
	if err != nil || len(replay) != 0 {
		t.Fatal("migration replay", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var pgVersion, vectorVersion string
	if err = pool.QueryRow(ctx, `SELECT version(),(SELECT extversion FROM pg_extension WHERE extname='vector')`).Scan(&pgVersion, &vectorVersion); err != nil {
		t.Fatal(err)
	}
	t.Log("database:", pgVersion, "vector:", vectorVersion)
	t.Run("failed migration is rejected", func(t *testing.T) {
		bad := fstest.MapFS{"00002_bad.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nTHIS IS NOT SQL;\n-- +goose Down\nSELECT 1;")}}
		p, e := goose.NewProvider(database.DialectPostgres, db, bad)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = p.Up(ctx); e == nil {
			t.Fatal("bad migration accepted")
		}
	})
	reset := func(t *testing.T) {
		t.Helper()
		_, e := pool.Exec(ctx, `TRUNCATE sessions,oauth_states,settings,users RESTART IDENTITY CASCADE; UPDATE installation SET initialized=false,bootstrap_hash=NULL,bootstrap_expires=NULL,registration_approval=true`)
		if e != nil {
			t.Fatal(e)
		}
	}
	t.Run("optional embeddings do not block foundation", func(t *testing.T) {
		provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		endpoint := provider.URL
		provider.Close() // A configured but unreachable service must not block the foundation.
		for _, endpoint := range []string{"", endpoint} {
			s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Fixture", EmbeddingsURL: endpoint, EmbeddingsModel: "fixture-only"}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			ts := httptest.NewServer(s.Handler())
			for path, want := range map[string]string{"/api/health/live": `"status":"ok"`, "/api/health/ready": `"status":"degraded"`, "/api/setup": `"initialized":false`} {
				resp, err := ts.Client().Get(ts.URL + path)
				if err != nil {
					t.Fatal(err)
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				if resp.StatusCode != 200 || !bytes.Contains(body, []byte(want)) {
					t.Fatalf("%s with embeddings %q: %d %s", path, endpoint, resp.StatusCode, body)
				}
			}
			ts.Close()
		}
	})
	t.Run("local authentication security", func(t *testing.T) {
		reset(t)
		testLocalAuthentication(t, pool)
	})
	t.Run("registration and approval", func(t *testing.T) {
		reset(t)
		testRegistration(t, pool)
	})
	t.Run("Discord settings", func(t *testing.T) {
		reset(t)
		testDiscordSettings(t, pool)
	})
	t.Run("HTTP contract and permissions", func(t *testing.T) {
		reset(t)
		code, e := setup.Issue(ctx, pool)
		if e != nil {
			t.Fatal(e)
		}
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
		s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Test Receipts"}, logger)
		// High test allowance isolates security tests from test-suite request volume.
		ts := httptest.NewUnstartedServer(s.Handler())
		s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
		ts.Start()
		defer ts.Close()
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		anon := &http.Client{CheckRedirect: client.CheckRedirect}
		spec, e := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
		if e != nil {
			t.Fatal(e)
		}
		if e = spec.Validate(ctx); e != nil {
			t.Fatal(e)
		}
		spec.Servers = nil
		router, e := legacy.NewRouter(spec)
		if e != nil {
			t.Fatal(e)
		}
		seen := map[string]bool{}
		call := func(c *http.Client, method, path string, body any, csrf string, want int) []byte {
			t.Helper()
			s.rates.Clear()
			var data []byte
			if body != nil {
				data, _ = json.Marshal(body)
			}
			req, e := http.NewRequest(method, ts.URL+path, bytes.NewReader(data))
			if e != nil {
				t.Fatal(e)
			}
			req.Header.Set("Origin", ts.URL)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", csrf)
			resp, e := c.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != want {
				t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, b)
			}
			route, params, e := router.FindRoute(req)
			if e != nil {
				t.Fatal(e)
			}
			req.Body = io.NopCloser(bytes.NewReader(data))
			input := &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
			if e = openapi3filter.ValidateRequest(ctx, input); e != nil {
				t.Fatalf("request contract %s %s: %v", method, path, e)
			}
			out := &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: resp.StatusCode, Header: resp.Header, Body: io.NopCloser(bytes.NewReader(b))}
			if e = openapi3filter.ValidateResponse(ctx, out); e != nil {
				t.Fatalf("contract %s %s: %v", method, path, e)
			}
			seen[method+" "+route.Path] = true
			return b
		}
		creds := map[string]string{"username": "administrator", "password": "synthetic-password"}
		call(anon, "GET", "/api/health/live", nil, "", 200)
		call(anon, "GET", "/api/health/ready", nil, "", 200)
		call(anon, "GET", "/api/setup", nil, "", 200)
		call(anon, "GET", "/api/admin/settings", nil, "", 401)
		call(anon, "POST", "/api/auth/register", creds, "", 409)
		call(client, "POST", "/api/setup", map[string]string{"code": "wrong", "username": "administrator", "password": "synthetic-password"}, "", 409)
		call(client, "POST", "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", 201)
		call(client, "POST", "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", 409)
		var me struct {
			CSRF string `json:"csrf_token"`
		}
		json.Unmarshal(call(client, "GET", "/api/auth/me", nil, "", 200), &me)
		call(client, "POST", "/api/auth/logout", nil, "wrong", 403)
		call(client, "PUT", "/api/admin/settings", map[string]any{"client_id": "test-id", "client_secret": "secret-test-only", "registration_approval": true}, me.CSRF, 200)
		b := call(client, "GET", "/api/admin/settings", nil, "", 200)
		if bytes.Contains(b, []byte("secret-test-only")) {
			t.Fatal("secret leaked")
		}
		var stored []byte
		if e = pool.QueryRow(ctx, `SELECT value FROM settings WHERE key='discord_client_secret'`).Scan(&stored); e != nil || bytes.Contains(stored, []byte("secret-test-only")) {
			t.Fatal("unencrypted setting", e)
		}
		call(anon, "POST", "/api/auth/register", map[string]string{"username": "player", "password": "synthetic-password"}, "", 201)
		call(anon, "POST", "/api/auth/login", map[string]string{"username": "player", "password": "synthetic-password"}, "", 401)
		var users []struct {
			ID int64 `json:"id"`
		}
		json.Unmarshal(call(client, "GET", "/api/admin/users", nil, "", 200), &users)
		if len(users) != 1 {
			t.Fatal("pending", string(b))
		}
		call(client, "POST", fmt.Sprintf("/api/admin/users/%d/approve", users[0].ID), nil, me.CSRF, 200)
		userJar, _ := cookiejar.New(nil)
		user := &http.Client{Jar: userJar, CheckRedirect: client.CheckRedirect}
		call(user, "POST", "/api/auth/login", map[string]string{"username": "player", "password": "synthetic-password"}, "", 200)
		call(user, "GET", "/api/admin/settings", nil, "", 403)
		call(user, "GET", "/api/admin/users", nil, "", 403)
		call(client, "POST", "/api/admin/users/99999/approve", nil, me.CSRF, 404)
		call(client, "POST", "/api/auth/logout", nil, me.CSRF, 200)
		call(client, "GET", "/api/auth/me", nil, "", 401)
		call(client, "POST", "/api/auth/login", creds, "", 200)
		json.Unmarshal(call(client, "GET", "/api/auth/me", nil, "", 200), &me)
		// Simulated Discord: verify scope, state, browser binding, replay, approval and explicit linking.
		mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/token" {
				r.ParseForm()
				if r.Form.Get("client_secret") != "secret-test-only" {
					t.Error("wrong OAuth secret")
				}
				reply(w, 200, map[string]string{"access_token": "mock-token"})
				return
			}
			if r.Header.Get("Authorization") != "Bearer mock-token" {
				t.Error("missing OAuth bearer")
			}
			reply(w, 200, map[string]string{"id": "123456"})
		}))
		defer mock.Close()
		s.Config.DiscordAuthorizeURL = mock.URL + "/authorize"
		s.Config.DiscordTokenURL = mock.URL + "/token"
		s.Config.DiscordUserURL = mock.URL + "/me"
		var redirect struct {
			URL string `json:"url"`
		}
		start := func(c *http.Client, path, csrf string) string {
			json.Unmarshal(call(c, "POST", path, nil, csrf, 200), &redirect)
			req, _ := http.NewRequest("GET", redirect.URL, nil)
			if req.URL.Query().Get("scope") != "identify" {
				t.Fatal("OAuth scope")
			}
			return req.URL.Query().Get("state")
		}
		state := start(user, "/api/auth/discord/start", "")
		call(anon, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 400)
		call(user, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 303)
		call(user, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 400)
		var discordUID int64
		if e = pool.QueryRow(ctx, `SELECT id FROM users WHERE discord_id='123456' AND NOT approved`).Scan(&discordUID); e != nil {
			t.Fatal("Discord approval", e)
		}
		// Linking never merges an identity belonging to another account.
		state = start(client, "/api/auth/discord/link", me.CSRF)
		call(client, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 409)
		_, e = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, discordUID)
		if e != nil {
			t.Fatal(e)
		}
		state = start(client, "/api/auth/discord/link", me.CSRF)
		call(client, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 303)
		call(client, "POST", "/api/auth/logout", nil, me.CSRF, 200)
		state = start(client, "/api/auth/discord/start", "")
		call(client, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 303)
		call(client, "GET", "/api/auth/me", nil, "", 200)
		for path, item := range spec.Paths.Map() {
			for method := range item.Operations() {
				if !seen[method+" "+path] {
					t.Errorf("untested contract operation %s %s", method, path)
				}
			}
		}
		// Origin and rate limiter are tested separately from the contract helper.
		req, _ := http.NewRequest("POST", ts.URL+"/api/auth/login", strings.NewReader(`{}`))
		req.Header.Set("Origin", "https://attacker.invalid")
		resp, e := anon.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		resp.Body.Close()
		if resp.StatusCode != 403 {
			t.Fatal("origin check")
		}
		s.rates.Clear()
		throttled := false
		for i := 0; i < 12; i++ {
			req, _ = http.NewRequest("POST", ts.URL+"/api/auth/login", strings.NewReader(`{}`))
			req.Header.Set("Origin", ts.URL)
			req.Header.Set("Content-Type", "application/json")
			resp, e = anon.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			resp.Body.Close()
			if resp.StatusCode == 429 {
				throttled = true
			}
		}
		if !throttled {
			t.Fatal("no rate limit")
		}
		resp, e = client.Get(ts.URL + "/")
		if e != nil {
			t.Fatal(e)
		}
		html, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !bytes.Contains(html, []byte("<div id=\"root\"")) {
			t.Fatal("embedded SPA missing")
		}
	})
	if os.Getenv("RECEIPTS_E2E") == "1" {
		suites := []string{"foundation.spec.ts", "local-auth.spec.ts", "registration.spec.ts", "discord-settings.spec.ts"}
		if os.Getenv("RECEIPTS_E2E_FOUNDATION") == "1" {
			suites = suites[:1]
		}
		for _, suite := range suites {
			t.Run("browser/"+suite, func(t *testing.T) {
				reset(t)
				code, e := setup.Issue(ctx, pool)
				if e != nil {
					t.Fatal(e)
				}
				s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts"}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
				ts := httptest.NewUnstartedServer(s.Handler())
				s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
				ts.Start()
				defer ts.Close()
				args := []string{"run", "test:e2e", "--", suite}
				cmd := exec.Command("npm", args...)
				cmd.Dir = "../../../web"
				cmd.Env = append(os.Environ(), "RECEIPTS_TEST_URL="+ts.URL, "RECEIPTS_TEST_CODE="+code)
				out, e := cmd.CombinedOutput()
				t.Log(string(out))
				if e != nil {
					t.Fatal(e)
				}
			})
		}
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	t.Logf("Go test process heap in use %.1f MiB (NOT production RSS)", float64(mem.HeapInuse)/1024/1024)
	stats, e := exec.Command("docker", "stats", "--no-stream", "--format", "{{.MemUsage}}", pg.GetContainerID()).CombinedOutput()
	if e == nil {
		t.Log("PostgreSQL container at test end:", string(stats))
	}
}
