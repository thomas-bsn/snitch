//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/auth"
)

// hfSyntheticToken is a made-up token in the expected format. It must never appear in any answer, log or stored
// plain value.
const hfSyntheticToken = "hf_SyntheticTokenForTestsOnly0123456789abcdef"

// hfSyntheticReplacement is a second made-up token, used for the replacement and deletion cases.
const hfSyntheticReplacement = "hf_SyntheticReplacementTokenOnly0123456789"

type hfAdminView struct {
	Configured  bool   `json:"configured"`
	Fingerprint string `json:"fingerprint"`
	Unreadable  bool   `json:"unreadable"`
	Workers     []struct {
		ID        string `json:"id"`
		Source    string `json:"source"`
		State     string `json:"state"`
		ErrorCode string `json:"error_code"`
	} `json:"workers"`
}

type hfWorkerAnswer struct {
	Version string  `json:"version"`
	Token   *string `json:"token"`
}

// hfLogin signs in another administrator (not the Super-administrator) and returns its cookie client and CSRF token.
func hfLogin(t *testing.T, h *installHarness, pool *pgxpool.Pool, username, role string) (*http.Client, string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO users(username,password_hash,role,approved) VALUES($1,$2,$3,true)`, username, auth.Password("synthetic-password-hf"), role); err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Transport: h.ts.Client().Transport, Jar: jar}
	h.do(c, http.MethodPost, h.ts.URL+"/api/auth/login", []byte(`{"username":"`+username+`","password":"synthetic-password-hf"}`), "", "", true, http.StatusOK)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(h.do(c, http.MethodGet, h.ts.URL+"/api/auth/me", nil, "", "", false, http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	return c, me.CSRF
}

func hfAdminGet(t *testing.T, h *installHarness) ([]byte, hfAdminView) {
	t.Helper()
	body := h.adminCall(http.MethodGet, "/api/admin/hf-token", nil, http.StatusOK)
	var view hfAdminView
	if err := json.Unmarshal(body, &view); err != nil {
		t.Fatal(err)
	}
	return body, view
}

func hfWorkerRow(view hfAdminView, id string) (state, source string) {
	for _, w := range view.Workers {
		if w.ID == id {
			return w.State, w.Source
		}
	}
	return "", ""
}

func hfWorkerFetch(t *testing.T, h *installHarness, token string) hfWorkerAnswer {
	t.Helper()
	var answer hfWorkerAnswer
	if err := json.Unmarshal(h.get(h.ts.URL+"/api/worker/hf-token", token, http.StatusOK), &answer); err != nil {
		t.Fatal(err)
	}
	return answer
}

func hfAck(h *installHarness, token, body string, want int) {
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/hf-token/ack", []byte(body), "", token, false, want)
}

// testHFToken covers #194: the central Hugging Face token is written once by the Super-administrator, never read back,
// distributed only to authenticated, non-revoked Workers through dedicated endpoints, acknowledged, and kept or purged
// according to its provenance. Negative cases come first: no read exposes the value, no unauthenticated or revoked
// Worker receives anything, writes are reserved to the Super-administrator and protected by CSRF, and no log, error or
// answer carries the value.
func testHFToken(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newInstallHarness(t, pool, true)
	var logs bytes.Buffer
	h.server.Log = slog.New(slog.NewJSONHandler(&logs, nil))

	id, token, _, _ := h.register("")
	otherID, otherToken, _, _ := h.register("")

	// Negative: anonymous and non-Super-administrator callers never write; reads stay free of the value.
	h.do(h.anon, http.MethodGet, h.ts.URL+"/api/admin/hf-token", nil, "", "", false, http.StatusUnauthorized)
	h.do(h.anon, http.MethodPut, h.ts.URL+"/api/admin/hf-token", []byte(`{"token":"`+hfSyntheticToken+`"}`), "", "", true, http.StatusUnauthorized)
	admin, adminCSRF := hfLogin(t, h, pool, "hf-administrator", "admin")
	h.do(admin, http.MethodPut, h.ts.URL+"/api/admin/hf-token", []byte(`{"token":"`+hfSyntheticToken+`"}`), adminCSRF, "", true, http.StatusForbidden)
	h.do(admin, http.MethodDelete, h.ts.URL+"/api/admin/hf-token", nil, adminCSRF, "", true, http.StatusForbidden)
	if raw := h.do(admin, http.MethodGet, h.ts.URL+"/api/admin/hf-token", nil, "", "", false, http.StatusOK); bytes.Contains(raw, []byte(hfSyntheticToken)) {
		t.Fatalf("administrator read exposes the value: %s", raw)
	}

	// Negative: the write needs the Super-administrator's session and its CSRF token.
	h.do(h.admin, http.MethodPut, h.ts.URL+"/api/admin/hf-token", []byte(`{"token":"`+hfSyntheticToken+`"}`), "", "", true, http.StatusForbidden)
	h.do(h.admin, http.MethodPut, h.ts.URL+"/api/admin/hf-token", []byte(`{"token":"`+hfSyntheticToken+`"}`), "wrong-csrf", "", true, http.StatusForbidden)
	h.do(h.admin, http.MethodDelete, h.ts.URL+"/api/admin/hf-token", nil, "", "", true, http.StatusForbidden)

	// Negative: malformed values are refused, and the refusal does not echo them.
	refused := h.do(h.admin, http.MethodPut, h.ts.URL+"/api/admin/hf-token", []byte(`{"token":"hf_bad value with spaces"}`), h.csrf, "", true, http.StatusBadRequest)
	if bytes.Contains(refused, []byte("bad value")) {
		t.Fatalf("refusal echoes the value: %s", refused)
	}

	// Negative: Workers without a valid, non-revoked identity receive nothing and acknowledge nothing.
	for _, bearer := range []string{"", "rw_unknown-synthetic-token"} {
		h.get(h.ts.URL+"/api/worker/hf-token", bearer, http.StatusUnauthorized)
		h.get(h.ts.URL+"/api/worker/hf-token/version", bearer, http.StatusUnauthorized)
		hfAck(h, bearer, `{"version":"1","outcome":"applied","source":"central"}`, http.StatusUnauthorized)
	}

	// Save once: the value is kept sealed, never returned.
	h.do(h.admin, http.MethodPut, h.ts.URL+"/api/admin/hf-token", []byte(`{"token":"`+hfSyntheticToken+`"}`), h.csrf, "", true, http.StatusNoContent)
	raw, view := hfAdminGet(t, h)
	if !view.Configured || view.Fingerprint == "" || view.Unreadable || bytes.Contains(raw, []byte(hfSyntheticToken)) {
		t.Fatalf("admin view after save: %s", raw)
	}
	if bytes.Contains(logs.Bytes(), []byte(hfSyntheticToken)) || bytes.Contains(logs.Bytes(), []byte(hfFingerprint(hfSyntheticToken))) {
		t.Fatalf("save log carries the value or its fingerprint: %s", logs.String())
	}
	var stored int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM settings WHERE position(convert_to($1,'UTF8') in value) > 0`, hfSyntheticToken).Scan(&stored); err != nil || stored != 0 {
		t.Fatalf("plain value stored: %d %v", stored, err)
	}

	// The optional wizard step is done once a central token is stored, and no other step changes with it.
	var progress struct {
		Steps []struct {
			Step  string `json:"step"`
			State string `json:"state"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/setup-progress", nil, http.StatusOK), &progress); err != nil {
		t.Fatal(err)
	}
	for _, step := range progress.Steps {
		if step.Step == "huggingface" && step.State != "done" {
			t.Fatalf("wizard step not done after the save: %s", step.State)
		}
	}

	// The Worker obtains the value through its own endpoints; the version endpoint carries no value.
	first := hfWorkerFetch(t, h, token)
	if first.Token == nil || *first.Token != hfSyntheticToken || first.Version == "" || first.Version == "0" {
		t.Fatalf("worker did not receive the token: %+v", first)
	}
	versionBody := h.get(h.ts.URL+"/api/worker/hf-token/version", token, http.StatusOK)
	var version hfWorkerAnswer
	if err := json.Unmarshal(versionBody, &version); err != nil || version.Version != first.Version || bytes.Contains(versionBody, []byte(hfSyntheticToken)) {
		t.Fatalf("version endpoint: %s %v", versionBody, err)
	}

	// Acknowledgement: stale and unknown outcomes are refused; the right one is recorded and shown as synchronised.
	hfAck(h, token, `{"version":"1","outcome":"applied","source":"central"}`, http.StatusConflict)
	hfAck(h, token, `{"version":"`+first.Version+`","outcome":"forged","source":"central"}`, http.StatusBadRequest)
	// Inconsistent combinations are refused: an error code only with a failure, a source matching the outcome.
	hfAck(h, token, `{"version":"`+first.Version+`","outcome":"applied","source":"central","error_code":"hf_token_store_failed"}`, http.StatusBadRequest)
	hfAck(h, token, `{"version":"`+first.Version+`","outcome":"failed","source":"central","error_code":"other_code"}`, http.StatusBadRequest)
	hfAck(h, token, `{"version":"`+first.Version+`","outcome":"kept_local","source":"central"}`, http.StatusBadRequest)
	hfAck(h, token, `{"version":"`+first.Version+`","outcome":"applied","source":"central"}`, http.StatusNoContent)
	_, view = hfAdminGet(t, h)
	if state, source := hfWorkerRow(view, id); state != "synced" || source != "central" {
		t.Fatalf("acknowledged worker: %s %s", state, source)
	}

	// Replacement: the other Worker has not acknowledged the new version yet, so it is not synchronised.
	h.do(h.admin, http.MethodPut, h.ts.URL+"/api/admin/hf-token", []byte(`{"token":"`+hfSyntheticReplacement+`"}`), h.csrf, "", true, http.StatusNoContent)
	_, view = hfAdminGet(t, h)
	if state, _ := hfWorkerRow(view, id); state != "pending" {
		t.Fatalf("replaced token not shown as pending: %s", state)
	}
	second := hfWorkerFetch(t, h, otherToken)
	if second.Token == nil || *second.Token != hfSyntheticReplacement || second.Version == first.Version {
		t.Fatalf("replacement not served: %+v", second)
	}

	// A removal is refused while a token is stored, and a copy is never acknowledged as applied for the wrong state.
	hfAck(h, token, `{"version":"`+second.Version+`","outcome":"removed","source":"none"}`, http.StatusConflict)

	// Local priority: a Worker keeping its own token acknowledges it, and the admin view says so.
	hfAck(h, token, `{"version":"`+second.Version+`","outcome":"kept_local","source":"local"}`, http.StatusNoContent)
	_, view = hfAdminGet(t, h)
	if state, source := hfWorkerRow(view, id); state != "synced" || source != "local" {
		t.Fatalf("local priority not shown: %s %s", state, source)
	}

	// A failed application is reported with its stable cause only.
	hfAck(h, otherToken, `{"version":"`+second.Version+`","outcome":"failed","source":"central","error_code":"hf_token_store_failed"}`, http.StatusNoContent)
	_, view = hfAdminGet(t, h)
	if state, _ := hfWorkerRow(view, id); state != "synced" {
		t.Fatalf("local worker affected by another failure: %s", state)
	}
	otherState, _ := hfWorkerRow(view, otherID)
	if otherState != "error" {
		t.Fatalf("failed application not reported: %s", otherState)
	}

	// Deletion: the central copy is purged at the next version; the answer carries no value.
	h.do(h.admin, http.MethodDelete, h.ts.URL+"/api/admin/hf-token", nil, h.csrf, "", true, http.StatusNoContent)
	raw, view = hfAdminGet(t, h)
	if view.Configured || bytes.Contains(raw, []byte(hfSyntheticReplacement)) {
		t.Fatalf("deleted token still visible: %s", raw)
	}
	removed := hfWorkerFetch(t, h, otherToken)
	if removed.Token != nil || removed.Version == second.Version {
		t.Fatalf("deletion not served as a purge: %+v", removed)
	}
	hfAck(h, otherToken, `{"version":"`+removed.Version+`","outcome":"applied","source":"central"}`, http.StatusConflict)
	hfAck(h, otherToken, `{"version":"`+removed.Version+`","outcome":"removed","source":"none"}`, http.StatusNoContent)

	// Unreadable value (master key changed or row damaged): reported, counted as absent, never served or purged.
	if _, err := pool.Exec(ctx, `UPDATE settings SET value=$1 WHERE key='hf_token_central'`, []byte("damaged")); err != nil {
		t.Fatal(err)
	}
	raw, view = hfAdminGet(t, h)
	if view.Configured || !view.Unreadable {
		t.Fatalf("unreadable value not reported: %s", raw)
	}
	unreadable := hfWorkerFetch(t, h, token)
	if unreadable.Token != nil || unreadable.Version != "0" {
		t.Fatalf("unreadable value served: %+v", unreadable)
	}
	h.do(h.admin, http.MethodPut, h.ts.URL+"/api/admin/hf-token", []byte(`{"token":"`+hfSyntheticToken+`"}`), h.csrf, "", true, http.StatusNoContent)

	// Revoked Worker: refused everywhere, even with the identity it had until now.
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+id, []byte(`{"reason":"machine retired"}`), http.StatusNoContent)
	h.get(h.ts.URL+"/api/worker/hf-token", token, http.StatusUnauthorized)
	h.get(h.ts.URL+"/api/worker/hf-token/version", token, http.StatusUnauthorized)
	hfAck(h, token, `{"version":"`+first.Version+`","outcome":"applied","source":"central"}`, http.StatusUnauthorized)

	// Nothing the application logged carries the value.
	for _, secret := range []string{hfSyntheticToken, hfSyntheticReplacement, "bad value"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("value in logs: %s", logs.String())
		}
	}
	if !strings.Contains(logs.String(), "hf token") {
		t.Fatal("the save and the deletion are not audited")
	}
}
