package httpapi

import "context"

// The Reference of a Person as matching uses it (#396), for the directory and the record: read in the active
// fingerprint group under the active score mode, with its level judged against every Person of that group. In the
// cumulative mode the Reference uses the contributive samples; in the best-sample mode only the comparable ones
// (voice_score.go), so the level always reflects the samples matching really compares with.

// referenceSamplesFrom selects the samples a cumulative Reference uses in the group ($1 model, $2 version,
// $3 dimensions): active validated samples of a Person whose voice profile still exists, through the
// contributive production speaker fingerprint of their local speaker. Matching (loadReferenceSamples) and the
// figures of the directory read the same rows.
const referenceSamplesFrom = `FROM voice_samples vs
  JOIN voice_fingerprints f ON f.clip_id=vs.clip_id AND f.local_speaker_id=vs.local_speaker_id
  WHERE vs.retired_at IS NULL AND NOT EXISTS (SELECT 1 FROM persons p WHERE p.id=vs.person_id AND p.voice_profile_deleted_at IS NOT NULL)
    AND f.contributive AND NOT f.benchmark AND f.scope='speaker' AND f.model=$1 AND f.model_version=$2 AND f.dimensions=$3`

// referenceUseContributive is the use of a sample that enters the Reference; any other use is the reason it is
// set aside: the unusable reason of its fingerprint, or referenceNoFingerprint.
const (
	referenceUseContributive = "contributive"
	referenceNoFingerprint   = "no_fingerprint"
)

// referenceDiscardOrder lists the reasons a sample is set aside, in the order the record shows them.
var referenceDiscardOrder = []string{"too_short", "overlap_only", "clipped", "low_quality", referenceNoFingerprint}

type personReferenceDiscard struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// personReference is what the Reference of a Person holds in the active group and the level it gets.
type personReference struct {
	Level               referenceLevel           `json:"level"`
	Basis               referenceBasis           `json:"basis"`
	ContributiveSamples int                      `json:"contributive_samples"`
	Clips               int                      `json:"clips"`
	NetSpeechSeconds    float64                  `json:"net_speech_seconds"`
	Discarded           []personReferenceDiscard `json:"discarded"`
}

// groupReferenceFigures reads the figures of the Reference of every Person of the group under a score mode: the
// median of the level is taken on all of them, whoever reads and whatever page is read.
func groupReferenceFigures(ctx context.Context, q pgQuerier, g matchGroup, mode scoreMode) (map[int64]referenceFigure, error) {
	rows, err := q.Query(ctx, `SELECT vs.person_id,vs.clip_id,COALESCE(f.net_speech_seconds,f.duration_seconds),f.usable `+referenceSamplesFrom, g.model, g.version, g.dimensions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var samples []referenceSample
	for rows.Next() {
		var sample referenceSample
		if err = rows.Scan(&sample.Person, &sample.Clip, &sample.NetSpeech, &sample.Comparable); err != nil {
			return nil, err
		}
		if mode == scoreBestSample && !sample.Comparable {
			continue // only compared alone, which a sample that is not comparable never is
		}
		samples = append(samples, sample)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return referenceFigures(samples), nil
}

type sampleReferenceUse struct {
	Person, LocalSpeaker int64
	Use                  string
}

// sampleReferenceUses tells, for each active sample of the given Persons, what the Reference of the group makes
// of it under a score mode: used (referenceUseContributive), or the reason it is set aside. In the cumulative
// mode the used ones are exactly the rows of referenceSamplesFrom for these Persons; in the best-sample mode only
// the comparable ones among them, a contributive sample that is not comparable being too short to be compared
// alone.
func sampleReferenceUses(ctx context.Context, q pgQuerier, g matchGroup, mode scoreMode, persons []int64) ([]sampleReferenceUse, error) {
	rows, err := q.Query(ctx, `SELECT vs.person_id,vs.local_speaker_id,
  COALESCE((SELECT CASE WHEN u.used THEN 'contributive' ELSE u.unusable_reason END FROM voice_fingerprints f
    CROSS JOIN LATERAL (SELECT CASE WHEN $5='best_sample' THEN f.usable ELSE f.contributive END AS used, f.unusable_reason) u
    WHERE f.clip_id=vs.clip_id AND f.local_speaker_id=vs.local_speaker_id AND NOT f.benchmark AND f.scope='speaker'
      AND f.model=$2 AND f.model_version=$3 AND f.dimensions=$4
    ORDER BY u.used DESC,f.id DESC LIMIT 1),'no_fingerprint')
  FROM voice_samples vs
  WHERE vs.person_id=ANY($1) AND vs.retired_at IS NULL
    AND NOT EXISTS (SELECT 1 FROM persons p WHERE p.id=vs.person_id AND p.voice_profile_deleted_at IS NOT NULL)
  ORDER BY vs.person_id,vs.id`, persons, g.model, g.version, g.dimensions, mode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sampleReferenceUse
	for rows.Next() {
		var u sampleReferenceUse
		if err = rows.Scan(&u.Person, &u.LocalSpeaker, &u.Use); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// loadPersonReferences fills the Reference of the given Persons. Without an active group (no production
// fingerprint yet), no Person has a Reference and every sample is set aside for want of a fingerprint.
func loadPersonReferences(ctx context.Context, q pgQuerier, persons []personView) error {
	if len(persons) == 0 {
		return nil
	}
	g, ok, err := activeFingerprintGroup(ctx, q)
	if err != nil {
		return err
	}
	mode, err := activeScoreMode(ctx, q)
	if err != nil {
		return err
	}
	figures := map[int64]referenceFigure{}
	if ok {
		if figures, err = groupReferenceFigures(ctx, q, g, mode); err != nil {
			return err
		}
	}
	standing := newReferenceStanding(figures)
	ids := make([]int64, len(persons))
	for i := range persons {
		ids[i] = persons[i].ID
	}
	uses, err := sampleReferenceUses(ctx, q, g, mode, ids)
	if err != nil {
		return err
	}
	set := map[int64]map[string]int{}
	for _, u := range uses {
		if u.Use == referenceUseContributive {
			continue
		}
		if set[u.Person] == nil {
			set[u.Person] = map[string]int{}
		}
		set[u.Person][u.Use]++
	}
	for i := range persons {
		f := figures[persons[i].ID]
		level, basis := standing.judge(f)
		r := &personReference{Level: level, Basis: basis, ContributiveSamples: f.Samples, Clips: f.Clips, NetSpeechSeconds: f.NetSpeechSeconds, Discarded: []personReferenceDiscard{}}
		for _, reason := range referenceDiscardOrder {
			if n := set[persons[i].ID][reason]; n > 0 {
				r.Discarded = append(r.Discarded, personReferenceDiscard{Reason: reason, Count: n})
			}
		}
		persons[i].Reference = r
	}
	return nil
}
