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

type rootScanRun struct {
	ID                int64    `json:"id"`
	Status            string   `json:"status"`
	Walking           bool     `json:"walking"`
	Seen              int      `json:"seen"`
	Queued            int      `json:"queued"`
	Done              int      `json:"done"`
	Pending           int      `json:"pending"`
	UpToDate          int      `json:"up_to_date"`
	UnreadableFolders int      `json:"unreadable_folders"`
	UnreadablePaths   []string `json:"unreadable_paths"`
	Excluded          struct {
		PrivateFolders int `json:"private_folders"`
		Hidden         int `json:"hidden"`
		System         int `json:"system"`
		PrivateClips   int `json:"private_clips"`
	} `json:"excluded"`
}

func decodeRootScan(t *testing.T, body []byte) rootScanRun {
	t.Helper()
	var out struct {
		Run *rootScanRun `json:"run"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.Run == nil {
		t.Fatalf("scan run answer %s: %v", body, err)
	}
	return *out.Run
}

// waitRootScanWalked reads the current scan until its walk of the disk is over.
func waitRootScanWalked(t *testing.T, h *voiceHarness) rootScanRun {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		run := decodeRootScan(t, h.adminCall(http.MethodGet, "/api/admin/video-root/scan-runs/current", nil, http.StatusOK))
		if !run.Walking {
			return run
		}
		if time.Now().After(deadline) {
			t.Fatalf("the walk never ended: %+v", run)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func writeSynthetic(t *testing.T, root string, files ...string) {
	t.Helper()
	for _, rel := range files {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), []byte("synthetic-scan "+rel), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// testRootScan covers #404: « Scanner toute la racine » takes every eligible video of the Racine, without
// a size limit, in the background, and leaves out Dossiers privés, hidden entries, NAS system folders and
// private clips, whatever the role. All files are synthetic and tiny.
func testRootScan(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	root := t.TempDir()
	h.adminCall(http.MethodPut, "/api/admin/video-root", map[string]string{"path": root}, http.StatusOK)
	var sourceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clip_sources WHERE managed_root`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}

	t.Run("more than 10 000 videos are all queued, in the background", func(t *testing.T) {
		const count = 10_050
		files := make([]string, count)
		for i := range files {
			files[i] = fmt.Sprintf("bulk/%03d/clip-%05d.mp4", i/100, i)
		}
		writeSynthetic(t, root, files...)
		started := time.Now()
		run := decodeRootScan(t, h.adminCall(http.MethodPost, "/api/admin/video-root/scan-runs", nil, http.StatusAccepted))
		if run.Status != "running" {
			t.Fatalf("started: %+v (%s)", run, time.Since(started))
		}
		walked := waitRootScanWalked(t, h)
		if walked.ID != run.ID || walked.Seen != count || walked.Queued != count {
			t.Fatalf("walked: %+v", walked)
		}
		var requested int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE source_id=$1 AND scan_requested AND index_run_id=$2`, sourceID, run.ID).Scan(&requested); err != nil || requested != count {
			t.Fatalf("requested %d of %d: %v", requested, count, err)
		}
		// The queue the screen polls is bounded by the server, with the total of the Racine.
		var listed struct {
			Items []struct {
				RelativePath string `json:"relative_path"`
			} `json:"items"`
			Total int `json:"total"`
		}
		filesPath := "/api/admin/sources/" + itoa(sourceID) + "/files"
		if err := json.Unmarshal(h.adminCall(http.MethodGet, filesPath+"?limit=200", nil, http.StatusOK), &listed); err != nil {
			t.Fatal(err)
		}
		if len(listed.Items) != 200 || listed.Total != count || listed.Items[0].RelativePath != files[0] {
			t.Fatalf("a bounded queue: %d items of %d, first %+v", len(listed.Items), listed.Total, listed.Items[:1])
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, filesPath, nil, http.StatusOK), &listed); err != nil {
			t.Fatal(err)
		}
		if len(listed.Items) != 1000 || listed.Total != count {
			t.Fatalf("without a limit the list is still bounded: %d items of %d", len(listed.Items), listed.Total)
		}
		for _, bad := range []string{"0", "1001", "many"} {
			h.adminCall(http.MethodGet, filesPath+"?limit="+bad, nil, http.StatusBadRequest)
		}
		// One scan at a time per Racine, shared with the indexation of the Racine.
		h.adminCall(http.MethodPost, "/api/admin/video-root/scan-runs", nil, http.StatusConflict)
		h.adminCall(http.MethodPost, "/api/admin/library/index-runs", map[string]string{"mode": "cards"}, http.StatusConflict)

		cancelled := decodeRootScan(t, h.adminCall(http.MethodPost, "/api/admin/video-root/scan-runs/"+itoa(run.ID)+"/cancel", nil, http.StatusOK))
		if cancelled.Status != "cancelled" || cancelled.Pending != 0 {
			t.Fatalf("cancelled: %+v", cancelled)
		}
		h.adminCall(http.MethodPost, "/api/admin/video-root/scan-runs/"+itoa(run.ID)+"/cancel", nil, http.StatusConflict)
		// The indexation endpoints do not drive a scan.
		h.adminCall(http.MethodPost, "/api/admin/library/index-runs/"+itoa(run.ID)+"/resume", nil, http.StatusNotFound)
		if err := os.RemoveAll(filepath.Join(root, "bulk")); err != nil {
			t.Fatal(err)
		}
	})

	// The scenario: new, changed and indexed videos, next to everything a scan must leave out.
	writeSynthetic(t, root,
		"top.mp4", "open/deep/new.mkv", "open/notes.txt",
		"secret/plan.mp4",
		".hidden/skip.mp4", "open/.dot.mp4",
		"#recycle/trash.mp4", "open/@eaDir/thumb.mp4",
		"known/analysed.mp4", "known/private.mp4", "known/fresh.mp4",
		"locked/inside.mp4",
	)
	if err := os.Symlink(filepath.Join(root, "top.mp4"), filepath.Join(root, "link.mp4")); err != nil {
		t.Fatal(err)
	}
	superAdmin := h.adminUser()
	h.userCall(superAdmin, http.MethodPut, "/api/admin/library/folders/privacy", map[string]any{"source_id": sourceID, "folder": "secret", "private": true}, http.StatusOK)
	// Three indexed videos, each with its own clip: one already analysed, one private, one never analysed.
	clipIDs := map[string]int64{}
	for i, name := range []string{"analysed", "private", "fresh"} {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,visibility,original_name,size_bytes,modified_at,recorded_at,content_sha256,tracks) SELECT id,'shared',$1,1,now(),now(),sha256($2::bytea),'[{"index":0,"type":"video","codec":"h264"},{"index":1,"type":"audio","codec":"aac"}]' FROM users ORDER BY id LIMIT 1 RETURNING id`, name+".mp4", []byte(fmt.Sprintf("root-scan-%d", i))).Scan(&id); err != nil {
			t.Fatal(err)
		}
		clipIDs[name] = id
		info, err := os.Stat(filepath.Join(root, "known", name+".mp4"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,clip_id,size_bytes,modified_at,last_seen_generation,status) VALUES($1,$2,$3,$4,$5,0,'ready')`, sourceID, "known/"+name+".mp4", id, info.Size(), info.ModTime().UTC().Truncate(time.Microsecond)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'transcription.audio.1','synthetic','done')`, clipIDs["analysed"]); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private' WHERE id=$1`, clipIDs["private"]); err != nil {
		t.Fatal(err)
	}
	// An unreadable folder (not testable as root, who reads everything).
	unreadable := os.Geteuid() != 0
	if unreadable {
		if err := os.Chmod(filepath.Join(root, "locked"), 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "locked"), 0o700) })
	}
	wantUnreadable := 0
	if unreadable {
		wantUnreadable = 1
	}
	admin := h.newUser("scan-admin")
	if _, err := pool.Exec(ctx, `UPDATE users SET role=$1 WHERE username='scan-admin'`, roleAdministrator); err != nil {
		t.Fatal(err)
	}
	member := h.newUser("scan-member")

	t.Run("members neither preview, start, read nor cancel a scan", func(t *testing.T) {
		h.userCall(member, http.MethodGet, "/api/admin/video-root/scan-runs/preview", nil, http.StatusForbidden)
		h.userCall(member, http.MethodPost, "/api/admin/video-root/scan-runs", nil, http.StatusForbidden)
		h.userCall(member, http.MethodGet, "/api/admin/video-root/scan-runs/current", nil, http.StatusForbidden)
		h.userCall(member, http.MethodPost, "/api/admin/video-root/scan-runs/1/cancel", nil, http.StatusForbidden)
	})

	t.Run("the preview counts what will be queued and left out, the same for every administrator", func(t *testing.T) {
		for _, u := range []*voiceUser{superAdmin, admin} {
			var got struct {
				Seen              int      `json:"seen"`
				ToQueue           int      `json:"to_queue"`
				UpToDate          int      `json:"up_to_date"`
				UnreadableFolders int      `json:"unreadable_folders"`
				UnreadablePaths   []string `json:"unreadable_paths"`
				Excluded          struct {
					PrivateFolders int `json:"private_folders"`
					Hidden         int `json:"hidden"`
					System         int `json:"system"`
					PrivateClips   int `json:"private_clips"`
				} `json:"excluded"`
			}
			body := h.userCall(u, http.MethodGet, "/api/admin/video-root/scan-runs/preview", nil, http.StatusOK)
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatal(err)
			}
			// top.mp4, open/deep/new.mkv and the never analysed known/fresh.mp4 are queued.
			if got.ToQueue != 3 || got.UpToDate != 1 || got.Seen != 5 || got.Excluded.PrivateFolders != 1 || got.Excluded.Hidden != 2 || got.Excluded.System != 2 || got.Excluded.PrivateClips != 1 || got.UnreadableFolders != wantUnreadable {
				t.Fatalf("%s preview: %s", u.name, body)
			}
			if unreadable && (len(got.UnreadablePaths) != 1 || got.UnreadablePaths[0] != "locked") {
				t.Fatalf("%s preview names the unreadable folder: %s", u.name, body)
			}
			if strings.Contains(string(body), "plan.mp4") {
				t.Fatalf("the preview reveals a private path: %s", body)
			}
		}
	})

	var scanned rootScanRun
	t.Run("the scan queues every eligible video and nothing that must be left out, whatever the role", func(t *testing.T) {
		started := decodeRootScan(t, h.userCall(superAdmin, http.MethodPost, "/api/admin/video-root/scan-runs", nil, http.StatusAccepted))
		scanned = waitRootScanWalked(t, h)
		if scanned.ID != started.ID || scanned.Queued != 3 || scanned.Pending != 3 || scanned.UpToDate != 1 || scanned.Seen != 5 ||
			scanned.Excluded.PrivateFolders != 1 || scanned.Excluded.Hidden != 2 || scanned.Excluded.System != 2 || scanned.Excluded.PrivateClips != 1 || scanned.UnreadableFolders != wantUnreadable {
			t.Fatalf("report: %+v", scanned)
		}
		rows, err := pool.Query(ctx, `SELECT relative_path FROM source_files WHERE source_id=$1 AND scan_requested ORDER BY relative_path`, sourceID)
		if err != nil {
			t.Fatal(err)
		}
		var asked []string
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err != nil {
				t.Fatal(err)
			}
			asked = append(asked, p)
		}
		rows.Close()
		if strings.Join(asked, ",") != "known/fresh.mp4,open/deep/new.mkv,top.mp4" {
			t.Fatalf("asked of the managed scan: %v", asked)
		}
		var indexOnly bool
		if err := pool.QueryRow(ctx, `SELECT index_only FROM source_files WHERE relative_path='known/fresh.mp4'`).Scan(&indexOnly); err != nil || indexOnly {
			t.Fatalf("a never analysed clip is queued for its analysis: index_only=%v %v", indexOnly, err)
		}
	})

	t.Run("relaunching creates no duplicate and never re-analyses an analysed clip", func(t *testing.T) {
		h.adminCall(http.MethodPost, "/api/admin/video-root/scan-runs/"+itoa(scanned.ID)+"/cancel", nil, http.StatusOK)
		again := decodeRootScan(t, h.userCall(admin, http.MethodPost, "/api/admin/video-root/scan-runs", nil, http.StatusAccepted))
		walked := waitRootScanWalked(t, h)
		if walked.ID != again.ID || walked.Queued != 3 || walked.UpToDate != 1 || walked.Excluded.PrivateFolders != 1 {
			t.Fatalf("an administrator's scan leaves the private folder out too: %+v", walked)
		}
		var rows, jobs int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE source_id=$1 AND relative_path IN ('top.mp4','known/fresh.mp4','open/deep/new.mkv')`, sourceID).Scan(&rows); err != nil || rows != 3 {
			t.Fatalf("one row per video: %d %v", rows, err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE clip_id=$1`, clipIDs["analysed"]).Scan(&jobs); err != nil || jobs != 1 {
			t.Fatalf("an analysed clip got new work: %d %v", jobs, err)
		}
		var touched int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE source_id=$1 AND (relative_path LIKE 'secret/%' OR relative_path LIKE '%.dot.mp4' OR relative_path LIKE '.hidden/%' OR relative_path LIKE '%#recycle%' OR relative_path LIKE '%@eaDir%' OR relative_path='link.mp4' OR relative_path LIKE 'locked/%' OR (relative_path IN ('known/analysed.mp4','known/private.mp4') AND scan_requested))`, sourceID).Scan(&touched); err != nil || touched != 0 {
			t.Fatalf("%d excluded videos were queued: %v", touched, err)
		}
		// Once the managed scan has taken every video, the run is completed and keeps its report.
		if _, err := pool.Exec(ctx, `UPDATE source_files SET scan_requested=false WHERE index_run_id=$1`, again.ID); err != nil {
			t.Fatal(err)
		}
		done := decodeRootScan(t, h.adminCall(http.MethodGet, "/api/admin/video-root/scan-runs/current", nil, http.StatusOK))
		if done.Status != "completed" || done.Done != 3 || done.Pending != 0 || done.UpToDate != 1 {
			t.Fatalf("completed: %+v", done)
		}
	})

	t.Run("the Super-administrator still scans a Dossier privé selected explicitly; others are refused", func(t *testing.T) {
		h.userCall(admin, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{"secret"}, "confirm": false}, http.StatusForbidden)
		var preview struct {
			Paths []string `json:"paths"`
		}
		if err := json.Unmarshal(h.userCall(superAdmin, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{"secret"}, "confirm": true}, http.StatusOK), &preview); err != nil {
			t.Fatal(err)
		}
		if len(preview.Paths) != 1 || preview.Paths[0] != "secret/plan.mp4" {
			t.Fatalf("explicit private selection: %v", preview.Paths)
		}
		if _, err := pool.Exec(ctx, `UPDATE source_files SET scan_requested=false WHERE relative_path='secret/plan.mp4'`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("scanning a selection again leaves an analysed video alone, like the scan of the whole Racine", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE source_files SET scan_requested=false WHERE relative_path LIKE 'known/%'`); err != nil {
			t.Fatal(err)
		}
		requested := func() map[string]bool {
			rows, err := pool.Query(ctx, `SELECT relative_path,scan_requested FROM source_files WHERE source_id=$1 AND relative_path LIKE 'known/%'`, sourceID)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			out := map[string]bool{}
			for rows.Next() {
				var p string
				var r bool
				if err = rows.Scan(&p, &r); err != nil {
					t.Fatal(err)
				}
				out[p] = r
			}
			return out
		}
		var preview struct {
			Paths []string `json:"paths"`
		}
		if err := json.Unmarshal(h.userCall(superAdmin, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{"known"}, "confirm": true}, http.StatusOK), &preview); err != nil {
			t.Fatal(err)
		}
		if len(preview.Paths) != 3 {
			t.Fatalf("the selection lists its videos: %v", preview.Paths)
		}
		if got := requested(); got["known/analysed.mp4"] || !got["known/fresh.mp4"] {
			t.Fatalf("only a video not analysed yet is queued: %v", got)
		}
		var jobs int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE clip_id=$1`, clipIDs["analysed"]).Scan(&jobs); err != nil || jobs != 1 {
			t.Fatalf("an analysed clip got new work: %d %v", jobs, err)
		}
		// A changed file is new content: it is queued again. The file is put back as it was afterwards.
		analysedPath := filepath.Join(root, "known", "analysed.mp4")
		original, err := os.ReadFile(analysedPath)
		if err != nil {
			t.Fatal(err)
		}
		before, err := os.Stat(analysedPath)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = os.WriteFile(analysedPath, original, 0o600)
			_ = os.Chtimes(analysedPath, before.ModTime(), before.ModTime())
		})
		if err = os.WriteFile(analysedPath, []byte("changed content"), 0o600); err != nil {
			t.Fatal(err)
		}
		h.userCall(superAdmin, http.MethodPost, "/api/admin/video-root/scan", map[string]any{"paths": []string{"known/analysed.mp4"}, "confirm": true}, http.StatusOK)
		if got := requested(); !got["known/analysed.mp4"] {
			t.Fatalf("a changed video is queued: %v", got)
		}
		if err = os.WriteFile(analysedPath, original, 0o600); err != nil {
			t.Fatal(err)
		}
		if err = os.Chtimes(analysedPath, before.ModTime(), before.ModTime()); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `UPDATE source_files SET scan_requested=false WHERE relative_path LIKE 'known/%'`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("a walk interrupted by a stop starts over and ends", func(t *testing.T) {
		writeSynthetic(t, root, "later/after-restart.mp4")
		// A run left walking by a stop: its row says so, no process walks it.
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO library_index_runs(source_id,mode,walking,seen) VALUES($1,'scan',true,2) RETURNING id`, sourceID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if err := h.api.Library.ResumeRootScans(ctx); err != nil {
			t.Fatal(err)
		}
		resumed := waitRootScanWalked(t, h)
		if resumed.ID != id || resumed.Seen != 6 || resumed.Queued != 4 || resumed.UpToDate != 1 {
			t.Fatalf("resumed: %+v", resumed)
		}
		var asked bool
		if err := pool.QueryRow(ctx, `SELECT scan_requested FROM source_files WHERE relative_path='later/after-restart.mp4' AND index_run_id=$1`, id).Scan(&asked); err != nil || !asked {
			t.Fatalf("the resumed walk queues the new video: %v %v", asked, err)
		}
	})

	t.Run("the managed scan then analyses a never analysed clip as a new one, and leaves the analysed one alone", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE workers SET last_seen_at=now()`); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 20; i++ {
			var left int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM source_files WHERE source_id=$1 AND scan_requested AND relative_path='known/fresh.mp4'`, sourceID).Scan(&left); err != nil {
				t.Fatal(err)
			}
			if left == 0 {
				break
			}
			if err := h.api.Library.ScanSource(ctx, sourceID); err != nil {
				t.Fatal(err)
			}
		}
		var analysed bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clip_analysis_runs WHERE clip_id=$1) OR EXISTS(SELECT 1 FROM jobs WHERE clip_id=$1)`, clipIDs["fresh"]).Scan(&analysed); err != nil || !analysed {
			var code, status string
			var asked bool
			_ = pool.QueryRow(ctx, `SELECT COALESCE(error_code,''),status,scan_requested FROM source_files WHERE relative_path='known/fresh.mp4'`).Scan(&code, &status, &asked)
			t.Fatalf("the never analysed clip was not handed to the analysis: %v (%s %s %v)", err, code, status, asked)
		}
		var jobs, runs int
		if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM jobs WHERE clip_id=$1),(SELECT count(*) FROM clip_analysis_runs WHERE clip_id=$1)`, clipIDs["analysed"]).Scan(&jobs, &runs); err != nil || jobs != 1 || runs != 0 {
			t.Fatalf("the analysed clip got new work: %d jobs, %d runs, %v", jobs, runs, err)
		}
	})
}
