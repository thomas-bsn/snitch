package httpapi

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// Traced requalification of the stored fingerprints (#394, spec voix-breves section 6, step 2). The
// requalification is the rule's save (putVoiceEligibility); each one is traced with its author, its rule and a
// before/after report of the stored eligibility.

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
