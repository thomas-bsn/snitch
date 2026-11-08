package audioplans

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"snitch/internal/httpapi/contract"
)

// Acoustic deduplication (#343, ADR 0053). The Worker adapter produces evidence from
// decoded samples; this file turns accepted evidence into manifest groups. It never reads
// text, speakers or Corrections: text equality or timestamp proximity can only shortlist.

// EvidenceSchemaVersion is the version of the Worker adapter report parsed here.
const EvidenceSchemaVersion = 1

// BoundaryTolerance (seconds) absorbs the 10 ms block quantization of the adapter and ASR
// timestamp rounding at segment edges. It is not an acoustic threshold.
const BoundaryTolerance = 0.05

// enabledStrategies lists the rules allowed to suppress, with their rule version. It must
// equal acoustic_dedup.ENABLED_STRATEGIES in the Worker (contract test). Codec and
// clean-versus-mix comparisons are evaluated by the Worker but never appear here.
var enabledStrategies = map[string]string{"exact_pcm": "1", "gain": "1"}

// MergeVersion is the AudioPlanInput.MergeVersion of the acoustic strategy. It derives
// from the enabled rules, so enabling, disabling or changing one changes the Plan's input
// version and goes through the same replacement control as any other reanalysis.
func MergeVersion() string {
	names := make([]string, 0, len(enabledStrategies))
	for name := range enabledStrategies {
		names = append(names, name)
	}
	sort.Strings(names)
	out := "acoustic-dedup/1"
	for _, name := range names {
		out += fmt.Sprintf("+%s@%s", name, enabledStrategies[name])
	}
	return out
}

//go:embed fixtures/acoustic-evidence.json
var evidenceFixture []byte

// EvidenceFixture is a real adapter report over synthetic speech (see
// worker/tests/fixtures/acoustic/generate_evidence.py): tracks 2 and 7 are the same PCM,
// 11 the same speech with a 0.5 gain, 9 the same words by another voice, 12 an AAC copy,
// 13 the speech plus a synthetic game bed.
func EvidenceFixture() []byte { return append([]byte(nil), evidenceFixture...) }

var ErrEvidence = errors.New("unsupported acoustic evidence report")

