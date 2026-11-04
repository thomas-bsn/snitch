package httpapi

import "testing"

// The reproducible case of #273: the diarization detects three voices, the transcript
// attributes words to two. Each detected voice must keep a distinct, explicit status.
func TestSpeakerCoverageThreeDetectedTwoTranscribed(t *testing.T) {
	turns := []coverageTurn{
		{SpeakerID: 1, Start: 0, End: 10},
		{SpeakerID: 2, Start: 10, End: 20},
		{SpeakerID: 3, Start: 4, End: 6},   // talks over voice 1, always outweighed
		{SpeakerID: 3, Start: 40, End: 45}, // talks where ASR produced nothing
	}
	segments := []coverageSegment{
		{Start: 0, End: 10},
		{Start: 10, End: 20},
	}
	got := speakerCoverage(turns, segments)
	want := map[int64]SpeakerCoverage{
		1: {SpeakerID: 1, Status: CoverageWords, SpeakingSeconds: 10, AttributedSegments: 1},
		2: {SpeakerID: 2, Status: CoverageWords, SpeakingSeconds: 10, AttributedSegments: 1},
		3: {SpeakerID: 3, Status: CoverageSpeechWithoutWords, SpeakingSeconds: 7, OverlappedSeconds: 2, UncoveredSeconds: 5},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d coverages, want %d: %+v", len(got), len(want), got)
	}
	for _, c := range got {
		if c != want[c.SpeakerID] {
			t.Errorf("speaker %d: got %+v, want %+v", c.SpeakerID, c, want[c.SpeakerID])
		}
	}
}

func TestSpeakerCoverageNoTranscriptAtAll(t *testing.T) {
	got := speakerCoverage([]coverageTurn{{SpeakerID: 7, Start: 5, End: 8}}, nil)
	if len(got) != 1 || got[0].Status != CoverageNoTranscript || got[0].SpeakingSeconds != 3 {
		t.Fatalf("got %+v", got)
	}
}

// A manual correction moves a segment to another voice: the coverage follows it.
func TestSpeakerCoverageFollowsOverride(t *testing.T) {
	override := int64(2)
	turns := []coverageTurn{{SpeakerID: 1, Start: 0, End: 10}, {SpeakerID: 2, Start: 12, End: 14}}
	got := speakerCoverage(turns, []coverageSegment{{Start: 0, End: 10, OverrideSpeakerID: &override}})
	byID := map[int64]SpeakerCoverage{}
	for _, c := range got {
		byID[c.SpeakerID] = c
	}
	if byID[2].Status != CoverageWords || byID[2].AttributedSegments != 1 {
		t.Errorf("override target: %+v", byID[2])
	}
	if byID[1].Status != CoverageSpeechWithoutWords {
		t.Errorf("overridden-away voice: %+v", byID[1])
	}
}

// Ties go to the first voice in input order, as the segment attribution does.
func TestSpeakerCoverageTieGoesToFirstVoice(t *testing.T) {
	turns := []coverageTurn{{SpeakerID: 1, Start: 0, End: 5}, {SpeakerID: 2, Start: 5, End: 10}}
	got := speakerCoverage(turns, []coverageSegment{{Start: 0, End: 10}})
	if got[0].Status != CoverageWords || got[1].Status != CoverageSpeechWithoutWords {
		t.Fatalf("got %+v", got)
	}
}

// Overlapping turns of one voice are not counted twice.
func TestSpeakerCoverageSpeakingSecondsIgnoresOverlappingTurns(t *testing.T) {
	got := speakerCoverage([]coverageTurn{{SpeakerID: 1, Start: 0, End: 6}, {SpeakerID: 1, Start: 4, End: 10}}, nil)
	if got[0].SpeakingSeconds != 10 {
		t.Fatalf("got %+v", got)
	}
}

// Speech that no segment covers is measured, and told apart: inside the transcribed span it is
// a gap of the ASR; beyond it, the transcript stops before the voice (truncation, offset).
func TestSpeakerCoverageTellsAGapFromSpeechBeyondTheTranscript(t *testing.T) {
	segments := []coverageSegment{{Start: 0, End: 10}, {Start: 20, End: 30}}
	got := speakerCoverage([]coverageTurn{
		{SpeakerID: 1, Start: 0, End: 10},
		{SpeakerID: 2, Start: 12, End: 18}, // between two segments
		{SpeakerID: 3, Start: 50, End: 55}, // after the last segment
		{SpeakerID: 4, Start: 8, End: 14},  // half under a segment, half in the gap
	}, segments)
	byID := map[int64]SpeakerCoverage{}
	for _, c := range got {
		byID[c.SpeakerID] = c
	}
	if c := byID[2]; c.Status != CoverageNoTranscript || c.UncoveredSeconds != 6 {
		t.Errorf("gap: %+v", c)
	}
	if c := byID[3]; c.Status != CoverageOutsideTranscript || c.UncoveredSeconds != 5 {
		t.Errorf("beyond: %+v", c)
	}
	if c := byID[4]; c.Status != CoverageSpeechWithoutWords || c.UncoveredSeconds != 4 || c.OverlappedSeconds != 2 {
		t.Errorf("partial: %+v", c)
	}
	if c := byID[1]; c.UncoveredSeconds != 0 {
		t.Errorf("fully covered: %+v", c)
	}
}
