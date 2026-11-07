//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testExplorerProperties covers the size, duration, owners and dates of a folder or a video, with the
// same Dossiers privés and clip visibility rules as the listing. Fixtures are synthetic files.
func testExplorerProperties(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	root := t.TempDir()
	files := map[string]string{
		"league/match.mp4":      "match!",
		"league/other.mp4":      "other--",
		"league/2026/inner.mp4": "inner-video",
		"top.mp4":               "top!",
		"secret/plan.mp4":       "plan--",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	h.adminCall(http.MethodPut, "/api/admin/video-root", map[string]string{"path": root}, http.StatusOK)
	var sourceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clip_sources WHERE managed_root`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, "/api/admin/library/folders/privacy", map[string]any{"source_id": sourceID, "folder": "secret", "private": true}, http.StatusOK)

	h.newUser("explorer-owner-a")
	h.newUser("explorer-owner-b")
	member := h.newUser("explorer-properties-member")
	var ownerAID, ownerBID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='explorer-owner-a'`).Scan(&ownerAID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='explorer-owner-b'`).Scan(&ownerBID); err != nil {
		t.Fatal(err)
	}
	type fixture struct {
		path, name string
		owner      int64
		visibility string
		duration   float64
		discovered time.Time
	}
	fixtures := []fixture{
		{"league/match.mp4", "match.mp4", ownerAID, "shared", 60, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)},
		{"league/2026/inner.mp4", "inner.mp4", ownerBID, "shared", 120, time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)},
		{"top.mp4", "top.mp4", ownerBID, "private", 30, time.Date(2026, 1, 4, 10, 0, 0, 0, time.UTC)},
		{"secret/plan.mp4", "plan.mp4", ownerAID, "shared", 15, time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)},
	}
	clips := map[string]int64{}
	for i, f := range fixtures {
		id := insertExplorerClip(t, pool, f.owner, f.visibility, f.name, byte(i+1))
		if _, err := pool.Exec(ctx, `UPDATE clips SET duration_seconds=$2,discovered_at=$3 WHERE id=$1`, id, f.duration, f.discovered); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,clip_id,size_bytes,modified_at,last_seen_generation,status) VALUES($1,$2,$3,$4,now(),0,'ready')`, sourceID, f.path, id, len(files[f.path])); err != nil {
			t.Fatal(err)
		}
		clips[f.path] = id
	}

	get := func(u *voiceUser, path string, want int) explorerProperties {
		t.Helper()
		var props explorerProperties
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/library/explorer/properties?path="+path, nil, want), &props); err != nil && want == http.StatusOK {
			t.Fatal(err)
		}
		return props
	}

	folder := get(member, "league", http.StatusOK)
	if !folder.Directory || folder.Name != "league" || folder.Path != "league" {
		t.Fatalf("folder identity: %+v", folder)
	}
	if folder.SizeBytes != int64(len(files["league/match.mp4"]+files["league/other.mp4"]+files["league/2026/inner.mp4"])) {
		t.Fatalf("folder size: %d", folder.SizeBytes)
	}
	if folder.VideoCount != 3 || folder.IndexedCount != 2 || folder.FolderCount != 1 {
		t.Fatalf("folder counts: %+v", folder)
	}
	if folder.Duration == nil || *folder.Duration != 180 {
		t.Fatalf("folder duration: %+v", folder.Duration)
	}
	if folder.AddedAt == nil || !folder.AddedAt.Equal(time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("folder added date: %+v", folder.AddedAt)
	}
	if len(folder.Owners) != 2 {
		t.Fatalf("folder owners: %+v", folder.Owners)
	}
	owned := map[string]int{}
	for _, owner := range folder.Owners {
		owned[owner.Name] = owner.Videos
	}
	if owned["explorer-owner-a"] != 1 || owned["explorer-owner-b"] != 1 {
		t.Fatalf("folder owner counts: %+v", folder.Owners)
	}

	file := get(member, "league/match.mp4", http.StatusOK)
	if file.Directory || file.SizeBytes != int64(len(files["league/match.mp4"])) || file.VideoCount != 1 || file.IndexedCount != 1 {
		t.Fatalf("video properties: %+v", file)
	}
	if file.Duration == nil || *file.Duration != 60 || file.AddedAt == nil || file.AddedAt.Day() != 2 {
		t.Fatalf("video metadata: %+v", file)
	}
	if file.State != "not_analyzed" || len(file.Owners) != 1 || file.Owners[0].Name != "explorer-owner-a" {
		t.Fatalf("video owner and state: %+v", file)
	}
	// A clip that is indexed but private to another owner must not leak metadata through its path.
	get(member, "top.mp4", http.StatusNotFound)
	unindexed := get(member, "league/other.mp4", http.StatusOK)
	if unindexed.State != "not_indexed" || unindexed.IndexedCount != 0 || unindexed.Duration != nil || len(unindexed.Owners) != 0 {
		t.Fatalf("unindexed video properties: %+v", unindexed)
	}

	loose := get(member, fmt.Sprintf("clip:%d", clips["league/2026/inner.mp4"]), http.StatusOK)
	if loose.Directory || loose.Duration == nil || *loose.Duration != 120 || loose.IndexedCount != 1 || loose.Owners[0].Name != "explorer-owner-b" {
		t.Fatalf("clip properties: %+v", loose)
	}
	get(member, fmt.Sprintf("clip:%d", clips["top.mp4"]), http.StatusNotFound)
	get(member, "clip:not-a-number", http.StatusBadRequest)

	// A private folder is refused to a member, open to the Super-administrateur.
	get(member, "secret", http.StatusForbidden)
	secret := get(h.adminUser(), "secret", http.StatusOK)
	if secret.VideoCount != 1 || secret.SizeBytes != int64(len(files["secret/plan.mp4"])) || secret.Duration == nil || *secret.Duration != 15 {
		t.Fatalf("private folder properties: %+v", secret)
	}

	// From the root, a member neither counts the private clip nor the contents of the private folder;
	// the Super-administrateur counts everything.
	rootMember := get(member, "", http.StatusOK)
	if rootMember.VideoCount != 3 || rootMember.SizeBytes != folder.SizeBytes || rootMember.FolderCount != 3 {
		t.Fatalf("root properties for a member: %+v", rootMember)
	}
	rootAdmin := get(h.adminUser(), "", http.StatusOK)
	if rootAdmin.VideoCount != 5 || rootAdmin.SizeBytes != 34 || rootAdmin.Duration == nil || *rootAdmin.Duration != 225 || rootAdmin.FolderCount != 3 {
		t.Fatalf("root properties for the Super-administrateur: %+v", rootAdmin)
	}
	get(member, "missing", http.StatusUnprocessableEntity)
	get(member, "league%2F..%2F..", http.StatusUnprocessableEntity)
}
