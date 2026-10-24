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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

func testWorkerRegistration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	mediaRoot := t.TempDir()
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts", UploadRoot: mediaRoot}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewTLSServer(server.Handler())
	defer ts.Close()
	server.Config.PublicURL = ts.URL
	jar, _ := cookiejar.New(nil)
	admin := ts.Client()
	admin.Jar = jar
	anon := &http.Client{Transport: ts.Client().Transport}
	callWithLease := func(client *http.Client, method, path string, body []byte, csrf, token string, withOrigin bool, leaseToken string, want int) []byte {
		t.Helper()
		req, requestErr := http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		if withOrigin {
			req.Header.Set("Origin", ts.URL)
		}
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if leaseToken != "" {
			req.Header.Set("X-Worker-Lease-Token", leaseToken)
		}
		resp, requestErr := client.Do(req)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer resp.Body.Close()
		data, requestErr := io.ReadAll(resp.Body)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return data
	}
	call := func(client *http.Client, method, path string, body []byte, csrf, token string, withOrigin bool, want int) []byte {
		t.Helper()
		return callWithLease(client, method, path, body, csrf, token, withOrigin, "", want)
	}
	call(admin, http.MethodPost, "/api/setup", []byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`), "", "", true, http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/auth/me", nil, "", "", false, http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	call(admin, http.MethodPut, "/api/admin/job-controls", mustJSON(t, map[string]any{"paused": false, "max_concurrency": 2}), me.CSRF, "", true, http.StatusOK)
	call(anon, http.MethodGet, "/api/admin/workers", nil, "", "", false, http.StatusUnauthorized)
	created := call(admin, http.MethodPost, "/api/admin/workers", mustJSON(t, map[string]any{"name": "synthetic-gpu", "max_concurrency": 1}), me.CSRF, "", true, http.StatusCreated)
	var registration struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Token          string `json:"token"`
		MaxConcurrency int    `json:"max_concurrency"`
	}
	if err = json.Unmarshal(created, &registration); err != nil {
		t.Fatal(err)
	}
	if registration.ID == "" || registration.Name != "synthetic-gpu" || !strings.HasPrefix(registration.Token, "rw_") || registration.MaxConcurrency != 1 {
		t.Fatalf("unexpected worker registration: %+v", registration)
	}
	listed := call(admin, http.MethodGet, "/api/admin/workers", nil, "", "", false, http.StatusOK)
	if bytes.Contains(listed, []byte(registration.Token)) {
		t.Fatal("worker listing exposed the one-time token")
	}
	call(anon, http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1}`), "", "", false, http.StatusUnauthorized)
	pulseBody := call(anon, http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1,"supported_steps":["fixture.*"]}`), "", registration.Token, false, http.StatusOK)
	if !bytes.Contains(pulseBody, []byte(`"connected":true`)) {
		t.Fatalf("worker pulse did not report connected: %s", pulseBody)
	}
	call(anon, http.MethodGet, "/api/admin/workers", nil, "", registration.Token, false, http.StatusUnauthorized)
	call(anon, http.MethodGet, "/api/admin/jobs", nil, "", registration.Token, false, http.StatusUnauthorized)
	call(anon, http.MethodGet, "/api/clips", nil, "", registration.Token, false, http.StatusUnauthorized)
	listed = call(admin, http.MethodGet, "/api/admin/workers", nil, "", "", false, http.StatusOK)
	if !bytes.Contains(listed, []byte(`"last_seen_at":`)) {
		t.Fatalf("worker activity was not recorded: %s", listed)
	}
	setPresence := func(age string) {
		t.Helper()
		if _, updateErr := pool.Exec(ctx, `UPDATE workers SET last_seen_at=now()-$1::interval WHERE id=$2`, age, registration.ID); updateErr != nil {
			t.Fatal(updateErr)
		}
	}
	assertConnected := func(want bool) {
		t.Helper()
		var page struct {
			Items []struct {
				ID        string  `json:"id"`
				Connected bool    `json:"connected"`
				RevokedAt *string `json:"revoked_at"`
			} `json:"items"`
		}
		if decodeErr := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/workers", nil, "", "", false, http.StatusOK), &page); decodeErr != nil {
			t.Fatal(decodeErr)
		}
		found := false
		for _, item := range page.Items {
			if item.ID == registration.ID && item.Connected != want {
				t.Fatalf("worker connected=%v, want %v", item.Connected, want)
			}
			if item.ID == registration.ID {
				found = true
			}
		}
		if !found {
			t.Fatalf("worker %s missing from API response", registration.ID)
		}
	}
	assertConnected(true)
	setPresence("2 minutes")
	assertConnected(false)
	setPresence("0 seconds")
	var ownerID int64
	if err = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	var clipID int64
	if err = pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'worker-fixture.mp4',25,now(),now()) RETURNING id`, ownerID).Scan(&clipID); err != nil {
		t.Fatal(err)
	}
	const uploadID, storageKey = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	content := []byte("synthetic-worker-video-input")
	if err = os.WriteFile(filepath.Join(mediaRoot, storageKey+".media"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO uploads(id,owner_id,original_name,total_bytes,offset_bytes,visibility,status,clip_id,expires_at) VALUES($1,$2,'worker-fixture.mp4',$3,$3,'shared','complete',$4,now()+interval '1 hour')`, uploadID, ownerID, len(content), clipID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO uploaded_media(upload_id,clip_id,owner_id,storage_key,original_name) VALUES($1,$2,$3,$4,'worker-fixture.mp4')`, uploadID, clipID, ownerID, storageKey); err != nil {
		t.Fatal(err)
	}
	jobBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.echo", "input_version": "worker-v1"}), me.CSRF, "", true, http.StatusCreated)
	var queued struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(jobBody, &queued); err != nil || queued.ID < 1 {
		t.Fatalf("job was not scheduled: %s %v", jobBody, err)
	}
	secondJobBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.echo.second", "input_version": "worker-v2"}), me.CSRF, "", true, http.StatusCreated)
	var secondQueued struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(secondJobBody, &secondQueued); err != nil || secondQueued.ID < 1 || secondQueued.ID == queued.ID {
		t.Fatalf("second job was not scheduled: %s %v", secondJobBody, err)
	}
	unsupportedBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "transcription.whisper", "input_version": "future-v1", "priority": 20}), me.CSRF, "", true, http.StatusCreated)
	var unsupportedQueued struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(unsupportedBody, &unsupportedQueued); err != nil || unsupportedQueued.ID < 1 {
		t.Fatalf("future job was not scheduled: %s %v", unsupportedBody, err)
	}
	call(anon, http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1,"supported_steps":["fixture.*"]}`), "", registration.Token, false, http.StatusOK)
	claimedBody := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", registration.Token, false, http.StatusOK)
	var claimed struct {
		LeaseToken string `json:"lease_token"`
		Job        struct {
			ID           int64  `json:"id"`
			Status       string `json:"status"`
			InputVersion string `json:"input_version"`
		} `json:"job"`
	}
	if err = json.Unmarshal(claimedBody, &claimed); err != nil || claimed.Job.ID != queued.ID || claimed.Job.Status != "leased" || claimed.LeaseToken == "" {
		t.Fatalf("worker did not receive scheduled job: %s %v", claimedBody, err)
	}
	var unsupportedStatus string
	if err = pool.QueryRow(ctx, `SELECT status FROM jobs WHERE id=$1`, unsupportedQueued.ID).Scan(&unsupportedStatus); err != nil || unsupportedStatus != "pending" {
		t.Fatalf("unsupported future step status = %q, want pending (err %v)", unsupportedStatus, err)
	}
	secondClaim := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", registration.Token, false, http.StatusOK)
	if !bytes.Contains(secondClaim, []byte(`"job":null`)) {
		t.Fatalf("worker exceeded its configured concurrency: %s", secondClaim)
	}
	otherBody := call(admin, http.MethodPost, "/api/admin/workers", mustJSON(t, map[string]any{"name": "other-synthetic-gpu", "max_concurrency": 1}), me.CSRF, "", true, http.StatusCreated)
	var other struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err = json.Unmarshal(otherBody, &other); err != nil || other.Token == "" {
		t.Fatalf("second worker registration: %v %s", err, otherBody)
	}
	call(anon, http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1,"supported_steps":["fixture.*"]}`), "", other.Token, false, http.StatusOK)
	otherClaimBody := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", other.Token, false, http.StatusOK)
	var otherClaim struct {
		LeaseToken string `json:"lease_token"`
		Job        struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err = json.Unmarshal(otherClaimBody, &otherClaim); err != nil || otherClaim.Job.ID != secondQueued.ID || otherClaim.LeaseToken == "" {
		t.Fatalf("second worker did not receive the next available job: %s %v", otherClaimBody, err)
	}
	var unsupportedAttempts, unsupportedEvents int
	if err = pool.QueryRow(ctx, `SELECT attempts,(SELECT count(*) FROM job_events WHERE job_id=jobs.id) FROM jobs WHERE id=$1 AND status='pending'`, unsupportedQueued.ID).Scan(&unsupportedAttempts, &unsupportedEvents); err != nil || unsupportedAttempts != 0 || unsupportedEvents != 0 {
		t.Fatalf("unsupported future step was executed (attempts=%d events=%d, err %v)", unsupportedAttempts, unsupportedEvents, err)
	}
	callWithLease(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(queued.ID)+"/input", nil, "", other.Token, false, otherClaim.LeaseToken, http.StatusNotFound)
	callWithLease(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(queued.ID)+"/media", nil, "", other.Token, false, otherClaim.LeaseToken, http.StatusNotFound)
	inputBody := callWithLease(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(queued.ID)+"/input", nil, "", registration.Token, false, claimed.LeaseToken, http.StatusOK)
	if bytes.Contains(inputBody, []byte(mediaRoot)) || bytes.Contains(inputBody, []byte(storageKey)) || !bytes.Contains(inputBody, []byte(`"media_url":"/api/worker/jobs/`)) {
		t.Fatalf("worker input exposed a storage path or omitted scoped media URL: %s", inputBody)
	}
	media := callWithLease(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(queued.ID)+"/media", nil, "", registration.Token, false, claimed.LeaseToken, http.StatusOK)
	if !bytes.Equal(media, content) {
		t.Fatalf("worker received wrong media bytes: %q", media)
	}
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(queued.ID)+"/heartbeat", []byte(`{}`), "", registration.Token, false, claimed.LeaseToken, http.StatusOK)
	completed := callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(queued.ID)+"/complete", []byte(`{"input_version":"worker-v1","result":{"provider":"fixture","sha256":"synthetic"}}`), "", registration.Token, false, claimed.LeaseToken, http.StatusOK)
	if !bytes.Contains(completed, []byte(`"status":"done"`)) || !bytes.Contains(completed, []byte(`"provider":"fixture"`)) {
		t.Fatalf("worker result was not recorded: %s", completed)
	}
	completedAgain := callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(queued.ID)+"/complete", []byte(`{"input_version":"worker-v1","result":{"provider":"fixture","sha256":"synthetic"}}`), "", registration.Token, false, claimed.LeaseToken, http.StatusOK)
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(queued.ID)+"/complete", []byte(`{"input_version":"worker-v1","result":{"provider":"fixture","sha256":"different"}}`), "", registration.Token, false, claimed.LeaseToken, http.StatusConflict)
	if !bytes.Contains(completedAgain, []byte(`"status":"done"`)) || !bytes.Contains(completedAgain, []byte(`"provider":"fixture"`)) {
		t.Fatalf("repeated completion was not idempotent: %s", completedAgain)
	}
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(secondQueued.ID)+"/fail", []byte(`{"error":"invalid_input"}`), "", other.Token, false, otherClaim.LeaseToken, http.StatusOK)
	concurrentJobBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.concurrent", "input_version": "worker-v1"}), me.CSRF, "", true, http.StatusCreated)
	var concurrentJob struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(concurrentJobBody, &concurrentJob); err != nil || concurrentJob.ID < 1 {
		t.Fatalf("concurrent claim job was not scheduled: %s %v", concurrentJobBody, err)
	}
	type claimResponse struct {
		body        []byte
		workerToken string
		err         error
	}
	claimResults := make(chan claimResponse, 2)
	for _, workerToken := range []string{registration.Token, other.Token} {
		go func(workerToken string) {
			req, requestErr := http.NewRequest(http.MethodPost, ts.URL+"/api/worker/jobs/claim", nil)
			if requestErr != nil {
				claimResults <- claimResponse{workerToken: workerToken, err: requestErr}
				return
			}
			req.Header.Set("Authorization", "Bearer "+workerToken)
			resp, requestErr := anon.Do(req)
			if requestErr != nil {
				claimResults <- claimResponse{workerToken: workerToken, err: requestErr}
				return
			}
			defer resp.Body.Close()
			body, readErr := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK && readErr == nil {
				readErr = fmt.Errorf("claim returned HTTP %d: %s", resp.StatusCode, body)
			}
			claimResults <- claimResponse{body: body, workerToken: workerToken, err: readErr}
		}(workerToken)
	}
	var concurrentClaims []struct {
		workerToken string
		LeaseToken  string `json:"lease_token"`
		Job         *struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	for range 2 {
		response := <-claimResults
		if response.err != nil {
			t.Fatal(response.err)
		}
		var result struct {
			LeaseToken string `json:"lease_token"`
			Job        *struct {
				ID int64 `json:"id"`
			} `json:"job"`
		}
		if err = json.Unmarshal(response.body, &result); err != nil {
			t.Fatalf("decode concurrent claim: %s %v", response.body, err)
		}
		concurrentClaims = append(concurrentClaims, struct {
			workerToken string
			LeaseToken  string `json:"lease_token"`
			Job         *struct {
				ID int64 `json:"id"`
			} `json:"job"`
		}{workerToken: response.workerToken, LeaseToken: result.LeaseToken, Job: result.Job})
	}
	claimedCount := 0
	for _, result := range concurrentClaims {
		if result.Job != nil {
			if result.Job.ID != concurrentJob.ID || result.LeaseToken == "" {
				t.Fatalf("unexpected concurrent claim: %+v", result)
			}
			claimedCount++
			callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(concurrentJob.ID)+"/complete", []byte(`{"input_version":"worker-v1","result":{"fixture":"concurrent"}}`), "", result.workerToken, false, result.LeaseToken, http.StatusOK)
		}
	}
	if claimedCount != 1 {
		t.Fatalf("one pending task must have exactly one concurrent claimant, got %d: %+v", claimedCount, concurrentClaims)
	}
	expiringJobBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.expire", "input_version": "worker-v1"}), me.CSRF, "", true, http.StatusCreated)
	var expiringJob struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(expiringJobBody, &expiringJob); err != nil || expiringJob.ID < 1 {
		t.Fatalf("expiring job was not scheduled: %s %v", expiringJobBody, err)
	}
	expiringClaimBody := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", registration.Token, false, http.StatusOK)
	var expiringClaim struct {
		LeaseToken string `json:"lease_token"`
		Job        struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err = json.Unmarshal(expiringClaimBody, &expiringClaim); err != nil || expiringClaim.Job.ID != expiringJob.ID || expiringClaim.LeaseToken == "" {
		t.Fatalf("worker did not acquire expiring job: %s %v", expiringClaimBody, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE jobs SET lease_expires_at=now()-interval '1 second' WHERE id=$1`, expiringJob.ID); err != nil {
		t.Fatal(err)
	}
	callWithLease(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/input", nil, "", registration.Token, false, expiringClaim.LeaseToken, http.StatusNotFound)
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/heartbeat", []byte(`{}`), "", registration.Token, false, expiringClaim.LeaseToken, http.StatusConflict)
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/complete", []byte(`{"input_version":"worker-v1","result":{"fixture":true}}`), "", registration.Token, false, expiringClaim.LeaseToken, http.StatusConflict)
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/fail", []byte(`{"error":"provider_unavailable"}`), "", registration.Token, false, expiringClaim.LeaseToken, http.StatusConflict)
	reclaimedBody := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", registration.Token, false, http.StatusOK)
	var reclaimed struct {
		LeaseToken string `json:"lease_token"`
		Job        struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"job"`
	}
	if err = json.Unmarshal(reclaimedBody, &reclaimed); err != nil || reclaimed.Job.ID != expiringJob.ID || reclaimed.Job.Status != "leased" || reclaimed.LeaseToken == "" || reclaimed.LeaseToken == expiringClaim.LeaseToken {
		t.Fatalf("another worker did not reclaim expired lease: %s %v", reclaimedBody, err)
	}
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/complete", []byte(`{"input_version":"worker-v1","result":{"fixture":"stale-owner"}}`), "", registration.Token, false, expiringClaim.LeaseToken, http.StatusConflict)
	callWithLease(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/input", nil, "", registration.Token, false, reclaimed.LeaseToken, http.StatusOK)
	currentResult := []byte(`{"input_version":"worker-v1","result":{"fixture":true}}`)
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/complete", currentResult, "", registration.Token, false, reclaimed.LeaseToken, http.StatusOK)
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/complete", currentResult, "", registration.Token, false, reclaimed.LeaseToken, http.StatusOK)
	versionV1Body := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.versioned", "input_version": "v1"}), me.CSRF, "", true, http.StatusCreated)
	var versionV1 struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(versionV1Body, &versionV1); err != nil || versionV1.ID < 1 {
		t.Fatalf("version 1 job was not scheduled: %s %v", versionV1Body, err)
	}
	call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.sibling", "input_version": "v1"}), me.CSRF, "", true, http.StatusCreated)
	versionV1ClaimBody := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", registration.Token, false, http.StatusOK)
	var versionV1Claim struct {
		LeaseToken string `json:"lease_token"`
		Job        struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err = json.Unmarshal(versionV1ClaimBody, &versionV1Claim); err != nil || versionV1Claim.Job.ID != versionV1.ID || versionV1Claim.LeaseToken == "" {
		t.Fatalf("worker did not acquire version 1 job: %s %v", versionV1ClaimBody, err)
	}
	versionV2Body := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.versioned", "input_version": "v2", "priority": 10}), me.CSRF, "", true, http.StatusCreated)
	var versionV2 struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(versionV2Body, &versionV2); err != nil || versionV2.ID < 1 || versionV2.ID == versionV1.ID {
		t.Fatalf("version 2 job was not scheduled independently: %s %v", versionV2Body, err)
	}
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(versionV1.ID)+"/complete", []byte(`{"input_version":"v1","result":{"fixture":"stale"}}`), "", registration.Token, false, versionV1Claim.LeaseToken, http.StatusConflict)
	call(admin, http.MethodGet, "/api/admin/jobs?status=failed&step=fixture.versioned", nil, "", "", false, http.StatusOK)
	siblingPage := call(admin, http.MethodGet, "/api/admin/jobs?status=pending&step=fixture.sibling", nil, "", "", false, http.StatusOK)
	if !bytes.Contains(siblingPage, []byte(`"total":1`)) {
		t.Fatalf("scheduling a new version invalidated a different step: %s", siblingPage)
	}
	versionV2ClaimBody := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", other.Token, false, http.StatusOK)
	var versionV2Claim struct {
		LeaseToken string `json:"lease_token"`
		Job        struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err = json.Unmarshal(versionV2ClaimBody, &versionV2Claim); err != nil || versionV2Claim.Job.ID != versionV2.ID || versionV2Claim.LeaseToken == "" {
		t.Fatalf("worker did not claim current version: %s %v", versionV2ClaimBody, err)
	}
	callWithLease(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(versionV2.ID)+"/complete", []byte(`{"input_version":"v2","result":{"fixture":"current"}}`), "", other.Token, false, versionV2Claim.LeaseToken, http.StatusOK)
	call(admin, http.MethodDelete, "/api/admin/workers/"+registration.ID, nil, me.CSRF, "", true, http.StatusNoContent)
	assertConnected(false)
	call(anon, http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1}`), "", registration.Token, false, http.StatusUnauthorized)
	call(admin, http.MethodDelete, "/api/admin/workers/"+other.ID, nil, me.CSRF, "", true, http.StatusNoContent)
}
