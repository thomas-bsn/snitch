package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"snitch/internal/analysis"
)

// Groups of unknown voices (phase 5).
//
// A group is never stored. It is the connected set of unknown local speakers joined by
// proposed proximity links (voice_group_links, rebuilt from the stored fingerprints after
// every matching run), computed in SQL on every read: member and clip counts therefore
// always come from the database, and every unknown voice is a singleton group until a link
// joins it to another one. A link is only a proposal. Only an explicit naming by a human
// creates validations, exactly like validating the voices one by one (same attribution,
// same validated sample, same history, action "group_validate").
//
// Unknown means: diarized voice without human validation and without an automatic
// attribution. Suggestions and ambiguous voices are still unknown, like everywhere else.
// A member excluded by a human is kept in voice_group_exclusions: its own links are
// ignored, so it becomes a singleton and the other members are not modified (when the
// excluded member was the only path between two members, they are no longer joined
// unless a direct link exists, which nearest-neighbor links make rare).
//
// Groups are seen through the clips of the caller: hidden members are neither listed nor
// counted nor named, and the id of a group is its smallest visible member, so it never
// reveals a hidden voice. The review view reuses loadVoiceGroups.

const unknownSpeakerPredicate = analysis.UnknownSpeakerPredicate

// voiceGroupsCTE defines members(id, clip_id, clip_name, label, speaking_seconds, excluded,
// gid) for the caller ($1 user id, $2 administrator): the visible unknown voices with the
// id of their group. Visibility is applied before the connected components are computed,
// so a voice of a clip the caller cannot see never joins two voices they can see.
const voiceGroupsCTE = `WITH RECURSIVE eligible AS (
  SELECT ls.id,ls.clip_id,c.original_name AS clip_name,ls.label,ls.speaking_seconds,
         EXISTS (SELECT 1 FROM voice_group_exclusions x WHERE x.local_speaker_id=ls.id) AS excluded
  FROM local_speakers ls JOIN clips c ON c.id=ls.clip_id
  WHERE (c.visibility='shared' OR c.owner_id=$1 OR $2) AND ` + unknownSpeakerPredicate + `
), edges AS (
  SELECT l.speaker_a_id AS a,l.speaker_b_id AS b FROM voice_group_links l
    JOIN eligible x ON x.id=l.speaker_a_id AND NOT x.excluded JOIN eligible y ON y.id=l.speaker_b_id AND NOT y.excluded
  UNION ALL
  SELECT l.speaker_b_id,l.speaker_a_id FROM voice_group_links l
    JOIN eligible x ON x.id=l.speaker_a_id AND NOT x.excluded JOIN eligible y ON y.id=l.speaker_b_id AND NOT y.excluded
), reach(root,node) AS (
  SELECT id,id FROM eligible
  UNION
  SELECT r.root,e.b FROM reach r JOIN edges e ON e.a=r.node
), comp AS (
  SELECT node AS id,min(root) AS gid FROM reach GROUP BY node
), visible AS (
  SELECT e.id,e.clip_id,e.clip_name,e.label,e.speaking_seconds,e.excluded,comp.gid
  FROM eligible e JOIN comp ON comp.id=e.id
), members AS (
  SELECT v.*,min(v.id) OVER (PARTITION BY v.gid) AS group_id FROM visible v
)`

type voiceGroupMember struct {
	LocalSpeakerID  int64   `json:"local_speaker_id"`
	ClipID          int64   `json:"clip_id"`
	ClipName        string  `json:"clip_name"`
	Label           string  `json:"label"`
	SpeakingSeconds float64 `json:"speaking_seconds"`
	Excluded        bool    `json:"excluded"`
}

type voiceGroupView struct {
	ID          int64              `json:"id"`
	MemberCount int                `json:"member_count"`
	ClipCount   int                `json:"clip_count"`
	Proposed    bool               `json:"proposed"`
	Members     []voiceGroupMember `json:"members"`
}

type voiceGroupFilter struct {
	proposedOnly  bool
	sortByMembers bool
	limit, offset int
	id            int64 // a single group when > 0
	voice         int64 // the group holding this visible voice when > 0
}

