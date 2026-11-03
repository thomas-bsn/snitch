package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
)

// Person filter shared by search (all modes) and the library.
//
// A voice is attributed to a Person only through a traced attribution of its local
// speaker: the validated row if any, otherwise the machine row. Only the decision
// levels "human" and "automatic" identify a Person; a suggestion, an ambiguous
// decision or the absence of attribution is the unknown state. Nothing here ever
// recognises a voice outside speaker_attributions.

// personFilter is the parsed person_id / person_unknown query parameters.
type personFilter struct {
	PersonID int64
	Unknown  bool
}

func parsePersonFilter(q url.Values) (personFilter, bool) {
	var f personFilter
	if raw := q.Get("person_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 1 {
			return f, false
		}
		f.PersonID = id
	}
	switch q.Get("person_unknown") {
	case "", "false":
	case "true":
		f.Unknown = true
	default:
		return f, false
	}
	return f, !(f.PersonID > 0 && f.Unknown)
}

func (f personFilter) active() bool { return f.PersonID > 0 || f.Unknown }

// identifiedLevels are the decision levels that identify a Person.
const identifiedLevels = `('human','automatic')`

// effectiveAttributionOf is a correlated subselect that returns the effective
// attribution (validated first, else machine) of the local speaker `ls`.
func effectiveAttributionOf(ls string) string {
	return `SELECT a.person_id,a.decision_level FROM speaker_attributions a WHERE a.local_speaker_id=` + ls + `.id ORDER BY (a.validation_state='validated') DESC LIMIT 1`
}

// segmentSpeakerJoin joins, for one transcript segment, the effective attribution of
// the voice that speaks it. That voice is the Local speaker a user designated for the
// segment (segment_corrections.speaker_override_id, phase 6) when it still belongs to
// the same clip and audio stream, otherwise the one that speaks most over it (same rule
// as attachLocalSpeakers: maximal overlap, ties to the lowest ordinal). The alias
// exposes person_id and person_name (NULL unless identified) and level ('unknown' when
// nothing is attributed). Callers pass SQL expressions for the clip id, audio stream,
// segment start and end, and for the segment identity (transcription job id and
// segment index; NULL when unknown, which disables the override).
func segmentSpeakerJoin(alias, clip, stream, start, end, job, idx string) string {
	overlap := fmt.Sprintf("LEAST(%s,st.end_seconds)-GREATEST(%s,st.start_seconds)", end, start)
	return fmt.Sprintf(` LEFT JOIN LATERAL (SELECT CASE WHEN eff.decision_level IN %[7]s THEN eff.person_id END person_id,
 CASE WHEN eff.decision_level IN %[7]s THEN p.display_name END person_name,COALESCE(eff.decision_level,'unknown') level
 FROM (SELECT x.id FROM (
  SELECT ls.id,0 pri,0::double precision weight,ls.ordinal FROM segment_corrections sso JOIN local_speakers ls ON ls.id=sso.speaker_override_id WHERE sso.transcription_job_id=%[9]s AND sso.segment_index=%[10]s AND sso.stale_at IS NULL AND ls.kind<>'archived' AND ls.clip_id=%[2]s AND ls.audio_stream_index=%[3]s AND NOT EXISTS (SELECT 1 FROM local_speaker_dispositions dsp WHERE dsp.local_speaker_id=ls.id AND dsp.disposition='ignored')
  UNION ALL
  SELECT ls.id,1,-sum(%[6]s),ls.ordinal FROM speaker_turns st JOIN local_speakers ls ON ls.id=st.local_speaker_id WHERE ls.clip_id=%[2]s AND ls.kind<>'archived' AND ls.audio_stream_index=%[3]s AND %[6]s>0 AND NOT EXISTS (SELECT 1 FROM local_speaker_dispositions dsp WHERE dsp.local_speaker_id=ls.id AND dsp.disposition='ignored') GROUP BY ls.id,ls.ordinal) x ORDER BY x.pri,x.weight,x.ordinal LIMIT 1) dom
 LEFT JOIN LATERAL (%[8]s) eff ON true LEFT JOIN persons p ON p.id=eff.person_id) %[1]s ON true`,
		alias, clip, stream, start, end, overlap, identifiedLevels, effectiveAttributionOf("dom"), job, idx)
}

