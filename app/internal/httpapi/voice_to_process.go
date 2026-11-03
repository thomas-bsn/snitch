package httpapi

import (
	"net/http"
	"strconv"
)

// "To process" (lot C): the clips that still have a voice to settle, one row per clip.
// Computed at read time from the same unknown voices as the Verification list and the
// groups (validated, automatic, ignored and random voices are out, ADR 0031), over the
// clips the caller can read. A voice counts as "to confirm" when it carries a suggestion,
// ambiguous or unknown otherwise; "groups" counts the proposed groups (two voices or more)
// the clip's voices belong to.

type toProcessClip struct {
	ClipID    int64  `json:"clip_id"`
	ClipName  string `json:"clip_name"`
	ToConfirm int    `json:"to_confirm"`
	Ambiguous int    `json:"ambiguous"`
	Unknown   int    `json:"unknown"`
	Groups    int    `json:"groups"`
}

type toProcessTotals struct {
	Clips     int `json:"clips"`
	ToConfirm int `json:"to_confirm"`
	Ambiguous int `json:"ambiguous"`
	Unknown   int `json:"unknown"`
	Groups    int `json:"groups"`
}

func (s *Server) registerToProcessRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/people/to-process", s.protected(false, http.HandlerFunc(s.listToProcess)))
}

func (s *Server) listToProcess(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	query := r.URL.Query()
	filter := query.Get("filter")
	switch filter {
	case "":
		filter = "all"
	case "all", "to_confirm", "unknown_ambiguous", "group":
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
	admin := isAdministrator(ss.User.Role)
	const perVoice = `, gsize AS (
  SELECT group_id,count(*)::int AS n FROM members GROUP BY group_id
), per AS (
  SELECT m.clip_id,m.clip_name,m.group_id,g.n,COALESCE(a.decision_level,'unknown') AS level
  FROM members m JOIN gsize g ON g.group_id=m.group_id
  LEFT JOIN speaker_attributions a ON a.local_speaker_id=m.id AND a.validation_state='unvalidated'
), by_clip AS (
  SELECT clip_id,min(clip_name) AS clip_name,
    count(*) FILTER (WHERE level='suggestion')::int AS to_confirm,
    count(*) FILTER (WHERE level='ambiguous')::int AS ambiguous,
    count(*) FILTER (WHERE level NOT IN ('suggestion','ambiguous'))::int AS unknown,
    count(DISTINCT group_id) FILTER (WHERE n>1)::int AS groups
  FROM per GROUP BY clip_id
)`
	rows, err := s.Pool.Query(ctx, voiceGroupsCTE+perVoice+`
SELECT clip_id,clip_name,to_confirm,ambiguous,unknown,groups,count(*) OVER ()::int
FROM by_clip
WHERE $3='all' OR ($3='to_confirm' AND to_confirm>0) OR ($3='unknown_ambiguous' AND ambiguous+unknown>0) OR ($3='group' AND groups>0)
ORDER BY to_confirm+ambiguous+unknown DESC,clip_id LIMIT $4 OFFSET $5`, ss.User.ID, admin, filter, size, (page-1)*size)
	if err != nil {
		s.internal(w, err)
		return
	}
	items := []toProcessClip{}
	total := 0
	for rows.Next() {
		var c toProcessClip
		if err = rows.Scan(&c.ClipID, &c.ClipName, &c.ToConfirm, &c.Ambiguous, &c.Unknown, &c.Groups, &total); err != nil {
			rows.Close()
			s.internal(w, err)
			return
		}
		items = append(items, c)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	var totals toProcessTotals
	err = s.Pool.QueryRow(ctx, voiceGroupsCTE+perVoice+`
SELECT count(*)::int,COALESCE(sum(to_confirm),0)::int,COALESCE(sum(ambiguous),0)::int,COALESCE(sum(unknown),0)::int,
  (SELECT count(DISTINCT group_id) FROM per WHERE n>1)::int
FROM by_clip`, ss.User.ID, admin).Scan(&totals.Clips, &totals.ToConfirm, &totals.Ambiguous, &totals.Unknown, &totals.Groups)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items, "total": total, "totals": totals})
}