type EvidenceEvent struct {
	ID         string  `json:"id"`
	TrackIndex int     `json:"track_index"`
	JobID      *int64  `json:"job_id"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
}
type EvidenceRange struct {
	LeftStart  float64 `json:"left_start"`
	LeftEnd    float64 `json:"left_end"`
	RightStart float64 `json:"right_start"`
	RightEnd   float64 `json:"right_end"`
}
type Comparison struct {
	Left            EvidenceEvent   `json:"left"`
	Right           EvidenceEvent   `json:"right"`
	Decision        string          `json:"decision"`
	EvidenceKind    string          `json:"evidence_kind"`
	Strategy        *string         `json:"strategy"`
	StrategyVersion *string         `json:"strategy_version"`
	ReasonCodes     []string        `json:"reason_codes"`
	LagSeconds      *float64        `json:"lag_seconds"`
	Gain            *float64        `json:"gain"`
	Matched         []EvidenceRange `json:"matched"`
}

// EvidenceReport is the JSON result of the Worker adapter (acoustic_dedup.run).
type EvidenceReport struct {
	SchemaVersion int          `json:"schema_version"`
	Comparisons   []Comparison `json:"comparisons"`
}

// ParseEvidence decodes an adapter report. An unknown schema is an error, not a silent
// "no evidence": the caller then keeps every source.
func ParseEvidence(raw []byte) (EvidenceReport, error) {
	var r EvidenceReport
	if err := json.Unmarshal(raw, &r); err != nil || r.SchemaVersion != EvidenceSchemaVersion {
		return EvidenceReport{}, ErrEvidence
	}
	return r, nil
}

// SourceSegment is one segment of a completed transcription child, in clip time.
type SourceSegment struct {
	Source contract.TranscriptSource
	Start  float64
	End    float64
	Words  []contract.TranscriptManifestWord
}

// MergeResult carries the manifest segments (unsorted indices are fixed by
// CanonicalRevision) and, for each suppressed source, the retained one it joined.
type MergeResult struct {
	Segments   []contract.TranscriptManifestSegment
	Suppressed []contract.TranscriptGrouping // Source is the suppressed one
	Retained   map[SourceKey]contract.TranscriptSource
}

// SourceKey identifies a source segment independently of word indices.
type SourceKey struct {
	Track   int
	Job     int64
	Segment int
}

func keyOf(s contract.TranscriptSource) SourceKey {
	return SourceKey{s.TrackIndex, s.JobId, s.SegmentIndex}
}

// piece is one proven identical range: retained side, grouped side (clip seconds).
type piece struct {
	low, high [2]float64
	info      contract.TranscriptGrouping
}

type group struct {
	seg     SourceSegment
	members []contract.TranscriptGrouping
}

func (p piece) offset() float64 { return p.low[0] - p.high[0] }

// MergeAcoustic groups segments proven duplicate by accepted evidence. Rules:
//
//   - only `duplicate` comparisons of an enabled rule at its current version count; every
//     other decision keeps both sources;
//   - the representative is always the segment of the lowest real track index (then job,
//     segment), the one whose identity already carries corrections: a correction is never
//     moved to another source, and the result does not depend on arrival order;
//   - a segment is grouped only if its whole interval is covered by direct proven ranges against
//     a track of lower index and the matching interval is covered by segments that are
//     themselves retained: there is direct evidence against the retained representative,
//     never a transitive chain A~B, B~C;
//   - a segment only partly proven (partial overlap) or uncovered on the other side stays
//     whole; text is never cut from acoustic windows;
//   - every source joins exactly one retained segment (the one it overlaps most).
func MergeAcoustic(segments []SourceSegment, report EvidenceReport) (MergeResult, error) {
	ordered := append([]SourceSegment(nil), segments...)
	seen := map[SourceKey]bool{}
	for _, s := range ordered {
		k := keyOf(s.Source)
		if s.Source.WordIndex != nil || s.Source.TrackIndex < 0 || s.Source.JobId < 1 || s.Source.SegmentIndex < 0 || seen[k] || !validTimes(s.Start, s.End) || s.End <= s.Start {
			return MergeResult{}, ErrConflict
		}
		seen[k] = true
	}
	sort.Slice(ordered, func(i, j int) bool { return sourceLess(ordered[i].Source, ordered[j].Source) })

	pieces := map[[2]int][]piece{} // (retained track, grouped track)
	for _, c := range report.Comparisons {
		if c.Decision != "duplicate" || c.Strategy == nil || c.StrategyVersion == nil || enabledStrategies[*c.Strategy] != *c.StrategyVersion || c.Left.TrackIndex == c.Right.TrackIndex {
			continue
		}
		low, high := c.Left, c.Right
		swap := c.Left.TrackIndex > c.Right.TrackIndex
		if swap {
			low, high = c.Right, c.Left
		}
		key := [2]int{low.TrackIndex, high.TrackIndex}
		for _, m := range c.Matched {
			p := piece{low: [2]float64{m.LeftStart, m.LeftEnd}, high: [2]float64{m.RightStart, m.RightEnd}}
			if swap {
				p.low, p.high = p.high, p.low
			}
			if !validTimes(p.low[0], p.low[1]) || !validTimes(p.high[0], p.high[1]) || p.low[1] <= p.low[0] || p.high[1] <= p.high[0] {
				continue
			}
			p.info = contract.TranscriptGrouping{Strategy: contract.TranscriptGroupingStrategy(*c.Strategy), StrategyVersion: *c.StrategyVersion, Gain: c.Gain}
			if c.LagSeconds != nil {
				p.info.LagSeconds = *c.LagSeconds
			}
			pieces[key] = append(pieces[key], p)
		}
	}

	kept := []*group{}
	result := MergeResult{Retained: map[SourceKey]contract.TranscriptSource{}}
	byTrack := map[int][]*group{}
	for _, s := range ordered {
		target, grouping := (*group)(nil), contract.TranscriptGrouping{}
		tracks := []int{}
		for key := range pieces {
			if key[1] == s.Source.TrackIndex {
				tracks = append(tracks, key[0])
			}
		}
		sort.Ints(tracks)
		for _, track := range tracks {
			key := [2]int{track, s.Source.TrackIndex}
			proofs := append([]piece(nil), pieces[key]...)
			sort.SliceStable(proofs, func(i, j int) bool { return proofs[i].high[0] < proofs[j].high[0] })
			cursor := s.Start
			var representative *group
			var proof contract.TranscriptGrouping
			bestOverlap := 0.0
			for _, p := range proofs {
				if p.high[1] <= cursor || p.high[0] > cursor+BoundaryTolerance {
					continue
				}
				from, to := math.Max(s.Start, p.high[0]), math.Min(s.End, p.high[1])
				if to <= from {
					continue
				}
				image := [2]float64{math.Max(p.low[0], from+p.offset()), math.Min(p.low[1], to+p.offset())}
				best, ok := coverImage(byTrack[track], image)
				if !ok {
					continue
				}
				if overlap := image[1] - image[0]; overlap > bestOverlap {
					representative, proof, bestOverlap = best, p.info, overlap
					proof.MatchedStart, proof.MatchedEnd = image[0], image[1]
				}
				cursor = math.Max(cursor, to)
				if cursor >= s.End-BoundaryTolerance {
					break
				}
			}
			if representative != nil && cursor >= s.End-BoundaryTolerance {
				target, grouping = representative, proof
				grouping.Source = s.Source
			}
			if target != nil {
				break
			}
		}
		if target == nil {
			r := &group{seg: s}
			kept = append(kept, r)
			byTrack[s.Source.TrackIndex] = append(byTrack[s.Source.TrackIndex], r)
			continue
		}
		target.members = append(target.members, grouping)
		result.Suppressed = append(result.Suppressed, grouping)
		result.Retained[keyOf(s.Source)] = target.seg.Source
	}
	for _, r := range kept {
		segment := contract.TranscriptManifestSegment{Start: r.seg.Start, End: r.seg.End, Sources: []contract.TranscriptSource{r.seg.Source}, Words: append([]contract.TranscriptManifestWord(nil), r.seg.Words...)}
		for _, m := range r.members {
			segment.Sources = append(segment.Sources, m.Source)
		}
		if len(r.members) > 0 {
			g := append([]contract.TranscriptGrouping(nil), r.members...)
			segment.Grouping = &g
		}
		result.Segments = append(result.Segments, segment)
	}
	return result, nil
}

// coverImage returns the retained segment overlapping image the most, provided the
// retained segments of that track cover the whole image (gaps up to BoundaryTolerance).
// If the other track has nothing there, dropping the grouped source would lose its words.
func coverImage(candidates []*group, image [2]float64) (*group, bool) {
	if image[1]-image[0] <= 0 {
		return nil, false
	}
	cursor := image[0]
	var best *group
	bestOverlap := 0.0
	for _, c := range candidates { // candidates are in canonical source order, so by start within a job
		if c.seg.End <= image[0] || c.seg.Start >= image[1] {
			continue
		}
		overlap := math.Min(c.seg.End, image[1]) - math.Max(c.seg.Start, image[0])
		if overlap > bestOverlap+1e-9 {
			best, bestOverlap = c, overlap
		}
	}
	covering := []*group{}
	for _, c := range candidates {
		if c.seg.End > image[0]-BoundaryTolerance && c.seg.Start < image[1]+BoundaryTolerance {
			covering = append(covering, c)
		}
	}
	sort.SliceStable(covering, func(i, j int) bool { return covering[i].seg.Start < covering[j].seg.Start })
	for _, c := range covering {
		if c.seg.Start > cursor+BoundaryTolerance {
			break
		}
		cursor = math.Max(cursor, c.seg.End)
	}
	return best, best != nil && cursor >= image[1]-BoundaryTolerance
}