// segmentPersonPredicate filters rows joined with segmentSpeakerJoin. personArg and
// unknownArg are the SQL placeholders (for example $10 and $11).
func segmentPersonPredicate(alias, personArg, unknownArg string) string {
	return fmt.Sprintf("(%[2]s::bigint=0 OR %[1]s.person_id=%[2]s::bigint) AND (NOT %[3]s::boolean OR %[1]s.person_id IS NULL)", alias, personArg, unknownArg)
}

// clipPersonPredicate keeps the clips `c` that have a voice identified as the
// Person, or (unknown) that have a voice that is not identified or no Local speaker
// at all. This is the clip-level reading of the segment rule (segmentPersonPredicate):
// a segment without identified voice is unknown, including in a clip never diarized.
func clipPersonPredicate(personArg, unknownArg string) string {
	return fmt.Sprintf(`(%[1]s::bigint=0 OR EXISTS(SELECT 1 FROM local_speakers ls CROSS JOIN LATERAL (%[3]s) eff WHERE ls.clip_id=c.id AND ls.kind<>'archived' AND eff.decision_level IN %[4]s AND eff.person_id=%[1]s::bigint)) AND (NOT %[2]s::boolean OR NOT EXISTS(SELECT 1 FROM local_speakers ls WHERE ls.clip_id=c.id AND ls.kind<>'archived') OR EXISTS(SELECT 1 FROM local_speakers ls LEFT JOIN LATERAL (%[3]s) eff ON true WHERE ls.clip_id=c.id AND ls.kind<>'archived' AND (eff.person_id IS NULL OR eff.decision_level NOT IN %[4]s)))`,
		personArg, unknownArg, effectiveAttributionOf("ls"), identifiedLevels)
}

type clipPerson struct {
	ID            int64         `json:"id"`
	DisplayName   string        `json:"display_name"`
	DecisionLevel DecisionLevel `json:"decision_level"`
}

// clipPersons lists, per clip, the Persons identified in it. A Person who is both
// validated and matched in the same clip is reported at the human level.
func (s *Server) clipPersons(ctx context.Context, clipIDs []int64) (map[int64][]clipPerson, error) {
	out := map[int64][]clipPerson{}
	if len(clipIDs) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT ls.clip_id,p.id,p.display_name,eff.decision_level FROM local_speakers ls CROSS JOIN LATERAL (`+effectiveAttributionOf("ls")+`) eff JOIN persons p ON p.id=eff.person_id WHERE ls.clip_id=ANY($1) AND ls.kind<>'archived' AND eff.decision_level IN `+identifiedLevels, clipIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	index := map[[2]int64]int{}
	for rows.Next() {
		var clipID int64
		var v clipPerson
		if err = rows.Scan(&clipID, &v.ID, &v.DisplayName, &v.DecisionLevel); err != nil {
			return nil, err
		}
		key := [2]int64{clipID, v.ID}
		if at, seen := index[key]; seen {
			if v.DecisionLevel == LevelHuman {
				out[clipID][at].DecisionLevel = LevelHuman
			}
			continue
		}
		index[key] = len(out[clipID])
		out[clipID] = append(out[clipID], v)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for _, list := range out {
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].DisplayName != list[j].DisplayName {
				return list[i].DisplayName < list[j].DisplayName
			}
			return list[i].ID < list[j].ID
		})
	}
	return out, nil
}

// rejectPersonFilter answers 400 invalid_filter when the parameters are malformed.
func rejectPersonFilter(w http.ResponseWriter, r *http.Request) (personFilter, bool) {
	f, ok := parsePersonFilter(r.URL.Query())
	if !ok {
		fail(w, http.StatusBadRequest, "invalid_filter")
	}
	return f, ok
}
