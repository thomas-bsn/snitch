package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jackc/pgx/v5"
	"snitch/internal/library"
)

type deletionFile struct{ root, name, staged, phase string }

// deleteClip journals each file before touching it, so a later request can resume
// after a crash or a partial failure. The advisory lock serializes attempts.
func (s *Server) deleteClip(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_clip_id")
		return
	}
	ctx := r.Context()
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer conn.Release()
	var locked bool
	err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended('clip-deletion:' || $1::bigint::text, 0))`, id).Scan(&locked)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !locked {
		fail(w, 409, "clip_delete_failed")
		return
	}
	defer conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock(hashtextextended('clip-deletion:' || $1::bigint::text, 0))`, id)
	tx, err := conn.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	var digest []byte
	err = tx.QueryRow(ctx, `SELECT content_sha256 FROM clips WHERE id=$1 FOR UPDATE`, id).Scan(&digest)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "clip_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if len(digest) != sha256.Size {
		fail(w, 409, "clip_delete_failed")
		return
	}
	{
		var missing bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=$1 AND sf.status='missing' AND NOT EXISTS (SELECT 1 FROM clip_deletion_files d WHERE d.clip_id=sf.clip_id AND d.root_path=cs.root_path AND d.relative_path=sf.relative_path))`, id).Scan(&missing); err != nil {
			s.internal(w, err)
			return
		}
		if missing {
			fail(w, 409, "clip_delete_failed")
			return
		}
		rows, e := tx.Query(ctx, `SELECT cs.root_path,sf.relative_path FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=$1 UNION SELECT $2,um.storage_key || '.media' FROM uploaded_media um WHERE um.clip_id=$1`, id, s.Uploads.MediaRoot())
		if e != nil {
			s.internal(w, e)
			return
		}
		var files []deletionFile
		for rows.Next() {
			var f deletionFile
			if e = rows.Scan(&f.root, &f.name); e != nil {
				break
			}
			files = append(files, f)
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			s.internal(w, e)
			return
		}
		for _, f := range files {
			var nonce [16]byte
			if _, e = rand.Read(nonce[:]); e != nil {
				s.internal(w, e)
				return
			}
			staged := filepath.Join(filepath.Dir(filepath.FromSlash(f.name)), ".snitch-delete-"+hex.EncodeToString(nonce[:]))
			if _, e = tx.Exec(ctx, `INSERT INTO clip_deletion_files VALUES($1,$2,$3,$4,'planned') ON CONFLICT DO NOTHING`, id, f.root, f.name, staged); e != nil {
				s.internal(w, e)
				return
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	rows, err := conn.Query(ctx, `SELECT root_path,relative_path,staged_path,phase FROM clip_deletion_files WHERE clip_id=$1 ORDER BY root_path,relative_path`, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	var files []deletionFile
	for rows.Next() {
		var f deletionFile
		if err = rows.Scan(&f.root, &f.name, &f.staged, &f.phase); err != nil {
			break
		}
		files = append(files, f)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		s.internal(w, err)
		return
	}
	setPhase := func(f deletionFile, phase string) error {
		_, e := conn.Exec(ctx, `UPDATE clip_deletion_files SET phase=$4 WHERE clip_id=$1 AND root_path=$2 AND relative_path=$3`, id, f.root, f.name, phase)
		return e
	}
	for _, f := range files {
		if f.phase == "removed" {
			continue
		}
		root, e := os.OpenRoot(f.root)
		if e != nil {
			fail(w, 409, "clip_delete_failed")
			return
		}
		name := filepath.FromSlash(f.name)
		if f.phase == "planned" {
			if _, e = library.ManagedPath(f.root, f.name); e != nil {
				root.Close()
				fail(w, 409, "clip_delete_failed")
				return
			}
			if _, e = root.Lstat(f.staged); !errors.Is(e, os.ErrNotExist) {
				root.Close()
				fail(w, 409, "clip_delete_failed")
				return
			}
			if e = setPhase(f, "moving"); e != nil {
				root.Close()
				s.internal(w, e)
				return
			}
			f.phase = "moving"
		}
		if f.phase == "moving" {
			_, oldErr := root.Lstat(name)
			_, stagedErr := root.Lstat(f.staged)
			if oldErr == nil && errors.Is(stagedErr, os.ErrNotExist) {
				e = root.Rename(name, f.staged)
			} else if errors.Is(oldErr, os.ErrNotExist) && stagedErr == nil {
				e = nil
			} else {
				e = os.ErrInvalid
			}
			if e != nil {
				root.Close()
				fail(w, 409, "clip_delete_failed")
				return
			}
			info, e := root.Lstat(f.staged)
			if e != nil || !info.Mode().IsRegular() {
				if _, originalErr := root.Lstat(name); errors.Is(originalErr, os.ErrNotExist) {
					if root.Rename(f.staged, name) == nil {
						_ = setPhase(f, "planned")
					}
				}
				root.Close()
				fail(w, 409, "clip_delete_failed")
				return
			}
			handle, e := root.Open(f.staged)
			if e != nil {
				root.Close()
				fail(w, 409, "clip_delete_failed")
				return
			}
			info, e = handle.Stat()
			hash := sha256.New()
			if e == nil && info.Mode().IsRegular() {
				_, e = io.Copy(hash, handle)
			} else if e == nil {
				e = os.ErrInvalid
			}
			handle.Close()
			if e != nil || !bytes.Equal(hash.Sum(nil), digest) {
				if _, oldErr = root.Lstat(name); errors.Is(oldErr, os.ErrNotExist) {
					if root.Rename(f.staged, name) == nil {
						_ = setPhase(f, "planned")
					}
				}
				root.Close()
				fail(w, 409, "clip_delete_failed")
				return
			}
			if e = setPhase(f, "removing"); e != nil {
				root.Close()
				s.internal(w, e)
				return
			}
			f.phase = "removing"
		}
		if f.phase == "removing" {
			e = root.Remove(f.staged)
			if e != nil && !errors.Is(e, os.ErrNotExist) {
				root.Close()
				fail(w, 409, "clip_delete_failed")
				return
			}
			if e = setPhase(f, "removed"); e != nil {
				root.Close()
				s.internal(w, e)
				return
			}
		}
		root.Close()
	}
	tx, err = conn.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	// Publication serializes on the same path lock before touching the disk.
	// Files are sorted by root and path, so simultaneous deletions take them in order.
	for _, f := range files {
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, f.root+"/"+f.name); err != nil {
			s.internal(w, err)
			return
		}
	}
	// Lock the clip and its current provenance before checking the journal. New
	// references require a foreign-key lock on the clip; existing rows are locked
	// against concurrent reconciliation or publication.
	var lockedID int64
	if err = tx.QueryRow(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, id).Scan(&lockedID); err != nil {
		s.internal(w, err)
		return
	}
	for _, q := range []string{`SELECT source_id FROM source_files WHERE clip_id=$1 FOR UPDATE`, `SELECT upload_id FROM uploaded_media WHERE clip_id=$1 FOR UPDATE`} {
		r, e := tx.Query(ctx, q, id)
		if e != nil {
			s.internal(w, e)
			return
		}
		for r.Next() {
		}
		e = r.Err()
		r.Close()
		if e != nil {
			s.internal(w, e)
			return
		}
	}
	var unplanned bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM (SELECT cs.root_path,sf.relative_path FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=$1 UNION SELECT $2,um.storage_key || '.media' FROM uploaded_media um WHERE um.clip_id=$1) f WHERE NOT EXISTS(SELECT 1 FROM clip_deletion_files d WHERE d.clip_id=$1 AND d.root_path=f.root_path AND d.relative_path=f.relative_path AND d.phase='removed'))`, id, s.Uploads.MediaRoot()).Scan(&unplanned)
	if err != nil {
		s.internal(w, err)
		return
	}
	if unplanned {
		fail(w, 409, "clip_delete_failed")
		return
	}
	// A path can have been repopulated since its journal entry was removed.
	// Replan it rather than claiming that the new copy was deleted.
	for _, f := range files {
		root, e := os.OpenRoot(f.root)
		if e != nil {
			fail(w, 409, "clip_delete_failed")
			return
		}
		_, e = root.Lstat(filepath.FromSlash(f.name))
		root.Close()
		if e == nil {
			if _, err = tx.Exec(ctx, `UPDATE clip_deletion_files SET phase='planned' WHERE clip_id=$1 AND root_path=$2 AND relative_path=$3`, id, f.root, f.name); err != nil {
				s.internal(w, err)
				return
			}
			if err = tx.Commit(ctx); err != nil {
				s.internal(w, err)
				return
			}
			fail(w, 409, "clip_delete_failed")
			return
		}
		if !errors.Is(e, os.ErrNotExist) {
			fail(w, 409, "clip_delete_failed")
			return
		}
	}
	for _, query := range []string{`DELETE FROM source_files WHERE clip_id=$1`, `DELETE FROM uploaded_media WHERE clip_id=$1`, `DELETE FROM uploads WHERE clip_id=$1`, `DELETE FROM clips WHERE id=$1`} {
		if _, err = tx.Exec(ctx, query, id); err != nil {
			s.internal(w, err)
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]string{"status": "deleted"})
}