func (s *Server) registerVoiceGroupRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/voice-groups", s.protected(false, http.HandlerFunc(s.listVoiceGroups)))
	mux.Handle("GET /api/voice-groups/{id}", s.protected(false, http.HandlerFunc(s.getVoiceGroup)))
	mux.Handle("POST /api/voice-groups/{id}/naming", s.protected(false, http.HandlerFunc(s.nameVoiceGroup)))
	mux.Handle("PUT /api/local-speakers/{id}/group-exclusion", s.protected(false, http.HandlerFunc(s.excludeGroupMember)))
	mux.Handle("DELETE /api/local-speakers/{id}/group-exclusion", s.protected(false, http.HandlerFunc(s.restoreGroupMember)))
}

// loadVoiceGroups returns the groups visible to the caller, biggest in clips first, with
// the total before pagination. Counts are aggregated by the database.
func (s *Server) loadVoiceGroups(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, userID int64, admin bool, f voiceGroupFilter) ([]voiceGroupView, int, error) {
	order := `clip_count DESC,member_count DESC,group_id`
	if f.sortByMembers {
		order = `member_count DESC,clip_count DESC,group_id`
	}
	rows, err := q.Query(ctx, voiceGroupsCTE+`, grouped AS (
  SELECT group_id,count(*)::int AS member_count,count(DISTINCT clip_id)::int AS clip_count FROM members GROUP BY group_id
  HAVING (NOT $3 OR count(*) > 1) AND ($4::bigint = 0 OR group_id = $4)
  AND ($7::bigint = 0 OR group_id IN (SELECT m.group_id FROM members m WHERE m.id = $7))
)
SELECT group_id,member_count,clip_count,count(*) OVER ()::int FROM grouped ORDER BY `+order+` LIMIT $5 OFFSET $6`, userID, admin, f.proposedOnly, f.id, f.limit, f.offset, f.voice)
	if err != nil {
		return nil, 0, err
	}
	groups := []voiceGroupView{}
	total := 0
	for rows.Next() {
		var g voiceGroupView
		if err = rows.Scan(&g.ID, &g.MemberCount, &g.ClipCount, &total); err != nil {
			rows.Close()
			return nil, 0, err
		}
		g.Proposed = g.MemberCount > 1
		g.Members = []voiceGroupMember{}
		groups = append(groups, g)
	}
	rows.Close()
	if err = rows.Err(); err != nil || len(groups) == 0 {
		return groups, total, err
	}
	ids := make([]int64, len(groups))
	index := make(map[int64]*voiceGroupView, len(groups))
	for i := range groups {
		ids[i] = groups[i].ID
		index[groups[i].ID] = &groups[i]
	}
	rows, err = q.Query(ctx, voiceGroupsCTE+` SELECT group_id,id,clip_id,clip_name,label,speaking_seconds,excluded FROM members WHERE group_id=ANY($3) ORDER BY group_id,clip_id,id`, userID, admin, ids)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var gid int64
		var m voiceGroupMember
		if err = rows.Scan(&gid, &m.LocalSpeakerID, &m.ClipID, &m.ClipName, &m.Label, &m.SpeakingSeconds, &m.Excluded); err != nil {
			return nil, 0, err
		}
		if g := index[gid]; g != nil {
			g.Members = append(g.Members, m)
		}
	}
	return groups, total, rows.Err()
}

func (s *Server) listVoiceGroups(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	query := r.URL.Query()
	f := voiceGroupFilter{limit: 20}
	switch query.Get("sort") {
	case "", "clips":
	case "members":
		f.sortByMembers = true
	default:
		fail(w, http.StatusBadRequest, "invalid_sort")
		return
	}
	if v := query.Get("proposed"); v != "" {
		proposed, err := strconv.ParseBool(v)
		if err != nil {
			fail(w, http.StatusBadRequest, "invalid_filter")
			return
		}
		f.proposedOnly = proposed
	}
	page := 1
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
		f.limit = n
	}
	f.offset = (page - 1) * f.limit
	if v := query.Get("voice"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			fail(w, http.StatusBadRequest, "invalid_filter")
			return
		}
		f.voice = n
	}
	groups, total, err := s.loadVoiceGroups(r.Context(), s.Pool, ss.User.ID, isAdministrator(ss.User.Role), f)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": groups, "total": total})
}

