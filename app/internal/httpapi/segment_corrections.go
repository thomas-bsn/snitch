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

// Codes of the answers of the correction routes.
const (
	codeInvalidRequest       = "invalid_request"
	codeInvalidReason        = "invalid_reason"
	codeInvalidText          = "invalid_text"
	codeTextTooLong          = "text_too_long"
	codeInvalidSegmentIndex  = "invalid_segment_index"
	codeSegmentNotFound      = "segment_not_found"
	codeSegmentConflict      = "segment_conflict"
	codeTranscriptionChanged = "transcription_changed"
	codeTextUnchanged        = "text_unchanged"
	codeSpeakerUnchanged     = "speaker_unchanged"
	codeInvalidLocalSpeaker  = "invalid_local_speaker"
	codeEventNotFound        = "event_not_found"
	codeEventNotRestorable   = "event_not_restorable"
	codeCorrectionsReplaced  = "corrections_would_be_replaced"
)

// Kinds of history entries (segment_correction_events.kind).
const (
	eventText    = "text"
	eventSpeaker = "speaker"
	eventRestore = "restore"
)

// registerSegmentCorrectionRoutes is the only registration point of the phase 6
// routes in Handler(); each family of routes registers itself from its own file.
func (s *Server) registerSegmentCorrectionRoutes(mux *http.ServeMux) {
	mux.Handle("PUT /api/clips/{id}/segments/{index}/text", s.protected(false, http.HandlerFunc(s.putSegmentText)))
	s.registerSegmentHistoryRoutes(mux)
	s.registerSegmentSpeakerRoutes(mux)
	s.registerVoiceTraceRoutes(mux)
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
	ExpectedRevisionID *string `json:"expected_revision_id"`
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
		return "", codeInvalidText
	case utf8.RuneCountInString(text) > maxSegmentTextRunes:
		return "", codeTextTooLong
	}
	return text, ""
}

// correctionEnvelope is what every correction request states besides its payload: the
// version it believes it replaces (0 for a never corrected segment), an optional
// reason, and the transcription job it was made on.
type correctionEnvelope struct {
	RevisionID      *string
	ExpectedVersion int
	Reason          string
	JobID           *int64
}

// parseEnvelope validates the envelope fields, answering 400 invalid_request or
// invalid_reason itself.
func parseEnvelope(w http.ResponseWriter, expectedVersion *int, reason string, jobID *int64, revisionID *string) (correctionEnvelope, bool) {
	if (revisionID != nil && strings.TrimSpace(*revisionID) == "") || expectedVersion == nil || *expectedVersion < 0 || (jobID != nil && *jobID < 1) {
		fail(w, http.StatusBadRequest, codeInvalidRequest)
		return correctionEnvelope{}, false
	}
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > maxCorrectionReasonRu || strings.ContainsRune(reason, 0) {
		fail(w, http.StatusBadRequest, codeInvalidReason)
		return correctionEnvelope{}, false
	}
	return correctionEnvelope{ExpectedVersion: *expectedVersion, Reason: reason, JobID: jobID, RevisionID: revisionID}, true
}

// lockedSegment is the segment of a correction in progress: its row is locked for the
// transaction.
type lockedSegment struct {
	Tx      pgx.Tx
	Ref     segmentRef
	Current segmentState
	Exists  bool
}

// effectiveText is the text the segment shows before the correction in progress.
func (l lockedSegment) effectiveText() (text string, corrected bool) {
	if l.Exists && l.Current.Text != nil {
		return *l.Current.Text, true
	}
	return l.Ref.OriginalText, false
}

// refusal is a 400 answer of a correction that changes nothing.
type refusal struct {
	Status int
	Code   string
}

// assignment sets one column of the correction row. Expr is used by the UPDATE and,
// unless InsertExpr is given, by the INSERT of the first correction of the segment.
// Placeholders start at $4.
type assignment struct {
	Column, Expr, InsertExpr string
}

// segmentEvent is the history entry of a correction (effective values before and after).
type segmentEvent struct {
	Kind                             string
	OldText, NewText                 string
	OldCorrected, NewCorrected       bool
	OldSpeakerID, NewSpeakerID       *int64
	OldSpeakerLabel, NewSpeakerLabel string
	RestoredFromEvent                *int64
	RestoredToOriginal               bool
}

// segmentChange is what a correction decides once the segment is locked.
type segmentChange struct {
	Set   []assignment
	Args  []any
	Event segmentEvent
	// View builds the answer from the row as written.
	View func(written segmentState) any
}

