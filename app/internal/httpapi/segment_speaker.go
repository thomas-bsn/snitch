package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// registerSegmentSpeakerRoutes registers the speaker correction of a segment (phase 6,
// ticket #54).
func (s *Server) registerSegmentSpeakerRoutes(mux *http.ServeMux) {
	mux.Handle("PUT /api/clips/{id}/segments/{index}/speaker", s.protected(false, http.HandlerFunc(s.putSegmentSpeaker)))
}

type segmentSpeakerInput struct {
	// LocalSpeakerID must be present: a number designates a Local speaker, null returns
	// to the speaker deduced from the speaker turns.
	LocalSpeakerID     json.RawMessage `json:"local_speaker_id"`
	ExpectedVersion    *int            `json:"expected_version"`
	Reason             string          `json:"reason"`
	TranscriptionJobID *int64          `json:"transcription_job_id"`
}

// segmentSpeakerView is the contract shape (SegmentSpeaker) of the speaker of a segment
// after a correction.
type segmentSpeakerView struct {
	TranscriptionJobID int64   `json:"transcription_job_id"`
	Index              int     `json:"index"`
	Version            int     `json:"version"`
	LocalSpeakerID     *int64  `json:"local_speaker_id"`
	LocalSpeakerLabel  *string `json:"local_speaker_label"`
	Corrected          bool    `json:"corrected"`
}

// deducedSpeaker is the Local speaker that speaks most over [start,end] of the audio
// stream of a clip (ties to the lowest ordinal): the rule of segmentSpeakerJoin and
// attachLocalSpeakers. found is false without any overlapping turn.
func deducedSpeaker(ctx context.Context, q queryer, clipID int64, stream int, start, end float64) (id int64, label string, found bool, err error) {
	err = q.QueryRow(ctx, `SELECT ls.id,ls.label FROM speaker_turns st JOIN local_speakers ls ON ls.id=st.local_speaker_id
 WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 AND LEAST($4::double precision,st.end_seconds)-GREATEST($3::double precision,st.start_seconds)>0
 GROUP BY ls.id,ls.label,ls.ordinal ORDER BY sum(LEAST($4::double precision,st.end_seconds)-GREATEST($3::double precision,st.start_seconds)) DESC,ls.ordinal LIMIT 1`, clipID, stream, start, end).Scan(&id, &label)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", false, nil
	}
	return id, label, err == nil, err
}

