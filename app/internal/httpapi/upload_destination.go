package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/library"
	"snitch/internal/uploads"
)

func fileVersion(info os.FileInfo) string {
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
}

func (s *Server) uploadDestination(w http.ResponseWriter, r *http.Request, name, folder, expected string) (int64, string, string, bool) {
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		fail(w, 409, "video_root_unconfigured")
		return 0, "", "", false
	}
	if name != filepath.Base(name) || strings.ContainsAny(name, "\\\x00") || strings.HasPrefix(name, ".") {
		fail(w, 400, "invalid_upload")
		return 0, "", "", false
	}
	parent, err := library.ManagedPath(root.Path, folder)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return 0, "", "", false
	}
	if st, e := os.Stat(parent); e != nil || !st.IsDir() {
		fail(w, 422, "video_path_unavailable")
		return 0, "", "", false
	}
	relative := filepath.ToSlash(filepath.Join(folder, name))
	version := "absent"
	info, err := os.Lstat(filepath.Join(parent, name))
	if err == nil {
		if !info.Mode().IsRegular() {
			fail(w, 409, "upload_conflict")
			return 0, "", "", false
		}
		version = fileVersion(info)
		if expected != version {
			reply(w, 409, map[string]string{"error": "overwrite_confirmation_required", "version": version})
			return 0, "", "", false
		}
		// Overwriting a shared library file is an administration operation.
		if !isAdministrator(r.Context().Value(sessionKey{}).(session).User.Role) {
			fail(w, 403, "forbidden")
			return 0, "", "", false
		}
	} else if !os.IsNotExist(err) {
		fail(w, 422, "video_path_unavailable")
		return 0, "", "", false
	}
	return root.ID, relative, version, true
}

