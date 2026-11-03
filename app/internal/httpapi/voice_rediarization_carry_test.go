package httpapi

import (
	"fmt"
	"testing"
)

// carryTarget: an old voice goes to the new voice that covers at least half of its speaking
// time, never to one of two new voices covering it equally (ADR 0031).
func TestCarryTargetFollowsTheLargestCoverage(t *testing.T) {
	old := [][2]float64{{0, 1}, {2, 3}} // two seconds of speech
	for _, tc := range []struct {
		name   string
		news   [][][2]float64
		target int
		share  float64
		tied   bool
	}{
		{"covered entirely", [][][2]float64{{{0, 3}}}, 0, 1, false},
		{"the larger of two", [][][2]float64{{{0, 0.5}}, {{0.5, 3}}}, 1, 0.75, false},
		{"exactly half", [][][2]float64{{{0, 1}}, {{2, 2.5}}}, 0, 0.5, false},
		{"below half", [][][2]float64{{{0, 0.8}}, {{0.8, 1}, {2, 2.2}}, {{2.5, 3}}}, -1, 0.4, false},
		{"two equal covers", [][][2]float64{{{0, 1}}, {{2, 3}}}, -1, 0.5, true},
		{"no new voice", nil, -1, 0, false},
		{"turns spread over one new voice", [][][2]float64{{{0.5, 1}}, {{0, 0.5}, {2, 3}}}, 1, 0.75, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target, share, tied := carryTarget(old, tc.news)
			if target != tc.target || share != tc.share || tied != tc.tied {
				t.Fatalf("got target %d share %v tied %v", target, share, tied)
			}
		})
	}
	if target, share, _ := carryTarget(nil, [][][2]float64{{{0, 1}}}); target != -1 || share != 0 {
		t.Fatalf("a voice without speech goes nowhere: %d %v", target, share)
	}
}

// carryConflicts: old voices that land on the same new voice conflict when their
// dispositions differ, the absence of a disposition included.
func TestCarryConflictsCompareDispositionsPerNewVoice(t *testing.T) {
	voices := []oldVoice{
		{Target: 0, Disposition: dispositionIgnored},
		{Target: 0, Disposition: dispositionIgnored},
		{Target: 1, Disposition: dispositionIgnored},
		{Target: 1, Disposition: ""},
		{Target: 2, Disposition: dispositionRandom},
		{Target: -1, Disposition: dispositionIgnored},
	}
	got := carryConflicts(voices)
	if fmt.Sprint(got) != "map[0:false 1:true 2:false]" {
		t.Fatalf("got %v", got)
	}
}
