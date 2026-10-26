package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const (
	maxSegmentTextRunes   = 4000
	maxCorrectionReasonRu = 500
)

// registerSegmentCorrectionRoutes is the only registration point of the phase 6
// routes in Handler(). Each later step registers its routes from its own file through
// the register* hooks below, so steps do not edit the same function.
func (s *Server) registerSegmentCorrectionRoutes(mux *http.ServeMux) {
	mux.Handle("PUT /api/clips/{id}/segments/{index}/text", s.protected(false, http.HandlerFunc(s.putSegmentText)))
	s.registerSegmentHistoryRoutes(mux)
	s.registerSegmentSpeakerRoutes(mux)
	s.registerSegmentRetranscriptionRoutes(mux)
	s.registerSegmentExportRoutes(mux)
}

// segmentCorrectionView is the contract shape of the current state of one segment.
type segmentCorrectionView struct {
	TranscriptionJobID int64      `json:"transcription_job_id"`
	Index              int        `json:"index"`
	Version            int        `json:"version"`
	Text               string     `json:"text"`
	OriginalText       string     `json:"original_text"`
	Corrected          bool       `json:"corrected"`
	Author             *authorRef `json:"author"`
	CorrectedAt        *time.Time `json:"corrected_at"`
}

func newSegmentCorrectionView(ref segmentRef, st *segmentState) segmentCorrectionView {
	v := segmentCorrectionView{TranscriptionJobID: ref.JobID, Index: ref.Index, Text: ref.OriginalText, OriginalText: ref.OriginalText}
	if st == nil || st.StaleAt != nil {
		return v
	}
	v.Version = st.Version
	if st.Text != nil {
		v.Text, v.Corrected, v.CorrectedAt = *st.Text, true, st.CorrectedAt
		v.Author = &authorRef{ID: st.AuthorID, Username: st.AuthorUsername}
	}
	return v
}

type segmentConflictView struct {
	Error   string                 `json:"error"`
	Current *segmentCorrectionView `json:"current,omitempty"`
}

type segmentTextInput struct {
	Text               *string `json:"text"`
	ExpectedVersion    *int    `json:"expected_version"`
	Reason             string  `json:"reason"`
	TranscriptionJobID *int64  `json:"transcription_job_id"`
}

// cleanSegmentText trims the text and refuses empty, NUL-containing or oversized text.
func cleanSegmentText(text string) (string, string) {
	text = strings.TrimSpace(text)
	switch {
	case text == "" || strings.ContainsRune(text, 0):
		return "", "invalid_text"
	case utf8.RuneCountInString(text) > maxSegmentTextRunes:
		return "", "text_too_long"
	}
	return text, ""
}

// putSegmentText corrects the text of a segment of the visible transcription. The
// caller states the version it believes it replaces (0 for a never corrected segment);
// the row is locked, the version compared, and the new text, the incremented version
// and the history entry are written in one transaction.
func (s *Server) putSegmentText(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	index, ok := parseSegmentIndex(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var input segmentTextInput
	if !decode(w, r, &input) {
		return
	}
	if input.Text == nil || input.ExpectedVersion == nil || *input.ExpectedVersion < 0 || (input.TranscriptionJobID != nil && *input.TranscriptionJobID < 1) {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	text, code := cleanSegmentText(*input.Text)
	if code != "" {
		fail(w, http.StatusBadRequest, code)
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
	current, err := scanSegmentState(tx.QueryRow(ctx, `SELECT `+segmentStateColumns+` FROM segment_corrections WHERE transcription_job_id=$1 AND segment_index=$2 FOR UPDATE`, ref.JobID, ref.Index))
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	version := 0
	oldText, oldCorrected := ref.OriginalText, false
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
	if text == oldText {
		fail(w, http.StatusBadRequest, "text_unchanged")
		return
	}
	var written segmentState
	if exists {
		written, err = scanSegmentState(tx.QueryRow(ctx, `UPDATE segment_corrections SET text=$3,text_author_id=$4,text_author_username=$5,text_corrected_at=now(),version=version+1,
 reindex_pending=true,reindex_requested_at=now(),reindex_attempts=0,reindex_error='',reindex_lease_expires_at=NULL,updated_at=now()
 WHERE transcription_job_id=$1 AND segment_index=$2 AND version=$6 RETURNING `+segmentStateColumns, ref.JobID, ref.Index, text, ss.User.ID, ss.User.Username, version))
	} else {
		written, err = scanSegmentState(tx.QueryRow(ctx, `INSERT INTO segment_corrections(transcription_job_id,segment_index,clip_id,text,text_author_id,text_author_username,text_corrected_at,reindex_pending,reindex_requested_at)
 VALUES($1,$2,$3,$4,$5,$6,now(),true,now()) ON CONFLICT (transcription_job_id,segment_index) DO NOTHING RETURNING `+segmentStateColumns, ref.JobID, ref.Index, ref.Clip.ID, text, ss.User.ID, ss.User.Username))
	}
	if errors.Is(err, pgx.ErrNoRows) {
		// A concurrent writer won between the check and the write.
		tx.Rollback(ctx)
		s.segmentConflict(w, r, ref, "segment_conflict")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO segment_correction_events(transcription_job_id,segment_index,clip_id,kind,old_text,new_text,old_corrected,new_corrected,reason,version_before,version_after,author_id,author_username)
 VALUES($1,$2,$3,'text',$4,$5,$6,true,$7,$8,$9,$10,$11)`, ref.JobID, ref.Index, ref.Clip.ID, oldText, text, oldCorrected, reason, version, written.Version, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, newSegmentCorrectionView(ref, &written))
}

func parseSegmentIndex(w http.ResponseWriter, r *http.Request) (int, bool) {
	index, ok := segmentIndexOf(r.PathValue("index"))
	if !ok {
		fail(w, http.StatusBadRequest, "invalid_segment_index")
	}
	return index, ok
}

// segmentConflict answers 409 with the current state of the segment of the visible
// transcription (read outside the aborted transaction).
func (s *Server) segmentConflict(w http.ResponseWriter, r *http.Request, ref segmentRef, code string) {
	body := segmentConflictView{Error: code}
	if visible, found, err := s.resolveSegment(r.Context(), s.Pool, ref.Clip, ref.Index); err == nil && found {
		body.Current = s.currentSegmentView(r.Context(), visible)
	}
	reply(w, http.StatusConflict, body)
}

func (s *Server) currentSegmentView(ctx context.Context, ref segmentRef) *segmentCorrectionView {
	st, err := scanSegmentState(s.Pool.QueryRow(ctx, `SELECT `+segmentStateColumns+` FROM segment_corrections WHERE transcription_job_id=$1 AND segment_index=$2`, ref.JobID, ref.Index))
	var view segmentCorrectionView
	if err != nil {
		view = newSegmentCorrectionView(ref, nil)
	} else {
		view = newSegmentCorrectionView(ref, &st)
	}
	return &view
}
