package httpapi

import "sort"

// The level of a Reference (#396): how much the cumulative Reference of a Person holds compared with those of
// the other Persons of the active fingerprint group. One rule for the directory card, the record and its
// weak-Reference warning, decided here so the web never recomputes it.
//
// The figure compared is the net speech the Reference uses (contributive validated samples, capped per sample,
// see referenceFigures), never the number of validations or of attributions.

type referenceLevel string

const (
	referenceGood referenceLevel = "good"
	referenceWeak referenceLevel = "weak"
	referenceNone referenceLevel = "none"
)

// referenceBasis says why a Reference has its level, for the tooltip of the badge.
type referenceBasis string

const (
	referenceNoSample      referenceBasis = "no_sample"      // no contributive sample
	referenceSingleClip    referenceBasis = "single_clip"    // contributive samples on one clip only
	referenceFewReferences referenceBasis = "few_references" // several clips, too few References to compare
	referenceBelowMedian   referenceBasis = "below_median"   // less net speech than the median Reference
	referenceMedianOrMore  referenceBasis = "median_or_more" // at least the net speech of the median Reference
)

// referenceComparisonMinPersons is the number of Persons with a Reference from which a Reference is compared
// with the median; under it, several distinct clips are enough for a good Reference.
const referenceComparisonMinPersons = 3

// referenceStanding is the median net speech of the References of one group, when there are enough of them.
type referenceStanding struct {
	compared bool
	median   float64
}

// newReferenceStanding reads the figures of every Person of the group (all of them, whoever reads: the level is
// the same for every reader).
func newReferenceStanding(figures map[int64]referenceFigure) referenceStanding {
	var speech []float64
	for _, f := range figures {
		if f.Samples > 0 {
			speech = append(speech, f.NetSpeechSeconds)
		}
	}
	if len(speech) < referenceComparisonMinPersons {
		return referenceStanding{}
	}
	sort.Float64s(speech)
	n := len(speech)
	median := speech[n/2]
	if n%2 == 0 {
		median = (speech[n/2-1] + speech[n/2]) / 2
	}
	return referenceStanding{compared: true, median: median}
}

// judge gives the level of one Person's Reference. The absolute guards come first: no contributive sample is
// none, a single clip is weak, whatever the ranking.
func (s referenceStanding) judge(f referenceFigure) (referenceLevel, referenceBasis) {
	switch {
	case f.Samples == 0:
		return referenceNone, referenceNoSample
	case f.Clips < 2:
		return referenceWeak, referenceSingleClip
	case !s.compared:
		return referenceGood, referenceFewReferences
	case f.NetSpeechSeconds >= s.median:
		return referenceGood, referenceMedianOrMore
	default:
		return referenceWeak, referenceBelowMedian
	}
}
