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
	"receipts/internal/config"
	"receipts/internal/setup"
)

func testJobQueue(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	mediaRoot := t.TempDir()
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts", UploadRoot: mediaRoot}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
}
