package httpapi

import "sort"

// Speaker-attributed word error rate of the end-to-end benchmark (#287, ADR 0060), the cpWER of the CHiME
// campaigns: the words of each speaker are concatenated, the speakers of the model are matched to those of the
// reference by the assignment with the fewest word errors, and the errors are divided by the reference words. A
// right text given to the wrong speaker, a speaker split in two or two merged all count as errors.

type timedText struct {
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

// attributeSegments gives each timed text the speaker whose turns overlap it the most; none, "" (unassigned).
func attributeSegments(segments []timedText, turns []speakerTurn) []speakerText {
	out := make([]speakerText, 0, len(segments))
	for _, seg := range segments {
		overlap := map[string]float64{}
		for _, t := range turns {
			if o := min(seg.end, t.end) - max(seg.start, t.start); o > 0 {
				overlap[t.speaker] += o
			}
		}
		best, most := "", 0.0
		names := make([]string, 0, len(overlap))
		for name := range overlap {
			names = append(names, name)
		}
		sort.Strings(names) // a tie goes to the first name: the result does not depend on map order
		for _, name := range names {
			if overlap[name] > most {
				best, most = name, overlap[name]
			}
		}
		out = append(out, speakerText{seg.start, seg.end, best, seg.text})
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
