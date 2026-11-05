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
	"strconv"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

func testRegistration(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewUnstartedServer(s.Handler())
	ts.Start()
	defer ts.Close()
	spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec.Servers = nil
	router, err := legacy.NewRouter(spec)
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		body  []byte
		token string
	}
	call := func(method, path, body, token, csrf, origin string, want int, code string, valid bool) result {
		t.Helper()
		s.rates.Clear()
		req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		req.Header.Set("X-CSRF-Token", csrf)
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "snitch_session", Value: token})
		}
		route, params, err := router.FindRoute(req)
		if err != nil {
			t.Fatal(err)
		}
		input := &openapi3filter.RequestValidationInput{Request: req, Route: route, PathParams: params, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
		err = openapi3filter.ValidateRequest(ctx, input)
		if (err == nil) != valid {
			t.Fatalf("request contract %s %s valid=%v: %v", method, path, valid, err)
		}
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want || (code != "" && !bytes.Contains(b, []byte(`"error":"`+code+`"`))) {
			t.Fatalf("%s %s got %d %s; want %d %s", method, path, resp.StatusCode, b, want, code)
		}
		if err = openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: resp.StatusCode, Header: resp.Header, Body: io.NopCloser(bytes.NewReader(b))}); err != nil {
			t.Fatal(err)
		}
		r := result{body: b}
		for _, c := range resp.Cookies() {
			if c.Name == "snitch_session" {
				r.token = c.Value
			}
		}
		return r
	}
	credentials := func(name string) string {
		b, _ := json.Marshal(map[string]string{"username": name, "password": "synthetic-password"})
		return string(b)
	}
	call("POST", "/api/auth/register", credentials("visitor"), "", "", ts.URL, 409, "setup_required", true)
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"})
	admin := call("POST", "/api/setup", string(b), "", "", ts.URL, 201, "", true).token
	session := func(token string) string {
		var me struct {
			CSRF string `json:"csrf_token"`
		}
		json.Unmarshal(call("GET", "/api/auth/me", "", token, "", ts.URL, 200, "", true).body, &me)
		return me.CSRF
	}
	csrf := session(admin)
	// Missing/null policy must never silently turn approval off.
	for _, body := range []string{`{}`, `null`, `{"client_id":""}`, `{"client_id":"","registration_approval":null}`, `{"client_id":"","registration_approval":"false"}`} {
		call("PUT", "/api/admin/settings", body, admin, csrf, ts.URL, 400, "invalid_request", false)
	}
	settings := call("GET", "/api/admin/settings", "", admin, "", ts.URL, 200, "", true)
	if !bytes.Contains(settings.body, []byte(`"registration_approval":true`)) {
		t.Fatal(string(settings.body))
	}
	// Malformed credentials cannot create a usable or privileged account.
	for _, body := range []string{`null`, `{}`, `{"username":"administrator","password":"synthetic-password","code":"bootstrap-only"}`, `{"username":"éé","password":"synthetic-password"}`, `{"username":"visitor","password":"éééééé"}`, `{"username":"visitor","password":"synthetic-password","role":"admin"}`} {
		call("POST", "/api/auth/register", body, "", "", ts.URL, 400, "", false)
	}
	pending := call("POST", "/api/auth/register", credentials("pending-player"), "", "", ts.URL, 201, "", true)
	if pending.token != "" || !bytes.Contains(pending.body, []byte(`"status":"pending"`)) {
		t.Fatal("pending registration opened session", string(pending.body))
	}
	call("POST", "/api/auth/login", credentials("pending-player"), "", "", ts.URL, 401, "invalid_credentials_or_pending", true)
	call("GET", "/api/auth/me", "", "", "", ts.URL, 401, "unauthorized", true)
	call("POST", "/api/auth/register", credentials("PENDING-PLAYER"), "", "", ts.URL, 409, "username_unavailable", true)
	policy := `{"client_id":"","registration_approval":false}`
	call("PUT", "/api/admin/settings", policy, admin, "", ts.URL, 403, "csrf_rejected", true)
	call("PUT", "/api/admin/settings", policy, admin, csrf, "https://attacker.invalid", 403, "origin_rejected", true)
	call("PUT", "/api/admin/settings", policy, admin, csrf, ts.URL, 200, "", true)
	active := call("POST", "/api/auth/register", credentials("active-player"), "", "", ts.URL, 201, "", true)
	if active.token != "" || !bytes.Contains(active.body, []byte(`"status":"active"`)) {
		t.Fatal(string(active.body))
	}
	user := call("POST", "/api/auth/login", credentials("active-player"), "", "", ts.URL, 200, "", true).token
	userCSRF := session(user)
	// Policy changes affect new accounts only.
	call("POST", "/api/auth/login", credentials("pending-player"), "", "", ts.URL, 401, "invalid_credentials_or_pending", true)
	var users []struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
	}
	json.Unmarshal(call("GET", "/api/admin/users", "", admin, "", ts.URL, 200, "", true).body, &users)
	if len(users) != 1 || users[0].Username != "pending-player" || users[0].Role != "user" || users[0].Approved {
		t.Fatalf("pending list: %+v", users)
	}
	path := "/api/admin/users/" + strconv.FormatInt(users[0].ID, 10) + "/approve"
	call("GET", "/api/admin/users", "", user, "", ts.URL, 403, "forbidden", true)
	call("PUT", "/api/admin/settings", policy, user, userCSRF, ts.URL, 403, "forbidden", true)
	call("POST", path, "", user, userCSRF, ts.URL, 403, "forbidden", true)
	call("GET", "/api/admin/users", "", "", "", ts.URL, 401, "unauthorized", true)
	call("POST", path, "", "", "", ts.URL, 401, "unauthorized", true)
	call("POST", path, "", admin, "", ts.URL, 403, "csrf_rejected", true)
	call("POST", path, "", admin, csrf, "https://attacker.invalid", 403, "origin_rejected", true)
	for _, id := range []string{"0", "-1", "abc", "9223372036854775808"} {
		call("POST", "/api/admin/users/"+id+"/approve", "", admin, csrf, ts.URL, 400, "invalid_id", false)
	}
	call("POST", "/api/admin/users/99999/approve", "", admin, csrf, ts.URL, 404, "not_pending", true)
	call("POST", path, "", admin, csrf, ts.URL, 200, "", true)
	call("POST", path, "", admin, csrf, ts.URL, 404, "not_pending", true)
	if b := call("GET", "/api/admin/users", "", admin, "", ts.URL, 200, "", true).body; string(bytes.TrimSpace(b)) != "[]" {
		t.Fatal(string(b))
	}
	approved := call("POST", "/api/auth/login", credentials("pending-player"), "", "", ts.URL, 200, "", true).token
	if b := call("GET", "/api/auth/me", "", approved, "", ts.URL, 200, "", true).body; !bytes.Contains(b, []byte(`"role":"user"`)) {
		t.Fatal(string(b))
	}
	call("PUT", "/api/admin/settings", `{"client_id":"","registration_approval":true}`, admin, csrf, ts.URL, 200, "", true)
	// Unicode boundaries agree with OpenAPI rather than UTF-8 byte lengths.
	unicodeBody, _ := json.Marshal(map[string]string{"username": strings.Repeat("é", 64), "password": strings.Repeat("é", 256)})
	call("POST", "/api/auth/register", string(unicodeBody), "", "", ts.URL, 201, "", true)
	for _, value := range []map[string]string{{"username": strings.Repeat("é", 65), "password": "synthetic-password"}, {"username": "visitor", "password": strings.Repeat("é", 257)}} {
		b, _ := json.Marshal(value)
		call("POST", "/api/auth/register", string(b), "", "", ts.URL, 400, "invalid_credentials_format", false)
	}

}
