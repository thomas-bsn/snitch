package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// registerSegmentHistoryRoutes registers the history and restoration routes of a
// segment (phase 6).
func (s *Server) registerSegmentHistoryRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/clips/{id}/segments/{index}/history", s.protected(false, http.HandlerFunc(s.getSegmentHistory)))
	mux.Handle("POST /api/clips/{id}/segments/{index}/restore", s.protected(false, http.HandlerFunc(s.restoreSegmentText)))
	mux.Handle("GET /api/clips/{id}/obsolete-corrections", s.protected(false, http.HandlerFunc(s.getObsoleteCorrections)))
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

// historyColumns are the columns of segment_correction_events read as a segmentHistoryEntry.
const historyColumns = `e.id,e.kind,e.old_text,e.new_text,e.old_corrected,e.new_corrected,e.old_speaker_label,e.new_speaker_label,e.restored_from_event_id,e.restored_to_original,e.reason,e.version_before,e.version_after,e.author_id,e.author_username,e.created_at`

func scanHistoryEntry(row pgx.Row) (segmentHistoryEntry, error) {
	var e segmentHistoryEntry
	err := row.Scan(&e.ID, &e.Kind, &e.OldText, &e.NewText, &e.OldCorrected, &e.NewCorrected, &e.OldSpeakerLabel, &e.NewSpeakerLabel, &e.RestoredFromEvent, &e.RestoredToOriginal, &e.Reason, &e.VersionBefore, &e.VersionAfter, &e.Author.ID, &e.Author.Username, &e.CreatedAt)
	return e, err
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
	rows, err := s.Pool.Query(ctx, `SELECT `+historyColumns+` FROM segment_correction_events e WHERE e.transcription_job_id=$1 AND e.segment_index=$2 ORDER BY e.id DESC`, ref.JobID, ref.Index)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []segmentHistoryEntry{}
	for rows.Next() {
		e, err := scanHistoryEntry(rows)
		if err != nil {
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

// obsoleteCorrection is a segment whose correction was replaced by a retranscription:
// the text of the replaced transcription and its history, read-only.
type obsoleteCorrection struct {
	TranscriptionJobID int64                 `json:"transcription_job_id"`
	Index              int                   `json:"index"`
	OriginalText       string                `json:"original_text"`
	Text               *string               `json:"text"`
	ObsoleteAt         time.Time             `json:"obsolete_at"`
	History            []segmentHistoryEntry `json:"history"`
}

// getObsoleteCorrections lists the corrections of the transcriptions a retranscription
// replaced (segment_corrections.stale_at set), with their history, for whoever can see
// the clip. They are never applied again and cannot be restored; they stay readable.
func (s *Server) getObsoleteCorrections(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	rows, err := s.Pool.Query(ctx, `SELECT sc.transcription_job_id,sc.segment_index,COALESCE(j.result->'segments'->sc.segment_index->>'text',''),sc.text,sc.stale_at
 FROM segment_corrections sc JOIN jobs j ON j.id=sc.transcription_job_id WHERE sc.clip_id=$1 AND sc.stale_at IS NOT NULL ORDER BY sc.transcription_job_id DESC,sc.segment_index`, clip.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []obsoleteCorrection{}
	position := map[[2]int64]int{}
	for rows.Next() {
		o := obsoleteCorrection{History: []segmentHistoryEntry{}}
		if err = rows.Scan(&o.TranscriptionJobID, &o.Index, &o.OriginalText, &o.Text, &o.ObsoleteAt); err != nil {
			s.internal(w, err)
			return
		}
		position[[2]int64{o.TranscriptionJobID, int64(o.Index)}] = len(items)
		items = append(items, o)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	events, err := s.Pool.Query(ctx, `SELECT e.transcription_job_id,e.segment_index,`+historyColumns+` FROM segment_correction_events e JOIN segment_corrections sc ON sc.transcription_job_id=e.transcription_job_id AND sc.segment_index=e.segment_index
 WHERE sc.clip_id=$1 AND sc.stale_at IS NOT NULL ORDER BY e.id DESC`, clip.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer events.Close()
	for events.Next() {
		var job int64
		var index int
		var e segmentHistoryEntry
		if err = events.Scan(&job, &index, &e.ID, &e.Kind, &e.OldText, &e.NewText, &e.OldCorrected, &e.NewCorrected, &e.OldSpeakerLabel, &e.NewSpeakerLabel, &e.RestoredFromEvent, &e.RestoredToOriginal, &e.Reason, &e.VersionBefore, &e.VersionAfter, &e.Author.ID, &e.Author.Username, &e.CreatedAt); err != nil {
			s.internal(w, err)
			return
		}
		if at, found := position[[2]int64{job, int64(index)}]; found {
			items[at].History = append(items[at].History, e)
		}
	}
	if err = events.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items})
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
	if (input.EventID == nil) == !input.Original || (input.EventID != nil && *input.EventID < 1) {
		fail(w, http.StatusBadRequest, codeInvalidRequest)
		return
	}
	env, ok := parseEnvelope(w, input.ExpectedVersion, input.Reason, input.TranscriptionJobID)
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	s.correctSegment(w, r, clip, index, env, func(l lockedSegment) (segmentChange, *refusal, error) {
		ctx, ref := r.Context(), l.Ref
		// The target: the original text, or what a past entry of this very segment produced.
		// A speaker entry carries no text decision and cannot be restored here.
		target, toOriginal := ref.OriginalText, input.Original
		var fromEvent *int64
		if input.EventID != nil {
			var kind string
			var newText *string
			var newCorrected bool
			err := l.Tx.QueryRow(ctx, `SELECT kind,new_text,new_corrected FROM segment_correction_events WHERE id=$1 AND transcription_job_id=$2 AND segment_index=$3`, *input.EventID, ref.JobID, ref.Index).Scan(&kind, &newText, &newCorrected)
			if errors.Is(err, pgx.ErrNoRows) {
				return segmentChange{}, &refusal{http.StatusNotFound, codeEventNotFound}, nil
			}
			if err != nil {
				return segmentChange{}, nil, err
			}
			if kind == eventSpeaker || newText == nil {
				return segmentChange{}, &refusal{http.StatusBadRequest, codeEventNotRestorable}, nil
			}
			fromEvent, toOriginal = input.EventID, !newCorrected
			if newCorrected {
				target = *newText
			}
		}
		oldText, oldCorrected := l.effectiveText()
		if (toOriginal && !oldCorrected) || (!toOriginal && oldCorrected && target == oldText) {
			return segmentChange{}, &refusal{http.StatusBadRequest, codeTextUnchanged}, nil
		}
		// Restoring the original clears the correction (text NULL) and its author.
		var newText, authorName any = nil, ""
		var authorID any
		if !toOriginal {
			newText, authorID, authorName = target, ss.User.ID, ss.User.Username
		}
		return segmentChange{
			Set:  append([]assignment{{"text", "$4::text", ""}, {"text_author_id", "$5", ""}, {"text_author_username", "$6", ""}, {"text_corrected_at", "CASE WHEN $4::text IS NULL THEN NULL ELSE now() END", ""}}, reindexSet...),
			Args: []any{newText, authorID, authorName},
			Event: segmentEvent{Kind: eventRestore, OldText: oldText, NewText: target, OldCorrected: oldCorrected, NewCorrected: !toOriginal,
				RestoredFromEvent: fromEvent, RestoredToOriginal: toOriginal},
			View: func(written segmentState) any { return newSegmentCorrectionView(ref, &written) },
		}, nil, nil
	})
}
