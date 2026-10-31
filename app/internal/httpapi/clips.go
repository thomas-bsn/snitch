package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type clipOrigin struct {
	Kind string  `json:"kind"`
	Name *string `json:"name"`
}

type clipItem struct {
	ID                  int64        `json:"id"`
	OwnerID             int64        `json:"owner_id"`
	Owner               string       `json:"owner"`
	OwnerDeleted        bool         `json:"owner_deleted"`
	Visibility          string       `json:"visibility"`
	OriginalName        string       `json:"original_name"`
	SizeBytes           int64        `json:"size_bytes"`
	DiscoveredAt        time.Time    `json:"discovered_at"`
	RecordedAt          *time.Time   `json:"recorded_at"`
	Game                *string      `json:"game"`
	GameOrigin          *string      `json:"game_origin"`
	CaptureTimeOrigin   *string      `json:"capture_time_origin"`
	RecordedAtEstimated bool         `json:"recorded_at_estimated"`
	DurationSeconds     *float64     `json:"duration_seconds"`
	FormatName          string       `json:"format_name"`
	VideoCodec          string       `json:"video_codec"`
	AudioCodec          string       `json:"audio_codec"`
	Tracks              any          `json:"tracks"`
	Status              string       `json:"status"`
	Sources             []string     `json:"sources"`
	Origins             []clipOrigin `json:"origins"`
	Uploaders           []string     `json:"uploaders"`
	// Persons are the Persons identified in the clip (human or automatic attribution).
	Persons []clipPerson `json:"persons"`
}

