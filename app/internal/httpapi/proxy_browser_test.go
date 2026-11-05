package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"snitch/internal/config"
)

// A TLS terminator followed by an HTTP proxy can replace X-Forwarded-Proto with
// http. Replay the browser's setup request without a database: an empty payload
// must reach validation (400), rather than being rejected by the origin gate (403).
func TestSetupOriginBehindHTTPProxyChain(t *testing.T) {
	s := &Server{Config: config.Config{TrustedProxies: config.DefaultTrustedProxies()}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	r := httptest.NewRequest("POST", "http://snitch.example/api/setup/code", strings.NewReader(`{}`))
	r.RemoteAddr = "172.18.0.2:12345"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://snitch.example")
	r.Header.Set("Referer", "https://snitch.example/")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.Header.Set("X-Forwarded-Host", "snitch.example")
	r.Header.Set("X-Forwarded-Proto", "http")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "invalid_request") {
		t.Fatalf("setup payload should reach validation, got %d %s", w.Code, w.Body.String())
	}
	if !s.secureCookies(r) {
		t.Fatal("the HTTPS browser must receive Secure session cookies")
	}
}

func TestProxyBrowserHTTPSBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, method, origin, referer, fetchSite, peer, want string
	}{
		{"read uses same-origin referer", "GET", "", "https://snitch.example/setup?step=3", "same-origin", "172.18.0.2:42", "https://snitch.example"},
		{"head uses referer", "HEAD", "", "https://snitch.example/", "same-origin", "172.18.0.2:42", "https://snitch.example"},
		{"default HTTPS port", "POST", "https://snitch.example:443", "", "same-origin", "172.18.0.2:42", "https://snitch.example"},
		{"foreign origin", "POST", "https://attacker.invalid", "https://snitch.example/", "same-origin", "172.18.0.2:42", "http://snitch.example"},
		{"foreign port", "POST", "https://snitch.example:8443", "", "same-origin", "172.18.0.2:42", "http://snitch.example"},
		{"foreign read referer", "GET", "", "https://attacker.invalid/", "same-origin", "172.18.0.2:42", "http://snitch.example"},
		{"cross-site metadata", "POST", "https://snitch.example", "", "cross-site", "172.18.0.2:42", "http://snitch.example"},
		{"same-site is insufficient", "POST", "https://snitch.example", "", "same-site", "172.18.0.2:42", "http://snitch.example"},
		{"missing metadata", "POST", "https://snitch.example", "", "", "172.18.0.2:42", "http://snitch.example"},
		{"public peer", "POST", "https://snitch.example", "", "same-origin", "203.0.113.1:42", "http://snitch.example"},
		{"mutation without origin", "POST", "", "https://snitch.example/", "same-origin", "172.18.0.2:42", "http://snitch.example"},
		{"opaque origin takes precedence", "POST", "null", "https://snitch.example/", "same-origin", "172.18.0.2:42", "http://snitch.example"},
		{"HTTP origin takes precedence", "POST", "http://snitch.example", "https://snitch.example/", "same-origin", "172.18.0.2:42", "http://snitch.example"},
		{"origin with credentials", "POST", "https://user@snitch.example", "", "same-origin", "172.18.0.2:42", "http://snitch.example"},
		{"origin with path", "POST", "https://snitch.example/path", "", "same-origin", "172.18.0.2:42", "http://snitch.example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{Config: config.Config{TrustedProxies: config.DefaultTrustedProxies()}}
			r := httptest.NewRequest(tc.method, "http://snitch.example/api/setup/code", nil)
			r.RemoteAddr = tc.peer
			r.Header.Set("X-Forwarded-Proto", "http")
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Referer", tc.referer)
			r.Header.Set("Sec-Fetch-Site", tc.fetchSite)
			if got := s.publicOrigin(r); got != tc.want {
				t.Fatalf("public origin = %q, want %q", got, tc.want)
			}
			if tc.method == http.MethodPost && tc.origin != tc.want {
				w := httptest.NewRecorder()
				s.Handler().ServeHTTP(w, r)
				if w.Code != 403 || !strings.Contains(w.Body.String(), "origin_rejected") {
					t.Fatalf("unexpected origin accepted: %d %s", w.Code, w.Body.String())
				}
			}
		})
	}
}
