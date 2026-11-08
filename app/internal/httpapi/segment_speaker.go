package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// registerSegmentSpeakerRoutes registers the speaker correction of a segment (phase 6).
func (s *Server) registerSegmentSpeakerRoutes(mux *http.ServeMux) {
	mux.Handle("PUT /api/clips/{id}/segments/{index}/speaker", s.protected(false, http.HandlerFunc(s.putSegmentSpeaker)))
}

type segmentSpeakerInput struct {
	ExpectedRevisionID *string `json:"expected_revision_id"`
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
// attachLocalSpeakers. An ignored voice is no speaker (ADR 0031) and never deduced. found is
// false without any overlapping turn.
func deducedSpeaker(ctx context.Context, q queryer, clipID int64, stream int, start, end float64) (id int64, label string, found bool, err error) {
	err = q.QueryRow(ctx, `SELECT ls.id,ls.label FROM speaker_turns st JOIN local_speakers ls ON ls.id=st.local_speaker_id
 WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 AND LEAST($4::double precision,st.end_seconds)-GREATEST($3::double precision,st.start_seconds)>0 AND `+notIgnoredVoice+`
 GROUP BY ls.id,ls.label,ls.ordinal ORDER BY sum(LEAST($4::double precision,st.end_seconds)-GREATEST($3::double precision,st.start_seconds)) DESC,ls.ordinal LIMIT 1`, clipID, stream, start, end).Scan(&id, &label)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", false, nil
	}
	return id, label, err == nil, err
}

// Causes of the deletion of Local speakers, journaled by dropSpeakerOverrides.
const (
	droppedByDiarization          = "Locuteur supprimé par une nouvelle diarisation"
	droppedByMicrophone           = "Locuteur supprimé par la désignation de la piste microphone"
	droppedByMicrophoneWithdrawal = "Locuteur supprimé par le retrait de la désignation de la piste microphone"
)

// dropSpeakerOverrides runs before Local speakers are deleted, with the cause of the
// deletion (a new diarization, the designation of a microphone track or its withdrawal): the
// segments that designate one of them return to their deduced speaker, and each loss is
// recorded in the history (author "system", version incremented) instead of vanishing
// through the foreign key's ON DELETE SET NULL. The dispositions and refusals still held by
// those speakers are dropped with them, each one journaled by the system first: the journal
// outlives the voice. A new diarization has already carried or dropped them itself
// (voiceCarry), so none is left in that case.
// condition selects the Local speakers (alias ls, placeholders from $1); the cause is the
// next placeholder.
func dropSpeakerOverrides(ctx context.Context, q queryer, cause, condition string, args ...any) error {
	reason := "$" + strconv.Itoa(len(args)+1)
	args = append(args, cause)
	if _, err := q.Exec(ctx, `INSERT INTO voice_disposition_events(local_speaker_id,clip_id,old_disposition,new_disposition,author_id,author_username,reason)
SELECT ls.id,ls.clip_id,d.disposition,'',NULL,'system',`+reason+`::text
FROM local_speaker_dispositions d JOIN local_speakers ls ON ls.id=d.local_speaker_id WHERE `+condition, args...); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `INSERT INTO voice_refusal_events(local_speaker_id,clip_id,person_id,person_name,action,author_id,author_username,reason)
SELECT ls.id,ls.clip_id,vr.person_id,COALESCE(p.display_name,''),'drop',NULL,'system',`+reason+`::text
FROM voice_refusals vr JOIN local_speakers ls ON ls.id=vr.local_speaker_id LEFT JOIN persons p ON p.id=vr.person_id WHERE `+condition, args...); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `WITH dropped AS (
 UPDATE segment_corrections sc SET speaker_override_id=NULL,speaker_author_id=NULL,speaker_author_username='',speaker_corrected_at=NULL,version=sc.version+1,updated_at=now()
 FROM local_speakers ls WHERE ls.id=sc.speaker_override_id AND `+condition+`
 RETURNING sc.transcription_job_id job,sc.segment_index idx,sc.clip_id,sc.version,sc.text,ls.id speaker_id,ls.label)
INSERT INTO segment_correction_events(transcription_job_id,segment_index,clip_id,kind,old_text,new_text,old_corrected,new_corrected,old_speaker_id,old_speaker_label,new_speaker_id,new_speaker_label,reason,version_before,version_after,author_id,author_username)
SELECT d.job,d.idx,d.clip_id,'`+eventSpeaker+`',COALESCE(d.text,j.result->'segments'->d.idx->>'text',''),COALESCE(d.text,j.result->'segments'->d.idx->>'text',''),d.text IS NOT NULL,d.text IS NOT NULL,d.speaker_id,d.label,NULL,'',
 `+reason+`::text,d.version-1,d.version,NULL,'system'
FROM dropped d JOIN jobs j ON j.id=d.job`, args...)
	return err
}

