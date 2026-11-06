//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

type installHarness struct {
	t      *testing.T
	ts     *httptest.Server
	server *Server
	admin  *http.Client
	anon   *http.Client
	csrf   string
}

func newInstallHarness(t *testing.T, pool *pgxpool.Pool, tls bool) *installHarness {
	t.Helper()
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), UploadRoot: t.TempDir()}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	var ts *httptest.Server
	if tls {
		ts = httptest.NewTLSServer(server.Handler())
	} else {
		ts = httptest.NewServer(server.Handler())
	}
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	h := &installHarness{t: t, ts: ts, server: server, admin: ts.Client(), anon: &http.Client{Transport: ts.Client().Transport}}
	h.admin.Jar = jar
	h.do(h.admin, http.MethodPost, ts.URL+"/api/setup", []byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`), "", "", true, http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(h.do(h.admin, http.MethodGet, ts.URL+"/api/auth/me", nil, "", "", false, http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	h.csrf = me.CSRF
	return h
}

func (h *installHarness) do(c *http.Client, method, url string, body []byte, csrf, token string, origin bool, want int) []byte {
	h.t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		h.t.Fatal(err)
	}
	if origin {
		req.Header.Set("Origin", h.ts.URL)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		h.t.Fatalf("%s %s: got %d want %d: %s", method, url, resp.StatusCode, want, data)
	}
	return data
}

func (h *installHarness) adminCall(method, path string, body []byte, want int) []byte {
	h.t.Helper()
	return h.do(h.admin, method, h.ts.URL+path, body, h.csrf, "", true, want)
}

func (h *installHarness) get(url, token string, want int) []byte {
	h.t.Helper()
	return h.do(h.anon, http.MethodGet, url, nil, "", token, false, want)
}

var installCodePattern = regexp.MustCompile(`/api/worker/install/([A-Za-z0-9_-]{43})`)

func (h *installHarness) register(query string) (id, token, command string, raw []byte) {
	h.t.Helper()
	raw = h.adminCall(http.MethodPost, "/api/admin/workers"+query, []byte(`{"name":"synthetic-linux","max_concurrency":1}`), http.StatusCreated)
	var r struct {
		ID      string `json:"id"`
		Token   string `json:"token"`
		Command string `json:"install_command"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		h.t.Fatal(err)
	}
	return r.ID, r.Token, r.Command, raw
}

func (h *installHarness) inviteURL(command string) string {
	h.t.Helper()
	m := installCodePattern.FindStringSubmatch(command)
	if m == nil {
		h.t.Fatalf("no invitation in command %q", command)
	}
	return h.ts.URL + "/api/worker/install/" + m[1]
}

