package httpapi

import (
	"math"
	"sort"
)

// Score of a voice against the References of the Persons (ADR 0064, #391). It is the one score of Snitch:
// production matching, the measured accuracy and the fingerprint benchmark all build their References and
// score their voices here, so a threshold read in one of them holds for the others.
//
// The References are always those of one fingerprint group (model, version, dimensions): the caller gives the
// samples of one group, and a vector of other dimensions is ignored, never compared.
//
// Two score modes, a versioned administrator setting (voice_score_modes):
//   - cumulative (default): the Reference of a Person is the mean of the unit vectors of its contributive
//     validated samples, weighted by their net speech capped per sample; the score is the cosine with it, so
//     several short extracts add up to what none of them is alone;
//   - best_sample (the phase 5 score): each comparable sample is a reference on its own and the score of a
//     Person is the best cosine with one of them. Samples that are only contributive are left out, since
//     this mode compares each sample alone.

type scoreMode string

const (
	scoreCumulative scoreMode = "cumulative"
	scoreBestSample scoreMode = "best_sample"
)

func (m scoreMode) valid() bool { return m == scoreCumulative || m == scoreBestSample }

// referenceSampleCapSeconds caps the weight of one sample in a cumulative Reference, so that a single long clip
// does not outweigh every other one. It matches the speech at which a Worker fingerprint reaches full quality.
const referenceSampleCapSeconds = 10.0

// referenceWeight is the weight of a sample in a cumulative Reference: its net speech, clamped to [0, cap].
func referenceWeight(netSpeech float64) float64 {
	if netSpeech <= 0 || math.IsNaN(netSpeech) {
		return 0
	}
	return math.Min(netSpeech, referenceSampleCapSeconds)
}

// referenceSample is a validated voice sample as a Reference reads it: the vector of its local speaker's
// fingerprint in one group, the net speech that vector was computed on, and whether it is comparable alone.
// Callers only pass contributive samples (validated by a human, never an automatic attribution).
type referenceSample struct {
	Person     int64
	Clip       int64
	Vector     []float32
	NetSpeech  float64
	Comparable bool
}

// voiceReferences holds the References of the Persons of one group under one mode, as unit vectors: one per
// Person in the cumulative mode, one per comparable sample in the best-sample mode.
type voiceReferences struct {
	mode    scoreMode
	dims    int
	order   []int64
	vectors map[int64][][]float64
}

func unitVector(v []float32) ([]float64, bool) {
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	if norm == 0 || math.IsNaN(norm) || math.IsInf(norm, 0) {
		return nil, false
	}
	norm = math.Sqrt(norm)
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = float64(x) / norm
	}
	return out, true
}

// newVoiceReferences builds the References of the given samples, which all belong to one group of dims
// dimensions.
func newVoiceReferences(mode scoreMode, dims int, samples []referenceSample) *voiceReferences {
	r := &voiceReferences{mode: mode, dims: dims, vectors: map[int64][][]float64{}}
	sums := map[int64][]float64{}
	for _, sample := range samples {
		if len(sample.Vector) != dims {
			continue
		}
		u, ok := unitVector(sample.Vector)
		if !ok {
			continue
		}
		if mode == scoreBestSample {
			if sample.Comparable {
				r.vectors[sample.Person] = append(r.vectors[sample.Person], u)
			}
			continue
		}
		w := referenceWeight(sample.NetSpeech)
		if w == 0 {
			continue
		}
		sum := sums[sample.Person]
		if sum == nil {
			sum = make([]float64, dims)
			sums[sample.Person] = sum
		}
		for i, x := range u {
			sum[i] += w * x
		}
	}
	for person, sum := range sums {
		// The weighted mean and its direction are the same up to a positive factor: only the direction scores.
		v := make([]float32, dims)
		for i, x := range sum {
			v[i] = float32(x)
		}
		if u, ok := unitVector(v); ok {
			r.vectors[person] = [][]float64{u}
		}
	}
	for person := range r.vectors {
		r.order = append(r.order, person)
	}
	sort.Slice(r.order, func(i, j int) bool { return r.order[i] < r.order[j] })
	return r
}

// persons lists the Persons that have a Reference, in id order.
func (r *voiceReferences) persons() []int64 { return r.order }

func (r *voiceReferences) empty() bool { return len(r.order) == 0 }

// targetUnit normalises a voice to score; false when it cannot be compared (other dimensions, null vector).
func (r *voiceReferences) targetUnit(target []float32) ([]float64, bool) {
	if len(target) != r.dims {
		return nil, false
	}
	return unitVector(target)
}

func (r *voiceReferences) scoreUnit(t []float64, person int64) (float64, bool) {
	refs, ok := r.vectors[person]
	if !ok {
		return 0, false
	}
	best := math.Inf(-1)
	for _, ref := range refs {
		var dot float64
		for i, x := range ref {
			dot += x * t[i]
		}
		if dot > best {
			best = dot
		}
	}
	return best, true
}

// score is the cosine similarity of a voice with the Reference of a Person; false when the Person has no
// Reference in this group or the voice cannot be compared.
func (r *voiceReferences) score(target []float32, person int64) (float64, bool) {
	t, ok := r.targetUnit(target)
	if !ok {
		return 0, false
	}
	return r.scoreUnit(t, person)
}

// rank scores a voice against every Person kept by keep (nil keeps all), and returns those at or above min,
// best first, ties in Person id order.
func (r *voiceReferences) rank(target []float32, keep func(person int64) bool, min float64) []matchScore {
	t, ok := r.targetUnit(target)
	if !ok {
		return nil
	}
	var out []matchScore
	for _, person := range r.order {
		if keep != nil && !keep(person) {
			continue
		}
		if s, _ := r.scoreUnit(t, person); s >= min {
			out = append(out, matchScore{personID: person, score: s})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out
}

// referenceFigure is what the cumulative Reference of a Person actually uses: its contributive samples, their
// distinct clips and their net speech after the per-sample cap.
type referenceFigure struct {
	Samples          int     `json:"samples"`
	Clips            int     `json:"clips"`
	NetSpeechSeconds float64 `json:"net_speech_seconds"`
}

// referenceFigures sums, per Person, the contributive samples of one group (loadReferenceSamples).
func referenceFigures(samples []referenceSample) map[int64]referenceFigure {
	out := map[int64]referenceFigure{}
	clips := map[int64]map[int64]bool{}
	for _, sample := range samples {
		f := out[sample.Person]
		f.Samples++
		f.NetSpeechSeconds += referenceWeight(sample.NetSpeech)
		if clips[sample.Person] == nil {
			clips[sample.Person] = map[int64]bool{}
		}
		clips[sample.Person][sample.Clip] = true
		f.Clips = len(clips[sample.Person])
		out[sample.Person] = f
	}
	return out
}
