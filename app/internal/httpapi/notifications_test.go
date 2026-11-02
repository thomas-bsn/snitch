//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type notificationT struct {
	ID           int64  `json:"id"`
	Kind         string `json:"kind"`
	State        string `json:"state"`
	ClipID       int64  `json:"clip_id"`
	PendingCount int    `json:"pending_count"`
	Read         bool   `json:"read"`
}

type notificationsT struct {
	Items   []notificationT `json:"items"`
	Unread  int             `json:"unread"`
	Pending int             `json:"pending"`
	Total   int             `json:"total"`
}

func (h *voiceHarness) notifications(u *voiceUser, query string) notificationsT {
	h.t.Helper()
	var n notificationsT
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/notifications"+query, nil, http.StatusOK), &n); err != nil {
		h.t.Fatal(err)
	}
	return n
}

func testNotifications(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM notifications`); err != nil {
		t.Fatal(err)
	}
	h := newVoiceHarnessClips(t, pool, 2)
	if _, err := pool.Exec(ctx, `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clipA, clipB := h.clipIDs[0], h.clipIDs[1]
	admin := h.adminUser()
	alice := h.newUser("alice")

	// Nothing to decide yet: no notification.
	if n := h.notifications(admin, ""); len(n.Items) != 0 || n.Unread != 0 || n.Pending != 0 {
		t.Fatalf("no voice, no notification: %+v", n)
	}

	// Two unknown voices in clip A and one in clip B: one grouped notification per clip.
	a1 := h.fixtureSpeaker(clipA, "A1", usableAt(100))
	a2 := h.fixtureSpeaker(clipA, "A2", usableAt(200))
	b1 := h.fixtureSpeaker(clipB, "B1", usableAt(300))
	h.rematch()
	n := h.notifications(admin, "")
	if len(n.Items) != 2 || n.Unread != 2 || n.Pending != 2 {
		t.Fatalf("one notification per clip: %+v", n)
	}
	byClip := map[int64]notificationT{}
	for _, item := range n.Items {
		if item.Kind != "needs_speaker_review" || item.State != "active" || item.Read {
			t.Fatalf("active unread speaker review: %+v", item)
		}
		byClip[item.ClipID] = item
	}
	if byClip[clipA].PendingCount != 2 || byClip[clipB].PendingCount != 1 {
		t.Fatalf("voices are grouped per clip: %+v", byClip)
	}
	// A rematch refreshes the same notification, never duplicates it.
	h.rematch()
	if again := h.notifications(admin, ""); len(again.Items) != 2 {
		t.Fatalf("refreshing must not duplicate: %+v", again)
	}
	// A user who is neither administrator nor owner is not a recipient.
	if seen := h.notifications(alice, ""); len(seen.Items) != 0 {
		t.Fatalf("not a recipient: %+v", seen)
	}

	// Reading is individual and does not resolve the shared action.
	idA := byClip[clipA].ID
	h.userCall(admin, http.MethodPut, fmt.Sprintf("/api/notifications/%d/read", idA), nil, http.StatusOK)
	n = h.notifications(admin, "")
	if n.Unread != 1 || n.Pending != 2 {
		t.Fatalf("read is not resolved: %+v", n)
	}
	h.userCall(admin, http.MethodDelete, fmt.Sprintf("/api/notifications/%d/read", idA), nil, http.StatusOK)
	if n = h.notifications(admin, ""); n.Unread != 2 {
		t.Fatalf("mark unread: %+v", n)
	}
	h.userCall(admin, http.MethodPost, "/api/notifications/read", nil, http.StatusOK)
	if n = h.notifications(admin, ""); n.Unread != 0 || n.Pending != 2 {
		t.Fatalf("read all: %+v", n)
	}
	// An action still pending cannot be deleted.
	h.userCall(admin, http.MethodDelete, fmt.Sprintf("/api/notifications/%d", idA), nil, http.StatusConflict)
	h.userCall(admin, http.MethodDelete, "/api/notifications/999999", nil, http.StatusNotFound)
	if status, _ := h.userStatus(admin, http.MethodGet, "/api/notifications?state=bogus", nil); status != http.StatusBadRequest {
		t.Fatalf("invalid state must be refused, got %d", status)
	}

	// The Verification list is filterable on the clip a notification leads to.
	if only := h.verification(admin, fmt.Sprintf("?clip=%d", clipA)); only.Total != 2 {
		t.Fatalf("clip filter: %+v", only)
	}

	// Handling one voice refreshes the count; handling the last one resolves the action
	// for everybody, and the history stays (read state untouched).
	h.validate(a1, "Alice")
	if n = h.notifications(admin, "?state=active"); len(n.Items) != 2 {
		t.Fatalf("one voice left in clip A: %+v", n)
	}
	for _, item := range h.notifications(admin, "").Items {
		if item.ClipID == clipA && (item.PendingCount != 1 || item.State != "active") {
			t.Fatalf("clip A after one attribution: %+v", item)
		}
	}
	h.validate(a2, "Bob")
	n = h.notifications(admin, "")
	if n.Pending != 1 || len(n.Items) != 2 {
		t.Fatalf("clip A resolved, B still pending: %+v", n)
	}
	resolved := h.notifications(admin, "?state=resolved")
	if len(resolved.Items) != 1 || resolved.Items[0].ID != idA || resolved.Items[0].PendingCount != 0 || !resolved.Items[0].Read {
		t.Fatalf("resolved history: %+v", resolved)
	}
	// Active actions come first.
	if n.Items[0].ClipID != clipB {
		t.Fatalf("active actions before history: %+v", n.Items)
	}

	// Deleting history is individual: another recipient keeps it.
	if _, err := pool.Exec(ctx, `INSERT INTO notification_recipients(notification_id,user_id) SELECT $1,id FROM users WHERE username='alice'`, idA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE clips SET owner_id=(SELECT id FROM users WHERE username='alice') WHERE id=$1`, clipA); err != nil {
		t.Fatal(err)
	}
	h.userCall(admin, http.MethodDelete, fmt.Sprintf("/api/notifications/%d", idA), nil, http.StatusNoContent)
	if n = h.notifications(admin, ""); len(n.Items) != 1 {
		t.Fatalf("deleted from the caller's history: %+v", n)
	}
	h.userCall(admin, http.MethodDelete, fmt.Sprintf("/api/notifications/%d", idA), nil, http.StatusNotFound)
	if seen := h.notifications(alice, ""); len(seen.Items) != 1 || seen.Items[0].ID != idA {
		t.Fatalf("another recipient keeps the history: %+v", seen)
	}
	// Another user cannot read or delete a notification that is not theirs.
	bob := h.newUser("bob")
	h.userCall(bob, http.MethodPut, fmt.Sprintf("/api/notifications/%d/read", idA), nil, http.StatusNotFound)
	h.userCall(bob, http.MethodDelete, fmt.Sprintf("/api/notifications/%d", idA), nil, http.StatusNotFound)

	// Access is checked again on every read: a private clip of someone else disappears,
	// history included.
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private',owner_id=(SELECT id FROM users WHERE username='bob') WHERE id=$1`, clipA); err != nil {
		t.Fatal(err)
	}
	if seen := h.notifications(alice, ""); len(seen.Items) != 0 || seen.Unread != 0 {
		t.Fatalf("lost access, no leak: %+v", seen)
	}
	h.userCall(alice, http.MethodPut, fmt.Sprintf("/api/notifications/%d/read", idA), nil, http.StatusNotFound)

	// New voices reopen a fresh action and make the news unread again.
	h.fixtureSpeaker(clipB, "B2", usableAt(400))
	h.rematch()
	n = h.notifications(admin, "")
	if n.Unread != 1 || n.Pending != 1 || n.Items[0].PendingCount != 2 {
		t.Fatalf("more voices, unread again: %+v", n)
	}
	h.validate(b1, "Carol")
	if n = h.notifications(admin, ""); n.Items[0].PendingCount != 1 {
		t.Fatalf("count refreshed: %+v", n)
	}
	if got := h.count(`SELECT count(*) FROM notifications WHERE kind='needs_speaker_review' AND state='active' AND clip_id=$1`, clipB); got != 1 {
		t.Fatalf("at most one active notification per clip, got %d", got)
	}
}
