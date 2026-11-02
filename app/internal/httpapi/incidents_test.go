//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type incidentT struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	State     string `json:"state"`
	Count     int    `json:"pending_count"`
	JobID     *int64 `json:"job_id"`
	Step      string `json:"step"`
	ErrorCode string `json:"error_code"`
	Generic   bool   `json:"generic"`
}

func (h *voiceHarness) incidents(u *voiceUser) ([]incidentT, string) {
	h.t.Helper()
	raw := h.userCall(u, http.MethodGet, "/api/notifications", nil, http.StatusOK)
	var list struct {
		Items []incidentT `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		h.t.Fatal(err)
	}
	return list.Items, string(raw)
}

func kinds(items []incidentT, state string) map[string]int {
	out := map[string]int{}
	for _, i := range items {
		if i.State == state {
			out[i.Kind]++
		}
	}
	return out
}

func testIncidentNotifications(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 1)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`DELETE FROM notifications`)
	exec(`DELETE FROM jobs`)
	clip := h.clipIDs[0]
	admin := h.adminUser()
	alice := h.newUser("alice")
	job := func(step, status, lastError string, ageMinutes int) {
		t.Helper()
		exec(`INSERT INTO jobs(clip_id,step,input_version,status,last_error,created_at) VALUES($1,$2,'v-'||gen_random_uuid()::text,$3,$4,now()-($5 * interval '1 minute'))`, clip, step, status, lastError, ageMinutes)
	}
	exec(`UPDATE workers SET last_seen_at=now()-interval '1 hour'`)

	// Jobs wait with no worker: one grouped incident, however many jobs and syncs.
	job("diarize.test", "pending", "", 10)
	job("diarize.test", "pending", "", 10)
	h.api.syncIncidents(ctx)
	h.api.syncIncidents(ctx)
	items, _ := h.incidents(admin)
	if k := kinds(items, "active"); k["worker_unavailable"] != 1 || len(items) != 1 || items[0].Count != 2 {
		t.Fatalf("one grouped worker incident: %+v", items)
	}
	// A recent job is not an incident yet.
	exec(`DELETE FROM jobs`)
	exec(`DELETE FROM notifications`)
	job("diarize.test", "pending", "", 0)
	h.api.syncIncidents(ctx)
	if items, _ = h.incidents(admin); len(items) != 0 {
		t.Fatalf("grace period: %+v", items)
	}
	job("diarize.test", "pending", "", 10)
	job("diarize.test", "retry_wait", "provider_unavailable", 10)
	job("diarize.test", "dead", "provider_unavailable", 10)
	job("other.test", "dead", "worker_failed", 10)
	job("third.test", "dead", "worker_failed", 10)
	h.api.syncIncidents(ctx)
	items, _ = h.incidents(admin)
	// Retries refresh the same notification; each cause stays distinct.
	if k := kinds(items, "active"); k["worker_unavailable"] != 1 || k["provider_unavailable"] != 1 || k["job_failed"] != 2 || len(items) != 4 {
		t.Fatalf("one notification per cause: %+v", items)
	}
	for _, i := range items {
		if i.Kind == "provider_unavailable" && (i.Count != 2 || i.ErrorCode != "provider_unavailable") {
			t.Fatalf("provider incident: %+v", i)
		}
		if i.Kind == "job_failed" && (i.Step == "" || i.JobID == nil) {
			t.Fatalf("job incident needs its step and job: %+v", i)
		}
	}

	// The clip owner only gets the generic, safe message; others get nothing.
	if seen, _ := h.incidents(alice); len(seen) != 0 {
		t.Fatalf("not a recipient: %+v", seen)
	}
	exec(`UPDATE clips SET owner_id=(SELECT id FROM users WHERE username='alice') WHERE id=$1`, clip)
	h.api.syncIncidents(ctx)
	seen, raw := h.incidents(alice)
	if len(seen) != 4 {
		t.Fatalf("owner sees generic incidents: %+v", seen)
	}
	for _, i := range seen {
		if !i.Generic || i.Kind != "technical_incident" || i.Step != "" || i.ErrorCode != "" || i.JobID != nil || i.Count != 0 {
			t.Fatalf("generic only: %+v", i)
		}
	}
	for _, leak := range []string{"provider_unavailable", "worker_failed", "diarize.test", "worker_unavailable"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("non-admin payload leaks %q: %s", leak, raw)
		}
	}
	// Access is rechecked on every read: once the clip is no longer theirs, nothing remains.
	exec(`UPDATE clips SET owner_id=(SELECT id FROM users WHERE role IN ('admin','super_admin') ORDER BY id LIMIT 1) WHERE id=$1`, clip)
	if seen, _ = h.incidents(alice); len(seen) != 0 {
		t.Fatalf("access lost: %+v", seen)
	}

	// The worker comes back: only its incident resolves.
	exec(`UPDATE workers SET last_seen_at=now(),supported_steps='{diarize.*}'`)
	h.api.syncIncidents(ctx)
	items, _ = h.incidents(admin)
	if a, r := kinds(items, "active"), kinds(items, "resolved"); r["worker_unavailable"] != 1 || a["provider_unavailable"] != 1 || a["job_failed"] != 2 {
		t.Fatalf("worker return resolves its incident only: %+v", items)
	}
	// A new failure on a new step is a new notification.
	job("fourth.test", "dead", "worker_failed", 10)
	h.api.syncIncidents(ctx)
	items, _ = h.incidents(admin)
	if k := kinds(items, "active"); k["job_failed"] != 3 {
		t.Fatalf("new cause stays distinct: %+v", items)
	}
	// Once the jobs succeed or are cancelled, their incidents resolve by themselves.
	exec(`UPDATE jobs SET status='done',last_error='' WHERE last_error='provider_unavailable'`)
	exec(`UPDATE jobs SET status='cancelled' WHERE status='dead'`)
	h.api.syncIncidents(ctx)
	items, _ = h.incidents(admin)
	if k := kinds(items, "active"); len(k) != 0 {
		t.Fatalf("everything resolved: %+v", items)
	}
}
