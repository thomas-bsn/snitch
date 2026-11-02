//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPrivateFolders covers #93: a top-level folder the Super-administrator makes private keeps its
// name (with a lock) for everyone else, but none of its clips, in any listing or by direct address.
// A clip also present in another visible folder stays reachable; its owner always keeps it.
func testPrivateFolders(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	ctx := context.Background()
	hidden, copied, open := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]

	// Lay the synthetic files out in folders: hidden only in "secret", copied in "secret" and in
	// "open" (same content, so the same clip), open only in "open".
	var sourceID int64
	var root string
	if err := pool.QueryRow(ctx, `SELECT id,root_path FROM clip_sources ORDER BY id LIMIT 1`).Scan(&sourceID, &root); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"secret", "open"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	nameOf := func(clipID int64) string {
		var name string
		if err := pool.QueryRow(ctx, `SELECT relative_path FROM source_files WHERE clip_id=$1`, clipID).Scan(&name); err != nil {
			t.Fatal(err)
		}
		return name
	}
	hiddenName, copiedName, openName := nameOf(hidden), nameOf(copied), nameOf(open)
	content, err := os.ReadFile(filepath.Join(root, copiedName))
	if err != nil {
		t.Fatal(err)
	}
	for from, to := range map[string]string{hiddenName: "secret/" + hiddenName, copiedName: "secret/" + copiedName, openName: "open/" + openName} {
		if err = os.Rename(filepath.Join(root, from), filepath.Join(root, to)); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.WriteFile(filepath.Join(root, "open", "copy.mp4"), content, 0o400); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/sources/%d/scan", sourceID), map[string]any{}, http.StatusAccepted)
	want := map[string]int64{"secret/" + hiddenName: hidden, "secret/" + copiedName: copied, "open/" + openName: open, "open/copy.mp4": copied}
	deadline := time.Now().Add(10 * time.Second)
	for {
		var ready int
		for path, clipID := range want {
			var ok bool
			_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM source_files WHERE source_id=$1 AND relative_path=$2 AND clip_id=$3 AND status='ready')`, sourceID, path, clipID).Scan(&ok)
			if ok {
				ready++
			}
		}
		if ready == len(want) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the synthetic folders were not indexed")
		}
		h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/sources/%d/scan", sourceID), map[string]any{}, http.StatusAccepted)
		time.Sleep(50 * time.Millisecond)
	}

	member := h.newUser("folder-member")
	admin := h.newUser("folder-admin")
	if _, err = pool.Exec(ctx, `UPDATE users SET role='admin' WHERE username='folder-admin'`); err != nil {
		t.Fatal(err)
	}
	superAdmin := h.adminUser()

	type clipPage struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	listed := func(u *voiceUser, query string) map[int64]bool {
		t.Helper()
		var page clipPage
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/clips?page_size=100"+query, nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		seen := map[int64]bool{}
		for _, item := range page.Items {
			seen[item.ID] = true
		}
		if page.Total != len(seen) {
			t.Fatalf("the count must match the visible clips: total %d for %v", page.Total, seen)
		}
		return seen
	}
	type folderView struct {
		SourceID int64  `json:"source_id"`
		Name     string `json:"name"`
		Private  bool   `json:"private"`
		Locked   bool   `json:"locked"`
	}
	folders := func(u *voiceUser) map[string]folderView {
		t.Helper()
		var page struct {
			Items []folderView `json:"items"`
		}
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/library/folders", nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		out := map[string]folderView{}
		for _, f := range page.Items {
			out[f.Name] = f
		}
		return out
	}
	privacy := func(u *voiceUser, body map[string]any, want int) {
		t.Helper()
		h.userCall(u, http.MethodPut, "/api/admin/library/folders/privacy", body, want)
	}

	if seen := listed(member, ""); !seen[hidden] || !seen[copied] || !seen[open] {
		t.Fatalf("before any private folder, the member sees every shared clip: %v", seen)
	}

	// Only the Super-administrator may change the setting, and only for a real top-level folder.
	secret := map[string]any{"source_id": sourceID, "folder": "secret", "private": true}
	privacy(member, secret, http.StatusForbidden)
	privacy(admin, secret, http.StatusForbidden)
	privacy(superAdmin, map[string]any{"source_id": sourceID, "folder": "secret/deeper", "private": true}, http.StatusBadRequest)
	privacy(superAdmin, map[string]any{"source_id": sourceID, "folder": "..", "private": true}, http.StatusBadRequest)
	privacy(superAdmin, map[string]any{"source_id": sourceID, "folder": "secret"}, http.StatusBadRequest)
	privacy(superAdmin, map[string]any{"source_id": 999999, "folder": "secret", "private": true}, http.StatusNotFound)
	privacy(superAdmin, secret, http.StatusOK)
	privacy(superAdmin, secret, http.StatusOK) // idempotent

	// Everyone else: the folder is listed, locked, and nothing of its own content leaks.
	for _, u := range []*voiceUser{member, admin} {
		seen := listed(u, "")
		if seen[hidden] || !seen[copied] || !seen[open] {
			t.Fatalf("%s: a private folder hides only the clips it alone provides: %v", u.name, seen)
		}
		if inSecret := listed(u, "&folder=secret"); len(inSecret) != 0 {
			t.Fatalf("%s: browsing a private folder must reveal none of its clips: %v", u.name, inSecret)
		}
		if f := folders(u); !f["secret"].Private || !f["secret"].Locked || f["open"].Private || f["open"].Locked {
			t.Fatalf("%s: the private folder must show locked, the other one open: %+v", u.name, f)
		}
		for _, path := range []string{"", "/media", "/voice", "/analysis"} {
			if status, _ := h.userStatus(u, http.MethodGet, fmt.Sprintf("/api/clips/%d%s", hidden, path), nil); status != http.StatusNotFound {
				t.Fatalf("%s: knowing the address of a hidden clip must not open %q: %d", u.name, path, status)
			}
		}
		h.userCall(u, http.MethodGet, fmt.Sprintf("/api/clips/%d", copied), nil, http.StatusOK)
		h.userCall(u, http.MethodGet, "/api/clip-filters", nil, http.StatusOK)
	}

	// The Super-administrator browses it as before.
	if seen := listed(superAdmin, "&folder=secret"); !seen[hidden] || !seen[copied] || seen[open] {
		t.Fatalf("the Super-administrator browses the private folder: %v", seen)
	}
	if f := folders(superAdmin); !f["secret"].Private || f["secret"].Locked {
		t.Fatalf("the private folder is marked but open for the Super-administrator: %+v", f)
	}
	h.userCall(superAdmin, http.MethodGet, fmt.Sprintf("/api/clips/%d", hidden), nil, http.StatusOK)

	// The owner of a clip always keeps it, even in a private folder.
	if _, err = pool.Exec(ctx, `UPDATE clips SET owner_id=(SELECT id FROM users WHERE username='folder-member') WHERE id=$1`, hidden); err != nil {
		t.Fatal(err)
	}
	h.userCall(member, http.MethodGet, fmt.Sprintf("/api/clips/%d", hidden), nil, http.StatusOK)
	h.userCall(admin, http.MethodGet, fmt.Sprintf("/api/clips/%d", hidden), nil, http.StatusNotFound)

	// Making the folder visible again restores every clip, according to its own visibility.
	privacy(superAdmin, map[string]any{"source_id": sourceID, "folder": "secret", "private": false}, http.StatusOK)
	if seen := listed(admin, ""); !seen[hidden] || !seen[copied] || !seen[open] {
		t.Fatalf("a visible folder hides nothing: %v", seen)
	}
	if f := folders(member); f["secret"].Private || f["secret"].Locked {
		t.Fatalf("the folder is no longer private: %+v", f)
	}
	if strings.Contains(string(h.userCall(member, http.MethodGet, "/api/library/folders", nil, http.StatusOK)), `"locked":true`) {
		t.Fatal("no folder is locked any more")
	}
}
