// Package audioplans owns immutable analysis inputs and transcript provenance.
// It deliberately does not schedule work or publish multitrack results.
package audioplans

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"snitch/internal/httpapi/contract"
)

type Input = contract.AudioPlanInput
type Plan = contract.AudioPlan
type State = contract.AudioPlanState
type Diagnostic = contract.AudioTrackDiagnostic
type Child = contract.AudioPlanChild
type Revision = contract.TranscriptRevision

type Canonical struct {
	Input   Input
	JSON    []byte
	Version string
}

// CanonicalInput takes ownership of a deep copy. Selection order and duplicate
// indices do not change identity; any evidence, decision or version change does.
func CanonicalInput(input Input) (Canonical, error) {
	b, err := json.Marshal(input)
	if err != nil {
		return Canonical{}, err
	}
	var v Input
	if err = json.Unmarshal(b, &v); err != nil {
		return Canonical{}, err
	}
	if v.Mode != "auto" && v.Mode != "manual" {
		return Canonical{}, errors.New("invalid mode")
	}
	for _, s := range []string{v.DetectorVersion, v.DetectorConfigVersion, v.MergeVersion, v.PipelineVersion, v.VocabularyVersion} {
		if strings.TrimSpace(s) == "" {
			return Canonical{}, errors.New("all input versions are required")
		}
	}
	inventory := map[int]bool{}
	for i := range v.Inventory {
		t := &v.Inventory[i]
		if t.Evidence == nil {
			t.Evidence = []string{}
		}
		if t.Index < 0 || inventory[t.Index] || t.Channels < 0 || !finite(float64(t.StartSeconds)) || (t.DurationSeconds != nil && (!finite(float64(*t.DurationSeconds)) || *t.DurationSeconds < 0)) {
			return Canonical{}, errors.New("invalid or duplicate inventory track")
		}
		inventory[t.Index] = true
	}
	sort.Slice(v.Inventory, func(i, j int) bool { return v.Inventory[i].Index < v.Inventory[j].Index })
	sort.Ints(v.TrackIndices)
	selected := map[int]bool{}
	indices := []int{}
	for _, i := range v.TrackIndices {
		if !inventory[i] {
			return Canonical{}, fmt.Errorf("unknown track %d", i)
		}
		if !selected[i] {
			indices = append(indices, i)
			selected[i] = true
		}
	}
	if len(indices) == 0 {
		return Canonical{}, errors.New("empty selection")
	}
	v.TrackIndices = indices
	decisions := map[int]bool{}
	for _, d := range v.Decisions {
		if !inventory[d.TrackIndex] || decisions[d.TrackIndex] || d.ActorId <= 0 || d.DecidedAt.IsZero() || strings.TrimSpace(d.Reason) == "" || (d.Action != "include" && d.Action != "exclude") {
			return Canonical{}, errors.New("invalid track decision")
		}
		if (d.Action == "include") != selected[d.TrackIndex] {
			return Canonical{}, errors.New("decision contradicts selection")
		}
		decisions[d.TrackIndex] = true
	}
	if v.Decisions == nil {
		v.Decisions = []contract.AudioTrackDecision{}
	}
	sort.Slice(v.Decisions, func(i, j int) bool { return v.Decisions[i].TrackIndex < v.Decisions[j].TrackIndex })
	b, err = json.Marshal(v)
	if err != nil {
		return Canonical{}, err
	}
	sum := sha256.Sum256(b)
	return Canonical{v, b, hex.EncodeToString(sum[:])}, nil
}
func finite(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }

//go:embed fixtures/audio-plan-input.json
var contractFixture []byte

// ContractFixture is synthetic and shared by diagnostic, orchestration, publication
// and UI consumers. Indices intentionally differ from positions in the array.
func ContractFixture() []byte { return append([]byte(nil), contractFixture...) }
