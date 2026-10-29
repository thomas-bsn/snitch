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
	for _, relative := range paths {
		absolute, err := ManagedPath(source.Path, relative)
		if err != nil {
			_, _ = s.pool.Exec(ctx, `UPDATE source_files SET status='missing',scan_requested=false,error_code='video_unreadable' WHERE source_id=$1 AND relative_path=$2`, source.ID, relative)
			continue
		}
		info, err := os.Stat(absolute)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		pending, unreadable, err := s.observe(ctx, source, relative, absolute, info, 0)
		if err != nil {
			return err
		}
		if !pending {
			if !unreadable && s.OnManagedReady != nil {
				var clipID int64
				if e := s.pool.QueryRow(ctx, `SELECT clip_id FROM source_files WHERE source_id=$1 AND relative_path=$2`, source.ID, relative).Scan(&clipID); e != nil {
					return e
				}
				if e := s.OnManagedReady(ctx, clipID); e != nil {
					continue
				}
			}
			_, err = s.pool.Exec(ctx, `UPDATE source_files SET scan_requested=false WHERE source_id=$1 AND relative_path=$2`, source.ID, relative)
			if err != nil {
				return err
			}
			if unreadable {
				continue
			}
		}
	}
	if len(paths) > 0 {
		s.ScheduleScan(source.ID, 3*time.Second)
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
		// Changing the location does not move files or silently rebind indexed paths.
		return Source{}, ErrSourceExists
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