// Publish from immutable upload staging. The DB record is a durable intent: after
// a crash, re-running publication recognizes the installed content by SHA-256.
func (s *Server) publishUpload(ctx context.Context, id string, clipID int64) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var source int64
	var root, relative, expected string
	var published bool
	err = tx.QueryRow(ctx, `SELECT d.source_id,cs.root_path,d.relative_path,d.expected_version,d.published FROM upload_destinations d JOIN clip_sources cs ON cs.id=d.source_id WHERE d.upload_id=$1 FOR UPDATE OF d`, id).Scan(&source, &root, &relative, &expected, &published)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if published {
		_, _ = tx.Exec(ctx, `UPDATE upload_destinations SET attempts=0,last_error=NULL WHERE upload_id=$1`, id)
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		_ = os.Remove(filepath.Join(s.Uploads.MediaRoot(), id+".media"))
		return nil
	}
	// Serialize publications to the same destination across app instances.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, root+"/"+relative); err != nil {
		return err
	}
	if _, err = library.ManagedPath(root, filepath.Dir(relative)); err != nil {
		return err
	}
	handle, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer handle.Close()
	var digest []byte
	if err = tx.QueryRow(ctx, `SELECT content_sha256 FROM clips WHERE id=$1`, clipID).Scan(&digest); err != nil {
		return err
	}
	alreadyInstalled := false
	info, statErr := handle.Lstat(relative)
	if statErr == nil {
		if !info.Mode().IsRegular() {
			return uploads.ErrConflict
		}
		if fileVersion(info) != expected {
			f, e := handle.Open(relative)
			if e != nil {
				return e
			}
			h := sha256.New()
			_, e = io.Copy(h, f)
			f.Close()
			if e != nil {
				return e
			}
			alreadyInstalled = hex.EncodeToString(h.Sum(nil)) == hex.EncodeToString(digest)
			if !alreadyInstalled {
				return uploads.ErrConflict
			}
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	} else if expected != "absent" {
		return uploads.ErrConflict
	}
	if !alreadyInstalled {
		staged, err := os.Open(filepath.Join(s.Uploads.MediaRoot(), id+".media"))
		if err != nil {
			return err
		}
		defer staged.Close()
		temp := filepath.Join(filepath.Dir(relative), ".snitch-upload-"+id)
		// A leftover temp is private to this durable upload intent.
		_ = handle.Remove(temp)
		out, err := handle.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, staged)
		syncErr := out.Sync()
		closeErr := out.Close()
		if err = errors.Join(copyErr, syncErr, closeErr); err != nil {
			_ = handle.Remove(temp)
			return err
		}
		latest, e := handle.Lstat(relative)
		if (expected == "absent" && !os.IsNotExist(e)) || (expected != "absent" && (e != nil || !latest.Mode().IsRegular() || fileVersion(latest) != expected)) {
			_ = handle.Remove(temp)
			return uploads.ErrConflict
		}
		if err = handle.Rename(temp, relative); err != nil {
			return err
		}
	}
	info, err = handle.Stat(relative)
	if err != nil {
		return err
	}
	var old *int64
	err = tx.QueryRow(ctx, `SELECT clip_id FROM source_files WHERE source_id=$1 AND relative_path=$2 FOR UPDATE`, source, relative).Scan(&old)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,clip_id,size_bytes,modified_at,stable_since,last_seen_generation,status,scan_requested) VALUES($1,$2,$3,$4,$5,now(),0,'ready',false) ON CONFLICT(source_id,relative_path) DO UPDATE SET clip_id=EXCLUDED.clip_id,size_bytes=EXCLUDED.size_bytes,modified_at=EXCLUDED.modified_at,status='ready',scan_requested=false,error_code=NULL`, source, relative, clipID, info.Size(), info.ModTime().UTC().Truncate(time.Microsecond))
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE upload_destinations SET published=true,attempts=0,last_error=NULL WHERE upload_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM uploaded_media WHERE upload_id=$1`, id); err != nil {
		return err
	}
	if old != nil && *old != clipID {
		if _, err = tx.Exec(ctx, `DELETE FROM uploaded_media WHERE upload_id IN (SELECT upload_id FROM upload_destinations WHERE source_id=$1 AND relative_path=$2 AND upload_id<>$3)`, source, relative, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE uploads SET clip_id=NULL,status='cancelled' WHERE clip_id=$1 AND id IN (SELECT upload_id FROM upload_destinations WHERE source_id=$2 AND relative_path=$3 AND upload_id<>$4)`, *old, source, relative, id); err != nil {
			return err
		}

		// Other copies of the same content keep their clip and results. Only the
		// replaced path is detached; a clip with no remaining location is removed.
		_, err = tx.Exec(ctx, `DELETE FROM clips WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM source_files WHERE clip_id=$1) AND NOT EXISTS(SELECT 1 FROM uploaded_media WHERE clip_id=$1) AND NOT EXISTS(SELECT 1 FROM uploads WHERE clip_id=$1)`, *old)
		if err != nil {
			return err
		}
	}
	if expected != "absent" {
		if _, err = tx.Exec(ctx, `UPDATE source_files SET scan_requested=true WHERE source_id=$1 AND relative_path=$2`, source, relative); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(s.Uploads.MediaRoot(), id+".media"))
	if expected != "absent" {
		s.Library.ScheduleScan(source, 0)
	}
	return nil
}

// Resume the already confirmed uploads after a process interruption. Conflicts
// remain untouched and require a new explicit confirmation from the uploader.
// Five failed attempts stop the automatic loop; the upload stays listed with its
// stable error so an administrator can retry or cancel it explicitly.
func (s *Server) resumeUploadPublications(ctx context.Context) {
	rows, err := s.Pool.Query(ctx, `SELECT u.id,u.clip_id FROM uploads u JOIN upload_destinations d ON d.upload_id=u.id WHERE u.status='complete' AND u.clip_id IS NOT NULL AND NOT d.published AND d.attempts < 5 ORDER BY u.created_at LIMIT 20`)
	if err != nil {
		return
	}
	type publication struct {
		id   string
		clip int64
	}
	items := []publication{}
	for rows.Next() {
		var item publication
		if rows.Scan(&item.id, &item.clip) != nil {
			rows.Close()
			return
		}
		items = append(items, item)
	}
	rows.Close()
	for _, item := range items {
		if ctx.Err() != nil {
			return
		}
		if err := s.publishUpload(ctx, item.id, item.clip); err != nil {
			s.recordPublicationFailure(ctx, item.id, err)
		}
	}
}

// recordPublicationFailure stores a stable code rather than raw text, so the
// interface can translate it and a later explicit retry can replace it.
func (s *Server) recordPublicationFailure(ctx context.Context, id string, err error) {
	if ctx.Err() != nil {
		return
	}
	code := "publication_failed"
	switch {
	case errors.Is(err, uploads.ErrConflict):
		code = "destination_changed"
	case errors.Is(err, os.ErrPermission):
		code = "video_root_permission"
	case errors.Is(err, os.ErrNotExist):
		code = "video_root_unavailable"
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE upload_destinations SET attempts=attempts+1,last_error=$2 WHERE upload_id=$1 AND NOT published`, id, code)
}
