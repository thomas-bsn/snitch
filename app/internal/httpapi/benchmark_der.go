package httpapi

import "math"

// Diarization error rate of the benchmark (#287, ADR 0059): the share of reference speech time that the model
// missed, attributed to the wrong speaker, or invented. The model's speakers are anonymous, so they are first
// matched to the reference speakers by the assignment that maximises their common speaking time (Hungarian
// algorithm). Boundaries are forgiven within a collar, as is usual, and overlapped speech counts one error per
// missing or extra speaker.

const (
	derResolution = 0.01      // seconds per frame
	derCollar     = 0.25      // seconds forgiven around each reference boundary
	derMaxFrames  = 3_600_000 // ten hours: beyond it a clip is left out rather than held in memory
)

type speakerTurn struct {
	start, end float64
	speaker    string
}

type diarizationMetrics struct {
	Clips           int      `json:"clips"`
	SkippedClips    int      `json:"skipped_clips"`
	ReferenceSpeech float64  `json:"reference_speech_seconds"`
	Missed          float64  `json:"missed_speech_seconds"`
	FalseAlarm      float64  `json:"false_alarm_seconds"`
	Confusion       float64  `json:"confusion_seconds"`
	CollarSeconds   float64  `json:"collar_seconds"`
	DER             *float64 `json:"der"`
	MissedRate      *float64 `json:"missed_speech_rate"`
	FalseAlarmRate  *float64 `json:"false_alarm_rate"`
	ConfusionRate   *float64 `json:"confusion_rate"`
}

// derComponents is the raw measure of one clip, in seconds.
type derComponents struct {
	total, missed, falseAlarm, confusion float64
	skipped                              bool
}

func speakerIndex(turns []speakerTurn) ([]string, map[string]int) {
	index := map[string]int{}
	var names []string
	for _, t := range turns {
		if _, ok := index[t.speaker]; !ok {
			index[t.speaker] = len(names)
			names = append(names, t.speaker)
		}
	}
	return names, index
}

// activity marks, per speaker, the frames where the speaker talks.
func activity(turns []speakerTurn, index map[string]int, count, frames int) [][]bool {
	rows := make([][]bool, count)
	for i := range rows {
		rows[i] = make([]bool, frames)
	}
	for _, t := range turns {
		from, to := int(math.Floor(t.start/derResolution)), int(math.Ceil(t.end/derResolution))
		for f := max(from, 0); f < min(to, frames); f++ {
			rows[index[t.speaker]][f] = true
		}
	}
	return rows
}

func diarizationErrors(reference, hypothesis []speakerTurn) derComponents {
	var end float64
	for _, t := range append(append([]speakerTurn{}, reference...), hypothesis...) {
		end = math.Max(end, t.end)
	}
	frames := int(math.Ceil(end/derResolution)) + 1
	if frames > derMaxFrames {
		return derComponents{skipped: true}
	}
	refNames, refIndex := speakerIndex(reference)
	hypNames, hypIndex := speakerIndex(hypothesis)
	ref := activity(reference, refIndex, len(refNames), frames)
	hyp := activity(hypothesis, hypIndex, len(hypNames), frames)

	scored := make([]bool, frames)
	for f := range scored {
		scored[f] = true
	}
	radius := int(math.Round(derCollar / derResolution))
	for _, t := range reference {
		for _, boundary := range []float64{t.start, t.end} {
			centre := int(math.Round(boundary / derResolution))
			for f := max(centre-radius, 0); f <= min(centre+radius, frames-1); f++ {
				scored[f] = false
			}
		}
	}

	// Common speaking time of every reference and hypothesis speaker, on the scored frames.
	overlap := make([][]int, len(refNames))
	for r := range overlap {
		overlap[r] = make([]int, len(hypNames))
		for h := range overlap[r] {
			for f := 0; f < frames; f++ {
				if scored[f] && ref[r][f] && hyp[h][f] {
					overlap[r][h]++
				}
			}
		}
	}
	mapping := assignSpeakers(overlap) // reference speaker -> hypothesis speaker, or -1

	var c derComponents
	for f := 0; f < frames; f++ {
		if !scored[f] {
			continue
		}
		nr, nh, matched := 0, 0, 0
		for r := range ref {
			if ref[r][f] {
				nr++
				if h := mapping[r]; h >= 0 && hyp[h][f] {
					matched++
				}
			}
		}
		for h := range hyp {
			if hyp[h][f] {
				nh++
			}
		}
		c.total += float64(nr)
		c.missed += float64(max(nr-nh, 0))
		c.falseAlarm += float64(max(nh-nr, 0))
		c.confusion += float64(min(nr, nh) - matched)
	}
	c.total *= derResolution
	c.missed *= derResolution
	c.falseAlarm *= derResolution
	c.confusion *= derResolution
	return c
}

