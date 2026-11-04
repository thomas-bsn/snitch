package httpapi

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Indexation of the whole Racine vidéo (#275, ADR 0038). An Administrator asks once, explicitly,
// for every video to get its clip card (mode "cards") or its card then its full analysis (mode
// "full"). The run only tags the source_files it targets; the managed scan, which already
// survives restarts and retries, does the work. Videos that already have a clip are never touched.

const (
	indexModeCards = "cards"
	indexModeFull  = "full"
	// indexMaxFiles bounds the files one run targets, so a mistaken Racine cannot flood the database.
	indexMaxFiles = 50000
	// indexFailureLimit bounds the per-video failures a status answer lists.
	indexFailureLimit = 20
)

var errIndexTooLarge = errors.New("too many videos for one run")

type rootVideo struct {
	rel      string
	size     int64
	modified time.Time
}

type indexFailure struct {
	Path string `json:"path"`
	Code string `json:"code"`
}

type indexRunView struct {
	ID         int64      `json:"id"`
	Mode       string     `json:"mode"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`
	// Total videos the run targets; Indexed have their clip; Failed could not be read; Pending are left.
	Total   int `json:"total"`
	Indexed int `json:"indexed"`
	Failed  int `json:"failed"`
	Pending int `json:"pending"`
	// AnalysisStarted counts the clips of the run whose analysis was queued (mode full).
	AnalysisStarted int `json:"analysis_started"`
	// WaitingForWorker is true when a full run has videos left and no Worker is online to take them.
	WaitingForWorker bool           `json:"waiting_for_worker"`
	Failures         []indexFailure `json:"failures"`
}

// walkRootVideos lists the video files under the Racine an Administrator may see: no symlink, no
// hidden entry, and no Dossier privé unless the viewer is a Super-administrator (ADR 0026).
func walkRootVideos(ctx context.Context, root string, private map[string]bool, superAdmin bool) ([]rootVideo, error) {
	var out []rootVideo
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !superAdmin && private[firstSegment(rel)] {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !entry.Type().IsRegular() || !videoExtension(entry.Name()) {
			return nil
		}
		info, e := entry.Info()
		if e != nil {
			return nil
		}
		if len(out) >= indexMaxFiles {
			return errIndexTooLarge
		}
		out = append(out, rootVideo{rel: rel, size: info.Size(), modified: info.ModTime()})
		return nil
	})
	return out, err
}

// indexTargets resolves the Racine and the videos a run would target for this viewer.
func (s *Server) indexTargets(w http.ResponseWriter, r *http.Request) (sourceID int64, videos []rootVideo, ok bool) {
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		fail(w, http.StatusConflict, "video_root_unconfigured")
		return 0, nil, false
	}
	ss := r.Context().Value(sessionKey{}).(session)
	private, err := s.privateTopFolders(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return 0, nil, false
	}
	videos, err = walkRootVideos(r.Context(), root.Path, private, isSuperAdministrator(ss.User.Role))
	switch {
	case errors.Is(err, errIndexTooLarge):
		fail(w, http.StatusUnprocessableEntity, "video_selection_too_large")
		return 0, nil, false
	case r.Context().Err() != nil:
		return 0, nil, false
	case err != nil:
		fail(w, http.StatusUnprocessableEntity, "video_path_unavailable")
		return 0, nil, false
	}
	return root.ID, videos, true
}

// indexedPaths returns which of the videos already have a clip.
func (s *Server) indexedPaths(ctx context.Context, sourceID int64, videos []rootVideo) (map[string]bool, error) {
	paths := make([]string, len(videos))
	for i, v := range videos {
		paths[i] = v.rel
	}
	rows, err := s.Pool.Query(ctx, `SELECT relative_path FROM source_files WHERE source_id=$1 AND clip_id IS NOT NULL AND relative_path=ANY($2)`, sourceID, paths)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out[p] = true
	}
	return out, rows.Err()
}

