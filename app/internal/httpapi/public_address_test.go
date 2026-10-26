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
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

// testPublicAddress observes the public address Snitch deduces from each request: the origin
// check, the Discord return address, the worker deployment file and the session cookie.
func testPublicAddress(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	loopback := []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("::1/128")}
	base := func(mutate func(*config.Config)) config.Config {
		c := config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Fixture", UploadRoot: t.TempDir()}
		if mutate != nil {
			mutate(&c)
		}
		return c
	}
	start := func(t *testing.T, c config.Config, tls bool) (*Server, *httptest.Server) {
		t.Helper()
		s := New(pool, c, slog.New(slog.NewJSONHandler(io.Discard, nil)))
		ts := httptest.NewUnstartedServer(s.Handler())
		if tls {
			ts.StartTLS()
		} else {
			ts.Start()
		}
		t.Cleanup(ts.Close)
		return s, ts
	}
	type answer struct {
		status  int
		body    []byte
		cookies []*http.Cookie
	}
	call := func(t *testing.T, ts *httptest.Server, method, path, body string, header map[string]string, cookie *http.Cookie) answer {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		for k, v := range header {
			if strings.EqualFold(k, "Host") {
				req.Host = v
				continue
			}
			req.Header.Set(k, v)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		client := ts.Client()
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return answer{resp.StatusCode, data, resp.Cookies()}
	}
	credentials := `{"username":"administrator","password":"synthetic-password"}`
	login := func(t *testing.T, ts *httptest.Server, header map[string]string) (*http.Cookie, string) {
		t.Helper()
		r := call(t, ts, "POST", "/api/auth/login", credentials, header, nil)
		if r.status != 200 || len(r.cookies) != 1 {
			t.Fatalf("login: %d %s", r.status, r.body)
		}
		var me struct {
			CSRF string `json:"csrf_token"`
		}
		forwarded := map[string]string{}
		for k, v := range header {
			if k != "Origin" {
				forwarded[k] = v
			}
		}
		if err := json.Unmarshal(call(t, ts, "GET", "/api/auth/me", "", forwarded, r.cookies[0]).body, &me); err != nil || me.CSRF == "" {
			t.Fatal("session without CSRF token", err)
		}
		return r.cookies[0], me.CSRF
	}
	loginStatus := func(t *testing.T, ts *httptest.Server, header map[string]string) answer {
		t.Helper()
		return call(t, ts, "POST", "/api/auth/login", credentials, header, nil)
	}
	redirectURI := func(t *testing.T, ts *httptest.Server, header map[string]string) string {
		t.Helper()
		r := call(t, ts, "POST", "/api/auth/discord/start", "", header, nil)
		if r.status != 200 {
			t.Fatalf("discord start: %d %s", r.status, r.body)
		}
		var out struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(r.body, &out); err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(out.URL)
		if err != nil {
			t.Fatal(err)
		}
		return u.Query().Get("redirect_uri")
	}
	callbackShown := func(t *testing.T, ts *httptest.Server, header map[string]string, cookie *http.Cookie) string {
		t.Helper()
		r := call(t, ts, "GET", "/api/admin/settings", "", header, cookie)
		var out struct {
			CallbackURL string `json:"callback_url"`
		}
		if r.status != 200 || json.Unmarshal(r.body, &out) != nil {
			t.Fatalf("settings: %d %s", r.status, r.body)
		}
		return out.CallbackURL
	}
	workerAPIURL := func(t *testing.T, ts *httptest.Server, header map[string]string, cookie *http.Cookie, csrf string) (string, bool) {
		t.Helper()
		h := map[string]string{"X-CSRF-Token": csrf}
		for k, v := range header {
			h[k] = v
		}
		r := call(t, ts, "POST", "/api/admin/workers", `{"name":"synthetic-gpu","max_concurrency":1}`, h, cookie)
		var out struct {
			Bundle struct {
				Env      string `json:"env"`
				Insecure bool   `json:"insecure_api_url"`
			} `json:"bundle"`
		}
		if r.status != 201 || json.Unmarshal(r.body, &out) != nil {
			t.Fatalf("worker registration: %d %s", r.status, r.body)
		}
		for _, line := range strings.Split(out.Bundle.Env, "\n") {
			if v, ok := strings.CutPrefix(line, "SNITCH_API_URL="); ok {
				return v, out.Bundle.Insecure
			}
		}
		t.Fatal("deployment file without SNITCH_API_URL")
		return "", false
	}

	// A first instance without any address setting installs the administrator and Discord.
	directConfig := base(nil)
	direct, directServer := start(t, directConfig, false)
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	boot := call(t, directServer, "POST", "/api/setup", `{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`, map[string]string{"Origin": directServer.URL}, nil)
	if boot.status != 201 || len(boot.cookies) != 1 || boot.cookies[0].Secure {
		t.Fatalf("start without address setting over HTTP: %d %s %+v", boot.status, boot.body, boot.cookies)
	}
	if err := direct.Store.Set(ctx, "discord_client_id", "fixture-client"); err != nil {
		t.Fatal(err)
	}
	if err := direct.Store.Set(ctx, "discord_client_secret", "fixture-secret"); err != nil {
		t.Fatal(err)
	}

	t.Run("direct request uses its own scheme and host", func(t *testing.T) {
		origin := map[string]string{"Origin": directServer.URL}
		cookie, csrf := login(t, directServer, origin)
		if cookie.Secure {
			t.Fatal("cookie secured on a plain HTTP request")
		}
		want := directServer.URL + "/api/auth/discord/callback"
		if got := redirectURI(t, directServer, map[string]string{"Origin": directServer.URL}); got != want {
			t.Fatalf("discord return address %q, want %q", got, want)
		}
		if got := callbackShown(t, directServer, nil, cookie); got != want {
			t.Fatalf("displayed return address %q, want %q", got, want)
		}
		apiURL, insecure := workerAPIURL(t, directServer, origin, cookie, csrf)
		if apiURL != directServer.URL || !insecure {
			t.Fatalf("worker deployment file address %q insecure=%v", apiURL, insecure)
		}
		alias := "snitch.lan:9000"
		if got := redirectURI(t, directServer, map[string]string{"Host": alias, "Origin": "http://" + alias}); got != "http://"+alias+"/api/auth/discord/callback" {
			t.Fatalf("return address for another Host header: %q", got)
		}
		if r := loginStatus(t, directServer, map[string]string{"Host": alias, "Origin": "http://" + alias}); r.status != 200 {
			t.Fatalf("origin matching the requested host: %d %s", r.status, r.body)
		}
	})

	t.Run("an origin that does not match the request address is refused", func(t *testing.T) {
		host := strings.TrimPrefix(directServer.URL, "http://")
		for name, header := range map[string]map[string]string{
			"other site":                     {"Origin": "https://attacker.invalid"},
			"other scheme":                   {"Origin": "https://" + host},
			"other port":                     {"Origin": "http://127.0.0.1:1"},
			"no origin":                      {},
			"opaque origin":                  {"Origin": "null"},
			"proxy headers do not rescue it": {"Origin": "https://attacker.invalid", "X-Forwarded-Host": "attacker.invalid", "X-Forwarded-Proto": "https"},
		} {
			if r := loginStatus(t, directServer, header); r.status != 403 || !bytes.Contains(r.body, []byte("origin_rejected")) {
				t.Fatalf("%s: %d %s", name, r.status, r.body)
			}
		}
	})

	t.Run("proxy headers from an untrusted peer are ignored", func(t *testing.T) {
		forged := map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "public.example"}
		withOrigin := func(origin string) map[string]string {
			h := map[string]string{"Origin": origin}
			for k, v := range forged {
				h[k] = v
			}
			return h
		}
		// No trusted proxy configured, and a trusted range that does not contain the peer.
		for name, trusted := range map[string][]netip.Prefix{"none": nil, "other range": {netip.MustParsePrefix("10.0.0.0/8")}} {
			_, ts := start(t, base(func(c *config.Config) { c.TrustedProxies = trusted }), false)
			if r := loginStatus(t, ts, withOrigin("https://public.example")); r.status != 403 {
				t.Fatalf("%s: forged forwarded origin accepted: %d", name, r.status)
			}
			cookie, _ := login(t, ts, withOrigin(ts.URL))
			if cookie.Secure {
				t.Fatalf("%s: forged forwarded scheme secured the cookie", name)
			}
			if got := redirectURI(t, ts, withOrigin(ts.URL)); got != ts.URL+"/api/auth/discord/callback" {
				t.Fatalf("%s: forged return address %q", name, got)
			}
		}
	})

	t.Run("proxy headers from a trusted proxy give the public address", func(t *testing.T) {
		_, ts := start(t, base(func(c *config.Config) { c.TrustedProxies = loopback }), false)
		forwarded := map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "snitch.example.org"}
		with := func(origin string) map[string]string {
			h := map[string]string{"Origin": origin}
			for k, v := range forwarded {
				h[k] = v
			}
			return h
		}
		cookie, csrf := login(t, ts, with("https://snitch.example.org"))
		if !cookie.Secure {
			t.Fatal("cookie not secured behind an HTTPS proxy")
		}
		if r := loginStatus(t, ts, with(ts.URL)); r.status != 403 {
			t.Fatalf("the proxy-facing address was accepted as origin: %d", r.status)
		}
		if r := loginStatus(t, ts, with("http://snitch.example.org")); r.status != 403 {
			t.Fatalf("an origin with the wrong scheme was accepted: %d", r.status)
		}
		want := "https://snitch.example.org/api/auth/discord/callback"
		if got := redirectURI(t, ts, with("https://snitch.example.org")); got != want {
			t.Fatalf("discord return address %q, want %q", got, want)
		}
		if got := callbackShown(t, ts, forwarded, cookie); got != want {
			t.Fatalf("displayed return address %q, want %q", got, want)
		}
		apiURL, insecure := workerAPIURL(t, ts, with("https://snitch.example.org"), cookie, csrf)
		if apiURL != "https://snitch.example.org" || insecure {
			t.Fatalf("worker deployment file address %q insecure=%v", apiURL, insecure)
		}
		// A default port is written the way browsers write origins.
		h := with("https://snitch.example.org")
		h["X-Forwarded-Host"] = "snitch.example.org:443"
		if r := loginStatus(t, ts, h); r.status != 200 {
			t.Fatalf("default port in forwarded host: %d %s", r.status, r.body)
		}
		// Malformed values fall back to the request itself.
		bad := map[string]string{"Origin": ts.URL, "X-Forwarded-Proto": "javascript", "X-Forwarded-Host": "evil.example/path?x=1"}
		if got := redirectURI(t, ts, bad); got != ts.URL+"/api/auth/discord/callback" {
			t.Fatalf("malformed forwarded values honored: %q", got)
		}
	})

	t.Run("a configured public address wins over the request", func(t *testing.T) {
		_, ts := start(t, base(func(c *config.Config) { c.PublicURL = "https://pinned.example"; c.TrustedProxies = loopback }), false)
		cookie, _ := login(t, ts, map[string]string{"Origin": "https://pinned.example"})
		if !cookie.Secure {
			t.Fatal("cookie not secured for an HTTPS public address")
		}
		for name, origin := range map[string]string{"request address": ts.URL, "forwarded address": "https://snitch.example.org"} {
			h := map[string]string{"Origin": origin, "X-Forwarded-Host": "snitch.example.org", "X-Forwarded-Proto": "https"}
			if r := loginStatus(t, ts, h); r.status != 403 {
				t.Fatalf("%s accepted over the configured address: %d", name, r.status)
			}
		}
		if got := redirectURI(t, ts, map[string]string{"Origin": "https://pinned.example", "X-Forwarded-Host": "snitch.example.org"}); got != "https://pinned.example/api/auth/discord/callback" {
			t.Fatalf("discord return address %q", got)
		}
	})

	t.Run("the session cookie follows the scheme unless a setting decides", func(t *testing.T) {
		_, tlsServer := start(t, base(nil), true)
		cookie, _ := login(t, tlsServer, map[string]string{"Origin": tlsServer.URL})
		if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("cookie over HTTPS: %+v", cookie)
		}
		_, forcedOff := start(t, base(func(c *config.Config) { c.SecureExplicit = true }), true)
		if cookie, _ = login(t, forcedOff, map[string]string{"Origin": forcedOff.URL}); cookie.Secure {
			t.Fatal("explicit insecure cookie setting ignored")
		}
		_, forcedOn := start(t, base(func(c *config.Config) { c.Secure = true; c.SecureExplicit = true }), false)
		if cookie, _ = login(t, forcedOn, map[string]string{"Origin": forcedOn.URL}); !cookie.Secure {
			t.Fatal("explicit secure cookie setting ignored")
		}
	})
}
