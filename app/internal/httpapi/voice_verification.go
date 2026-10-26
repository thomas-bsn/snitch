package httpapi

import (
	"net/http"
	"strconv"
)

// Review of unknown and ambiguous voices (phase 5).
//
// A case is a visible diarized voice whose effective decision level is unknown or
// ambiguous: no human validation, no automatic attribution and no suggestion. Cases are
// ordered by frequency, which means the number of distinct clips of the proposed group
// (unknown voice groups) the voice belongs to, then the number of voices of that group, then the
// speaking time: naming one frequent voice settles the most uncertainty. A case is
// settled with the existing human validation (POST /api/local-speakers/{id}/validation),
// so nothing is written here. Counters per decision level are aggregated by the database
// on the clips visible to the caller. No quality figure is derived from sample counts.

type verificationCase struct {
	LocalSpeakerID  int64           `json:"local_speaker_id"`
	ClipID          int64           `json:"clip_id"`
	ClipName        string          `json:"clip_name"`
	Label           string          `json:"label"`
	SpeakingSeconds float64         `json:"speaking_seconds"`
	DecisionLevel   DecisionLevel   `json:"decision_level"`
	Candidates      []candidateView `json:"candidates"`
	GroupID         int64           `json:"group_id"`
	GroupVoiceCount int             `json:"group_voice_count"`
	GroupClipCount  int             `json:"group_clip_count"`
}

func (s *Server) registerVoiceVerificationRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/voice-verification", s.protected(false, http.HandlerFunc(s.voiceVerification)))
}

func (s *Server) voiceVerification(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	admin := isAdministrator(ss.User.Role)
	query := r.URL.Query()
	level := query.Get("level")
	switch level {
	case "":
		level = "all"
	case "all", string(LevelUnknown), string(LevelAmbiguous):
	default:
		fail(w, http.StatusBadRequest, "invalid_filter")
		return
	}
	page, size := 1, 20
	if v := query.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100000 {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
		page = n
	}
	if v := query.Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
		size = n
	}
	ctx := r.Context()
	rows, err := s.Pool.Query(ctx, voiceGroupsCTE+`, stat AS (
  SELECT group_id,count(*)::int AS voices,count(DISTINCT clip_id)::int AS clips FROM members GROUP BY group_id
), cases AS (
  SELECT m.id,m.clip_id,m.clip_name,m.label,m.speaking_seconds,m.group_id,s.voices,s.clips,a.id AS attribution_id,COALESCE(a.decision_level,'unknown') AS level
  FROM members m JOIN stat s ON s.group_id=m.group_id
  LEFT JOIN speaker_attributions a ON a.local_speaker_id=m.id AND a.validation_state='unvalidated'
  WHERE COALESCE(a.decision_level,'unknown') IN ('unknown','ambiguous') AND ($3='all' OR COALESCE(a.decision_level,'unknown')=$3)
)
SELECT id,clip_id,clip_name,label,speaking_seconds,level,attribution_id,group_id,voices,clips,count(*) OVER ()::int
FROM cases ORDER BY clips DESC,voices DESC,speaking_seconds DESC,id LIMIT $4 OFFSET $5`, ss.User.ID, admin, level, size, (page-1)*size)
	if err != nil {
		s.internal(w, err)
		return
	}
	items := []verificationCase{}
	total := 0
	byAttribution := map[int64]*attributionView{}
	attributionOf := []*int64{}
	for rows.Next() {
		var c verificationCase
		var attributionID *int64
		if err = rows.Scan(&c.LocalSpeakerID, &c.ClipID, &c.ClipName, &c.Label, &c.SpeakingSeconds, &c.DecisionLevel, &attributionID, &c.GroupID, &c.GroupVoiceCount, &c.GroupClipCount, &total); err != nil {
			rows.Close()
			s.internal(w, err)
			return
		}
		c.Candidates = []candidateView{}
		items = append(items, c)
		attributionOf = append(attributionOf, attributionID)
		if attributionID != nil {
			byAttribution[*attributionID] = &attributionView{Candidates: []candidateView{}}
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	if err = s.loadCandidates(ctx, byAttribution); err != nil {
		s.internal(w, err)
		return
	}
	for i, id := range attributionOf {
		if id != nil {
			items[i].Candidates = byAttribution[*id].Candidates
		}
	}
	counts := map[DecisionLevel]int{LevelHuman: 0, LevelAutomatic: 0, LevelSuggestion: 0, LevelAmbiguous: 0, LevelUnknown: 0}
	crows, err := s.Pool.Query(ctx, `SELECT CASE WHEN v.id IS NOT NULL THEN 'human' ELSE COALESCE(m.decision_level,'unknown') END AS level,count(*)::int
FROM local_speakers ls JOIN clips c ON c.id=ls.clip_id
LEFT JOIN speaker_attributions v ON v.local_speaker_id=ls.id AND v.validation_state='validated'
LEFT JOIN speaker_attributions m ON m.local_speaker_id=ls.id AND m.validation_state='unvalidated'
WHERE c.visibility='shared' OR c.owner_id=$1 OR $2 GROUP BY 1`, ss.User.ID, admin)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer crows.Close()
	for crows.Next() {
		var l DecisionLevel
		var n int
		if err = crows.Scan(&l, &n); err != nil {
			s.internal(w, err)
			return
		}
		counts[l] = n
	}
	if err = crows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items, "total": total, "counts": counts})
}
