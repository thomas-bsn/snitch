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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"receipts/internal/config"
	"receipts/internal/setup"
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
	call := func(client *http.Client, method, path string, body []byte, csrf, token string, withOrigin bool, want int) []byte {
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
	call(admin, http.MethodPost, "/api/setup", []byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`), "", "", true, http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/auth/me", nil, "", "", false, http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
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
	call(anon, http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1}`), "", registration.Token, false, http.StatusOK)
	call(anon, http.MethodGet, "/api/admin/workers", nil, "", registration.Token, false, http.StatusUnauthorized)
	call(anon, http.MethodGet, "/api/admin/jobs", nil, "", registration.Token, false, http.StatusUnauthorized)
	call(anon, http.MethodGet, "/api/clips", nil, "", registration.Token, false, http.StatusUnauthorized)
	listed = call(admin, http.MethodGet, "/api/admin/workers", nil, "", "", false, http.StatusOK)
	if !bytes.Contains(listed, []byte(`"last_seen_at":`)) {
		t.Fatalf("worker activity was not recorded: %s", listed)
	}
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
	secondJobBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.echo", "input_version": "worker-v2"}), me.CSRF, "", true, http.StatusCreated)
	var secondQueued struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(secondJobBody, &secondQueued); err != nil || secondQueued.ID < 1 || secondQueued.ID == queued.ID {
		t.Fatalf("second job was not scheduled: %s %v", secondJobBody, err)
	}
	call(anon, http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1}`), "", registration.Token, false, http.StatusOK)
	claimedBody := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", registration.Token, false, http.StatusOK)
	var claimed struct {
		Job struct {
			ID           int64  `json:"id"`
			Status       string `json:"status"`
			InputVersion string `json:"input_version"`
		} `json:"job"`
	}
	if err = json.Unmarshal(claimedBody, &claimed); err != nil || claimed.Job.ID != queued.ID || claimed.Job.Status != "leased" {
		t.Fatalf("worker did not receive scheduled job: %s %v", claimedBody, err)
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
	call(anon, http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1}`), "", other.Token, false, http.StatusOK)
	otherClaimBody := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", other.Token, false, http.StatusOK)
	var otherClaim struct {
		Job struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err = json.Unmarshal(otherClaimBody, &otherClaim); err != nil || otherClaim.Job.ID != secondQueued.ID {
		t.Fatalf("second worker did not receive the next available job: %s %v", otherClaimBody, err)
	}
	call(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(queued.ID)+"/input", nil, "", other.Token, false, http.StatusNotFound)
	call(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(queued.ID)+"/media", nil, "", other.Token, false, http.StatusNotFound)
	inputBody := call(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(queued.ID)+"/input", nil, "", registration.Token, false, http.StatusOK)
	if bytes.Contains(inputBody, []byte(mediaRoot)) || bytes.Contains(inputBody, []byte(storageKey)) || !bytes.Contains(inputBody, []byte(`"media_url":"/api/worker/jobs/`)) {
		t.Fatalf("worker input exposed a storage path or omitted scoped media URL: %s", inputBody)
	}
	media := call(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(queued.ID)+"/media", nil, "", registration.Token, false, http.StatusOK)
	if !bytes.Equal(media, content) {
		t.Fatalf("worker received wrong media bytes: %q", media)
	}
	call(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(queued.ID)+"/heartbeat", []byte(`{}`), "", registration.Token, false, http.StatusOK)
	completed := call(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(queued.ID)+"/complete", []byte(`{"input_version":"worker-v1","result":{"provider":"fixture","sha256":"synthetic"}}`), "", registration.Token, false, http.StatusOK)
	if !bytes.Contains(completed, []byte(`"status":"done"`)) || !bytes.Contains(completed, []byte(`"provider":"fixture"`)) {
		t.Fatalf("worker result was not recorded: %s", completed)
	}
	call(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(secondQueued.ID)+"/fail", []byte(`{"error":"invalid_input"}`), "", other.Token, false, http.StatusOK)
	expiringJobBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.expire", "input_version": "worker-v1"}), me.CSRF, "", true, http.StatusCreated)
	var expiringJob struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(expiringJobBody, &expiringJob); err != nil || expiringJob.ID < 1 {
		t.Fatalf("expiring job was not scheduled: %s %v", expiringJobBody, err)
	}
	expiringClaimBody := call(anon, http.MethodPost, "/api/worker/jobs/claim", nil, "", registration.Token, false, http.StatusOK)
	var expiringClaim struct {
		Job struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err = json.Unmarshal(expiringClaimBody, &expiringClaim); err != nil || expiringClaim.Job.ID != expiringJob.ID {
		t.Fatalf("worker did not acquire expiring job: %s %v", expiringClaimBody, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE jobs SET lease_expires_at=now()-interval '1 second' WHERE id=$1`, expiringJob.ID); err != nil {
		t.Fatal(err)
	}
	call(anon, http.MethodGet, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/input", nil, "", registration.Token, false, http.StatusNotFound)
	call(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/heartbeat", []byte(`{}`), "", registration.Token, false, http.StatusConflict)
	call(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/complete", []byte(`{"input_version":"worker-v1","result":{"fixture":true}}`), "", registration.Token, false, http.StatusConflict)
	call(anon, http.MethodPost, "/api/worker/jobs/"+strconv64(expiringJob.ID)+"/fail", []byte(`{"error":"provider_unavailable"}`), "", registration.Token, false, http.StatusConflict)
	if _, err = pool.Exec(ctx, `UPDATE jobs SET status='failed',lease_owner=NULL,lease_expires_at=NULL,last_error='fixture_expired' WHERE id=$1`, expiringJob.ID); err != nil {
		t.Fatal(err)
	}
	call(admin, http.MethodDelete, "/api/admin/workers/"+registration.ID, nil, me.CSRF, "", true, http.StatusNoContent)
	call(anon, http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1}`), "", registration.Token, false, http.StatusUnauthorized)
	call(admin, http.MethodDelete, "/api/admin/workers/"+other.ID, nil, me.CSRF, "", true, http.StatusNoContent)
}
