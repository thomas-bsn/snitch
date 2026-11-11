package httpapi

import "testing"

func TestReferenceStanding(t *testing.T) {
	judge := func(figures map[int64]referenceFigure, person int64) (referenceLevel, referenceBasis) {
		return newReferenceStanding(figures).judge(figures[person])
	}
	expect := func(t *testing.T, gotLevel referenceLevel, gotBasis referenceBasis, level referenceLevel, basis referenceBasis) {
		t.Helper()
		if gotLevel != level || gotBasis != basis {
			t.Fatalf("got %s (%s), want %s (%s)", gotLevel, gotBasis, level, basis)
		}
	}
	// Four Persons with a Reference: net speech 4, 12, 20 and 30 s, median 16 s.
	library := map[int64]referenceFigure{
		1: {Samples: 2, Clips: 2, NetSpeechSeconds: 4},
		2: {Samples: 3, Clips: 3, NetSpeechSeconds: 12},
		3: {Samples: 2, Clips: 2, NetSpeechSeconds: 20},
		4: {Samples: 1, Clips: 1, NetSpeechSeconds: 30},
	}

	t.Run("below the median is weak", func(t *testing.T) {
		l, b := judge(library, 2)
		expect(t, l, b, referenceWeak, referenceBelowMedian)
	})
	t.Run("from the median is good", func(t *testing.T) {
		l, b := judge(library, 3)
		expect(t, l, b, referenceGood, referenceMedianOrMore)
		odd := map[int64]referenceFigure{
			1: {Samples: 2, Clips: 2, NetSpeechSeconds: 5},
			2: {Samples: 2, Clips: 2, NetSpeechSeconds: 9},
			3: {Samples: 2, Clips: 2, NetSpeechSeconds: 14},
		}
		l, b = judge(odd, 2)
		expect(t, l, b, referenceGood, referenceMedianOrMore)
		l, b = judge(odd, 1)
		expect(t, l, b, referenceWeak, referenceBelowMedian)
	})
	t.Run("a single clip is weak even above the median", func(t *testing.T) {
		l, b := judge(library, 4)
		expect(t, l, b, referenceWeak, referenceSingleClip)
	})
	t.Run("no contributive sample is none", func(t *testing.T) {
		l, b := newReferenceStanding(library).judge(referenceFigure{})
		expect(t, l, b, referenceNone, referenceNoSample)
	})
	t.Run("under three References there is no comparison", func(t *testing.T) {
		pair := map[int64]referenceFigure{
			1: {Samples: 2, Clips: 2, NetSpeechSeconds: 2},
			2: {Samples: 9, Clips: 9, NetSpeechSeconds: 90},
		}
		l, b := judge(pair, 1)
		expect(t, l, b, referenceGood, referenceFewReferences)
		alone := map[int64]referenceFigure{1: {Samples: 1, Clips: 1, NetSpeechSeconds: 10}}
		l, b = judge(alone, 1)
		expect(t, l, b, referenceWeak, referenceSingleClip)
		l, b = newReferenceStanding(nil).judge(referenceFigure{})
		expect(t, l, b, referenceNone, referenceNoSample)
	})
}
