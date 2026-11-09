package httpapi

import (
	"strings"
	"unicode"
)

// Word and character error rates of the transcription benchmark (#287, ADR 0058). Both texts are normalised the
// same way before comparison: lower case, every punctuation mark or symbol read as a space. Accents are kept, since
// a French transcript that drops them is wrong. The rates are micro-averaged: errors and reference sizes are summed
// over all the clips before dividing, so a long clip weighs more than a short one.

// maxCERCells bounds the character comparison of one clip: beyond it the clip is left out of the CER (and counted),
// rather than holding the request for minutes. The word comparison has no such limit.
const maxCERCells = 250_000_000

type transcriptionMetrics struct {
	Clips          int      `json:"clips"`
	ReferenceWords int      `json:"reference_words"`
	WordErrors     int      `json:"word_errors"`
	ReferenceChars int      `json:"reference_chars"`
	CharErrors     int      `json:"character_errors"`
	CERSkipped     int      `json:"cer_skipped_clips"`
	WER            *float64 `json:"wer"`
	CER            *float64 `json:"cer"`
}

// normalizeWords lower-cases a text and cuts it into words, punctuation and symbols acting as separators.
func normalizeWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r)
	})
}

// editDistance is the Levenshtein distance between two sequences, in two rows of memory.
func editDistance[T comparable](a, b []T) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}

type transcriptPair struct{ reference, hypothesis string }

// evaluateTranscripts measures a model against the frozen reference on the clips (and tracks) it transcribed.
func evaluateTranscripts(pairs []transcriptPair) transcriptionMetrics {
	m := transcriptionMetrics{}
	charCompared := 0
	for _, p := range pairs {
		ref, hyp := normalizeWords(p.reference), normalizeWords(p.hypothesis)
		m.Clips++
		m.ReferenceWords += len(ref)
		m.WordErrors += editDistance(ref, hyp)
		refChars, hypChars := []rune(strings.Join(ref, " ")), []rune(strings.Join(hyp, " "))
		if len(refChars)*len(hypChars) > maxCERCells {
			m.CERSkipped++
			continue
		}
		charCompared++
		m.ReferenceChars += len(refChars)
		m.CharErrors += editDistance(refChars, hypChars)
	}
	if m.ReferenceWords > 0 {
		v := float64(m.WordErrors) / float64(m.ReferenceWords)
		m.WER = &v
	}
	if m.ReferenceChars > 0 && charCompared > 0 {
		v := float64(m.CharErrors) / float64(m.ReferenceChars)
		m.CER = &v
	}
	return m
}
