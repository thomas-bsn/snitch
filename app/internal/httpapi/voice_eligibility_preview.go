package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// Preview and traced requalification of the stored fingerprints (#394, spec voix-breves section 6, steps 1 and
// 2). The preview decides the production speaker fingerprints under a candidate rule and under a few minimum
// durations, with the statement of the requalification itself (eligibilityDecidedSQL), inside a read-only
// transaction: it writes nothing and reads no media. The requalification is the rule's save
// (putVoiceEligibility); each one is traced with its author, its rule and a before/after report.
//
// A Person has a Reference, in a fingerprint group, when at least one of their active validated samples has a
// contributive speaker fingerprint in that group (the definition of the cumulative Reference, ADR 0064).

// defaultPreviewDurations are the minimum durations of the preview's scenarios, each applied to both uses.
var defaultPreviewDurations = []float64{1, 1.5, 2}

const maxPreviewDurations = 6

// clipSourceAbsentSQL is true when the database knows no audio source of the clip: no ready file in an enabled
// source and no uploaded media. It reads no file: a source the database believes ready may still turn out
// unreadable when a job reads it. The recalculation of the fingerprints (#395) counts its missing sources with it.
func clipSourceAbsentSQL(clip string) string {
	return `(NOT EXISTS (SELECT 1 FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=` + clip + ` AND sf.status='ready' AND cs.enabled)
  AND NOT EXISTS (SELECT 1 FROM uploaded_media um WHERE um.clip_id=` + clip + `))`
}

// activeSamplesSQL lists the active validated samples whose Person keeps a voice profile: those a Reference is
// made of. At most one per local speaker.
const activeSamplesSQL = `SELECT vs.local_speaker_id,vs.person_id FROM voice_samples vs
  WHERE vs.retired_at IS NULL AND NOT EXISTS (SELECT 1 FROM persons p WHERE p.id=vs.person_id AND p.voice_profile_deleted_at IS NOT NULL)`

// eligibilityOutcome counts production speaker fingerprints by eligibility. Contributive includes comparable;
// the reasons count the fingerprints that are not even contributive.
type eligibilityOutcome struct {
	Present              int `json:"present"`
	Comparable           int `json:"comparable"`
	Contributive         int `json:"contributive"`
	TooShort             int `json:"too_short"`
	OverlapOnly          int `json:"overlap_only"`
	Clipped              int `json:"clipped"`
	LowQuality           int `json:"low_quality"`
	PersonsWithReference int `json:"persons_with_reference"`
}

type eligibilityScenario struct {
	Rule eligibilityValues `json:"rule"`
	eligibilityOutcome
	RecoveredComparable   int `json:"recovered_comparable"`
	RecoveredContributive int `json:"recovered_contributive"`
	LostComparable        int `json:"lost_comparable"`
	LostContributive      int `json:"lost_contributive"`
}

type eligibilityPersonPreview struct {
	Person         personRef `json:"person"`
	Present        int       `json:"present"`
	Comparable     int       `json:"comparable"`
	Contributive   int       `json:"contributive"`
	TooShort       int       `json:"too_short"`
	OverlapOnly    int       `json:"overlap_only"`
	Clipped        int       `json:"clipped"`
	MissingSources int       `json:"missing_sources"`
	Estimated      int       `json:"estimated"`
	Reference      bool      `json:"reference"`
	ReferenceNow   bool      `json:"reference_now"`
}

type eligibilityGroupPreview struct {
	Model           string                     `json:"model"`
	ModelVersion    string                     `json:"model_version"`
	Dimensions      int                        `json:"dimensions"`
	MissingSources  int                        `json:"missing_sources"`
	Estimated       int                        `json:"estimated"`
	OriginalVectors int                        `json:"original_vectors"`
	OverlapShare    *float64                   `json:"overlap_share"`
	Current         eligibilityOutcome         `json:"current"`
	Candidate       eligibilityScenario        `json:"candidate"`
	Scenarios       []eligibilityScenario      `json:"scenarios"`
	Persons         []eligibilityPersonPreview `json:"persons"`
}

type eligibilityPreview struct {
	ActiveVersion      int                       `json:"active_version"`
	PersonsWithSamples int                       `json:"persons_with_samples"`
	Groups             []eligibilityGroupPreview `json:"groups"`
}