func scriptToken(t *testing.T, script string) string {
	t.Helper()
	m := regexp.MustCompile(`seed_token=\$\(printf '%s' '([A-Za-z0-9+/=]+)' \| base64 -d\)`).FindStringSubmatch(script)
	if m == nil {
		t.Fatalf("script carries no worker token:\n%s", script)
	}
	raw, err := base64.StdEncoding.DecodeString(m[1])
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func testWorkerInstallInvitation(t *testing.T, pool *pgxpool.Pool) {
	h := newInstallHarness(t, pool, true)
	id, token, command, _ := h.register("")
	if strings.Contains(command, token) {
		t.Fatal("install command leaks the Worker token")
	}
	url := h.inviteURL(command)

	// Unknown invitation: not found.
	h.get(h.ts.URL+"/api/worker/install/"+strings.Repeat("A", 43), "", http.StatusNotFound)

	// First download succeeds and delivers the durable identity.
	script := string(h.get(url, "", http.StatusOK))
	if scriptToken(t, script) != token {
		t.Fatal("script does not carry the Worker token created with the invitation")
	}
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/pulse", []byte(`{"capacity":1}`), "", token, false, http.StatusOK)

	// Replay: the invitation is single-use.
	body := h.get(url, "", http.StatusGone)
	if !bytes.Contains(body, []byte("install_invite_used")) || bytes.Contains(body, []byte(token)) {
		t.Fatalf("unexpected replay answer: %s", body)
	}

	// Deleting a Worker that never connected does not revive its invitation.
	id2, token2, command2, _ := h.register("")
	url2 := h.inviteURL(command2)
	h.get(url2, "", http.StatusOK)
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+id2, nil, http.StatusNoContent)
	if resp, err := h.anon.Get(url2); err != nil || resp.StatusCode == http.StatusOK {
		t.Fatalf("deleted Worker invitation was served again: %v", err)
	}
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/pulse", []byte(`{"capacity":1}`), "", token2, false, http.StatusUnauthorized)

	// Resume: the launcher reads current configuration with the durable token only.
	h.adminCall(http.MethodPut, "/api/admin/transcription-settings", mustJSON(t, map[string]any{"mode": "enabled", "endpoint": "http://127.0.0.1:9000", "model": "fixture-model", "language": "fr"}), http.StatusOK)
	cfg := h.get(h.ts.URL+"/api/worker/transcription-config", token, http.StatusOK)
	if !bytes.Contains(cfg, []byte(`"model":"fixture-model"`)) {
		t.Fatalf("resume did not read the current configuration: %s", cfg)
	}
	h.get(h.ts.URL+"/api/worker/transcription-config", "", http.StatusUnauthorized)

	// Lost credentials: a new invitation replaces the token; the old one stops working.
	// An invitation that was never consumed belongs to the identity being replaced.
	staleID, _, staleCommand, _ := h.register("")
	staleURL := h.inviteURL(staleCommand)
	renewed := h.adminCall(http.MethodPost, "/api/admin/workers/"+id+"/token", nil, http.StatusCreated)
	var rr struct {
		ID      string `json:"id"`
		Token   string `json:"token"`
		Command string `json:"install_command"`
	}
	if err := json.Unmarshal(renewed, &rr); err != nil {
		t.Fatal(err)
	}
	if rr.ID == id {
		t.Fatal("a lost token must create a new identity")
	}
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/pulse", []byte(`{"capacity":1}`), "", token, false, http.StatusUnauthorized)
	h.get(h.ts.URL+"/api/worker/transcription-config", token, http.StatusUnauthorized)
	h.adminCall(http.MethodPost, "/api/admin/workers/"+id+"/token", nil, http.StatusNotFound)
	h.adminCall(http.MethodPost, "/api/admin/workers/"+staleID+"/token", nil, http.StatusCreated)
	if resp, err := h.anon.Get(staleURL); err != nil || resp.StatusCode == http.StatusOK {
		t.Fatalf("unconsumed invitation of the replaced identity was served: %v", err)
	}
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/pulse", []byte(`{"capacity":1}`), "", token, false, http.StatusUnauthorized)
	newScript := string(h.get(h.inviteURL(rr.Command), "", http.StatusOK))
	if scriptToken(t, newScript) != rr.Token || rr.Token == token {
		t.Fatal("renewed invitation does not carry the new token")
	}
	h.get(h.inviteURL(rr.Command), "", http.StatusGone)

	// Concurrent downloads: exactly one wins.
	_, _, command3, _ := h.register("")
	url3 := h.inviteURL(command3)
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := h.anon.Get(url3)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("%d concurrent downloads succeeded, want 1", ok)
	}
}

func testWorkerInstallRefusesInsecureOrigin(t *testing.T, pool *pgxpool.Pool) {
	h := newInstallHarness(t, pool, false)
	_, _, command, raw := h.register("")
	if command != "" || !bytes.Contains(raw, []byte(`"install_command_unavailable":"insecure_origin"`)) {
		t.Fatalf("http origin must be refused with a reason: %s", raw)
	}
	_, _, command, raw = h.register("?allow_insecure_install=true")
	if !strings.Contains(command, "http://") || bytes.Contains(raw, []byte("install_command_unavailable")) {
		t.Fatalf("explicit opt-in must produce the command: %s", raw)
	}
}
