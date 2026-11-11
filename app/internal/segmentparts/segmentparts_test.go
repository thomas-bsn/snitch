package segmentparts

import (
	"reflect"
	"testing"
)

func words(bounds ...float64) []Word {
	out := []Word{}
	for i := 0; i+1 < len(bounds); i += 2 {
		out = append(out, Word{Start: bounds[i], End: bounds[i+1]})
	}
	return out
}

func part(speaker string, first, count int) Part[string] {
	return Part[string]{Speaker: speaker, Known: true, First: first, Count: count}
}

func check(t *testing.T, got []Part[string], want ...Part[string]) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestTwoVoicesAnsweringEachOtherGiveTwoParts(t *testing.T) {
	turns := []Turn[string]{{"A", 0, 2}, {"B", 2, 3}}
	got := Attribute(RuleWord, Span{0, 3}, words(0, 1, 1, 2, 2, 2.5, 2.5, 3), turns)
	check(t, got, part("A", 0, 2), part("B", 2, 2))
}

func TestSegmentRuleGivesTheWholeSegmentToTheDominantVoice(t *testing.T) {
	turns := []Turn[string]{{"A", 0, 2}, {"B", 2, 3}}
	got := Attribute(RuleSegment, Span{0, 3}, words(0, 1, 1, 2, 2, 2.5, 2.5, 3), turns)
	check(t, got, part("A", 0, 4))
}

func TestAWordAloneBetweenTwoWordsOfAnotherVoiceMakesNoPart(t *testing.T) {
	turns := []Turn[string]{{"A", 0, 1}, {"B", 1, 2}, {"A", 2, 3}}
	got := Attribute(RuleWord, Span{0, 3}, words(0, 1, 1, 2, 2, 3), turns)
	check(t, got, part("A", 0, 3))
}

func TestAWordAloneAtTheEdgeOfTheSegmentKeepsItsVoice(t *testing.T) {
	turns := []Turn[string]{{"A", 0, 2}, {"B", 2, 3}}
	got := Attribute(RuleWord, Span{0, 3}, words(0, 1, 1, 2, 2, 3), turns)
	check(t, got, part("A", 0, 2), part("B", 2, 1))
}

func TestAWordWithoutTurnTakesItsNeighbour(t *testing.T) {
	// No turn over the first and third words: the first takes the next covered word, the third the previous one.
	turns := []Turn[string]{{"A", 1, 2}, {"B", 3, 5}}
	got := Attribute(RuleWord, Span{0, 5}, words(0, 1, 1, 2, 2, 3, 3, 4, 4, 5), turns)
	check(t, got, part("A", 0, 3), part("B", 3, 2))
}

func TestASegmentWithoutAnyTurnHasNoSpeaker(t *testing.T) {
	for _, rule := range []Rule{RuleSegment, RuleWord} {
		got := Attribute(rule, Span{0, 2}, words(0, 1, 1, 2), []Turn[string]{{"A", 5, 6}})
		check(t, got, Part[string]{First: 0, Count: 2})
	}
}

func TestATieKeepsTheVoiceOfThePreviousWord(t *testing.T) {
	// Both voices cover the second word entirely; the first word is B's alone.
	turns := []Turn[string]{{"A", 1, 2}, {"B", 0, 2}, {"A", 2, 4}}
	got := Attribute(RuleWord, Span{0, 4}, words(0, 1, 1, 2, 2, 3, 3, 4), turns)
	check(t, got, part("B", 0, 2), part("A", 2, 2))
}

func TestATieWithoutPreviousWordGoesToTheFirstVoiceOfTheTurns(t *testing.T) {
	turns := []Turn[string]{{"A", 0, 1}, {"B", 0, 1}}
	got := Attribute(RuleWord, Span{0, 1}, words(0, 1), turns)
	check(t, got, part("A", 0, 1))
}

func TestWordsThatNoTurnCoversFallBackToTheSegmentRule(t *testing.T) {
	// The turn overlaps the segment but none of its words (or the words carry no time).
	turns := []Turn[string]{{"A", 1, 2}}
	for _, ws := range [][]Word{words(0, 1, 2, 3), words(0, 0, 0, 0), nil} {
		got := Attribute(RuleWord, Span{0, 3}, ws, turns)
		check(t, got, part("A", 0, len(ws)))
	}
}

func TestAlternatingSingleWordsCollapseIntoOneVoice(t *testing.T) {
	turns := []Turn[string]{{"A", 0, 1}, {"B", 1, 2}, {"A", 2, 3}, {"B", 3, 4}, {"A", 4, 5}}
	got := Attribute(RuleWord, Span{0, 5}, words(0, 1, 1, 2, 2, 3, 3, 4, 4, 5), turns)
	check(t, got, part("A", 0, 5))
}

func TestAnUnknownRuleIsReadAsTheSegmentRule(t *testing.T) {
	turns := []Turn[string]{{"A", 0, 2}, {"B", 2, 3}}
	got := Attribute(Rule(0), Span{0, 3}, words(0, 1, 1, 2, 2, 3), turns)
	check(t, got, part("A", 0, 3))
}
