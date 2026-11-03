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
	"snitch/internal/analysis"
	"snitch/internal/library"
)

// The explorer (#195, ADR 0032) lists the Racine vidéo as it is on disk and crosses it with the
// database: each video carries its state, its clip when the viewer may read that clip, and nothing
// else. Listing reads directory entries only, never file contents; no file is indexed unless someone
// opens it (POST /api/library/explorer/index), and that indexing never transcribes it.

const explorerMaxEntries = 10000

// explorerStates are the states a video can carry, in the order the interface lists them.
var explorerStates = []string{"not_indexed", "in_progress", "not_analyzed", "transcribed", "analyzed", "failed"}

type explorerEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Folder    string `json:"folder"`
	Directory bool   `json:"directory"`
	SizeBytes int64  `json:"size_bytes"`
	// Private and Locked only appear on top-level folders (ADR 0026): a locked folder shows its name only.
	Private bool `json:"private"`
	Locked  bool `json:"locked"`
	// The fields below concern videos only.
	State           string     `json:"state,omitempty"`
	ClipID          *int64     `json:"clip_id"`
	OwnerID         *int64     `json:"owner_id"`
	Owner           string     `json:"owner"`
	Visibility      string     `json:"visibility,omitempty"`
	DurationSeconds *float64   `json:"duration_seconds"`
	AddedAt         *time.Time `json:"added_at"`
	RecordedAt      *time.Time `json:"recorded_at"`
	Game            string     `json:"game"`
	VoicesPending   int        `json:"voices_pending"`
}

// explorerRecord is what the database knows about one video path of the root.
type explorerRecord struct {
	status, errorCode string
	clipID            *int64
	pending           bool
	readable          bool
	state             string
	voices            int
	hidden            bool
	ownerID           *int64
	owner, visibility string
	duration          *float64
	added, recorded   *time.Time
	game              string
}

type explorerFilters struct {
	query, state, game, visibility, from, to string
	owner                                    int64
}

func (f explorerFilters) active() bool {
	return f.query != "" || f.state != "" || f.game != "" || f.visibility != "" || f.from != "" || f.to != "" || f.owner != 0
}

