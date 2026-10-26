package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// registerSegmentHistoryRoutes registers the history and restoration routes of a
// segment (phase 6, ticket #50).
func (s *Server) registerSegmentHistoryRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/clips/{id}/segments/{index}/history", s.protected(false, http.HandlerFunc(s.getSegmentHistory)))
	mux.Handle("POST /api/clips/{id}/segments/{index}/restore", s.protected(false, http.HandlerFunc(s.restoreSegmentText)))
}

// segmentHistoryEntry is the contract shape of one immutable history entry.
type segmentHistoryEntry struct {
	ID                 int64     `json:"id"`
	Kind               string    `json:"kind"`
	OldText            *string   `json:"old_text"`
	NewText            *string   `json:"new_text"`
	OldCorrected       bool      `json:"old_corrected"`
	NewCorrected       bool      `json:"new_corrected"`
	OldSpeakerLabel    string    `json:"old_speaker_label"`
	NewSpeakerLabel    string    `json:"new_speaker_label"`
	RestoredFromEvent  *int64    `json:"restored_from_event_id"`
	RestoredToOriginal bool      `json:"restored_to_original"`
	Reason             string    `json:"reason"`
	VersionBefore      int       `json:"version_before"`
	VersionAfter       int       `json:"version_after"`
	Author             authorRef `json:"author"`
	CreatedAt          time.Time `json:"created_at"`
}

type segmentHistoryView struct {
	TranscriptionJobID int64                 `json:"transcription_job_id"`
	Index              int                   `json:"index"`
	OriginalText       string                `json:"original_text"`
	Current            segmentCorrectionView `json:"current"`
	Items              []segmentHistoryEntry `json:"items"`
}