func (s *Server) getVoiceGroup(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	id, ok := pathID(w, r, "invalid_voice_group_id")
	if !ok {
		return
	}
	groups, _, err := s.loadVoiceGroups(r.Context(), s.Pool, ss.User.ID, isAdministrator(ss.User.Role), voiceGroupFilter{limit: 1, id: id})
	if err != nil {
		s.internal(w, err)
		return
	}
	if len(groups) == 0 {
		fail(w, http.StatusNotFound, "voice_group_not_found")
		return
	}
	reply(w, http.StatusOK, groups[0])
}

type voiceGroupExclusion struct {
	LocalSpeakerID int64 `json:"local_speaker_id"`
	Excluded       bool  `json:"excluded"`
}

// excludeGroupMember removes one voice from its group. Idempotent; refused for a voice
// that is no longer unknown (it is not part of any group).
func (s *Server) excludeGroupMember(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	var unknown bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM local_speakers ls WHERE ls.id=$1 AND `+unknownSpeakerPredicate+`)`, sp.ID).Scan(&unknown); err != nil {
		s.internal(w, err)
		return
	}
	if !unknown {
		fail(w, http.StatusConflict, "not_unknown")
		return
	}
	if _, err := s.Pool.Exec(r.Context(), `INSERT INTO voice_group_exclusions(local_speaker_id,excluded_by) VALUES($1,$2) ON CONFLICT DO NOTHING`, sp.ID, ss.User.ID); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, voiceGroupExclusion{LocalSpeakerID: sp.ID, Excluded: true})
}

func (s *Server) restoreGroupMember(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	if _, err := s.Pool.Exec(r.Context(), `DELETE FROM voice_group_exclusions WHERE local_speaker_id=$1`, sp.ID); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, voiceGroupExclusion{LocalSpeakerID: sp.ID, Excluded: false})
}

type voiceGroupNaming struct {
	PersonID        *int64  `json:"person_id"`
	NewPersonName   *string `json:"new_person_name"`
	LocalSpeakerIDs []int64 `json:"local_speaker_ids"`
}

// nameVoiceGroup validates every member of a group as one Person. The caller sends the
// voices they listened to: if the group is no longer exactly that set (a member was
// excluded, validated or linked meanwhile) nothing is written and the answer is 409
// group_changed. A member validated concurrently is skipped and reported; the others are
// validated. One transaction: the Person, the attributions, the samples and the history.
func (s *Server) nameVoiceGroup(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	groupID, ok := pathID(w, r, "invalid_voice_group_id")
	if !ok {
		return
	}
	var input voiceGroupNaming
	if !decode(w, r, &input) {
		return
	}
	name := ""
	if input.NewPersonName != nil {
		var valid bool
		if name, valid = cleanPersonName(*input.NewPersonName); !valid {
			fail(w, http.StatusBadRequest, "invalid_person")
			return
		}
	}
	if (input.PersonID == nil) == (input.NewPersonName == nil) || (input.PersonID != nil && *input.PersonID < 1) || len(input.LocalSpeakerIDs) == 0 || len(input.LocalSpeakerIDs) > 500 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	sent := map[int64]bool{}
	for _, id := range input.LocalSpeakerIDs {
		if id < 1 || sent[id] {
			fail(w, http.StatusBadRequest, "invalid_request")
			return
		}
		sent[id] = true
	}
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	groups, _, err := s.loadVoiceGroups(ctx, tx, ss.User.ID, isAdministrator(ss.User.Role), voiceGroupFilter{limit: 1, id: groupID})
	if err != nil {
		s.internal(w, err)
		return
	}
	if len(groups) == 0 {
		fail(w, http.StatusNotFound, "voice_group_not_found")
		return
	}
	members := groups[0].Members
	if len(members) != len(sent) {
		fail(w, http.StatusConflict, "group_changed")
		return
	}
	clipSet := map[int64]bool{}
	for _, m := range members {
		if !sent[m.LocalSpeakerID] {
			fail(w, http.StatusConflict, "group_changed")
			return
		}
		clipSet[m.ClipID] = true
	}
	// Same locks as a single validation, in a fixed order: a re-diarization (which takes
	// the clip FOR UPDATE and replaces the local speakers) cannot interleave.
	clips := make([]int64, 0, len(clipSet))
	for id := range clipSet {
		clips = append(clips, id)
	}
	sort.Slice(clips, func(i, j int) bool { return clips[i] < clips[j] })
	if _, err = tx.Exec(ctx, `SELECT id FROM clips WHERE id=ANY($1) ORDER BY id FOR SHARE`, clips); err != nil {
		s.internal(w, err)
		return
	}
	ids := append([]int64(nil), input.LocalSpeakerIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var locked int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM (SELECT id FROM local_speakers WHERE id=ANY($1) ORDER BY id FOR KEY SHARE) l`, ids).Scan(&locked); err != nil {
		s.internal(w, err)
		return
	}
	if locked != len(ids) {
		fail(w, http.StatusConflict, "group_changed")
		return
	}
	var personID int64
	if input.NewPersonName != nil {
		if err = tx.QueryRow(ctx, `INSERT INTO persons(display_name,created_by) VALUES($1,$2) RETURNING id`, name, ss.User.ID).Scan(&personID); err != nil {
			s.internal(w, err)
			return
		}
	} else {
		personID = *input.PersonID
		var optedOut bool
		if err = tx.QueryRow(ctx, `SELECT display_name,voice_profile_deleted_at IS NOT NULL FROM persons WHERE id=$1 FOR SHARE`, personID).Scan(&name, &optedOut); errors.Is(err, pgx.ErrNoRows) {
			fail(w, http.StatusNotFound, "person_not_found")
			return
		} else if err != nil {
			s.internal(w, err)
			return
		}
		if optedOut {
			// Same rule as a single validation: no new reference after a profile deletion.
			fail(w, http.StatusConflict, "voice_profile_deleted")
			return
		}
	}
	validated := []int64{}
	skipped := []int64{}
	for _, m := range members {
		done, validateErr := groupValidate(ctx, tx, m, personID, name, ss)
		if validateErr != nil {
			s.internal(w, validateErr)
			return
		}
		if done {
			validated = append(validated, m.LocalSpeakerID)
		} else {
			skipped = append(skipped, m.LocalSpeakerID)
		}
	}
	if len(validated) == 0 {
		fail(w, http.StatusConflict, "group_changed")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	// New references: the stored fingerprints are matched again and the proximity links
	// rebuilt, without any worker.
	s.matchingChanged(ctx, 0)
	person, err := scanPerson(s.Pool.QueryRow(ctx, `SELECT `+personColumns+` FROM persons p WHERE p.id=$1`, personID))
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"person": person, "validated": validated, "skipped": skipped})
}

