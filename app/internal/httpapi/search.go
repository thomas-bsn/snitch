package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type searchItem struct {
	ClipID      int64     `json:"clip_id"`
	ClipName    string    `json:"clip_name"`
	Visibility  string    `json:"visibility"`
	RecordedAt  time.Time `json:"recorded_at"`
	Status      string    `json:"status"`
	Sources     []string  `json:"sources"`
	MatchType   string    `json:"match_type"`
	Context     string    `json:"context"`
	MatchedText string    `json:"matched_text"`
	Timestamp   float64   `json:"timestamp"`
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" || len([]rune(q)) > 200 {
		fail(w, http.StatusBadRequest, "invalid_query")
		return
	}
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "full_text"
	}
	if mode != "exact" && mode != "full_text" {
		fail(w, 400, "invalid_mode")
		return
	}
	page, size := 1, 20
	var err error
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 || page > 100000 {
			fail(w, 400, "invalid_pagination")
			return
		}
	}
	if raw := r.URL.Query().Get("page_size"); raw != "" {
		size, err = strconv.Atoi(raw)
		if err != nil || size < 1 || size > 100 {
			fail(w, 400, "invalid_pagination")
			return
		}
	}
	visibility, status := r.URL.Query().Get("visibility"), r.URL.Query().Get("status")
	if visibility != "" && visibility != "private" && visibility != "shared" {
		fail(w, 400, "invalid_filter")
		return
	}
	if status != "" && status != "ready" && status != "waiting_stability" && status != "unreadable" && status != "missing" {
		fail(w, 400, "invalid_filter")
		return
	}
	sourceID := int64(0)
	if raw := r.URL.Query().Get("source_id"); raw != "" {
		sourceID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || sourceID < 1 {
			fail(w, 400, "invalid_filter")
			return
		}
	}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from != "" {
		if _, err = timeParseDate(from); err != nil {
			fail(w, 400, "invalid_filter")
			return
		}
	}
	if to != "" {
		if _, err = timeParseDate(to); err != nil {
			fail(w, 400, "invalid_filter")
			return
		}
	}
	if from != "" && to != "" && from > to {
		fail(w, 400, "invalid_filter")
		return
	}
	admin := ss.User.Role == "admin"
	// Translate composes with lower() to match French accented and unaccented input
	// without requiring a database extension or changing the worker result format.
	queryNorm := normalizeSearch(q)
	match := "wn=$1"
	if mode == "full_text" {
		match = "sn LIKE '%' || $1 || '%'"
	}
	baseQuery := `WITH transcript AS (
 SELECT DISTINCT ON (j.clip_id) j.clip_id,j.result FROM jobs j WHERE j.status='done' AND j.step LIKE 'transcription.audio.%' ORDER BY j.clip_id,j.updated_at DESC,j.id DESC
), parts AS (
 SELECT c.id clip_id,c.original_name clip_name,c.visibility,c.recorded_at,
 CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END status,
 COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb) sources,
	 seg->>'text' context, COALESCE((seg->>'start')::double precision,0) segment_start, word->>'word' matched_text, COALESCE((word->>'start')::double precision,(seg->>'start')::double precision,0) timestamp,
 CASE WHEN word IS NOT NULL AND ` + fmtNorm("word->>'word'") + `=$1 THEN 'exact' ELSE 'full_text' END match_type,
 ` + fmtNorm("seg->>'text'") + ` sn, ` + fmtNorm("word->>'word'") + ` wn
 FROM transcript t JOIN clips c ON c.id=t.clip_id CROSS JOIN LATERAL jsonb_array_elements(COALESCE(t.result->'segments','[]'::jsonb)) seg
 LEFT JOIN LATERAL jsonb_array_elements(COALESCE(seg->'words','[]'::jsonb)) word ON true
 WHERE (c.visibility='shared' OR c.owner_id=$2 OR $3)
 AND ($4='' OR c.visibility=$4) AND ($5=0 OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.source_id=$5))
 AND ($6='' OR c.recorded_at::date >= $6::date) AND ($7='' OR c.recorded_at::date <= $7::date)
 AND ($8='' OR CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END=$8)
)
	`
	countQuery := `WITH transcript AS (SELECT DISTINCT ON (j.clip_id) j.clip_id,j.result FROM jobs j WHERE j.status='done' AND j.step LIKE 'transcription.audio.%' ORDER BY j.clip_id,j.updated_at DESC,j.id DESC), parts AS (SELECT c.id clip_id,seg->>'text' context,COALESCE((seg->>'start')::double precision,0) segment_start,word->>'word' matched_text,` + fmtNorm("seg->>'text'") + ` sn,` + fmtNorm("word->>'word'") + ` wn FROM transcript t JOIN clips c ON c.id=t.clip_id CROSS JOIN LATERAL jsonb_array_elements(COALESCE(t.result->'segments','[]'::jsonb)) seg LEFT JOIN LATERAL jsonb_array_elements(COALESCE(seg->'words','[]'::jsonb)) word ON true WHERE (c.visibility='shared' OR c.owner_id=$2 OR $3) AND ($4='' OR c.visibility=$4) AND ($5=0 OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.source_id=$5)) AND ($6='' OR c.recorded_at::date >= $6::date) AND ($7='' OR c.recorded_at::date <= $7::date) AND ($8='' OR CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END=$8)) SELECT count(*) FROM parts WHERE ` + match
	resultQuery := "SELECT clip_id,clip_name,visibility,recorded_at,status,sources,context,matched_text,timestamp,match_type FROM parts WHERE " + match
	if mode == "full_text" {
		resultQuery = "SELECT clip_id,clip_name,visibility,recorded_at,status,sources,context,matched_text,timestamp,match_type FROM (SELECT DISTINCT ON (clip_id,context,segment_start) clip_id,clip_name,visibility,recorded_at,status,sources,context,matched_text,timestamp,match_type,segment_start,wn FROM parts WHERE " + match + " ORDER BY clip_id,context,segment_start,(wn LIKE '%' || $1 || '%') DESC,timestamp) matches"
		countQuery = strings.Replace(countQuery, "SELECT count(*) FROM parts WHERE "+match, "SELECT count(DISTINCT (clip_id,context,segment_start)) FROM parts WHERE "+match, 1)
	}
	query := baseQuery + resultQuery + " ORDER BY recorded_at DESC,clip_id DESC,timestamp LIMIT $9 OFFSET $10"
	args := []any{queryNorm, ss.User.ID, admin, visibility, sourceID, from, to, status}
	var total int
	if err = s.Pool.QueryRow(r.Context(), countQuery, args...).Scan(&total); err != nil {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(r.Context(), query, append(args, size, (page-1)*size)...)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []searchItem{}
	for rows.Next() {
		var item searchItem
		if err = rows.Scan(&item.ClipID, &item.ClipName, &item.Visibility, &item.RecordedAt, &item.Status, &item.Sources, &item.Context, &item.MatchedText, &item.Timestamp, &item.MatchType); err != nil {
			s.internal(w, err)
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]any{"items": items, "page": page, "page_size": size, "total": total})
}

func fmtNorm(value string) string {
	return "translate(lower(" + value + "),'àáâãäåæçèéêëìíîïñòóôõöøùúûüýÿœ','aaaaaaaceeeeiiiinoooooouuuuyyo')"
}

func normalizeSearch(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 'à', 'á', 'â', 'ã', 'ä', 'å', 'æ':
			return 'a'
		case 'ç':
			return 'c'
		case 'è', 'é', 'ê', 'ë':
			return 'e'
		case 'ì', 'í', 'î', 'ï':
			return 'i'
		case 'ñ':
			return 'n'
		case 'ò', 'ó', 'ô', 'õ', 'ö', 'ø', 'œ':
			return 'o'
		case 'ù', 'ú', 'û', 'ü':
			return 'u'
		case 'ý', 'ÿ':
			return 'y'
		default:
			return r
		}
	}, strings.ToLower(value))
}
