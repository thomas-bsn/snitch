package httpapi

import "math"

// Coverage of the detected voices by the transcript (#273). The diarization and the ASR are
// independent: a voice can be detected without any word attributed to it. Snitch attributes
// each segment to a single voice (the one that speaks most over it), so a voice that is always
// outweighed, or that speaks where the ASR produced no segment, vanishes from the transcript.
// speakerCoverage makes that explicit instead of silent.

// CoverageStatus tells what the transcript holds for a detected voice.
type CoverageStatus string

const (
	// CoverageWords: at least one segment is attributed to the voice.
	CoverageWords CoverageStatus = "words"
	// CoverageSpeechWithoutWords: the voice speaks over transcribed segments, but none is
	// attributed to it (another voice outweighs it).
	CoverageSpeechWithoutWords CoverageStatus = "speech_without_words"
	// CoverageNoTranscript: no segment overlaps any turn of the voice (ASR gap, silence
	// seen by the ASR, offset or wrong track).
	CoverageNoTranscript CoverageStatus = "no_transcript"
)

type coverageTurn struct {
	SpeakerID  int64
	Start, End float64
}

type coverageSegment struct {
	Start, End float64
	// OverrideSpeakerID is a manual correction of the segment's speaker.
	OverrideSpeakerID *int64
}

// SpeakerCoverage is the per-voice result, in order of first appearance in the turns.
type SpeakerCoverage struct {
	SpeakerID          int64          `json:"local_speaker_id"`
	Status             CoverageStatus `json:"coverage_status"`
	SpeakingSeconds    float64        `json:"speaking_seconds"`
	OverlappedSeconds  float64        `json:"overlapped_seconds"` // speech over segments attributed to another voice
	AttributedSegments int            `json:"attributed_segments"`
}

// speakerCoverage applies the attribution rule of attachLocalSpeakers: most overlap wins,
// ties go to the voice that comes first in turns, a correction overrides.
func speakerCoverage(turns []coverageTurn, segments []coverageSegment) []SpeakerCoverage {
	result := []SpeakerCoverage{}
	index := map[int64]int{}
	for _, t := range turns {
		i, seen := index[t.SpeakerID]
		if !seen {
			i = len(result)
			index[t.SpeakerID] = i
			result = append(result, SpeakerCoverage{SpeakerID: t.SpeakerID})
		}
		result[i].SpeakingSeconds += t.End - t.Start
	}
	for _, seg := range segments {
		overlap := map[int64]float64{}
		order := []int64{}
		for _, t := range turns {
			if shared := math.Min(seg.End, t.End) - math.Max(seg.Start, t.Start); shared > 0 {
				if _, seen := overlap[t.SpeakerID]; !seen {
					order = append(order, t.SpeakerID)
				}
				overlap[t.SpeakerID] += shared
			}
		}
		winner, found := int64(0), false
		for _, id := range order {
			if !found || overlap[id] > overlap[winner] {
				winner, found = id, true
			}
		}
		if seg.OverrideSpeakerID != nil {
			if _, known := index[*seg.OverrideSpeakerID]; known {
				winner, found = *seg.OverrideSpeakerID, true
			}
		}
		for _, id := range order {
			if !found || id != winner {
				result[index[id]].OverlappedSeconds += overlap[id]
			}
		}
		if found {
			result[index[winner]].AttributedSegments++
		}
	}
	for i := range result {
		c := &result[i]
		switch {
		case c.AttributedSegments > 0:
			c.Status = CoverageWords
		case c.OverlappedSeconds > 0:
			c.Status = CoverageSpeechWithoutWords
		default:
			c.Status = CoverageNoTranscript
		}
	}
	return result
}
