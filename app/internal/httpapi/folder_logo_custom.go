package httpapi

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"snitch/internal/library"
)

// A custom logo for a folder of the Racine vidéo (#323): an administrator uploads a PNG or JPEG, keeps the
// last four images, and may restore an older one or go back to the automatic Steam cover. The image is
// identified by the source and the folder path, never by the folder name, and is served locally: no call
// leaves the instance. Changing a logo of a Dossier privé is reserved to the Super-administrateur, like
// browsing that folder.

const folderLogoHistoryDepth = 4

var errInvalidFolderLogo = errors.New("invalid folder logo")

type folderLogoVersion struct {
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

type folderLogoState struct {
	Path     string              `json:"path"`
	Custom   bool                `json:"custom"`
	Versions []folderLogoVersion `json:"versions"`
}

// normalizeFolderLogo checks that the bytes are a complete PNG or JPEG image and keeps them untouched, so
// that the uploaded cover keeps its quality. The size limit is applied before this function.
func normalizeFolderLogo(data []byte) ([]byte, string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 16_000_000 {
		return nil, "", errInvalidFolderLogo
	}
	if _, _, err = image.Decode(bytes.NewReader(data)); err != nil {
		return nil, "", errInvalidFolderLogo
	}
	if format == "jpeg" {
		return data, "image/jpeg", nil
	}
	return data, "image/png", nil
}

// folderLogoTarget validates a folder path for a logo write: the root, the folder on disk, and the rights
// the caller needs. A Dossier privé is readable only by the Super-administrateur, so only they may change
// its logo. The answer is already written when ok is false.
func (s *Server) folderLogoTarget(w http.ResponseWriter, r *http.Request, rawPath string) (library.Source, string, bool) {
	relative, ok := folderLogoPath(rawPath)
	if !ok {
		fail(w, 422, "video_path_unavailable")
		return library.Source{}, "", false
	}
	ss := r.Context().Value(sessionKey{}).(session)
	if !isAdministrator(ss.User.Role) {
		fail(w, 403, "forbidden")
		return library.Source{}, "", false
	}
	root, err := s.Library.ManagedRoot(r.Context())
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 409, "video_root_unconfigured")
		return library.Source{}, "", false
	}
	if err != nil {
		s.internal(w, err)
		return library.Source{}, "", false
	}
	absolute, err := library.ManagedPath(root.Path, relative)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return library.Source{}, "", false
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.IsDir() {
		fail(w, 422, "video_path_unavailable")
		return library.Source{}, "", false
	}
	if !s.folderLogoFolderAllowed(w, r, root.ID, relative) {
		return library.Source{}, "", false
	}
	return root, relative, true
}

// folderLogoFolderAllowed applies the Dossiers privés rule (ADR 0026) to a folder logo.
func (s *Server) folderLogoFolderAllowed(w http.ResponseWriter, r *http.Request, sourceID int64, relative string) bool {
	ss := r.Context().Value(sessionKey{}).(session)
	private, err := s.privateTopFolders(r, sourceID)
	if err != nil {
		s.internal(w, err)
		return false
	}
	if !isSuperAdministrator(ss.User.Role) && private[firstSegment(relative)] {
		fail(w, 403, "forbidden")
		return false
	}
	return true
}

// folderLogoPath normalizes a folder path relative to the Racine vidéo. The root itself has no logo.
func folderLogoPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || filepath.IsAbs(raw) || strings.Contains(raw, "\\") {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(raw))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

