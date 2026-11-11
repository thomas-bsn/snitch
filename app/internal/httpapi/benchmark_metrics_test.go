package httpapi

import (
	"math"
	"testing"
)

func unit(degrees float64) []float32 {
	r := degrees * math.Pi / 180
	return []float32{float32(math.Cos(r)), float32(math.Sin(r))}
}

// voice is a benchmark fingerprint of 5 s of net speech, comparable and contributive.
func voice(person int64, v []float32) benchSample {
	return benchSample{Person: person, Vector: v, NetSpeech: 5, Comparable: true, Contributive: true}
}

func TestEvaluateFingerprintsSeparatedPersons(t *testing.T) {
	refs := []benchSample{voice(1, unit(0)), voice(1, unit(4)), voice(2, unit(90)), voice(2, unit(86))}
	tests := []benchSample{voice(1, unit(8)), voice(2, unit(80)), voice(1, unit(-5))}
	m := evaluateFingerprints(scoreCumulative, refs, tests, 2)
	if m.EER == nil || *m.EER != 0 {
		t.Fatalf("separated persons must give a null EER, got %v", m.EER)
	}
	if m.Top1Accuracy == nil || *m.Top1Accuracy != 1 || m.TruePositives != 3 || m.FalsePositives != 0 || m.FalseNegatives != 0 {
		t.Fatalf("unexpected decisions: %+v", m)
	}
	if m.GenuineTrials != 3 || m.ImpostorTrials != 3 || m.Persons != 2 || m.References != 4 || m.Tests != 3 {
		t.Fatalf("unexpected counts: %+v", m)
	}
}

func TestEvaluateFingerprintsConfusedPersons(t *testing.T) {
	refs := []benchSample{voice(1, unit(0)), voice(2, unit(10))}
	tests := []benchSample{voice(1, unit(2)), voice(2, unit(8)), voice(1, unit(9)), voice(2, unit(3))}
	m := evaluateFingerprints(scoreCumulative, refs, tests, 2)
	if m.EER == nil || *m.EER <= 0 {
		t.Fatalf("overlapping persons must give a positive EER, got %v", m.EER)
	}
	if m.Top1Accuracy == nil || *m.Top1Accuracy != 0.5 {
		t.Fatalf("two tests out of four are closer to the other person, got %v", m.Top1Accuracy)
	}
	if m.FalsePositives+m.TruePositives > m.Tests || m.FalseNegatives < m.FalsePositives {
		t.Fatalf("inconsistent decision counts: %+v", m)
	}
}

func TestEvaluateFingerprintsSkipsUnscorableAndForeignDimensions(t *testing.T) {
	refs := []benchSample{voice(1, unit(0)), voice(2, []float32{1, 0, 0})}
	tests := []benchSample{voice(1, unit(1)), voice(3, unit(40)), voice(2, []float32{1, 0, 0})}
	m := evaluateFingerprints(scoreCumulative, refs, tests, 2)
	if m.IgnoredDimension != 2 {
		t.Fatalf("the 3-dimensional vectors are ignored and counted, got %d", m.IgnoredDimension)
	}
	if m.Persons != 1 || m.GenuineTrials != 1 || m.ImpostorTrials != 0 {
		t.Fatalf("a test without reference is not scored: %+v", m)
	}
	if m.EER != nil {
		t.Fatalf("no impostor trial means no EER, got %v", *m.EER)
	}
}

func TestEvaluateFingerprintsEmpty(t *testing.T) {
	m := evaluateFingerprints(scoreCumulative, nil, nil, 2)
	if m.EER != nil || m.Top1Accuracy != nil || m.Precision != nil {
		t.Fatalf("nothing to measure must report nothing: %+v", m)
	}
}

// The benchmark scores like production (#397): in the cumulative mode the Reference of a Person is weighted by
// the net speech of its samples, capped at 10 s, and a sample only contributive still enters it.
func TestEvaluateFingerprintsScoresAgainstTheCumulativeReference(t *testing.T) {
	short := benchSample{Person: 1, Vector: unit(0), NetSpeech: 2, Contributive: true}                    // too short to be compared alone
	long := benchSample{Person: 1, Vector: unit(90), NetSpeech: 30, Comparable: true, Contributive: true} // weighs 10 s, the cap
	m := evaluateFingerprints(scoreCumulative, []benchSample{short, long}, []benchSample{voice(1, unit(70))}, 2)
	// The Reference points at atan(10/2) = 78.69°: the test at 70° scores cos(8.69°).
	want := math.Cos((70 - math.Atan2(10, 2)*180/math.Pi) * math.Pi / 180)
	if m.GenuineMean == nil || !closeTo(*m.GenuineMean, want) || m.References != 2 || m.Persons != 1 {
		t.Fatalf("the test must be scored against the weighted Reference (%.6f): %+v (mean %v)", want, m, m.GenuineMean)
	}
}

// In the best-sample mode each comparable sample is a reference on its own: the only contributive one is left out.
func TestEvaluateFingerprintsScoresAgainstTheBestComparableSample(t *testing.T) {
	short := benchSample{Person: 1, Vector: unit(60), NetSpeech: 2, Contributive: true}
	long := benchSample{Person: 1, Vector: unit(0), NetSpeech: 30, Comparable: true, Contributive: true}
	m := evaluateFingerprints(scoreBestSample, []benchSample{short, long}, []benchSample{voice(1, unit(50))}, 2)
	if want := math.Cos(50 * math.Pi / 180); m.GenuineMean == nil || !closeTo(*m.GenuineMean, want) || m.References != 1 || m.Ineligible != 1 {
		t.Fatalf("only the comparable sample is a reference (%.6f): %+v (mean %v)", want, m, m.GenuineMean)
	}
}

// The eligibility rule decides as in production: a reference that does not contribute and a test that is not
// comparable are left out and counted, never scored.
func TestEvaluateFingerprintsLeavesOutIneligibleFingerprints(t *testing.T) {
	refs := []benchSample{voice(1, unit(0)), {Person: 2, Vector: unit(90), NetSpeech: 0.4}}
	tests := []benchSample{voice(1, unit(5)), {Person: 1, Vector: unit(3), NetSpeech: 1.5, Contributive: true}}
	m := evaluateFingerprints(scoreCumulative, refs, tests, 2)
	if m.Ineligible != 2 || m.References != 1 || m.Tests != 1 || m.Persons != 1 || m.GenuineTrials != 1 {
		t.Fatalf("ineligible fingerprints are counted, not scored: %+v", m)
	}
}