// previewIndexRun estimates a run without starting it: only directories are read, never the videos.
func (s *Server) previewIndexRun(w http.ResponseWriter, r *http.Request) {
	sourceID, videos, ok := s.indexTargets(w, r)
	if !ok {
		return
	}
	known, err := s.indexedPaths(r.Context(), sourceID, videos)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]int{"detected": len(videos), "already_indexed": len(known), "new": len(videos) - len(known)})
}

func (s *Server) startIndexRun(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Mode string `json:"mode"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Mode != indexModeCards && input.Mode != indexModeFull {
		fail(w, http.StatusBadRequest, "invalid_index_mode")
		return
	}
	sourceID, videos, ok := s.indexTargets(w, r)
	if !ok {
		return
	}
	known, err := s.indexedPaths(r.Context(), sourceID, videos)
	if err != nil {
		s.internal(w, err)
		return
	}
	var fresh []rootVideo
	for _, v := range videos {
		if !known[v.rel] {
			fresh = append(fresh, v)
		}
	}
	if len(fresh) == 0 {
		fail(w, http.StatusUnprocessableEntity, "nothing_to_index")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var runID int64
	err = tx.QueryRow(ctx, `INSERT INTO library_index_runs(source_id,mode,created_by) VALUES($1,$2,$3) RETURNING id`, sourceID, input.Mode, ss.User.ID).Scan(&runID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		fail(w, http.StatusConflict, "index_run_active")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	for start := 0; start < len(fresh); start += 1000 {
		end := min(start+1000, len(fresh))
		paths := make([]string, 0, end-start)
		sizes := make([]int64, 0, end-start)
		modified := make([]time.Time, 0, end-start)
		for _, v := range fresh[start:end] {
			paths, sizes, modified = append(paths, v.rel), append(sizes, v.size), append(modified, v.modified)
		}
		// A video that already has a clip is left as it is; a pending request of the explorer is taken over by the run.
		if _, err = tx.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,size_bytes,modified_at,stable_since,last_seen_generation,status,scan_requested,index_only,index_run_id)
			SELECT $1,p,sz,m,now(),0,'waiting_stability',true,$5,$6 FROM unnest($2::text[],$3::bigint[],$4::timestamptz[]) AS t(p,sz,m)
			ON CONFLICT(source_id,relative_path) DO UPDATE SET scan_requested=true,index_only=EXCLUDED.index_only,index_run_id=EXCLUDED.index_run_id,error_code=NULL WHERE source_files.clip_id IS NULL`,
			sourceID, paths, sizes, modified, input.Mode == indexModeCards, runID); err != nil {
			s.internal(w, err)
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	s.Library.ScheduleScan(sourceID, 0)
	view, err := s.indexRunView(ctx, runID)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusCreated, map[string]any{"run": view})
}

const indexRunColumns = `id,source_id,mode,status,created_at,finished_at`

