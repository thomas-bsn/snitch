package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Managed roots are browsed only by explicit user requests. Background work may
// inspect only the individual paths the administrator already put in the queue.
func (s *Service) scanManaged(ctx context.Context, source Source) error {
	var requested bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM source_files WHERE source_id=$1 AND scan_requested)`, source.ID).Scan(&requested); err != nil {
		return err
	}
	if !requested {
		return nil
	}
	var available bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workers WHERE revoked_at IS NULL AND last_seen_at > now()-interval '2 minutes')`).Scan(&available); err != nil {
		return err
	}
	if !available {
		s.ScheduleScan(source.ID, 10*time.Second)
		return nil
	}
	rows, err := s.pool.Query(ctx, `SELECT relative_path FROM source_files WHERE source_id=$1 AND scan_requested ORDER BY relative_path LIMIT 50`, source.ID)
	if err != nil {
		return err
	}
	paths := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return err
		}
		paths = append(paths, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var pending, slowRetry, warning bool
	for _, relative := range paths {
		absolute, err := ManagedPath(source.Path, relative)
		if err != nil {
			_, _ = s.pool.Exec(ctx, `UPDATE source_files SET status='missing',scan_requested=false,error_code='video_unreadable' WHERE source_id=$1 AND relative_path=$2`, source.ID, relative)
			warning = true
			continue
		}
		info, err := os.Stat(absolute)
		if err != nil || !info.Mode().IsRegular() {
			code := "video_unreadable"
			if os.IsNotExist(err) {
				code = "video_missing"
			}
			_, _ = s.pool.Exec(ctx, `UPDATE source_files SET status='missing',scan_requested=false,error_code=$3 WHERE source_id=$1 AND relative_path=$2`, source.ID, relative, code)
			warning = true
			continue
		}
		isPending, unreadable, err := s.observe(ctx, source, relative, absolute, info, 0)
		if err != nil {
			return err
		}
		if isPending {
			pending = true
			continue
		}
		if unreadable {
			warning = true
			_, err = s.pool.Exec(ctx, `UPDATE source_files SET scan_requested=false WHERE source_id=$1 AND relative_path=$2`, source.ID, relative)
			if err != nil {
				return err
			}
			continue
		}
		if s.OnManagedReady != nil {
			var clipID int64
			if e := s.pool.QueryRow(ctx, `SELECT clip_id FROM source_files WHERE source_id=$1 AND relative_path=$2`, source.ID, relative).Scan(&clipID); e != nil {
				return e
			}
			if e := s.OnManagedReady(ctx, clipID); e != nil {
				code := "transcription_scheduling_failed"
				var scanErr *ManagedScanError
				if errors.As(e, &scanErr) && scanErr.Code != "" {
					code = scanErr.Code
				}
				_, updateErr := s.pool.Exec(ctx, `UPDATE source_files SET error_code=$3 WHERE source_id=$1 AND relative_path=$2`, source.ID, relative, code)
				if updateErr != nil {
					return updateErr
				}
				// A missing configuration will not fix itself in three seconds:
				// keep the request visible and retry slowly until it is provided.
				slowRetry = true
				warning = true
				continue
			}
		}
		_, err = s.pool.Exec(ctx, `UPDATE source_files SET scan_requested=false,error_code=NULL WHERE source_id=$1 AND relative_path=$2`, source.ID, relative)
		if err != nil {
			return err
		}
	}
	health, message := "ready", ""
	if warning {
		health, message = "warning", "Certains fichiers demandent une vérification ou une configuration avant leur traitement."
	}
	if _, err = s.pool.Exec(ctx, `UPDATE clip_sources SET health=$2,health_message=$3,last_scan_at=now() WHERE id=$1`, source.ID, health, message); err != nil {
		return err
	}
	switch {
	case pending:
		s.ScheduleScan(source.ID, 3*time.Second)
	case slowRetry:
		s.ScheduleScan(source.ID, 30*time.Second)
	case len(paths) == 50:
		s.ScheduleScan(source.ID, time.Second)
	}
	return nil
}

// ManagedPath rejects absolute/traversing paths and every symlink component.
// All callers work under an administrator-selected, already mounted root.
func ManagedPath(root, relative string) (string, error) {
	if relative == "" {
		relative = "."
	}
	if filepath.IsAbs(relative) || strings.Contains(relative, "\\") {
		return "", ErrInvalidSource
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", ErrInvalidSource
	}
	current := root
	for _, part := range strings.Split(clean, string(os.PathSeparator)) {
		current = filepath.Join(current, part)
		st, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return "", ErrInvalidSource
		}
	}
	return current, nil
}

func (s *Service) ManagedRoot(ctx context.Context) (Source, error) {
	var id int64
	if err := s.pool.QueryRow(ctx, `SELECT id FROM clip_sources WHERE managed_root`).Scan(&id); err != nil {
		return Source{}, err
	}
	return s.GetSource(ctx, id)
}

// SetManagedRoot creates the single managed root or changes its location.
// Changing the location never moves files and never rebinds an indexed path to
// another file: every previous entry becomes "missing" and keeps its clip and
// results, while the new root starts with no indexed file until an explicit
// scan. Pending upload publications block the change because their relative
// paths would otherwise silently point into the new root.
func (s *Service) SetManagedRoot(ctx context.Context, owner int64, path string) (Source, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) == string(os.PathSeparator) {
		return Source{}, ErrInvalidSource
	}
	path = filepath.Clean(path)
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return Source{}, ErrUnavailable
	}
	path = real
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return Source{}, ErrUnavailable
	}
	probe, err := os.CreateTemp(path, ".snitch-write-check-")
	if err != nil {
		return Source{}, ErrUnavailable
	}
	probe.Close()
	os.Remove(probe.Name())
	old, err := s.ManagedRoot(ctx)
	if err == nil {
		if old.Path == path {
			return old, nil
		}
		tx, beginErr := s.pool.Begin(ctx)
		if beginErr != nil {
			return Source{}, beginErr
		}
		defer tx.Rollback(ctx)
		var pending int
		if beginErr = tx.QueryRow(ctx, `SELECT count(*) FROM upload_destinations WHERE source_id=$1 AND NOT published`, old.ID).Scan(&pending); beginErr != nil {
			return Source{}, beginErr
		}
		if pending > 0 {
			return Source{}, ErrSourceBusy
		}
		if _, beginErr = tx.Exec(ctx, `UPDATE clip_sources SET root_path=$2,health='ready',health_message='',last_scan_at=NULL WHERE id=$1`, old.ID, path); beginErr != nil {
			return Source{}, beginErr
		}
		if _, beginErr = tx.Exec(ctx, `UPDATE source_files SET status='missing',scan_requested=false,error_code=NULL WHERE source_id=$1 AND status<>'missing'`, old.ID); beginErr != nil {
			return Source{}, beginErr
		}
		if beginErr = tx.Commit(ctx); beginErr != nil {
			return Source{}, beginErr
		}
		return s.GetSource(ctx, old.ID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Source{}, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `INSERT INTO clip_sources(name,root_path,owner_id,managed_root) VALUES('Video library',$1,$2,true) RETURNING id`, path, owner).Scan(&id)
	if err != nil {
		return Source{}, err
	}
	source, err := s.GetSource(ctx, id)
	if err == nil {
		s.ScheduleScan(id, time.Second)
	}
	return source, err
}
