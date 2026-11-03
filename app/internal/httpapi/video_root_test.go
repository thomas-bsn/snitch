//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testVideoRootPrivacy covers #223 and ADR 0026 on the Racine vidéo: a private top-level folder keeps
// its name (with a lock) for everyone else, but its content is neither listed, previewed, scanned nor
// extended by anyone who is not the Super-administrator. Session is required for every route.
func testVideoRootPrivacy(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	root := t.TempDir()
	for _, dir := range []string{"secret", "open"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"secret/hidden.mp4", "open/shown.mp4", "top.mp4"} {
		if err := os.WriteFile(filepath.Join(root, file), []byte("synthetic-video"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	h.adminCall(http.MethodPut, "/api/admin/video-root", map[string]string{"path": root}, http.StatusOK)
	var sourceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clip_sources WHERE managed_root`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	superAdmin := h.adminUser()
	h.adminCall(http.MethodPut, "/api/admin/library/folders/privacy", map[string]any{"source_id": sourceID, "folder": "secret", "private": true}, http.StatusOK)

	member := h.newUser("root-member")
	admin := h.newUser("root-admin")
	if _, err := pool.Exec(ctx, `UPDATE users SET role='admin' WHERE username='root-admin'`); err != nil {
		t.Fatal(err)
	}
	anonymous := &voiceUser{name: "anonymous", client: &http.Client{Jar: mustJar(t)}}

	type entry struct {
		Path    string `json:"path"`
		Private bool   `json:"private"`
	}
	browse := func(u *voiceUser, path string) map[string]entry {
		t.Helper()
		var page struct {
			Items []entry `json:"items"`
		}
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/admin/video-root/browse?path="+path, nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		out := map[string]entry{}
		for _, item := range page.Items {
			out[item.Path] = item
		}
		return out
	}

	// Without a session, nothing of the root is listed.
	if status, _ := h.userStatus(anonymous, http.MethodGet, "/api/admin/video-root/browse?path=", nil); status != http.StatusUnauthorized {
		t.Fatalf("browsing the video root requires a session: %d", status)
	}

	for _, u := range []*voiceUser{member, admin} {
		top := browse(u, "")
		if _, ok := top["secret"]; !ok || !top["secret"].Private {
			t.Fatalf("%s: the private folder is listed with its lock: %+v", u.name, top)
		}
		if _, ok := top["open"]; !ok || top["open"].Private || top["top.mp4"].Path == "" {
			t.Fatalf("%s: other folders and files stay listed: %+v", u.name, top)
		}
		if _, ok := top["secret/hidden.mp4"]; ok {
			t.Fatalf("%s: the private folder content must not be listed", u.name)
		}
		h.userCall(u, http.MethodGet, "/api/admin/video-root/browse?path=secret", nil, http.StatusForbidden)
		h.userCall(u, http.MethodGet, "/api/admin/video-root/browse?path=secret/", nil, http.StatusForbidden)
		h.userCall(u, http.MethodPost, "/api/admin/video-root/folders", map[string]string{"parent": "secret", "name": "inner"}, http.StatusForbidden)
		if u == member {
			// Scanning is an administrator action: a regular member is refused before any preview.
			h.userCall(u, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{""}, "confirm": false}, http.StatusForbidden)
			continue
		}
		h.userCall(u, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{"secret"}, "confirm": false}, http.StatusForbidden)
		h.userCall(u, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{"secret/hidden.mp4"}, "confirm": false}, http.StatusForbidden)
		var preview struct {
			Paths []string `json:"paths"`
		}
		if err := json.Unmarshal(h.userCall(u, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{""}, "confirm": false}, http.StatusOK), &preview); err != nil {
			t.Fatal(err)
		}
		for _, p := range preview.Paths {
			if p == "secret/hidden.mp4" || p == "secret" {
				t.Fatalf("%s: a whole-root preview must skip the private folder: %v", u.name, preview.Paths)
			}
		}
	}

	// The Super-administrator browses and previews it as before.
	if inside := browse(superAdmin, "secret"); inside["secret/hidden.mp4"].Path == "" {
		t.Fatalf("the Super-administrator browses the private folder: %+v", inside)
	}
	if top := browse(superAdmin, ""); !top["secret"].Private {
		t.Fatalf("the Super-administrator still sees the lock: %+v", top)
	}
	h.userCall(superAdmin, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{"secret"}, "confirm": false}, http.StatusOK)

	// Once the folder is visible again, everyone browses it.
	h.adminCall(http.MethodPut, "/api/admin/library/folders/privacy", map[string]any{"source_id": sourceID, "folder": "secret", "private": false}, http.StatusOK)
	if inside := browse(admin, "secret"); inside["secret/hidden.mp4"].Path == "" {
		t.Fatalf("a visible folder is browsable by administrators: %+v", inside)
	}
}

func mustJar(t *testing.T) *cookiejar.Jar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return jar
}
