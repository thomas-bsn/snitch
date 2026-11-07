package httpapi

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"snitch/internal/library"
)

// The properties of an explorer entry (#323): the size on disk, the durations and owners the database
// knows, and the dates of a folder or a video. A folder aggregates its whole subtree, bounded like the
// listing; a clip the viewer may not read is never counted, so the numbers follow the same permissions as
// the rows.

type explorerOwner struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Videos int    `json:"videos"`
}

type explorerProperties struct {
	Path         string          `json:"path"`
	Name         string          `json:"name"`
	Directory    bool            `json:"directory"`
	SizeBytes    int64           `json:"size_bytes"`
	VideoCount   int             `json:"video_count"`
	FolderCount  int             `json:"folder_count"`
	IndexedCount int             `json:"indexed_count"`
	Duration     *float64        `json:"duration_seconds"`
	AddedAt      *time.Time      `json:"added_at"`
	RecordedAt   *time.Time      `json:"recorded_at"`
	Game         string          `json:"game"`
	State        string          `json:"state,omitempty"`
	Owners       []explorerOwner `json:"owners"`
}

var errExplorerPropertiesTooLarge = errors.New("explorer properties: too many entries")

func (s *Server) explorerProperties(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if strings.HasPrefix(path, "clip:") {
		s.explorerClipProperties(w, r, strings.TrimPrefix(path, "clip:"))
		return
	}
	root, err := s.Library.ManagedRoot(r.Context())
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 409, "video_root_unconfigured")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	relative, ok := folderLogoPath(path)
	if !ok && path != "" {
		fail(w, 422, "video_path_unavailable")
		return
	}
	absolute, err := library.ManagedPath(root.Path, relative)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return
	}
	info, err := os.Stat(absolute)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	if !s.folderLogoFolderAllowed(w, r, root.ID, relative) {
		return
	}
	if info.IsDir() {
		s.explorerFolderProperties(w, r, root, relative)
		return
	}
	if !info.Mode().IsRegular() || !videoExtension(relative) {
		fail(w, 422, "video_path_unavailable")
		return
	}
	s.explorerFileProperties(w, r, root, relative, info, ss.User.ID)
}