func (s *Server) listClips(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	page, size := 1, 24
	var err error
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 || page > 100000 {
			fail(w, 400, "invalid_pagination")
			return
		}
	}
	if raw := r.URL.Query().Get("page_size"); raw != "" {
		size, err = strconv.Atoi(raw)
		if err != nil || size < 1 || size > 100 {
			fail(w, 400, "invalid_pagination")
			return
		}
	}
	visibility := r.URL.Query().Get("visibility")
	if visibility != "" && visibility != "private" && visibility != "shared" {
		fail(w, 400, "invalid_filter")
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != "ready" && status != "waiting_stability" && status != "unreadable" && status != "missing" {
		fail(w, 400, "invalid_filter")
		return
	}
	owner, source := int64(0), int64(0)
	if v := r.URL.Query().Get("owner_id"); v != "" {
		owner, err = strconv.ParseInt(v, 10, 64)
		if err != nil || owner < 1 {
			fail(w, 400, "invalid_filter")
			return
		}
	}
	if v := r.URL.Query().Get("source_id"); v != "" {
		source, err = strconv.ParseInt(v, 10, 64)
		if err != nil || source < 1 {
			fail(w, 400, "invalid_filter")
			return
		}
	}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from != "" {
		if _, err = timeParseDate(from); err != nil {
			fail(w, 400, "invalid_filter")
			return
		}
	}
	if to != "" {
		if _, err = timeParseDate(to); err != nil {
			fail(w, 400, "invalid_filter")
			return
		}
	}
	person, ok := rejectPersonFilter(w, r)
	if !ok {
		return
	}
	admin := isAdministrator(ss.User.Role)
	visibilityClause := "(c.visibility='shared' OR c.owner_id=$1 OR $2)"
	args := []any{ss.User.ID, admin, visibility, status, owner, source, from, to, person.PersonID, person.Unknown, strings.TrimSpace(r.URL.Query().Get("game"))}
	filters := "($3='' OR c.visibility=$3) AND ($4='' OR c.id IN (SELECT sf.clip_id FROM source_files sf WHERE sf.status=$4) OR ($4='ready' AND EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id)) OR ($4='missing' AND NOT EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) AND NOT EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready'))) AND ($5=0 OR c.owner_id=$5) AND ($6=0 OR EXISTS(SELECT 1 FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id AND cs.id=$6)) AND ($7='' OR c.capture_time::date >= $7::date) AND ($8='' OR c.capture_time::date <= $8::date) AND " + clipPersonPredicate("$9", "$10") + " AND ($11='' OR c.game ILIKE '%' || $11 || '%')"
	var total int
	err = s.Pool.QueryRow(r.Context(), "SELECT count(*) FROM clips c WHERE "+visibilityClause+" AND "+filters, args...).Scan(&total)
	if err != nil {
		s.internal(w, err)
		return
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.Pool.Query(r.Context(), `SELECT c.id,c.owner_id,u.username,(u.deleted_at IS NOT NULL),c.visibility,c.original_name,c.size_bytes,c.discovered_at,c.capture_time,c.game,c.game_origin,c.capture_time_origin,c.recorded_at_estimated,c.duration_seconds,c.format_name,c.video_codec,c.audio_codec,c.tracks,
 CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END,
 COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(DISTINCT uploader.username ORDER BY uploader.username) FROM uploaded_media um JOIN users uploader ON uploader.id=um.owner_id WHERE um.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(jsonb_build_object('kind',o.kind,'name',o.name) ORDER BY o.created_at,o.kind,o.sort_id) FROM (SELECT 'source' AS kind,cs.name AS name,sf.first_seen_at AS created_at,cs.id AS sort_id FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id UNION ALL SELECT 'upload',CASE WHEN u2.deleted_at IS NULL THEN u2.username ELSE NULL END,um.created_at,um.owner_id FROM uploaded_media um JOIN users u2 ON u2.id=um.owner_id WHERE um.clip_id=c.id) o),'[]'::jsonb)
 FROM clips c JOIN users u ON u.id=c.owner_id WHERE `+visibilityClause+" AND "+filters+" ORDER BY c.capture_time DESC NULLS LAST,c.discovered_at DESC,c.id DESC LIMIT $12 OFFSET $13", args...)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []clipItem{}
	for rows.Next() {
		var v clipItem
		if err = rows.Scan(&v.ID, &v.OwnerID, &v.Owner, &v.OwnerDeleted, &v.Visibility, &v.OriginalName, &v.SizeBytes, &v.DiscoveredAt, &v.RecordedAt, &v.Game, &v.GameOrigin, &v.CaptureTimeOrigin, &v.RecordedAtEstimated, &v.DurationSeconds, &v.FormatName, &v.VideoCodec, &v.AudioCodec, &v.Tracks, &v.Status, &v.Sources, &v.Uploaders, &v.Origins); err != nil {
			s.internal(w, err)
			return
		}
		items = append(items, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	rows.Close()
	ids := make([]int64, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	persons, err := s.clipPersons(r.Context(), ids)
	if err != nil {
		s.internal(w, err)
		return
	}
	for i := range items {
		items[i].Persons = persons[items[i].ID]
		if items[i].Persons == nil {
			items[i].Persons = []clipPerson{}
		}
	}
	reply(w, 200, map[string]any{"items": items, "page": page, "page_size": size, "total": total})
}

func timeParseDate(v string) (time.Time, error) { return time.Parse("2006-01-02", v) }

func (s *Server) authorizedClip(w http.ResponseWriter, r *http.Request) (clipItem, bool) {
	ss := r.Context().Value(sessionKey{}).(session)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_clip_id")
		return clipItem{}, false
	}
	var v clipItem
	err = s.Pool.QueryRow(r.Context(), `SELECT c.id,c.owner_id,u.username,(u.deleted_at IS NOT NULL),c.visibility,c.original_name,c.size_bytes,c.discovered_at,c.capture_time,c.game,c.game_origin,c.capture_time_origin,c.recorded_at_estimated,c.duration_seconds,c.format_name,c.video_codec,c.audio_codec,c.tracks,
 CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END,
 COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(DISTINCT uploader.username ORDER BY uploader.username) FROM uploaded_media um JOIN users uploader ON uploader.id=um.owner_id WHERE um.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(jsonb_build_object('kind',o.kind,'name',o.name) ORDER BY o.created_at,o.kind,o.sort_id) FROM (SELECT 'source' AS kind,cs.name AS name,sf.first_seen_at AS created_at,cs.id AS sort_id FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id UNION ALL SELECT 'upload',CASE WHEN u2.deleted_at IS NULL THEN u2.username ELSE NULL END,um.created_at,um.owner_id FROM uploaded_media um JOIN users u2 ON u2.id=um.owner_id WHERE um.clip_id=c.id) o),'[]'::jsonb)
 FROM clips c JOIN users u ON u.id=c.owner_id WHERE c.id=$1 AND (c.visibility='shared' OR c.owner_id=$2 OR $3)`, id, ss.User.ID, isAdministrator(ss.User.Role)).Scan(&v.ID, &v.OwnerID, &v.Owner, &v.OwnerDeleted, &v.Visibility, &v.OriginalName, &v.SizeBytes, &v.DiscoveredAt, &v.RecordedAt, &v.Game, &v.GameOrigin, &v.CaptureTimeOrigin, &v.RecordedAtEstimated, &v.DurationSeconds, &v.FormatName, &v.VideoCodec, &v.AudioCodec, &v.Tracks, &v.Status, &v.Sources, &v.Uploaders, &v.Origins)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "clip_not_found")
		return clipItem{}, false
	}
	if err != nil {
		s.internal(w, err)
		return clipItem{}, false
	}
	return v, true
}
func (s *Server) getClip(w http.ResponseWriter, r *http.Request) {
	v, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	selection, selectionErr := s.audioSelection(r.Context(), v.ID, tracksFromClip(v.Tracks))
	if selectionErr != nil {
		s.internal(w, selectionErr)
		return
	}
	var transcript json.RawMessage
	var transcriptJobID int64
	var states map[int]segmentState
	if selection.AudioStreamIndex != nil {
		// Keep the last successful transcript visible while a newer input version
		// is queued or running. A completed replacement naturally becomes newest.
		visible, visibleStates, found, err := s.readVisibleTranscript(r.Context(), v.ID)
		if err != nil {
			s.internal(w, err)
			return
		}
		if found {
			transcript, transcriptJobID, states = visible.Result, visible.JobID, visibleStates
		}
	}
	persons, personsErr := s.clipPersons(r.Context(), []int64{v.ID})
	if personsErr != nil {
		s.internal(w, personsErr)
		return
	}
	clipPeople := persons[v.ID]
	if clipPeople == nil {
		clipPeople = []clipPerson{}
	}
	response := map[string]any{"persons": clipPeople, "audio_track_selection": selection, "id": v.ID, "owner_id": v.OwnerID, "owner": v.Owner, "visibility": v.Visibility, "original_name": v.OriginalName, "size_bytes": v.SizeBytes, "discovered_at": v.DiscoveredAt, "recorded_at": v.RecordedAt, "game": v.Game, "game_origin": v.GameOrigin, "capture_time_origin": v.CaptureTimeOrigin, "recorded_at_estimated": v.RecordedAtEstimated, "duration_seconds": v.DurationSeconds, "format_name": v.FormatName, "video_codec": v.VideoCodec, "audio_codec": v.AudioCodec, "tracks": v.Tracks, "status": v.Status, "sources": v.Sources, "uploaders": v.Uploaders, "origins": v.Origins}
	progress, progressErr := s.clipProgress(r.Context(), v.ID, transcriptJobID, transcript != nil)
	if progressErr != nil {
		s.internal(w, progressErr)
		return
	}
	response["progress"] = progress
	if transcript != nil {
		var value any
		if json.Unmarshal(transcript, &value) == nil {
			overlaySegments(value, transcriptJobID, states)
			if err := s.attachLocalSpeakers(r.Context(), v.ID, value, states); err != nil {
				s.internal(w, err)
				return
			}
			response["transcription"] = value
		}
	}
	reply(w, 200, response)
}

// scheduleClipTranscription schedules a transcription job for a clip from the clip page.
// It is accessible to any authenticated user who can view the clip (not just admins).
// If the clip has corrections, the caller must confirm they want to replace them.
func (s *Server) scheduleClipTranscription(w http.ResponseWriter, r *http.Request) {
	v, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}

	var input struct {
		AudioStreamIndex *int `json:"audio_stream_index,omitempty"`
		ConfirmReplace   bool `json:"confirm_replace,omitempty"`
	}
	if !decode(w, r, &input) {
		return
	}

	// Check if the clip has existing corrections that would be replaced
	hasCorrections, err := s.hasUnstaleCorrections(r.Context(), v.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if hasCorrections && !input.ConfirmReplace {
		fail(w, http.StatusConflict, "corrections_would_be_replaced")
		return
	}

	if input.AudioStreamIndex != nil && *input.AudioStreamIndex < 0 {
		fail(w, 400, "invalid_audio_stream")
		return
	}

	job, created, err := s.scheduleResolvedTranscription(r.Context(), v.ID, input.AudioStreamIndex, true)
	if err != nil {
		if err.Error() == "audio selection required" {
			fail(w, http.StatusConflict, "audio_track_selection_required")
			return
		}
		if err.Error() == "audio stream not found" {
			fail(w, 400, "audio_stream_not_found")
			return
		}
		if err.Error() == "clip not ready" {
			fail(w, http.StatusConflict, "clip_not_ready")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 404, "clip_not_found")
			return
		}
		if strings.Contains(err.Error(), "audio stream") {
			fail(w, 400, "audio_stream_not_found")
			return
		}
		s.internal(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	reply(w, status, map[string]any{"status": "scheduled", "job": job})
}

// hasUnstaleCorrections checks if a clip has non-stale corrections that would be replaced
// by a new transcription job.
func (s *Server) hasUnstaleCorrections(ctx context.Context, clipID int64) (bool, error) {
	var count int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND stale_at IS NULL`, clipID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// updateClip updates a clip's visibility. Admin only.
func (s *Server) updateClip(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	if !isAdministrator(ss.User.Role) {
		fail(w, 403, "forbidden")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_clip_id")
		return
	}

	var input struct {
		Visibility string `json:"visibility"`
	}
	if !decode(w, r, &input) {
		return
	}

	if input.Visibility != "shared" && input.Visibility != "private" {
		fail(w, 400, "invalid_visibility")
		return
	}

	// Update visibility
	_, err = s.Pool.Exec(r.Context(), `UPDATE clips SET visibility=$1 WHERE id=$2`, input.Visibility, id)
	if err != nil {
		s.internal(w, err)
		return
	}

	// Return updated clip (reuse authorizedClip logic but without visibility filter since admin)
	var v clipItem
	err = s.Pool.QueryRow(r.Context(), `SELECT c.id,c.owner_id,u.username,(u.deleted_at IS NOT NULL),c.visibility,c.original_name,c.size_bytes,c.discovered_at,c.capture_time,c.game,c.game_origin,c.capture_time_origin,c.recorded_at_estimated,c.duration_seconds,c.format_name,c.video_codec,c.audio_codec,c.tracks,
 CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END,
 COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(DISTINCT uploader.username ORDER BY uploader.username) FROM uploaded_media um JOIN users uploader ON uploader.id=um.owner_id WHERE um.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(jsonb_build_object('kind',o.kind,'name',o.name) ORDER BY o.created_at,o.kind,o.sort_id) FROM (SELECT 'source' AS kind,cs.name AS name,sf.first_seen_at AS created_at,cs.id AS sort_id FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id UNION ALL SELECT 'upload',CASE WHEN u2.deleted_at IS NULL THEN u2.username ELSE NULL END,um.created_at,um.owner_id FROM uploaded_media um JOIN users u2 ON u2.id=um.owner_id WHERE um.clip_id=c.id) o),'[]'::jsonb)
 FROM clips c JOIN users u ON u.id=c.owner_id WHERE c.id=$1`, id).Scan(&v.ID, &v.OwnerID, &v.Owner, &v.OwnerDeleted, &v.Visibility, &v.OriginalName, &v.SizeBytes, &v.DiscoveredAt, &v.RecordedAt, &v.Game, &v.GameOrigin, &v.CaptureTimeOrigin, &v.RecordedAtEstimated, &v.DurationSeconds, &v.FormatName, &v.VideoCodec, &v.AudioCodec, &v.Tracks, &v.Status, &v.Sources, &v.Uploaders, &v.Origins)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "clip_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}

	reply(w, 200, map[string]any{
		"id":                    v.ID,
		"owner_id":              v.OwnerID,
		"owner":                 v.Owner,
		"owner_deleted":         v.OwnerDeleted,
		"visibility":            v.Visibility,
		"original_name":         v.OriginalName,
		"size_bytes":            v.SizeBytes,
		"discovered_at":         v.DiscoveredAt,
		"recorded_at":           v.RecordedAt,
		"game":                  v.Game,
		"game_origin":           v.GameOrigin,
		"capture_time_origin":   v.CaptureTimeOrigin,
		"recorded_at_estimated": v.RecordedAtEstimated,
		"duration_seconds":      v.DurationSeconds,
		"format_name":           v.FormatName,
		"video_codec":           v.VideoCodec,
		"audio_codec":           v.AudioCodec,
		"tracks":                v.Tracks,
		"status":                v.Status,
		"sources":               v.Sources,
		"uploaders":             v.Uploaders,
		"origins":               v.Origins,
	})
}

func (s *Server) serveClipMedia(w http.ResponseWriter, r *http.Request) {
	v, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	candidate, err := s.resolveClipMedia(r.Context(), v.ID)
	if errors.Is(err, os.ErrNotExist) {
		fail(w, 404, "media_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	f, err := os.Open(candidate)
	if err != nil {
		fail(w, 404, "media_not_found")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		fail(w, 404, "media_not_found")
		return
	}
	w.Header().Set("Accept-Ranges", "bytes")
	extension := strings.ToLower(path.Ext(v.OriginalName))
	contentType := map[string]string{".mp4": "video/mp4", ".m4v": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime", ".mkv": "video/x-matroska", ".ogv": "video/ogg"}[extension]
	if contentType == "" {
		contentType = mime.TypeByExtension(extension)
	}
	if contentType == "" || !strings.HasPrefix(contentType, "video/") {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	filename := filepath.Base(v.OriginalName)
	contentDisposition := mime.FormatMediaType("inline", map[string]string{"filename": filename})
	if contentDisposition == "" {
		contentDisposition = `inline; filename="clip"`
	}
	w.Header().Set("Content-Disposition", contentDisposition)
	start, end, status, valid := parseRange(r.Header.Get("Range"), info.Size())
	if !valid {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", info.Size()))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	if info.Size() == 0 {
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Header.Get("Range") != "" {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, info.Size()))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		start, end, status = 0, info.Size()-1, 200
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		w.WriteHeader(status)
	}
	if r.Method == http.MethodHead {
		return
	}
	_, _ = f.Seek(start, io.SeekStart)
	_, _ = io.CopyN(w, f, end-start+1)
}

func (s *Server) resolveClipMedia(ctx context.Context, clipID int64) (string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT cs.root_path,sf.relative_path,cs.managed_root,sf.size_bytes,sf.modified_at FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=$1 AND sf.status='ready' AND cs.enabled ORDER BY sf.modified_at DESC`, clipID)
	if err != nil {
		return "", fmt.Errorf("query clip source media: %w", err)
	}
	var candidate string
	for rows.Next() {
		var root, relative string
		var managed bool
		var expectedSize int64
		var expectedModified time.Time
		if rows.Scan(&root, &relative, &managed, &expectedSize, &expectedModified) != nil {
			continue
		}
		base, e := filepath.EvalSymlinks(root)
		if e != nil {
			continue
		}
		path := filepath.Join(base, filepath.FromSlash(relative))
		resolved, e := filepath.EvalSymlinks(path)
		if e != nil || !inside(base, resolved) {
			continue
		}
		info, e := os.Stat(resolved)
		if e != nil || !info.Mode().IsRegular() {
			continue
		}
		if managed && (info.Size() != expectedSize || !info.ModTime().UTC().Truncate(time.Microsecond).Equal(expectedModified)) {
			continue
		}
		candidate = resolved
		break
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return "", fmt.Errorf("read clip source media: %w", err)
	}
	rows.Close()
	if candidate == "" {
		var key string
		err = s.Pool.QueryRow(ctx, `SELECT um.storage_key FROM uploaded_media um WHERE um.clip_id=$1 ORDER BY um.created_at DESC LIMIT 1`, clipID).Scan(&key)
		if err == nil {
			root, rootErr := filepath.EvalSymlinks(s.Uploads.MediaRoot())
			if rootErr == nil {
				candidate = filepath.Join(root, key+".media")
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("query uploaded clip media: %w", err)
		}
	}
	if candidate == "" {
		return "", os.ErrNotExist
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil || resolved != candidate {
		return "", os.ErrNotExist
	}
	info, err := os.Lstat(candidate)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", os.ErrNotExist
	}
	return candidate, nil
}

func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}
func parseRange(raw string, size int64) (int64, int64, int, bool) {
	if raw == "" {
		return 0, size - 1, 200, true
	}
	if size <= 0 || !strings.HasPrefix(raw, "bytes=") || strings.Contains(raw, ",") {
		return 0, 0, 416, false
	}
	parts := strings.Split(strings.TrimPrefix(raw, "bytes="), "-")
	if len(parts) != 2 {
		return 0, 0, 416, false
	}
	if parts[0] == "" {
		n, e := strconv.ParseInt(parts[1], 10, 64)
		if e != nil || n <= 0 {
			return 0, 0, 416, false
		}
		if n > size {
			n = size
		}
		return size - n, size - 1, 206, true
	}
	start, e := strconv.ParseInt(parts[0], 10, 64)
	if e != nil || start < 0 || start >= size {
		return 0, 0, 416, false
	}
	end := size - 1
	if parts[1] != "" {
		end, e = strconv.ParseInt(parts[1], 10, 64)
		if e != nil || end < start {
			return 0, 0, 416, false
		}
		if end >= size {
			end = size - 1
		}
	}
	return start, end, 206, true
}