type eligibilityRequalification struct {
	ID              int64              `json:"id"`
	RuleVersion     int                `json:"rule_version"`
	Author          *string            `json:"author"`
	CreatedAt       time.Time          `json:"created_at"`
	Changed         int                `json:"changed"`
	MissingSources  int                `json:"missing_sources"`
	Estimated       int                `json:"estimated"`
	OriginalVectors int                `json:"original_vectors"`
	Before          eligibilityOutcome `json:"before"`
	After           eligibilityOutcome `json:"after"`
}

// previewRuleCTE turns the candidate rule and the scenarios ($1, $2, $3: comparable, contributive and clipped
// arrays) into the rule CTE; version is the scenario's position, 1 being the candidate.
const previewRuleCTE = `WITH rule AS (
  SELECT k::int AS version,c AS comparable_min_seconds,b AS contributive_min_seconds,x AS max_clipped_ratio
  FROM unnest($1::float8[],$2::float8[],$3::float8[]) WITH ORDINALITY AS u(c,b,x,k)
), `

var previewGroupsSQL = previewRuleCTE + eligibilityDecidedSQL(`f.scope='speaker'`) + `
SELECT d.version,d.model,d.model_version,d.dimensions,
  count(*),count(*) FILTER (WHERE d.reason=''),count(*) FILTER (WHERE d.now_contributive),
  count(*) FILTER (WHERE NOT d.now_contributive AND d.reason='too_short'),
  count(*) FILTER (WHERE NOT d.now_contributive AND d.reason='overlap_only'),
  count(*) FILTER (WHERE NOT d.now_contributive AND d.reason='clipped'),
  count(DISTINCT s.person_id) FILTER (WHERE d.now_contributive),
  count(*) FILTER (WHERE d.reason='' AND NOT d.usable),count(*) FILTER (WHERE d.now_contributive AND NOT d.contributive),
  count(*) FILTER (WHERE d.reason<>'' AND d.usable),count(*) FILTER (WHERE NOT d.now_contributive AND d.contributive),
  count(*) FILTER (WHERE d.usable),count(*) FILTER (WHERE d.contributive),
  count(*) FILTER (WHERE NOT d.contributive AND d.unusable_reason='too_short'),
  count(*) FILTER (WHERE NOT d.contributive AND d.unusable_reason='overlap_only'),
  count(*) FILTER (WHERE NOT d.contributive AND d.unusable_reason='clipped'),
  count(*) FILTER (WHERE NOT d.contributive AND d.unusable_reason='low_quality'),
  count(DISTINCT s.person_id) FILTER (WHERE d.contributive),
  count(*) FILTER (WHERE ` + clipSourceAbsentSQL("d.clip_id") + `),
  count(*) FILTER (WHERE d.net_source='estimated'),count(*) FILTER (WHERE d.original_vector),
  sum(d.overlap_ratio*ls.speaking_seconds) FILTER (WHERE d.overlap_ratio IS NOT NULL)/NULLIF(sum(ls.speaking_seconds) FILTER (WHERE d.overlap_ratio IS NOT NULL),0)
FROM decided d JOIN local_speakers ls ON ls.id=d.local_speaker_id LEFT JOIN (` + activeSamplesSQL + `) s ON s.local_speaker_id=d.local_speaker_id
GROUP BY d.version,d.model,d.model_version,d.dimensions ORDER BY d.model,d.model_version,d.dimensions,d.version`

var previewPersonsSQL = previewRuleCTE + eligibilityDecidedSQL(`f.scope='speaker'`) + `
SELECT d.model,d.model_version,d.dimensions,p.id,p.display_name,
  count(*),count(*) FILTER (WHERE d.reason=''),count(*) FILTER (WHERE d.now_contributive),
  count(*) FILTER (WHERE NOT d.now_contributive AND d.reason='too_short'),
  count(*) FILTER (WHERE NOT d.now_contributive AND d.reason='overlap_only'),
  count(*) FILTER (WHERE NOT d.now_contributive AND d.reason='clipped'),
  count(*) FILTER (WHERE ` + clipSourceAbsentSQL("d.clip_id") + `),count(*) FILTER (WHERE d.net_source='estimated'),
  bool_or(d.now_contributive),bool_or(d.contributive)
FROM decided d JOIN (` + activeSamplesSQL + `) s ON s.local_speaker_id=d.local_speaker_id JOIN persons p ON p.id=s.person_id
WHERE d.version=1
GROUP BY d.model,d.model_version,d.dimensions,p.id,p.display_name ORDER BY d.model,d.model_version,d.dimensions,lower(p.display_name),p.id`

