package httpapi

import (
	"context"
	"math"
	"sort"
)

// Calibration of the matching thresholds (ADR 0064, #393). The measured accuracy already splits the validated
// samples into references and tests on distinct clips; the calibration replays, for one fingerprint group and
// one score mode, the production score (voice_score.go) and decision (decideMatch) on those tests:
//
//   - the distribution of the scores of a test against the Reference of its own Person ("same Person") and
//     against the References of the others ("other Person");
//   - a sweep of the automatic threshold: for each value, the tests that would be attributed automatically,
//     the wrong ones among them and the ones left to verify;
//   - simulated unknowns: every test replayed without the Reference of its own Person, as a voice nobody has
//     validated yet. It must be refused: an automatic attribution of a simulated unknown is a false recognition;
//   - a proposal of thresholds, withheld when the figures are too small or no threshold is safe.
//
// A score is a cosine similarity between -1 and 1, never a probability: nothing here converts it into one.

// Below these figures the measure says it is too small instead of proposing thresholds.
const (
	calibrationMinTests   = 20
	calibrationMinPersons = 3
)

// Reasons a proposal is withheld.
const (
	withheldTooFew       = "too_few"
	withheldNoSafeChoice = "no_safe_threshold"
)

// calibrationTest is a test sample: the Person a human validated and the comparable vector of the voice.
type calibrationTest struct {
	Person int64
	Vector []float32
}

type scoreBin struct {
	From        float64 `json:"from"`
	To          float64 `json:"to"`
	SamePerson  int     `json:"same_person"`
	OtherPerson int     `json:"other_person"`
}

// sweepRow gives what the decision does on the tests at one automatic threshold (the suggestion threshold
// being lowered to it when higher).
type sweepRow struct {
	AutomaticThreshold  float64 `json:"automatic_threshold"`
	SuggestionThreshold float64 `json:"suggestion_threshold"`
	AmbiguityMargin     float64 `json:"ambiguity_margin"`
	Automatic           int     `json:"automatic"`
	Errors              int     `json:"errors"`
	ToVerify            int     `json:"to_verify"`
	UnknownAutomatic    int     `json:"unknown_automatic"`
}

type thresholdProposal struct {
	SuggestionThreshold float64  `json:"suggestion_threshold"`
	AutomaticThreshold  float64  `json:"automatic_threshold"`
	AmbiguityMargin     float64  `json:"ambiguity_margin"`
	Figures             sweepRow `json:"figures"`
}

type calibrationMode struct {
	Mode scoreMode `json:"mode"`
	// Tests holds the tests with a comparable vector in the group; each is also replayed as an unknown.
	Tests int `json:"tests"`
	// Persons counts the Persons with a Reference in this mode and at least one test.
	Persons           int        `json:"persons"`
	SamePersonScores  int        `json:"same_person_scores"`
	OtherPersonScores int        `json:"other_person_scores"`
	Distribution      []scoreBin `json:"distribution"`
	Sweep             []sweepRow `json:"sweep"`
	// Thresholds are the thresholds applied to the group for this mode; nil when none is applied.
	Thresholds *groupThresholds `json:"thresholds"`
	// Current gives the figures at the thresholds applied to the group for this mode; nil when none is applied.
	Current  *sweepRow          `json:"current"`
	Proposal *thresholdProposal `json:"proposal"`
	Withheld string             `json:"withheld"`
}

// calibrationScores holds, per test, its score against every Reference (best first) and the score against the
// Reference of its own Person.
type calibrationScore struct {
	truth    int64
	all      []matchScore // every Person with a Reference, best first
	own      float64
	hasOwn   bool
	scorable bool
}

func (c calibrationScore) above(min float64, keep func(int64) bool) []matchScore {
	var out []matchScore
	for _, s := range c.all {
		if s.score >= min && (keep == nil || keep(s.personID)) {
			out = append(out, s)
		}
	}
	return out
}

func percentRound(x float64) float64 { return math.Round(x*100) / 100 }

// sweepFigures decides every test, and every test replayed as an unknown, at the given thresholds.
func sweepFigures(scores []calibrationScore, base matchingSettings, suggestion, automatic, margin float64) sweepRow {
	row := sweepRow{AutomaticThreshold: automatic, SuggestionThreshold: suggestion, AmbiguityMargin: margin}
	settings := base
	settings.AutoEnabled, settings.AutomaticThreshold, settings.SuggestionThreshold, settings.AmbiguityMargin = true, automatic, suggestion, margin
	for _, s := range scores {
		level, person, _, _ := decideMatch(s.above(suggestion, nil), settings)
		if level == LevelAutomatic && person != nil {
			row.Automatic++
			if *person != s.truth {
				row.Errors++
			}
		} else {
			row.ToVerify++
		}
		truth := s.truth
		if level, _, _, _ = decideMatch(s.above(suggestion, func(p int64) bool { return p != truth }), settings); level == LevelAutomatic {
			row.UnknownAutomatic++
		}
	}
	return row
}

