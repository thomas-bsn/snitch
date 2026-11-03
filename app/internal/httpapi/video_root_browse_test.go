//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testVideoRootBrowse covers #223: the Racine vidéo is browsed only with a session, and a Dossier privé
// (ADR 0026) shows its name and a lock to anyone who may not open it, never its content. The same rule
// applies to the scan endpoint that the import selector relies on.
func testVideoRootBrowse(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	root := t.TempDir()
	for dir, files := range map[string][]string{"": {"top.mp4"}, "secret": {"plan.mp4"}, "open": {"visible.mp4"}} {
		if dir != "" {
			if err := os.Mkdir(filepath.Join(root, dir), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		for _, name := range files {
			if err := os.WriteFile(filepath.Join(root, dir, name), []byte("synthetic-video"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	superAdmin := h.adminUser()
	h.adminCall(http.MethodPut, "/api/admin/video-root", map[string]string{"path": root}, http.StatusOK)
	var sourceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clip_sources WHERE managed_root`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	h.userCall(superAdmin, http.MethodPut, "/api/admin/library/folders/privacy", map[string]any{"source_id": sourceID, "folder": "secret", "private": true}, http.StatusOK)

	member := h.newUser("video-member")
	admin := h.newUser("video-admin")
	if _, err := pool.Exec(ctx, `UPDATE users SET role='admin' WHERE username='video-admin'`); err != nil {
		t.Fatal(err)
	}
	anonymous := &voiceUser{name: "anonymous", client: &http.Client{}}

	type entry struct {
		Path      string `json:"path"`
		Directory bool   `json:"directory"`
		Private   bool   `json:"private"`
		Locked    bool   `json:"locked"`
	}
	browse := func(u *voiceUser, path string, want int) []entry {
		t.Helper()
		var page struct {
			Items []entry `json:"items"`
		}
		data := h.userCall(u, http.MethodGet, "/api/admin/video-root/browse?path="+path, nil, want)
		if want == http.StatusOK {
			if err := json.Unmarshal(data, &page); err != nil {
				t.Fatal(err)
			}
		}
		return page.Items
	}
	paths := func(items []entry) map[string]entry {
		out := map[string]entry{}
		for _, item := range items {
			out[item.Path] = item
		}
		return out
	}

	// A session is required: an anonymous request lists nothing.
	if status, _ := h.userStatus(anonymous, http.MethodGet, "/api/admin/video-root/browse?path=", nil); status != http.StatusUnauthorized {
		t.Fatalf("the video root listed without a session: %d", status)
	}

	for _, u := range []*voiceUser{member, admin} {
		items := paths(browse(u, "", http.StatusOK))
		if _, ok := items["top.mp4"]; !ok {
			t.Fatalf("%s: a visible file of the root is missing: %+v", u.name, items)
		}
		if secret, ok := items["secret"]; !ok || !secret.Directory || !secret.Private || !secret.Locked {
			t.Fatalf("%s: a private folder must show its name and a lock: %+v", u.name, items)
		}
		if _, ok := items["secret/plan.mp4"]; ok {
			t.Fatalf("%s: a private folder lists its content at the root: %+v", u.name, items)
		}
		if open := items["open"]; open.Private || open.Locked {
			t.Fatalf("%s: an open folder is marked private: %+v", u.name, open)
		}
		browse(u, "secret", http.StatusForbidden)
		browse(u, "secret/plan.mp4", http.StatusForbidden)
		if visible := paths(browse(u, "open", http.StatusOK)); len(visible) != 1 || visible["open/visible.mp4"].Path == "" {
			t.Fatalf("%s: an open folder must still list its files: %+v", u.name, visible)
		}
	}

	// The Super-administrator browses the private folder, shown open.
	if inside := paths(browse(superAdmin, "secret", http.StatusOK)); inside["secret/plan.mp4"].Path == "" {
		t.Fatalf("the Super-administrator cannot browse the private folder: %+v", inside)
	}
	if listing := paths(browse(superAdmin, "", http.StatusOK)); listing["secret"].Locked {
		t.Fatalf("the Super-administrator sees a lock on the private folder: %+v", listing["secret"])
	}

	// The scan endpoint behind the import selector obeys the same rule for administrators.
	scan := func(u *voiceUser, paths []string, want int) []byte {
		t.Helper()
		return h.userCall(u, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": paths, "confirm": false}, want)
	}
	scan(admin, []string{"secret"}, http.StatusForbidden)
	scan(admin, []string{"secret/plan.mp4"}, http.StatusForbidden)
	if data := scan(admin, []string{""}, http.StatusOK); strings.Contains(string(data), "plan.mp4") {
		t.Fatalf("a whole-root scan by an administrator reveals a private folder: %s", data)
	}
	if data := scan(superAdmin, []string{"secret"}, http.StatusOK); !strings.Contains(string(data), "secret/plan.mp4") {
		t.Fatalf("the Super-administrator scan misses the private folder: %s", data)
	}
}