// previewInput reads the candidate rule (the active one for every value left out) and the durations of the
// scenarios. ok is false when a value is not valid.
func previewInput(query url.Values, active eligibilityValues) (rule eligibilityValues, durations []float64, ok bool) {
	rule = active
	for name, target := range map[string]*float64{
		"comparable_min_seconds":   &rule.ComparableMinSeconds,
		"contributive_min_seconds": &rule.ContributiveMinSeconds,
		"max_clipped_ratio":        &rule.MaxClippedRatio,
	} {
		if raw := query.Get(name); raw != "" {
			value, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return rule, nil, false
			}
			*target = value
		}
	}
	if !validEligibility(rule) {
		return rule, nil, false
	}
	durations = defaultPreviewDurations
	if raw := query["durations"]; len(raw) > 0 {
		if len(raw) > maxPreviewDurations {
			return rule, nil, false
		}
		durations = make([]float64, 0, len(raw))
		for _, value := range raw {
			d, err := strconv.ParseFloat(value, 64)
			if err != nil || d < 0.5 || d > 60 {
				return rule, nil, false
			}
			durations = append(durations, d)
		}
	}
	return rule, durations, true
}

func (s *Server) getVoiceEligibilityPreview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var out eligibilityPreview
	var active eligibilityValues
	if err = tx.QueryRow(ctx, `SELECT version,comparable_min_seconds,contributive_min_seconds,max_clipped_ratio FROM voice_eligibility_rules ORDER BY version DESC LIMIT 1`).
		Scan(&out.ActiveVersion, &active.ComparableMinSeconds, &active.ContributiveMinSeconds, &active.MaxClippedRatio); err != nil {
		s.internal(w, err)
		return
	}
	rule, durations, ok := previewInput(r.URL.Query(), active)
	if !ok {
		fail(w, http.StatusBadRequest, "invalid_voice_eligibility")
		return
	}
	out.Groups, err = previewGroups(ctx, tx, rule, durations)
	if err == nil {
		err = tx.QueryRow(ctx, `SELECT count(DISTINCT person_id) FROM (`+activeSamplesSQL+`) s`).Scan(&out.PersonsWithSamples)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}