// correctSegment is the common skeleton of the corrections (text, restoration,
// speaker). In one transaction it resolves the segment of the visible transcription,
// checks the transcription job, locks the correction row, compares the version, calls
// plan to decide the change, writes the row (incremented version) and its history
// entry, and commits. A stale version, a concurrent writer or a replaced transcription
// answer 409 with the current state; a refusal of plan answers 400 and writes nothing.
func (s *Server) correctSegment(w http.ResponseWriter, r *http.Request, clip clipItem, index int, env correctionEnvelope, plan func(l lockedSegment) (segmentChange, *refusal, error)) {
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
		fail(w, http.StatusNotFound, codeSegmentNotFound)
		return
	}
	if (env.JobID != nil && *env.JobID != ref.JobID) || (env.RevisionID != nil && *env.RevisionID != ref.RevisionID) {
		tx.Rollback(ctx)
		s.segmentConflict(w, r, ref, codeTranscriptionChanged)
		return
	}
	locked := lockedSegment{Tx: tx, Ref: ref}
	locked.Current, err = scanSegmentState(tx.QueryRow(ctx, `SELECT `+segmentStateColumns+` FROM segment_corrections WHERE transcription_job_id=$1 AND segment_index=$2 FOR UPDATE`, ref.JobID, ref.Index))
	locked.Exists = err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	version := 0
	if locked.Exists {
		version = locked.Current.Version
	}
	if env.ExpectedVersion != version {
		tx.Rollback(ctx)
		s.segmentConflict(w, r, ref, codeSegmentConflict)
		return
	}
	change, refused, err := plan(locked)
	if err != nil {
		s.internal(w, err)
		return
	}
	if refused != nil {
		fail(w, refused.Status, refused.Code)
		return
	}
	var written segmentState
	if locked.Exists {
		sets := make([]string, 0, len(change.Set)+2)
		for _, a := range change.Set {
			sets = append(sets, a.Column+"="+a.Expr)
		}
		sets = append(sets, "version=version+1", "updated_at=now()")
		written, err = scanSegmentState(tx.QueryRow(ctx, `UPDATE segment_corrections SET `+strings.Join(sets, ",")+` WHERE transcription_job_id=$1 AND segment_index=$2 AND version=$3 RETURNING `+segmentStateColumns,
			append([]any{ref.JobID, ref.Index, version}, change.Args...)...))
	} else {
		columns, values := []string{}, []string{}
		for _, a := range change.Set {
			expr := a.InsertExpr
			if expr == "" {
				expr = a.Expr
			}
			columns, values = append(columns, a.Column), append(values, expr)
		}
		written, err = scanSegmentState(tx.QueryRow(ctx, `INSERT INTO segment_corrections(transcription_job_id,segment_index,clip_id,`+strings.Join(columns, ",")+`) VALUES($1,$2,$3,`+strings.Join(values, ",")+`) ON CONFLICT (transcription_job_id,segment_index) DO NOTHING RETURNING `+segmentStateColumns,
			append([]any{ref.JobID, ref.Index, ref.Clip.ID}, change.Args...)...))
	}
	if errors.Is(err, pgx.ErrNoRows) {
		// A concurrent writer won between the check and the write.
		tx.Rollback(ctx)
		s.segmentConflict(w, r, ref, codeSegmentConflict)
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	e := change.Event
	if _, err = tx.Exec(ctx, `INSERT INTO segment_correction_events(transcription_job_id,segment_index,clip_id,kind,old_text,new_text,old_corrected,new_corrected,old_speaker_id,old_speaker_label,new_speaker_id,new_speaker_label,restored_from_event_id,restored_to_original,reason,version_before,version_after,author_id,author_username)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`, ref.JobID, ref.Index, ref.Clip.ID, e.Kind, e.OldText, e.NewText, e.OldCorrected, e.NewCorrected, e.OldSpeakerID, e.OldSpeakerLabel, e.NewSpeakerID, e.NewSpeakerLabel, e.RestoredFromEvent, e.RestoredToOriginal, env.Reason, version, written.Version, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, change.View(written))
}

// reindexSet are the assignments that mark a segment for semantic reindexing. The text
// version moves with them (a text correction or a restoration, never a speaker one):
// indexing compares it, so a speaker correction does not make it skip a segment.
var reindexSet = []assignment{
	{"text_version", "text_version+1", "1"},
	{"reindex_pending", "true", ""},
	{"reindex_requested_at", "now()", ""},
	{"reindex_attempts", "0", ""},
	{"reindex_error", "''", ""},
	{"reindex_lease_expires_at", "NULL", ""},
}

// putSegmentText corrects the text of a segment of the visible transcription. The
// caller states the version it believes it replaces (0 for a never corrected segment);
// the new text, the incremented version and the history entry are written in one
// transaction (correctSegment).
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
	if input.Text == nil {
		fail(w, http.StatusBadRequest, codeInvalidRequest)
		return
	}
	env, ok := parseEnvelope(w, input.ExpectedVersion, input.Reason, input.TranscriptionJobID, input.ExpectedRevisionID)
	if !ok {
		return
	}
	text, code := cleanSegmentText(*input.Text)
	if code != "" {
		fail(w, http.StatusBadRequest, code)
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	s.correctSegment(w, r, clip, index, env, func(l lockedSegment) (segmentChange, *refusal, error) {
		oldText, oldCorrected := l.effectiveText()
		// The original text is the provider's raw one (often with a leading space): it is
		// compared as the cleaned text is, so a correction that only changes the
		// surrounding spaces is refused.
		if text == strings.TrimSpace(oldText) {
			return segmentChange{}, &refusal{http.StatusBadRequest, codeTextUnchanged}, nil
		}
		return segmentChange{
			Set:   append([]assignment{{"text", "$4", ""}, {"text_author_id", "$5", ""}, {"text_author_username", "$6", ""}, {"text_corrected_at", "now()", ""}}, reindexSet...),
			Args:  []any{text, ss.User.ID, ss.User.Username},
			Event: segmentEvent{Kind: eventText, OldText: oldText, NewText: text, OldCorrected: oldCorrected, NewCorrected: true},
			View:  func(written segmentState) any { return newSegmentCorrectionView(l.Ref, &written) },
		}, nil, nil
	})
}

func parseSegmentIndex(w http.ResponseWriter, r *http.Request) (int, bool) {
	index, ok := segmentIndexOf(r.PathValue("index"))
	if !ok {
		fail(w, http.StatusBadRequest, codeInvalidSegmentIndex)
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
