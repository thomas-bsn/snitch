package httpapi

import (
	"math"
	"sort"
)

// Metrics of the voice identification benchmark (#287). They are computed on one model at a time, from the
// vectors of that model only: cosine scores of two models are never merged or compared, each model carries
// its own operating threshold (the one of its equal error rate).

type benchSample struct {
	Person int64
	Vector []float32
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
}

type benchTrial struct {
	score   float64
	genuine bool
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// centroids builds the reference of each Person: the mean of its unit-normalised reference vectors.
func centroids(references []benchSample, dims int) map[int64][]float32 {
	sums := map[int64][]float64{}
	for _, r := range references {
		var norm float64
		for _, x := range r.Vector {
			norm += float64(x) * float64(x)
		}
		if norm == 0 {
			continue
		}
		norm = math.Sqrt(norm)
		sum, ok := sums[r.Person]
		if !ok {
			sum = make([]float64, dims)
			sums[r.Person] = sum
		}
		for i, x := range r.Vector {
			sum[i] += float64(x) / norm
		}
	}
	out := make(map[int64][]float32, len(sums))
	for person, sum := range sums {
		v := make([]float32, dims)
		for i, x := range sum {
			v[i] = float32(x)
		}
		out[person] = v
	}
	return out
}

// evaluateFingerprints scores every test against the centroid of every Person that has references. A trial is
// genuine when the centroid is the one of the test's Person. Samples whose dimension differs from the
// references' are ignored and counted, never compared.
func evaluateFingerprints(references, tests []benchSample, dims int) fingerprintMetrics {
	m := fingerprintMetrics{}
	var refs, tsts []benchSample
	for _, s := range references {
		if len(s.Vector) == dims {
			refs = append(refs, s)
		} else {
			m.IgnoredDimension++
		}
	}
	for _, s := range tests {
		if len(s.Vector) == dims {
			tsts = append(tsts, s)
		} else {
			m.IgnoredDimension++
		}
	}
	m.References, m.Tests = len(refs), len(tsts)
	cs := centroids(refs, dims)
	m.Persons = len(cs)
	persons := make([]int64, 0, len(cs))
	for p := range cs {
		persons = append(persons, p)
	}
	sort.Slice(persons, func(i, j int) bool { return persons[i] < persons[j] })

	type decision struct {
		best    int64
		score   float64
		truth   int64
		scoring bool
	}
	var trials []benchTrial
	var decisions []decision
	var genuineSum, impostorSum float64
	for _, t := range tsts {
		if _, ok := cs[t.Person]; !ok {
			continue // a test whose Person has no usable reference cannot be scored
		}
		d := decision{truth: t.Person, scoring: true, score: math.Inf(-1)}
		for _, p := range persons {
			score := cosine(t.Vector, cs[p])
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