// tryCustomFolderLogo serves a stored custom logo before the automatic Steam cover. It answers false when
// the request carries no path, or when the folder has no active custom logo: the automatic cover is then
// looked up as before. A version parameter serves one precise image of the history.
func (s *Server) tryCustomFolderLogo(w http.ResponseWriter, r *http.Request) bool {
	relative, ok := folderLogoPath(r.URL.Query().Get("path"))
	if !ok {
		if r.URL.Query().Get("path") == "" {
			return false
		}
		fail(w, 422, "video_path_unavailable")
		return true
	}
	root, err := s.Library.ManagedRoot(r.Context())
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	if err != nil {
		s.internal(w, err)
		return true
	}
	if _, err = library.ManagedPath(root.Path, relative); err != nil {
		return false
	}
	if !s.folderLogoFolderAllowed(w, r, root.ID, relative) {
		return true
	}
	var contentType string
	var imageBytes []byte
	if rawVersion := r.URL.Query().Get("version"); rawVersion != "" {
		version, valid := parsePositive(rawVersion)
		if !valid {
			fail(w, 400, "invalid_request")
			return true
		}
		contentType, imageBytes, err = s.folderLogoVersionImage(r.Context(), root.ID, relative, version)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 404, "folder_logo_not_found")
			return true
		}
		if err != nil {
			s.internal(w, err)
			return true
		}
	} else {
		contentType, imageBytes, err = s.currentFolderLogo(r.Context(), root.ID, relative)
		if errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		if err != nil {
			s.internal(w, err)
			return true
		}
	}
	s.writeCustomFolderLogo(w, contentType, imageBytes)
	return true
}

func (s *Server) folderLogoVersionImage(ctx context.Context, sourceID int64, relative string, version int64) (string, []byte, error) {
	var contentType string
	var imageBytes []byte
	err := s.Pool.QueryRow(ctx, `SELECT content_type,image FROM folder_logo_versions WHERE source_id=$1 AND relative_path=$2 AND version=$3`, sourceID, relative, version).Scan(&contentType, &imageBytes)
	return contentType, imageBytes, err
}

// currentFolderLogo reads the image shown for a folder, only while the custom logo is the chosen one.
func (s *Server) currentFolderLogo(ctx context.Context, sourceID int64, relative string) (string, []byte, error) {
	var contentType string
	var imageBytes []byte
	err := s.Pool.QueryRow(ctx, `SELECT v.content_type,v.image FROM folder_logo_choices c
	  JOIN folder_logo_versions v ON v.source_id=c.source_id AND v.relative_path=c.relative_path
	  WHERE c.source_id=$1 AND c.relative_path=$2 AND c.use_custom
	  ORDER BY v.version DESC LIMIT 1`, sourceID, relative).Scan(&contentType, &imageBytes)
	return contentType, imageBytes, err
}

func (s *Server) writeCustomFolderLogo(w http.ResponseWriter, contentType string, imageBytes []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(imageBytes)
}

// putFolderLogo stores an uploaded image as the new current logo. A PNG or JPEG up to 2 MB is accepted.
func (s *Server) putFolderLogo(w http.ResponseWriter, r *http.Request) {
	root, relative, ok := s.folderLogoTarget(w, r, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, folderLogoMaxBytes+1))
	if err != nil || len(data) > folderLogoMaxBytes {
		fail(w, 413, "folder_logo_too_large")
		return
	}
	imageBytes, contentType, err := normalizeFolderLogo(data)
	if errors.Is(err, errInvalidFolderLogo) {
		fail(w, 400, "invalid_folder_logo")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if _, err = s.storeFolderLogo(r.Context(), root.ID, relative, imageBytes, contentType); err != nil {
		s.internal(w, err)
		return
	}
	s.replyFolderLogoState(w, r, root.ID, relative)
}

