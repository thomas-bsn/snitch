package httpapi

import (
	"math"
	"testing"
)

// The calibration of the thresholds (#393) replays the production score and decision on reference and test
// samples: pure arithmetic on vectors, no database.

func planeAt(degrees float64) []float32 {
	r := degrees * math.Pi / 180
	return []float32{float32(math.Cos(r)), float32(math.Sin(r)), 0, 0}
}

func calibrationSettings() matchingSettings {
	return matchingSettings{AutomaticThreshold: 0.90, SuggestionThreshold: 0.75, AmbiguityMargin: 0.05, MaxCandidates: 5, PrecisionThreshold: 0.95}
}

// fourVoices gives four Persons far apart (0, 90, 180 and 270 degrees), one comparable reference each, and
// testsPerPerson tests per Person, 1 to testsPerPerson degrees away from her reference.
func fourVoices(testsPerPerson int) ([]referenceSample, []calibrationTest) {
	var refs []referenceSample
	var tests []calibrationTest
	for p := int64(1); p <= 4; p++ {
		base := float64(p-1) * 90
		refs = append(refs, referenceSample{Person: p, Clip: p, Vector: planeAt(base), NetSpeech: 5, Comparable: true})
		for i := 1; i <= testsPerPerson; i++ {
			tests = append(tests, calibrationTest{Person: p, Vector: planeAt(base + float64(i))})
		}
	}
	return refs, tests
}

func sweepAt(t *testing.T, c calibrationMode, threshold float64) sweepRow {
	t.Helper()
	for _, row := range c.Sweep {
		if math.Abs(row.AutomaticThreshold-threshold) < 1e-9 {
			return row
		}
	}
	t.Fatalf("no sweep row at %v: %+v", threshold, c.Sweep)
	return sweepRow{}
}

func TestCalibrationSeparatesSameAndOtherPersonScores(t *testing.T) {
	refs, tests := fourVoices(6)
	c := calibrate(scoreCumulative, 4, refs, tests, calibrationSettings(), true)
	if c.Mode != scoreCumulative || c.Tests != 24 || c.Persons != 4 || c.SamePersonScores != 24 || c.OtherPersonScores != 72 {
		t.Fatalf("every test is scored against every Reference: %+v", c)
	}
	if len(c.Distribution) != 20 || c.Distribution[0].From != 0 || math.Abs(c.Distribution[19].To-1) > 1e-9 {
		t.Fatalf("twenty score classes of 0.05 between 0 and 1: %+v", c.Distribution)
	}
	// Same Person: within 6 degrees, cosine above 0.99. Other Persons: 84 to 96 degrees (cosine 0.10 to -0.10,
	// a negative score counted in the first class) and 174 to 186 degrees (negative).
	same, other := 0, 0
	for i, bin := range c.Distribution {
		same += bin.SamePerson
		other += bin.OtherPerson
		if bin.SamePerson > 0 && i != 19 {
			t.Fatalf("same-Person scores belong to the top class: %+v", c.Distribution)
		}
		if bin.OtherPerson > 0 && i > 2 {
			t.Fatalf("other-Person scores belong to the bottom classes: %+v", c.Distribution)
		}
	}
	if same != 24 || other != 72 {
		t.Fatalf("every score is counted once: %d %d", same, other)
	}

	// Sweep: at 0.95 every test is automatic and right; at 1.0 none is, all are left to verify.
	if row := sweepAt(t, c, 0.95); row.Automatic != 24 || row.Errors != 0 || row.ToVerify != 0 || row.UnknownAutomatic != 0 {
		t.Fatalf("sweep at 0.95: %+v", row)
	}
	if row := sweepAt(t, c, 1.0); row.Automatic != 0 || row.ToVerify != 24 {
		t.Fatalf("sweep at 1.0: %+v", row)
	}
	// At 0.05, a simulated unknown (a test replayed without its own Reference) is recognised as its neighbour.
	if row := sweepAt(t, c, 0.05); row.UnknownAutomatic == 0 {
		t.Fatalf("a very low threshold recognises simulated unknowns: %+v", row)
	}
	if c.Current == nil || c.Current.AutomaticThreshold != 0.90 || c.Current.Automatic != 24 || c.Current.UnknownAutomatic != 0 {
		t.Fatalf("figures at the applied thresholds: %+v", c.Current)
	}

	// Proposal: the lowest automatic threshold at and above which the measure keeps the precision criterion and
	// recognises no simulated unknown. The best simulated unknown scores cos(84) = 0.1045: 0.11.
	p := c.Proposal
	if c.Withheld != "" || p == nil || math.Abs(p.AutomaticThreshold-0.11) > 1e-9 || p.Figures.UnknownAutomatic != 0 || p.Figures.Errors != 0 || p.Figures.Automatic != 24 {
		t.Fatalf("unexpected proposal: %q %+v", c.Withheld, p)
	}
	// The suggestion threshold never exceeds the automatic one; no confusion was measured, so the margin stays.
	if p.SuggestionThreshold > p.AutomaticThreshold || p.AmbiguityMargin != 0.05 {
		t.Fatalf("unexpected proposal: %+v", p)
	}
}