func previewGroups(ctx context.Context, tx pgx.Tx, candidate eligibilityValues, durations []float64) ([]eligibilityGroupPreview, error) {
	rules := []eligibilityValues{candidate}
	for _, d := range durations {
		rules = append(rules, eligibilityValues{ComparableMinSeconds: d, ContributiveMinSeconds: d, MaxClippedRatio: candidate.MaxClippedRatio})
	}
	comparable, contributive, clipped := make([]float64, len(rules)), make([]float64, len(rules)), make([]float64, len(rules))
	for i, v := range rules {
		comparable[i], contributive[i], clipped[i] = v.ComparableMinSeconds, v.ContributiveMinSeconds, v.MaxClippedRatio
	}
	rows, err := tx.Query(ctx, previewGroupsSQL, comparable, contributive, clipped)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := []eligibilityGroupPreview{}
	for rows.Next() {
		var key int
		var g eligibilityGroupPreview
		var sc eligibilityScenario
		c := &g.Current
		if err = rows.Scan(&key, &g.Model, &g.ModelVersion, &g.Dimensions,
			&sc.Present, &sc.Comparable, &sc.Contributive, &sc.TooShort, &sc.OverlapOnly, &sc.Clipped, &sc.PersonsWithReference,
			&sc.RecoveredComparable, &sc.RecoveredContributive, &sc.LostComparable, &sc.LostContributive,
			&c.Comparable, &c.Contributive, &c.TooShort, &c.OverlapOnly, &c.Clipped, &c.LowQuality, &c.PersonsWithReference,
			&g.MissingSources, &g.Estimated, &g.OriginalVectors, &g.OverlapShare); err != nil {
			return nil, err
		}
		c.Present = sc.Present
		sc.Rule = rules[key-1]
		last := len(groups) - 1
		if last < 0 || groups[last].Model != g.Model || groups[last].ModelVersion != g.ModelVersion || groups[last].Dimensions != g.Dimensions {
			g.Scenarios, g.Persons = []eligibilityScenario{}, []eligibilityPersonPreview{}
			groups = append(groups, g)
			last++
		}
		if key == 1 {
			groups[last].Candidate = sc
		} else {
			groups[last].Scenarios = append(groups[last].Scenarios, sc)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	persons, err := tx.Query(ctx, previewPersonsSQL, comparable[:1], contributive[:1], clipped[:1])
	if err != nil {
		return nil, err
	}
	defer persons.Close()
	for persons.Next() {
		var model, version string
		var dimensions int
		var p eligibilityPersonPreview
		if err = persons.Scan(&model, &version, &dimensions, &p.Person.ID, &p.Person.DisplayName, &p.Present, &p.Comparable, &p.Contributive,
			&p.TooShort, &p.OverlapOnly, &p.Clipped, &p.MissingSources, &p.Estimated, &p.Reference, &p.ReferenceNow); err != nil {
			return nil, err
		}
		for i := range groups {
			if groups[i].Model == model && groups[i].ModelVersion == version && groups[i].Dimensions == dimensions {
				groups[i].Persons = append(groups[i].Persons, p)
			}
		}
	}
	return groups, persons.Err()
}

// storedEligibilitySQL counts the production speaker fingerprints by their stored eligibility: the before and
// after of a requalification report.
var storedEligibilitySQL = `SELECT count(*),count(*) FILTER (WHERE f.usable),count(*) FILTER (WHERE f.contributive),
  count(*) FILTER (WHERE NOT f.contributive AND f.unusable_reason='too_short'),
  count(*) FILTER (WHERE NOT f.contributive AND f.unusable_reason='overlap_only'),
  count(*) FILTER (WHERE NOT f.contributive AND f.unusable_reason='clipped'),
  count(*) FILTER (WHERE NOT f.contributive AND f.unusable_reason='low_quality'),
  count(DISTINCT s.person_id) FILTER (WHERE f.contributive),
  count(*) FILTER (WHERE ` + clipSourceAbsentSQL("f.clip_id") + `),
  count(*) FILTER (WHERE f.net_speech_source='estimated'),count(*) FILTER (WHERE f.measures_version IS NULL)
FROM voice_fingerprints f LEFT JOIN (` + activeSamplesSQL + `) s ON s.local_speaker_id=f.local_speaker_id
WHERE NOT f.benchmark AND f.scope='speaker'`

// storedEligibility reads the stored eligibility of the production speaker fingerprints; the report keeps the
// figures of the fingerprints left without source, estimated or with their original vector.
func storedEligibility(ctx context.Context, q querier, report *eligibilityRequalification) (eligibilityOutcome, error) {
	var o eligibilityOutcome
	var missing, estimated, original int
	err := q.QueryRow(ctx, storedEligibilitySQL).Scan(&o.Present, &o.Comparable, &o.Contributive, &o.TooShort, &o.OverlapOnly, &o.Clipped, &o.LowQuality,
		&o.PersonsWithReference, &missing, &estimated, &original)
	if report != nil {
		report.MissingSources, report.Estimated, report.OriginalVectors = missing, estimated, original
	}
	return o, err
}

// traceRequalification records a requalification run in the caller's transaction.
func traceRequalification(ctx context.Context, tx pgx.Tx, report eligibilityRequalification, author authorRef) error {
	before, err := json.Marshal(report.Before)
	if err != nil {
		return err
	}
	after, err := json.Marshal(report.After)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO voice_eligibility_requalifications(rule_version,author_id,author_username,changed,missing_sources,estimated,original_vectors,before,after) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		report.RuleVersion, author.ID, author.Username, report.Changed, report.MissingSources, report.Estimated, report.OriginalVectors, before, after)
	return err
}

// loadRequalifications lists the last requalifications, the most recent first.
func loadRequalifications(ctx context.Context, q querier) ([]eligibilityRequalification, error) {
	rows, err := q.Query(ctx, `SELECT id,rule_version,author_username,created_at,changed,missing_sources,estimated,original_vectors,before,after FROM voice_eligibility_requalifications ORDER BY id DESC LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []eligibilityRequalification{}
	for rows.Next() {
		var r eligibilityRequalification
		var before, after []byte
		if err = rows.Scan(&r.ID, &r.RuleVersion, &r.Author, &r.CreatedAt, &r.Changed, &r.MissingSources, &r.Estimated, &r.OriginalVectors, &before, &after); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(before, &r.Before); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(after, &r.After); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
