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

// benchmarkStageFingerprint is the benchmark stage of the fingerprint models, the only one scored this way.
const benchmarkStageFingerprint = "fingerprint"

// scoreCentroidLegacy is the score of a fingerprint run recorded before #397; never a production mode.
const scoreCentroidLegacy scoreMode = "centroid"

type benchmarkScoring struct {
	ScoreMode        scoreMode             `json:"score_mode"`
	ScoreModeVersion *int                  `json:"score_mode_version"`
	EligibilityRule  *versionedEligibility `json:"eligibility_rule"`
	// SampleCapSeconds is the cap of one sample's weight in a cumulative Reference (referenceSampleCapSeconds).
	SampleCapSeconds *float64 `json:"reference_sample_cap_seconds"`
}

// activeBenchmarkScoring reads the score mode and the eligibility rule production uses now.
func activeBenchmarkScoring(ctx context.Context, q pgQuerier) (benchmarkScoring, error) {
	sampleCap := referenceSampleCapSeconds
	out := benchmarkScoring{SampleCapSeconds: &sampleCap}
	version, mode, err := activeScoreModeVersion(ctx, q)
	if err != nil {
		return out, err
	}
	rule, err := activeEligibilityRule(ctx, q)
	out.ScoreMode, out.ScoreModeVersion, out.EligibilityRule = mode, &version, &rule
	return out, err
}

// recordedScoring reads the scoring of a run from its parameters: nil for a stage other than the fingerprint,
// the legacy centroid for a fingerprint run recorded before #397.
func recordedScoring(stage string, params []byte) *benchmarkScoring {
	if stage != benchmarkStageFingerprint {
		return nil
	}
	var p struct {
		Scoring *benchmarkScoring `json:"scoring"`
	}
	if json.Unmarshal(params, &p) == nil && p.Scoring != nil && p.Scoring.ScoreMode.valid() && p.Scoring.EligibilityRule != nil {
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