// explorerFolderProperties walks the folder as the listing does and aggregates only the videos the viewer
// may see. A Dossier privé never reaches this function without the Super-administrateur.
func (s *Server) explorerFolderProperties(w http.ResponseWriter, r *http.Request, root library.Source, relative string) {
	base, err := library.ManagedPath(root.Path, relative)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	superAdmin := isSuperAdministrator(ss.User.Role)
	private, err := s.privateTopFolders(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	paths := []string{}
	sizes := map[string]int64{}
	folders := 0
	entries := 0
	err = library.WalkMediaDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if r.Context().Err() != nil {
			return r.Context().Err()
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root.Path, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		// A Dossier privé is counted as one folder, but its contents are invisible to anyone else.
		hiddenFolder := !superAdmin && private[firstSegment(rel)]
		if entry.IsDir() {
			if path != base {
				entries++
				folders++
			}
			if entries > explorerMaxEntries {
				return errExplorerPropertiesTooLarge
			}
			if hiddenFolder {
				return filepath.SkipDir
			}
			return nil
		}
		if hiddenFolder || !entry.Type().IsRegular() || !videoExtension(entry.Name()) {
			return nil
		}
		entries++
		if entries > explorerMaxEntries {
			return errExplorerPropertiesTooLarge
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		paths = append(paths, rel)
		sizes[rel] = info.Size()
		return nil
	})
	if err != nil {
		if errors.Is(err, errExplorerPropertiesTooLarge) {
			fail(w, 422, "video_selection_too_large")
			return
		}
		fail(w, 422, "video_path_unavailable")
		return
	}
	records, err := s.explorerRecords(r, root.ID, paths, ss.User.ID, isAdministrator(ss.User.Role), 0)
	if err != nil {
		s.internal(w, err)
		return
	}
	out := explorerProperties{
		Path:      relative,
		Name:      filepath.Base(relative),
		Directory: true,
		Owners:    []explorerOwner{},
	}
	if relative == "" {
		out.Name = filepath.Base(filepath.Clean(root.Path))
	}
	owners := map[int64]*explorerOwner{}
	var duration float64
	var added *time.Time
	for _, path := range paths {
		rec, found := records[path]
		if found && rec.hidden {
			continue
		}
		out.VideoCount++
		out.SizeBytes += sizes[path]
		if found && rec.clipID != nil && rec.readable {
			out.IndexedCount++
			if rec.duration != nil {
				duration += *rec.duration
			}
			if rec.added != nil && (added == nil || rec.added.Before(*added)) {
				at := *rec.added
				added = &at
			}
			if rec.ownerID != nil {
				owner, ok := owners[*rec.ownerID]
				if !ok {
					owner = &explorerOwner{ID: *rec.ownerID, Name: rec.owner}
					owners[*rec.ownerID] = owner
				}
				owner.Videos++
			}
		}
	}
	if out.IndexedCount > 0 {
		out.Duration = &duration
	}
	out.AddedAt = added
	out.FolderCount = folders
	for _, owner := range owners {
		out.Owners = append(out.Owners, *owner)
	}
	sort.Slice(out.Owners, func(i, j int) bool {
		if out.Owners[i].Videos != out.Owners[j].Videos {
			return out.Owners[i].Videos > out.Owners[j].Videos
		}
		return strings.ToLower(out.Owners[i].Name) < strings.ToLower(out.Owners[j].Name)
	})
	reply(w, 200, out)
}

func (s *Server) explorerFileProperties(w http.ResponseWriter, r *http.Request, root library.Source, relative string, info os.FileInfo, userID int64) {
	ss := r.Context().Value(sessionKey{}).(session)
	records, err := s.explorerRecords(r, root.ID, []string{relative}, userID, isAdministrator(ss.User.Role), 0)
	if err != nil {
		s.internal(w, err)
		return
	}
	rec, found := records[relative]
	if found && rec.hidden {
		fail(w, 404, "clip_not_found")
		return
	}
	state, _ := explorerState(rec, found)
	out := explorerProperties{
		Path:       relative,
		Name:       filepath.Base(relative),
		SizeBytes:  info.Size(),
		VideoCount: 1,
		State:      state,
		Owners:     []explorerOwner{},
	}
	if found && rec.clipID != nil && !rec.readable {
		fail(w, 404, "clip_not_found")
		return
	}
	if found {
		fillExplorerProperties(&out, rec)
	}
	reply(w, 200, out)
}

// explorerClipProperties answers for a clip with no file in the Racine vidéo (clip:<id>).
func (s *Server) explorerClipProperties(w http.ResponseWriter, r *http.Request, rawID string) {
	id, ok := parsePositive(rawID)
	if !ok {
		fail(w, 400, "invalid_request")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	var ownerID int64
	var owner, visibility, title, original, game string
	var size int64
	var duration *float64
	var added, recorded time.Time
	err := s.Pool.QueryRow(r.Context(), `SELECT c.owner_id,u.username,c.visibility,COALESCE(c.display_title,''),c.original_name,c.size_bytes,c.duration_seconds,c.discovered_at,c.recorded_at,COALESCE(c.game,'')
	  FROM clips c JOIN users u ON u.id=c.owner_id
	  WHERE c.id=$1 AND clip_readable(c.id,c.owner_id,c.visibility,$2,$3)`, id, ss.User.ID, isAdministrator(ss.User.Role)).
		Scan(&ownerID, &owner, &visibility, &title, &original, &size, &duration, &added, &recorded, &game)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "clip_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	name := original
	if title != "" {
		name = title
	}
	states, err := s.explorerAnalysis(r, []int64{id})
	if err != nil {
		s.internal(w, err)
		return
	}
	st := states[id]
	state := analysisLabel(st.done, st.latest)
	if state == "transcribed" && st.voicesPending == 0 {
		state = "analyzed"
	}
	out := explorerProperties{
		Path:         fmt.Sprintf("clip:%d", id),
		Name:         name,
		SizeBytes:    size,
		VideoCount:   1,
		IndexedCount: 1,
		Duration:     duration,
		AddedAt:      &added,
		RecordedAt:   &recorded,
		Game:         game,
		State:        state,
		Owners:       []explorerOwner{{ID: ownerID, Name: owner, Videos: 1}},
	}
	reply(w, 200, out)
}

func fillExplorerProperties(out *explorerProperties, rec explorerRecord) {
	out.IndexedCount = 1
	out.Duration = rec.duration
	out.AddedAt = rec.added
	out.RecordedAt = rec.recorded
	out.Game = rec.game
	if rec.ownerID != nil {
		out.Owners = []explorerOwner{{ID: *rec.ownerID, Name: rec.owner, Videos: 1}}
	}
}
