package httpapi

import (
	"net/http"
	"time"
)

type personVoiceView struct {
	LocalSpeakerID  int64     `json:"local_speaker_id"`
	ClipID          int64     `json:"clip_id"`
	ClipName        string    `json:"clip_name"`
	Label           string    `json:"label"`
	SpeakingSeconds float64   `json:"speaking_seconds"`
	RecordedAt      time.Time `json:"recorded_at"`
	// SampleCount is the number of active voice samples this voice gives to the references:
	// what a separation would take out.
	SampleCount int `json:"sample_count"`
	// ReferenceUse is what the Reference of the active fingerprint group makes of the sample of this voice:
	// contributive, or the reason it is set aside; null when the voice gives no active sample.
	ReferenceUse *string `json:"reference_use"`
}

// listPersonVoices lists the voices a human validated for a Person, on the clips the
// caller can see: the voices a separation can name. It writes nothing.
func (s *Server) listPersonVoices(w http.ResponseWriter, r *http.Request) {
	id, ok := personPathID(w, r)
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM persons WHERE id=$1)`, id).Scan(&exists); err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT ls.id,ls.clip_id,COALESCE(c.display_title,c.original_name),ls.label,ls.speaking_seconds,c.recorded_at,(SELECT count(*) FROM voice_samples vs WHERE vs.local_speaker_id=ls.id AND vs.person_id=a.person_id AND vs.retired_at IS NULL)
		FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id JOIN clips c ON c.id=ls.clip_id
		WHERE a.person_id=$1 AND a.validation_state='validated' AND clip_readable(c.id,c.owner_id,c.visibility,$2,$3)
		ORDER BY COALESCE(c.display_title,c.original_name),ls.id LIMIT 500`, id, ss.User.ID, isAdministrator(ss.User.Role))
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []personVoiceView{}
	for rows.Next() {
		var v personVoiceView
		if err = rows.Scan(&v.LocalSpeakerID, &v.ClipID, &v.ClipName, &v.Label, &v.SpeakingSeconds, &v.RecordedAt, &v.SampleCount); err != nil {
			s.internal(w, err)
			return
		}
		items = append(items, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	g, _, err := activeFingerprintGroup(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	mode, err := activeScoreMode(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	uses, err := sampleReferenceUses(r.Context(), s.Pool, g, mode, []int64{id})
	if err != nil {
		s.internal(w, err)
		return
	}
	bySpeaker := map[int64]string{}
	for _, u := range uses {
		bySpeaker[u.LocalSpeaker] = u.Use
	}
	for i := range items {
		if use, ok := bySpeaker[items[i].LocalSpeakerID]; ok {
			items[i].ReferenceUse = &use
		}
	}
	reply(w, http.StatusOK, map[string]any{"items": items})
}
