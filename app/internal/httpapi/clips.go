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
	ID           int64  `json:"id"`
	OwnerID      int64  `json:"owner_id"`
	Owner        string `json:"owner"`
	OwnerDeleted bool   `json:"owner_deleted"`
	Visibility   string `json:"visibility"`
	OriginalName string `json:"original_name"`
	// DisplayTitle is the optional label chosen by an administrator; the file name stays OriginalName.
	DisplayTitle *string `json:"display_title"`
	// Detected holds the automatically detected game and capture time kept when they were edited by hand.
	Detected            *detectedMetadata `json:"detected"`
	detectedRaw         []byte
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
	folder := strings.Trim(strings.TrimSpace(r.URL.Query().Get("folder")), "/")
	if strings.Contains(folder, "..") || strings.Contains(folder, "\\") {
		fail(w, 400, "invalid_filter")
		return
	}
	game := strings.TrimSpace(r.URL.Query().Get("game"))
	admin := isAdministrator(ss.User.Role)
	visibilityClause := "clip_readable(c.id,c.owner_id,c.visibility,$1,$2)"
	args := []any{ss.User.ID, admin, visibility, status, owner, source, from, to, person.PersonID, person.Unknown, game, folder}
	filters := "($3='' OR c.visibility=$3) AND ($4='' OR c.id IN (SELECT sf.clip_id FROM source_files sf WHERE sf.status=$4) OR ($4='ready' AND EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id)) OR ($4='missing' AND NOT EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) AND NOT EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready'))) AND ($5=0 OR c.owner_id=$5) AND ($6=0 OR EXISTS(SELECT 1 FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id AND cs.id=$6)) AND ($7='' OR (c.capture_time AT TIME ZONE COALESCE((SELECT NULLIF(timezone,'') FROM clip_metadata_settings WHERE id=true),'UTC'))::date >= $7::date) AND ($8='' OR (c.capture_time AT TIME ZONE COALESCE((SELECT NULLIF(timezone,'') FROM clip_metadata_settings WHERE id=true),'UTC'))::date <= $8::date) AND " + clipPersonPredicate("$9", "$10") + " AND ($11='' OR c.game ILIKE '%' || $11 || '%') AND ($12='' OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.relative_path LIKE $12 || '/%' AND (c.owner_id=$1 OR NOT path_in_private_folder(sf.source_id,sf.relative_path) OR EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='super_admin'))))"
	var total int
	err = s.Pool.QueryRow(r.Context(), "SELECT count(*) FROM clips c WHERE "+visibilityClause+" AND "+filters, args...).Scan(&total)
	if err != nil {
		s.internal(w, err)
		return
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.Pool.Query(r.Context(), `SELECT c.id,c.owner_id,u.username,(u.deleted_at IS NOT NULL),c.visibility,c.original_name,c.display_title,c.detected_metadata,c.size_bytes,c.discovered_at,c.capture_time,c.game,c.game_origin,c.capture_time_origin,COALESCE(c.capture_time_origin='filename',true),c.duration_seconds,c.format_name,c.video_codec,c.audio_codec,c.tracks,
 CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END,
 COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(DISTINCT uploader.username ORDER BY uploader.username) FROM uploaded_media um JOIN users uploader ON uploader.id=um.owner_id WHERE um.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(jsonb_build_object('kind',o.kind,'name',o.name) ORDER BY o.created_at,o.kind,o.sort_id) FROM (SELECT 'source' AS kind,cs.name AS name,sf.first_seen_at AS created_at,cs.id AS sort_id FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id UNION ALL SELECT 'upload',CASE WHEN u2.deleted_at IS NULL THEN u2.username ELSE NULL END,um.created_at,um.owner_id FROM uploaded_media um JOIN users u2 ON u2.id=um.owner_id WHERE um.clip_id=c.id) o),'[]'::jsonb)
 FROM clips c JOIN users u ON u.id=c.owner_id WHERE `+visibilityClause+" AND "+filters+" ORDER BY c.capture_time DESC NULLS LAST,c.discovered_at DESC,c.id DESC LIMIT $13 OFFSET $14", args...)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []clipItem{}
	for rows.Next() {
		var v clipItem
		if err = rows.Scan(&v.ID, &v.OwnerID, &v.Owner, &v.OwnerDeleted, &v.Visibility, &v.OriginalName, &v.DisplayTitle, &v.detectedRaw, &v.SizeBytes, &v.DiscoveredAt, &v.RecordedAt, &v.Game, &v.GameOrigin, &v.CaptureTimeOrigin, &v.RecordedAtEstimated, &v.DurationSeconds, &v.FormatName, &v.VideoCodec, &v.AudioCodec, &v.Tracks, &v.Status, &v.Sources, &v.Uploaders, &v.Origins); err != nil {
			s.internal(w, err)
			return
		}
		v.Detected = decodeDetected(v.detectedRaw)
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

// listLibraryFolders returns the top-level folders a user may browse: those holding at least one
// clip they can read through that very folder, plus every private folder (#93). A private folder is
// listed for everyone so its name shows with a lock, but only the Super-administrator may open it:
// for anyone else it is "locked" and nothing of its content is listed here.
func (s *Server) listLibraryFolders(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	superAdmin := isSuperAdministrator(ss.User.Role)
	rows, err := s.Pool.Query(r.Context(), `SELECT cs.id,cs.name,f.folder,f.private FROM (
	  SELECT DISTINCT sf.source_id,split_part(sf.relative_path,'/',1) AS folder,path_in_private_folder(sf.source_id,sf.relative_path) AS private
	  FROM source_files sf JOIN clips c ON c.id=sf.clip_id
	  WHERE sf.status='ready' AND position('/' in sf.relative_path)>0 AND clip_readable(c.id,c.owner_id,c.visibility,$1,$2)
	  AND ($3 OR NOT path_in_private_folder(sf.source_id,sf.relative_path))
	  UNION SELECT pf.source_id,pf.folder,true FROM private_folders pf
	 ) f JOIN clip_sources cs ON cs.id=f.source_id ORDER BY cs.id,f.folder`, ss.User.ID, isAdministrator(ss.User.Role), superAdmin)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	type folder struct {
		SourceID int64  `json:"source_id"`
		Source   string `json:"source"`
		Name     string `json:"name"`
		Path     string `json:"path"`
		Private  bool   `json:"private"`
		Locked   bool   `json:"locked"`
	}
	items := []folder{}
	for rows.Next() {
		var f folder
		if err = rows.Scan(&f.SourceID, &f.Source, &f.Name, &f.Private); err != nil {
			s.internal(w, err)
			return
		}
		f.Path = f.Name
		f.Locked = f.Private && !superAdmin
		items = append(items, f)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items})
}

// putFolderPrivacy makes a top-level folder of a source private or visible again (#93). Only the
// Super-administrator may: the folder's clips then disappear for everyone else, except their owner
// and clips that stay reachable through another visible provenance.
func (s *Server) putFolderPrivacy(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var input struct {
		SourceID int64  `json:"source_id"`
		Folder   string `json:"folder"`
		Private  *bool  `json:"private"`
	}
	if !decode(w, r, &input) {
		return
	}
	folder := strings.TrimSpace(input.Folder)
	if input.SourceID < 1 || input.Private == nil || folder == "" || folder == "." || folder == ".." || strings.ContainsAny(folder, "/\\") {
		fail(w, http.StatusBadRequest, "invalid_folder")
		return
	}
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM clip_sources WHERE id=$1)`, input.SourceID).Scan(&exists); err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "source_not_found")
		return
	}
	var err error
	if *input.Private {
		_, err = s.Pool.Exec(r.Context(), `INSERT INTO private_folders(source_id,folder,set_by) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, input.SourceID, folder, ss.User.ID)
	} else {
		_, err = s.Pool.Exec(r.Context(), `DELETE FROM private_folders WHERE source_id=$1 AND folder=$2`, input.SourceID, folder)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"source_id": input.SourceID, "folder": folder, "private": *input.Private})
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
	err = s.Pool.QueryRow(r.Context(), `SELECT c.id,c.owner_id,u.username,(u.deleted_at IS NOT NULL),c.visibility,c.original_name,c.display_title,c.detected_metadata,c.size_bytes,c.discovered_at,c.capture_time,c.game,c.game_origin,c.capture_time_origin,COALESCE(c.capture_time_origin='filename',true),c.duration_seconds,c.format_name,c.video_codec,c.audio_codec,c.tracks,
 CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END,
 COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(DISTINCT uploader.username ORDER BY uploader.username) FROM uploaded_media um JOIN users uploader ON uploader.id=um.owner_id WHERE um.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(jsonb_build_object('kind',o.kind,'name',o.name) ORDER BY o.created_at,o.kind,o.sort_id) FROM (SELECT 'source' AS kind,cs.name AS name,sf.first_seen_at AS created_at,cs.id AS sort_id FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id UNION ALL SELECT 'upload',CASE WHEN u2.deleted_at IS NULL THEN u2.username ELSE NULL END,um.created_at,um.owner_id FROM uploaded_media um JOIN users u2 ON u2.id=um.owner_id WHERE um.clip_id=c.id) o),'[]'::jsonb)
 FROM clips c JOIN users u ON u.id=c.owner_id WHERE c.id=$1 AND clip_readable(c.id,c.owner_id,c.visibility,$2,$3)`, id, ss.User.ID, isAdministrator(ss.User.Role)).Scan(&v.ID, &v.OwnerID, &v.Owner, &v.OwnerDeleted, &v.Visibility, &v.OriginalName, &v.DisplayTitle, &v.detectedRaw, &v.SizeBytes, &v.DiscoveredAt, &v.RecordedAt, &v.Game, &v.GameOrigin, &v.CaptureTimeOrigin, &v.RecordedAtEstimated, &v.DurationSeconds, &v.FormatName, &v.VideoCodec, &v.AudioCodec, &v.Tracks, &v.Status, &v.Sources, &v.Uploaders, &v.Origins)
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
	{
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
	response := map[string]any{"persons": clipPeople, "audio_track_selection": selection, "id": v.ID, "owner_id": v.OwnerID, "owner": v.Owner, "visibility": v.Visibility, "original_name": v.OriginalName, "display_title": v.DisplayTitle, "detected": decodeDetected(v.detectedRaw), "size_bytes": v.SizeBytes, "discovered_at": v.DiscoveredAt, "recorded_at": v.RecordedAt, "game": v.Game, "game_origin": v.GameOrigin, "capture_time_origin": v.CaptureTimeOrigin, "recorded_at_estimated": v.RecordedAtEstimated, "duration_seconds": v.DurationSeconds, "format_name": v.FormatName, "video_codec": v.VideoCodec, "audio_codec": v.AudioCodec, "tracks": v.Tracks, "status": v.Status, "sources": v.Sources, "uploaders": v.Uploaders, "origins": v.Origins}
	progress, progressErr := s.clipProgress(r.Context(), v.ID, transcriptJobID, transcript != nil)
	if progressErr != nil {
		s.internal(w, progressErr)
		return
	}
	response["progress"] = progress
	waveforms, waveformsErr := s.clipWaveforms(r.Context(), v.ID)
	if waveformsErr != nil {
		s.internal(w, waveformsErr)
		return
	}
	response["waveforms"] = waveforms
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

	if s.refuseWhileAnalyzing(w, r, v.ID) {
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
	err = s.Pool.QueryRow(r.Context(), `SELECT c.id,c.owner_id,u.username,(u.deleted_at IS NOT NULL),c.visibility,c.original_name,c.display_title,c.detected_metadata,c.size_bytes,c.discovered_at,c.capture_time,c.game,c.game_origin,c.capture_time_origin,COALESCE(c.capture_time_origin='filename',true),c.duration_seconds,c.format_name,c.video_codec,c.audio_codec,c.tracks,
 CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END,
 COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(DISTINCT uploader.username ORDER BY uploader.username) FROM uploaded_media um JOIN users uploader ON uploader.id=um.owner_id WHERE um.clip_id=c.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_agg(jsonb_build_object('kind',o.kind,'name',o.name) ORDER BY o.created_at,o.kind,o.sort_id) FROM (SELECT 'source' AS kind,cs.name AS name,sf.first_seen_at AS created_at,cs.id AS sort_id FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id UNION ALL SELECT 'upload',CASE WHEN u2.deleted_at IS NULL THEN u2.username ELSE NULL END,um.created_at,um.owner_id FROM uploaded_media um JOIN users u2 ON u2.id=um.owner_id WHERE um.clip_id=c.id) o),'[]'::jsonb)
 FROM clips c JOIN users u ON u.id=c.owner_id WHERE c.id=$1`, id).Scan(&v.ID, &v.OwnerID, &v.Owner, &v.OwnerDeleted, &v.Visibility, &v.OriginalName, &v.DisplayTitle, &v.detectedRaw, &v.SizeBytes, &v.DiscoveredAt, &v.RecordedAt, &v.Game, &v.GameOrigin, &v.CaptureTimeOrigin, &v.RecordedAtEstimated, &v.DurationSeconds, &v.FormatName, &v.VideoCodec, &v.AudioCodec, &v.Tracks, &v.Status, &v.Sources, &v.Uploaders, &v.Origins)
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
		"display_title":         v.DisplayTitle,
		"detected":              decodeDetected(v.detectedRaw),
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
	return s.findClipMedia(ctx, clipID, nil)
}

// findClipMedia checks every available copy until one passes accept. Paths are
// resolved and constrained before the callback is allowed to read them.
func (s *Server) findClipMedia(ctx context.Context, clipID int64, accept func(string) bool) (string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT cs.root_path,sf.relative_path,cs.managed_root,sf.size_bytes,sf.modified_at FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=$1 AND sf.status='ready' AND cs.enabled ORDER BY sf.modified_at DESC`, clipID)
	if err != nil {
		return "", fmt.Errorf("query clip source media: %w", err)
	}
	var paths []string
	for rows.Next() {
		var root, relative string
		var managed bool
		var expectedSize int64
		var expectedModified time.Time
		if err = rows.Scan(&root, &relative, &managed, &expectedSize, &expectedModified); err != nil {
			rows.Close()
			return "", err
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
		paths = append(paths, resolved)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()
	choose := func(paths []string) (string, error) {
		for _, candidate := range paths {
			resolved, e := filepath.EvalSymlinks(candidate)
			if e != nil || resolved != candidate {
				continue
			}
			info, e := os.Lstat(candidate)
			if e != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				continue
			}
			if accept == nil || accept(candidate) {
				return candidate, nil
			}
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}
		return "", nil
	}
	if candidate, err := choose(paths); candidate != "" || err != nil {
		return candidate, err
	}
	paths = nil
	rows, err = s.Pool.Query(ctx, `SELECT storage_key FROM uploaded_media WHERE clip_id=$1 ORDER BY created_at DESC`, clipID)
	if err != nil {
		return "", fmt.Errorf("query uploaded clip media: %w", err)
	}
	root, rootErr := filepath.EvalSymlinks(s.Uploads.MediaRoot())
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			rows.Close()
			return "", err
		}
		if rootErr == nil {
			paths = append(paths, filepath.Join(root, key+".media"))
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()
	if candidate, err := choose(paths); candidate != "" || err != nil {
		return candidate, err
	}
	return "", os.ErrNotExist
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
