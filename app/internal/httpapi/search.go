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
	ClipID      int64      `json:"clip_id"`
	ClipName    string     `json:"clip_name"`
	Visibility  string     `json:"visibility"`
	RecordedAt  *time.Time `json:"recorded_at"`
	Status      string     `json:"status"`
	Sources     []string   `json:"sources"`
	MatchType   string     `json:"match_type"`
	Context     string     `json:"context"`
	MatchedText string     `json:"matched_text"`
	Timestamp   float64    `json:"timestamp"`
	// Corrected tells that Context is the corrected text of the segment (phase 6), not
	// the original transcription.
	Corrected bool `json:"corrected"`
	// Person is the identified Person of the voice that speaks the segment (human or
	// automatic attribution), nil otherwise; DecisionLevel is the traced level of the
	// effective attribution, "unknown" when the voice is not attributed.
	Person        *personRef `json:"person"`
	DecisionLevel string     `json:"decision_level"`
}

// scanSpeaker fills the Person of the speaker columns shared by every search query.
func (item *searchItem) scanSpeaker(personID *int64, personName *string) {
	if personID != nil && personName != nil {
		item.Person = &personRef{ID: *personID, DisplayName: *personName}
	}
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
	person, ok := rejectPersonFilter(w, r)
	if !ok {
		return
	}
	admin := isAdministrator(ss.User.Role)
	if mode == "hybrid" {
		s.hybridSearch(w, r, q, page, size, visibility, status, sourceID, from, to, admin, person)
		return
	}
	// Translate composes with lower() to match French accented and unaccented input
	// without requiring a database extension or changing the worker result format.
	queryNorm := normalizeSearch(q)
	// An exact query is a word or a phrase: the words must follow each other, while the
	// punctuation between them is ignored. The transcript is normalised the same way, so
	// "tres bien" finds "Très bien, dit" and "bientot" finds "À bientôt".
	phraseNorm := "btrim(regexp_replace(" + fmtNorm("$1") + ",'[^[:alnum:]]+',' ','g'))"
	exactWord := fmtNorm("wd.w") + "=" + phraseNorm
	exactMatch := "(" + phraseNorm + " <> '' AND (' ' || pn || ' ') LIKE ('% ' || " + phraseNorm + " || ' %'))"
	match := exactMatch
	fuzzy := "FALSE"
	if mode == "full_text" {
		match = "(wn=$1 OR sn LIKE '%' || $1 || '%')"
		// pg_trgm is only used for sufficiently long single transcript words;
		// short/common queries would otherwise produce noisy matches.
		fuzzy = "(char_length($1)>=4 AND char_length(wn)>=4 AND similarity(wn,$1)>=$9)"
		match = "(" + match + " OR " + fuzzy + ")"
	}
	// Exact mode returns one result per passage, not one per word: the matched text is the
	// word when it is known, the normalised query otherwise, and a phrase starts at its segment.
	matchedText := "wd.w"
	timestamp := "COALESCE(wd.ts,e.seg_start)"
	matchType := "CASE WHEN wd.w IS NOT NULL AND " + exactWord + " THEN 'exact' WHEN " + fmtNorm("e.text") + " LIKE '%' || $1 || '%' THEN 'full_text' ELSE 'approximate' END"
	if mode == "exact" {
		matchedText = "CASE WHEN wd.w IS NOT NULL AND " + exactWord + " THEN wd.w ELSE " + phraseNorm + " END"
		timestamp = "CASE WHEN wd.w IS NOT NULL AND " + exactWord + " THEN COALESCE(wd.ts,e.seg_start) ELSE e.seg_start END"
		matchType = "'exact'"
	}
	// The text searched and displayed is the effective text of each segment (the
	// correction when there is one). The words of a corrected segment are those of the
	// corrected text, all dated at the segment start: the original words of a corrected
	// segment never match, so the replaced text cannot be found.
	baseQuery := `WITH ` + visibleTranscriptsCTE("") + `, ` + effectiveSegmentsCTE() + `, parts AS (
 SELECT c.id clip_id,COALESCE(c.display_title,c.original_name) clip_name,c.visibility,c.capture_time AS recorded_at,
 CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END status,
 COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb) sources,
	 e.text context, e.corrected, e.job_id, e.idx, e.seg_start segment_start, e.seg_end segment_end, e.stream,
	 ` + matchedText + ` matched_text,
	 ` + timestamp + ` timestamp,
	 ` + matchType + ` match_type,
 ` + fmtNorm("e.text") + ` sn, ` + fmtNorm("wd.w") + ` wn, btrim(regexp_replace(` + fmtNorm("e.text") + `,'[^[:alnum:]]+',' ','g')) pn
 FROM eff_segments e JOIN clips c ON c.id=e.clip_id
 LEFT JOIN LATERAL (
  SELECT word->>'word' w, COALESCE((word->>'start')::double precision,e.seg_start) ts FROM jsonb_array_elements(COALESCE(e.seg->'words','[]'::jsonb)) word WHERE NOT e.corrected
  UNION ALL
  SELECT w, e.seg_start FROM (SELECT NULLIF(regexp_replace(x,'^[[:punct:]]+|[[:punct:]]+$','','g'),'') w FROM regexp_split_to_table(btrim(e.text),'\s+') x) cw WHERE e.corrected AND w IS NOT NULL
 ) wd ON true
 WHERE clip_readable(c.id,c.owner_id,c.visibility,$2,$3)
 AND ($4='' OR c.visibility=$4) AND ($5=0 OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.source_id=$5))
 AND ($6='' OR (c.capture_time AT TIME ZONE COALESCE((SELECT NULLIF(timezone,'') FROM clip_metadata_settings WHERE id=true),'UTC'))::date >= $6::date) AND ($7='' OR (c.capture_time AT TIME ZONE COALESCE((SELECT NULLIF(timezone,'') FROM clip_metadata_settings WHERE id=true),'UTC'))::date <= $7::date)
 AND ($8='' OR CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END=$8)
)`
	// Matching rows (deduplicated per segment in full-text mode) are enriched with the
	// speaker of the segment; the Person filter, then total and page, apply to them.
	cols := "clip_id,clip_name,visibility,recorded_at,status,sources,context,matched_text,timestamp,match_type,segment_start,segment_end,stream,wn,corrected,job_id,idx"
	matches := "SELECT " + cols + " FROM parts WHERE (" + match + ") AND $9::double precision>=0"
	if mode == "full_text" {
		matches = "SELECT DISTINCT ON (clip_id,job_id,idx) " + cols + " FROM parts WHERE (" + match + ") AND $9::double precision>=0 ORDER BY clip_id,job_id,idx,CASE match_type WHEN 'exact' THEN 0 WHEN 'full_text' THEN 1 ELSE 2 END,similarity(wn,$1) DESC,timestamp"
	}
	if mode == "exact" {
		// The passage matches on every word row of its segment: keep one row per passage,
		// preferring the word row that is the query itself (its timestamp and spelling),
		// then the earliest, so the pick is deterministic.
		matches = "SELECT DISTINCT ON (clip_id,job_id,idx) " + cols + " FROM parts WHERE (" + match + ") AND $9::double precision>=0 ORDER BY clip_id,job_id,idx,CASE WHEN wn=" + phraseNorm + " THEN 0 ELSE 1 END,timestamp"
	}
	enrich := " FROM matches m" + segmentSpeakerJoin("spk", "m.clip_id", "m.stream", "m.segment_start", "m.segment_end", "m.job_id", "m.idx") + " WHERE " + segmentPersonPredicate("spk", "$10", "$11")
	withMatches := baseQuery + ", matches AS (" + matches + ") "
	countQuery := withMatches + "SELECT count(*)" + enrich
	query := withMatches + "SELECT m.clip_id,m.clip_name,m.visibility,m.recorded_at,m.status,m.sources,m.context,COALESCE(m.matched_text,''),m.timestamp,m.match_type,m.corrected,spk.person_id,spk.person_name,COALESCE(spk.level,'unknown')" + enrich + " ORDER BY CASE m.match_type WHEN 'exact' THEN 0 WHEN 'full_text' THEN 1 ELSE 2 END,m.recorded_at DESC NULLS LAST,m.clip_id DESC,m.timestamp LIMIT $12 OFFSET $13"
	args := []any{queryNorm, ss.User.ID, admin, visibility, sourceID, from, to, status}
	var total int
	if err = s.Pool.QueryRow(r.Context(), countQuery, append(args, 0.30, person.PersonID, person.Unknown)...).Scan(&total); err != nil {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(r.Context(), query, append(args, 0.30, person.PersonID, person.Unknown, size, (page-1)*size)...)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []searchItem{}
	for rows.Next() {
		var item searchItem
		var personID *int64
		var personName *string
		if err = rows.Scan(&item.ClipID, &item.ClipName, &item.Visibility, &item.RecordedAt, &item.Status, &item.Sources, &item.Context, &item.MatchedText, &item.Timestamp, &item.MatchType, &item.Corrected, &personID, &personName, &item.DecisionLevel); err != nil {
			s.internal(w, err)
			return
		}
		item.scanSpeaker(personID, personName)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]any{"items": items, "page": page, "page_size": size, "total": total})
}

