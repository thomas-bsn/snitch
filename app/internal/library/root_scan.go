package library

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// « Scanner toute la racine » (#404, ADR 0038 amended). A scan of the whole Racine vidéo is a
// library_index_runs row of mode "scan". The request only records it; the walk of the disk runs
// here, in the background, and starts over after a restart while the run is still walking. Each
// eligible video is tagged with the run and asked of the managed scan, which indexes it; its
// analysis then follows the processing settings, as for any new clip. Nothing is ever queued from
// a Dossier privé (whatever the role), a hidden entry, a NAS system folder or a private clip, and
// a clip already analysed is left alone: re-analysis stays a separate action.

var (
	// ErrRootScanActive is returned when an indexation or a scan of the Racine is already in progress.
	ErrRootScanActive = errors.New("an indexation of the Racine is already in progress")
	// ErrRootScanState is returned when a scan cannot change from its current status.
	ErrRootScanState = errors.New("the scan of the Racine is not running")
)

const (
	// rootScanBatch is how many videos one transaction classifies and queues.
	rootScanBatch = 500
	// rootScanUnreadableNames bounds the unreadable folders a run names.
	rootScanUnreadableNames = 20
	// rootScanFailureLimit bounds the per-video failures a status lists.
	rootScanFailureLimit = 20
)

// RootScanExcluded counts what a scan of the Racine leaves out, by reason.
type RootScanExcluded struct {
	// PrivateFolders are the Dossiers privés skipped without being opened.
	PrivateFolders int `json:"private_folders"`
	// Hidden are the folders and videos whose name starts with a dot.
	Hidden int `json:"hidden"`
	// System are the NAS folders #recycle and @eaDir.
	System int `json:"system"`
	// PrivateClips are indexed videos whose clip is private.
	PrivateClips int `json:"private_clips"`
}

// RootScanTally is what one walk of the Racine found.
type RootScanTally struct {
	// Seen counts the eligible videos met (queued, up to date or private clip).
	Seen int `json:"seen"`
	// ToQueue counts the videos the scan asks for: not indexed, changed, or indexed but never analysed.
	ToQueue int `json:"to_queue"`
	// UpToDate counts the indexed videos whose clip was already analysed.
	UpToDate          int              `json:"up_to_date"`
	Excluded          RootScanExcluded `json:"excluded"`
	UnreadableFolders int              `json:"unreadable_folders"`
	UnreadablePaths   []string         `json:"unreadable_paths"`
}

// RootScanFailure is a queued video the managed scan could not take.
type RootScanFailure struct {
	Path string `json:"path"`
	Code string `json:"code"`
}

// RootScanRun is the progress, then the report, of a scan of the Racine.
type RootScanRun struct {
	ID         int64      `json:"id"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`
	// Walking is true while the disk is still being walked.
	Walking  bool `json:"walking"`
	Seen     int  `json:"seen"`
	Queued   int  `json:"queued"`
	Done     int  `json:"done"`
	Pending  int  `json:"pending"`
	Failed   int  `json:"failed"`
	UpToDate int  `json:"up_to_date"`
	// WaitingForWorker is true when queued videos wait and no Worker is online.
	WaitingForWorker  bool              `json:"waiting_for_worker"`
	Excluded          RootScanExcluded  `json:"excluded"`
	UnreadableFolders int               `json:"unreadable_folders"`
	UnreadablePaths   []string          `json:"unreadable_paths"`
	Failures          []RootScanFailure `json:"failures"`
}

// IsVideoName tells whether a file name has the extension of a video the library takes.
func IsVideoName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp4", ".mkv", ".mov", ".avi", ".webm", ".m4v", ".ts":
		return true
	}
	return false
}

// rootVideo is one eligible video met by the walk.
type rootVideo struct {
	rel      string
	size     int64
	modified time.Time
}

