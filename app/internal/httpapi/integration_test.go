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
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

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
		bad := fstest.MapFS{"99999_bad.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nTHIS IS NOT SQL;\n-- +goose Down\nSELECT 1;")}}
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
	t.Run("Discord OAuth", func(t *testing.T) { reset(t); testDiscordOAuth(t, pool) })
	t.Run("Discord settings", func(t *testing.T) {
		reset(t)
		testDiscordSettings(t, pool)
	})
	t.Run("folder sources", func(t *testing.T) {
		reset(t)
		testFolderSources(t, pool)
	})
	t.Run("source reconciliation", func(t *testing.T) {
		reset(t)
		testSourceReconciliation(t, pool)
	})
	t.Run("resumable uploads", func(t *testing.T) {
		reset(t)
		testResumableUploads(t, pool)
	})
	t.Run("clip library", func(t *testing.T) {
		reset(t)
		testClipLibrary(t, pool)
	})
	t.Run("job queue", func(t *testing.T) {
		reset(t)
		testJobQueue(t, pool)
	})
	t.Run("worker authentication and registration", func(t *testing.T) {
		reset(t)
		testWorkerRegistration(t, pool)
	})
	t.Run("HTTP contract and permissions", func(t *testing.T) {
		reset(t)
		clipRoot := t.TempDir()
		code, e := setup.Issue(ctx, pool)
		if e != nil {
			t.Fatal(e)
		}
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
		s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Test Receipts", ClipRoots: []string{clipRoot}, ProbePath: "ffprobe", StabilityWindow: time.Hour}, logger)
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
		workerToken := ""
		leaseToken := ""
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
			if workerToken != "" {
				req.Header.Set("Authorization", "Bearer "+workerToken)
			}
			if leaseToken != "" {
				req.Header.Set("X-Worker-Lease-Token", leaseToken)
			}
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
		call(anon, "GET", "/api/admin/sources", nil, "", 401)
		call(anon, "POST", "/api/auth/register", creds, "", 409)
		call(client, "POST", "/api/setup", map[string]string{"code": "wrong", "username": "administrator", "password": "synthetic-password"}, "", 409)
		call(client, "POST", "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", 201)
		call(client, "POST", "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", 409)
		noRoots := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Test Receipts"}, logger)
		noRootsTS := httptest.NewServer(noRoots.Handler())
		defer noRootsTS.Close()
		noRootsResponse, e := client.Get(noRootsTS.URL + "/api/admin/sources")
		if e != nil {
			t.Fatal(e)
		}
		var noRootsBody struct {
			AllowedRoots []string `json:"allowed_roots"`
		}
		if noRootsResponse.StatusCode != http.StatusOK || json.NewDecoder(noRootsResponse.Body).Decode(&noRootsBody) != nil {
			noRootsResponse.Body.Close()
			t.Fatalf("unconfigured source roots response: status %d", noRootsResponse.StatusCode)
		}
		noRootsResponse.Body.Close()
		if noRootsBody.AllowedRoots == nil || len(noRootsBody.AllowedRoots) != 0 {
			t.Fatalf("unconfigured source roots must be an empty array, got %#v", noRootsBody.AllowedRoots)
		}
		var me struct {
			CSRF string `json:"csrf_token"`
		}
		json.Unmarshal(call(client, "GET", "/api/auth/me", nil, "", 200), &me)
		call(client, "GET", "/api/clips", nil, "", 200)
		call(client, "GET", "/api/clips/999999", nil, "", 404)
		var adminID, contractClipID int64
		if e = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&adminID); e != nil {
			t.Fatal(e)
		}
		if e = pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'contract-synthetic.mp4',1,now(),now()) RETURNING id`, adminID).Scan(&contractClipID); e != nil {
			t.Fatal(e)
		}
		contractJobBody := call(client, "POST", "/api/admin/jobs", map[string]any{"clip_id": contractClipID, "step": "contract.synthetic", "input_version": "contract-v1"}, me.CSRF, 201)
		var contractJob struct {
			ID int64 `json:"id"`
		}
		if e = json.Unmarshal(contractJobBody, &contractJob); e != nil || contractJob.ID < 1 {
			t.Fatalf("contract worker job: %s %v", contractJobBody, e)
		}
		call(client, "GET", "/api/admin/jobs?status=pending", nil, "", 200)
		workerBody := call(client, "POST", "/api/admin/workers", map[string]any{"name": "contract-worker", "max_concurrency": 1}, me.CSRF, 201)
		var contractWorker struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		}
		if e = json.Unmarshal(workerBody, &contractWorker); e != nil || contractWorker.Token == "" {
			t.Fatalf("contract worker registration: %s %v", workerBody, e)
		}
		call(client, "GET", "/api/admin/workers", nil, "", 200)
		workerToken = contractWorker.Token
		call(client, "POST", "/api/worker/pulse", map[string]any{"capacity": 1}, "", 200)
		claimedBody := call(client, "POST", "/api/worker/jobs/claim", nil, "", 200)
		var claimedContractJob struct {
			LeaseToken string `json:"lease_token"`
			Job        struct {
				ID int64 `json:"id"`
			} `json:"job"`
		}
		if e = json.Unmarshal(claimedBody, &claimedContractJob); e != nil || claimedContractJob.Job.ID != contractJob.ID || claimedContractJob.LeaseToken == "" {
			t.Fatalf("contract worker claim: %s %v", claimedBody, e)
		}
		leaseToken = claimedContractJob.LeaseToken
		workerJobPath := "/api/worker/jobs/" + strconv64(contractJob.ID)
		call(client, "GET", workerJobPath+"/input", nil, "", 200)
		call(client, "GET", workerJobPath+"/media", nil, "", 404)
		call(client, "POST", workerJobPath+"/heartbeat", map[string]any{}, "", 200)
		call(client, "POST", workerJobPath+"/complete", map[string]any{"input_version": "contract-v1", "result": map[string]any{"fixture": true}}, "", 200)
		call(client, "POST", "/api/admin/jobs", map[string]any{"clip_id": contractClipID, "step": "contract.failure", "input_version": "failure-v1"}, me.CSRF, 201)
		failedClaim := call(client, "POST", "/api/worker/jobs/claim", nil, "", 200)
		var failedJob struct {
			LeaseToken string `json:"lease_token"`
			Job        struct {
				ID int64 `json:"id"`
			} `json:"job"`
		}
		if e = json.Unmarshal(failedClaim, &failedJob); e != nil || failedJob.Job.ID < 1 || failedJob.LeaseToken == "" {
			t.Fatalf("second worker claim: %s %v", failedClaim, e)
		}
		leaseToken = failedJob.LeaseToken
		call(client, "POST", "/api/worker/jobs/"+strconv64(failedJob.Job.ID)+"/fail", map[string]any{"error": "provider_unavailable"}, "", 200)
		workerToken = ""
		call(client, "DELETE", "/api/admin/workers/"+contractWorker.ID, nil, me.CSRF, 204)
		call(client, "POST", "/api/auth/logout", nil, "wrong", 403)
		call(client, "PUT", "/api/admin/settings", map[string]any{"client_id": "test-id", "client_secret": "secret-test-only", "registration_approval": true}, me.CSRF, 200)
		call(client, "GET", "/api/admin/sources", nil, "", 200)
		sourceBody := map[string]string{"name": "contract-fixture", "path": clipRoot}
		sourceResponse := call(client, "POST", "/api/admin/sources", sourceBody, me.CSRF, 201)
		var createdSource struct {
			ID int64 `json:"id"`
		}
		if e = json.Unmarshal(sourceResponse, &createdSource); e != nil || createdSource.ID == 0 {
			t.Fatalf("source response: %v %s", e, sourceResponse)
		}
		sourcePath := fmt.Sprintf("/api/admin/sources/%d", createdSource.ID)
		call(client, "GET", sourcePath+"/files", nil, "", 200)
		call(client, "POST", sourcePath+"/scan", map[string]any{}, me.CSRF, 202)
		call(client, "DELETE", sourcePath, nil, me.CSRF, 200)
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
				if strings.HasPrefix(path, "/api/uploads") || path == "/api/clips/{id}/media" {
					continue // Upload and binary range bodies are exercised by their dedicated HTTP suites.
				}
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
		suites := []string{"foundation.spec.ts", "local-auth.spec.ts", "registration.spec.ts", "discord-settings.spec.ts", "discord-oauth.spec.ts", "sources.spec.ts", "uploads.spec.ts", "library.spec.ts", "jobs.spec.ts", "workers.spec.ts"}
		if os.Getenv("RECEIPTS_E2E_FOUNDATION") == "1" {
			suites = suites[:1]
		}
		if os.Getenv("RECEIPTS_E2E_SOURCES") == "1" {
			suites = []string{"sources.spec.ts"}
		}
		if os.Getenv("RECEIPTS_E2E_UPLOADS") == "1" {
			suites = []string{"uploads.spec.ts"}
		}
		if os.Getenv("RECEIPTS_E2E_LIBRARY") == "1" {
			suites = []string{"library.spec.ts"}
		}
		if os.Getenv("RECEIPTS_E2E_JOBS") == "1" {
			suites = []string{"jobs.spec.ts"}
		}
		if os.Getenv("RECEIPTS_E2E_WORKERS") == "1" {
			suites = []string{"workers.spec.ts"}
		}
		for _, suite := range suites {
			t.Run("browser/"+suite, func(t *testing.T) {
				reset(t)
				code, e := setup.Issue(ctx, pool)
				if e != nil {
					t.Fatal(e)
				}
				config := config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts"}
				sourceRoot := ""
				if suite == "sources.spec.ts" {
					sourceRoot = t.TempDir()
					if e = os.WriteFile(filepath.Join(sourceRoot, "browser-fixture.mp4"), []byte("synthetic-browser-video"), 0o400); e != nil {
						t.Fatal(e)
					}
					config.ClipRoots = []string{sourceRoot}
					config.UploadRoot = t.TempDir()
					config.ProbePath = syntheticProbeFixture(t)
					config.StabilityWindow = 10 * time.Millisecond
					config.ReconcileInterval = 100 * time.Millisecond
				}
				if suite == "uploads.spec.ts" || suite == "library.spec.ts" || suite == "jobs.spec.ts" {
					config.UploadRoot = t.TempDir()
					config.ProbePath = syntheticProbeFixture(t)
					config.UploadMaxBytes = 1 << 20
					config.UploadChunkBytes = 16 * 1024
					config.UploadSessionTTL = time.Hour
				}
				s := New(pool, config, slog.New(slog.NewJSONHandler(io.Discard, nil)))
				if suite == "sources.spec.ts" || suite == "uploads.spec.ts" || suite == "library.spec.ts" || suite == "jobs.spec.ts" {
					if e = s.Start(ctx); e != nil {
						t.Fatal(e)
					}
					defer s.Close()
				}
				if suite == "discord-oauth.spec.ts" {
					browserDiscord(t, s)
				}
				ts := httptest.NewUnstartedServer(s.Handler())
				s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
				ts.Start()
				defer ts.Close()
				args := []string{"run", "test:e2e", "--", suite}
				cmd := exec.Command("npm", args...)
				cmd.Dir = "../../../web"
				cmd.Env = append(os.Environ(), "RECEIPTS_TEST_URL="+ts.URL, "RECEIPTS_TEST_CODE="+code, "RECEIPTS_TEST_SOURCE_ROOT="+sourceRoot, "RECEIPTS_TEST_UPLOAD_CHUNK=16384")
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
