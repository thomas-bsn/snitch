package httpapi

import (
	"encoding/json"
	"testing"

	"snitch/internal/jobs"
)

// Without and with separation are read under the word rule that lists the speakers without words (#403): B's words
// of a segment A dominates stay B's, and only C, silent, gets the added word. The speakers without words after the
// merge are counted on the merged transcript.
func TestSeparateTrackReadsBothHypothesesUnderTheWordRule(t *testing.T) {
	var speakers []jobs.BenchmarkSpeaker
	var segments []jobs.BenchmarkSegment
	if err := json.Unmarshal([]byte(`[{"label":"A","turns":[{"start":0,"end":3}]},{"label":"B","turns":[{"start":3,"end":5}]},{"label":"C","turns":[{"start":3.5,"end":4.5}]}]`), &speakers); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`[{"start":0,"end":5,"text":"a b c d","words":[{"start":0,"end":1,"word":"a"},{"start":1,"end":2,"word":"b"},{"start":3,"end":4,"word":"c"},{"start":4,"end":5,"word":"d"}]}]`), &segments); err != nil {
		t.Fatal(err)
	}
	reference := []speakerText{{0, 2, "a", "a b"}, {3, 5, "b", "c d"}, {3.6, 3.9, "c", "x"}}
	raw := []byte(`{"audio_stream_index":0,"silent_speakers":["C"],"zones":[{"start":3.5,"end":4.5}],"sources":[{"zone":0,"speaker":"C","words":[{"start":3.6,"end":3.9,"word":"x"}]}],"added":[{"zone":0,"speaker":"C","start":3.6,"end":3.9,"word":"x"}]}`)
	var m separationMetrics
	var without, with []attributedClip
	separateTrack(&m, &without, &with, raw, speakers, segments, reference)
	// The segment rule would give "c d" to A: 3 errors out of 5 words instead of the one word C lacks.
	if got := evaluateCPWER(without); got.CPWordErrors != 1 || got.ReferenceWords != 5 {
		t.Fatalf("without separation: %+v", got)
	}
	if got := evaluateCPWER(with); got.CPWordErrors != 0 {
		t.Fatalf("with separation: %+v", got)
	}
	if m.SpeakersWithoutWordsBefore != 1 || m.SpeakersWithoutWordsAfter != 0 || m.WordsAdded != 1 {
		t.Fatalf("separation counts: %+v", m)
	}
}
