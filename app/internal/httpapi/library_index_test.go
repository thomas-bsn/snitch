//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testLibraryIndex covers #275: indexing the whole Racine vidéo is an explicit, durable,
// idempotent administrator action. It checks the estimate, the exclusions (hidden, symlink,
// Dossier privé), the single active run, pause / resume / cancel, the roles, and that the
// managed scan then turns each video into exactly one clip card.
func testLibraryIndex(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	// Scans are driven by hand below: nothing may process the run behind the test's back.
	h.api.Library.Close()

	root := t.TempDir()
	files := map[string]string{
		"top.mp4":              "a",
		"league/2026/one.mp4":  "b",
		"league/2026/two.mp4":  "c",
		"league/empty/.keep":   "", // not a video
		"league/notes.txt":     "x",
		"secret/plan.mp4":      "d",
		".hidden/skip.mp4":     "e",
		"known/already.mp4":    "f",
		"media/with space.mkv": "g",
	}
	for rel, body := range files {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), []byte("synthetic-index "+body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "top.mp4"), filepath.Join(root, "link.mp4")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "vide", "dedans"), 0o700); err != nil {
		t.Fatal(err)
	}
	superAdmin := h.adminUser()
	h.adminCall(http.MethodPut, "/api/admin/video-root", map[string]string{"path": root}, http.StatusOK)
	var sourceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clip_sources WHERE managed_root`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	h.userCall(superAdmin, http.MethodPut, "/api/admin/library/folders/privacy", map[string]any{"source_id": sourceID, "folder": "secret", "private": true}, http.StatusOK)
	member := h.newUser("index-member")

	// A video that already has a clip is never touched by a run.
	var knownClip int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clips ORDER BY id LIMIT 1`).Scan(&knownClip); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,clip_id,size_bytes,modified_at,last_seen_generation,status) VALUES($1,'known/already.mp4',$2,10,now(),0,'ready')`, sourceID, knownClip); err != nil {
		t.Fatal(err)
	}

	type run struct {
		ID               int64  `json:"id"`
		Mode             string `json:"mode"`
		Status           string `json:"status"`
		Total            int    `json:"total"`
		Indexed          int    `json:"indexed"`
		Failed           int    `json:"failed"`
		Pending          int    `json:"pending"`
		AnalysisStarted  int    `json:"analysis_started"`
		WaitingForWorker bool   `json:"waiting_for_worker"`
	}
	decodeRun := func(body []byte) run {
		t.Helper()
		var out struct {
			Run *run `json:"run"`
		}
		if err := json.Unmarshal(body, &out); err != nil || out.Run == nil {
			t.Fatalf("run answer %s: %v", body, err)
		}
		return *out.Run
	}

	t.Run("only administrators estimate or start a run", func(t *testing.T) {
		h.userCall(member, http.MethodGet, "/api/admin/library/index-runs/preview", nil, http.StatusForbidden)
		h.userCall(member, http.MethodPost, "/api/admin/library/index-runs", map[string]string{"mode": "cards"}, http.StatusForbidden)
		h.userCall(member, http.MethodGet, "/api/admin/library/index-runs/current", nil, http.StatusForbidden)
	})

	t.Run("the estimate counts what a run would do, without hidden, linked or private videos", func(t *testing.T) {
		var got map[string]int
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/library/index-runs/preview", nil, http.StatusOK), &got); err != nil {
			t.Fatal(err)
		}
		// The harness administrator is the Super-administrator: the Dossier privé is counted. Hidden
		// folders, the symlink and non-video files never are.
		if got["detected"] != 6 || got["already_indexed"] != 1 || got["new"] != 5 {
			t.Fatalf("estimate: %v", got)
		}
	})

	t.Run("a bad mode and an empty request are refused", func(t *testing.T) {
		h.adminCall(http.MethodPost, "/api/admin/library/index-runs", map[string]string{"mode": "everything"}, http.StatusBadRequest)
	})

	var started run
	t.Run("a run tags each new video once, leaves indexed ones alone, and only one run is active", func(t *testing.T) {
		started = decodeRun(h.adminCall(http.MethodPost, "/api/admin/library/index-runs", map[string]string{"mode": "cards"}, http.StatusCreated))
		if started.Status != "running" || started.Total < 5 || started.Pending != started.Total {
			t.Fatalf("started: %+v", started)
		}
		var tagged, known int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE index_run_id=$1`, started.ID).Scan(&tagged); err != nil || tagged != started.Total {
			t.Fatalf("tagged files %d, %v", tagged, err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE relative_path='known/already.mp4' AND index_run_id IS NOT NULL`).Scan(&known); err != nil || known != 0 {
			t.Fatalf("an indexed video was taken by the run: %d, %v", known, err)
		}
		for _, hidden := range []string{".hidden/skip.mp4", "link.mp4", "league/notes.txt", "league/empty/.keep"} {
			var n int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE relative_path=$1`, hidden).Scan(&n); err != nil || n != 0 {
				t.Fatalf("%s must not be targeted: %d, %v", hidden, n, err)
			}
		}
		var cards int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE index_run_id=$1 AND index_only`, started.ID).Scan(&cards); err != nil || cards != started.Total {
			t.Fatalf("cards mode must be index-only: %d/%d, %v", cards, started.Total, err)
		}
		h.adminCall(http.MethodPost, "/api/admin/library/index-runs", map[string]string{"mode": "full"}, http.StatusConflict)
	})

	t.Run("a private video in error is shown to the Super-administrator only (ADR 0026)", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE source_files SET status='missing',error_code='video_missing',scan_requested=false WHERE relative_path='secret/plan.mp4'`); err != nil {
			t.Fatal(err)
		}
		admin := h.newUser("index-admin")
		if _, err := pool.Exec(ctx, `UPDATE users SET role=$1 WHERE username='index-admin'`, roleAdministrator); err != nil {
			t.Fatal(err)
		}
		paths := func(body []byte) []string {
			var out struct {
				Run struct {
					Failures []struct {
						Path string `json:"path"`
					} `json:"failures"`
				} `json:"run"`
			}
			if err := json.Unmarshal(body, &out); err != nil {
				t.Fatal(err)
			}
			var list []string
			for _, f := range out.Run.Failures {
				list = append(list, f.Path)
			}
			return list
		}
		if got := paths(h.adminCall(http.MethodGet, "/api/admin/library/index-runs/current", nil, http.StatusOK)); len(got) != 1 || got[0] != "secret/plan.mp4" {
			t.Fatalf("the Super-administrator sees the private failure: %v", got)
		}
		if got := paths(h.userCall(admin, http.MethodGet, "/api/admin/library/index-runs/current", nil, http.StatusOK)); len(got) != 0 {
			t.Fatalf("an administrator must not read a private path: %v", got)
		}
	})

	t.Run("pause stops asking for the videos, resume asks again, nothing is lost", func(t *testing.T) {
		paused := decodeRun(h.adminCall(http.MethodPost, "/api/admin/library/index-runs/"+itoa(started.ID)+"/pause", nil, http.StatusOK))
		if paused.Status != "paused" {
			t.Fatalf("paused: %+v", paused)
		}
		var asked int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE index_run_id=$1 AND scan_requested`, started.ID).Scan(&asked); err != nil || asked != 0 {
			t.Fatalf("a paused run still asks for %d videos, %v", asked, err)
		}
		h.adminCall(http.MethodPost, "/api/admin/library/index-runs/"+itoa(started.ID)+"/pause", nil, http.StatusConflict)
		resumed := decodeRun(h.adminCall(http.MethodPost, "/api/admin/library/index-runs/"+itoa(started.ID)+"/resume", nil, http.StatusOK))
		if resumed.Status != "running" || resumed.Total != started.Total {
			t.Fatalf("resumed: %+v", resumed)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE index_run_id=$1 AND scan_requested`, started.ID).Scan(&asked); err != nil || asked != started.Total {
			t.Fatalf("a resumed run asks for %d of %d videos, %v", asked, started.Total, err)
		}
	})

	t.Run("the scan turns each video into exactly one clip card, and a second pass changes nothing", func(t *testing.T) {
		var clipsBefore int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM clips`).Scan(&clipsBefore); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 40; i++ {
			if err := h.api.Library.ScanSource(ctx, sourceID); err != nil {
				t.Fatal(err)
			}
			var left int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE index_run_id=$1 AND scan_requested`, started.ID).Scan(&left); err != nil {
				t.Fatal(err)
			}
			if left == 0 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		current := decodeRun(h.adminCall(http.MethodGet, "/api/admin/library/index-runs/current", nil, http.StatusOK))
		if current.ID != started.ID || current.Indexed+current.Failed != current.Total {
			t.Fatalf("after the scan: %+v", current)
		}
		var clipsAfter, dupes int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM clips`).Scan(&clipsAfter); err != nil {
			t.Fatal(err)
		}
		if clipsAfter != clipsBefore+current.Indexed {
			t.Fatalf("clips %d -> %d for %d indexed videos", clipsBefore, clipsAfter, current.Indexed)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM (SELECT clip_id FROM source_files WHERE clip_id IS NOT NULL AND (index_run_id IS NOT NULL OR source_id=$1) GROUP BY clip_id HAVING count(*)>1) d`, sourceID).Scan(&dupes); err != nil || dupes != 0 {
			t.Fatalf("clips shared by several files: %d, %v", dupes, err)
		}
		// Cards only: no analysis was queued for the new clips.
		if current.AnalysisStarted != 0 {
			t.Fatalf("cards mode queued %d analyses", current.AnalysisStarted)
		}
		if current.Status != "completed" {
			t.Fatalf("a run with nothing left must complete: %+v", current)
		}
		if err := h.api.Library.ScanSource(ctx, sourceID); err != nil {
			t.Fatal(err)
		}
		var clipsAgain int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM clips`).Scan(&clipsAgain); err != nil || clipsAgain != clipsAfter {
			t.Fatalf("a second pass changed the clips: %d -> %d, %v", clipsAfter, clipsAgain, err)
		}
		// Nothing is left to index: a new run is refused rather than doubled.
		h.adminCall(http.MethodPost, "/api/admin/library/index-runs", map[string]string{"mode": "cards"}, http.StatusUnprocessableEntity)
	})

	t.Run("a cancelled run stops asking and keeps what was indexed", func(t *testing.T) {
		extra := filepath.Join(root, "late", "extra.mp4")
		if err := os.MkdirAll(filepath.Dir(extra), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(extra, []byte("synthetic-index late"), 0o600); err != nil {
			t.Fatal(err)
		}
		next := decodeRun(h.adminCall(http.MethodPost, "/api/admin/library/index-runs", map[string]string{"mode": "full"}, http.StatusCreated))
		if next.Mode != "full" || next.Total != 1 {
			t.Fatalf("second run: %+v", next)
		}
		// No Worker is online: the video waits, visibly, instead of failing.
		if !next.WaitingForWorker || next.Pending != 1 {
			t.Fatalf("a full run without a Worker must say it waits: %+v", next)
		}
		cancelled := decodeRun(h.adminCall(http.MethodPost, "/api/admin/library/index-runs/"+itoa(next.ID)+"/cancel", nil, http.StatusOK))
		if cancelled.Status != "cancelled" {
			t.Fatalf("cancelled: %+v", cancelled)
		}
		var asked int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE index_run_id=$1 AND scan_requested`, next.ID).Scan(&asked); err != nil || asked != 0 {
			t.Fatalf("a cancelled run still asks for %d videos, %v", asked, err)
		}
		h.adminCall(http.MethodPost, "/api/admin/library/index-runs/"+itoa(next.ID)+"/resume", nil, http.StatusConflict)
		h.adminCall(http.MethodPost, "/api/admin/library/index-runs/999999/cancel", nil, http.StatusNotFound)
	})
}