// walkRoot visits every eligible video of the Racine in lexical order and counts what it leaves out.
// An unreadable folder is counted and skipped, never fatal. It never opens a Dossier privé, a hidden
// folder or a NAS system folder, and never follows a symbolic link.
func walkRoot(ctx context.Context, root string, private map[string]bool, tally *RootScanTally, visit func(rootVideo) error) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if err != nil {
			// WalkDir reports a folder it cannot read after visiting it: count it, keep the rest.
			if entry == nil || entry.IsDir() {
				tally.UnreadableFolders++
				if rel == "." {
					rel = ""
				}
				if len(tally.UnreadablePaths) < rootScanUnreadableNames {
					tally.UnreadablePaths = append(tally.UnreadablePaths, rel)
				}
			}
			return nil
		}
		if rel == "." {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			switch {
			case IsSystemDirectory(entry.Name()):
				tally.Excluded.System++
			case strings.HasPrefix(entry.Name(), "."):
				tally.Excluded.Hidden++
			case !strings.Contains(rel, "/") && private[rel]:
				tally.Excluded.PrivateFolders++
			default:
				return nil
			}
			return filepath.SkipDir
		}
		if !entry.Type().IsRegular() || !IsVideoName(entry.Name()) {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			tally.Excluded.Hidden++
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil
		}
		return visit(rootVideo{rel: rel, size: info.Size(), modified: info.ModTime().UTC().Truncate(time.Microsecond)})
	})
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// processedClip is true for a clip whose analysis was already asked for: an analysis run or a transcription
// or voice job outside the benchmarks. Such a clip is up to date for a scan; re-analysis is a separate action.
const processedClip = `(EXISTS(SELECT 1 FROM clip_analysis_runs a WHERE a.clip_id=c.id) OR EXISTS(SELECT 1 FROM jobs j WHERE j.clip_id=c.id AND j.requested_model_id IS NULL))`

// classifyRootVideos splits a batch into the videos to queue and those left as they are. A video
// already tagged by this run (a walk started over) is neither queued again nor counted twice.
func classifyRootVideos(ctx context.Context, q querier, sourceID, runID int64, batch []rootVideo, tally *RootScanTally) ([]rootVideo, error) {
	paths := make([]string, len(batch))
	for i, v := range batch {
		paths[i] = v.rel
	}
	rows, err := q.Query(ctx, `SELECT sf.relative_path,COALESCE(sf.index_run_id,0),sf.size_bytes,sf.modified_at,sf.status,
		c.id IS NOT NULL,COALESCE(c.visibility='private',false),c.id IS NOT NULL AND `+processedClip+`
		FROM source_files sf LEFT JOIN clips c ON c.id=sf.clip_id WHERE sf.source_id=$1 AND sf.relative_path=ANY($2)`, sourceID, paths)
	if err != nil {
		return nil, err
	}
	type known struct {
		run                         int64
		size                        int64
		modified                    time.Time
		status                      string
		clip, privateClip, analysed bool
	}
	found := map[string]known{}
	for rows.Next() {
		var p string
		var k known
		if err := rows.Scan(&p, &k.run, &k.size, &k.modified, &k.status, &k.clip, &k.privateClip, &k.analysed); err != nil {
			rows.Close()
			return nil, err
		}
		found[p] = k
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var queue []rootVideo
	for _, v := range batch {
		tally.Seen++
		k, ok := found[v.rel]
		if ok && runID != 0 && k.run == runID {
			continue
		}
		// An unchanged, indexed file is judged by its clip; a changed or new file is new content.
		unchanged := ok && k.clip && k.status == "ready" && k.size == v.size && k.modified.Equal(v.modified)
		switch {
		case unchanged && k.privateClip:
			tally.Excluded.PrivateClips++
		case unchanged && k.analysed:
			tally.UpToDate++
		default:
			tally.ToQueue++
			queue = append(queue, v)
		}
	}
	return queue, nil
}

// PreviewRootScan counts what a scan of the whole Racine would do, without queuing anything. It reads
// the folders, never the videos.
func (s *Service) PreviewRootScan(ctx context.Context, source Source) (RootScanTally, error) {
	private, err := s.privateFolders(ctx, source.ID)
	if err != nil {
		return RootScanTally{}, err
	}
	tally := RootScanTally{UnreadablePaths: []string{}}
	var batch []rootVideo
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		_, err := classifyRootVideos(ctx, s.pool, source.ID, 0, batch, &tally)
		batch = batch[:0]
		return err
	}
	err = walkRoot(ctx, source.Path, private, &tally, func(v rootVideo) error {
		batch = append(batch, v)
		if len(batch) >= rootScanBatch {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	return tally, err
}

func (s *Service) privateFolders(ctx context.Context, sourceID int64) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT folder FROM private_folders WHERE source_id=$1`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var folder string
		if err := rows.Scan(&folder); err != nil {
			return nil, err
		}
		out[folder] = true
	}
	return out, rows.Err()
}

// StartRootScan records a scan of the whole Racine and starts its walk in the background. The run is
// durable: Start resumes its walk after a restart.
func (s *Service) StartRootScan(ctx context.Context, sourceID, createdBy int64) (int64, error) {
	var by *int64
	if createdBy != 0 {
		by = &createdBy
	}
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO library_index_runs(source_id,mode,created_by,walking) VALUES($1,'scan',$2,true) RETURNING id`, sourceID, by).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return 0, ErrRootScanActive
	}
	if err != nil {
		return 0, err
	}
	s.walkInBackground(id)
	return id, nil
}

