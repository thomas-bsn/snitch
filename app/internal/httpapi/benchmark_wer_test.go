package httpapi

import (
	"math"
	"testing"
)

func TestNormalizeWords(t *testing.T) {
	got := normalizeWords("Bonjour, l’équipe ! C'est 3,5 %… Été.")
	want := []string{"bonjour", "l", "équipe", "c", "est", "3", "5", "été"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestEditDistance(t *testing.T) {
	cases := []struct {
		a, b []string
		want int
	}{
		{nil, nil, 0},
		{[]string{"a"}, nil, 1},
		{nil, []string{"a", "b"}, 2},
		{[]string{"a", "b", "c"}, []string{"a", "x", "c"}, 1},
		{[]string{"a", "b", "c"}, []string{"a", "c"}, 1},
		{[]string{"le", "chat", "dort"}, []string{"un", "chat", "dort", "ici"}, 2},
	}
	for _, c := range cases {
		if got := editDistance(c.a, c.b); got != c.want {
			t.Fatalf("%v vs %v: got %d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestEvaluateTranscripts(t *testing.T) {
	// Case and punctuation are ignored; one substituted word out of five.
	m := evaluateTranscripts([]transcriptPair{{"Le chat dort sur le canapé.", "le chat dort sur LE canape"}})
	if m.WER == nil || math.Abs(*m.WER-1.0/6.0) > 1e-9 || m.ReferenceWords != 6 || m.WordErrors != 1 {
		t.Fatalf("an accent is a character the model got wrong: %+v", m)
	}
	if m.CER == nil || *m.CER <= 0 || *m.CER >= 0.2 {
		t.Fatalf("one wrong accent is a small character error, got %v", m.CER)
	}
	// Micro-average: errors and reference words are summed over the clips.
	m = evaluateTranscripts([]transcriptPair{{"a b c d", "a b c d"}, {"e f", "x y"}})
	if m.WER == nil || math.Abs(*m.WER-2.0/6.0) > 1e-9 || m.Clips != 2 {
		t.Fatalf("clips are summed, not averaged: %+v", m)
	}
	// Nothing to compare gives no rate rather than zero.
	m = evaluateTranscripts(nil)
	if m.WER != nil || m.CER != nil {
		t.Fatalf("no clip, no rate: %+v", m)
	}
	// A model that says nothing makes every reference word an error.
	m = evaluateTranscripts([]transcriptPair{{"un deux trois", ""}})
	if m.WER == nil || *m.WER != 1 {
		t.Fatalf("a silent model has a WER of 100%%: %+v", m)
	}
}
