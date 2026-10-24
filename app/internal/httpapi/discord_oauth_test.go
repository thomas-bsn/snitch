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
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/jackc/pgx/v5/pgxpool"
	"receipts/internal/config"
	"receipts/internal/setup"
)

func testDiscordOAuth(t *testing.T, pool *pgxpool.Pool) {
	var logs bytes.Buffer
	s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(&logs, nil)))
	ts := httptest.NewUnstartedServer(s.Handler())
	s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()
	var identity atomic.Value
	identity.Store("123456")
	var mode atomic.Value
	mode.Store("ok")
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			if mode.Load().(string) == "blocked" {
				entered <- struct{}{}
				<-release
			}
			if mode.Load().(string) == "token-error" {
				reply(w, 500, map[string]string{"error": "fixture-secret"})
				return
			}
			if mode.Load().(string) == "malformed-token" {
				w.Write([]byte("not-json fixture-secret"))
				return
			}
			reply(w, 200, map[string]string{"access_token": "fixture-token"})
			return
		}
		if mode.Load().(string) == "invalid-user" {
			reply(w, 200, map[string]string{"id": ""})
			return
		}
		reply(w, 200, map[string]string{"id": identity.Load().(string), "username": "administrator"})
	}))
	defer provider.Close()
	s.Config.DiscordTokenURL = provider.URL + "/token"
	s.Config.DiscordUserURL = provider.URL + "/me"
	s.Config.DiscordAuthorizeURL = provider.URL + "/authorize"
	client := func() *http.Client {
		j, _ := cookiejar.New(nil)
		return &http.Client{Jar: j, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	admin := client()
	spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec.Servers = nil
	router, err := legacy.NewRouter(spec)
	if err != nil {
		t.Fatal(err)
	}

	call := func(c *http.Client, method, path, body, csrf string, want int) (*http.Response, []byte) {
		t.Helper()
		s.rates.Clear()
		r, _ := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		r.Header.Set("Origin", ts.URL)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Token", csrf)
		route, params, err := router.FindRoute(r)
		if err != nil {
			t.Fatal(err)
		}
		input := &openapi3filter.RequestValidationInput{Request: r, Route: route, PathParams: params, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
		if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
			t.Fatal(err)
		}
		resp, e := c.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("%s %s: %d %s want %d", method, path, resp.StatusCode, b, want)
		}
		if err := openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: resp.StatusCode, Header: resp.Header, Body: io.NopCloser(bytes.NewReader(b))}); err != nil {
			t.Fatal(err)
		}
		return resp, b
	}
	code, e := setup.Issue(context.Background(), pool)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"})
	call(admin, "POST", "/api/setup", string(b), "", 201)
	csrf := func(c *http.Client) string {
		_, b := call(c, "GET", "/api/auth/me", "", "", 200)
		var m struct {
			CSRF string `json:"csrf_token"`
		}
		json.Unmarshal(b, &m)
		return m.CSRF
	}
	call(admin, "PUT", "/api/admin/settings", `{"client_id":"fixture-app","client_secret":"fixture-secret","registration_approval":true}`, csrf(admin), 200)
	start := func(c *http.Client, link bool) string {
		path := "/api/auth/discord/start"
		token := ""
		if link {
			path = "/api/auth/discord/link"
			token = csrf(c)
		}
		_, b := call(c, "POST", path, "", token, 200)
		var out struct {
			URL string `json:"url"`
		}
		json.Unmarshal(b, &out)
		u, _ := url.Parse(out.URL)
		if u.Query().Get("scope") != "identify" {
			t.Fatal(out.URL)
		}
		return "/api/auth/discord/callback?code=fixture-code&state=" + u.Query().Get("state")
	}
	t.Run("callback decryption failures return browsers to the SPA and API errors stay JSON", func(t *testing.T) {
		for _, accept := range []string{"text/html", "application/json"} {
			visitor := client()
			path := start(visitor, false)
			if _, err := pool.Exec(context.Background(), `UPDATE settings SET value='\\x00'::bytea WHERE key='discord_client_secret'`); err != nil {
				t.Fatal(err)
			}
			req, _ := http.NewRequest("GET", ts.URL+path, nil)
			req.Header.Set("Accept", accept)
			resp, err := visitor.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if accept == "text/html" {
				if resp.StatusCode != 303 || resp.Header.Get("Location") != "/?notice=unavailable" {
					t.Fatalf("browser callback: %d %s %s", resp.StatusCode, resp.Header.Get("Location"), body)
				}
			} else if resp.StatusCode != 503 || string(bytes.TrimSpace(body)) != `{"error":"unavailable"}` {
				t.Fatalf("API callback: %d %s", resp.StatusCode, body)
			}
			call(visitor, "GET", path, "", "", 400)
			if err := s.Store.Set(context.Background(), "discord_client_secret", "fixture-secret"); err != nil {
				t.Fatal(err)
			}
		}
	})
	// Returning in another valid session for the same account must not inherit a link.
	path := start(admin, true)
	call(admin, "POST", "/api/auth/logout", "", csrf(admin), 200)
	call(admin, "POST", "/api/auth/login", `{"username":"administrator","password":"synthetic-password"}`, "", 200)
	_, b = call(admin, "GET", path, "", "", 403)
	if !bytes.Contains(b, []byte("link_session_required")) {
		t.Fatal(string(b))
	}
	// A local name that resembles the generated Discord name never absorbs or blocks an identity.
	call(admin, "POST", "/api/auth/register", `{"username":"discord_123456","password":"synthetic-password"}`, "", 201)
	visitor := client()
	response, _ := call(visitor, "GET", start(visitor, false), "", "", 303)
	if response.Header.Get("Location") != "/?notice=pending" {
		t.Fatal(response.Header)
	}
	call(visitor, "GET", "/api/auth/me", "", "", 401)
	_, b = call(admin, "GET", "/api/admin/users", "", "", 200)
	var pending []struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	}
	json.Unmarshal(b, &pending)
	if len(pending) != 2 || pending[0].ID == pending[1].ID {
		t.Fatalf("distinct pending accounts: %s", b)
	}

	// Consent refusal consumes the state and returns to a usable login page.
	denied := client()
	path = start(denied, false)
	u, _ := url.Parse(path)
	q := u.Query()
	q.Del("code")
	q.Set("error", "access_denied")
	denial := "/api/auth/discord/callback?" + q.Encode()
	response, _ = call(denied, "GET", denial, "", "", 303)
	if response.Header.Get("Location") != "/?notice=discord_denied" {
		t.Fatal(response.Header)
	}
	call(denied, "GET", path, "", "", 400)

	// Approved identities reconnect to the same account and preserve existing local identities.
	var discordID int64
	for _, u := range pending {
		if u.Username != "discord_123456" {
			discordID = u.ID
		}
	}
	call(admin, "POST", fmt.Sprintf("/api/admin/users/%d/approve", discordID), "", csrf(admin), 200)
	call(visitor, "GET", start(visitor, false), "", "", 303)
	_, b = call(visitor, "GET", "/api/auth/me", "", "", 200)
	if !bytes.Contains(b, []byte(fmt.Sprintf(`"id":%d`, discordID))) {
		t.Fatal(string(b))
	}
	path = start(visitor, false)
	call(client(), "GET", path, "", "", 400) // Another browser cannot consume the valid flow.
	call(visitor, "GET", path, "", "", 303)
	call(visitor, "GET", path, "", "", 400)
	path = start(visitor, false)
	call(visitor, "GET", path+"wrong", "", "", 400)
	call(visitor, "GET", path, "", "", 303)
	path = start(visitor, false)
	if _, err := pool.Exec(context.Background(), `UPDATE oauth_states SET expires_at=now()-interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	call(visitor, "GET", path, "", "", 400)
	for _, failure := range []string{"token-error", "malformed-token", "invalid-user"} {
		mode.Store(failure)
		path = start(visitor, false)
		_, body := call(visitor, "GET", path, "", "", 502)
		if string(bytes.TrimSpace(body)) != `{"error":"discord_unavailable"}` {
			t.Fatal(string(body))
		}
		call(visitor, "GET", path, "", "", 400)
		call(admin, "POST", "/api/auth/login", `{"username":"administrator","password":"synthetic-password"}`, "", 200)
	}
	// Slow Discord calls must never occupy the bounded local password slots.
	mode.Store("blocked")
	a, browserB := client(), client()
	pa, pb := start(a, false), start(browserB, false)
	done := make(chan error, 2)
	for _, flow := range []struct {
		c *http.Client
		p string
	}{{a, pa}, {browserB, pb}} {
		go func(c *http.Client, p string) {
			r, e := c.Get(ts.URL + p)
			if e == nil {
				r.Body.Close()
			}
			done <- e
		}(flow.c, flow.p)
	}
	<-entered
	<-entered
	func() {
		defer close(release)
		call(admin, "POST", "/api/auth/login", `{"username":"administrator","password":"synthetic-password"}`, "", 200)
	}()
	for i := 0; i < 2; i++ {
		if e := <-done; e != nil {
			t.Fatal(e)
		}
	}

	// A session revoked while Discord is responding cannot finish a link.
	identity.Store("888888")
	mode.Store("blocked")
	release = make(chan struct{})
	path = start(admin, true)
	go func() {
		r, e := admin.Get(ts.URL + path)
		if e != nil {
			done <- e
			return
		}
		defer r.Body.Close()
		if r.StatusCode != 403 {
			done <- fmt.Errorf("revoked linking session accepted: %d", r.StatusCode)
		} else {
			done <- nil
		}
	}()
	<-entered
	call(admin, "POST", "/api/auth/logout", "", csrf(admin), 200)
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}

	// Registration policy also applies to new Discord identities without approval.
	mode.Store("ok")
	identity.Store("999999")
	call(admin, "POST", "/api/auth/login", `{"username":"administrator","password":"synthetic-password"}`, "", 200)
	call(admin, "PUT", "/api/admin/settings", `{"client_id":"fixture-app","registration_approval":false}`, csrf(admin), 200)
	active := client()
	call(active, "GET", start(active, false), "", "", 303)
	_, body := call(active, "GET", "/api/auth/me", "", "", 200)
	if !bytes.Contains(body, []byte(`"username":"discord_999999"`)) {
		t.Fatal(string(body))
	}
	t.Run("session persistence failures return browser navigation to a friendly notice", func(t *testing.T) {
		visitor := client()
		path := start(visitor, false)
		if _, err := pool.Exec(context.Background(), `ALTER TABLE sessions ADD CONSTRAINT fixture_session_failure CHECK (false) NOT VALID`); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if _, err := pool.Exec(context.Background(), `ALTER TABLE sessions DROP CONSTRAINT fixture_session_failure`); err != nil {
				t.Error(err)
			}
		}()
		req, _ := http.NewRequest("GET", ts.URL+path, nil)
		req.Header.Set("Accept", "text/html")
		resp, err := visitor.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 303 || resp.Header.Get("Location") != "/?notice=unavailable" {
			t.Fatalf("session failure: %d %s", resp.StatusCode, resp.Header.Get("Location"))
		}
		call(visitor, "GET", "/api/auth/me", "", "", 401)
	})
	// Concurrent explicit linking has exactly one winner, whose account remains the owner.
	identity.Store("777777")
	locals := []*http.Client{client(), client()}
	paths := make([]string, 2)
	ids := make([]int64, 2)
	for i, c := range locals {
		name := fmt.Sprintf("local-%d", i)
		credentials := fmt.Sprintf(`{"username":%q,"password":"synthetic-password"}`, name)
		call(c, "POST", "/api/auth/register", credentials, "", 201)
		call(c, "POST", "/api/auth/login", credentials, "", 200)
		_, b := call(c, "GET", "/api/auth/me", "", "", 200)
		var me struct {
			User struct {
				ID int64 `json:"id"`
			} `json:"user"`
		}
		json.Unmarshal(b, &me)
		ids[i] = me.User.ID
		paths[i] = start(c, true)
	}
	type outcome struct {
		i, status int
		err       error
	}
	results := make(chan outcome, 2)
	for i, c := range locals {
		go func(i int, c *http.Client) {
			r, e := c.Get(ts.URL + paths[i])
			o := outcome{i: i, err: e}
			if e == nil {
				o.status = r.StatusCode
				r.Body.Close()
			}
			results <- o
		}(i, c)
	}
	winner := -1
	conflicts := 0
	for i := 0; i < 2; i++ {
		o := <-results
		if o.err != nil {
			t.Fatal(o.err)
		}
		switch o.status {
		case 303:
			winner = o.i
		case 409:
			conflicts++
		default:
			t.Fatalf("link status %d", o.status)
		}
	}
	if winner < 0 || conflicts != 1 {
		t.Fatalf("winner %d conflicts %d", winner, conflicts)
	}
	reconnect := client()
	call(reconnect, "GET", start(reconnect, false), "", "", 303)
	_, body = call(reconnect, "GET", "/api/auth/me", "", "", 200)
	if !bytes.Contains(body, []byte(fmt.Sprintf(`"id":%d`, ids[winner]))) {
		t.Fatal(string(body))
	}
	// An already linked account cannot silently replace its chosen identity.
	identity.Store("666666")
	call(locals[winner], "GET", start(locals[winner], true), "", "", 409)
	identity.Store("777777")
	call(reconnect, "GET", start(reconnect, false), "", "", 303)
	s.Config.DiscordTokenURL = ":invalid-provider-url"
	unavailable := client()
	call(unavailable, "GET", start(unavailable, false), "", "", 502)
	call(admin, "POST", "/api/auth/login", `{"username":"administrator","password":"synthetic-password"}`, "", 200)
	for _, sensitive := range []string{"fixture-secret", "fixture-token", "fixture-code"} {
		if strings.Contains(logs.String(), sensitive) {
			t.Fatal("OAuth credentials appeared in logs")
		}
	}
}
