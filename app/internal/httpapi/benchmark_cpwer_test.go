package httpapi

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"snitch/internal/segmentparts"
)

func refTexts() []speakerText {
	return []speakerText{
		{0, 5, "alice", "bonjour tout le monde"},
		{5, 10, "bob", "salut alice"},
		{10, 15, "alice", "on commence"},
	}
}

func TestCPWERPerfectWhateverTheSpeakerNames(t *testing.T) {
	hyp := []speakerText{{0, 5, "SPEAKER_01", "Bonjour tout le monde."}, {5, 10, "SPEAKER_00", "Salut Alice"}, {10, 15, "SPEAKER_01", "on commence"}}
	m := evaluateCPWER([]attributedClip{{refTexts(), hyp}})
	if m.CPWER == nil || *m.CPWER != 0 || m.WER == nil || *m.WER != 0 || m.ReferenceWords != 8 {
		t.Fatalf("anonymous speaker names must not cost anything: %+v", m)
	}
}

func TestCPWERCountsAWrongSpeakerButTheSpeakerBlindWERDoesNot(t *testing.T) {
	// The right words, but bob's sentence is given to alice: the text is perfect, the attribution is not.
	hyp := []speakerText{{0, 5, "a", "bonjour tout le monde"}, {5, 10, "a", "salut alice"}, {10, 15, "a", "on commence"}}
	m := evaluateCPWER([]attributedClip{{refTexts(), hyp}})
	if m.WER == nil || *m.WER != 0 {
		t.Fatalf("the plain WER is blind to who speaks: %+v", m)
	}
	// Alice gets two words too many (insertions) and Bob loses his two (deletions): four errors out of eight.
	if m.CPWER == nil || math.Abs(*m.CPWER-4.0/8.0) > 1e-9 {
		t.Fatalf("a sentence given to the wrong speaker is an error twice: %+v", m)
	}
}

func TestCPWERWrongWordsAndMissingSpeaker(t *testing.T) {
	hyp := []speakerText{{0, 5, "a", "bonjour tout le monde"}, {10, 15, "a", "on commence"}}
	m := evaluateCPWER([]attributedClip{{refTexts(), hyp}})
	if m.CPWER == nil || math.Abs(*m.CPWER-2.0/8.0) > 1e-9 || m.ModelSpeakers != 1 || m.ReferenceSpeakers != 2 {
		t.Fatalf("a speaker the model never heard loses all their words: %+v", m)
	}
	if m := evaluateCPWER(nil); m.CPWER != nil {
		t.Fatalf("no clip, no rate: %+v", m)
	}
}

func TestAttributeSegmentsByOverlap(t *testing.T) {
	turns := []speakerTurn{{0, 6, "x"}, {6, 12, "y"}}
	got := attributeSegments(segmentparts.RuleSegment, []timedText{{start: 1, end: 5, text: "un"}, {start: 5, end: 12, text: "deux"}, {start: 20, end: 25, text: "trois"}}, turns)
	if got[0].speaker != "x" || got[1].speaker != "y" || got[2].speaker != "" {
		t.Fatalf("a segment goes to the speaker it overlaps the most, none when nobody does: %+v", got)
	}
}

// The segment and word cpWER of a chain show what the word rule changes: a segment where two voices answer each
// other is all one speaker's under the segment rule, split between both under the word rule.
func TestAttributeSegmentsUnderBothRules(t *testing.T) {
	turns := []speakerTurn{{0, 5, "S0"}, {5, 10, "S1"}, {10, 15, "S0"}}
	segments := []timedText{
		{start: 0, end: 10, text: "Bonjour tout le monde, salut Alice", words: []timedWord{
			{start: 0, end: 1, text: "Bonjour"}, {start: 1, end: 2, text: "tout"}, {start: 2, end: 3, text: "le"}, {start: 3, end: 4, text: "monde,"},
			{start: 5, end: 7, text: "salut"}, {start: 7, end: 9, text: "Alice"}}},
		{start: 10, end: 15, text: "on commence"}, // no word times: the whole segment under both rules
	}
	bySegment := evaluateCPWER([]attributedClip{{refTexts(), attributeSegments(segmentparts.RuleSegment, segments, turns)}})
	byWord := evaluateCPWER([]attributedClip{{refTexts(), attributeSegments(segmentparts.RuleWord, segments, turns)}})
	if *bySegment.CPWER != 4.0/8.0 || *byWord.CPWER != 0 || *bySegment.WER != 0 || *byWord.WER != 0 {
		t.Fatalf("the word rule gives bob his words back: segment %+v, word %+v", bySegment, byWord)
	}
}

// Non-drift (#400): the benchmark gives each word the voice production gives it, under both rules.
func TestBenchmarkAttributionIsProductionAttribution(t *testing.T) {
	turns := []speakerTurn{{0, 2, "A"}, {2, 3, "B"}, {2.5, 4, "C"}, {4, 5, "A"}, {7, 9, "B"}}
	ruleTurns := []segmentparts.Turn[string]{{Speaker: "A", Start: 0, End: 2}, {Speaker: "B", Start: 2, End: 3}, {Speaker: "C", Start: 2.5, End: 4}, {Speaker: "A", Start: 4, End: 5}, {Speaker: "B", Start: 7, End: 9}}
	var segments []timedText
	for _, bounds := range [][]float64{{0, 1, 1, 2, 2, 2.5, 2.5, 3}, {2, 3, 3, 3.5, 3.5, 4.5, 4.5, 5}, {0, 1, 1, 1.5, 1.5, 2.5}, {5.5, 6, 6, 7.5, 7.5, 8}, {}, {10, 11}} {
		seg := timedText{start: 0, end: 12}
		for i := 0; i+1 < len(bounds); i += 2 {
			seg.words = append(seg.words, timedWord{start: bounds[i], end: bounds[i+1], text: fmt.Sprintf("w%d", i/2)})
			seg.text = strings.TrimSpace(seg.text + fmt.Sprintf(" w%d", i/2))
		}
		if len(seg.words) > 0 {
			seg.start, seg.end = seg.words[0].start, seg.words[len(seg.words)-1].end
		}
		segments = append(segments, seg)
	}
	for _, rule := range []segmentparts.Rule{segmentparts.RuleSegment, segmentparts.RuleWord} {
		got := attributeSegments(rule, segments, turns)
		var benchmark, production []string
		for _, s := range got {
			for range max(1, len(strings.Fields(s.text))) {
				benchmark = append(benchmark, s.speaker)
			}
		}
		for _, seg := range segments {
			times := make([]segmentparts.Word, len(seg.words))
			for i, w := range seg.words {
				times[i] = segmentparts.Word{Start: w.start, End: w.end}
			}
			for _, p := range segmentparts.Attribute(rule, segmentparts.Span{Start: seg.start, End: seg.end}, times, ruleTurns) {
				for range max(1, p.Count) {
					production = append(production, p.Speaker)
				}
			}
		}
		if !slices.Equal(benchmark, production) {
			t.Fatalf("rule %d: the benchmark gives %v, production %v", rule, benchmark, production)
		}
	}
}
