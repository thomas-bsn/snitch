//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// roundTripFunc is a transport written inline.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// fakeWebhook answers the webhook calls with the scripted statuses (the last one repeats) and records the bodies.
type fakeWebhook struct {
	mu       sync.Mutex
	statuses []int
	down     bool
	bodies   []map[string]any
	paths    []string
	server   *httptest.Server
}

func (f *fakeWebhook) script(down bool, statuses ...int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.down, f.statuses = down, statuses
}

func (f *fakeWebhook) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.bodies)
}

func (f *fakeWebhook) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	down := f.down
	f.mu.Unlock()
	if down {
		return nil, errors.New("dial tcp: connection refused to " + r.URL.String())
	}
	target, _ := url.Parse(f.server.URL)
	out := r.Clone(r.Context())
	out.URL.Scheme, out.URL.Host, out.Host = target.Scheme, target.Host, ""
	return http.DefaultTransport.RoundTrip(out)
}

func newFakeWebhook(t *testing.T) *fakeWebhook {
	f := &fakeWebhook{statuses: []int{204}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.bodies, f.paths = append(f.bodies, body), append(f.paths, r.URL.Path)
		status := f.statuses[0]
		if len(f.statuses) > 1 {
			f.statuses = f.statuses[1:]
		}
		f.mu.Unlock()
		if status == http.StatusTooManyRequests {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"retry_after":2.5,"global":false}`))
			return
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(f.server.Close)
	return f
}

// testNotificationWebhook covers tranche B of #283: the Super-administrator configures a Discord webhook; allowed
// notifications are posted as redacted summaries from a durable queue, never inside the workflow that raised them;
// Discord failures are retried a bounded number of times, shown to the administration, and never leak the secret.
func testNotificationWebhook(t *testing.T, pool *pgxpool.Pool) {
	logs := &lockedBuffer{}
	h := newNotificationHarness(t, pool, logs)
	discord := newFakeWebhook(t)
	h.api.webhookTransport = discord
	h.api.webhookRetryBase = time.Millisecond
	ctx := context.Background()
	const secret = "SyntheticWebhookToken_0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJ"
	address := "https://discord.com/api/webhooks/111122223333444455/" + secret
	super := h.adminUser()
	ops := h.newUser("ops")
	member := h.newUser("member")
	if _, err := pool.Exec(ctx, `UPDATE users SET role='admin' WHERE username='ops'`); err != nil {
		t.Fatal(err)
	}
	view := func(data []byte) webhookView {
		t.Helper()
		if strings.Contains(string(data), secret) {
			t.Fatal("the webhook address is never sent back")
		}
		var v webhookView
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	current := func() webhookView {
		t.Helper()
		return view(h.userCall(ops, http.MethodGet, "/api/admin/notification-webhook", nil, http.StatusOK))
	}
	deliver := func() {
		t.Helper()
		// Every pending delivery is due now: the retry waits are the subject of other assertions.
		if _, err := pool.Exec(ctx, `UPDATE notification_deliveries SET next_attempt_at=now() WHERE status='pending'`); err != nil {
			t.Fatal(err)
		}
		if _, err := h.api.deliverNotificationBatch(ctx); err != nil {
			t.Fatal(err)
		}
	}
	deliveries := func() (status string, attempts int, code string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT status,attempts,last_error FROM notification_deliveries ORDER BY id DESC LIMIT 1`).Scan(&status, &attempts, &code); err != nil {
			t.Fatal(err)
		}
		return
	}

	// Nothing is configured: reading is open to administrators, changing is the Super-administrator's.
	if v := current(); v.Configured || v.Enabled || v.LastStatus != "never" {
		t.Fatalf("no webhook by default: %+v", v)
	}
	settings := map[string]any{"url": address, "enabled": true, "categories": []string{"access_requests", "worker_offline"}}
	h.userCall(member, http.MethodGet, "/api/admin/notification-webhook", nil, http.StatusForbidden)
	h.userCall(member, http.MethodPut, "/api/admin/notification-webhook", settings, http.StatusForbidden)
	h.userCall(ops, http.MethodPut, "/api/admin/notification-webhook", settings, http.StatusForbidden)
	h.userCall(ops, http.MethodPost, "/api/admin/notification-webhook/test", nil, http.StatusForbidden)
	h.userCall(super, http.MethodPost, "/api/admin/notification-webhook/test", nil, http.StatusConflict)
	h.userCall(super, http.MethodPut, "/api/admin/notification-webhook", map[string]any{"url": "https://127.0.0.1/api/webhooks/111122223333444455/" + secret, "enabled": true, "categories": []string{}}, http.StatusBadRequest)
	h.userCall(super, http.MethodPut, "/api/admin/notification-webhook", map[string]any{"enabled": true, "categories": []string{}}, http.StatusBadRequest)
	h.userCall(super, http.MethodPut, "/api/admin/notification-webhook", map[string]any{"url": address, "enabled": true, "categories": []string{"transcripts"}}, http.StatusBadRequest)
	// Without the CSRF token, nothing changes.
	h.userCall(&voiceUser{name: "administrator", client: h.admin}, http.MethodPut, "/api/admin/notification-webhook", settings, http.StatusForbidden)
	if discord.calls() != 0 {
		t.Fatal("a refused setting never reaches Discord")
	}

	v := view(h.userCall(super, http.MethodPut, "/api/admin/notification-webhook", settings, http.StatusOK))
	if !v.Configured || !v.Enabled || v.Hint != "4455" || len(v.Categories) != 2 || len(v.Audit) != 1 || v.Audit[0].Action != "created" || v.Audit[0].ActorUsername != "administrator" {
		t.Fatalf("the webhook is configured: %+v", v)
	}
	var sealed []byte
	if err := pool.QueryRow(ctx, `SELECT value FROM settings WHERE key=$1`, webhookURLKey).Scan(&sealed); err != nil || strings.Contains(string(sealed), secret) {
		t.Fatalf("the address is stored encrypted: %v", err)
	}

	// A sign-up queues its delivery without calling Discord, even when Discord is unreachable.
	discord.script(true)
	h.requestAccess("visitor")
	if discord.calls() != 0 {
		t.Fatal("the sign-up never calls Discord itself")
	}
	if status, attempts, _ := deliveries(); status != "pending" || attempts != 0 {
		t.Fatalf("the delivery is queued: %s %d", status, attempts)
	}

	// Unreachable, then a server error, then rate limited: retried, visible in the administration.
	deliver()
	if status, attempts, code := deliveries(); status != "pending" || attempts != 1 || code != "network" {
		t.Fatalf("an unreachable Discord is retried: %s %d %s", status, attempts, code)
	}
	discord.script(false, 500)
	deliver()
	if v := current(); v.LastStatus != "failed" || v.LastError != "http_500" || v.Pending != 1 {
		t.Fatalf("the failure is shown: %+v", v)
	}
	discord.script(false, 429)
	deliver()
	var wait int64
	if err := pool.QueryRow(ctx, `SELECT (EXTRACT(EPOCH FROM next_attempt_at-now())*1000)::bigint FROM notification_deliveries ORDER BY id DESC LIMIT 1`).Scan(&wait); err != nil {
		t.Fatal(err)
	}
	if v := current(); v.LastStatus != "rate_limited" || v.Pending != 1 || wait < 2000 {
		t.Fatalf("a rate limit waits as long as Discord asks: %+v %d ms", v, wait)
	}

	// Delivered: a redacted summary, mentions disabled, to the configured webhook.
	discord.script(false, 204)
	deliver()
	if status, _, _ := deliveries(); status != "sent" {
		t.Fatalf("the delivery is sent: %s", status)
	}
	body := discord.bodies[len(discord.bodies)-1]
	content, _ := body["content"].(string)
	mentions, _ := body["allowed_mentions"].(map[string]any)
	if !strings.Contains(content, "Nouvelle demande d'accès") || strings.Contains(content, "visitor") || !strings.Contains(content, h.ts.URL+"/admin/users/signups") || mentions == nil || len(mentions["parse"].([]any)) != 0 {
		t.Fatalf("a redacted summary is posted: %v", body)
	}
	if discord.paths[len(discord.paths)-1] != "/api/webhooks/111122223333444455/"+secret {
		t.Fatalf("posted to the configured webhook: %s", discord.paths[len(discord.paths)-1])
	}
	if v := current(); v.LastStatus != "sent" || v.LastSuccessAt == nil || v.Pending != 0 {
		t.Fatalf("the success is shown: %+v", v)
	}
	calls := discord.calls()
	deliver()
	if discord.calls() != calls {
		t.Fatal("a sent delivery is never posted again")
	}

	// Only the allowed categories are relayed; a folder decision never is.
	var before int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_deliveries`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO notifications(kind,cause,pending_count) VALUES('job_failed','synthetic-failure',1);
INSERT INTO notifications(kind,state,resolved_at,folder_path,folder_outcome) VALUES('folder_request_decided','resolved',now(),'private/folder','approved');
INSERT INTO notifications(kind,cause,pending_count) VALUES('worker_unavailable','synthetic-worker',2)`); err != nil {
		t.Fatal(err)
	}
	var queued []string
	rows, err := pool.Query(ctx, `SELECT category FROM notification_deliveries ORDER BY id OFFSET $1`, before)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c string
		_ = rows.Scan(&c)
		queued = append(queued, c)
	}
	rows.Close()
	if len(queued) != 1 || queued[0] != "worker_offline" {
		t.Fatalf("only the allowed categories are queued: %v", queued)
	}

	// A Discord that keeps failing: bounded retries, then failed.
	discord.script(false, 503)
	for range webhookMaxAttempts + 2 {
		deliver()
	}
	if status, attempts, code := deliveries(); status != "failed" || attempts != webhookMaxAttempts || code != "http_503" {
		t.Fatalf("retries are bounded: %s %d %s", status, attempts, code)
	}
	if v := current(); v.Failed != 1 || v.Deliveries[0].Status != "failed" {
		t.Fatalf("the failed delivery is shown: %+v", v)
	}

	// Explicit test message: recorded as the last delivery; a deleted webhook is reported.
	discord.script(false, 204)
	if v := view(h.userCall(super, http.MethodPost, "/api/admin/notification-webhook/test", nil, http.StatusOK)); v.LastStatus != "sent" {
		t.Fatalf("the test message is sent: %+v", v)
	}
	discord.script(false, 404)
	h.userCall(super, http.MethodPost, "/api/admin/notification-webhook/test", nil, http.StatusBadRequest)

	// A webhook deleted on Discord's side disables itself instead of failing forever.
	h.requestAccess("second-visitor")
	deliver()
	if v := current(); v.Enabled || v.Audit[0].Action != "auto_disabled" || v.Audit[0].ActorUsername != "" {
		t.Fatalf("a deleted webhook disables itself: %+v", v)
	}
	h.requestAccess("third-visitor")
	if status, _, _ := deliveries(); status != "failed" {
		t.Fatalf("nothing is queued while disabled: %s", status)
	}

	// Rotation keeps the settings and replaces the address; disabling a category drops its pending deliveries.
	rotated := "https://discord.com/api/webhooks/999988887777666655/" + strings.ToLower(secret)
	v = view(h.userCall(super, http.MethodPut, "/api/admin/notification-webhook", map[string]any{"url": rotated, "enabled": true, "categories": []string{"access_requests"}}, http.StatusOK))
	if v.Hint != "6655" || v.Audit[0].Action != "rotated" || v.LastStatus != "never" {
		t.Fatalf("the address is rotated: %+v", v)
	}
	h.requestAccess("fourth-visitor")
	v = view(h.userCall(super, http.MethodPut, "/api/admin/notification-webhook", map[string]any{"enabled": true, "categories": []string{"worker_offline"}}, http.StatusOK))
	if v.Hint != "6655" || v.Pending != 0 || v.Audit[0].Action != "updated" {
		t.Fatalf("an update keeps the address and drops what is no longer allowed: %+v", v)
	}

	// Removal forgets the address.
	h.userCall(super, http.MethodDelete, "/api/admin/notification-webhook", nil, http.StatusNoContent)
	if v := current(); v.Configured || v.Enabled || v.Audit[0].Action != "deleted" {
		t.Fatalf("the webhook is removed: %+v", v)
	}
	if got, _ := h.api.Store.Get(ctx, webhookURLKey); got != "" {
		t.Fatal("the address is forgotten")
	}

	// Never a secret in the logs, the audit or the live events.
	if strings.Contains(logs.String(), secret) || strings.Contains(strings.ToLower(logs.String()), strings.ToLower(secret)) {
		t.Fatal("the webhook address never reaches the logs")
	}
	var leaked bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM notification_webhook_events e WHERE e::text ILIKE '%'||$1||'%') OR EXISTS (SELECT 1 FROM events e WHERE e::text ILIKE '%'||$1||'%') OR EXISTS (SELECT 1 FROM notification_webhook w WHERE w::text ILIKE '%'||$1||'%')`, secret).Scan(&leaked); err != nil || leaked {
		t.Fatalf("the address never reaches the audit or the events: %v", err)
	}
}
