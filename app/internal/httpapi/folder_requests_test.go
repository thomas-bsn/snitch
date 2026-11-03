//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// testFolderRequests covers #225 (ADR 0033): a member asks for a folder in the Racine vidéo, an
// administrator creates it directly, and an administrator approves or refuses a member's request. The
// requester is told of the decision, nothing can be uploaded into a pending request, and the trail is
// immutable.
func testFolderRequests(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	root := t.TempDir()
	for _, dir := range []string{"secret", "open"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	superAdmin := h.adminUser()
	h.adminCall(http.MethodPut, "/api/admin/video-root", map[string]string{"path": root}, http.StatusOK)
	var sourceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clip_sources WHERE managed_root`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	h.userCall(superAdmin, http.MethodPut, "/api/admin/library/folders/privacy", map[string]any{"source_id": sourceID, "folder": "secret", "private": true}, http.StatusOK)
	member := h.newUser("request-member")
	admin := h.newUser("request-admin")
	if _, err := pool.Exec(ctx, `UPDATE users SET role='admin' WHERE username='request-admin'`); err != nil {
		t.Fatal(err)
	}
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(root, rel))
		return err == nil
	}
	type requestView struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
		Path   string `json:"path"`
	}
	decode := func(data []byte) requestView {
		t.Helper()
		var v requestView
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	notified := func(u *voiceUser) []map[string]any {
		t.Helper()
		var page struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/notifications", nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		var out []map[string]any
		for _, item := range page.Items {
			if item["kind"] == "folder_request_decided" {
				out = append(out, item)
			}
		}
		return out
	}

	// Members: a new folder waits for an administrator; nothing is created yet.
	first := decode(h.userCall(member, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "open", "name": "Parties"}, http.StatusAccepted))
	if first.Status != "pending" || exists("open/Parties") {
		t.Fatalf("a member's folder is only requested: %+v", first)
	}
	h.userCall(member, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "open", "name": "Parties"}, http.StatusConflict)
	h.userCall(member, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "", "name": "open"}, http.StatusConflict)
	h.userCall(member, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "secret", "name": "Sub"}, http.StatusForbidden)
	h.userCall(member, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "open", "name": "../escape"}, http.StatusBadRequest)

	// Uploading into the pending folder is refused, with or without the pending name's sub-folder.
	h.userCall(member, http.MethodPost, "/api/uploads", map[string]any{"filename": "x.mp4", "size_bytes": 10, "visibility": "shared", "destination": "open/Parties"}, http.StatusConflict)
	h.userCall(member, http.MethodPost, "/api/uploads", map[string]any{"filename": "x.mp4", "size_bytes": 10, "visibility": "shared", "destination": "open/Parties/Sub"}, http.StatusConflict)

	// Only administrators list the requests and decide them.
	h.userCall(member, http.MethodGet, "/api/admin/folder-requests", nil, http.StatusForbidden)
	h.userCall(member, http.MethodPost, "/api/admin/folder-requests/1/approve", nil, http.StatusForbidden)
	var listing struct {
		Items []requestView `json:"items"`
	}
	if err := json.Unmarshal(h.userCall(admin, http.MethodGet, "/api/admin/folder-requests", nil, http.StatusOK), &listing); err != nil {
		t.Fatal(err)
	}
	if len(listing.Items) != 1 || listing.Items[0].ID != first.ID {
		t.Fatalf("the pending request is listed for administrators: %+v", listing.Items)
	}
	h.userCall(admin, http.MethodPost, "/api/admin/folder-requests/999999/approve", nil, http.StatusNotFound)

	// Approval creates the folder and notifies the requester.
	approved := decode(h.userCall(admin, http.MethodPost, "/api/admin/folder-requests/"+itoa(first.ID)+"/approve", nil, http.StatusOK))
	if approved.Status != "approved" || !exists("open/Parties") {
		t.Fatalf("an approved request creates the folder: %+v", approved)
	}
	h.userCall(admin, http.MethodPost, "/api/admin/folder-requests/"+itoa(first.ID)+"/approve", nil, http.StatusConflict)
	h.userCall(admin, http.MethodPost, "/api/admin/folder-requests/"+itoa(first.ID)+"/refuse", nil, http.StatusConflict)
	if got := notified(member); len(got) != 1 || got[0]["folder_outcome"] != "approved" {
		t.Fatalf("the requester is told the request was approved: %v", got)
	}
	if got := notified(admin); len(got) != 0 {
		t.Fatalf("the administrator is not notified of its own decision: %v", got)
	}

	// Refusal creates nothing and notifies the requester.
	refused := decode(h.userCall(member, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "", "name": "Archives"}, http.StatusAccepted))
	if decided := decode(h.userCall(admin, http.MethodPost, "/api/admin/folder-requests/"+itoa(refused.ID)+"/refuse", nil, http.StatusOK)); decided.Status != "refused" || exists("Archives") {
		t.Fatalf("a refused request creates nothing: %+v", decided)
	}
	if got := notified(member); len(got) != 2 || got[0]["folder_outcome"] != "refused" {
		t.Fatalf("the requester is told the request was refused: %v", got)
	}
	// Once refused, the same name can be asked again.
	h.userCall(member, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "", "name": "Archives"}, http.StatusAccepted)

	// Administrators create directly; a private folder still refuses them.
	h.userCall(admin, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "open", "name": "Direct"}, http.StatusCreated)
	if !exists("open/Direct") {
		t.Fatal("an administrator's folder is created at once")
	}
	h.userCall(admin, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "secret", "name": "Sub"}, http.StatusForbidden)
	h.userCall(superAdmin, http.MethodPost, "/api/folder-requests", map[string]any{"parent": "secret", "name": "Sub"}, http.StatusCreated)

	// The trail keeps every request and decision; it cannot be rewritten.
	var events int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM folder_request_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 5 {
		t.Fatalf("the trail records each request and each decision, and no direct creation: %d", events)
	}
	if _, err := pool.Exec(ctx, `UPDATE folder_request_events SET actor_username='nobody'`); err == nil {
		t.Fatal("the trail accepts rewrites")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM folder_request_events`); err == nil {
		t.Fatal("the trail accepts deletions")
	}
}