func (s *Server) hybridSearch(w http.ResponseWriter, r *http.Request, q string, page, size int, visibility, status string, sourceID int64, from, to string, admin bool, person personFilter) {
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
	settings, settingsErr := s.embeddingSettings(r.Context())
	if settingsErr != nil || settings.URL == "" || settings.Model == "" {
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
	filter := `clip_readable(c.id,c.owner_id,c.visibility,$2,$3) AND ($4='' OR c.visibility=$4) AND ($5=0 OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.source_id=$5)) AND ($6='' OR (c.capture_time AT TIME ZONE COALESCE((SELECT NULLIF(timezone,'') FROM clip_metadata_settings WHERE id=true),'UTC'))::date >= $6::date) AND ($7='' OR (c.capture_time AT TIME ZONE COALESCE((SELECT NULLIF(timezone,'') FROM clip_metadata_settings WHERE id=true),'UTC'))::date <= $7::date) AND ($8='' OR CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END=$8)`
	args := []any{vector, ss.User.ID, admin, visibility, sourceID, from, to, status}
	// The Person filter applies to the semantic candidates before their 100-row cut.
	speaker := segmentSpeakerJoin("spk", "c.id", "v.stream", "v.seg_start", "v.seg_end", "e.transcription_job_id", "e.segment_index")
	filter += " AND " + segmentPersonPredicate("spk", "$11", "$12")
	query := fmt.Sprintf(`SELECT e.clip_id,COALESCE(c.display_title,c.original_name),c.visibility,c.capture_time,CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN 'ready' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='unreadable') THEN 'unreadable' WHEN EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='waiting_stability') THEN 'waiting_stability' ELSE 'missing' END,COALESCE((SELECT jsonb_agg(DISTINCT cs.name) FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id),'[]'::jsonb),COALESCE(sc.text,e.text),'',v.seg_start,'semantic',(sc.text IS NOT NULL),spk.person_id,spk.person_name,COALESCE(spk.level,'unknown') FROM transcript_embeddings e JOIN clips c ON c.id=e.clip_id JOIN jobs j ON j.id=e.transcription_job_id JOIN audio_visible_segments v ON v.job_id=e.transcription_job_id AND v.idx=e.segment_index LEFT JOIN segment_corrections sc ON sc.transcription_job_id=e.transcription_job_id AND sc.segment_index=e.segment_index AND sc.stale_at IS NULL %s WHERE e.profile_id=%d AND NOT COALESCE(sc.reindex_pending,false) AND EXISTS(SELECT 1 FROM audio_visible_segments visible WHERE visible.job_id=j.id AND visible.idx=e.segment_index) AND `+filter+` ORDER BY e.embedding::halfvec(%d) <=> $1::halfvec(%d) LIMIT $9 OFFSET $10`, speaker, profileID, dimensions, dimensions)
	rows, err := s.Pool.Query(r.Context(), query, append(args, 100, 0, person.PersonID, person.Unknown)...)
	if err != nil {
		fallback()
		return
	}
	defer rows.Close()
	semanticItems := []searchItem{}
	for rows.Next() {
		var it searchItem
		var personID *int64
		var personName *string
		if err = rows.Scan(&it.ClipID, &it.ClipName, &it.Visibility, &it.RecordedAt, &it.Status, &it.Sources, &it.Context, &it.MatchedText, &it.Timestamp, &it.MatchType, &it.Corrected, &personID, &personName, &it.DecisionLevel); err != nil {
			s.internal(w, err)
			return
		}
		it.scanSpeaker(personID, personName)
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