// putSegmentSpeaker designates the Local speaker of a segment of the visible
// transcription, or returns to the deduced one. Same optimistic-concurrency skeleton as
// putSegmentText: the row is locked, the version compared, and the override, the
// incremented version and the history entry are written in one transaction. Speaker
// turns, fingerprints, attributions and voice samples are never touched, and the
// segment is not marked for reindexing (the speaker does not change its embedding).
func (s *Server) putSegmentSpeaker(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	index, ok := parseSegmentIndex(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var input segmentSpeakerInput
	if !decode(w, r, &input) {
		return
	}
	if len(input.LocalSpeakerID) == 0 || input.ExpectedVersion == nil || *input.ExpectedVersion < 0 || (input.TranscriptionJobID != nil && *input.TranscriptionJobID < 1) {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	var target *int64
	if !bytes.Equal(bytes.TrimSpace(input.LocalSpeakerID), []byte("null")) {
		var id int64
		if err := json.Unmarshal(input.LocalSpeakerID, &id); err != nil || id < 1 {
			fail(w, http.StatusBadRequest, "invalid_request")
			return
		}
		target = &id
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
	// The designated voice must be a Local speaker of this clip and audio stream.
	newLabel := ""
	if target != nil {
		err = tx.QueryRow(ctx, `SELECT label FROM local_speakers WHERE id=$1 AND clip_id=$2 AND audio_stream_index=$3`, *target, ref.Clip.ID, ref.Stream).Scan(&newLabel)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, http.StatusBadRequest, "invalid_local_speaker")
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
	}
	current, err := scanSegmentState(tx.QueryRow(ctx, `SELECT `+segmentStateColumns+` FROM segment_corrections WHERE transcription_job_id=$1 AND segment_index=$2 FOR UPDATE`, ref.JobID, ref.Index))
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	version := 0
	var oldOverride *int64
	text, textCorrected := ref.OriginalText, false
	if exists {
		version, oldOverride = current.Version, current.SpeakerOverrideID
		if current.Text != nil {
			text, textCorrected = *current.Text, true
		}
	}
	if *input.ExpectedVersion != version {
		tx.Rollback(ctx)
		s.segmentConflict(w, r, ref, "segment_conflict")
		return
	}
	if (target == nil && oldOverride == nil) || (target != nil && oldOverride != nil && *target == *oldOverride) {
		fail(w, http.StatusBadRequest, "speaker_unchanged")
		return
	}
	// Effective speakers before and after, for the history (id NULL = deduced).
	var start, end float64
	if err = tx.QueryRow(ctx, `SELECT COALESCE((result->'segments'->$2::int->>'start')::double precision,0),COALESCE((result->'segments'->$2::int->>'end')::double precision,(result->'segments'->$2::int->>'start')::double precision,0) FROM jobs WHERE id=$1`, ref.JobID, ref.Index).Scan(&start, &end); err != nil {
		s.internal(w, err)
		return
	}
	deducedID, deducedLabel, deducedFound, err := deducedSpeaker(ctx, tx, ref.Clip.ID, ref.Stream, start, end)
	if err != nil {
		s.internal(w, err)
		return
	}
	oldLabel := ""
	if oldOverride != nil {
		if err = tx.QueryRow(ctx, `SELECT label FROM local_speakers WHERE id=$1`, *oldOverride).Scan(&oldLabel); err != nil {
			s.internal(w, err)
			return
		}
	} else if deducedFound {
		oldLabel = deducedLabel
	}
	if target == nil && deducedFound {
		newLabel = deducedLabel
	}
	var written segmentState
	switch {
	case exists && target == nil:
		written, err = scanSegmentState(tx.QueryRow(ctx, `UPDATE segment_corrections SET speaker_override_id=NULL,speaker_author_id=NULL,speaker_author_username='',speaker_corrected_at=NULL,version=version+1,updated_at=now()
 WHERE transcription_job_id=$1 AND segment_index=$2 AND version=$3 RETURNING `+segmentStateColumns, ref.JobID, ref.Index, version))
	case exists:
		written, err = scanSegmentState(tx.QueryRow(ctx, `UPDATE segment_corrections SET speaker_override_id=$4,speaker_author_id=$5,speaker_author_username=$6,speaker_corrected_at=now(),version=version+1,updated_at=now()
 WHERE transcription_job_id=$1 AND segment_index=$2 AND version=$3 RETURNING `+segmentStateColumns, ref.JobID, ref.Index, version, *target, ss.User.ID, ss.User.Username))
	default:
		written, err = scanSegmentState(tx.QueryRow(ctx, `INSERT INTO segment_corrections(transcription_job_id,segment_index,clip_id,speaker_override_id,speaker_author_id,speaker_author_username,speaker_corrected_at)
 VALUES($1,$2,$3,$4,$5,$6,now()) ON CONFLICT (transcription_job_id,segment_index) DO NOTHING RETURNING `+segmentStateColumns, ref.JobID, ref.Index, ref.Clip.ID, *target, ss.User.ID, ss.User.Username))
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
	if _, err = tx.Exec(ctx, `INSERT INTO segment_correction_events(transcription_job_id,segment_index,clip_id,kind,old_text,new_text,old_corrected,new_corrected,old_speaker_id,old_speaker_label,new_speaker_id,new_speaker_label,reason,version_before,version_after,author_id,author_username)
 VALUES($1,$2,$3,'speaker',$4,$4,$5,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, ref.JobID, ref.Index, ref.Clip.ID, text, textCorrected, oldOverride, oldLabel, target, newLabel, reason, version, written.Version, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	view := segmentSpeakerView{TranscriptionJobID: ref.JobID, Index: ref.Index, Version: written.Version, Corrected: target != nil}
	switch {
	case target != nil:
		view.LocalSpeakerID, view.LocalSpeakerLabel = target, &newLabel
	case deducedFound:
		view.LocalSpeakerID, view.LocalSpeakerLabel = &deducedID, &deducedLabel
	}
	reply(w, http.StatusOK, view)
}