// calibrate measures one score mode on the tests of one group. settings are the effective thresholds of the
// group (applied, or the provisional ones), and carry the precision criterion of the automation; applied tells
// whether the group has thresholds applied by an administrator.
func calibrate(mode scoreMode, dims int, references []referenceSample, tests []calibrationTest, settings matchingSettings, applied bool) calibrationMode {
	refs := newVoiceReferences(mode, dims, references)
	out := calibrationMode{Mode: mode, Distribution: make([]scoreBin, 20)}
	for i := range out.Distribution {
		out.Distribution[i] = scoreBin{From: percentRound(float64(i) * 0.05), To: percentRound(float64(i+1) * 0.05)}
	}
	bin := func(score float64) int {
		i := int(math.Floor(score / 0.05))
		return min(max(i, 0), 19)
	}
	var scores []calibrationScore
	tested := map[int64]bool{}
	for _, test := range tests {
		s := calibrationScore{truth: test.Person, all: refs.rank(test.Vector, nil, math.Inf(-1))}
		if _, ok := refs.targetUnit(test.Vector); !ok {
			continue
		}
		out.Tests++
		for _, c := range s.all {
			if c.personID == test.Person {
				s.own, s.hasOwn = c.score, true
				out.SamePersonScores++
				out.Distribution[bin(c.score)].SamePerson++
				continue
			}
			out.OtherPersonScores++
			out.Distribution[bin(c.score)].OtherPerson++
		}
		if s.hasOwn {
			tested[test.Person] = true
		}
		scores = append(scores, s)
	}
	out.Persons = len(tested)

	for k := 1; k <= 20; k++ {
		t := percentRound(float64(k) * 0.05)
		out.Sweep = append(out.Sweep, sweepFigures(scores, settings, math.Min(settings.SuggestionThreshold, t), t, settings.AmbiguityMargin))
	}
	if applied {
		row := sweepFigures(scores, settings, settings.SuggestionThreshold, settings.AutomaticThreshold, settings.AmbiguityMargin)
		out.Current = &row
	}
	if out.Tests < calibrationMinTests || out.Persons < calibrationMinPersons {
		out.Withheld = withheldTooFew
		return out
	}
	automatic, ok := safeAutomaticThreshold(scores, settings)
	if !ok {
		out.Withheld = withheldNoSafeChoice
		return out
	}
	p := thresholdProposal{AutomaticThreshold: automatic, AmbiguityMargin: proposedMargin(scores, settings.AmbiguityMargin)}
	p.SuggestionThreshold = math.Min(proposedSuggestion(scores), automatic)
	p.Figures = sweepFigures(scores, settings, p.SuggestionThreshold, p.AutomaticThreshold, p.AmbiguityMargin)
	out.Proposal = &p
	return out
}

// safeAutomaticThreshold is the lowest automatic threshold, by steps of 0.01, at and above which the tests
// attributed automatically keep the precision criterion of the automation and no simulated unknown is
// recognised. False when no such threshold attributes at least one test.
func safeAutomaticThreshold(scores []calibrationScore, settings matchingSettings) (float64, bool) {
	best, found := 0.0, false
	for k := 100; k >= 1; k-- {
		t := float64(k) / 100
		row := sweepFigures(scores, settings, math.Min(settings.SuggestionThreshold, t), t, settings.AmbiguityMargin)
		if row.Automatic == 0 {
			continue
		}
		precise := float64(row.Automatic-row.Errors) >= settings.PrecisionThreshold*float64(row.Automatic)-1e-9
		if !precise || row.UnknownAutomatic > 0 {
			break
		}
		best, found = t, true
	}
	return best, found
}

// proposedSuggestion keeps 95% of the same-Person scores at or above the suggestion threshold: the score of the
// 5th percentile, rounded down to 0.01.
func proposedSuggestion(scores []calibrationScore) float64 {
	var own []float64
	for _, s := range scores {
		if s.hasOwn {
			own = append(own, s.own)
		}
	}
	sort.Float64s(own)
	v := own[int(math.Floor(0.05*float64(len(own)-1)))]
	return math.Max(0, math.Floor(v*100)/100)
}

// proposedMargin covers 95% of the measured confusions: tests whose own Person is outscored by another one, by
// the gap between the two, rounded up to 0.01 and kept within [0.01, 0.2]. Without any confusion the current
// margin stays.
func proposedMargin(scores []calibrationScore, current float64) float64 {
	var gaps []float64
	for _, s := range scores {
		if !s.hasOwn || len(s.all) == 0 || s.all[0].personID == s.truth {
			continue
		}
		gaps = append(gaps, s.all[0].score-s.own)
	}
	if len(gaps) == 0 {
		return current
	}
	sort.Float64s(gaps)
	v := gaps[int(math.Ceil(0.95*float64(len(gaps)-1)))]
	return math.Min(0.2, math.Max(0.01, math.Ceil(v*100-1e-9)/100))
}

