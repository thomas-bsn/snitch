//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testWorkerRevocation covers #116: immediate and definitive revocation, tasks back in the queue,
// late results refused, the narrow confirmation for the launcher and the minimal audit.
func testWorkerRevocation(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newInstallHarness(t, pool, true)
	id, token, _, _ := h.register("")
	_, otherToken, _, _ := h.register("")
	pulse := []byte(`{"capacity":1,"supported_steps":["fixture.*"]}`)
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/pulse", pulse, "", token, false, http.StatusOK)
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/pulse", pulse, "", otherToken, false, http.StatusOK)
	h.events(token, http.StatusNoContent, ev("image", "started", "", time.Now()))

	var ownerID, clipID, jobID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'revocation-fixture.mp4',7,now(),now()) RETURNING id`, ownerID).Scan(&clipID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,max_attempts) VALUES($1,'fixture.revoke','v1',3) RETURNING id`, clipID).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	var claim struct {
		LeaseToken string `json:"lease_token"`
	}
	if err := json.Unmarshal(h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/jobs/claim", nil, "", token, false, http.StatusOK), &claim); err != nil || claim.LeaseToken == "" {
		t.Fatalf("claim: %v", err)
	}
	jobPath := h.ts.URL + "/api/worker/jobs/" + strconv64(jobID)

	// Before revocation the confirmation endpoint says "not revoked" and only that.
	if body := string(h.get(h.ts.URL+"/api/worker/revocation", token, http.StatusOK)); !strings.Contains(body, `"revoked":false`) {
		t.Fatalf("active worker: %s", body)
	}
	h.get(h.ts.URL+"/api/worker/revocation", "", http.StatusUnauthorized)
	h.get(h.ts.URL+"/api/worker/revocation", "rw_unknown-synthetic-token", http.StatusUnauthorized)

	// Revocation with a reason: validated, then immediate.
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+id, []byte(`{"reason":"`+strings.Repeat("x", 201)+`"}`), http.StatusBadRequest)
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+id, []byte(`{"reason":"machine retired"}`), http.StatusNoContent)

	// Running task back in the queue at once, without waiting for the lease to expire.
	var status string
	var owner *string
	var attempts int
	if err := pool.QueryRow(ctx, `SELECT status,lease_owner,attempts FROM jobs WHERE id=$1`, jobID).Scan(&status, &owner, &attempts); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || owner != nil || attempts != 0 {
		t.Fatalf("task not requeued: %s %v %d", status, owner, attempts)
	}

	// Every normal call and new claim refused; late results refused.
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/pulse", pulse, "", token, false, http.StatusUnauthorized)
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/jobs/claim", nil, "", token, false, http.StatusUnauthorized)
	h.get(h.ts.URL+"/api/worker/transcription-config", token, http.StatusUnauthorized)
	h.events(token, http.StatusUnauthorized, ev("image", "started", "", time.Now()))
	late := func(path, body string) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, jobPath+path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Worker-Lease-Token", claim.LeaseToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := h.anon.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("late %s: %d", path, resp.StatusCode)
		}
	}
	late("/complete", `{"input_version":"v1","result":{}}`)
	late("/fail", `{"error":"late"}`)
	late("/heartbeat", `{}`)
	if err := pool.QueryRow(ctx, `SELECT status FROM jobs WHERE id=$1`, jobID).Scan(&status); err != nil || status != "pending" {
		t.Fatalf("late result changed the task: %s %v", status, err)
	}
	// The requeued task goes to the other Worker, never back to the revoked one.
	var got struct {
		Job *struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err := json.Unmarshal(h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/jobs/claim", nil, "", otherToken, false, http.StatusOK), &got); err != nil || got.Job == nil || got.Job.ID != jobID {
		t.Fatalf("other worker did not receive the task: %+v %v", got, err)
	}

	// Definitive: no new token, a second deletion keeps the first audit.
	h.adminCall(http.MethodPost, "/api/admin/workers/"+id+"/token", nil, http.StatusNotFound)
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+id, []byte(`{"reason":"other"}`), http.StatusNoContent)
	h.adminCall(http.MethodDelete, "/api/admin/workers/wk_unknown_synthetic_worker_id", nil, http.StatusNotFound)

	// Narrow confirmation: revoked flag and date only, no configuration, no action; it acknowledges.
	body := h.get(h.ts.URL+"/api/worker/revocation", token, http.StatusOK)
	var conf map[string]any
	if err := json.Unmarshal(body, &conf); err != nil || conf["revoked"] != true || len(conf) != 2 || conf["revoked_at"] == nil {
		t.Fatalf("confirmation: %s", body)
	}
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/revocation", nil, "", token, false, http.StatusNotFound)

	// Minimal audit: name, date, optional reason; detailed data gone; acknowledgement visible.
	var detail struct {
		Name                     string     `json:"name"`
		RevokedAt                *string    `json:"revoked_at"`
		RevokeReason             string     `json:"revoke_reason"`
		RevocationAcknowledgedAt *time.Time `json:"revocation_acknowledged_at"`
		Install                  *struct{}  `json:"install"`
		RemoteAddress            string     `json:"remote_address"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/workers/"+id, nil, http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Name != "synthetic-linux" || detail.RevokedAt == nil || detail.RevokeReason != "machine retired" || detail.RevocationAcknowledgedAt == nil || detail.Install != nil || detail.RemoteAddress != "" {
		t.Fatalf("audit: %+v", detail)
	}
	var events, invites, cfg int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM worker_install_events WHERE worker_id=$1),(SELECT count(*) FROM worker_install_invites WHERE worker_id=$1),(SELECT count(*) FROM worker_transcription_config WHERE worker_id=$1)`, id).Scan(&events, &invites, &cfg); err != nil || events+invites+cfg != 0 {
		t.Fatalf("details kept: %d %d %d %v", events, invites, cfg, err)
	}
	// Offline Worker: never acknowledged, still revoked server-side, cleanup pending.
	offline, _, _, _ := h.register("")
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+offline, nil, http.StatusNoContent)
	var off struct {
		RevokedAt *string `json:"revoked_at"`
		Reason    string  `json:"revoke_reason"`
		Ack       *string `json:"revocation_acknowledged_at"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/workers/"+offline, nil, http.StatusOK), &off); err != nil || off.RevokedAt == nil || off.Ack != nil || off.Reason != "" {
		t.Fatalf("offline: %+v %v", off, err)
	}

	// #261: without confirmation the entry stays by default; only an explicit force removes it,
	// and an active Worker cannot be removed even then. History (jobs, events) is untouched.
	record := "/api/admin/workers/" + offline + "/record"
	h.adminCall(http.MethodDelete, record, nil, http.StatusConflict)
	h.adminCall(http.MethodDelete, record+"?force=false", nil, http.StatusConflict)
	h.adminCall(http.MethodDelete, record+"?force=maybe", nil, http.StatusBadRequest)
	active, _, _, _ := h.register("")
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+active+"/record?force=true", nil, http.StatusConflict)
	var jobsBefore, eventsBefore int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM jobs),(SELECT count(*) FROM job_events)`).Scan(&jobsBefore, &eventsBefore); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodDelete, record+"?force=true", nil, http.StatusNoContent)
	h.adminCall(http.MethodGet, "/api/admin/workers/"+offline, nil, http.StatusNotFound)
	var jobsAfter, eventsAfter int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM jobs),(SELECT count(*) FROM job_events)`).Scan(&jobsAfter, &eventsAfter); err != nil || jobsAfter != jobsBefore || eventsAfter != eventsBefore {
		t.Fatalf("history changed: jobs %d->%d events %d->%d %v", jobsBefore, jobsAfter, eventsBefore, eventsAfter, err)
	}
}