// groupValidate is the first validation of one member (the same rows as
// changeAttribution) inside the naming transaction. It reports false when a human
// validation already exists for the voice.
func groupValidate(ctx context.Context, tx pgx.Tx, m voiceGroupMember, personID int64, personName string, ss session) (bool, error) {
	var oldPersonID *int64
	var oldName string
	var oldLevel DecisionLevel
	err := tx.QueryRow(ctx, `SELECT a.person_id,COALESCE(p.display_name,''),a.decision_level FROM speaker_attributions a LEFT JOIN persons p ON p.id=a.person_id WHERE a.local_speaker_id=$1 AND a.validation_state='unvalidated'`, m.LocalSpeakerID).Scan(&oldPersonID, &oldName, &oldLevel)
	if errors.Is(err, pgx.ErrNoRows) {
		oldLevel = LevelUnknown
	} else if err != nil {
		return false, err
	}
	var attributionID int64
	err = tx.QueryRow(ctx, `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,origin,author_id,author_username) VALUES($1,$2,'human','validated','human',$3,$4) ON CONFLICT (local_speaker_id) WHERE validation_state='validated' DO NOTHING RETURNING id`, m.LocalSpeakerID, personID, ss.User.ID, ss.User.Username).Scan(&attributionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return false, fmt.Errorf("local speaker %d or person %d disappeared: %w", m.LocalSpeakerID, personID, err)
		}
		return false, err
	}
	var sampleID int64
	if err = tx.QueryRow(ctx, `INSERT INTO voice_samples(person_id,local_speaker_id,clip_id,attribution_id,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id`, personID, m.LocalSpeakerID, m.ClipID, attributionID, ss.User.ID).Scan(&sampleID); err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO attribution_events(local_speaker_id,clip_id,action,old_person_id,old_person_name,old_decision_level,new_person_id,new_person_name,new_decision_level,new_sample_id,author_id,author_username) VALUES($1,$2,'group_validate',$3,$4,$5,$6,$7,'human',$8,$9,$10)`, m.LocalSpeakerID, m.ClipID, oldPersonID, oldName, oldLevel, personID, personName, sampleID, ss.User.ID, ss.User.Username)
	return err == nil, err
}

// refreshVoiceGroupLinks rebuilds the proposed proximity links between unknown voices
// from the stored fingerprints: for every unknown voice, its nearest neighbors (HNSW index
// of its model/version/dimensions group, database matching) at or above the suggestion threshold
// of the matching settings. Fingerprints of different models, versions or dimensions are
// never compared. Links are proposals only and written nowhere else.
func (s *Server) refreshVoiceGroupLinks(ctx context.Context) error {
	tx, err := s.beginLocked(ctx, lockVoiceGroups)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	settings, err := loadMatchingSettings(ctx, tx)
	if err != nil {
		return err
	}
	// The unknown filter is applied by the index scan itself, so the limit counts
	// unknown neighbors only.
	if _, err = tx.Exec(ctx, `SET LOCAL hnsw.iterative_scan = relaxed_order`); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT DISTINCT f.model,f.model_version,f.dimensions FROM voice_fingerprints f JOIN local_speakers ls ON ls.id=f.local_speaker_id WHERE f.usable AND f.scope='speaker' AND `+unknownSpeakerPredicate)
	if err != nil {
		return err
	}
	var groups []matchGroup
	for rows.Next() {
		var g matchGroup
		if err = rows.Scan(&g.model, &g.version, &g.dimensions); err != nil {
			rows.Close()
			return err
		}
		groups = append(groups, g)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM voice_group_links`); err != nil {
		return err
	}
	for _, g := range groups {
		query := fmt.Sprintf(`INSERT INTO voice_group_links(speaker_a_id,speaker_b_id,similarity,model,model_version,dimensions)
SELECT least(t.sid,n.sid),greatest(t.sid,n.sid),max(n.sim),$1::text,$2::text,%[1]d
FROM (SELECT f.local_speaker_id AS sid,f.embedding::vector(%[1]d) AS e FROM voice_fingerprints f JOIN local_speakers ls ON ls.id=f.local_speaker_id
      WHERE f.usable AND f.scope='speaker' AND f.model=$1 AND f.model_version=$2 AND f.dimensions=%[1]d AND `+unknownSpeakerPredicate+`) t
CROSS JOIN LATERAL (
  SELECT f2.local_speaker_id AS sid,1-(f2.embedding::vector(%[1]d) <=> t.e) AS sim
  FROM voice_fingerprints f2 JOIN local_speakers ls ON ls.id=f2.local_speaker_id
  WHERE f2.usable AND f2.scope='speaker' AND f2.model=$1 AND f2.model_version=$2 AND f2.dimensions=%[1]d AND f2.local_speaker_id<>t.sid AND `+unknownSpeakerPredicate+`
  ORDER BY f2.embedding::vector(%[1]d) <=> t.e LIMIT 20) n
WHERE n.sim>=$3
GROUP BY 1,2`, g.dimensions)
		if _, err = tx.Exec(ctx, query, g.model, g.version, settings.SuggestionThreshold); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
