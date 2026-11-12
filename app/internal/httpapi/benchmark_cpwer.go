package httpapi

import (
	"sort"
	"strings"

	"snitch/internal/segmentparts"
)

// Speaker-attributed word error rate of the end-to-end benchmark (#287, ADR 0060), the cpWER of the CHiME
// campaigns: the words of each speaker are concatenated, the speakers of the model are matched to those of the
// reference by the assignment with the fewest word errors, and the errors are divided by the reference words. A
// right text given to the wrong speaker, a speaker split in two or two merged all count as errors.

// timedText is a transcript segment, with its words when the model timed them.
type timedText struct {
	start, end float64
	text       string
	words      []timedWord
}

type timedWord struct {
	start, end float64
	text       string
}

type speakerText struct {
	start, end float64
	speaker    string
	text       string
}

type cpwerMetrics struct {
	Clips             int      `json:"clips"`
	ReferenceWords    int      `json:"reference_words"`
	CPWordErrors      int      `json:"cp_word_errors"`
	WordErrors        int      `json:"word_errors"`
	CPWER             *float64 `json:"cpwer"`
	WER               *float64 `json:"wer"`
	ReferenceSpeakers int      `json:"reference_speakers"`
	ModelSpeakers     int      `json:"model_speakers"`
}

// attributeSegments gives the text of each segment to the voices production gives it under rule (ADR 0065), through
// the one attribution function: the whole segment to one voice, or each run of its words to its voice when the
// rule splits it. "" when no voice speaks over the segment.
func attributeSegments(rule segmentparts.Rule, segments []timedText, turns []speakerTurn) []speakerText {
	ruleTurns := make([]segmentparts.Turn[string], len(turns))
	for i, t := range turns {
		ruleTurns[i] = segmentparts.Turn[string]{Speaker: t.speaker, Start: t.start, End: t.end}
	}
	out := make([]speakerText, 0, len(segments))
	for _, seg := range segments {
		times := make([]segmentparts.Word, len(seg.words))
		for i, w := range seg.words {
			times[i] = segmentparts.Word{Start: w.start, End: w.end}
		}
		parts := segmentparts.Attribute(rule, segmentparts.Span{Start: seg.start, End: seg.end}, times, ruleTurns)
		if len(parts) == 1 {
			out = append(out, speakerText{seg.start, seg.end, parts[0].Speaker, seg.text})
			continue
		}
		for _, p := range parts {
			run := seg.words[p.First : p.First+p.Count]
			texts := make([]string, len(run))
			for i, w := range run {
				texts[i] = w.text
			}
			out = append(out, speakerText{run[0].start, run[len(run)-1].end, p.Speaker, strings.Join(texts, " ")})
		}
	}
	return out
}

// speakerWords concatenates, per speaker and in time order, the normalised words of the texts.
func speakerWords(texts []speakerText) map[string][]string {
	sorted := append([]speakerText(nil), texts...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].start < sorted[j].start })
	out := map[string][]string{}
	for _, t := range sorted {
		out[t.speaker] = append(out[t.speaker], normalizeWords(t.text)...)
	}
	return out
}

// cpErrors returns the fewest word errors over the matchings of reference and model speakers.
func cpErrors(reference, hypothesis map[string][]string) int {
	refNames, hypNames := sortedKeys(reference), sortedKeys(hypothesis)
	n := max(len(refNames), len(hypNames))
	if n == 0 {
		return 0
	}
	// Square cost matrix: a missing speaker on either side costs all the words of the other one.
	weights := make([][]int, n)
	for i := range weights {
		weights[i] = make([]int, n)
		for j := range weights[i] {
			var cost int
			switch {
			case i < len(refNames) && j < len(hypNames):
				cost = editDistance(reference[refNames[i]], hypothesis[hypNames[j]])
			case i < len(refNames):
				cost = len(reference[refNames[i]])
			case j < len(hypNames):
				cost = len(hypothesis[hypNames[j]])
			}
			weights[i][j] = -cost // assignSpeakers maximises
		}
	}
	total := 0
	for i, j := range assignSpeakers(weights) {
		if j >= 0 {
			total -= weights[i][j]
		}
	}
	return total
}

func sortedKeys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type attributedClip struct{ reference, hypothesis []speakerText }

// evaluateCPWER sums the errors over the clips before dividing, as the other word rates do.
func evaluateCPWER(clips []attributedClip) cpwerMetrics {
	m := cpwerMetrics{}
	for _, c := range clips {
		ref, hyp := speakerWords(c.reference), speakerWords(c.hypothesis)
		m.Clips++
		m.ReferenceSpeakers += len(ref)
		m.ModelSpeakers += len(hyp)
		refAll, hypAll := []string{}, []string{}
		for _, w := range ref {
			m.ReferenceWords += len(w)
		}
		for _, t := range sortedTexts(c.reference) {
			refAll = append(refAll, normalizeWords(t.text)...)
		}
		for _, t := range sortedTexts(c.hypothesis) {
			hypAll = append(hypAll, normalizeWords(t.text)...)
		}
		m.CPWordErrors += cpErrors(ref, hyp)
		m.WordErrors += editDistance(refAll, hypAll)
	}
	if m.ReferenceWords > 0 {
		cp := float64(m.CPWordErrors) / float64(m.ReferenceWords)
		wer := float64(m.WordErrors) / float64(m.ReferenceWords)
		m.CPWER, m.WER = &cp, &wer
	}
	return m
}

func sortedTexts(texts []speakerText) []speakerText {
	out := append([]speakerText(nil), texts...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].start < out[j].start })
	return out
}
