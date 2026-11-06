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

// testImportSelection covers #224: the « Déjà sur le serveur » picker lets every signed-in user select
// videos of the Racine vidéo, folders included (recursively), and never reveals or queues the content of
// a Dossier privé (ADR 0026) unless the viewer is the Super-administrator. Upload destinations obey the
// same rule. The synthetic tree lives in a temporary root.
func testImportSelection(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	root := t.TempDir()
	for _, dir := range []string{"secret", "open", "open/deep"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"top.mp4", "secret/plan.mp4", "open/visible.mp4", "open/deep/nested.mp4", "open/notes.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("synthetic-video"), 0o600); err != nil {
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

	member := h.newUser("import-member")
	admin := h.newUser("import-admin")
	if _, err := pool.Exec(ctx, `UPDATE users SET role='admin' WHERE username='import-admin'`); err != nil {
		t.Fatal(err)
	}
	anonymous := &voiceUser{name: "anonymous", client: &http.Client{}}

	type preview struct {
		Paths  []string `json:"paths"`
		Queued bool     `json:"queued"`
	}
	scan := func(u *voiceUser, paths []string, confirm bool, want int) []string {
		t.Helper()
		data := h.userCall(u, http.MethodPost, "/api/library/import/scan", map[string]any{"paths": paths, "confirm": confirm}, want)
		if want != http.StatusOK {
			return nil
		}
		var out preview
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
		return out.Paths
	}
	has := func(list []string, name string) bool {
		for _, p := range list {
			if p == name {
				return true
			}
		}
		return false
	}

	// A session is required.
	if status, _ := h.userStatus(anonymous, http.MethodPost, "/api/library/import/scan", map[string]any{"paths": []string{""}, "confirm": false}); status != http.StatusUnauthorized {
		t.Fatalf("the import picker answered without a session: %d", status)
	}

	// A signed-in member selects a folder: its videos come recursively, its other files do not.
	for _, u := range []*voiceUser{member, admin} {
		got := scan(u, []string{"open"}, false, http.StatusOK)
		if !has(got, "open/visible.mp4") || !has(got, "open/deep/nested.mp4") || len(got) != 2 {
			t.Fatalf("%s: a folder selection is recursive and keeps videos only: %v", u.name, got)
		}
		whole := scan(u, []string{""}, false, http.StatusOK)
		if has(whole, "secret/plan.mp4") || !has(whole, "top.mp4") || !has(whole, "open/deep/nested.mp4") {
			t.Fatalf("%s: a root selection skips the private folder: %v", u.name, whole)
		}
		// Negative: nothing in a private folder can be picked, previewed or queued.
		scan(u, []string{"secret"}, false, http.StatusForbidden)
		scan(u, []string{"secret/plan.mp4"}, true, http.StatusForbidden)
		scan(u, []string{"open", "secret"}, true, http.StatusForbidden)
	}

	// Queuing a root selection never queues the private content.
	scan(member, []string{""}, true, http.StatusOK)
	var queued int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE source_id=$1 AND relative_path LIKE 'secret/%'`, sourceID).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 0 {
		t.Fatalf("a member's root import queued %d files of a private folder", queued)
	}

	// The Super-administrator selects the private folder as before.
	if got := scan(superAdmin, []string{"secret"}, false, http.StatusOK); !has(got, "secret/plan.mp4") {
		t.Fatalf("the Super-administrator cannot select the private folder: %v", got)
	}

	// The administration scan stays administrator-only for the members.
	h.userCall(member, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{"open"}, "confirm": false}, http.StatusForbidden)

	// An upload destination inside a private folder is refused to members.
	h.userCall(member, http.MethodPost, "/api/uploads", map[string]any{"filename": "new.mp4", "size_bytes": 10, "visibility": "shared", "destination": "secret"}, http.StatusForbidden)
	h.userCall(member, http.MethodPost, "/api/uploads", map[string]any{"filename": "new.mp4", "size_bytes": 10, "visibility": "shared", "destination": "secret/deep"}, http.StatusForbidden)
	if strings.Contains(string(h.userCall(member, http.MethodGet, "/api/admin/video-root/browse?path=", nil, http.StatusOK)), "plan.mp4") {
		t.Fatal("the root listing reveals the content of a private folder")
	}
	// #223: the private folder keeps its name and shows locked, but nobody else lists or previews it.
	for _, u := range []*voiceUser{member, admin} {
		var root struct {
			Items []struct {
				Path   string `json:"path"`
				Locked bool   `json:"locked"`
			} `json:"items"`
		}
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/admin/video-root/browse?path=", nil, http.StatusOK), &root); err != nil {
			t.Fatal(err)
		}
		locked := map[string]bool{}
		for _, item := range root.Items {
			locked[item.Path] = item.Locked
		}
		if l, ok := locked["secret"]; !ok || !l || locked["open"] {
			t.Fatalf("%s: the private folder must be listed and locked, the open one not: %v", u.name, locked)
		}
		h.userCall(u, http.MethodGet, "/api/admin/video-root/browse?path=secret", nil, http.StatusForbidden)
		scan(u, []string{"secret/plan.mp4"}, false, http.StatusForbidden)
	}
	h.userCall(superAdmin, http.MethodGet, "/api/admin/video-root/browse?path=secret", nil, http.StatusOK)
}