// assignSpeakers returns, for each row, the column of the assignment that maximises the sum of the matrix.
func assignSpeakers(weights [][]int) []int {
	rows := len(weights)
	out := make([]int, rows)
	for i := range out {
		out[i] = -1
	}
	if rows == 0 {
		return out
	}
	cols := len(weights[0])
	n := max(rows, cols)
	if n == 0 {
		return out
	}
	// Hungarian algorithm on the cost -weight of the square matrix padded with zeros (O(n^3)).
	const inf = math.MaxInt64 / 4
	cost := func(i, j int) int64 {
		if i < rows && j < cols {
			return -int64(weights[i][j])
		}
		return 0
	}
	u, v := make([]int64, n+1), make([]int64, n+1)
	p, way := make([]int, n+1), make([]int, n+1)
	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		minv := make([]int64, n+1)
		used := make([]bool, n+1)
		for j := range minv {
			minv[j] = inf
		}
		for {
			used[j0] = true
			i0, delta, j1 := p[j0], int64(inf), 0
			for j := 1; j <= n; j++ {
				if used[j] {
					continue
				}
				cur := cost(i0-1, j-1) - u[i0] - v[j]
				if cur < minv[j] {
					minv[j], way[j] = cur, j0
				}
				if minv[j] < delta {
					delta, j1 = minv[j], j
				}
			}
			for j := 0; j <= n; j++ {
				if used[j] {
					u[p[j]] += delta
					v[j] -= delta
				} else {
					minv[j] -= delta
				}
			}
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for {
			j1 := way[j0]
			p[j0] = p[j1]
			j0 = j1
			if j0 == 0 {
				break
			}
		}
	}
	for j := 1; j <= n; j++ {
		if p[j] >= 1 && p[j]-1 < rows && j-1 < cols {
			out[p[j]-1] = j - 1
		}
	}
	return out
}

type clipTurns struct{ reference, hypothesis []speakerTurn }

// evaluateDiarization sums the errors over the clips before dividing: a long clip weighs more.
func evaluateDiarization(clips []clipTurns) diarizationMetrics {
	m := diarizationMetrics{CollarSeconds: derCollar}
	for _, c := range clips {
		e := diarizationErrors(c.reference, c.hypothesis)
		if e.skipped {
			m.SkippedClips++
			continue
		}
		m.Clips++
		m.ReferenceSpeech += e.total
		m.Missed += e.missed
		m.FalseAlarm += e.falseAlarm
		m.Confusion += e.confusion
	}
	if m.ReferenceSpeech > 0 {
		rate := func(x float64) *float64 { v := x / m.ReferenceSpeech; return &v }
		m.MissedRate, m.FalseAlarmRate, m.ConfusionRate = rate(m.Missed), rate(m.FalseAlarm), rate(m.Confusion)
		der := (m.Missed + m.FalseAlarm + m.Confusion) / m.ReferenceSpeech
		m.DER = &der
	}
	return m
}
