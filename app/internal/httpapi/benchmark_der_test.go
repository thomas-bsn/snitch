package httpapi

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 0.03 }

func ref() []speakerTurn {
	return []speakerTurn{{0, 10, "alice"}, {10, 20, "bob"}, {20, 30, "alice"}}
}

func TestDERPerfectWhateverTheLabels(t *testing.T) {
	// The model's speakers are anonymous: swapped names must not cost anything.
	hyp := []speakerTurn{{0, 10, "SPEAKER_01"}, {10, 20, "SPEAKER_00"}, {20, 30, "SPEAKER_01"}}
	m := evaluateDiarization([]clipTurns{{ref(), hyp}})
	if m.DER == nil || *m.DER > 0.001 {
		t.Fatalf("a perfect diarization has no error whatever its labels: %+v", m)
	}
}

func TestDERMissedFalseAlarmAndConfusion(t *testing.T) {
	// Missed: the model never hears the last 10 seconds of alice (10 of 30 reference seconds, minus the collar).
	missed := evaluateDiarization([]clipTurns{{ref(), []speakerTurn{{0, 10, "a"}, {10, 20, "b"}}}})
	if missed.MissedRate == nil || !near(*missed.MissedRate, 9.5/29.0) || *missed.FalseAlarmRate > 0.001 {
		t.Fatalf("a silent last third is missed speech: %+v", missed)
	}
	// False alarm: the model speaks over silence that follows the reference.
	invented := evaluateDiarization([]clipTurns{{ref(), append(ref(), speakerTurn{40, 50, "x"})}})
	if invented.FalseAlarmRate == nil || *invented.FalseAlarmRate <= 0.3 || *invented.MissedRate > 0.001 {
		t.Fatalf("speech where nobody speaks is a false alarm: %+v", invented)
	}
	// Confusion: one speaker only, so bob's ten seconds go to the wrong one.
	merged := evaluateDiarization([]clipTurns{{ref(), []speakerTurn{{0, 30, "only"}}}})
	if merged.ConfusionRate == nil || !near(*merged.ConfusionRate, 9.5/29.0) || *merged.MissedRate > 0.001 {
		t.Fatalf("merging two speakers is confusion: %+v", merged)
	}
}

func TestDERForgivesBoundariesWithinTheCollar(t *testing.T) {
	hyp := []speakerTurn{{0, 10.2, "a"}, {10.2, 19.8, "b"}, {19.8, 30, "a"}}
	m := evaluateDiarization([]clipTurns{{ref(), hyp}})
	if m.DER == nil || *m.DER > 0.001 {
		t.Fatalf("a boundary 200 ms off is within the collar: %+v", m)
	}
	late := []speakerTurn{{0, 11, "a"}, {11, 19, "b"}, {19, 30, "a"}}
	if m := evaluateDiarization([]clipTurns{{ref(), late}}); m.DER == nil || *m.DER < 0.04 {
		t.Fatalf("a boundary a second off is an error: %+v", m)
	}
}

func TestDERNoReferenceSpeechAndOverlap(t *testing.T) {
	if m := evaluateDiarization([]clipTurns{{nil, []speakerTurn{{0, 5, "a"}}}}); m.DER != nil {
		t.Fatalf("no reference speech, no rate: %+v", m)
	}
	// Two speakers talk together; the model hears only one of them: half of the overlap is missed.
	overlap := []speakerTurn{{0, 20, "a"}, {5, 15, "b"}}
	m := evaluateDiarization([]clipTurns{{overlap, []speakerTurn{{0, 20, "x"}}}})
	if m.MissedRate == nil || *m.MissedRate <= 0.1 || *m.MissedRate >= 0.5 {
		t.Fatalf("a speaker lost in an overlap is missed speech: %+v", m)
	}
}

func TestAssignSpeakersMaximisesCommonTime(t *testing.T) {
	got := assignSpeakers([][]int{{9, 1, 0}, {2, 8, 0}})
	if got[0] != 0 || got[1] != 1 {
		t.Fatalf("rows go to their best columns: %v", got)
	}
	// More reference speakers than hypothesis speakers: one stays unmatched.
	got = assignSpeakers([][]int{{9}, {8}})
	if (got[0] == -1) == (got[1] == -1) {
		t.Fatalf("exactly one reference speaker is matched: %v", got)
	}
	if got := assignSpeakers(nil); len(got) != 0 {
		t.Fatalf("nothing to assign: %v", got)
	}
}
