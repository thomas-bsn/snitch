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
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"snitch/internal/auth"
	"snitch/internal/config"
	"snitch/internal/setup"
)

// testRoles covers the three roles: who the installation makes Super-administrator, who may do what, what
// protects the Super-administrator, and what an update does to an existing installation.
func testRoles(t *testing.T, pool *pgxpool.Pool, provider *goose.Provider) {
	ctx := context.Background()
	spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec.Servers = nil
	router, err := legacy.NewRouter(spec)
	if err != nil {
		t.Fatal(err)
	}
	s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewUnstartedServer(s.Handler())
	s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()

	// person is one browser: its own cookies, and the CSRF token of its session.
	type person struct {
		client *http.Client
		csrf   string
	}
	newPerson := func() *person {
		jar, _ := cookiejar.New(nil)
		return &person{client: &http.Client{Jar: jar}}
	}
	call := func(t *testing.T, p *person, method, path, data string, want int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", ts.URL)
		req.Header.Set("X-CSRF-Token", p.csrf)
		resp, err := p.client.Do(req)
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
		if err := openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: resp.StatusCode, Header: resp.Header, Body: io.NopCloser(bytes.NewReader(body))}); err != nil {
			t.Fatal("response contract", method, path, err)
		}
		return body
	}
	// role reads the role the API reports for this session (and keeps the session's CSRF token).
	role := func(t *testing.T, p *person) string {
		t.Helper()
		var me struct {
			User struct {
				Role string `json:"role"`
			} `json:"user"`
			CSRF string `json:"csrf_token"`
		}
		if err := json.Unmarshal(call(t, p, "GET", "/api/auth/me", "", 200), &me); err != nil {
			t.Fatal(err)
		}
		p.csrf = me.CSRF
		return me.User.Role
	}
	login := func(t *testing.T, username string) *person {
		t.Helper()
		p := newPerson()
		call(t, p, "POST", "/api/auth/login", `{"username":"`+username+`","password":"synthetic-password"}`, 200)
		role(t, p)
		return p
	}
	errorCode := func(body []byte) string {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &e)
		return e.Error
	}
	addMember := func(t *testing.T, username, memberRole string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO users(username,password_hash,role,approved) VALUES($1,$2,$3,true)`, username, auth.Password("synthetic-password"), memberRole); err != nil {
			t.Fatal(err)
		}
	}
	superAdminCount := func(t *testing.T) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE role='super_admin'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	owner := newPerson()
	call(t, owner, "POST", "/api/setup", `{"code":"`+code+`","username":"owner","password":"synthetic-password"}`, 201)
	addMember(t, "operator", "admin")
	addMember(t, "member", "user")
	operator, member := login(t, "operator"), login(t, "member")

	t.Run("the installation creates the Super-administrator, and only one", func(t *testing.T) {
		if got := role(t, owner); got != "super_admin" || superAdminCount(t) != 1 {
			t.Fatalf("role %q, super-administrators %d", got, superAdminCount(t))
		}
		if got := role(t, operator); got != "admin" {
			t.Fatalf("operator role %q", got)
		}
		if got := role(t, member); got != "user" {
			t.Fatalf("member role %q", got)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO users(username,role,approved) VALUES('second-owner','super_admin',true)`); err == nil {
			t.Fatal("a second Super-administrator was accepted")
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET role='super_admin' WHERE username='operator'`); err == nil {
			t.Fatal("an Administrator was promoted to a second Super-administrator")
		}
		if superAdminCount(t) != 1 {
			t.Fatal("the number of Super-administrators changed")
		}
	})

	t.Run("rights matrix", func(t *testing.T) {
		settings := `{"client_id":"synthetic-app","client_secret":"fixture-discord-secret","registration_approval":false}`
		reserved := []struct{ method, path, data string }{
			{"POST", "/api/admin/master-key/export", ""},
			{"PUT", "/api/admin/settings", settings},
		}
		shared := []string{
			"/api/admin/settings",
			"/api/admin/users",
			"/api/admin/workers",
			"/api/admin/jobs",
			"/api/admin/sources",
			"/api/admin/vocabulary",
			"/api/admin/job-controls",
		}
		// The Administrator is refused the reserved actions explicitly, and nothing changes.
		for _, r := range reserved {
			body := call(t, operator, r.method, r.path, r.data, 403)
			if errorCode(body) != "super_administrator_required" {
				t.Fatalf("%s %s: unexpected refusal %s", r.method, r.path, body)
			}
			body = call(t, member, r.method, r.path, r.data, 403)
			if errorCode(body) != "forbidden" {
				t.Fatalf("%s %s: unexpected refusal for a user %s", r.method, r.path, body)
			}
		}
		var approval bool
		if err := pool.QueryRow(ctx, `SELECT registration_approval FROM installation WHERE id=1`).Scan(&approval); err != nil || !approval {
			t.Fatal("a refused request changed the registration approval", err)
		}
		if value, err := s.Store.Get(ctx, "discord_client_id"); err != nil || value != "" {
			t.Fatal("a refused request changed the Discord configuration", value, err)
		}
		// The Administrator keeps the rest of the administration; an ordinary user has none of it.
		for _, path := range shared {
			call(t, operator, "GET", path, "", 200)
			call(t, owner, "GET", path, "", 200)
			call(t, member, "GET", path, "", 403)
		}
		// The Super-administrator does everything.
		call(t, owner, "POST", "/api/admin/master-key/export", "", 200)
		call(t, owner, "PUT", "/api/admin/settings", settings, 200)
		var settingsView struct {
			RegistrationApproval bool `json:"registration_approval"`
		}
		if err := json.Unmarshal(call(t, operator, "GET", "/api/admin/settings", "", 200), &settingsView); err != nil || settingsView.RegistrationApproval {
			t.Fatal("the Administrator cannot read the settings the Super-administrator saved", err)
		}
		// With approval switched off, new accounts are active at once; an Administrator still approves
		// the registrations that were waiting.
		var pending int64
		if err := pool.QueryRow(ctx, `INSERT INTO users(username,role,approved) VALUES('newcomer','user',false) RETURNING id`).Scan(&pending); err != nil {
			t.Fatal(err)
		}
		call(t, operator, "POST", "/api/admin/users/"+strconv.FormatInt(pending, 10)+"/approve", "", 200)
	})

	t.Run("the Super-administrator can be neither demoted nor deleted", func(t *testing.T) {
		for name, statement := range map[string]string{
			"demotion to Administrator": `UPDATE users SET role='admin' WHERE username='owner'`,
			"demotion to user":          `UPDATE users SET role='user' WHERE username='owner'`,
			"deletion":                  `DELETE FROM users WHERE username='owner'`,
		} {
			_, err := pool.Exec(ctx, statement)
			if err == nil || !strings.Contains(err.Error(), "super_administrator_protected") {
				t.Fatalf("%s: %v", name, err)
			}
		}
		if got := role(t, owner); got != "super_admin" {
			t.Fatalf("role after the refused changes: %q", got)
		}
		// Other accounts, and other columns of the Super-administrator, stay changeable.
		if _, err := pool.Exec(ctx, `UPDATE users SET role='user' WHERE username='member'; UPDATE users SET approved=true WHERE username='owner'`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("an existing installation keeps every access when it migrates", func(t *testing.T) {
		version := int64(0)
		for _, source := range provider.ListSources() {
			if strings.Contains(source.Path, "super_administrator") {
				version = source.Version
			}
		}
		if version == 0 {
			t.Fatal("migration not found")
		}
		if _, err := provider.ApplyVersion(ctx, version, false); err != nil {
			t.Fatal("rolling the migration back to simulate the previous version:", err)
		}
		if _, err := pool.Exec(ctx, `TRUNCATE sessions,users RESTART IDENTITY CASCADE`); err != nil {
			t.Fatal(err)
		}
		// Three Administrators, the oldest being neither the first inserted nor the first by name, and one user.
		for _, row := range []struct{ name, role, created string }{
			{"newest-admin", "admin", "2026-03-01"},
			{"oldest-admin", "admin", "2025-01-01"},
			{"middle-admin", "admin", "2025-06-01"},
			{"ordinary", "user", "2024-01-01"},
		} {
			if _, err := pool.Exec(ctx, `INSERT INTO users(username,role,approved,created_at) VALUES($1,$2,true,$3::timestamptz)`, row.name, row.role, row.created); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := provider.ApplyVersion(ctx, version, true); err != nil {
			t.Fatal("migration of the existing installation:", err)
		}
		rows, err := pool.Query(ctx, `SELECT username,role FROM users`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		roles := map[string]string{}
		for rows.Next() {
			var name, r string
			if err := rows.Scan(&name, &r); err != nil {
				t.Fatal(err)
			}
			roles[name] = r
		}
		want := map[string]string{"oldest-admin": "super_admin", "middle-admin": "admin", "newest-admin": "admin", "ordinary": "user"}
		for name, r := range want {
			if roles[name] != r {
				t.Fatalf("roles after migration: %v, want %v", roles, want)
			}
		}
		if superAdminCount(t) != 1 {
			t.Fatal("the migration must leave exactly one Super-administrator")
		}
	})
}
