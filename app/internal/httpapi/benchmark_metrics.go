package httpapi

import (
	"math"
	"sort"
)

// Metrics of the voice identification benchmark (#287). They are computed on one model at a time, from the
// vectors of that model only: cosine scores of two models are never merged or compared, each model carries
// its own operating threshold (the one of its equal error rate).
//
// The benchmark scores like production (#397, ADR 0064): its References are built and its voices scored by the
// one score of Snitch (voice_score.go), under the score mode recorded with the run, from fingerprints decided
// by the eligibility rule recorded with the run. A threshold read in a run therefore holds for production.

// benchSample is the benchmark fingerprint of a dataset voice, as the eligibility rule of the run decided it.
type benchSample struct {
	Person       int64
	Clip         int64
	Vector       []float32
	NetSpeech    float64
	Comparable   bool
	Contributive bool
}

type fingerprintMetrics struct {
	References       int      `json:"references"`
	Tests            int      `json:"tests"`
	Persons          int      `json:"persons"`
	GenuineTrials    int      `json:"genuine_trials"`
	ImpostorTrials   int      `json:"impostor_trials"`
	EER              *float64 `json:"eer"`
	EERThreshold     *float64 `json:"eer_threshold"`
	Top1Accuracy     *float64 `json:"top1_accuracy"`
	TruePositives    int      `json:"true_positives"`
	FalsePositives   int      `json:"false_positives"`
	FalseNegatives   int      `json:"false_negatives"`
	Precision        *float64 `json:"precision"`
	Recall           *float64 `json:"recall"`
	GenuineMean      *float64 `json:"genuine_mean_score"`
	ImpostorMean     *float64 `json:"impostor_mean_score"`
	IgnoredDimension int      `json:"ignored_dimension"`
	// Ineligible counts the fingerprints the rule leaves out, as production would: a reference that cannot enter
	// a Reference under the mode, a test that is not comparable.
	Ineligible int `json:"ineligible"`
}

type benchTrial struct {
	score   float64
	genuine bool
}

// evaluateFingerprints scores every test against the Reference of every Person that has one, as production
// matching does (voiceReferences). A trial is genuine when the Reference is the one of the test's Person.
// Samples whose dimension differs from dims are ignored and counted, never compared; samples the eligibility
// rule leaves out are counted as ineligible.
func evaluateFingerprints(mode scoreMode, references, tests []benchSample, dims int) fingerprintMetrics {
	m := fingerprintMetrics{}
	var samples []referenceSample
	for _, s := range references {
		switch {
		case len(s.Vector) != dims:
			m.IgnoredDimension++
		case !s.Contributive || (mode == scoreBestSample && !s.Comparable):
			m.Ineligible++
		default:
			samples = append(samples, referenceSample{Person: s.Person, Clip: s.Clip, Vector: s.Vector, NetSpeech: s.NetSpeech, Comparable: s.Comparable})
		}
	}
	var tsts []benchSample
	for _, s := range tests {
		switch {
		case len(s.Vector) != dims:
			m.IgnoredDimension++
		case !s.Comparable:
			m.Ineligible++ // production never matches a voice that is not comparable
		default:
			tsts = append(tsts, s)
		}
	}
	m.References, m.Tests = len(samples), len(tsts)
	refs := newVoiceReferences(mode, dims, samples)
	persons := refs.persons()
	m.Persons = len(persons)

	type decision struct {
		best  int64
		score float64
		truth int64
	}
	var trials []benchTrial
	var decisions []decision
	var genuineSum, impostorSum float64
	for _, t := range tsts {
		if _, ok := refs.score(t.Vector, t.Person); !ok {
			continue // a test whose Person has no Reference, or a null vector, cannot be scored
		}
		d := decision{truth: t.Person, score: math.Inf(-1)}
		for _, p := range persons {
			score, _ := refs.score(t.Vector, p)
			genuine := p == t.Person
			trials = append(trials, benchTrial{score, genuine})
			if genuine {
				m.GenuineTrials++
				genuineSum += score
			} else {
				m.ImpostorTrials++
				impostorSum += score
			}
			if score > d.score {
				d.best, d.score = p, score
			}
		}
		decisions = append(decisions, d)
	}
	if m.GenuineTrials > 0 {
		v := genuineSum / float64(m.GenuineTrials)
		m.GenuineMean = &v
	}
	if m.ImpostorTrials > 0 {
		v := impostorSum / float64(m.ImpostorTrials)
		m.ImpostorMean = &v
	}
	if len(decisions) == 0 {
		return m
	}
	correct := 0
	for _, d := range decisions {
		if d.best == d.truth {
			correct++
		}
	}
	top1 := float64(correct) / float64(len(decisions))
	m.Top1Accuracy = &top1
	if m.GenuineTrials == 0 || m.ImpostorTrials == 0 {
		return m
	}
	eer, threshold := equalErrorRate(trials)
	m.EER, m.EERThreshold = &eer, &threshold

	// Open-set decision at the model's own EER threshold: the best Person is accepted when its score reaches
	// the threshold. A wrong acceptance is a false positive; a rejected or wrongly named test is a false negative.
	for _, d := range decisions {
		accepted := d.score >= threshold
		switch {
		case accepted && d.best == d.truth:
			m.TruePositives++
		case accepted:
			m.FalsePositives++
			m.FalseNegatives++
		default:
			m.FalseNegatives++
		}
	}
	if accepted := m.TruePositives + m.FalsePositives; accepted > 0 {
		v := float64(m.TruePositives) / float64(accepted)
		m.Precision = &v
	}
	recall := float64(m.TruePositives) / float64(len(decisions))
	m.Recall = &recall
	return m
}

// equalErrorRate sweeps the threshold over the observed scores and returns the point where the false
// acceptance and false rejection rates are the closest, with their mean as the rate.
func equalErrorRate(trials []benchTrial) (float64, float64) {
	sorted := append([]benchTrial(nil), trials...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].score < sorted[j].score })
	var genuine, impostor int
	for _, t := range sorted {
		if t.genuine {
			genuine++
		} else {
			impostor++
		}
	}
	// threshold t accepts score >= t. Start below everything: nothing rejected, everything accepted.
	bestGap, bestEER, bestThreshold := math.Inf(1), 1.0, sorted[0].score
	rejectedGenuine, rejectedImpostor := 0, 0
	for i := 0; i < len(sorted); {
		threshold := sorted[i].score
		frr := float64(rejectedGenuine) / float64(genuine)
		far := float64(impostor-rejectedImpostor) / float64(impostor)
		if gap := math.Abs(frr - far); gap < bestGap {
			bestGap, bestEER, bestThreshold = gap, (frr+far)/2, threshold
		}
		for i < len(sorted) && sorted[i].score == threshold {
			if sorted[i].genuine {
				rejectedGenuine++
			} else {
				rejectedImpostor++
			}
			i++
		}
	}
	return bestEER, bestThreshold
}
