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

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

func testJobControls(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `UPDATE job_control SET paused=false,max_concurrency=1 WHERE id=true`)
	})
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts"}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewTLSServer(server.Handler())
	t.Cleanup(ts.Close)
	server.Config.PublicURL = ts.URL
	admin, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	adminClient := ts.Client()
	adminClient.Jar = admin
	workerClient := &http.Client{Transport: ts.Client().Transport}
	csrf := ""
	call := func(client *http.Client, method, path string, body any, bearer, lease string, want int) []byte {
		t.Helper()
		var content []byte
		var requestErr error
		if body != nil {
			content, requestErr = json.Marshal(body)
			if requestErr != nil {
				t.Fatal(requestErr)
			}
		}
		req, requestErr := http.NewRequest(method, ts.URL+path, bytes.NewReader(content))
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		req.Header.Set("Origin", ts.URL)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if client == adminClient && csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if lease != "" {
			req.Header.Set("X-Worker-Lease-Token", lease)
		}
		response, requestErr := client.Do(req)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer response.Body.Close()
		result, requestErr := io.ReadAll(response.Body)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		if response.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, response.StatusCode, want, result)
		}
		return result
	}
	call(adminClient, http.MethodPost, "/api/setup", map[string]any{"code": code, "username": "administrator", "password": "synthetic-password-123"}, "", "", http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(call(adminClient, http.MethodGet, "/api/auth/me", nil, "", "", http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	csrf = me.CSRF
	call(workerClient, http.MethodGet, "/api/admin/job-controls", nil, "", "", http.StatusUnauthorized)
	var controls struct {
		Paused         bool `json:"paused"`
		MaxConcurrency int  `json:"max_concurrency"`
	}
	if err = json.Unmarshal(call(adminClient, http.MethodGet, "/api/admin/job-controls", nil, "", "", http.StatusOK), &controls); err != nil {
		t.Fatal(err)
	}
	if controls.Paused || controls.MaxConcurrency != 1 {
		t.Fatalf("unexpected initial queue controls: %+v", controls)
	}
	call(adminClient, http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": true, "max_concurrency": 3}, "", "", http.StatusOK)
	var ownerID, clipID int64
	if err = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'synthetic-control.mp4',1,now(),now()) RETURNING id`, ownerID).Scan(&clipID); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 4; i++ {
		call(adminClient, http.MethodPost, "/api/admin/jobs", map[string]any{"clip_id": clipID, "step": "fixture.control" + string(rune('a'+i-1)), "input_version": string(rune('0' + i)), "priority": 5 - i}, "", "", http.StatusCreated)
	}
	var workerTokens []string
	for _, name := range []string{"control-worker-a", "control-worker-b"} {
		body := call(adminClient, http.MethodPost, "/api/admin/workers", map[string]any{"name": name, "max_concurrency": 2}, "", "", http.StatusCreated)
		var registration struct {
			Token string `json:"token"`
		}
		if err = json.Unmarshal(body, &registration); err != nil || registration.Token == "" {
			t.Fatalf("worker registration: %s %v", body, err)
		}
		workerTokens = append(workerTokens, registration.Token)
		call(workerClient, http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"fixture.*"}}, registration.Token, "", http.StatusOK)
	}
	claim := func(token string) (int64, string) {
		t.Helper()
		body := call(workerClient, http.MethodPost, "/api/worker/jobs/claim", nil, token, "", http.StatusOK)
		var response struct {
			Job *struct {
				ID int64 `json:"id"`
			} `json:"job"`
			LeaseToken string `json:"lease_token"`
		}
		if decodeErr := json.Unmarshal(body, &response); decodeErr != nil {
			t.Fatalf("decode claim: %s %v", body, decodeErr)
		}
		if response.Job == nil {
			return 0, ""
		}
		return response.Job.ID, response.LeaseToken
	}
	if id, _ := claim(workerTokens[0]); id != 0 {
		t.Fatalf("paused queue issued a job: %d", id)
	}
	if id, _ := claim(workerTokens[1]); id != 0 {
		t.Fatalf("paused queue issued a job to second worker: %d", id)
	}
	call(adminClient, http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 1}, "", "", http.StatusOK)
	type claimResult struct {
		id     int64
		lease  string
		worker string
	}
	claims := make(chan claimResult, len(workerTokens))
	for _, token := range workerTokens {
		go func(token string) {
			id, lease := claim(token)
			claims <- claimResult{id: id, lease: lease, worker: token}
		}(token)
	}
	var firstID int64
	var firstLease string
	var firstWorker string
	for range workerTokens {
		result := <-claims
		if result.id != 0 {
			if firstID != 0 {
				t.Fatalf("global concurrency limit issued multiple simultaneous claims: %d and %d", firstID, result.id)
			}
			firstID, firstLease, firstWorker = result.id, result.lease, result.worker
		}
	}
	if firstID == 0 || firstLease == "" {
		t.Fatal("resumed queue did not issue exactly one job under global concurrency 1")
	}
	call(adminClient, http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 3}, "", "", http.StatusOK)
	secondID, secondLease := claim(firstWorker)
	if secondID == 0 || secondLease == "" {
		t.Fatal("worker did not claim below its capacity")
	}
	if id, _ := claim(firstWorker); id != 0 {
		t.Fatalf("worker exceeded its own announced capacity: %d", id)
	}
	otherWorker := workerTokens[0]
	if otherWorker == firstWorker {
		otherWorker = workerTokens[1]
	}
	thirdID, thirdLease := claim(otherWorker)
	if thirdID == 0 || thirdLease == "" {
		t.Fatal("second worker could not use remaining global capacity")
	}
	call(adminClient, http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": true, "max_concurrency": 3}, "", "", http.StatusOK)
	call(workerClient, http.MethodPost, "/api/worker/jobs/"+strconv64(firstID)+"/heartbeat", map[string]any{}, firstWorker, firstLease, http.StatusOK)
	call(workerClient, http.MethodPost, "/api/worker/jobs/"+strconv64(firstID)+"/complete", map[string]any{"input_version": "1", "result": map[string]any{"synthetic": true}}, firstWorker, firstLease, http.StatusOK)
	if id, _ := claim(firstWorker); id != 0 {
		t.Fatalf("paused queue issued a new job after an active task finished: %d", id)
	}
	call(workerClient, http.MethodPost, "/api/worker/jobs/"+strconv64(secondID)+"/heartbeat", map[string]any{}, firstWorker, secondLease, http.StatusOK)
	call(workerClient, http.MethodPost, "/api/worker/jobs/"+strconv64(thirdID)+"/complete", map[string]any{"input_version": "3", "result": map[string]any{"synthetic": true}}, otherWorker, thirdLease, http.StatusOK)
	call(workerClient, http.MethodPost, "/api/worker/jobs/"+strconv64(secondID)+"/complete", map[string]any{"input_version": "2", "result": map[string]any{"synthetic": true}}, firstWorker, secondLease, http.StatusOK)
	call(adminClient, http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 3}, "", "", http.StatusOK)
	if id, _ := claim(otherWorker); id == 0 {
		t.Fatal("resumed queue did not issue the remaining pending job")
	}
}
