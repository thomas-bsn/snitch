package httpapi

import (
	"context"
	"encoding/json"
)

// How a fingerprint benchmark scores (#397, ADR 0064): the score mode and the eligibility rule active when the
// run started, recorded in its parameters (benchmark_runs.params.scoring) and shown with its results, so a
// threshold read in a run is known to hold for production under that mode and rule.
//
// A run recorded before #397 has no such parameters: it was scored against the unweighted centroid of the
// reference vectors, from fingerprints its pinned pipeline decided. Its results stay as they were stored and
// read as the legacy "centroid" mode, without a rule.

const scoreCentroidLegacy = "centroid"

type benchmarkEligibilityRule struct {
	Version                int     `json:"version"`
	ComparableMinSeconds   float64 `json:"comparable_min_seconds"`
	ContributiveMinSeconds float64 `json:"contributive_min_seconds"`
	MaxClippedRatio        float64 `json:"max_clipped_ratio"`
}

type benchmarkScoring struct {
	ScoreMode        string                    `json:"score_mode"`
	ScoreModeVersion *int                      `json:"score_mode_version"`
	EligibilityRule  *benchmarkEligibilityRule `json:"eligibility_rule"`
	// SampleCapSeconds is the cap of one sample's weight in a cumulative Reference (referenceSampleCapSeconds).
	SampleCapSeconds *float64 `json:"reference_sample_cap_seconds"`
}

// activeBenchmarkScoring reads the score mode and the eligibility rule production uses now.
func activeBenchmarkScoring(ctx context.Context, q pgQuerier) (benchmarkScoring, error) {
	var version int
	rule := benchmarkEligibilityRule{}
	sampleCap := referenceSampleCapSeconds
	out := benchmarkScoring{ScoreModeVersion: &version, EligibilityRule: &rule, SampleCapSeconds: &sampleCap}
	err := q.QueryRow(ctx, `SELECT m.version,m.mode,r.version,r.comparable_min_seconds,r.contributive_min_seconds,r.max_clipped_ratio
  FROM (SELECT version,mode FROM voice_score_modes ORDER BY version DESC LIMIT 1) m,
       (SELECT version,comparable_min_seconds,contributive_min_seconds,max_clipped_ratio FROM voice_eligibility_rules ORDER BY version DESC LIMIT 1) r`).
		Scan(&version, &out.ScoreMode, &rule.Version, &rule.ComparableMinSeconds, &rule.ContributiveMinSeconds, &rule.MaxClippedRatio)
	return out, err
}

// recordedScoring reads the scoring of a run from its parameters: nil for a stage other than the fingerprint,
// the legacy centroid for a fingerprint run recorded before #397.
func recordedScoring(stage string, params []byte) *benchmarkScoring {
	if stage != "fingerprint" {
		return nil
	}
	var p struct {
		Scoring *benchmarkScoring `json:"scoring"`
	}
	if json.Unmarshal(params, &p) == nil && p.Scoring != nil && scoreMode(p.Scoring.ScoreMode).valid() && p.Scoring.EligibilityRule != nil {
		return p.Scoring
	}
	return &benchmarkScoring{ScoreMode: scoreCentroidLegacy}
}

// withScoring records the active scoring in the parameters of a fingerprint run.
func withScoring(ctx context.Context, q pgQuerier, params []byte) ([]byte, error) {
	scoring, err := activeBenchmarkScoring(ctx, q)
	if err != nil {
		return nil, err
	}
	var p map[string]any
	if err = json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	p["scoring"] = scoring
	return json.Marshal(p)
}
