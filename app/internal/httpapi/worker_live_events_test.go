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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

// testWorkerLiveEvents covers the live Worker pages: the list and the page of one Worker
// receive a fresh snapshot when a Worker connects, when it goes silent (without any write, so
// the server publishes it), and when it is created. Only administrators can open the streams.
func testWorkerLiveEvents(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	server.Events.Poll, server.Events.Heartbeat = 50*time.Millisecond, 300*time.Millisecond
	ts := httptest.NewTLSServer(server.Handler())
	// The streams close first (deferred last): the server does not wait for open connections.
	var streams []io.Closer
	defer ts.Close()
	defer func() {
		for _, s := range streams {
			_ = s.Close()
		}
	}()
	server.Config.PublicURL = ts.URL
	jar, _ := cookiejar.New(nil)
	admin := ts.Client()
	admin.Jar = jar
	anon := &http.Client{Transport: ts.Client().Transport}
	call := func(client *http.Client, method, path string, body any, csrf, token string, want int) []byte {
		t.Helper()
		var payload []byte
		if body != nil {
			payload = mustJSON(t, body)
		}
		req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(payload))
		req.Header.Set("Origin", ts.URL)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return data
	}
	call(admin, http.MethodPost, "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", "", http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/auth/me", nil, "", "", http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	var created struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err = json.Unmarshal(call(admin, http.MethodPost, "/api/admin/workers", map[string]any{"name": "synthetic-live", "max_concurrency": 1}, me.CSRF, "", http.StatusCreated), &created); err != nil {
		t.Fatal(err)
	}

	call(anon, http.MethodGet, "/api/admin/workers/events", nil, "", "", http.StatusUnauthorized)
	call(anon, http.MethodGet, "/api/admin/workers/"+created.ID+"/events", nil, "", "", http.StatusUnauthorized)

	// untilList reads list snapshots until the Worker is in the wanted connection state.
	untilList := func(frames <-chan sseFrame, connected bool) {
		t.Helper()
		for {
			f := nextFrame(t, frames)
			if f.Event != "snapshot" {
				continue
			}
			var got struct {
				Snapshot struct {
					Items []struct {
						ID        string `json:"id"`
						Connected bool   `json:"connected"`
					} `json:"items"`
				} `json:"snapshot"`
			}
			if err := json.Unmarshal([]byte(f.Data), &got); err != nil {
				t.Fatal(err)
			}
			for _, item := range got.Snapshot.Items {
				if item.ID == created.ID && item.Connected == connected {
					return
				}
			}
		}
	}
	untilDetail := func(frames <-chan sseFrame, connected bool) {
		t.Helper()
		for {
			f := nextFrame(t, frames)
			if f.Event != "snapshot" {
				continue
			}
			var got struct {
				Snapshot struct {
					ID        string `json:"id"`
					Connected bool   `json:"connected"`
				} `json:"snapshot"`
			}
			if err := json.Unmarshal([]byte(f.Data), &got); err != nil {
				t.Fatal(err)
			}
			if got.Snapshot.ID == created.ID && got.Snapshot.Connected == connected {
				return
			}
		}
	}
	open := func(path string) <-chan sseFrame {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		resp, err := admin.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		streams = append(streams, resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("stream %s: %d", path, resp.StatusCode)
		}
		return readFrames(resp)
	}
	list := open("/api/admin/workers/events")
	detail := open("/api/admin/workers/" + created.ID + "/events")
	untilList(list, false)
	untilDetail(detail, false)

	// The Worker connects: both pages learn it without reloading.
	call(anon, http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"fixture.*"}}, "", created.Token, http.StatusOK)
	untilList(list, true)
	untilDetail(detail, true)

	// The Worker goes silent: nothing is written, the server publishes the change itself.
	if _, err = pool.Exec(ctx, `UPDATE workers SET last_seen_at=now()-interval '31 seconds' WHERE id=$1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if err = server.Workers.PublishDisconnections(ctx); err != nil {
		t.Fatal(err)
	}
	untilList(list, false)
	untilDetail(detail, false)
}
