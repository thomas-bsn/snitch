// Package segmentparts gives the words of a transcript segment to the Local speakers of its track (ADR 0065).
// It is the one Go attribution rule: the clip page, the Coverage of a Local speaker and the end-to-end benchmark call
// Attribute with plain values, so they cannot drift apart. SQL has its one twin, held to this one by a parity test:
// segment_part_speakers for the word rule (the Person filter of the search) and dominant_turn_speaker for the segment
// rule (the search under it and the deduced speaker of a segment correction). Callers pass turns in ordinal order, as
// SQL reads them, since that order decides ties.
package segmentparts

import "math"

// Rule is a version of the attribution rule. A new behaviour is a new version; a version never changes.
type Rule int

const (
	// RuleSegment gives the whole segment to the voice that speaks most over it.
	RuleSegment Rule = 1
	// RuleWord gives each word to the voice whose turns cover it most.
	RuleWord Rule = 2
)

// Span is a time interval, in seconds.
type Span struct{ Start, End float64 }

// Word is the time of one word of the segment. A word without time has End <= Start.
type Word = Span

// Turn is a speaker turn of the track of the segment. S identifies a voice (a Local speaker id, a label).
type Turn[S comparable] struct {
	Speaker    S
	Start, End float64
}

// Part is a run of consecutive words of a segment given to one voice: words [First, First+Count).
// Known is false when no voice speaks over the segment; Speaker is then the zero value.
type Part[S comparable] struct {
	Speaker S
	Known   bool
	First   int
	Count   int
}

// Attribute cuts a segment into parts. It always returns at least one part, and the parts cover every word in order.
//
// RuleSegment: one part, for the voice whose turns share the most time with the segment (ties to the first voice in
// turns), or an unknown part when no turn overlaps the segment.
//
// RuleWord: each word takes the voice whose turns share the most time with it; on a tie it keeps the voice of the
// previous word (else the first voice in turns). A word no turn covers takes the voice of the previous covered word,
// else of the next one. A voice holding a single word between two words of one other voice is given to that voice.
// When no word is covered (or the segment has no timed word), the segment rule applies.
//
// turns must be those of the track of the segment only; their order decides ties.
func Attribute[S comparable](rule Rule, segment Span, words []Word, turns []Turn[S]) []Part[S] {
	if rule == RuleWord {
		if parts := byWord(words, turns); parts != nil {
			return parts
		}
	}
	speaker, known := dominant(segment, turns, nil)
	return []Part[S]{{Speaker: speaker, Known: known, First: 0, Count: len(words)}}
}

// Segment is a segment of a track with the times of its words.
type Segment struct {
	Span  Span
	Words []Word
}

// Spans gives each voice the times of the words of the segments it is given under rule, in segment order. A segment
// without word times is speech of the voice it is given to, over the whole segment.
func Spans[S comparable](rule Rule, segments []Segment, turns []Turn[S]) map[S][]Span {
	spans := map[S][]Span{}
	for _, s := range segments {
		for _, p := range Attribute(rule, s.Span, s.Words, turns) {
			switch {
			case !p.Known:
			case len(s.Words) == 0:
				spans[p.Speaker] = append(spans[p.Speaker], s.Span)
			default:
				spans[p.Speaker] = append(spans[p.Speaker], s.Words[p.First:p.First+p.Count]...)
			}
		}
	}
	return spans
}

// Silent lists, in the order of turns, the voices given no word of the segments under rule (Spans): the Local
// speakers "without words" of a track (#403).
func Silent[S comparable](rule Rule, segments []Segment, turns []Turn[S]) []S {
	spans := Spans(rule, segments, turns)
	seen := map[S]bool{}
	out := []S{}
	for _, t := range turns {
		if len(spans[t.Speaker]) == 0 && !seen[t.Speaker] {
			seen[t.Speaker] = true // listed once
			out = append(out, t.Speaker)
		}
	}
	return out
}

// dominant is the voice sharing the most time with span. Among equals, previous wins when given, else the first
// voice in turns.
func dominant[S comparable](span Span, turns []Turn[S], previous *S) (winner S, found bool) {
	overlap := map[S]float64{}
	order := []S{}
	for _, t := range turns {
		if shared := math.Min(span.End, t.End) - math.Max(span.Start, t.Start); shared > 0 {
			if _, seen := overlap[t.Speaker]; !seen {
				order = append(order, t.Speaker)
			}
			overlap[t.Speaker] += shared
		}
	}
	for _, id := range order {
		if !found || overlap[id] > overlap[winner] {
			winner, found = id, true
		}
	}
	if found && previous != nil && overlap[*previous] == overlap[winner] {
		winner = *previous
	}
	return winner, found
}

// byWord applies the word rule; nil when no word is covered by a turn.
func byWord[S comparable](words []Word, turns []Turn[S]) []Part[S] {
	speakers := make([]S, len(words))
	covered := make([]bool, len(words))
	var previous *S
	for i, w := range words {
		if speaker, found := dominant(w, turns, previous); found {
			speakers[i], covered[i] = speaker, true
			previous = &speakers[i]
		}
	}
	if previous == nil {
		return nil
	}
	// A word without turn takes its neighbour: the previous covered word, else the next one.
	var neighbour S
	seen := false
	for i := range words {
		if covered[i] {
			neighbour, seen = speakers[i], true
		} else if seen {
			speakers[i], covered[i] = neighbour, true
		}
	}
	for i := len(words) - 1; i >= 0; i-- {
		if covered[i] {
			neighbour = speakers[i]
		} else {
			speakers[i] = neighbour
		}
	}
	// Word times are approximate: a single word between two words of one other voice is no change of voice.
	for i := 1; i+1 < len(words); i++ {
		if speakers[i-1] == speakers[i+1] && speakers[i] != speakers[i-1] {
			speakers[i] = speakers[i-1]
		}
	}
	parts := []Part[S]{}
	for i, speaker := range speakers {
		if n := len(parts); n > 0 && parts[n-1].Speaker == speaker {
			parts[n-1].Count++
			continue
		}
		parts = append(parts, Part[S]{Speaker: speaker, Known: true, First: i, Count: 1})
	}
	return parts
}
