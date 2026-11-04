//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testExplorerFilteredTree covers #282: with a processing filter, the explorer keeps the matching
// videos and only the folders that lead to one, counted over every descendant the viewer may read.
func testExplorerFilteredTree(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	root := t.TempDir()
	files := map[string][]string{
		"BO3":        {"a.mp4", "b.mp4"},
		"Jeux/BO3":   {"c.mp4", "hidden.mp4"},
		"Jeux/Other": {"d.mp4"},
		"solo":       {"e.mp4"},
		"secret":     {"f.mp4"},
		"empty":      {"x.mp4"},
	}
	for dir, names := range files {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			if err := os.WriteFile(filepath.Join(root, dir, name), []byte("synthetic "+name), 0o600); err != nil {
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
	member := h.newUser("tree-member")
	h.newUser("tree-owner")
	var ownerID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='tree-owner'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}

	seed := byte(10)
	index := func(path, visibility string) int64 {
		seed++
		id := insertExplorerClip(t, pool, ownerID, visibility, filepath.Base(path), seed)
		if _, err := pool.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,clip_id,size_bytes,modified_at,last_seen_generation,status) VALUES($1,$2,$3,10,now(),0,'ready')`, sourceID, path, id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	transcribe := func(id int64) {
		if _, err := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'transcription.audio.1','tree','done')`, id); err != nil {
			t.Fatal(err)
		}
	}
	run := func(id int64, scope string, steps map[string]string) {
		var runID int64
		if err := pool.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,status) VALUES($1,$2,'completed') RETURNING id`, id, scope).Scan(&runID); err != nil {
			t.Fatal(err)
		}
		for step, status := range steps {
			if _, err := pool.Exec(ctx, `INSERT INTO clip_analysis_steps(run_id,step,status) VALUES($1,$2,$3)`, runID, step, status); err != nil {
				t.Fatal(err)
			}
		}
	}
	done := func(steps ...string) map[string]string {
		out := map[string]string{}
		for _, s := range steps {
			out[s] = "done"
		}
		return out
	}
	index("BO3/a.mp4", "shared")
	index("BO3/b.mp4", "shared")
	c := index("Jeux/BO3/c.mp4", "shared")
	transcribe(c)
	hidden := index("Jeux/BO3/hidden.mp4", "private") // transcribed, but not readable by the member
	transcribe(hidden)
	index("Jeux/Other/d.mp4", "shared")
	e := index("solo/e.mp4", "shared")
	transcribe(e)
	run(e, "full", done("preparation", "transcription", "indexing", "diarization", "fingerprints", "matching"))
	f := index("secret/f.mp4", "shared")
	transcribe(f)
	type entry struct {
		Name       string `json:"name"`
		Path       string `json:"path"`
		Directory  bool   `json:"directory"`
		MatchCount *int   `json:"match_count"`
	}
	list := func(u *voiceUser, query string) (map[string]entry, int) {
		t.Helper()
		var page struct {
			Items []entry `json:"items"`
			Total int     `json:"total"`
		}
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/library/explorer?"+query, nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		out := map[string]entry{}
		for _, item := range page.Items {
			out[item.Path] = item
		}
		return out, page.Total
	}
	count := func(e entry) int {
		if e.MatchCount == nil {
			return -1
		}
		return *e.MatchCount
	}

	t.Run("a folder with no transcribed video disappears, ancestors of a match stay", func(t *testing.T) {
		got, total := list(member, "path=&stage=transcribed")
		if _, ok := got["BO3"]; ok {
			t.Fatalf("BO3 has no transcription and must disappear: %+v", got)
		}
		if _, ok := got["empty"]; ok {
			t.Fatalf("a folder of unindexed videos must disappear: %+v", got)
		}
		if len(got) != 2 || total != 2 || count(got["Jeux"]) != 1 || count(got["solo"]) != 1 {
			t.Fatalf("root under the filter: total=%d %+v", total, got)
		}
		inner, _ := list(member, "path=Jeux&stage=transcribed")
		if len(inner) != 1 || count(inner["Jeux/BO3"]) != 1 {
			t.Fatalf("Jeux under the filter: %+v", inner)
		}
		leaf, _ := list(member, "path=Jeux%2FBO3&stage=transcribed")
		if len(leaf) != 1 || leaf["Jeux/BO3/c.mp4"].Name == "" {
			t.Fatalf("only the transcribed video remains: %+v", leaf)
		}
	})
	t.Run("one transcription keeps the folder and shows exactly that video", func(t *testing.T) {
		transcribe(func() int64 {
			var id int64
			if err := pool.QueryRow(ctx, `SELECT clip_id FROM source_files WHERE relative_path='BO3/b.mp4'`).Scan(&id); err != nil {
				t.Fatal(err)
			}
			return id
		}())
		got, _ := list(member, "path=&stage=transcribed")
		if count(got["BO3"]) != 1 {
			t.Fatalf("BO3 must stay with one match: %+v", got)
		}
		leaf, _ := list(member, "path=BO3&stage=transcribed")
		if len(leaf) != 1 || leaf["BO3/b.mp4"].Name == "" {
			t.Fatalf("BO3 content: %+v", leaf)
		}
	})
	t.Run("counts never include what the viewer may not read", func(t *testing.T) {
		got, _ := list(member, "path=Jeux&stage=transcribed")
		if count(got["Jeux/BO3"]) != 1 {
			t.Fatalf("the private clip leaks through the counter: %+v", got)
		}
		if root, _ := list(member, "path=&stage=transcribed"); root["secret"].Name != "" {
			t.Fatal("a private folder leaks to a member")
		}
		admin, _ := list(superAdmin, "path=&stage=transcribed")
		if count(admin["secret"]) != 1 {
			t.Fatalf("the Super-administrator sees the private folder: %+v", admin)
		}
	})
	t.Run("indexed means a clip exists, complete means every step done", func(t *testing.T) {
		got, _ := list(member, "path=&stage=indexed")
		if count(got["empty"]) != -1 || count(got["BO3"]) != 2 || count(got["Jeux"]) != 2 {
			t.Fatalf("indexed: %+v", got)
		}
		got, _ = list(member, "path=&stage=complete")
		if len(got) != 1 || count(got["solo"]) != 1 {
			t.Fatalf("complete: %+v", got)
		}
	})
	t.Run("voices processed needs the voice steps done", func(t *testing.T) {
		got, _ := list(member, "path=&stage=voices")
		if len(got) != 1 || count(got["solo"]) != 1 {
			t.Fatalf("voices before any other run: %+v", got)
		}
		run(c, "voice", done("diarization", "fingerprints", "matching"))
		got, _ = list(member, "path=&stage=voices")
		if len(got) != 2 || count(got["Jeux"]) != 1 {
			t.Fatalf("voices after a voice run: %+v", got)
		}
		got, _ = list(member, "path=&stage=complete")
		if len(got) != 1 {
			t.Fatalf("a voice-only run does not complete the analysis: %+v", got)
		}
	})
	t.Run("filters combine, the flat view stays, an unknown stage is refused", func(t *testing.T) {
		got, _ := list(member, "path=&stage=transcribed&q=c.mp4")
		if len(got) != 1 || count(got["Jeux"]) != 1 {
			t.Fatalf("stage and name: %+v", got)
		}
		flat, _ := list(member, "path=&recursive=1&stage=transcribed")
		for _, e := range flat {
			if e.Directory {
				t.Fatalf("the flat view lists videos only: %+v", flat)
			}
		}
		all, _ := list(member, "path=")
		if all["BO3"].Name == "" || all["empty"].Name == "" || count(all["BO3"]) != -1 {
			t.Fatalf("without a filter the tree is complete and uncounted: %+v", all)
		}
		h.userCall(member, http.MethodGet, "/api/library/explorer?path=&stage=nope", nil, http.StatusBadRequest)
	})
}