// ResumeRootScans starts again the walk of every scan a stop interrupted.
func (s *Service) ResumeRootScans(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT id FROM library_index_runs WHERE mode='scan' AND walking ORDER BY id`)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return err
	}
	for _, id := range ids {
		s.walkInBackground(id)
	}
	return nil
}

// walkInBackground runs the walk of a scan once per process, retrying slowly after a failure.
func (s *Service) walkInBackground(id int64) {
	s.mu.Lock()
	if s.closed || s.walks[id] {
		s.mu.Unlock()
		return
	}
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	s.walks[id] = true
	s.walkers.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.walkers.Done()
		defer func() {
			s.mu.Lock()
			delete(s.walks, id)
			s.mu.Unlock()
		}()
		for {
			err := s.RunRootScan(ctx, id)
			if err == nil || ctx.Err() != nil {
				return
			}
			if s.logger != nil {
				s.logger.Warn("scan of the Racine interrupted; it will start over", "run", id, "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
			}
		}
	}()
}

var errRootScanStopped = errors.New("scan run no longer walking")

// RunRootScan walks the Racine for a scan run and queues its videos, batch by batch. It starts over from
// the beginning: queuing is idempotent and the counters are rewritten. It returns nil once the walk is
// over or the run no longer walks (cancelled).
func (s *Service) RunRootScan(ctx context.Context, id int64) error {
	var sourceID int64
	var walking bool
	err := s.pool.QueryRow(ctx, `SELECT source_id,walking FROM library_index_runs WHERE id=$1 AND mode='scan'`, id).Scan(&sourceID, &walking)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && !walking {
		return nil
	}
	if err != nil {
		return err
	}
	source, err := s.GetSource(ctx, sourceID)
	if err != nil {
		return err
	}
	private, err := s.privateFolders(ctx, sourceID)
	if err != nil {
		return err
	}
	tally := RootScanTally{UnreadablePaths: []string{}}
	var batch []rootVideo
	flush := func(last bool) error {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		// The run row is locked so a cancellation and a batch never cross.
		if err = tx.QueryRow(ctx, `SELECT walking FROM library_index_runs WHERE id=$1 FOR UPDATE`, id).Scan(&walking); err != nil {
			return err
		}
		if !walking {
			return errRootScanStopped
		}
		queue, err := classifyRootVideos(ctx, tx, sourceID, id, batch, &tally)
		if err != nil {
			return err
		}
		batch = batch[:0]
		if len(queue) > 0 {
			paths := make([]string, len(queue))
			sizes := make([]int64, len(queue))
			modified := make([]time.Time, len(queue))
			for i, v := range queue {
				paths[i], sizes[i], modified[i] = v.rel, v.size, v.modified
			}
			if _, err = tx.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,size_bytes,modified_at,stable_since,last_seen_generation,status,scan_requested,index_only,index_run_id)
				SELECT $1,p,sz,m,now(),0,'waiting_stability',true,false,$5 FROM unnest($2::text[],$3::bigint[],$4::timestamptz[]) AS t(p,sz,m)
				ON CONFLICT(source_id,relative_path) DO UPDATE SET scan_requested=true,index_only=false,index_run_id=EXCLUDED.index_run_id,error_code=NULL`,
				sourceID, paths, sizes, modified, id); err != nil {
				return err
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE library_index_runs SET seen=$2,up_to_date=$3,excluded_private_folders=$4,excluded_hidden=$5,excluded_system=$6,excluded_private_clips=$7,unreadable_folders=$8,unreadable_paths=$9,walking=$10 WHERE id=$1`,
			id, tally.Seen, tally.UpToDate, tally.Excluded.PrivateFolders, tally.Excluded.Hidden, tally.Excluded.System, tally.Excluded.PrivateClips, tally.UnreadableFolders, tally.UnreadablePaths, !last); err != nil {
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		if len(queue) > 0 || last {
			s.ScheduleScan(sourceID, 0)
		}
		return nil
	}
	err = walkRoot(ctx, source.Path, private, &tally, func(v rootVideo) error {
		batch = append(batch, v)
		if len(batch) >= rootScanBatch {
			return flush(false)
		}
		return nil
	})
	if err == nil {
		err = flush(true)
	}
	if errors.Is(err, errRootScanStopped) {
		return nil
	}
	return err
}

// CancelRootScan stops a scan: its walk ends and the videos it queued that were not taken yet are no
// longer asked for. What the managed scan already did is kept.
func (s *Service) CancelRootScan(ctx context.Context, id int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM library_index_runs WHERE id=$1 AND mode='scan' FOR UPDATE`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != "running" {
		return ErrRootScanState
	}
	if _, err = tx.Exec(ctx, `UPDATE library_index_runs SET status='cancelled',walking=false,finished_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE source_files SET scan_requested=false WHERE index_run_id=$1 AND scan_requested`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CurrentRootScan returns the latest scan of the Racine, or nil when none was ever asked. hidden, when
// set, keeps paths a viewer may not read out of the failures (a folder made private after the scan).
func (s *Service) CurrentRootScan(ctx context.Context, sourceID int64, hidden func(string) bool) (*RootScanRun, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM library_index_runs WHERE source_id=$1 AND mode='scan' ORDER BY id DESC LIMIT 1`, sourceID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	run, err := s.RootScan(ctx, id, hidden)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// RootScan reads a scan run. A running scan whose walk is over and whose videos were all taken is
// completed here, once: the managed scan is its only worker.
func (s *Service) RootScan(ctx context.Context, id int64, hidden func(string) bool) (RootScanRun, error) {
	var v RootScanRun
	err := s.pool.QueryRow(ctx, `SELECT id,status,created_at,finished_at,walking,seen,up_to_date,excluded_private_folders,excluded_hidden,excluded_system,excluded_private_clips,unreadable_folders,unreadable_paths
		FROM library_index_runs WHERE id=$1 AND mode='scan'`, id).Scan(&v.ID, &v.Status, &v.CreatedAt, &v.FinishedAt, &v.Walking, &v.Seen, &v.UpToDate,
		&v.Excluded.PrivateFolders, &v.Excluded.Hidden, &v.Excluded.System, &v.Excluded.PrivateClips, &v.UnreadableFolders, &v.UnreadablePaths)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	if err = s.pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE scan_requested),
		count(*) FILTER (WHERE NOT scan_requested AND (error_code IS NOT NULL OR (clip_id IS NULL AND status IN ('missing','unreadable'))))
		FROM source_files WHERE index_run_id=$1`, id).Scan(&v.Queued, &v.Pending, &v.Failed); err != nil {
		return v, err
	}
	v.Done = v.Queued - v.Pending
	if v.Status == "running" && !v.Walking && v.Pending == 0 {
		var finished time.Time
		err = s.pool.QueryRow(ctx, `UPDATE library_index_runs SET status='completed',finished_at=now() WHERE id=$1 AND status='running' AND NOT walking RETURNING finished_at`, id).Scan(&finished)
		if err == nil {
			v.Status, v.FinishedAt = "completed", &finished
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return v, err
		}
	}
	if v.Status == "running" && v.Pending > 0 {
		if err = s.pool.QueryRow(ctx, `SELECT NOT EXISTS(SELECT 1 FROM workers WHERE revoked_at IS NULL AND last_seen_at > now()-interval '2 minutes')`).Scan(&v.WaitingForWorker); err != nil {
			return v, err
		}
	}
	v.Failures = []RootScanFailure{}
	rows, err := s.pool.Query(ctx, `SELECT relative_path,COALESCE(error_code,status) FROM source_files WHERE index_run_id=$1 AND (error_code IS NOT NULL OR (clip_id IS NULL AND status IN ('missing','unreadable'))) ORDER BY relative_path LIMIT $2`, id, rootScanFailureLimit)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		var f RootScanFailure
		if err := rows.Scan(&f.Path, &f.Code); err != nil {
			return v, err
		}
		if hidden != nil && hidden(f.Path) {
			continue
		}
		v.Failures = append(v.Failures, f)
	}
	if v.UnreadablePaths == nil {
		v.UnreadablePaths = []string{}
	}
	return v, rows.Err()
}
