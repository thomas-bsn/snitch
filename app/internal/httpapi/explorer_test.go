//go:build integration

package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testExplorer covers #195 (ADR 0032): the explorer lists the Racine vidéo, crosses it with the
// database to give each file its state, indexes a file only when it is opened, and applies the
// Dossiers privés rule (ADR 0026) to every path it returns. Fixtures are synthetic files.
func testExplorer(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	root := t.TempDir()
	for dir, files := range map[string][]string{
		"":             {"top.mp4", "notes.txt"},
		"league/2026":  {"match.mp4", "other.mp4"},
		"secret":       {"plan.mp4"},
		"league/empty": {"late.mp4"},
	} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
		for _, name := range files {
			if err := os.WriteFile(filepath.Join(root, dir, name), []byte("synthetic-video "+name), 0o600); err != nil {
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
	member := h.newUser("explorer-member")
	h.newUser("explorer-owner")
	var ownerID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='explorer-owner'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}

	// match.mp4 is already indexed and analysed by its owner; other.mp4 is not indexed yet.
	clipID := insertExplorerClip(t, pool, ownerID, "shared", "match.mp4", 1)
	if _, err := pool.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,clip_id,size_bytes,modified_at,last_seen_generation,status) VALUES($1,'league/2026/match.mp4',$2,10,now(),0,'ready')`, sourceID, clipID); err != nil {
		t.Fatal(err)
	}
	// A private clip of the owner, indexed under top.mp4: nobody else sees it.
	hiddenID := insertExplorerClip(t, pool, ownerID, "private", "top.mp4", 2)
	if _, err := pool.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,clip_id,size_bytes,modified_at,last_seen_generation,status) VALUES($1,'top.mp4',$2,10,now(),0,'ready')`, sourceID, hiddenID); err != nil {
		t.Fatal(err)
	}

	type entry struct {
		Name      string `json:"name"`
		Path      string `json:"path"`
		Directory bool   `json:"directory"`
		Private   bool   `json:"private"`
		Locked    bool   `json:"locked"`
		State     string `json:"state"`
		ClipID    *int64 `json:"clip_id"`
		Folder    string `json:"folder"`
	}
	list := func(u *voiceUser, query string, want int) map[string]entry {
		t.Helper()
		out := map[string]entry{}
		if want != http.StatusOK {
			h.userCall(u, http.MethodGet, "/api/library/explorer?"+query, nil, want)
			return out
		}
		var page struct {
			Items []entry `json:"items"`
		}
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/library/explorer?"+query, nil, want), &page); err != nil {
			t.Fatal(err)
		}
		for _, item := range page.Items {
			out[item.Path] = item
		}
		return out
	}

	t.Run("the root lists folders and every video with its state, private folders locked", func(t *testing.T) {
		got := list(member, "path=", http.StatusOK)
		if got["league"].Name == "" || !got["league"].Directory {
			t.Fatalf("league folder missing: %+v", got)
		}
		if _, ok := got["top.mp4"]; ok {
			t.Fatalf("a clip the member may not read leaks through the explorer: %+v", got["top.mp4"])
		}
		if _, ok := got["notes.txt"]; ok {
			t.Fatal("non-video files must not be listed")
		}
		if !got["secret"].Private || !got["secret"].Locked {
			t.Fatalf("private folder must show a lock for a member: %+v", got["secret"])
		}
		if _, ok := got["secret"]; !ok {
			t.Fatal("a private folder keeps its name for everyone")
		}
		for path := range got {
			if strings.HasPrefix(path, "secret/") {
				t.Fatalf("private folder contents leak: %s", path)
			}
		}
		admin := list(superAdmin, "path=", http.StatusOK)
		if admin["secret"].Locked {
			t.Fatal("the Super-administrator opens private folders")
		}
	})
	t.Run("a folder shows its videos, indexed ones carry their clip and state", func(t *testing.T) {
		got := list(member, "path=league%2F2026", http.StatusOK)
		if got["league/2026/match.mp4"].State != "not_analyzed" || got["league/2026/match.mp4"].ClipID == nil || *got["league/2026/match.mp4"].ClipID != clipID {
			t.Fatalf("indexed file state: %+v", got["league/2026/match.mp4"])
		}
		if got["league/2026/other.mp4"].State != "not_indexed" || got["league/2026/other.mp4"].ClipID != nil {
			t.Fatalf("unindexed file must be listed as not indexed: %+v", got["league/2026/other.mp4"])
		}
	})
	t.Run("a private folder cannot be opened by a member", func(t *testing.T) {
		list(member, "path=secret", http.StatusForbidden)
		list(member, "path=secret%2F..%2Fsecret", http.StatusForbidden)
	})
	t.Run("opening an unindexed video only requests its indexing, without analysis", func(t *testing.T) {
		h.userCall(member, http.MethodPost, "/api/library/explorer/index", map[string]string{"path": "league/2026/other.mp4"}, http.StatusAccepted)
		var indexOnly, requested bool
		if err := pool.QueryRow(ctx, `SELECT index_only,scan_requested FROM source_files WHERE source_id=$1 AND relative_path='league/2026/other.mp4'`, sourceID).Scan(&indexOnly, &requested); err != nil {
			t.Fatal(err)
		}
		if !indexOnly || !requested {
			t.Fatalf("index request: index_only=%v scan_requested=%v", indexOnly, requested)
		}
		h.userCall(member, http.MethodPost, "/api/library/explorer/index", map[string]string{"path": "secret/plan.mp4"}, http.StatusForbidden)
		h.userCall(member, http.MethodPost, "/api/library/explorer/index", map[string]string{"path": "../outside.mp4"}, http.StatusUnprocessableEntity)
	})
	t.Run("a clip without a file in the root stays visible in all clips, unless the viewer may not read it", func(t *testing.T) {
		// An upload that is not published in the root: its clip lives outside the explorer's folders.
		looseID := insertExplorerClip(t, pool, ownerID, "shared", "upload.mp4", 3)
		hiddenLooseID := insertExplorerClip(t, pool, ownerID, "private", "upload-private.mp4", 4)
		got := list(member, "path=", http.StatusOK)
		entry, ok := got[fmt.Sprintf("clip:%d", looseID)]
		if !ok || entry.ClipID == nil || *entry.ClipID != looseID || entry.Name != "upload.mp4" {
			t.Fatalf("loose clip missing from all clips: %+v", got)
		}
		if _, ok := got[fmt.Sprintf("clip:%d", hiddenLooseID)]; ok {
			t.Fatal("a private clip of another user leaks through all clips")
		}
		if _, ok := list(member, "path=league", http.StatusOK)[fmt.Sprintf("clip:%d", looseID)]; ok {
			t.Fatal("loose clips belong to all clips only")
		}
	})
	t.Run("a filtered search is flat, carries the folder, and filters by state", func(t *testing.T) {
		got := list(member, "path=league&recursive=1&state=not_indexed", http.StatusOK)
		if len(got) != 1 || got["league/empty/late.mp4"].Folder != "league/empty" {
			t.Fatalf("flat unindexed results: %+v", got)
		}
		// other.mp4 was requested by the test above: it is in progress, not unindexed anymore.
		got = list(member, "path=league&recursive=1&state=in_progress", http.StatusOK)
		if len(got) != 1 || got["league/2026/other.mp4"].Folder != "league/2026" {
			t.Fatalf("flat in-progress results: %+v", got)
		}
		got = list(member, "path=&recursive=1&q=MATCH", http.StatusOK)
		if _, ok := got["league/2026/match.mp4"]; !ok || len(got) != 1 {
			t.Fatalf("name search: %+v", got)
		}
	})
}

// insertExplorerClip creates an analysable clip row owned by ownerID with a synthetic content hash.
func insertExplorerClip(t *testing.T, pool *pgxpool.Pool, ownerID int64, visibility, name string, seed byte) int64 {
	t.Helper()
	hash := sha256.Sum256([]byte(fmt.Sprintf("explorer-%s-%d", name, seed)))
	var id int64
	err := pool.QueryRow(context.Background(), `INSERT INTO clips(owner_id,visibility,original_name,size_bytes,modified_at,recorded_at,recorded_at_estimated,duration_seconds,format_name,video_codec,audio_codec,tracks,content_sha256) VALUES($1,$2,$3,10,now(),now(),true,12.5,'mp4','h264','aac','[]'::jsonb,$4) RETURNING id`, ownerID, visibility, name, hash[:]).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
