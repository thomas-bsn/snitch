package httpapi

import (
	"math"
	"testing"
)

// The score of a voice (ADR 0064) is pure arithmetic on vectors: these tests need no database.

func closeTo(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestCumulativeReferenceRecognisesWhatNoSampleDoesAlone(t *testing.T) {
	// Three short extracts of one voice, each pushed off its true direction (1,0,0) by a different noise: alone,
	// each is at cosine 0.64 from the true voice, below any suggestion threshold.
	samples := []referenceSample{
		{Person: 7, Vector: []float32{1, 1.2, 0}, NetSpeech: 2},
		{Person: 7, Vector: []float32{1, -0.6, 1.0392305}, NetSpeech: 2},
		{Person: 7, Vector: []float32{1, -0.6, -1.0392305}, NetSpeech: 2},
	}
	fourth := []float32{1, 0.1, 0}
	refs := newVoiceReferences(scoreCumulative, 3, samples)
	score, ok := refs.score(fourth, 7)
	if !ok || score < 0.99 {
		t.Fatalf("the cumulative Reference must recognise the fourth extract, got %v (%v)", score, ok)
	}
	for _, sample := range samples {
		alone := newVoiceReferences(scoreCumulative, 3, []referenceSample{sample})
		if s, _ := alone.score(fourth, 7); s > 0.75 {
			t.Fatalf("a single extract must not suffice on its own, got %v", s)
		}
	}
}

func TestBestSampleModeComparesComparableSamplesOneByOne(t *testing.T) {
	samples := []referenceSample{
		{Person: 1, Vector: []float32{1, 0}, NetSpeech: 3, Comparable: true},
		{Person: 1, Vector: []float32{0, 1}, NetSpeech: 3, Comparable: true},
		{Person: 2, Vector: []float32{1, 1}, NetSpeech: 1.5}, // contributive only: not a reference in this mode
	}
	refs := newVoiceReferences(scoreBestSample, 2, samples)
	if s, ok := refs.score([]float32{1, 0.1}, 1); !ok || s < 0.99 {
		t.Fatalf("the best sample must give the score, got %v (%v)", s, ok)
	}
	if _, ok := refs.score([]float32{1, 1}, 2); ok {
		t.Fatal("a sample that is not comparable on its own is no reference in the best-sample mode")
	}
	if got := refs.persons(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("only Person 1 has a reference, got %v", got)
	}
	cumulative := newVoiceReferences(scoreCumulative, 2, samples)
	if s, ok := cumulative.score([]float32{1, 1}, 1); !ok || !closeTo(s, 1) {
		t.Fatalf("the cumulative Reference of Person 1 lies between its two samples, got %v", s)
	}
	if _, ok := cumulative.score([]float32{1, 1}, 2); !ok {
		t.Fatal("a contributive sample enters the cumulative Reference")
	}
}

func TestReferenceWeightsSamplesByCappedNetSpeech(t *testing.T) {
	// 2 s at 0 degrees and 6 s at 90 degrees: the Reference points at atan(6/2).
	refs := newVoiceReferences(scoreCumulative, 2, []referenceSample{
		{Person: 1, Vector: []float32{1, 0}, NetSpeech: 2},
		{Person: 1, Vector: []float32{0, 5}, NetSpeech: 6}, // the vector's norm never weighs
	})
	if s, _ := refs.score([]float32{2, 6}, 1); !closeTo(s, 1) {
		t.Fatalf("weights must follow net speech, got %v", s)
	}
	// 30 s weighs as much as the cap (10 s): the Reference points at atan(10/2), not atan(30/2).
	capped := newVoiceReferences(scoreCumulative, 2, []referenceSample{
		{Person: 1, Vector: []float32{1, 0}, NetSpeech: 2},
		{Person: 1, Vector: []float32{0, 1}, NetSpeech: 30},
	})
	if s, _ := capped.score([]float32{2, 10}, 1); !closeTo(s, 1) {
		t.Fatalf("one long sample must not dominate beyond the cap, got %v", s)
	}
	if referenceWeight(30) != referenceSampleCapSeconds || referenceWeight(2) != 2 || referenceWeight(-1) != 0 {
		t.Fatal("referenceWeight must clamp net speech between 0 and the cap")
	}
}

func TestReferencesIgnoreUnusableVectors(t *testing.T) {
	refs := newVoiceReferences(scoreCumulative, 2, []referenceSample{
		{Person: 1, Vector: []float32{1, 0, 0}, NetSpeech: 2}, // other dimensions: never mixed
		{Person: 2, Vector: []float32{0, 0}, NetSpeech: 2},    // null vector
		{Person: 3, Vector: []float32{1, 0}, NetSpeech: 0},    // no net speech: weighs nothing
		{Person: 4, Vector: []float32{1, 0}, NetSpeech: 2},
	})
	if got := refs.persons(); len(got) != 1 || got[0] != 4 {
		t.Fatalf("only Person 4 has a usable sample, got %v", got)
	}
	if _, ok := refs.score([]float32{1, 0, 0}, 4); ok {
		t.Fatal("a vector of other dimensions must never be scored")
	}
	if s, ok := refs.score([]float32{0, 0}, 4); ok || s != 0 {
		t.Fatal("a null target vector has no score")
	}
}

func TestRankKeepsScoresAboveTheMinimumBestFirst(t *testing.T) {
	refs := newVoiceReferences(scoreCumulative, 2, []referenceSample{
		{Person: 3, Vector: unit(10), NetSpeech: 2},
		{Person: 1, Vector: unit(10), NetSpeech: 2},
		{Person: 2, Vector: unit(0), NetSpeech: 2},
		{Person: 4, Vector: unit(80), NetSpeech: 2},
		{Person: 5, Vector: unit(1), NetSpeech: 2},
	})
	got := refs.rank(unit(10), func(person int64) bool { return person != 5 }, 0.75)
	want := []int64{1, 3, 2}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %+v", want, got)
	}
	for i, person := range want {
		if got[i].personID != person {
			t.Fatalf("expected %v (ties by Person id), got %+v", want, got)
		}
	}
	if !closeTo(got[0].score, 1) || !closeTo(got[2].score, math.Cos(10*math.Pi/180)) {
		t.Fatalf("scores must be the cosine with the Reference: %+v", got)
	}
}

