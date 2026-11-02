//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type notificationFrame struct {
	Seq      int64          `json:"seq"`
	Snapshot notificationsT `json:"snapshot"`
}

func (h *voiceHarness) notificationStream(u *voiceUser) (*http.Response, <-chan sseFrame) {
	h.t.Helper()
	req, _ := http.NewRequest(http.MethodGet, h.ts.URL+"/api/notifications/events", nil)
	resp, err := u.client.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		h.t.Fatalf("stream: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	return resp, readFrames(resp)
}

// untilSnapshot reads snapshots until one satisfies ok, and returns it.
func untilSnapshot(t *testing.T, frames <-chan sseFrame, ok func(notificationsT) bool) notificationFrame {
	t.Helper()
	for {
		f := nextFrame(t, frames)
		if f.Event != "snapshot" {
			continue
		}
		var got notificationFrame
		if err := json.Unmarshal([]byte(f.Data), &got); err != nil {
			t.Fatal(err)
		}
		if ok(got.Snapshot) {
			return got
		}
	}
}

func testNotificationEvents(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM notifications`); err != nil {
		t.Fatal(err)
	}
	h := newVoiceHarnessClips(t, pool, 1)
	h.api.Events.Poll, h.api.Events.Heartbeat = 50*time.Millisecond, 300*time.Millisecond
	if _, err := pool.Exec(ctx, `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clip := h.clipIDs[0]
	admin := h.adminUser()
	alice := h.newUser("alice")

	anon, err := http.Get(h.ts.URL + "/api/notifications/events")
	if err != nil {
		t.Fatal(err)
	}
	anon.Body.Close()
	if anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous stream: %d", anon.StatusCode)
	}

	resp, frames := h.notificationStream(admin)
	defer resp.Body.Close()
	first := untilSnapshot(t, frames, func(notificationsT) bool { return true })
	if len(first.Snapshot.Items) != 0 || first.Snapshot.Unread != 0 || first.Snapshot.Pending != 0 {
		t.Fatalf("nothing yet: %+v", first)
	}

	// A new action is pushed, with distinct counters.
	voice := h.fixtureSpeaker(clip, "A1", usableAt(100))
	h.rematch()
	raised := untilSnapshot(t, frames, func(n notificationsT) bool { return n.Pending == 1 })
	if raised.Snapshot.Unread != 1 || len(raised.Snapshot.Items) != 1 || raised.Seq <= first.Seq {
		t.Fatalf("pushed action: %+v", raised)
	}
	// Reading is pushed too and leaves the action open.
	h.userCall(admin, http.MethodPost, "/api/notifications/read", nil, http.StatusOK)
	read := untilSnapshot(t, frames, func(n notificationsT) bool { return n.Unread == 0 })
	if read.Snapshot.Pending != 1 {
		t.Fatalf("read does not resolve: %+v", read)
	}
	// Handling the voice resolves it for the stream, history kept.
	h.validate(voice, "Alice")
	done := untilSnapshot(t, frames, func(n notificationsT) bool { return n.Pending == 0 })
	if len(done.Snapshot.Items) != 1 || done.Snapshot.Items[0].State != "resolved" {
		t.Fatalf("resolved in history: %+v", done)
	}
	// Deleting the resolved notification is pushed.
	h.userCall(admin, http.MethodDelete, fmt.Sprintf("/api/notifications/%d", done.Snapshot.Items[0].ID), nil, http.StatusNoContent)
	untilSnapshot(t, frames, func(n notificationsT) bool { return len(n.Items) == 0 })

	// A recipient who cannot see the clip gets nothing on connection, whatever the events say.
	h.fixtureSpeaker(clip, "A2", usableAt(200))
	h.rematch()
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private',owner_id=(SELECT id FROM users WHERE role IN ('admin','super_admin') ORDER BY id LIMIT 1) WHERE id=$1`, clip); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO notification_recipients(notification_id,user_id) SELECT n.id,u.id FROM notifications n,users u WHERE u.username='alice' AND n.clip_id=$1 AND n.state='active' ON CONFLICT DO NOTHING`, clip); err != nil {
		t.Fatal(err)
	}
	respAlice, aliceFrames := h.notificationStream(alice)
	defer respAlice.Body.Close()
	if seen := untilSnapshot(t, aliceFrames, func(notificationsT) bool { return true }); len(seen.Snapshot.Items) != 0 || seen.Snapshot.Unread != 0 || seen.Snapshot.Pending != 0 {
		t.Fatalf("access rechecked at connection: %+v", seen)
	}

	// Losing the session ends the stream with a notice.
	if _, err := pool.Exec(ctx, `DELETE FROM sessions`); err != nil {
		t.Fatal(err)
	}
	for {
		f, ok := <-frames
		if !ok {
			t.Fatal("the stream ended without telling the viewer")
		}
		if f.Event == "forbidden" {
			break
		}
	}
}
