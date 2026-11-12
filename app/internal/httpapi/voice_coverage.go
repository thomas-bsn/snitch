package httpapi

import (
	"math"
	"sort"

	"snitch/internal/segmentparts"
)

// Coverage of the detected voices by the transcript (#273). The diarization and the ASR are
// independent: a voice can be detected without any word attributed to it. Snitch gives the words
// of a segment to the voices that say them (parts, ADR 0065), so a voice that never holds a part,
// or that speaks where the ASR produced no segment, vanishes from the transcript.
// speakerCoverage makes that explicit instead of silent.

// CoverageStatus tells what the transcript holds for a detected voice.
type CoverageStatus string

const (
	// CoverageWords: the voice holds a part of at least one segment.
	CoverageWords CoverageStatus = "words"
	// CoverageSpeechWithoutWords: the voice speaks over transcribed segments, but holds a part of
	// none (other voices outweigh it).
	CoverageSpeechWithoutWords CoverageStatus = "speech_without_words"
	// CoverageNoTranscript: no segment overlaps any turn of the voice (ASR gap, silence
	// seen by the ASR, offset or wrong track).
	CoverageNoTranscript CoverageStatus = "no_transcript"
	// CoverageOutsideTranscript: like no_transcript, but all the speech lies before or after the
	// transcribed span (truncated transcript, offset, wrong track), not in a gap inside it.
	CoverageOutsideTranscript CoverageStatus = "outside_transcript"
)

type coverageTurn struct {
	SpeakerID  int64
	Start, End float64
}

type coverageSegment struct {
	Start, End float64
	// OverrideSpeakerID is a manual correction of the segment's speaker.
	OverrideSpeakerID *int64
	// Words are the times of the words of the segment, in order. A segment whose text was corrected has
	// none: its original word times no longer describe it (ADR 0012).
	Words []segmentparts.Word
}

// partTurns gives the turns to the attribution rule, in the same order.
func partTurns(turns []coverageTurn) []segmentparts.Turn[int64] {
	out := make([]segmentparts.Turn[int64], len(turns))
	for i, t := range turns {
		out[i] = segmentparts.Turn[int64]{Speaker: t.SpeakerID, Start: t.Start, End: t.End}
	}
	return out
}

// parts cuts the segment into its parts: the one attribution of the clip page, of the Coverage and of the trace.
// A speaker Correction gives the whole segment to the designated voice, whatever the rule (ADR 0012); the caller
// leaves OverrideSpeakerID nil when that voice cannot be shown.
func (seg coverageSegment) parts(rule segmentparts.Rule, turns []segmentparts.Turn[int64]) []segmentparts.Part[int64] {
	if seg.OverrideSpeakerID != nil {
		return []segmentparts.Part[int64]{{Speaker: *seg.OverrideSpeakerID, Known: true, Count: len(seg.Words)}}
	}
	return segmentparts.Attribute(rule, segmentparts.Span{Start: seg.Start, End: seg.End}, seg.Words, turns)
}

// SpeakerCoverage is the per-voice result, in order of first appearance in the turns.
type SpeakerCoverage struct {
	SpeakerID          int64          `json:"local_speaker_id"`
	Status             CoverageStatus `json:"coverage_status"`
	SpeakingSeconds    float64        `json:"speaking_seconds"`
	OverlappedSeconds  float64        `json:"overlapped_seconds"` // speech over segments where the voice holds no part
	UncoveredSeconds   float64        `json:"uncovered_seconds"`  // speech under no segment at all
	AttributedSegments int            `json:"attributed_segments"`
}

// segmentOverlaps sums, per voice, the time its turns share with [start,end]. order lists the
// overlapping voices by first appearance in turns.
func segmentOverlaps(turns []coverageTurn, start, end float64) (order []int64, overlap map[int64]float64) {
	overlap = map[int64]float64{}
	for _, t := range turns {
		if shared := math.Min(end, t.End) - math.Max(start, t.Start); shared > 0 {
			if _, seen := overlap[t.SpeakerID]; !seen {
				order = append(order, t.SpeakerID)
			}
			overlap[t.SpeakerID] += shared
		}
	}
	return order, overlap
}

