package httpapi

import (
	"math"
	"testing"
)

func unit(degrees float64) []float32 {
	r := degrees * math.Pi / 180
	return []float32{float32(math.Cos(r)), float32(math.Sin(r))}
}

func TestEvaluateFingerprintsSeparatedPersons(t *testing.T) {
	refs := []benchSample{{1, unit(0)}, {1, unit(4)}, {2, unit(90)}, {2, unit(86)}}
	tests := []benchSample{{1, unit(8)}, {2, unit(80)}, {1, unit(-5)}}
	m := evaluateFingerprints(refs, tests, 2)
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
	refs := []benchSample{{1, unit(0)}, {2, unit(10)}}
	tests := []benchSample{{1, unit(2)}, {2, unit(8)}, {1, unit(9)}, {2, unit(3)}}
	m := evaluateFingerprints(refs, tests, 2)
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
	refs := []benchSample{{1, unit(0)}, {2, []float32{1, 0, 0}}}
	tests := []benchSample{{1, unit(1)}, {3, unit(40)}, {2, []float32{1, 0, 0}}}
	m := evaluateFingerprints(refs, tests, 2)
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
	m := evaluateFingerprints(nil, nil, 2)
	if m.EER != nil || m.Top1Accuracy != nil || m.Precision != nil {
		t.Fatalf("nothing to measure must report nothing: %+v", m)
	}
}