func TestCalibrationWithoutAppliedThresholdsHasNoCurrentFigures(t *testing.T) {
	refs, tests := fourVoices(6)
	if c := calibrate(scoreCumulative, 4, refs, tests, calibrationSettings(), false); c.Current != nil || c.Proposal == nil {
		t.Fatalf("a group without applied thresholds is measured and proposed, never decided: %+v", c)
	}
}

func TestCalibrationWithholdsTheProposalOnTooFewTests(t *testing.T) {
	refs, tests := fourVoices(2) // 8 tests
	c := calibrate(scoreCumulative, 4, refs, tests, calibrationSettings(), true)
	if c.Proposal != nil || c.Withheld != "too_few" || len(c.Sweep) == 0 || c.SamePersonScores != 8 {
		t.Fatalf("too few tests: the figures are shown, the proposal withheld: %+v", c)
	}
	// Two Persons only: too few to simulate unknowns, whatever the number of tests.
	two := refs[:2]
	var twoTests []calibrationTest
	for i := 0; i < 30; i++ {
		twoTests = append(twoTests, calibrationTest{Person: int64(1 + i%2), Vector: planeAt(float64(i%2)*90 + 1)})
	}
	if c = calibrate(scoreCumulative, 4, two, twoTests, calibrationSettings(), true); c.Proposal != nil || c.Withheld != "too_few" {
		t.Fatalf("too few Persons: no proposal: %+v", c)
	}
}

func TestCalibrationWithholdsTheProposalWhenNoThresholdIsSafe(t *testing.T) {
	refs, tests := fourVoices(6)
	// Every test of Person 1 sounds exactly like Person 2: whatever the threshold, the measure errs.
	for i := range tests {
		if tests[i].Person == 1 {
			tests[i].Vector = planeAt(90)
		}
	}
	c := calibrate(scoreCumulative, 4, refs, tests, calibrationSettings(), true)
	if c.Proposal != nil || c.Withheld != "no_safe_threshold" {
		t.Fatalf("no threshold keeps the precision criterion: %+v", c)
	}
}

func TestCalibrationProposesAMarginThatCoversMeasuredConfusions(t *testing.T) {
	refs, tests := fourVoices(6)
	// One test of Person 1 at 47 degrees: 0.03 closer to Person 2 (43 degrees away) than to Person 1.
	tests[0].Vector = planeAt(47)
	c := calibrate(scoreCumulative, 4, refs, tests, calibrationSettings(), true)
	gap := math.Cos(43*math.Pi/180) - math.Cos(47*math.Pi/180)
	if c.Proposal == nil || c.Proposal.AmbiguityMargin < gap || c.Proposal.AmbiguityMargin > gap+0.01 {
		t.Fatalf("the margin must cover the measured confusion (%.4f): %+v", gap, c.Proposal)
	}
}

func TestCalibrationScoresEachModeWithItsOwnReferences(t *testing.T) {
	refs, tests := fourVoices(6)
	refs[0].Comparable = false // Person 1's only sample is contributive: no Reference in the best-sample mode
	best := calibrate(scoreBestSample, 4, refs, tests, calibrationSettings(), true)
	cumulative := calibrate(scoreCumulative, 4, refs, tests, calibrationSettings(), true)
	if best.Mode != scoreBestSample || best.Persons != 3 || best.SamePersonScores != 18 || best.OtherPersonScores != 18*2+6*3 {
		t.Fatalf("best sample: Person 1 has no Reference: %+v", best)
	}
	if cumulative.Persons != 4 || cumulative.SamePersonScores != 24 {
		t.Fatalf("cumulative: every Person has a Reference: %+v", cumulative)
	}
	// Person 1's tests have no Reference to be recognised by: never automatic, left to verify.
	if row := sweepAt(t, best, 0.95); row.Automatic != 18 || row.ToVerify != 6 || row.Errors != 0 {
		t.Fatalf("best-sample sweep: %+v", row)
	}
}
