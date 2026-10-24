package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
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
	if mode != "exact" && mode != "full_text" && mode != "hybrid" {
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
	if mode == "hybrid" {
		s.hybridSearch(w, r, q, page, size, visibility, status, sourceID, from, to, admin)
		return
	}
	// Translate composes with lower() to match French accented and unaccented input
	// without requiring a database extension or changing the worker result format.
	queryNorm := normalizeSearch(q)
	match := "wn=$1"
	fuzzy := "FALSE"
	if mode == "full_text" {
		match = "(wn=$1 OR sn LIKE '%' || $1 || '%')"
		// pg_trgm is only used for sufficiently long single transcript words;
		// short/common queries would otherwise produce noisy matches.
		fuzzy = "(char_length($1)>=4 AND char_length(wn)>=4 AND similarity(wn,$1)>=$9)"
		match = "(" + match + " OR " + fuzzy + ")"
	}
	baseQuery := `WITH transcript AS (
 SELECT DISTINCT ON (j.clip_id) j.clip_id,j.result FROM jobs j WHERE j.status='done' AND j.step LIKE 'transcription.audio.%' ORDER BY j.clip_id,j.updated_at DESC,j.id DESC
), parts AS (
 SELECT c.id clip_id,c.original_name clip_name,c.visibility,c.recorded_at,
 CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END status,
 COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb) sources,
	 seg->>'text' context, COALESCE((seg->>'start')::double precision,0) segment_start, word->>'word' matched_text, COALESCE((word->>'start')::double precision,(seg->>'start')::double precision,0) timestamp,
	 CASE WHEN word IS NOT NULL AND ` + fmtNorm("word->>'word'") + `=$1 THEN 'exact' WHEN ` + fmtNorm("seg->>'text'") + ` LIKE '%' || $1 || '%' THEN 'full_text' ELSE 'approximate' END match_type,
 ` + fmtNorm("seg->>'text'") + ` sn, ` + fmtNorm("word->>'word'") + ` wn
 FROM transcript t JOIN clips c ON c.id=t.clip_id CROSS JOIN LATERAL jsonb_array_elements(COALESCE(t.result->'segments','[]'::jsonb)) seg
 LEFT JOIN LATERAL jsonb_array_elements(COALESCE(seg->'words','[]'::jsonb)) word ON true
 WHERE (c.visibility='shared' OR c.owner_id=$2 OR $3)
 AND ($4='' OR c.visibility=$4) AND ($5=0 OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.source_id=$5))
 AND ($6='' OR c.recorded_at::date >= $6::date) AND ($7='' OR c.recorded_at::date <= $7::date)
 AND ($8='' OR CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END=$8)
)
	`
	countQuery := `WITH transcript AS (SELECT DISTINCT ON (j.clip_id) j.clip_id,j.result FROM jobs j WHERE j.status='done' AND j.step LIKE 'transcription.audio.%' ORDER BY j.clip_id,j.updated_at DESC,j.id DESC), parts AS (SELECT c.id clip_id,seg->>'text' context,COALESCE((seg->>'start')::double precision,0) segment_start,word->>'word' matched_text,` + fmtNorm("seg->>'text'") + ` sn,` + fmtNorm("word->>'word'") + ` wn FROM transcript t JOIN clips c ON c.id=t.clip_id CROSS JOIN LATERAL jsonb_array_elements(COALESCE(t.result->'segments','[]'::jsonb)) seg LEFT JOIN LATERAL jsonb_array_elements(COALESCE(seg->'words','[]'::jsonb)) word ON true WHERE (c.visibility='shared' OR c.owner_id=$2 OR $3) AND ($4='' OR c.visibility=$4) AND ($5=0 OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.source_id=$5)) AND ($6='' OR c.recorded_at::date >= $6::date) AND ($7='' OR c.recorded_at::date <= $7::date) AND ($8='' OR CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END=$8)) SELECT count(*) FROM parts WHERE ` + match + ` AND $9::double precision>=0`
	resultQuery := "SELECT clip_id,clip_name,visibility,recorded_at,status,sources,context,matched_text,timestamp,match_type FROM parts WHERE (" + match + ") AND $9::double precision>=0"
	if mode == "full_text" {
		resultQuery = "SELECT clip_id,clip_name,visibility,recorded_at,status,sources,context,matched_text,timestamp,match_type FROM (SELECT DISTINCT ON (clip_id,context,segment_start) clip_id,clip_name,visibility,recorded_at,status,sources,context,matched_text,timestamp,match_type,segment_start,wn FROM parts WHERE (" + match + ") AND $9::double precision>=0 ORDER BY clip_id,context,segment_start,CASE match_type WHEN 'exact' THEN 0 WHEN 'full_text' THEN 1 ELSE 2 END,similarity(wn,$1) DESC,timestamp) matches"
		countQuery = strings.Replace(countQuery, "SELECT count(*) FROM parts WHERE "+match, "SELECT count(DISTINCT (clip_id,context,segment_start)) FROM parts WHERE "+match, 1)
	}
	query := baseQuery + resultQuery + " ORDER BY CASE match_type WHEN 'exact' THEN 0 WHEN 'full_text' THEN 1 ELSE 2 END,recorded_at DESC,clip_id DESC,timestamp LIMIT $10 OFFSET $11"
	args := []any{queryNorm, ss.User.ID, admin, visibility, sourceID, from, to, status}
	var total int
	if err = s.Pool.QueryRow(r.Context(), countQuery, append(args, 0.30)...).Scan(&total); err != nil {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(r.Context(), query, append(args, 0.30, size, (page-1)*size)...)
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

func (s *Server) hybridSearch(w http.ResponseWriter, r *http.Request, q string, page, size int, visibility, status string, sourceID int64, from, to string, admin bool) {
	fallback := func() {
		r2 := r.Clone(r.Context())
		u := *r.URL
		values := u.Query()
		values.Set("mode", "full_text")
		u.RawQuery = values.Encode()
		r2.URL = &u
		rec := httptest.NewRecorder()
		s.search(rec, r2)
		var result map[string]any
		if json.Unmarshal(rec.Body.Bytes(), &result) != nil {
			w.WriteHeader(rec.Code)
			_, _ = w.Write(rec.Body.Bytes())
			return
		}
		result["degraded"] = true
		reply(w, rec.Code, result)
	}
	if s.Config.EmbeddingsURL == "" || s.Config.EmbeddingsModel == "" {
		fallback()
		return
	}
	vector, err := s.vectorFor(r.Context(), q)
	if err != nil {
		fallback()
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	var profileID int64
	var dimensions int
	if err = s.Pool.QueryRow(r.Context(), `SELECT p.id,COALESCE(p.dimensions,0) FROM embedding_index_state i JOIN embedding_profiles p ON p.id=i.active_profile_id WHERE i.id=1`).Scan(&profileID, &dimensions); err != nil || dimensions < 1 {
		fallback()
		return
	}
	filter := `(c.visibility='shared' OR c.owner_id=$2 OR $3) AND ($4='' OR c.visibility=$4) AND ($5=0 OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.source_id=$5)) AND ($6='' OR c.recorded_at::date >= $6::date) AND ($7='' OR c.recorded_at::date <= $7::date) AND ($8='' OR CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END=$8)`
	args := []any{vector, ss.User.ID, admin, visibility, sourceID, from, to, status}
	query := fmt.Sprintf(`SELECT e.clip_id,c.original_name,c.visibility,c.recorded_at,CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END,COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb),e.text,'',e.segment_start,'semantic' FROM transcript_embeddings e JOIN clips c ON c.id=e.clip_id JOIN jobs j ON j.id=e.transcription_job_id WHERE e.profile_id=%d AND j.status='done' AND j.step LIKE 'transcription.audio.%%' AND NOT EXISTS(SELECT 1 FROM jobs newer WHERE newer.clip_id=j.clip_id AND newer.status='done' AND newer.step LIKE 'transcription.audio.%%' AND (newer.updated_at>j.updated_at OR (newer.updated_at=j.updated_at AND newer.id>j.id))) AND `+filter+` ORDER BY e.embedding::halfvec(%d) <=> $1::halfvec(%d) LIMIT $9 OFFSET $10`, profileID, dimensions, dimensions)
	rows, err := s.Pool.Query(r.Context(), query, append(args, 100, 0)...)
	if err != nil {
		fallback()
		return
	}
	defer rows.Close()
	semanticItems := []searchItem{}
	for rows.Next() {
		var it searchItem
		if err = rows.Scan(&it.ClipID, &it.ClipName, &it.Visibility, &it.RecordedAt, &it.Status, &it.Sources, &it.Context, &it.MatchedText, &it.Timestamp, &it.MatchType); err != nil {
			s.internal(w, err)
			return
		}
		semanticItems = append(semanticItems, it)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	rows.Close()
	// Run the existing permission-filtered lexical search, then combine ranks with reciprocal rank fusion.
	r2 := r.Clone(r.Context())
	u := *r.URL
	values := u.Query()
	values.Set("mode", "full_text")
	values.Set("page", "1")
	values.Set("page_size", "100")
	u.RawQuery = values.Encode()
	r2.URL = &u
	lexicalResponse := httptest.NewRecorder()
	s.search(lexicalResponse, r2)
	var lexical struct {
		Items []searchItem `json:"items"`
		Total int          `json:"total"`
	}
	if lexicalResponse.Code != http.StatusOK || json.Unmarshal(lexicalResponse.Body.Bytes(), &lexical) != nil {
		fallback()
		return
	}
	type ranked struct {
		item  searchItem
		score float64
	}
	combined := map[string]ranked{}
	key := func(item searchItem) string {
		return fmt.Sprintf("%d:%s:%d", item.ClipID, strings.ToLower(strings.TrimSpace(item.Context)), int(item.Timestamp/10))
	}
	for rank, item := range semanticItems {
		k := key(item)
		combined[k] = ranked{item: item, score: 1 / float64(60+rank+1)}
	}
	for rank, item := range lexical.Items {
		k := key(item)
		entry := combined[k]
		if entry.score > 0 {
			entry.score += 1 / float64(60+rank+1)
			item.MatchType = "hybrid"
		} else {
			entry = ranked{item: item, score: 1 / float64(60+rank+1)}
		}
		entry.item = item
		combined[k] = entry
	}
	ordered := make([]ranked, 0, len(combined))
	for _, v := range combined {
		ordered = append(ordered, v)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].score > ordered[j].score })
	items := make([]searchItem, 0, size)
	start := (page - 1) * size
	for i := start; i < len(ordered) && i < start+size; i++ {
		items = append(items, ordered[i].item)
	}
	total := len(ordered)
	reply(w, 200, map[string]any{"items": items, "page": page, "page_size": size, "total": total, "degraded": false})
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
