//go:build integration

package httpapi

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testSystemDirectories(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	h.api.Library.Close()
	root := t.TempDir()
	for _, rel := range []string{"games/clip.mp4", "private/clip.mp4", "#recycle/trash.mp4", "@eaDir/thumb.mp4", "games/#recycle/trash.mp4", "games/@eaDir/thumb.mp4"} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("synthetic "+rel), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range []string{"#recycle", "@eaDir", "games/#recycle", "games/@eaDir"} {
		dir := filepath.Join(root, rel)
		if err := os.Chmod(dir, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
	}
	h.adminCall(http.MethodPut, "/api/admin/video-root", map[string]string{"path": root}, http.StatusOK)
	var sourceID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM clip_sources WHERE managed_root`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, "/api/admin/library/folders/privacy", map[string]any{"source_id": sourceID, "folder": "private", "private": true}, http.StatusOK)
	for _, endpoint := range []string{
		"/api/admin/video-root/browse", "/api/admin/video-root/browse?path=games",
		"/api/library/explorer", "/api/library/explorer?path=games",
		"/api/library/explorer?recursive=1", "/api/library/explorer?q=clip",
		"/api/library/explorer?stage=indexed", "/api/admin/library/index-runs/preview",
	} {
		body := h.adminCall(http.MethodGet, endpoint, nil, http.StatusOK)
		if strings.Contains(string(body), "#recycle") || strings.Contains(string(body), "@eaDir") {
			t.Fatalf("%s lists a system directory: %s", endpoint, body)
		}
	}
	for _, endpoint := range []string{"/api/admin/video-root/scan", "/api/library/import/scan"} {
		for _, confirm := range []bool{false, true} {
			body := h.adminCall(http.MethodPost, endpoint, map[string]any{"paths": []string{""}, "confirm": confirm}, http.StatusOK)
			if strings.Contains(string(body), "#recycle") || strings.Contains(string(body), "@eaDir") || !strings.Contains(string(body), "private/clip.mp4") {
				t.Fatalf("scan must retain private videos and exclude system directories: %s", body)
			}
		}
	}
	h.adminCall(http.MethodPost, "/api/admin/library/index-runs", map[string]string{"mode": "cards"}, http.StatusCreated)
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM source_files WHERE source_id=$1 AND index_run_id IS NOT NULL`, sourceID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("index targets including private video: %d, %v", count, err)
	}
	for _, rel := range []string{"%23recycle", "%40eaDir", "games/%23recycle"} {
		h.adminCall(http.MethodGet, "/api/admin/video-root/browse?path="+rel, nil, http.StatusUnprocessableEntity)
		h.adminCall(http.MethodGet, "/api/library/explorer?path="+rel, nil, http.StatusUnprocessableEntity)
	}
	h.adminCall(http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{"#recycle/trash.mp4"}}, http.StatusUnprocessableEntity)

	// Ordinary permission failures must still abort the traversal and remain visible.
	dir := filepath.Join(root, "ordinary")
	if err := os.Mkdir(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("permission checks require an unprivileged user")
	}
	for _, endpoint := range []string{"/api/admin/library/index-runs/preview", "/api/library/explorer?recursive=1", "/api/library/explorer?q=clip", "/api/admin/video-root/browse?path=ordinary"} {
		h.adminCall(http.MethodGet, endpoint, nil, http.StatusUnprocessableEntity)
	}
	h.adminCall(http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{""}}, http.StatusUnprocessableEntity)
}