// putSegmentSpeaker designates the Local speaker of a segment of the visible
// transcription, or returns to the deduced one (correctSegment). Speaker turns,
// fingerprints, attributions and voice samples are never touched, and the segment is
// not marked for reindexing (the speaker does not change its embedding).
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
	if len(input.LocalSpeakerID) == 0 {
		fail(w, http.StatusBadRequest, codeInvalidRequest)
		return
	}
	var target *int64
	if !bytes.Equal(bytes.TrimSpace(input.LocalSpeakerID), []byte("null")) {
		var id int64
		if err := json.Unmarshal(input.LocalSpeakerID, &id); err != nil || id < 1 {
			fail(w, http.StatusBadRequest, codeInvalidRequest)
			return
		}
		target = &id
	}
	env, ok := parseEnvelope(w, input.ExpectedVersion, input.Reason, input.TranscriptionJobID, input.ExpectedRevisionID)
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	s.correctSegment(w, r, clip, index, env, func(l lockedSegment) (segmentChange, *refusal, error) {
		ctx, tx, ref := r.Context(), l.Tx, l.Ref
		// The designated voice must be a Local speaker of this clip and audio stream.
		newLabel := ""
		if target != nil {
			err := tx.QueryRow(ctx, `SELECT label FROM local_speakers WHERE id=$1 AND clip_id=$2 AND audio_stream_index=$3 AND kind<>'archived'`, *target, ref.Clip.ID, ref.Stream).Scan(&newLabel)
			if errors.Is(err, pgx.ErrNoRows) {
				return segmentChange{}, &refusal{http.StatusBadRequest, codeInvalidLocalSpeaker}, nil
			}
			if err != nil {
				return segmentChange{}, nil, err
			}
		}
		var oldOverride *int64
		if l.Exists {
			oldOverride = l.Current.SpeakerOverrideID
		}
		if (target == nil && oldOverride == nil) || (target != nil && oldOverride != nil && *target == *oldOverride) {
			return segmentChange{}, &refusal{http.StatusBadRequest, codeSpeakerUnchanged}, nil
		}
		// Effective speakers before and after, for the history (id NULL = deduced).
		var start, end float64
		if err := tx.QueryRow(ctx, `SELECT seg_start,seg_end FROM audio_visible_segments WHERE job_id=$1 AND idx=$2`, ref.JobID, ref.Index).Scan(&start, &end); err != nil {
			return segmentChange{}, nil, err
		}
		deducedID, deducedLabel, deducedFound, err := deducedSpeaker(ctx, tx, ref.Clip.ID, ref.Stream, start, end)
		if err != nil {
			return segmentChange{}, nil, err
		}
		oldLabel := ""
		if oldOverride != nil {
			if err = tx.QueryRow(ctx, `SELECT label FROM local_speakers WHERE id=$1`, *oldOverride).Scan(&oldLabel); err != nil {
				return segmentChange{}, nil, err
			}
		} else if deducedFound {
			oldLabel = deducedLabel
		}
		if target == nil && deducedFound {
			newLabel = deducedLabel
		}
		text, textCorrected := l.effectiveText()
		change := segmentChange{
			Event: segmentEvent{Kind: eventSpeaker, OldText: text, NewText: text, OldCorrected: textCorrected, NewCorrected: textCorrected,
				OldSpeakerID: oldOverride, OldSpeakerLabel: oldLabel, NewSpeakerID: target, NewSpeakerLabel: newLabel},
			View: func(written segmentState) any {
				view := segmentSpeakerView{TranscriptionJobID: ref.JobID, Index: ref.VisibleIndex, Version: written.Version, Corrected: target != nil}
				switch {
				case target != nil:
					view.LocalSpeakerID, view.LocalSpeakerLabel = target, &newLabel
				case deducedFound:
					view.LocalSpeakerID, view.LocalSpeakerLabel = &deducedID, &deducedLabel
				}
				return view
			},
		}
		if target == nil {
			change.Set = []assignment{{"speaker_override_id", "NULL", ""}, {"speaker_author_id", "NULL", ""}, {"speaker_author_username", "''", ""}, {"speaker_corrected_at", "NULL", ""}}
		} else {
			change.Set = []assignment{{"speaker_override_id", "$4", ""}, {"speaker_author_id", "$5", ""}, {"speaker_author_username", "$6", ""}, {"speaker_corrected_at", "now()", ""}}
			change.Args = []any{*target, ss.User.ID, ss.User.Username}
		}
		return change, nil, nil
	})
}
