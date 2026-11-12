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
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

// newNotificationHarness is an installed server with its Super-administrator, never started: no background loop
// runs, so a test drives the webhook deliveries itself and may set the server's test fields first.
func newNotificationHarness(t *testing.T, pool *pgxpool.Pool, logs io.Writer) *voiceHarness {
	t.Helper()
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	api := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(logs, nil)))
	ts := httptest.NewServer(api.Handler())
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	h := &voiceHarness{t: t, api: api, pool: pool, ts: ts, admin: &http.Client{Jar: jar}}
	h.adminCall(http.MethodPost, "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(h.adminCall(http.MethodGet, "/api/auth/me", nil, http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	h.csrf = me.CSRF
	return h
}

// requestAccess signs up a visitor while approval is required: the account waits for an administrator.
func (h *voiceHarness) requestAccess(name string) int64 {
	h.t.Helper()
	ctx := context.Background()
	if _, err := h.pool.Exec(ctx, `UPDATE installation SET registration_approval=true`); err != nil {
		h.t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	visitor := &voiceUser{name: name, client: &http.Client{Jar: jar}}
	var out struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(h.userCall(visitor, http.MethodPost, "/api/auth/register", map[string]string{"username": name, "password": "synthetic-password"}, http.StatusCreated), &out); err != nil || out.Status != "pending" {
		h.t.Fatalf("a sign-up waits for approval: %+v %v", out, err)
	}
	var id int64
	if err := h.pool.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, name).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

// accountRequests lists the Demande d'accès notifications a user sees.
func (h *voiceHarness) accountRequests(u *voiceUser) []notificationView {
	h.t.Helper()
	var page notificationListView
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/notifications", nil, http.StatusOK), &page); err != nil {
		h.t.Fatal(err)
	}
	var out []notificationView
	for _, item := range page.Items {
		if item.Kind == kindAccountRequest {
			out = append(out, item)
		}
	}
	return out
}

// testAccountRequests covers tranche A of #283: a sign-up waiting for approval raises one notification for the
// administrators only, live, leading to the request; approving or refusing it resolves it for everybody.
func testAccountRequests(t *testing.T, pool *pgxpool.Pool) {
	h := newNotificationHarness(t, pool, io.Discard)
	ctx := context.Background()
	super := h.adminUser()
	ops := h.newUser("ops")
	member := h.newUser("member")
	if _, err := pool.Exec(ctx, `UPDATE users SET role='admin' WHERE username='ops'`); err != nil {
		t.Fatal(err)
	}
	liveEvents := func(u string) int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM events WHERE topic='user:'||(SELECT id FROM users WHERE username=$1) AND kind='notifications'`, u).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := liveEvents("administrator")

	visitor := h.requestAccess("visitor")
	if liveEvents("administrator") == before {
		t.Fatal("the bell of an administrator is told of the request live")
	}
	for _, admin := range []*voiceUser{super, ops} {
		got := h.accountRequests(admin)
		if len(got) != 1 || got[0].State != "active" || got[0].Read || got[0].AccountName != "visitor" || got[0].AccountID == nil || *got[0].AccountID != visitor {
			t.Fatalf("%s is told of the request: %+v", admin.name, got)
		}
	}
	if got := h.accountRequests(member); len(got) != 0 {
		t.Fatalf("a member is never told of a request: %+v", got)
	}
	var page notificationListView
	if err := json.Unmarshal(h.userCall(super, http.MethodGet, "/api/notifications?filter=errors", nil, http.StatusOK), &page); err != nil || page.Total != 0 {
		t.Fatalf("a request is not an error: %+v %v", page, err)
	}

	// One logical event: a repeated sign-up is refused and a repeated raise changes nothing.
	jar, _ := cookiejar.New(nil)
	h.userCall(&voiceUser{name: "again", client: &http.Client{Jar: jar}}, http.MethodPost, "/api/auth/register", map[string]string{"username": "visitor", "password": "synthetic-password"}, http.StatusConflict)
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error { return raiseAccountRequest(ctx, tx, visitor) }); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind='account_request' AND account_id=$1`, visitor).Scan(&count); err != nil || count != 1 {
		t.Fatalf("one notification per request: %d %v", count, err)
	}

	// Members neither decide nor read a request.
	h.userCall(member, http.MethodPost, "/api/admin/users/"+itoa(visitor)+"/approve", nil, http.StatusForbidden)
	h.userCall(member, http.MethodPost, "/api/admin/users/"+itoa(visitor)+"/refuse", nil, http.StatusForbidden)
	h.userCall(member, http.MethodGet, "/api/admin/users", nil, http.StatusForbidden)

	// Approval resolves it for every administrator.
	h.userCall(ops, http.MethodPost, "/api/admin/users/"+itoa(visitor)+"/approve", nil, http.StatusOK)
	for _, admin := range []*voiceUser{super, ops} {
		if got := h.accountRequests(admin); len(got) != 1 || got[0].State != "resolved" || got[0].AccountOutcome != "approved" {
			t.Fatalf("the approval resolves the request for %s: %+v", admin.name, got)
		}
	}
	h.userCall(super, http.MethodPost, "/api/admin/users/"+itoa(visitor)+"/refuse", nil, http.StatusNotFound)

	// Refusal retires the account and forgets its name.
	intruder := h.requestAccess("intruder")
	h.userCall(super, http.MethodPost, "/api/admin/users/"+itoa(intruder)+"/refuse", nil, http.StatusOK)
	h.userCall(super, http.MethodPost, "/api/admin/users/"+itoa(intruder)+"/refuse", nil, http.StatusNotFound)
	h.userCall(super, http.MethodPost, "/api/admin/users/"+itoa(intruder)+"/approve", nil, http.StatusNotFound)
	refused := 0
	for _, item := range h.accountRequests(ops) {
		if item.AccountID != nil && *item.AccountID == intruder {
			refused++
			if item.State != "resolved" || item.AccountOutcome != "refused" || item.AccountName != "" {
				t.Fatalf("the refusal resolves the request anonymously: %+v", item)
			}
		}
	}
	if refused != 1 {
		t.Fatal("the refused request stays in the history")
	}
	jar, _ = cookiejar.New(nil)
	h.userCall(&voiceUser{name: "intruder", client: &http.Client{Jar: jar}}, http.MethodPost, "/api/auth/login", map[string]string{"username": "intruder", "password": "synthetic-password"}, http.StatusUnauthorized)
	// A disabled administrator account is not a request: refusing it would retire an administrator.
	if _, err := pool.Exec(ctx, `UPDATE users SET approved=false WHERE username='ops'`); err != nil {
		t.Fatal(err)
	}
	h.userCall(super, http.MethodPost, "/api/admin/users/"+itoa(h.userID("ops"))+"/refuse", nil, http.StatusNotFound)
	if _, err := pool.Exec(ctx, `UPDATE users SET approved=true WHERE username='ops'`); err != nil {
		t.Fatal(err)
	}

	// A demoted administrator no longer sees the requests.
	if _, err := pool.Exec(ctx, `UPDATE users SET role='user' WHERE username='ops'`); err != nil {
		t.Fatal(err)
	}
	if got := h.accountRequests(ops); len(got) != 0 {
		t.Fatalf("a member never sees a request, even an old one: %+v", got)
	}
}

func (h *voiceHarness) userID(name string) int64 {
	h.t.Helper()
	var id int64
	if err := h.pool.QueryRow(context.Background(), `SELECT id FROM users WHERE username=$1`, name).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}
