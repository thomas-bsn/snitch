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
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

// testSetupProgress covers the guided installation: the installation code checked before it is used, the
// progress kept on the server, what the progress reports from the real state, and that nothing starts by itself.
func testSetupProgress(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)
	server.Config.PublicURL = ts.URL
	newClient := func() *http.Client {
		jar, err := cookiejar.New(nil)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Client{Jar: jar}
	}
	call := func(client *http.Client, csrf, method, path, body string, want int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", ts.URL)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, out)
		}
		return out
	}
	csrfOf := func(client *http.Client) string {
		t.Helper()
		var me struct {
			CSRF string `json:"csrf_token"`
		}
		if err := json.Unmarshal(call(client, "", "GET", "/api/auth/me", "", 200), &me); err != nil {
			t.Fatal(err)
		}
		return me.CSRF
	}
	type progress struct {
		Steps []struct {
			Step  string `json:"step"`
			State string `json:"state"`
		} `json:"steps"`
		Dismissed bool `json:"dismissed"`
	}
	read := func(out []byte) (map[string]string, bool) {
		t.Helper()
		var p progress
		if err := json.Unmarshal(out, &p); err != nil {
			t.Fatal(err)
		}
		states := map[string]string{}
		for _, s := range p.Steps {
			states[s.Step] = s.State
		}
		if len(p.Steps) != 6 || p.Steps[0].Step != "location" || p.Steps[4].Step != "huggingface" || p.Steps[5].Step != "import" {
			t.Fatalf("steps %+v", p.Steps)
		}
		return states, p.Dismissed
	}
	owner := newClient()
	creds := `"username":"administrator","password":"synthetic-password-123"`

	t.Run("a wrong or malformed code is refused before any account exists", func(t *testing.T) {
		out := call(owner, "", "POST", "/api/setup/code", `{"code":"not-the-code"}`, 409)
		if !bytes.Contains(out, []byte("setup_closed_or_invalid_code")) {
			t.Fatalf("got %s", out)
		}
		call(owner, "", "POST", "/api/setup/code", `{}`, 400)
		call(owner, "", "POST", "/api/setup/code", `{"code":"x","other":1}`, 400)
	})
	t.Run("an expired code is refused", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE installation SET bootstrap_expires=now()-interval '1 minute'`); err != nil {
			t.Fatal(err)
		}
		call(owner, "", "POST", "/api/setup/code", `{"code":"`+code+`"}`, 409)
		call(owner, "", "POST", "/api/setup", `{"code":"`+code+`",`+creds+`}`, 409)
		if _, err := pool.Exec(ctx, `UPDATE installation SET bootstrap_expires=now()+interval '10 minutes'`); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("the right code is accepted, any number of times, and is not used up", func(t *testing.T) {
		for range 3 {
			call(owner, "", "POST", "/api/setup/code", `{"code":"`+code+`"}`, 200)
		}
		call(owner, "", "POST", "/api/setup", `{"code":"`+code+`",`+creds+`}`, 201)
	})
	t.Run("once installed, the code is refused", func(t *testing.T) {
		call(newClient(), "", "POST", "/api/setup/code", `{"code":"`+code+`"}`, 409)
	})
	ownerCSRF := csrfOf(owner)

	t.Run("the account the installation creates is the Super-administrator", func(t *testing.T) {
		var role string
		if err := pool.QueryRow(ctx, `SELECT role FROM users WHERE username='administrator'`).Scan(&role); err != nil || role != "super_admin" {
			t.Fatalf("role %q %v", role, err)
		}
	})
	t.Run("nothing starts by itself: no job, no folder, no worker, no clip", func(t *testing.T) {
		var n int
		if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM jobs)+(SELECT count(*) FROM clip_sources)+(SELECT count(*) FROM workers)+(SELECT count(*) FROM clips)`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%d rows created without any action: %v", n, err)
		}
	})
	t.Run("the instance has no configurable name", func(t *testing.T) {
		if out := call(newClient(), "", "GET", "/api/setup", "", 200); bytes.Contains(out, []byte("display_name")) {
			t.Fatalf("setup status still carries a name: %s", out)
		}
	})
	t.Run("progress is reserved to administrators", func(t *testing.T) {
		call(newClient(), "", "GET", "/api/admin/setup-progress", "", 401)
		call(newClient(), "", "PUT", "/api/admin/setup-progress", `{"step":"discord","state":"skipped"}`, 401)
		call(newClient(), "", "POST", "/api/admin/setup-progress/dismiss", "", 401)
		call(newClient(), "", "POST", "/api/auth/register", `{"username":"plain-member","password":"synthetic-password-456"}`, 201)
		var id int64
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='plain-member'`).Scan(&id); err != nil {
			t.Fatal(err)
		}
		call(owner, ownerCSRF, "POST", "/api/admin/users/"+strconv.FormatInt(id, 10)+"/approve", "", 200)
		member := newClient()
		call(member, "", "POST", "/api/auth/login", `{"username":"plain-member","password":"synthetic-password-456"}`, 200)
		memberCSRF := csrfOf(member)
		call(member, "", "GET", "/api/admin/setup-progress", "", 403)
		call(member, memberCSRF, "PUT", "/api/admin/setup-progress", `{"step":"discord","state":"skipped"}`, 403)
		call(member, memberCSRF, "POST", "/api/admin/setup-progress/dismiss", "", 403)
	})
	t.Run("a fresh installation has every step to do and a banner to show", func(t *testing.T) {
		states, dismissed := read(call(owner, "", "GET", "/api/admin/setup-progress", "", 200))
		for step, state := range states {
			if state != "todo" {
				t.Errorf("%s is %s", step, state)
			}
		}
		if dismissed {
			t.Fatal("banner already dismissed")
		}
	})
	t.Run("a step can be skipped, and is remembered by the server", func(t *testing.T) {
		states, _ := read(call(owner, ownerCSRF, "PUT", "/api/admin/setup-progress", `{"step":"discord","state":"skipped"}`, 200))
		if states["discord"] != "skipped" || states["worker"] != "todo" {
			t.Fatalf("%v", states)
		}
		again := newClient()
		call(again, "", "POST", "/api/auth/login", `{`+creds+`}`, 200)
		states, _ = read(call(again, "", "GET", "/api/admin/setup-progress", "", 200))
		if states["discord"] != "skipped" {
			t.Fatalf("progress lost on a new session: %v", states)
		}
	})
	t.Run("an unknown step or state is refused and a write needs the CSRF token", func(t *testing.T) {
		for _, body := range []string{`{"step":"naming","state":"done"}`, `{"step":"discord","state":"todo"}`, `{"step":"discord"}`, `{"state":"done"}`, `{"step":"discord","state":"done","x":1}`} {
			call(owner, ownerCSRF, "PUT", "/api/admin/setup-progress", body, 400)
		}
		call(owner, "", "PUT", "/api/admin/setup-progress", `{"step":"import","state":"skipped"}`, 403)
		call(owner, "", "POST", "/api/admin/setup-progress/dismiss", "", 403)
	})
	t.Run("a step marked done but not configured is reported as skipped", func(t *testing.T) {
		states, _ := read(call(owner, ownerCSRF, "PUT", "/api/admin/setup-progress", `{"step":"import","state":"done"}`, 200))
		if states["import"] != "skipped" {
			t.Fatalf("%v", states)
		}
	})
	t.Run("the summary follows the real state: configuring a skipped step makes it done", func(t *testing.T) {
		call(owner, ownerCSRF, "PUT", "/api/admin/settings", `{"client_id":"123456789","client_secret":"synthetic-secret","registration_approval":true}`, 200)
		if _, err := pool.Exec(ctx, `INSERT INTO workers(id,name,token_hash) VALUES('wk_syntheticworker0000001','Synthetic worker',decode(repeat('ab',32),'hex'))`); err != nil {
			t.Fatal(err)
		}
		states, _ := read(call(owner, "", "GET", "/api/admin/setup-progress", "", 200))
		if states["discord"] != "done" || states["worker"] != "done" || states["semantic_search"] != "todo" {
			t.Fatalf("%v", states)
		}
		if _, err := pool.Exec(ctx, `UPDATE workers SET revoked_at=now()`); err != nil {
			t.Fatal(err)
		}
		states, _ = read(call(owner, "", "GET", "/api/admin/setup-progress", "", 200))
		if states["worker"] != "todo" {
			t.Fatalf("a revoked Worker counts as configured: %v", states)
		}
	})
	t.Run("the banner can be closed for good", func(t *testing.T) {
		if _, dismissed := read(call(owner, ownerCSRF, "POST", "/api/admin/setup-progress/dismiss", "", 200)); !dismissed {
			t.Fatal("not dismissed")
		}
		again := newClient()
		call(again, "", "POST", "/api/auth/login", `{`+creds+`}`, 200)
		if _, dismissed := read(call(again, "", "GET", "/api/admin/setup-progress", "", 200)); !dismissed {
			t.Fatal("the banner came back")
		}
	})
}
