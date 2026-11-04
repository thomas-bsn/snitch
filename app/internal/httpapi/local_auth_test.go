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
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/auth"
	"snitch/internal/config"
	"snitch/internal/setup"
)

func testLocalAuthentication(t *testing.T, pool *pgxpool.Pool) {
	var logs bytes.Buffer
	s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(&logs, nil)))
	ts := httptest.NewUnstartedServer(s.Handler())
	ts.Start()
	defer ts.Close()
	t.Run("login rejects setup-only code field", func(t *testing.T) {
		req, _ := http.NewRequest("POST", ts.URL+"/api/auth/login", bytes.NewBufferString(`{"username":"administrator","password":"synthetic-password","code":"setup-only"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", ts.URL)
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 400 {
			t.Fatalf("got %d, want invalid request 400", resp.StatusCode)
		}
	})

	t.Run("login rejects a media type that only starts with JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", ts.URL+"/api/auth/login", strings.NewReader(`{"username":"administrator","password":"synthetic-password"}`))
		req.Header.Set("Content-Type", "application/json-invalid")
		req.Header.Set("Origin", ts.URL)
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 415 {
			t.Fatalf("got %d, want JSON required 415", resp.StatusCode)
		}
	})
	t.Run("credential lengths follow Unicode contract", func(t *testing.T) {
		for _, body := range []string{
			`{"username":"éé","password":"synthetic-password"}`,
			`{"username":"administrator","password":"éééééé"}`,
			`{"username":"administrator","password":"short"}`,
		} {
			req, _ := http.NewRequest("POST", ts.URL+"/api/auth/login", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", ts.URL)
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != 400 {
				t.Errorf("got %d for %s, want 400", resp.StatusCode, body)
			}
		}
	})

	t.Run("setup recovery and serialized HTTP creation", func(t *testing.T) {
		ctx := context.Background()
		original, err := setup.Issue(ctx, pool)
		if err != nil {
			t.Fatal(err)
		}
		if replacement, err := setup.Issue(ctx, pool); err != nil || replacement != "" {
			t.Fatal("restart must preserve a live code", err)
		}
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
			status  int
			body    []byte
			cookies []*http.Cookie
		}
		call := func(method, path, body, token, csrf, origin string, validInput bool) result {
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
			if (err == nil) != validInput {
				t.Fatalf("request contract %s %s: valid=%v error=%v", method, path, validInput, err)
			}
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			if err = openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: resp.StatusCode, Header: resp.Header, Body: io.NopCloser(bytes.NewReader(b))}); err != nil {
				t.Fatal(err)
			}
			return result{resp.StatusCode, b, resp.Cookies()}
		}
		expect := func(r result, status int, code string) {
			t.Helper()
			if r.status != status || (code != "" && !bytes.Contains(r.body, []byte(`"error":"`+code+`"`))) {
				t.Fatalf("got %d %s; want %d %s", r.status, r.body, status, code)
			}
		}
		bootstrap := func(code, name string) string {
			b, _ := json.Marshal(map[string]string{"code": code, "username": name, "password": "synthetic-password"})
			return string(b)
		}

		for _, body := range []string{
			`null`, `{}`, `{"username":"administrator"}`, `{"username":true,"password":"synthetic-password"}`,
			`{"username":"administrator","password":"synthetic-password","unexpected":"secret-only-fixture"}`,
		} {
			s.rates.Clear()
			expect(call("POST", "/api/auth/login", body, "", "", ts.URL, false), 400, "invalid_request")
		}
		s.rates.Clear()
		expect(call("POST", "/api/setup", bootstrap("wrong", "administrator"), "", "", ts.URL, true), 409, "setup_closed_or_invalid_code")
		expect(call("POST", "/api/setup", `{"username":"administrator","password":"synthetic-password"}`, "", "", ts.URL, false), 400, "invalid_credentials_format")
		if _, err = pool.Exec(ctx, `UPDATE installation SET bootstrap_expires=now()-interval '1 second'`); err != nil {
			t.Fatal(err)
		}
		expect(call("POST", "/api/setup", bootstrap(original, "administrator"), "", "", ts.URL, true), 409, "setup_closed_or_invalid_code")
		recovered, err := setup.Issue(ctx, pool)
		if err != nil || recovered == "" || recovered == original {
			t.Fatal("expired code not renewed", err)
		}
		expect(call("POST", "/api/setup", bootstrap(original, "administrator"), "", "", ts.URL, true), 409, "setup_closed_or_invalid_code")
		s.rates.Clear()
		start := make(chan struct{})
		results := make(chan result, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				results <- call("POST", "/api/setup", bootstrap(recovered, fmt.Sprintf("admin%d", i)), "", "", ts.URL, true)
			}(i)
		}
		close(start)
		wg.Wait()
		close(results)
		var winner result
		successes := 0
		for r := range results {
			if r.status == 201 {
				successes++
				winner = r
			} else {
				expect(r, 409, "setup_closed_or_invalid_code")
			}
		}
		if successes != 1 {
			t.Fatalf("successful admins: %d", successes)
		}
		var count int
		if err = pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE role='super_admin'`).Scan(&count); err != nil || count != 1 {
			t.Fatal("admin count", count, err)
		}
		token := winner.cookies[0].Value
		// Restart the public server against the same persisted installation.
		ts.Close()
		restarted := New(pool, s.Config, s.Log)
		ts = httptest.NewUnstartedServer(restarted.Handler())
		ts.Start()
		defer ts.Close()
		if next, err := setup.Issue(ctx, pool); err != nil || next != "" {
			t.Fatal("closed installation issued code", err)
		}
		expect(call("POST", "/api/setup", bootstrap(recovered, "anotheradmin"), "", "", ts.URL, true), 409, "setup_closed_or_invalid_code")
		status := call("GET", "/api/setup", "", "", "", ts.URL, true)
		if !bytes.Contains(status.body, []byte(`"initialized":true`)) {
			t.Fatal(string(status.body))
		}
		me := call("GET", "/api/auth/me", "", token, "", ts.URL, true)
		expect(me, 200, "")
		var session struct {
			CSRF string `json:"csrf_token"`
			User struct {
				Username string `json:"username"`
			} `json:"user"`
		}
		if err = json.Unmarshal(me.body, &session); err != nil {
			t.Fatal(err)
		}
		var stored []byte
		var password string
		var expires time.Time
		if err = pool.QueryRow(ctx, `SELECT s.token_hash,u.password_hash,s.expires_at FROM sessions s JOIN users u ON u.id=s.user_id`).Scan(&stored, &password, &expires); err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(stored, []byte(token)) || !bytes.Equal(stored, auth.Hash(token)) || !strings.HasPrefix(password, "$argon2id$v=19$m=32768,t=3,p=1$") || !auth.Verify(password, "synthetic-password") {
			t.Fatal("credentials are not securely persisted")
		}
		if d := time.Until(expires); d < 7*24*time.Hour-time.Minute || d > 7*24*time.Hour {
			t.Fatal("session lifetime", d)
		}
		expect(call("POST", "/api/auth/logout", "", token, "", ts.URL, true), 403, "csrf_rejected")
		expect(call("POST", "/api/auth/logout", "", token, session.CSRF, "https://attacker.invalid", true), 403, "origin_rejected")
		expect(call("POST", "/api/auth/logout", "", token, session.CSRF, ts.URL, true), 200, "")
		expect(call("GET", "/api/auth/me", "", token, "", ts.URL, true), 401, "unauthorized")
		expect(call("POST", "/api/auth/logout", "", token, session.CSRF, ts.URL, true), 401, "unauthorized")
		credentials := func(name, pass string) string {
			b, _ := json.Marshal(map[string]string{"username": name, "password": pass})
			return string(b)
		}
		restarted.rates.Clear()
		login := call("POST", "/api/auth/login", credentials(session.User.Username, "synthetic-password"), "", "", ts.URL, true)
		expect(login, 200, "")
		token = login.cookies[0].Value
		if _, err = pool.Exec(ctx, `UPDATE sessions SET expires_at=now()-interval '1 second'`); err != nil {
			t.Fatal(err)
		}
		expect(call("GET", "/api/auth/me", "", token, "", ts.URL, true), 401, "unauthorized")
		bad := call("POST", "/api/auth/login", credentials(session.User.Username, "wrong-password"), "", "", ts.URL, true)
		absent := call("POST", "/api/auth/login", credentials("missing-user", "wrong-password"), "", "", ts.URL, true)
		expect(bad, 401, "invalid_credentials_or_pending")
		expect(absent, 401, "invalid_credentials_or_pending")
		if !bytes.Equal(bad.body, absent.body) {
			t.Fatal("account enumeration")
		}
		// Secure cookies are observed over a real TLS HTTP connection.
		secure := New(pool, config.Config{MasterKey: s.Config.MasterKey, Secure: true}, s.Log)
		tls := httptest.NewUnstartedServer(secure.Handler())
		tls.StartTLS()
		defer tls.Close()
		req, _ := http.NewRequest("POST", tls.URL+"/api/auth/login", strings.NewReader(credentials(session.User.Username, "synthetic-password")))
		req.Header.Set("Origin", tls.URL)
		req.Header.Set("Content-Type", "application/json")
		resp, err := tls.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 || len(resp.Cookies()) != 1 {
			t.Fatal("TLS login", resp.StatusCode)
		}
		cookie := resp.Cookies()[0]
		if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 604800 || cookie.Path != "/" {
			t.Fatalf("unsafe cookie: %+v", cookie)
		}
		for _, secret := range []string{"synthetic-password", "secret-only-fixture", original, recovered, token} {
			if bytes.Contains(logs.Bytes(), []byte(secret)) {
				t.Fatal("request handling leaked a secret in logs")
			}
		}

	})
}
