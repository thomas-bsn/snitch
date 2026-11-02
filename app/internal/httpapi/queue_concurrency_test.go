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
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

// testQueueConcurrency checks, through the HTTP API, what the queue bar shows
// and what the real scheduler does: only leased tasks with a valid lease fill
// the ceiling, clips waiting for a Worker occupy no slot, and the global pause
// stops new launches while started tasks finish.
func testQueueConcurrency(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	clean := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM jobs`)
		_, _ = pool.Exec(ctx, `UPDATE job_control SET paused=false,max_concurrency=1 WHERE id=true`)
	}
	clean()
	t.Cleanup(clean)
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewUnstartedServer(server.Handler())
	server.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	admin := &http.Client{Jar: jar}
	anonymous := &http.Client{}
	csrf := ""
	call := func(client *http.Client, method, path string, body any, bearer, lease string, want int) []byte {
		t.Helper()
		var content []byte
		if body != nil {
			content, _ = json.Marshal(body)
		}
		req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(content))
		req.Header.Set("Origin", ts.URL)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if client == admin && csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if lease != "" {
			req.Header.Set("X-Worker-Lease-Token", lease)
		}
		resp, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return data
	}
	call(admin, http.MethodPost, "/api/setup", map[string]any{"code": code, "username": "administrator", "password": "synthetic-password-123"}, "", "", http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	_ = json.Unmarshal(call(admin, http.MethodGet, "/api/auth/me", nil, "", "", http.StatusOK), &me)
	csrf = me.CSRF
	var ownerID int64
	if err = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	clip := func(name string) int64 {
		t.Helper()
		var id int64
		if e := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,$2,1,now(),now()) RETURNING id`, ownerID, name).Scan(&id); e != nil {
			t.Fatal(e)
		}
		return id
	}
	schedule := func(clipID int64, step string, priority int) {
		t.Helper()
		call(admin, http.MethodPost, "/api/admin/jobs", map[string]any{"clip_id": clipID, "step": step, "input_version": "v1", "priority": priority}, "", "", http.StatusCreated)
	}
	// Two clips wait for a Worker that supports their step; their tasks have the
	// highest priority and must still not hold back the ready tasks.
	for i := 0; i < 2; i++ {
		schedule(clip(fmt.Sprintf("suspended-%d.mp4", i)), "transcription.audio.1", 9)
	}
	for i := 0; i < 4; i++ {
		schedule(clip(fmt.Sprintf("ready-%d.mp4", i)), fmt.Sprintf("fixture.ready%d", i), 5-i)
	}
	call(admin, http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 3}, "", "", http.StatusOK)
	var registration struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(call(admin, http.MethodPost, "/api/admin/workers", map[string]any{"name": "queue-bar-worker", "max_concurrency": 4}, "", "", http.StatusCreated), &registration)
	call(anonymous, http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 4, "supported_steps": []string{"fixture.*"}}, registration.Token, "", http.StatusOK)
	type queueView struct {
		Queue struct {
			Running        int  `json:"running"`
			Waiting        int  `json:"waiting"`
			MaxConcurrency int  `json:"max_concurrency"`
			Paused         bool `json:"paused"`
		} `json:"queue"`
	}
	bar := func() queueView {
		t.Helper()
		var v queueView
		if e := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/task-cards", nil, "", "", http.StatusOK), &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	type lease struct {
		id    int64
		token string
	}
	claim := func() *lease {
		t.Helper()
		var r struct {
			Job *struct {
				ID int64 `json:"id"`
			} `json:"job"`
			LeaseToken string `json:"lease_token"`
		}
		_ = json.Unmarshal(call(anonymous, http.MethodPost, "/api/worker/jobs/claim", nil, registration.Token, "", http.StatusOK), &r)
		if r.Job == nil {
			return nil
		}
		return &lease{r.Job.ID, r.LeaseToken}
	}
	if v := bar(); v.Queue.Running != 0 || v.Queue.MaxConcurrency != 3 || v.Queue.Waiting != 6 || v.Queue.Paused {
		t.Fatalf("idle bar: %+v", v.Queue)
	}
	// Ceiling 3 with two suspended clips: the ready tasks use the free slots.
	var held []*lease
	for i := 0; i < 3; i++ {
		l := claim()
		if l == nil {
			t.Fatalf("ready task %d was not launched despite free slots", i)
		}
		held = append(held, l)
	}
	if l := claim(); l != nil {
		t.Fatalf("ceiling exceeded: %d", l.id)
	}
	if v := bar(); v.Queue.Running != 3 || v.Queue.Waiting != 3 {
		t.Fatalf("bar at the ceiling: %+v", v.Queue)
	}
	// Pause: no new launch, started tasks finish.
	call(admin, http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": true, "max_concurrency": 3}, "", "", http.StatusOK)
	call(anonymous, http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/complete", held[0].id), map[string]any{"input_version": "v1", "result": map[string]any{"synthetic": true}}, registration.Token, held[0].token, http.StatusOK)
	if l := claim(); l != nil {
		t.Fatalf("paused queue launched a task: %d", l.id)
	}
	if v := bar(); !v.Queue.Paused || v.Queue.Running != 2 {
		t.Fatalf("paused bar: %+v", v.Queue)
	}
	// A lease that expired no longer fills a slot.
	if _, err = pool.Exec(ctx, `UPDATE jobs SET lease_expires_at=now()-interval '1 second' WHERE id=$1`, held[1].id); err != nil {
		t.Fatal(err)
	}
	if v := bar(); v.Queue.Running != 1 {
		t.Fatalf("expired lease must not count: %+v", v.Queue)
	}
	call(admin, http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 3}, "", "", http.StatusOK)
	if l := claim(); l == nil {
		t.Fatal("resumed queue did not launch the remaining ready task")
	}
}