func explorerFiltersFrom(r *http.Request) (explorerFilters, bool) {
	q := r.URL.Query()
	f := explorerFilters{
		query:      strings.ToLower(strings.TrimSpace(q.Get("q"))),
		state:      q.Get("state"),
		game:       strings.TrimSpace(q.Get("game")),
		visibility: q.Get("visibility"),
		from:       q.Get("from"),
		to:         q.Get("to"),
	}
	if f.state != "" && !contains(explorerStates, f.state) {
		return f, false
	}
	if f.visibility != "" && f.visibility != "shared" && f.visibility != "private" {
		return f, false
	}
	for _, date := range []string{f.from, f.to} {
		if date != "" {
			if _, err := timeParseDate(date); err != nil {
				return f, false
			}
		}
	}
	if raw := q.Get("owner_id"); raw != "" {
		var ok bool
		f.owner, ok = parsePositive(raw)
		if !ok {
			return f, false
		}
	}
	return f, true
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func parsePositive(raw string) (int64, bool) {
	var n int64
	for _, c := range raw {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
		if n > 1<<53 {
			return 0, false
		}
	}
	return n, n > 0
}

// explorerRecords reads the state of the given root-relative video paths in two queries.
func (s *Server) explorerRecords(r *http.Request, sourceID int64, paths []string, userID int64, admin bool) (map[string]explorerRecord, error) {
	out := map[string]explorerRecord{}
	if len(paths) == 0 {
		return out, nil
	}
	// A file of the root is indexed through its scan row, or through the upload published to that path.
	rows, err := s.Pool.Query(r.Context(), `SELECT f.relative_path,f.status,f.error_code,f.requested,f.clip_id,
	  COALESCE(clip_readable(c.id,c.owner_id,c.visibility,$3,$4),false),c.owner_id,COALESCE(u.username,''),COALESCE(c.visibility,''),c.duration_seconds,c.discovered_at,c.recorded_at,COALESCE(c.game,'')
	  FROM (
	    SELECT sf.relative_path,sf.status,COALESCE(sf.error_code,'') AS error_code,sf.scan_requested AS requested,sf.clip_id FROM source_files sf WHERE sf.source_id=$1 AND sf.relative_path=ANY($2)
	    UNION ALL
	    SELECT d.relative_path,'ready','',false,um.clip_id FROM upload_destinations d JOIN uploaded_media um ON um.upload_id=d.upload_id WHERE d.source_id=$1 AND d.relative_path=ANY($2) AND um.clip_id IS NOT NULL
	  ) f LEFT JOIN clips c ON c.id=f.clip_id LEFT JOIN users u ON u.id=c.owner_id`, sourceID, paths, userID, admin)
	if err != nil {
		return nil, err
	}
	clipIDs := []int64{}
	for rows.Next() {
		var rel string
		var rec explorerRecord
		var requested bool
		if err := rows.Scan(&rel, &rec.status, &rec.errorCode, &requested, &rec.clipID, &rec.readable, &rec.ownerID, &rec.owner, &rec.visibility, &rec.duration, &rec.added, &rec.recorded, &rec.game); err != nil {
			rows.Close()
			return nil, err
		}
		if _, seen := out[rel]; seen {
			continue
		}
		rec.pending = requested || rec.status == "waiting_stability"
		if rec.clipID != nil && !rec.readable {
			// A clip the viewer may not read is hidden: neither its state nor its existence is shown.
			rec.hidden = true
		}
		if rec.clipID != nil && !rec.hidden {
			clipIDs = append(clipIDs, *rec.clipID)
		}
		out[rel] = rec
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(clipIDs) == 0 {
		return out, nil
	}
	states, err := s.explorerAnalysis(r, clipIDs)
	if err != nil {
		return nil, err
	}
	for rel, rec := range out {
		if rec.clipID == nil || rec.hidden {
			continue
		}
		st := states[*rec.clipID]
		rec.state, rec.voices = analysisLabel(st.done, st.latest), st.voicesPending
		if rec.state == "transcribed" && rec.voices == 0 {
			rec.state = "analyzed"
		}
		out[rel] = rec
	}
	return out, nil
}

type analysisState struct {
	done          bool
	latest        string
	voicesPending int
}

// explorerAnalysis reads the analysis state of clips: the newest transcription job, and the voices still
// waiting for a decision.
func (s *Server) explorerAnalysis(r *http.Request, clipIDs []int64) (map[int64]analysisState, error) {
	states := map[int64]analysisState{}
	if len(clipIDs) == 0 {
		return states, nil
	}
	analysisRows, err := s.Pool.Query(r.Context(), `SELECT c.id,
	  EXISTS(SELECT 1 FROM jobs j WHERE j.clip_id=c.id AND j.step LIKE 'transcription.audio.%' AND j.status='done'),
	  COALESCE((SELECT j.status FROM jobs j WHERE j.clip_id=c.id AND j.step LIKE 'transcription.audio.%' ORDER BY j.id DESC LIMIT 1),''),
	  (SELECT count(*) FROM local_speakers ls WHERE ls.clip_id=c.id AND `+analysis.UnknownSpeakerPredicate+`)
	  FROM clips c WHERE c.id=ANY($1)`, clipIDs)
	if err != nil {
		return nil, err
	}
	defer analysisRows.Close()
	for analysisRows.Next() {
		var id int64
		var st analysisState
		if err := analysisRows.Scan(&id, &st.done, &st.latest, &st.voicesPending); err != nil {
			return nil, err
		}
		states[id] = st
	}
	return states, analysisRows.Err()
}

// analysisLabel maps the transcription state of a clip to an explorer state.
func analysisLabel(done bool, latest string) string {
	switch {
	case done:
		return "transcribed"
	case latest == "pending" || latest == "retry_wait" || latest == "leased":
		return "in_progress"
	case latest == "failed" || latest == "dead":
		return "failed"
	}
	return "not_analyzed"
}

// explorerState is the state shown for a video path; ok is false when the viewer must not see it.
func explorerState(rec explorerRecord, found bool) (string, bool) {
	if !found {
		return "not_indexed", true
	}
	if rec.hidden {
		return "", false
	}
	if rec.clipID == nil {
		switch {
		case rec.pending:
			return "in_progress", true
		case rec.status == "unreadable":
			return "failed", true
		}
		return "not_indexed", true
	}
	return rec.state, true
}

// matches applies the filters of the explorer to a video. Metadata filters need a clip, so an
// unindexed video matches only when none of them is set.
func (f explorerFilters) matches(name, folder string, rec explorerRecord, found bool) bool {
	if f.query != "" && !strings.Contains(strings.ToLower(name), f.query) {
		return false
	}
	state, _ := explorerState(rec, found)
	if f.state != "" && state != f.state {
		return false
	}
	if !found || rec.clipID == nil {
		return f.game == "" && f.visibility == "" && f.from == "" && f.to == "" && f.owner == 0
	}
	if f.game != "" && !strings.Contains(strings.ToLower(rec.game), strings.ToLower(f.game)) {
		return false
	}
	if f.visibility != "" && rec.visibility != f.visibility {
		return false
	}
	if f.owner != 0 && (rec.ownerID == nil || *rec.ownerID != f.owner) {
		return false
	}
	if rec.recorded != nil {
		day := rec.recorded.UTC().Format("2006-01-02")
		if f.from != "" && day < f.from || f.to != "" && day > f.to {
			return false
		}
	}
	return true
}

func (s *Server) explorerList(w http.ResponseWriter, r *http.Request) {
	relative := r.URL.Query().Get("path")
	recursive := r.URL.Query().Get("recursive") == "1" || r.URL.Query().Get("recursive") == "true"
	ss := r.Context().Value(sessionKey{}).(session)
	superAdmin := isSuperAdministrator(ss.User.Role)
	admin := isAdministrator(ss.User.Role)
	filters, ok := explorerFiltersFrom(r)
	if !ok {
		fail(w, 400, "invalid_filter")
		return
	}
	root, err := s.Library.ManagedRoot(r.Context())
	if errors.Is(err, pgx.ErrNoRows) && relative == "" {
		// Without a Racine vidéo, all clips still list: uploads and older sources have no folder here.
		loose, looseRecords, err := s.explorerLoose(r, 0, ss.User.ID, admin)
		if err != nil {
			s.internal(w, err)
			return
		}
		out := []explorerEntry{}
		for _, e := range loose {
			rec := looseRecords[e.Path]
			if filters.matches(e.Name, "", rec, true) {
				e.State, _ = explorerState(rec, true)
				out = append(out, e)
			}
		}
		reply(w, 200, map[string]any{"source_id": 0, "root_configured": false, "items": out, "states": explorerStates})
		return
	}
	if err != nil {
		fail(w, 409, "video_root_unconfigured")
		return
	}
	base, err := library.ManagedPath(root.Path, relative)
	if err != nil {
		fail(w, 422, "video_path_unavailable")
		return
	}
	private, err := s.privateTopFolders(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !superAdmin && private[firstSegment(relative)] {
		fail(w, 403, "forbidden")
		return
	}
	type candidate struct {
		rel, folder, name string
		directory         bool
		size              int64
		private           bool
	}
	var candidates []candidate
	if recursive {
		err = filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
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
			rel = filepath.ToSlash(rel)
			if rel == "." {
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
			candidates = append(candidates, candidate{rel: rel, folder: filepath.ToSlash(filepath.Dir(rel)), name: entry.Name(), size: info.Size()})
			if len(candidates) > explorerMaxEntries {
				return errors.New("selection too large")
			}
			return nil
		})
		if err != nil {
			fail(w, 422, "video_selection_too_large")
			return
		}
	} else {
		entries, err := os.ReadDir(base)
		if err != nil {
			fail(w, 422, "video_path_unavailable")
			return
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") || entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			if !entry.IsDir() && !videoExtension(entry.Name()) {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(relative, entry.Name()))
			c := candidate{rel: rel, folder: filepath.ToSlash(relative), name: entry.Name(), directory: entry.IsDir(), size: info.Size()}
			if entry.IsDir() && relative == "" {
				c.private = private[entry.Name()]
			}
			candidates = append(candidates, c)
			if len(candidates) > explorerMaxEntries {
				fail(w, 422, "video_selection_too_large")
				return
			}
		}
	}

	var files []string
	for _, c := range candidates {
		if !c.directory {
			files = append(files, c.rel)
		}
	}
	records, err := s.explorerRecords(r, root.ID, files, ss.User.ID, admin)
	if err != nil {
		s.internal(w, err)
		return
	}
	out := []explorerEntry{}
	for _, c := range candidates {
		e := explorerEntry{Name: c.name, Path: c.rel, Folder: c.folder, Directory: c.directory, SizeBytes: c.size, Private: c.private, Locked: c.private && !superAdmin}
		if c.directory {
			// A filtered view is flat: its results are the videos, not the folders.
			if filters.active() {
				continue
			}
			out = append(out, e)
			continue
		}
		rec, found := records[c.rel]
		if found && rec.hidden {
			continue
		}
		if !filters.matches(c.name, c.folder, rec, found) {
			continue
		}
		e.State, _ = explorerState(rec, found)
		if found {
			e.ClipID, e.OwnerID, e.Owner, e.Visibility = rec.clipID, rec.ownerID, rec.owner, rec.visibility
			e.DurationSeconds, e.AddedAt, e.RecordedAt, e.Game = rec.duration, rec.added, rec.recorded, rec.game
			e.VoicesPending = rec.voices
		}
		out = append(out, e)
	}
	if relative == "" {
		loose, looseRecords, err := s.explorerLoose(r, root.ID, ss.User.ID, admin)
		if err != nil {
			s.internal(w, err)
			return
		}
		for _, e := range loose {
			rec := looseRecords[e.Path]
			if !filters.matches(e.Name, "", rec, true) {
				continue
			}
			e.State, _ = explorerState(rec, true)
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Directory != out[j].Directory {
			return out[i].Directory
		}
		return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path)
	})
	reply(w, 200, map[string]any{"source_id": root.ID, "root_configured": true, "items": out, "states": explorerStates})
}

// explorerIndex asks for the indexing of one video the viewer opened in the explorer (#195). The
// file is probed and fingerprinted by the scan, which creates its clip; no analysis is queued. A
// video already indexed answers with its clip. The indexing completes after the scan's stability
// window, so the explorer shows the file as in progress until its clip appears.
func (s *Server) explorerIndex(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &input) {
		return
	}
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		fail(w, 409, "video_root_unconfigured")
		return
	}
	absolute, err := library.ManagedPath(root.Path, input.Path)
	if err != nil || !videoExtension(input.Path) {
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
	if !superAdmin && private[firstSegment(input.Path)] {
		fail(w, 403, "forbidden")
		return
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.Mode().IsRegular() {
		fail(w, 422, "video_path_unavailable")
		return
	}
	var clipID *int64
	var readable bool
	err = s.Pool.QueryRow(r.Context(), `SELECT sf.clip_id, COALESCE(clip_readable(c.id,c.owner_id,c.visibility,$3,$4),false) FROM source_files sf LEFT JOIN clips c ON c.id=sf.clip_id WHERE sf.source_id=$1 AND sf.relative_path=$2`, root.ID, input.Path, ss.User.ID, isAdministrator(ss.User.Role)).Scan(&clipID, &readable)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	if clipID != nil {
		if !readable {
			fail(w, 404, "clip_not_found")
			return
		}
		reply(w, 200, map[string]any{"path": input.Path, "state": "indexed", "clip_id": *clipID})
		return
	}
	modified := info.ModTime().UTC().Truncate(time.Microsecond)
	_, err = s.Pool.Exec(r.Context(), `INSERT INTO source_files(source_id,relative_path,size_bytes,modified_at,stable_since,last_seen_generation,status,scan_requested,index_only) VALUES($1,$2,$3,$4,now(),0,'waiting_stability',true,true) ON CONFLICT(source_id,relative_path) DO UPDATE SET scan_requested=true WHERE source_files.clip_id IS NULL`, root.ID, input.Path, info.Size(), modified)
	if err != nil {
		s.internal(w, err)
		return
	}
	s.Library.ScheduleScan(root.ID, 0)
	reply(w, 202, map[string]any{"path": input.Path, "state": "in_progress"})
}

// explorerLoose lists the clips the viewer may read that have no file in the Racine vidéo: uploads not
// published there and clips of the older read-only sources. They appear in "Tous les clips" only, and
// open their clip page directly (ADR 0032 keeps them visible rather than hidden).
func (s *Server) explorerLoose(r *http.Request, rootID, userID int64, admin bool) ([]explorerEntry, map[string]explorerRecord, error) {
	rows, err := s.Pool.Query(r.Context(), `SELECT c.id,c.owner_id,u.username,c.visibility,COALESCE(c.display_title,''),c.original_name,c.size_bytes,c.duration_seconds,c.discovered_at,c.recorded_at,COALESCE(c.game,'')
	  FROM clips c JOIN users u ON u.id=c.owner_id
	  WHERE clip_readable(c.id,c.owner_id,c.visibility,$1,$2)
	  AND NOT EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.source_id=$3)
	  AND NOT EXISTS(SELECT 1 FROM uploaded_media um JOIN upload_destinations d ON d.upload_id=um.upload_id WHERE um.clip_id=c.id AND d.source_id=$3)
	  ORDER BY c.id DESC LIMIT $4`, userID, admin, rootID, explorerMaxEntries)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	type loose struct {
		entry explorerEntry
		rec   explorerRecord
	}
	var found []loose
	var ids []int64
	for rows.Next() {
		var l loose
		var id, ownerID int64
		var visibility, title, original string
		var size int64
		var duration *float64
		var added, recorded time.Time
		if err := rows.Scan(&id, &ownerID, &l.rec.owner, &visibility, &title, &original, &size, &duration, &added, &recorded, &l.rec.game); err != nil {
			return nil, nil, err
		}
		name := original
		if title != "" {
			name = title
		}
		l.rec.clipID, l.rec.ownerID, l.rec.visibility, l.rec.duration, l.rec.readable = &id, &ownerID, visibility, duration, true
		l.rec.added, l.rec.recorded = &added, &recorded
		l.entry = explorerEntry{Name: name, Path: fmt.Sprintf("clip:%d", id), SizeBytes: size, ClipID: &id, OwnerID: &ownerID}
		found = append(found, l)
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	states, err := s.explorerAnalysis(r, ids)
	if err != nil {
		return nil, nil, err
	}
	out := []explorerEntry{}
	records := map[string]explorerRecord{}
	for _, l := range found {
		st := states[*l.rec.clipID]
		l.rec.state, l.rec.voices = analysisLabel(st.done, st.latest), st.voicesPending
		if l.rec.state == "transcribed" && l.rec.voices == 0 {
			l.rec.state = "analyzed"
		}
		e := l.entry
		e.Owner, e.Visibility, e.DurationSeconds, e.RecordedAt, e.Game = l.rec.owner, l.rec.visibility, l.rec.duration, l.rec.recorded, l.rec.game
		e.AddedAt, e.VoicesPending = l.rec.added, l.rec.voices
		records[e.Path] = l.rec
		out = append(out, e)
	}
	return out, records, nil
}
