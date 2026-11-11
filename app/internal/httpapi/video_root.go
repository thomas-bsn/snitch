package httpapi

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"snitch/internal/library"
)

type videoEntry struct {
	Path      string `json:"path"`
	Directory bool   `json:"directory"`
	Size      int64  `json:"size_bytes"`
	// Private marks a top-level folder made private (#93, ADR 0026); Locked is set when this viewer
	// may not open it. Both only ever appear on top-level folders, and nothing inside a locked one is listed.
	Private bool `json:"private"`
	Locked  bool `json:"locked"`
}

// privateTopFolders returns the private top-level folders of a source (#93).
func (s *Server) privateTopFolders(r *http.Request, sourceID int64) (map[string]bool, error) {
	rows, err := s.Pool.Query(r.Context(), `SELECT folder FROM private_folders WHERE source_id=$1`, sourceID)
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

// firstSegment is the top-level folder a root-relative path lies in, or "" for the root itself.
func firstSegment(relative string) string {
	clean := strings.Trim(filepath.ToSlash(filepath.Clean("/"+relative)), "/")
	if i := strings.IndexByte(clean, '/'); i >= 0 {
		return clean[:i]
	}
	return clean
}

func (s *Server) videoRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut {
		var input struct {
			Path string `json:"path"`
		}
		if !decode(w, r, &input) {
			return
		}
		owner := r.Context().Value(sessionKey{}).(session).User.ID
		root, err := s.Library.SetManagedRoot(r.Context(), owner, input.Path)
		if errors.Is(err, library.ErrSourceBusy) {
			fail(w, 409, "video_root_busy")
			return
		}
		if err != nil {
			fail(w, 422, "video_root_unavailable")
			return
		}
		reply(w, 200, root)
		return
	}
	available := s.workerAvailable(r)
	root, err := s.Library.ManagedRoot(r.Context())
	if errors.Is(err, pgx.ErrNoRows) {
		reply(w, 200, map[string]any{"root": nil, "worker_available": available})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]any{"root": root, "worker_available": available})
}

// A connected Worker is the signal that a requested file can move from the
// waiting queue to indexing and transcription.
func (s *Server) workerAvailable(r *http.Request) bool {
	var available bool
	_ = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workers WHERE revoked_at IS NULL AND last_seen_at > now()-interval '2 minutes')`).Scan(&available)
	return available
}
func (s *Server) videoBrowse(w http.ResponseWriter, r *http.Request) {
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		fail(w, 409, "video_root_unconfigured")
		return
	}
	relative := r.URL.Query().Get("path")
	path, err := library.ManagedPath(root.Path, relative)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return
	}
	// ADR 0026: a private folder shows its name and lock to everyone; only the Super-administrator opens it.
	superAdmin := isSuperAdministrator(r.Context().Value(sessionKey{}).(session).User.Role)
	private, err := s.privateTopFolders(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	inside := firstSegment(relative)
	if !superAdmin && private[inside] {
		fail(w, 403, "forbidden")
		return
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return
	}
	out := []videoEntry{}
	for _, entry := range entries {
		if entry.IsDir() && library.IsSystemDirectory(entry.Name()) || strings.HasPrefix(entry.Name(), ".") || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !entry.IsDir() && !library.IsVideoName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		isPrivate := inside == "" && entry.IsDir() && private[entry.Name()]
		out = append(out, videoEntry{filepath.ToSlash(filepath.Join(relative, entry.Name())), entry.IsDir(), info.Size(), isPrivate, isPrivate && !superAdmin})
		if len(out) > 10000 {
			fail(w, 422, "video_selection_too_large")
			return
		}
	}
	reply(w, 200, map[string]any{"items": out})
}
func (s *Server) videoFolder(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Parent string `json:"parent"`
		Name   string `json:"name"`
	}
	if !decode(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.Name) == "" || input.Name == "." || input.Name == ".." || strings.ContainsAny(input.Name, "/\\\x00") || strings.HasPrefix(input.Name, ".") || len(input.Name) > 255 {
		fail(w, 400, "invalid_folder_name")
		return
	}
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		fail(w, 409, "video_root_unconfigured")
		return
	}
	parent, err := library.ManagedPath(root.Path, input.Parent)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return
	}
	superAdmin := isSuperAdministrator(r.Context().Value(sessionKey{}).(session).User.Role)
	private, err := s.privateTopFolders(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !superAdmin && private[firstSegment(input.Parent)] {
		fail(w, 403, "forbidden")
		return
	}
	// os.Root also confines the write if a local process changes a directory to a symlink.
	handle, err := os.OpenRoot(root.Path)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return
	}
	defer handle.Close()
	rel, err := filepath.Rel(root.Path, parent)
	if err != nil {
		fail(w, 400, "invalid_folder_name")
		return
	}
	if err = handle.Mkdir(filepath.Join(rel, input.Name), 0750); err != nil {
		fail(w, 409, "folder_not_created")
		return
	}
	reply(w, 201, map[string]string{"path": filepath.ToSlash(filepath.Join(rel, input.Name))})
}
func (s *Server) videoScan(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Paths   []string `json:"paths"`
		Confirm bool     `json:"confirm"`
	}
	if !decode(w, r, &input) {
		return
	}
	if len(input.Paths) == 0 || len(input.Paths) > 1000 {
		fail(w, 400, "invalid_video_selection")
		return
	}
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		fail(w, 409, "video_root_unconfigured")
		return
	}
	// ADR 0026: only the Super-administrator previews or queues files of a private folder.
	superAdmin := isSuperAdministrator(r.Context().Value(sessionKey{}).(session).User.Role)
	private, err := s.privateTopFolders(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	found := map[string]fs.FileInfo{}
	for _, relative := range input.Paths {
		absolute, e := library.ManagedPath(root.Path, relative)
		if e != nil {
			fail(w, 422, "video_path_unavailable")
			return
		}
		if !superAdmin && private[firstSegment(relative)] {
			fail(w, 403, "forbidden")
			return
		}
		e = library.WalkMediaDir(absolute, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if r.Context().Err() != nil {
				return r.Context().Err()
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			rel, e := filepath.Rel(root.Path, path)
			if e != nil {
				return e
			}
			if !superAdmin && private[firstSegment(rel)] {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if path != absolute && strings.HasPrefix(entry.Name(), ".") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !entry.Type().IsRegular() || !library.IsVideoName(path) {
				return nil
			}
			if _, e = library.ManagedPath(root.Path, rel); e != nil {
				return e
			}
			info, e := entry.Info()
			if e != nil {
				return e
			}
			found[filepath.ToSlash(rel)] = info
			if len(found) > 10000 {
				return errors.New("selection too large")
			}
			return nil
		})
		if e != nil {
			fail(w, 422, "video_scan_incomplete")
			return
		}
	}
	paths := make([]string, 0, len(found))
	for p := range found {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	if input.Confirm {
		tx, e := s.Pool.Begin(r.Context())
		if e != nil {
			s.internal(w, e)
			return
		}
		defer tx.Rollback(r.Context())
		if e = library.RequestSelectionScan(r.Context(), tx, root.ID, found); e != nil {
			s.internal(w, e)
			return
		}
		if e = tx.Commit(r.Context()); e != nil {
			s.internal(w, e)
			return
		}
		s.Library.ScheduleScan(root.ID, 0)
	}
	reply(w, 200, map[string]any{"paths": paths, "queued": input.Confirm})
}