// indexRunView builds the status of one run. A running run with no video left to process is
// completed here, once: there is no driver to do it, the scan is the only worker.
func (s *Server) indexRunView(ctx context.Context, id int64) (indexRunView, error) {
	var v indexRunView
	var sourceID int64
	if err := s.Pool.QueryRow(ctx, `SELECT `+indexRunColumns+` FROM library_index_runs WHERE id=$1`, id).Scan(&v.ID, &sourceID, &v.Mode, &v.Status, &v.CreatedAt, &v.FinishedAt); err != nil {
		return v, err
	}
	var left int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE clip_id IS NOT NULL), count(*) FILTER (WHERE clip_id IS NULL AND status IN ('missing','unreadable')), count(*) FILTER (WHERE scan_requested) FROM source_files WHERE index_run_id=$1`, id).Scan(&v.Total, &v.Indexed, &v.Failed, &left); err != nil {
		return v, err
	}
	v.Pending = v.Total - v.Indexed - v.Failed
	if v.Status == "running" && left == 0 {
		var finished time.Time
		err := s.Pool.QueryRow(ctx, `UPDATE library_index_runs SET status='completed',finished_at=now() WHERE id=$1 AND status='running' RETURNING finished_at`, id).Scan(&finished)
		if err == nil {
			v.Status, v.FinishedAt = "completed", &finished
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return v, err
		}
	}
	if err := s.Pool.QueryRow(ctx, `SELECT count(DISTINCT sf.clip_id) FROM source_files sf JOIN clip_analysis_runs a ON a.clip_id=sf.clip_id WHERE sf.index_run_id=$1`, id).Scan(&v.AnalysisStarted); err != nil {
		return v, err
	}
	if v.Mode == indexModeFull && v.Status == "running" && left > 0 {
		if err := s.Pool.QueryRow(ctx, `SELECT NOT EXISTS(SELECT 1 FROM workers WHERE revoked_at IS NULL AND last_seen_at > now()-interval '2 minutes')`).Scan(&v.WaitingForWorker); err != nil {
			return v, err
		}
	}
	v.Failures = []indexFailure{}
	rows, err := s.Pool.Query(ctx, `SELECT relative_path,COALESCE(error_code,status) FROM source_files WHERE index_run_id=$1 AND (error_code IS NOT NULL OR (clip_id IS NULL AND status IN ('missing','unreadable'))) ORDER BY relative_path LIMIT $2`, id, indexFailureLimit)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		var f indexFailure
		if err := rows.Scan(&f.Path, &f.Code); err != nil {
			return v, err
		}
		v.Failures = append(v.Failures, f)
	}
	return v, rows.Err()
}

// currentIndexRun answers the latest run of the Racine, or null when none was ever asked.
func (s *Server) currentIndexRun(w http.ResponseWriter, r *http.Request) {
	var id int64
	err := s.Pool.QueryRow(r.Context(), `SELECT r.id FROM library_index_runs r JOIN clip_sources c ON c.id=r.source_id AND c.managed_root ORDER BY r.id DESC LIMIT 1`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		reply(w, http.StatusOK, map[string]any{"run": nil})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	view, err := s.indexRunView(r.Context(), id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"run": view})
}

func (s *Server) pauseIndexRun(w http.ResponseWriter, r *http.Request) {
	s.changeIndexRun(w, r, "pause")
}
func (s *Server) resumeIndexRun(w http.ResponseWriter, r *http.Request) {
	s.changeIndexRun(w, r, "resume")
}
func (s *Server) cancelIndexRun(w http.ResponseWriter, r *http.Request) {
	s.changeIndexRun(w, r, "cancel")
}

// changeIndexRun pauses, resumes or cancels a run. The videos not yet turned into clips are the
// only ones concerned: a pause or a cancellation stops asking for them, a resume asks again.
func (s *Server) changeIndexRun(w http.ResponseWriter, r *http.Request, action string) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var status string
	var sourceID int64
	err = tx.QueryRow(ctx, `SELECT status,source_id FROM library_index_runs WHERE id=$1 FOR UPDATE`, id).Scan(&status, &sourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "index_run_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	allowed := map[string][]string{"pause": {"running"}, "resume": {"paused"}, "cancel": {"running", "paused"}}
	if !contains(allowed[action], status) {
		fail(w, http.StatusConflict, "index_run_state")
		return
	}
	switch action {
	case "pause":
		_, err = tx.Exec(ctx, `UPDATE library_index_runs SET status='paused' WHERE id=$1`, id)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE source_files SET scan_requested=false WHERE index_run_id=$1 AND clip_id IS NULL`, id)
		}
	case "resume":
		_, err = tx.Exec(ctx, `UPDATE library_index_runs SET status='running' WHERE id=$1`, id)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE source_files SET scan_requested=true,error_code=NULL WHERE index_run_id=$1 AND clip_id IS NULL`, id)
		}
	case "cancel":
		_, err = tx.Exec(ctx, `UPDATE library_index_runs SET status='cancelled',finished_at=now() WHERE id=$1`, id)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE source_files SET scan_requested=false WHERE index_run_id=$1 AND clip_id IS NULL`, id)
		}
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if action == "resume" {
		s.Library.ScheduleScan(sourceID, 0)
	}
	view, err := s.indexRunView(ctx, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"run": view})
}
