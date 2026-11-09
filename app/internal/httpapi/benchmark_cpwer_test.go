package httpapi

import (
	"math"
	"testing"
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
	got := attributeSegments([]timedText{{1, 5, "un"}, {5, 12, "deux"}, {20, 25, "trois"}}, turns)
	if got[0].speaker != "x" || got[1].speaker != "y" || got[2].speaker != "" {
		t.Fatalf("a segment goes to the speaker it overlaps the most, none when nobody does: %+v", got)
	}
}
