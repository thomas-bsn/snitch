package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// Direct read of one Local speaker: the target of a Verification link. It answers with
// the same visibility rule as every other local speaker route (authorizedSpeaker), so a
// voice of a hidden clip and a missing voice are the same 404 with the same body.

type localSpeakerReadView struct {
	LocalSpeakerID  int64            `json:"local_speaker_id"`
	ClipID          int64            `json:"clip_id"`
	ClipName        string           `json:"clip_name"`
	Label           string           `json:"label"`
	SpeakingSeconds float64          `json:"speaking_seconds"`
	DecisionLevel   DecisionLevel    `json:"decision_level"`
	Attribution     *attributionView `json:"attribution"`
	// Group fields are set only for a voice still to decide (unknown or ambiguous).
	GroupID         *int64 `json:"group_id,omitempty"`
	GroupVoiceCount *int   `json:"group_voice_count,omitempty"`
	GroupClipCount  *int   `json:"group_clip_count,omitempty"`
}

func (s *Server) getLocalSpeaker(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	ss := ctx.Value(sessionKey{}).(session)
	v := localSpeakerReadView{LocalSpeakerID: sp.ID, ClipID: sp.ClipID, Label: sp.Label}
	if err := s.Pool.QueryRow(ctx, `SELECT COALESCE(c.display_title,c.original_name),ls.speaking_seconds FROM local_speakers ls JOIN clips c ON c.id=ls.clip_id WHERE ls.id=$1`, sp.ID).Scan(&v.ClipName, &v.SpeakingSeconds); err != nil {
		s.internal(w, err)
		return
	}
	effective, err := s.effectiveAttributions(ctx, sp.ClipID)
	if err != nil {
		s.internal(w, err)
		return
	}
	v.DecisionLevel = LevelUnknown
	if a, found := effective[sp.ID]; found {
		v.DecisionLevel = a.DecisionLevel
		if a.ValidationState == "validated" {
			v.DecisionLevel = LevelHuman
		}
		// An unknown voice has no attribution to show, whatever empty row records it.
		if v.DecisionLevel != LevelUnknown {
			v.Attribution = &a
		}
	}
	if v.DecisionLevel == LevelUnknown || v.DecisionLevel == LevelAmbiguous {
		groupID, voices, clips := sp.ID, 1, 1
		err = s.Pool.QueryRow(ctx, voiceGroupsCTE+`, stat AS (
  SELECT group_id,count(*)::int AS voices,count(DISTINCT clip_id)::int AS clips FROM members GROUP BY group_id
) SELECT m.group_id,s.voices,s.clips FROM members m JOIN stat s ON s.group_id=m.group_id WHERE m.id=$3`, ss.User.ID, isAdministrator(ss.User.Role), sp.ID).Scan(&groupID, &voices, &clips)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			s.internal(w, err)
			return
		}
		v.GroupID, v.GroupVoiceCount, v.GroupClipCount = &groupID, &voices, &clips
	}
	reply(w, http.StatusOK, v)
}
