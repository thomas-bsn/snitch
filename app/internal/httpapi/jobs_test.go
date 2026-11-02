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
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/jobs"
	"snitch/internal/logging"
	"snitch/internal/setup"
)

func testJobQueue(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	mediaRoot := t.TempDir()
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), UploadRoot: mediaRoot}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	newTestServer := func() *httptest.Server {
		ts := httptest.NewUnstartedServer(server.Handler())
		server.Config.PublicURL = "http://" + ts.Listener.Addr().String()
		ts.Start()
		return ts
	}
	ts := newTestServer()
	defer func() { ts.Close() }()
	jar, _ := cookiejar.New(nil)
	admin := &http.Client{Jar: jar}
	call := func(c *http.Client, method, path string, body []byte, csrf string, want int) (*http.Response, []byte) {
		t.Helper()
		req, e := http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Origin", ts.URL)
		if len(body) != 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		resp, e := c.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(resp.Body)
		resp.Body.Close()
		if e != nil {
			t.Fatal(e)
		}
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return resp, data
	}
	call(admin, http.MethodPost, "/api/setup", []byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`), "", http.StatusCreated)
	_, body := call(admin, http.MethodGet, "/api/auth/me", nil, "", http.StatusOK)
	var current struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(body, &current); err != nil {
		t.Fatal(err)
	}
	var ownerID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	var clipID int64
	if err := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'synthetic-job-fixture.mp4',12,now(),now()) RETURNING id`, ownerID).Scan(&clipID); err != nil {
		t.Fatal(err)
	}
	call(http.DefaultClient, http.MethodGet, "/api/admin/transcriptions/batch-preview", nil, "", http.StatusUnauthorized)
	call(http.DefaultClient, http.MethodPost, "/api/admin/transcriptions/batch-schedule", mustJSON(t, map[string]bool{"confirm": true}), "", http.StatusUnauthorized)
	_, previewBody := call(admin, http.MethodGet, "/api/admin/transcriptions/batch-preview", nil, "", http.StatusOK)
	var preview struct {
		Eligible          int `json:"eligible"`
		AlreadyProcessed  int `json:"already_processed"`
		AlreadyQueued     int `json:"already_queued"`
		Unavailable       int `json:"unavailable"`
		SelectionRequired int `json:"selection_required"`
	}
	if err := json.Unmarshal(previewBody, &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Unavailable != 1 || preview.Eligible != 0 {
		t.Fatalf("preview should identify the synthetic clip without readable media: %+v", preview)
	}
	call(admin, http.MethodPost, "/api/admin/transcriptions/batch-schedule", []byte(`{"confirm":false}`), current.CSRF, http.StatusBadRequest)
	call(http.DefaultClient, http.MethodGet, "/api/admin/jobs", nil, "", http.StatusUnauthorized)
	input := map[string]any{"clip_id": clipID, "step": "fixture.inspect", "input_version": "fixture-v1", "priority": 5}
	_, body = call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, input), current.CSRF, http.StatusCreated)
	var first struct {
		ID           int64     `json:"id"`
		ClipID       int64     `json:"clip_id"`
		Step         string    `json:"step"`
		InputVersion string    `json:"input_version"`
		Status       string    `json:"status"`
		Priority     int       `json:"priority"`
		Attempts     int       `json:"attempts"`
		MaxAttempts  int       `json:"max_attempts"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		LastError    string    `json:"last_error"`
	}
	if err := json.Unmarshal(body, &first); err != nil {
		t.Fatal(err)
	}
	if first.ID < 1 || first.ClipID != clipID || first.Step != input["step"] || first.InputVersion != input["input_version"] || first.Status != "pending" || first.Priority != 5 || first.Attempts != 0 || first.MaxAttempts < 1 || first.CreatedAt.IsZero() || first.UpdatedAt.IsZero() {
		t.Fatalf("unexpected newly scheduled job: %+v", first)
	}
	resp, body := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, input), current.CSRF, http.StatusOK)
	if resp.StatusCode != http.StatusOK {
		t.Fatal("duplicate scheduling must return the existing job")
	}
	var duplicate struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &duplicate); err != nil || duplicate.ID != first.ID {
		t.Fatalf("duplicate schedule did not return existing job: %+v %v", duplicate, err)
	}
	_, body = call(admin, http.MethodGet, fmt.Sprintf("/api/admin/jobs?status=pending&clip_id=%d&step=fixture.inspect&page=1&page_size=10", clipID), nil, "", http.StatusOK)
	var page struct {
		Items    []json.RawMessage `json:"items"`
		Page     int               `json:"page"`
		PageSize int               `json:"page_size"`
		Total    int               `json:"total"`
	}
	if err := json.Unmarshal(body, &page); err != nil || len(page.Items) != 1 || page.Page != 1 || page.PageSize != 10 || page.Total != 1 {
		t.Fatalf("unexpected filtered job page: %s %v", body, err)
	}
	if strings.Contains(string(body), "/data/media") || strings.Contains(string(body), mediaRoot) {
		t.Fatalf("job response leaked a server path: %s", body)
	}
	call(admin, http.MethodGet, "/api/admin/jobs?status=unknown", nil, "", http.StatusBadRequest)
	call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "../shell", "input_version": "v1"}), current.CSRF, http.StatusBadRequest)
	call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": 999999, "step": "fixture.inspect", "input_version": "v1"}), current.CSRF, http.StatusNotFound)
	if _, err := pool.Exec(ctx, `INSERT INTO users(username,role,approved) VALUES('ordinary-member','user',true)`); err != nil {
		t.Fatal(err)
	}
	var memberID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='ordinary-member'`).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	member := httptest.NewRecorder()
	if err := server.issueSession(member, httptest.NewRequest(http.MethodGet, ts.URL, nil), memberID); err != nil {
		t.Fatal(err)
	}
	memberJar, _ := cookiejar.New(nil)
	parsed, _ := url.Parse(ts.URL)
	memberJar.SetCookies(parsed, member.Result().Cookies())
	memberClient := &http.Client{Jar: memberJar}
	call(memberClient, http.MethodGet, "/api/admin/jobs", nil, "", http.StatusForbidden)
	call(memberClient, http.MethodPost, "/api/admin/jobs", mustJSON(t, input), "", http.StatusForbidden)
	// A second API process over the same database can still read the pending task.
	ts.Close()
	ts = newTestServer()
	_, body = call(admin, http.MethodGet, "/api/admin/jobs?clip_id="+strconv64(clipID), nil, "", http.StatusOK)
	if err := json.Unmarshal(body, &page); err != nil || page.Total != 1 {
		t.Fatalf("job disappeared after application restart: %s %v", body, err)
	}
	for _, workflow := range []string{"active", "done"} {
		page, err := server.Jobs.List(ctx, jobs.Filter{Workflow: workflow, Page: 1, PageSize: 1})
		if err != nil {
			t.Fatalf("workflow %s: %v", workflow, err)
		}
		ids := map[int64]bool{}
		for _, job := range page.Items {
			ids[job.ClipID] = true
		}
		if len(ids) > 1 {
			t.Fatal("workflow pagination split clips")
		}
		if workflow == "done" {
			for _, job := range page.Items {
				if job.Status != jobs.Done {
					t.Fatal("unfinished clip in completed history")
				}
			}
		}
	}

}