// getSegmentHistory lists the history of a segment of the visible transcription, newest
// first, for whoever can see the clip.
func (s *Server) getSegmentHistory(w http.ResponseWriter, r *http.Request) {
	ref, ok := s.authorizedSegment(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	rows, err := s.Pool.Query(ctx, `SELECT id,kind,old_text,new_text,old_corrected,new_corrected,old_speaker_label,new_speaker_label,restored_from_event_id,restored_to_original,reason,version_before,version_after,author_id,author_username,created_at
 FROM segment_correction_events WHERE transcription_job_id=$1 AND segment_index=$2 ORDER BY id DESC`, ref.JobID, ref.Index)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []segmentHistoryEntry{}
	for rows.Next() {
		var e segmentHistoryEntry
		if err = rows.Scan(&e.ID, &e.Kind, &e.OldText, &e.NewText, &e.OldCorrected, &e.NewCorrected, &e.OldSpeakerLabel, &e.NewSpeakerLabel, &e.RestoredFromEvent, &e.RestoredToOriginal, &e.Reason, &e.VersionBefore, &e.VersionAfter, &e.Author.ID, &e.Author.Username, &e.CreatedAt); err != nil {
			s.internal(w, err)
			return
		}
		items = append(items, e)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, segmentHistoryView{TranscriptionJobID: ref.JobID, Index: ref.Index, OriginalText: ref.OriginalText, Current: *s.currentSegmentView(ctx, ref), Items: items})
}

type segmentRestoreInput struct {
	ExpectedVersion    *int   `json:"expected_version"`
	EventID            *int64 `json:"event_id"`
	Original           bool   `json:"original"`
	Reason             string `json:"reason"`
	TranscriptionJobID *int64 `json:"transcription_job_id"`
}

// restoreSegmentText restores the original text (original=true) or the text a past
// history entry produced (event_id). A restoration is itself a correction: it
// increments the version, marks the segment for reindexing and appends a 'restore'
// entry; no past entry is rewritten.
func (s *Server) restoreSegmentText(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	index, ok := parseSegmentIndex(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var input segmentRestoreInput
	if !decode(w, r, &input) {
		return
	}
	if input.ExpectedVersion == nil || *input.ExpectedVersion < 0 || (input.EventID == nil) == !input.Original ||
		(input.EventID != nil && *input.EventID < 1) || (input.TranscriptionJobID != nil && *input.TranscriptionJobID < 1) {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	reason := strings.TrimSpace(input.Reason)
	if utf8.RuneCountInString(reason) > maxCorrectionReasonRu || strings.ContainsRune(reason, 0) {
		fail(w, http.StatusBadRequest, "invalid_reason")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	ref, found, err := s.resolveSegment(ctx, tx, clip, index)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !found {
		fail(w, http.StatusNotFound, "segment_not_found")
		return
	}
	if input.TranscriptionJobID != nil && *input.TranscriptionJobID != ref.JobID {
		tx.Rollback(ctx)
		s.segmentConflict(w, r, ref, "transcription_changed")
		return
	}
	// The target: the original text, or what a past entry of this very segment produced.
	// A speaker entry carries no text decision and cannot be restored here.
	target, toOriginal := ref.OriginalText, input.Original
	var fromEvent *int64
	if input.EventID != nil {
		var kind string
		var newText *string
		var newCorrected bool
		err = tx.QueryRow(ctx, `SELECT kind,new_text,new_corrected FROM segment_correction_events WHERE id=$1 AND transcription_job_id=$2 AND segment_index=$3`, *input.EventID, ref.JobID, ref.Index).Scan(&kind, &newText, &newCorrected)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, http.StatusNotFound, "event_not_found")
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
		if kind == "speaker" || newText == nil {
			fail(w, http.StatusBadRequest, "event_not_restorable")
			return
		}
		fromEvent, toOriginal = input.EventID, !newCorrected
		if newCorrected {
			target = *newText
		}
	}
	current, err := scanSegmentState(tx.QueryRow(ctx, `SELECT `+segmentStateColumns+` FROM segment_corrections WHERE transcription_job_id=$1 AND segment_index=$2 FOR UPDATE`, ref.JobID, ref.Index))
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	version, oldText, oldCorrected := 0, ref.OriginalText, false
	if exists {
		version = current.Version
		if current.Text != nil {
			oldText, oldCorrected = *current.Text, true
		}
	}
	if *input.ExpectedVersion != version {
		tx.Rollback(ctx)
		s.segmentConflict(w, r, ref, "segment_conflict")
		return
	}
	if (toOriginal && !oldCorrected) || (!toOriginal && oldCorrected && target == oldText) {
		fail(w, http.StatusBadRequest, "text_unchanged")
		return
	}
	// Restoring the original clears the correction (text NULL) and its author.
	var newText, authorName any = nil, ""
	var authorID any
	if !toOriginal {
		newText, authorID, authorName = target, ss.User.ID, ss.User.Username
	}
	var written segmentState
	if exists {
		written, err = scanSegmentState(tx.QueryRow(ctx, `UPDATE segment_corrections SET text=$3::text,text_author_id=$4,text_author_username=$5,text_corrected_at=CASE WHEN $3::text IS NULL THEN NULL ELSE now() END,version=version+1,
 reindex_pending=true,reindex_requested_at=now(),reindex_attempts=0,reindex_error='',reindex_lease_expires_at=NULL,updated_at=now()
 WHERE transcription_job_id=$1 AND segment_index=$2 AND version=$6 RETURNING `+segmentStateColumns, ref.JobID, ref.Index, newText, authorID, authorName, version))
	} else {
		written, err = scanSegmentState(tx.QueryRow(ctx, `INSERT INTO segment_corrections(transcription_job_id,segment_index,clip_id,text,text_author_id,text_author_username,text_corrected_at,reindex_pending,reindex_requested_at)
 VALUES($1,$2,$3,$4::text,$5,$6,CASE WHEN $4::text IS NULL THEN NULL ELSE now() END,true,now()) ON CONFLICT (transcription_job_id,segment_index) DO NOTHING RETURNING `+segmentStateColumns, ref.JobID, ref.Index, ref.Clip.ID, newText, authorID, authorName))
	}
	if errors.Is(err, pgx.ErrNoRows) {
		tx.Rollback(ctx)
		s.segmentConflict(w, r, ref, "segment_conflict")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO segment_correction_events(transcription_job_id,segment_index,clip_id,kind,old_text,new_text,old_corrected,new_corrected,restored_from_event_id,restored_to_original,reason,version_before,version_after,author_id,author_username)
 VALUES($1,$2,$3,'restore',$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, ref.JobID, ref.Index, ref.Clip.ID, oldText, target, oldCorrected, !toOriginal, fromEvent, toOriginal, reason, version, written.Version, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, newSegmentCorrectionView(ref, &written))
}