// speakerCoverage applies the attribution rule, plus manual corrections, to every segment. A voice that holds a
// part of a segment is not counted as speaking under the words of the others there.
func speakerCoverage(rule segmentparts.Rule, turns []coverageTurn, segments []coverageSegment) []SpeakerCoverage {
	result := []SpeakerCoverage{}
	index := map[int64]int{}
	for _, t := range turns {
		i, seen := index[t.SpeakerID]
		if !seen {
			i = len(result)
			index[t.SpeakerID] = i
			result = append(result, SpeakerCoverage{SpeakerID: t.SpeakerID})
		}
	}
	for i := range result {
		result[i].SpeakingSeconds = unionSeconds(turns, result[i].SpeakerID)
	}
	ruleTurns := partTurns(turns)
	for _, seg := range segments {
		order, overlap := segmentOverlaps(turns, seg.Start, seg.End)
		if seg.OverrideSpeakerID != nil {
			if _, known := index[*seg.OverrideSpeakerID]; !known {
				seg.OverrideSpeakerID = nil
			}
		}
		holders := map[int64]bool{}
		for _, part := range seg.parts(rule, ruleTurns) {
			if part.Known && !holders[part.Speaker] {
				holders[part.Speaker] = true
				result[index[part.Speaker]].AttributedSegments++
			}
		}
		for _, id := range order {
			if !holders[id] {
				result[index[id]].OverlappedSeconds += overlap[id]
			}
		}
	}
	first, last := math.Inf(1), math.Inf(-1)
	covered := make([]coverageTurn, 0, len(segments))
	for _, seg := range segments {
		first, last = math.Min(first, seg.Start), math.Max(last, seg.End)
		covered = append(covered, coverageTurn{Start: seg.Start, End: seg.End})
	}
	covered = mergeSpans(covered)
	for i := range result {
		c := &result[i]
		c.UncoveredSeconds = c.SpeakingSeconds
		for _, span := range mergeSpans(spansOf(turns, c.SpeakerID)) {
			for _, seg := range covered {
				c.UncoveredSeconds -= math.Max(0, math.Min(span.End, seg.End)-math.Max(span.Start, seg.Start))
			}
		}
		switch {
		case c.AttributedSegments > 0:
			c.Status = CoverageWords
		case c.OverlappedSeconds > 0:
			c.Status = CoverageSpeechWithoutWords
		case len(segments) > 0 && !touchesSpan(turns, c.SpeakerID, first, last):
			c.Status = CoverageOutsideTranscript
		default:
			c.Status = CoverageNoTranscript
		}
	}
	return result
}

// spansOf are the non-empty turns of a voice.
func spansOf(turns []coverageTurn, speaker int64) []coverageTurn {
	spans := []coverageTurn{}
	for _, t := range turns {
		if t.SpeakerID == speaker && t.End > t.Start {
			spans = append(spans, t)
		}
	}
	return spans
}

// mergeSpans sorts spans and joins those that overlap.
func mergeSpans(spans []coverageTurn) []coverageTurn {
	sorted := append([]coverageTurn(nil), spans...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })
	merged := []coverageTurn{}
	for _, t := range sorted {
		if n := len(merged); n > 0 && t.Start <= merged[n-1].End {
			merged[n-1].End = math.Max(merged[n-1].End, t.End)
			continue
		}
		merged = append(merged, t)
	}
	return merged
}

// unionSeconds is the speech time of a voice, without counting twice turns that overlap.
func unionSeconds(turns []coverageTurn, speaker int64) float64 {
	total := 0.0
	for _, t := range mergeSpans(spansOf(turns, speaker)) {
		total += t.End - t.Start
	}
	return total
}

// touchesSpan tells whether any turn of the voice reaches into [first,last].
func touchesSpan(turns []coverageTurn, speaker int64, first, last float64) bool {
	for _, t := range spansOf(turns, speaker) {
		if t.End > first && t.Start < last {
			return true
		}
	}
	return false
}
