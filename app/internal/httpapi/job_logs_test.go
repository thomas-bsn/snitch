//go:build integration

package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

// testJobLogs covers the Worker's log batches: lease check, batch bounds, retention of the 50 most
// recent lines, administrator-only streaming, and deletion with the job.
func testJobLogs(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewTLSServer(server.Handler())
	t.Cleanup(ts.Close)
	server.Config.PublicURL = ts.URL
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	adminClient := ts.Client()
	adminClient.Jar = jar
	anonymous := &http.Client{Transport: ts.Client().Transport}
	csrf := ""
	call := func(client *http.Client, method, path string, body any, bearer, lease string, want int) []byte {
		t.Helper()
		var content []byte
		if body != nil {
			content, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}
		req, err := http.NewRequest(method, ts.URL+path, bytes.NewReader(content))
		if err != nil {
			t.Fatal(err)
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
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		result, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
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
	var ownerID, clipID int64
	if err = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'synthetic-logs.mp4',1,now(),now()) RETURNING id`, ownerID).Scan(&clipID); err != nil {
		t.Fatal(err)
	}
	call(adminClient, http.MethodPost, "/api/admin/jobs", map[string]any{"clip_id": clipID, "step": "fixture.logs", "input_version": "1", "priority": 5}, "", "", http.StatusCreated)
	body := call(adminClient, http.MethodPost, "/api/admin/workers", map[string]any{"name": "logs-worker", "max_concurrency": 1}, "", "", http.StatusCreated)
	var registration struct {
		Token string `json:"token"`
	}
	if err = json.Unmarshal(body, &registration); err != nil || registration.Token == "" {
		t.Fatalf("worker registration: %s %v", body, err)
	}
	token := registration.Token
	call(anonymous, http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"fixture.*"}}, token, "", http.StatusOK)
	var claimed struct {
		Job *struct {
			ID int64 `json:"id"`
		} `json:"job"`
		LeaseToken string `json:"lease_token"`
	}
	if err = json.Unmarshal(call(anonymous, http.MethodPost, "/api/worker/jobs/claim", nil, token, "", http.StatusOK), &claimed); err != nil || claimed.Job == nil {
		t.Fatalf("claim: %v", err)
	}
	jobID := claimed.Job.ID
	lease := claimed.LeaseToken
	logsPath := fmt.Sprintf("/api/worker/jobs/%d/logs", jobID)
	batch := func(from, to int) map[string]any {
		lines := []string{}
		for i := from; i <= to; i++ {
			lines = append(lines, fmt.Sprintf(`{"event":"step","n":%d}`, i))
		}
		return map[string]any{"lines": lines}
	}

	t.Run("only the lease holder can send lines", func(t *testing.T) {
		call(anonymous, http.MethodPost, logsPath, batch(1, 1), token, "lt_synthetic-wrong-lease-token-0000000000", http.StatusConflict)
		call(anonymous, http.MethodPost, logsPath, batch(1, 1), "", lease, http.StatusUnauthorized)
	})
	t.Run("batches are bounded", func(t *testing.T) {
		call(anonymous, http.MethodPost, logsPath, map[string]any{"lines": []string{}}, token, lease, http.StatusBadRequest)
		call(anonymous, http.MethodPost, logsPath, batch(1, 51), token, lease, http.StatusBadRequest)
		call(anonymous, http.MethodPost, logsPath, map[string]any{"lines": []string{strings.Repeat("x", 2001)}}, token, lease, http.StatusBadRequest)
	})
	t.Run("only the last fifty lines are kept, in order", func(t *testing.T) {
		call(anonymous, http.MethodPost, logsPath, batch(1, 50), token, lease, http.StatusOK)
		call(anonymous, http.MethodPost, logsPath, batch(51, 100), token, lease, http.StatusOK)
		call(anonymous, http.MethodPost, logsPath, batch(101, 120), token, lease, http.StatusOK)
		var count int
		var first, last string
		if err = pool.QueryRow(ctx, `SELECT count(*), COALESCE((SELECT text FROM job_log_lines WHERE job_id=$1 ORDER BY id ASC LIMIT 1), ''), COALESCE((SELECT text FROM job_log_lines WHERE job_id=$1 ORDER BY id DESC LIMIT 1), '') FROM job_log_lines WHERE job_id=$1`, jobID).Scan(&count, &first, &last); err != nil {
			t.Fatal(err)
		}
		if count != 50 || first != `{"event":"step","n":71}` || last != `{"event":"step","n":120}` {
			t.Fatalf("kept %d lines from %q to %q", count, first, last)
		}
	})
	t.Run("administrators stream the kept lines and nobody else", func(t *testing.T) {
		call(anonymous, http.MethodGet, fmt.Sprintf("/api/admin/jobs/%d/logs/events", jobID), nil, "", "", http.StatusUnauthorized)
		call(adminClient, http.MethodGet, "/api/admin/jobs/999999/logs/events", nil, "", "", http.StatusNotFound)
		ctxStream, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctxStream, http.MethodGet, fmt.Sprintf("%s/api/admin/jobs/%d/logs/events", ts.URL, jobID), nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := adminClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("stream status %d", response.StatusCode)
		}
		reader := bufio.NewReader(response.Body)
		var data string
		for data == "" {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				t.Fatalf("no snapshot: %v", readErr)
			}
			if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(strings.TrimSpace(line), "data: ")
			}
		}
		var frame struct {
			Snapshot struct {
				JobID int64 `json:"job_id"`
				Lines []struct {
					Seq  int64  `json:"seq"`
					Text string `json:"text"`
				} `json:"lines"`
			} `json:"snapshot"`
		}
		if err = json.Unmarshal([]byte(data), &frame); err != nil {
			t.Fatal(err)
		}
		if frame.Snapshot.JobID != jobID || len(frame.Snapshot.Lines) != 50 || frame.Snapshot.Lines[49].Text != `{"event":"step","n":120}` {
			t.Fatalf("unexpected snapshot: job %d, %d lines", frame.Snapshot.JobID, len(frame.Snapshot.Lines))
		}
	})
	t.Run("credentials and clip paths are masked before storage", func(t *testing.T) {
		line := `{"event":"probe","detail":"Bearer abc.def hf_SyntheticToken123 https://ann:pw-synth@hub.example/x?sig=abc /private/var/synthetic/audio.wav"}`
		call(anonymous, http.MethodPost, logsPath, map[string]any{"lines": []string{line}}, token, lease, http.StatusOK)
		var stored string
		if err = pool.QueryRow(ctx, `SELECT text FROM job_log_lines WHERE job_id=$1 ORDER BY id DESC LIMIT 1`, jobID).Scan(&stored); err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"abc.def", "hf_SyntheticToken123", "ann:pw-synth", "sig=abc", "/private/var"} {
			if strings.Contains(stored, secret) {
				t.Fatalf("a stored log line keeps %q: %s", secret, stored)
			}
		}
		if !strings.Contains(stored, "Bearer [masked]") || !strings.Contains(stored, "https://hub.example/[masked]") || !strings.Contains(stored, "[masked_path]") {
			t.Fatalf("the masked values must be marked: %s", stored)
		}
	})
	t.Run("deleting the job deletes its lines", func(t *testing.T) {
		if _, err = pool.Exec(ctx, `DELETE FROM jobs WHERE id=$1`, jobID); err != nil {
			t.Fatal(err)
		}
		var remaining int
		if err = pool.QueryRow(ctx, `SELECT count(*) FROM job_log_lines WHERE job_id=$1`, jobID).Scan(&remaining); err != nil {
			t.Fatal(err)
		}
		if remaining != 0 {
			t.Fatalf("%d lines survived the job", remaining)
		}
	})
}
