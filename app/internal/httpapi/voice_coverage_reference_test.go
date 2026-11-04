package httpapi

import "testing"

// Reference scenarios of #273: synthetic ground truth (who really speaks when, what the ASR
// produced), one per cause of the ticket. They document the diagnosis and are the entry of
// the benchmark of #268 (docs/voice-coverage-benchmark.md). No media, no real clip.
func TestSpeakerCoverageReferenceScenarios(t *testing.T) {
	seg := func(a, b float64) coverageSegment { return coverageSegment{Start: a, End: b} }
	turn := func(id int64, a, b float64) coverageTurn { return coverageTurn{SpeakerID: id, Start: a, End: b} }
	scenarios := []struct {
		name     string
		turns    []coverageTurn
		segments []coverageSegment
		want     map[int64]CoverageStatus
	}{
		{
			name:     "four voices in turn, all transcribed",
			turns:    []coverageTurn{turn(1, 0, 5), turn(2, 5, 10), turn(3, 10, 15), turn(4, 15, 20)},
			segments: []coverageSegment{seg(0, 5), seg(5, 10), seg(10, 15), seg(15, 20)},
			want:     map[int64]CoverageStatus{1: CoverageWords, 2: CoverageWords, 3: CoverageWords, 4: CoverageWords},
		},
		{
			// (b) the words exist but the single speaker of a long segment is the loudest voice.
			name:     "simultaneous speech: a segment spanning two voices keeps one",
			turns:    []coverageTurn{turn(1, 0, 8), turn(2, 6, 10)},
			segments: []coverageSegment{seg(0, 10)},
			want:     map[int64]CoverageStatus{1: CoverageWords, 2: CoverageSpeechWithoutWords},
		},
		{
			// (a) a weak voice detected under a strong one, ASR heard only the strong one.
			name:     "weak voice under a strong one",
			turns:    []coverageTurn{turn(1, 0, 20), turn(2, 9, 11)},
			segments: []coverageSegment{seg(0, 20)},
			want:     map[int64]CoverageStatus{1: CoverageWords, 2: CoverageSpeechWithoutWords},
		},
		{
			// (a) the ASR produced nothing where the voice speaks, inside the transcribed span.
			name:     "ASR gap inside the transcript",
			turns:    []coverageTurn{turn(1, 0, 5), turn(2, 10, 14), turn(1, 20, 25)},
			segments: []coverageSegment{seg(0, 5), seg(20, 25)},
			want:     map[int64]CoverageStatus{1: CoverageWords, 2: CoverageNoTranscript},
		},
		{
			// (e) a constant offset between the two clocks: no voice meets any segment.
			name:     "transcript shifted by 30 s",
			turns:    []coverageTurn{turn(1, 0, 10), turn(2, 10, 20)},
			segments: []coverageSegment{seg(30, 40), seg(40, 50)},
			want:     map[int64]CoverageStatus{1: CoverageOutsideTranscript, 2: CoverageOutsideTranscript},
		},
		{
			// (d) the wrong track was transcribed: an empty transcript for a track with voices.
			name:     "empty transcript for a track with voices",
			turns:    []coverageTurn{turn(1, 0, 10), turn(2, 10, 20)},
			segments: nil,
			want:     map[int64]CoverageStatus{1: CoverageNoTranscript, 2: CoverageNoTranscript},
		},
	}
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			got := speakerCoverage(sc.turns, sc.segments)
			if len(got) != len(sc.want) {
				t.Fatalf("got %d voices, want %d: %+v", len(got), len(sc.want), got)
			}
			for _, c := range got {
				if c.Status != sc.want[c.SpeakerID] {
					t.Errorf("voice %d: %s, want %s (%+v)", c.SpeakerID, c.Status, sc.want[c.SpeakerID], c)
				}
			}
		})
	}
}

// Separate tracks (microphone / PC) are diagnosed apart: the same voice ids never mix, and a
// voice alone on its track is never outweighed.
func TestSpeakerCoverageSeparateTracksAreIndependent(t *testing.T) {
	mic := speakerCoverage([]coverageTurn{{SpeakerID: 1, Start: 0, End: 10}}, []coverageSegment{{Start: 0, End: 10}})
	pc := speakerCoverage([]coverageTurn{{SpeakerID: 2, Start: 0, End: 10}, {SpeakerID: 3, Start: 2, End: 4}}, []coverageSegment{{Start: 0, End: 10}})
	if mic[0].Status != CoverageWords || pc[0].Status != CoverageWords || pc[1].Status != CoverageSpeechWithoutWords {
		t.Fatalf("mic %+v pc %+v", mic, pc)
	}
}