func TestReferenceFiguresCountWhatTheReferenceUses(t *testing.T) {
	figures := referenceFigures([]referenceSample{
		{Person: 1, Clip: 10, Vector: unit(0), NetSpeech: 2},
		{Person: 1, Clip: 10, Vector: unit(5), NetSpeech: 30},
		{Person: 1, Clip: 11, Vector: unit(5), NetSpeech: 1.5},
		{Person: 2, Clip: 12, Vector: unit(5), NetSpeech: 4},
	})
	one := figures[1]
	if one.Samples != 3 || one.Clips != 2 || !closeTo(one.NetSpeechSeconds, 2+referenceSampleCapSeconds+1.5) {
		t.Fatalf("unexpected figures for Person 1: %+v", one)
	}
	if two := figures[2]; two.Samples != 1 || two.Clips != 1 || !closeTo(two.NetSpeechSeconds, 4) {
		t.Fatalf("unexpected figures for Person 2: %+v", two)
	}
}

// BenchmarkRankLibrary measures the score of one voice against the References of 200 Persons of 20 samples each,
// in 256 dimensions (the order of the target library): a full rematch of 30,000 voices costs 30,000 times this.
func BenchmarkRankLibrary(b *testing.B) {
	for _, mode := range []scoreMode{scoreCumulative, scoreBestSample} {
		b.Run(string(mode), func(b *testing.B) {
			const persons, perPerson, dims = 200, 20, 256
			samples := make([]referenceSample, 0, persons*perPerson)
			for p := 0; p < persons; p++ {
				for k := 0; k < perPerson; k++ {
					v := make([]float32, dims)
					for i := range v {
						v[i] = float32(math.Sin(float64(p*dims+i) + float64(k)*0.01))
					}
					samples = append(samples, referenceSample{Person: int64(p), Vector: v, NetSpeech: 2, Comparable: true})
				}
			}
			refs := newVoiceReferences(mode, dims, samples)
			target := samples[17].Vector
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				refs.rank(target, nil, 0.75)
			}
		})
	}
}
