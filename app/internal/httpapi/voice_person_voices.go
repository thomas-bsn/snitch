package httpapi

import "net/http"

type personVoiceView struct {
	LocalSpeakerID  int64   `json:"local_speaker_id"`
	ClipID          int64   `json:"clip_id"`
	ClipName        string  `json:"clip_name"`
	Label           string  `json:"label"`
	SpeakingSeconds float64 `json:"speaking_seconds"`
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
	rows, err := s.Pool.Query(r.Context(), `SELECT ls.id,ls.clip_id,c.original_name,ls.label,ls.speaking_seconds
		FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id JOIN clips c ON c.id=ls.clip_id
		WHERE a.person_id=$1 AND a.validation_state='validated' AND (c.visibility='shared' OR c.owner_id=$2 OR $3)
		ORDER BY c.original_name,ls.id LIMIT 500`, id, ss.User.ID, isAdministrator(ss.User.Role))
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []personVoiceView{}
	for rows.Next() {
		var v personVoiceView
		if err = rows.Scan(&v.LocalSpeakerID, &v.ClipID, &v.ClipName, &v.Label, &v.SpeakingSeconds); err != nil {
			s.internal(w, err)
			return
		}
		items = append(items, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items})
}