// voiceCalibration is the calibration part of the measured accuracy: the active fingerprint group, the
// thresholds applied to it for the active score mode and the history of its thresholds, and the measure in both
// score modes side by side, each with the thresholds applied for it.
type voiceCalibration struct {
	Group      *fingerprintGroupView `json:"group"`
	Thresholds *groupThresholds      `json:"thresholds"`
	History    []groupThresholds     `json:"history"`
	ActiveMode scoreMode             `json:"active_mode"`
	MinTests   int                   `json:"min_tests"`
	MinPersons int                   `json:"min_persons"`
	Modes      []calibrationMode     `json:"modes"`
}

// calibration measures the active fingerprint group on the split of the measured accuracy: the reference
// samples build the References, the test samples are scored through their comparable fingerprint in that group.
// The rule of the split holds: no test shares a clip with a reference.
func (s *Server) calibration(ctx context.Context, settings matchingSettings, split accuracySplit) (voiceCalibration, error) {
	out := voiceCalibration{ActiveMode: settings.ScoreMode, MinTests: calibrationMinTests, MinPersons: calibrationMinPersons, History: []groupThresholds{}, Modes: []calibrationMode{}}
	g, ok, err := activeFingerprintGroup(ctx, s.Pool)
	if err != nil || !ok {
		return out, err
	}
	out.Group = g.view()
	if out.History, err = groupThresholdHistory(ctx, s.Pool, g); err != nil {
		return out, err
	}
	references, tests, err := s.calibrationSamples(ctx, g, split)
	if err != nil {
		return out, err
	}
	for _, mode := range []scoreMode{scoreCumulative, scoreBestSample} {
		m, applied, err := s.calibrateMode(ctx, settings, g, mode, references, tests)
		if err != nil {
			return out, err
		}
		if mode == settings.ScoreMode {
			out.Thresholds = applied
		}
		out.Modes = append(out.Modes, m)
	}
	return out, nil
}

// calibrateMode measures one score mode of a group with the thresholds applied to it for that mode (or the
// provisional ones).
func (s *Server) calibrateMode(ctx context.Context, settings matchingSettings, g matchGroup, mode scoreMode, references []referenceSample, tests []calibrationTest) (calibrationMode, *groupThresholds, error) {
	applied, err := appliedThresholds(ctx, s.Pool, g, mode)
	if err != nil {
		return calibrationMode{}, nil, err
	}
	m := calibrate(mode, g.dimensions, references, tests, settings.forGroup(applied), applied != nil)
	m.Thresholds = applied
	return m, applied, nil
}

// currentProposal measures one group in one score mode as the calibration of the measured accuracy does, and
// returns its proposal; nil when it is withheld. Applying thresholds as a proposal is checked against it, so a
// typed value is never recorded as coming from a measure.
func (s *Server) currentProposal(ctx context.Context, g matchGroup, mode scoreMode) (*thresholdProposal, error) {
	settings, err := loadMatchingSettings(ctx, s.Pool)
	if err != nil {
		return nil, err
	}
	samples, err := s.accuracySamples(ctx)
	if err != nil {
		return nil, err
	}
	var split accuracySplit
	split.references, split.tests, _ = splitSamples(samples)
	references, tests, err := s.calibrationSamples(ctx, g, split)
	if err != nil {
		return nil, err
	}
	m, _, err := s.calibrateMode(ctx, settings, g, mode, references, tests)
	return m.Proposal, err
}

// sameThresholds tells whether applied values are those of a proposal (both come from the same JSON numbers).
func (p *thresholdProposal) sameThresholds(suggestion, automatic, margin float64) bool {
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	return p != nil && near(p.SuggestionThreshold, suggestion) && near(p.AutomaticThreshold, automatic) && near(p.AmbiguityMargin, margin)
}

// calibrationSamples reads, in one group, the reference samples of the split and the comparable vector of each
// of its tests.
func (s *Server) calibrationSamples(ctx context.Context, g matchGroup, split accuracySplit) ([]referenceSample, []calibrationTest, error) {
	refIDs := make([]int64, len(split.references))
	for i, sample := range split.references {
		refIDs[i] = sample.id
	}
	testIDs := make([]int64, len(split.tests))
	truth := map[int64]int64{}
	for i, sample := range split.tests {
		testIDs[i] = sample.id
		truth[sample.id] = sample.personID
	}
	references, err := loadReferenceSamples(ctx, s.Pool, g, refIDs)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT ON (vs.id) vs.id,f.embedding::real[]
  FROM voice_samples vs JOIN voice_fingerprints f ON f.clip_id=vs.clip_id AND f.local_speaker_id=vs.local_speaker_id
  WHERE vs.id=ANY($1) AND f.usable AND NOT f.benchmark AND f.scope='speaker' AND f.model=$2 AND f.model_version=$3 AND f.dimensions=$4
  ORDER BY vs.id,f.id`, testIDs, g.model, g.version, g.dimensions)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var tests []calibrationTest
	for rows.Next() {
		var id int64
		var vector []float32
		if err = rows.Scan(&id, &vector); err != nil {
			return nil, nil, err
		}
		tests = append(tests, calibrationTest{Person: truth[id], Vector: vector})
	}
	return references, tests, rows.Err()
}