func testJobRetryOperations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	logs := &lockedBuffer{}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), UploadRoot: t.TempDir()}, logging.New(logs, "snitch", "INFO"))
	ts := httptest.NewTLSServer(server.Handler())
	defer ts.Close()
	server.Config.PublicURL = ts.URL
	jar, _ := cookiejar.New(nil)
	admin := ts.Client()
	admin.Jar = jar
	workerClient := &http.Client{Transport: ts.Client().Transport}
	call := func(c *http.Client, method, path string, body []byte, csrf string, want int) []byte {
		t.Helper()
		req, requestErr := http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		req.Header.Set("Origin", ts.URL)
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		response, requestErr := c.Do(req)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer response.Body.Close()
		data, requestErr := io.ReadAll(response.Body)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		if response.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, response.StatusCode, want, data)
		}
		return data
	}
	callWorker := func(method, path string, body []byte, csrf, token, leaseToken string, want int) []byte {
		t.Helper()
		req, requestErr := http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("Origin", ts.URL)
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if leaseToken != "" {
			req.Header.Set("X-Worker-Lease-Token", leaseToken)
		}
		response, requestErr := workerClient.Do(req)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer response.Body.Close()
		data, requestErr := io.ReadAll(response.Body)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		if response.StatusCode != want {
			t.Fatalf("worker %s %s: got %d want %d: %s", method, path, response.StatusCode, want, data)
		}
		return data
	}
	call(admin, http.MethodPost, "/api/setup", []byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`), "", http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/auth/me", nil, "", http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	var ownerID int64
	if err = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	var clipID int64
	if err = pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'retry-fixture.mp4',7,now(),now()) RETURNING id`, ownerID).Scan(&clipID); err != nil {
		t.Fatal(err)
	}
	jobBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.retry", "input_version": "v1", "max_attempts": 2}), me.CSRF, http.StatusCreated)
	var job struct {
		ID          int64     `json:"id"`
		Status      string    `json:"status"`
		Attempts    int       `json:"attempts"`
		MaxAttempts int       `json:"max_attempts"`
		Cycle       int       `json:"cycle"`
		NextAttempt time.Time `json:"next_attempt_at"`
	}
	if err = json.Unmarshal(jobBody, &job); err != nil || job.ID < 1 || job.MaxAttempts != 2 {
		t.Fatalf("scheduled retry job: %s %v", jobBody, err)
	}
	workerBody := call(admin, http.MethodPost, "/api/admin/workers", mustJSON(t, map[string]any{"name": "retry-fixture-worker", "max_concurrency": 1}), me.CSRF, http.StatusCreated)
	var worker struct {
		Token string `json:"token"`
	}
	if err = json.Unmarshal(workerBody, &worker); err != nil || worker.Token == "" {
		t.Fatalf("worker registration: %s %v", workerBody, err)
	}
	callWorker(http.MethodPost, "/api/worker/pulse", []byte(`{"capacity":1,"supported_steps":["fixture.*"]}`), "", worker.Token, "", http.StatusOK)
	claim := func() struct {
		LeaseToken string `json:"lease_token"`
		Job        struct {
			ID           int64  `json:"id"`
			InputVersion string `json:"input_version"`
		} `json:"job"`
	} {
		t.Helper()
		var got struct {
			LeaseToken string `json:"lease_token"`
			Job        struct {
				ID           int64  `json:"id"`
				InputVersion string `json:"input_version"`
			} `json:"job"`
		}
		if err := json.Unmarshal(callWorker(http.MethodPost, "/api/worker/jobs/claim", nil, "", worker.Token, "", http.StatusOK), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	firstClaim := claim()
	if firstClaim.Job.ID != job.ID || firstClaim.LeaseToken == "" {
		t.Fatalf("first attempt was not leased: %+v", firstClaim)
	}
	jobPath := "/api/worker/jobs/" + strconv64(job.ID) + "/fail"
	failed := callWorker(http.MethodPost, jobPath, []byte(`{"error":"provider_unavailable"}`), "", worker.Token, firstClaim.LeaseToken, http.StatusOK)
	if err = json.Unmarshal(failed, &job); err != nil || job.Status != "retry_wait" || job.Attempts != 1 || job.NextAttempt.Before(time.Now()) {
		t.Fatalf("transient failure was not scheduled for retry: %s %v", failed, err)
	}
	duplicateRetry := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.retry", "input_version": "v1", "max_attempts": 2}), me.CSRF, http.StatusOK)
	var retryDuplicate struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err = json.Unmarshal(duplicateRetry, &retryDuplicate); err != nil || retryDuplicate.ID != job.ID || retryDuplicate.Status != "retry_wait" {
		t.Fatalf("retry_wait task was not treated as active for idempotency: %s %v", duplicateRetry, err)
	}
	var emptyClaim struct {
		Job *struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err = json.Unmarshal(callWorker(http.MethodPost, "/api/worker/jobs/claim", nil, "", worker.Token, "", http.StatusOK), &emptyClaim); err != nil || emptyClaim.Job != nil {
		t.Fatalf("retry was claimable before next_attempt_at: %+v %v", emptyClaim, err)
	}
	time.Sleep(time.Until(job.NextAttempt) + 50*time.Millisecond)
	secondClaim := claim()
	if secondClaim.Job.ID != job.ID || secondClaim.LeaseToken == "" {
		t.Fatalf("retry was not claimed when due: %+v", secondClaim)
	}
	failed = callWorker(http.MethodPost, jobPath, []byte(`{"error":"provider_unavailable"}`), "", worker.Token, secondClaim.LeaseToken, http.StatusOK)
	if err = json.Unmarshal(failed, &job); err != nil || job.Status != "dead" || job.Attempts != 2 {
		t.Fatalf("exhausted task did not become dead: %s %v", failed, err)
	}
	jobPathAdmin := "/api/admin/jobs/" + strconv64(job.ID)
	relaunched := call(admin, http.MethodPost, jobPathAdmin+"/retry", nil, me.CSRF, http.StatusOK)
	if err = json.Unmarshal(relaunched, &job); err != nil || job.Status != "pending" || job.Attempts != 0 || job.MaxAttempts != 2 || job.Cycle != 1 {
		t.Fatalf("relaunch did not preserve attempt history and restore a retry budget: %s %v", relaunched, err)
	}
	history := call(admin, http.MethodGet, jobPathAdmin+"/history", nil, "", http.StatusOK)
	for _, event := range []string{"attempt_started", "retry_scheduled", "marked_dead", "admin_relaunched"} {
		if !bytes.Contains(history, []byte(event)) {
			t.Fatalf("job history omitted %q: %s", event, history)
		}
	}
	if bytes.Contains(history, []byte("/private/")) || bytes.Contains(history, []byte("/tmp/")) {
		t.Fatalf("job history leaked a local path: %s", history)
	}
	thirdClaim := claim()
	if thirdClaim.Job.ID != job.ID || thirdClaim.LeaseToken == "" {
		t.Fatalf("relaunched task could not be claimed: %+v", thirdClaim)
	}
	call(admin, http.MethodDelete, jobPathAdmin, nil, me.CSRF, http.StatusNoContent)
	callWorker(http.MethodPost, "/api/worker/jobs/"+strconv64(job.ID)+"/complete", []byte(`{"input_version":"v1","result":{"fixture":true}}`), "", worker.Token, thirdClaim.LeaseToken, http.StatusConflict)
	var cancelledPage struct {
		Items []struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/admin/jobs?status=cancelled", nil, "", http.StatusOK), &cancelledPage); err != nil || len(cancelledPage.Items) != 1 || cancelledPage.Items[0].ID != job.ID || cancelledPage.Items[0].Status != "cancelled" {
		t.Fatalf("cancelled job is missing from filtered queue: %+v %v", cancelledPage, err)
	}
	call(admin, http.MethodGet, jobPathAdmin+"/history", nil, "", http.StatusOK)
	permanentBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.permanent", "input_version": "v1", "max_attempts": 3}), me.CSRF, http.StatusCreated)
	var permanent struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(permanentBody, &permanent); err != nil || permanent.ID < 1 {
		t.Fatalf("schedule permanent-failure fixture: %s %v", permanentBody, err)
	}
	permanentClaim := claim()
	if permanentClaim.Job.ID != permanent.ID {
		t.Fatalf("permanent-failure job not claimed: %+v", permanentClaim)
	}
	permanentPath := "/api/worker/jobs/" + strconv64(permanent.ID) + "/fail"
	permanentResult := callWorker(http.MethodPost, permanentPath, []byte(`{"error":"invalid_input"}`), "", worker.Token, permanentClaim.LeaseToken, http.StatusOK)
	if err = json.Unmarshal(permanentResult, &job); err != nil || job.Status != "dead" || job.Attempts != 1 {
		t.Fatalf("permanent input error was not dead immediately: %s %v", permanentResult, err)
	}
	redactionBody := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.redaction", "input_version": "v1", "max_attempts": 1}), me.CSRF, http.StatusCreated)
	var redactionJob struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(redactionBody, &redactionJob); err != nil || redactionJob.ID < 1 {
		t.Fatalf("schedule redaction fixture: %s %v", redactionBody, err)
	}
	redactionClaim := claim()
	if redactionClaim.Job.ID != redactionJob.ID {
		t.Fatalf("redaction fixture was not claimed: %+v", redactionClaim)
	}
	rawError := "worker crashed at /tmp/synthetic-private/video.mp4 token=secret"
	redacted := callWorker(http.MethodPost, "/api/worker/jobs/"+strconv64(redactionJob.ID)+"/fail", mustJSON(t, map[string]string{"error": rawError}), "", worker.Token, redactionClaim.LeaseToken, http.StatusOK)
	if !bytes.Contains(redacted, []byte(`"last_error":"worker_failed"`)) || bytes.Contains(redacted, []byte(rawError)) || bytes.Contains(redacted, []byte("synthetic-private")) || bytes.Contains(redacted, []byte("secret")) {
		t.Fatalf("worker error was not safely redacted: %s", redacted)
	}
	var memberID int64
	if err = pool.QueryRow(ctx, `INSERT INTO users(username,role,approved) VALUES('diagnostic-member','user',true) RETURNING id`).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	memberSession := httptest.NewRecorder()
	if err = server.issueSession(memberSession, httptest.NewRequest(http.MethodGet, ts.URL, nil), memberID); err != nil {
		t.Fatal(err)
	}
	memberJar, _ := cookiejar.New(nil)
	memberURL, _ := url.Parse(ts.URL)
	memberJar.SetCookies(memberURL, memberSession.Result().Cookies())
	memberClient := &http.Client{Transport: ts.Client().Transport, Jar: memberJar}
	// Failure causes: every Whisper family keeps its own code and diagnostic, and only Administrators read them.
	for i, want := range []struct {
		code, action string
		status       int
	}{
		{"whisper_network", "check_transcription_service", 0},
		{"whisper_timeout", "check_transcription_service", 0},
		{"whisper_http_error", "check_transcription_service", 503},
		{"whisper_model_missing", "install_transcription_model", 404},
		{"whisper_invalid_response", "check_transcription_service", 200},
		{"gpu_unavailable", "check_gpu", 0},
	} {
		scheduled := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": fmt.Sprintf("fixture.cause%d", i), "input_version": "v1", "max_attempts": 1}), me.CSRF, http.StatusCreated)
		var causeJob struct {
			ID int64 `json:"id"`
		}
		if err = json.Unmarshal(scheduled, &causeJob); err != nil || causeJob.ID < 1 {
			t.Fatalf("schedule cause fixture: %s %v", scheduled, err)
		}
		causeClaim := claim()
		if causeClaim.Job.ID != causeJob.ID {
			t.Fatalf("cause fixture not claimed: %+v", causeClaim)
		}
		payload := map[string]any{"error": want.code, "provider": "openai-compatible", "model": "whisper-synthetic", "revision": "rev-synthetic", "duration_ms": 1234}
		if want.status != 0 {
			payload["http_status"] = want.status
		}
		stored := callWorker(http.MethodPost, "/api/worker/jobs/"+strconv64(causeJob.ID)+"/fail", mustJSON(t, payload), "", worker.Token, causeClaim.LeaseToken, http.StatusOK)
		if !bytes.Contains(stored, []byte(`"last_error":"`+want.code+`"`)) {
			t.Fatalf("known cause %s was overwritten: %s", want.code, stored)
		}
		causeHistory := call(admin, http.MethodGet, "/api/admin/jobs/"+strconv64(causeJob.ID)+"/history", nil, "", http.StatusOK)
		var events struct {
			Items []map[string]any `json:"items"`
		}
		if err = json.Unmarshal(causeHistory, &events); err != nil {
			t.Fatal(err)
		}
		var dead map[string]any
		for _, e := range events.Items {
			if e["event"] == "marked_dead" {
				dead = e
			}
		}
		if dead == nil || dead["error_code"] != want.code || dead["provider"] != "openai-compatible" || dead["model"] != "whisper-synthetic" || dead["revision"] != "rev-synthetic" || dead["duration_ms"] != float64(1234) || dead["step"] != fmt.Sprintf("fixture.cause%d", i) || dead["recommended_action"] != want.action {
			t.Fatalf("diagnostic for %s not persisted for the Administrator: %s", want.code, causeHistory)
		}
		if want.status != 0 && dead["http_status"] != float64(want.status) {
			t.Fatalf("http status of %s not persisted: %s", want.code, causeHistory)
		}
		if want.status == 0 && dead["http_status"] != nil {
			t.Fatalf("unexpected http status for %s: %s", want.code, causeHistory)
		}
		call(memberClient, http.MethodGet, "/api/admin/jobs/"+strconv64(causeJob.ID)+"/history", nil, "", http.StatusForbidden)
		call(memberClient, http.MethodGet, "/api/admin/jobs?clip_id="+strconv64(clipID), nil, "", http.StatusForbidden)
	}
	// An old Worker sends an unknown code without any diagnostic field: it is stored as worker_failed.
	legacy := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.legacy", "input_version": "v1", "max_attempts": 1}), me.CSRF, http.StatusCreated)
	var legacyJob struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(legacy, &legacyJob); err != nil {
		t.Fatal(err)
	}
	legacyClaim := claim()
	callWorker(http.MethodPost, "/api/worker/jobs/"+strconv64(legacyJob.ID)+"/fail", []byte(`{"error":"some_future_code"}`), "", worker.Token, legacyClaim.LeaseToken, http.StatusOK)
	legacyHistory := call(admin, http.MethodGet, "/api/admin/jobs/"+strconv64(legacyJob.ID)+"/history", nil, "", http.StatusOK)
	if !bytes.Contains(legacyHistory, []byte(`"error_code":"worker_failed"`)) || bytes.Contains(legacyHistory, []byte(`"provider"`)) || bytes.Contains(legacyHistory, []byte(`"http_status"`)) {
		t.Fatalf("old Worker failure not stored as worker_failed without detail: %s", legacyHistory)
	}
	// Diagnostic fields are bounded and typed.
	rejected := call(admin, http.MethodPost, "/api/admin/jobs", mustJSON(t, map[string]any{"clip_id": clipID, "step": "fixture.bounds", "input_version": "v1", "max_attempts": 1}), me.CSRF, http.StatusCreated)
	var boundsJob struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(rejected, &boundsJob); err != nil {
		t.Fatal(err)
	}
	boundsClaim := claim()
	for _, bad := range []map[string]any{{"error": "whisper_http_error", "http_status": 99}, {"error": "whisper_http_error", "http_status": 600}, {"error": "whisper_timeout", "duration_ms": -1}, {"error": "whisper_timeout", "model": strings.Repeat("m", 200)}, {"error": "whisper_timeout", "provider": "has spaces and /tmp/path"}} {
		callWorker(http.MethodPost, "/api/worker/jobs/"+strconv64(boundsJob.ID)+"/fail", mustJSON(t, bad), "", worker.Token, boundsClaim.LeaseToken, http.StatusBadRequest)
	}
	callWorker(http.MethodPost, "/api/worker/jobs/"+strconv64(boundsJob.ID)+"/fail", []byte(`{"error":"invalid_input"}`), "", worker.Token, boundsClaim.LeaseToken, http.StatusOK)
	// Observability: a failed then relaunched job is traceable by clip, job, cycle, attempt and worker.
	out := logs.String()
	c, j := float64(clipID), float64(firstClaim.Job.ID)
	for _, want := range []struct {
		event string
		match map[string]any
	}{
		{"job_claimed", map[string]any{"clip_id": c, "job_id": j, "cycle": float64(0), "attempt": float64(1)}},
		{"job_claimed", map[string]any{"job_id": j, "cycle": float64(0), "attempt": float64(2)}},
		{"job_claimed", map[string]any{"job_id": j, "cycle": float64(1), "attempt": float64(1)}},
		{"job_failed", map[string]any{"clip_id": c, "job_id": j, "cycle": float64(0), "attempt": float64(1), "error_code": "provider_unavailable"}},
		{"job_retry_scheduled", map[string]any{"job_id": j, "attempt": float64(1)}},
		{"job_dead", map[string]any{"job_id": j, "attempt": float64(2), "error_code": "provider_unavailable"}},
		{"job_cancelled", map[string]any{"job_id": j, "cycle": float64(1)}},
		{"job_failed", map[string]any{"job_id": float64(permanent.ID), "error_code": "invalid_input"}},
		{"job_failed", map[string]any{"job_id": float64(redactionJob.ID), "error_code": "worker_failed"}},
	} {
		entries := logEvents(t, out, want.event)
		if !hasLog(entries, want.match) {
			t.Fatalf("log event %s %v missing in:\n%s", want.event, want.match, out)
		}
		for _, e := range entries {
			if e["service"] != "snitch" || (e["worker_id"] == nil && want.event != "job_cancelled") {
				t.Fatalf("event %s lacks service or worker context: %v", want.event, e)
			}
		}
	}
	for _, secret := range []string{firstClaim.LeaseToken, secondClaim.LeaseToken, thirdClaim.LeaseToken, worker.Token, "synthetic-private", "token=secret"} {
		if strings.Contains(out, secret) {
			t.Fatalf("log leaked a secret: %s", out)
		}
	}
	// Polls are silent at INFO: the empty claim and heartbeats write nothing.
	for _, noisy := range []string{"claim_empty", "job_heartbeat"} {
		if len(logEvents(t, out, noisy)) != 0 {
			t.Fatalf("%s must not be logged at INFO", noisy)
		}
	}
}