// clearFolderLogo returns to the automatic Steam cover, keeping the history for a later restore.
func (s *Server) clearFolderLogo(w http.ResponseWriter, r *http.Request) {
	root, relative, ok := s.folderLogoTarget(w, r, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	if _, err := s.Pool.Exec(r.Context(), `UPDATE folder_logo_choices SET use_custom=false,updated_at=now() WHERE source_id=$1 AND relative_path=$2`, root.ID, relative); err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) folderLogoHistory(w http.ResponseWriter, r *http.Request) {
	root, relative, ok := s.folderLogoTarget(w, r, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	s.replyFolderLogoState(w, r, root.ID, relative)
}

// restoreFolderLogo makes an older image the current one by copying it into a new version, as Discord does.
func (s *Server) restoreFolderLogo(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Path    string `json:"path"`
		Version int64  `json:"version"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Version <= 0 {
		fail(w, 400, "invalid_request")
		return
	}
	root, relative, ok := s.folderLogoTarget(w, r, input.Path)
	if !ok {
		return
	}
	contentType, imageBytes, err := s.folderLogoVersionImage(r.Context(), root.ID, relative, input.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "folder_logo_version_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if _, err = s.storeFolderLogo(r.Context(), root.ID, relative, imageBytes, contentType); err != nil {
		s.internal(w, err)
		return
	}
	s.replyFolderLogoState(w, r, root.ID, relative)
}

func (s *Server) replyFolderLogoState(w http.ResponseWriter, r *http.Request, sourceID int64, relative string) {
	state, err := s.folderLogoState(r.Context(), sourceID, relative)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, state)
}

// storeFolderLogo appends a version under the folder's lock and trims the history to four images.
func (s *Server) storeFolderLogo(ctx context.Context, sourceID int64, relative string, imageBytes []byte, contentType string) (int64, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	// The upsert locks the choice row, so two uploads of the same folder cannot produce the same version.
	if _, err = tx.Exec(ctx, `INSERT INTO folder_logo_choices(source_id,relative_path,use_custom) VALUES($1,$2,true)
	  ON CONFLICT(source_id,relative_path) DO UPDATE SET use_custom=true,updated_at=now()`, sourceID, relative); err != nil {
		return 0, err
	}
	var version int64
	if err = tx.QueryRow(ctx, `INSERT INTO folder_logo_versions(source_id,relative_path,version,image,content_type)
	  SELECT $1,$2,COALESCE(MAX(version),0)+1,$3,$4 FROM folder_logo_versions WHERE source_id=$1 AND relative_path=$2
	  RETURNING version`, sourceID, relative, imageBytes, contentType).Scan(&version); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM folder_logo_versions WHERE source_id=$1 AND relative_path=$2 AND version NOT IN (
	  SELECT version FROM folder_logo_versions WHERE source_id=$1 AND relative_path=$2 ORDER BY version DESC LIMIT $3)`, sourceID, relative, folderLogoHistoryDepth); err != nil {
		return 0, err
	}
	return version, tx.Commit(ctx)
}

func (s *Server) folderLogoState(ctx context.Context, sourceID int64, relative string) (folderLogoState, error) {
	state := folderLogoState{Path: relative, Versions: []folderLogoVersion{}}
	err := s.Pool.QueryRow(ctx, `SELECT use_custom FROM folder_logo_choices WHERE source_id=$1 AND relative_path=$2`, sourceID, relative).Scan(&state.Custom)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return state, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT version,created_at FROM folder_logo_versions WHERE source_id=$1 AND relative_path=$2 ORDER BY version DESC LIMIT $3`, sourceID, relative, folderLogoHistoryDepth)
	if err != nil {
		return state, err
	}
	defer rows.Close()
	for rows.Next() {
		var version folderLogoVersion
		if err := rows.Scan(&version.Version, &version.CreatedAt); err != nil {
			return state, err
		}
		state.Versions = append(state.Versions, version)
	}
	return state, rows.Err()
}

// folderLogoVersionsByPath reads the active custom version of the listed folders, so that the explorer
// answers the version its glyph must request and a changed logo breaks the browser cache.
func (s *Server) fillFolderLogoVersions(ctx context.Context, sourceID int64, entries []explorerEntry) error {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Directory {
			paths = append(paths, entry.Path)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT c.relative_path,MAX(v.version) FROM folder_logo_choices c
	  JOIN folder_logo_versions v ON v.source_id=c.source_id AND v.relative_path=c.relative_path
	  WHERE c.source_id=$1 AND c.use_custom AND c.relative_path=ANY($2)
	  GROUP BY c.relative_path`, sourceID, paths)
	if err != nil {
		return err
	}
	defer rows.Close()
	versions := map[string]int64{}
	for rows.Next() {
		var path string
		var version int64
		if err := rows.Scan(&path, &version); err != nil {
			return err
		}
		versions[path] = version
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range entries {
		if version, ok := versions[entries[i].Path]; ok {
			entries[i].LogoVersion = version
		}
	}
	return nil
}
