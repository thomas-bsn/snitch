package analysis

import (
	"errors"
	"fmt"
	"testing"
)

// The chosen steps are closed over the step graph: a missing prerequisite and the steps that
// depend on a chosen one join the run, visibly, and the scope widens to hold them.
func TestResolveClosesChosenSteps(t *testing.T) {
	for _, tc := range []struct {
		name             string
		scope            string
		steps            []string
		hasTranscription bool
		wantScope        string
		wantForced       string
		wantAdded        string
	}{
		{"voice on a transcribed clip", ScopeVoice, []string{StepDiarization}, true, ScopeVoice, "[diarization]",
			"[{fingerprints dependent diarization} {matching dependent fingerprints}]"},
		{"voice without transcription", ScopeTranscription, []string{StepDiarization}, false, ScopeFull, "[diarization]",
			"[{transcription prerequisite diarization} {indexing dependent transcription} {fingerprints dependent diarization} {matching dependent fingerprints}]"},
		{"voice from the transcription scope", ScopeTranscription, []string{StepDiarization}, true, ScopeFull, "[diarization]",
			"[{fingerprints dependent diarization} {matching dependent fingerprints}]"},
		{"indexing without transcription", ScopeVoice, []string{StepIndexing}, false, ScopeFull, "[indexing]",
			"[{transcription prerequisite indexing}]"},
		{"indexing on a transcribed clip", ScopeTranscription, []string{StepIndexing}, true, ScopeFull, "[indexing]", "[]"},
		{"retranscription", ScopeTranscription, []string{StepTranscription}, true, ScopeFull, "[transcription]",
			"[{indexing dependent transcription}]"},
		{"retranscription from the voice scope", ScopeVoice, []string{StepTranscription}, true, ScopeFull, "[transcription]",
			"[{indexing dependent transcription}]"},
		{"everything chosen", ScopeFull, []string{StepDiarization, StepTranscription, StepIndexing}, true, ScopeFull, "[transcription indexing diarization]",
			"[{fingerprints dependent diarization} {matching dependent fingerprints}]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := Resolve(tc.scope, Options{Mode: ModeSteps, Steps: tc.steps}, tc.hasTranscription)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Scope != tc.wantScope || fmt.Sprint(plan.Forced) != tc.wantForced || fmt.Sprint(plan.Added) != tc.wantAdded {
				t.Fatalf("got scope %s, forced %v, added %v", plan.Scope, plan.Forced, plan.Added)
			}
		})
	}
}

// The shortcuts keep their meaning: nothing is added, the scope stays the chosen one.
func TestResolveKeepsTheShortcuts(t *testing.T) {
	missing, err := Resolve(ScopeFull, Options{Mode: ModeMissing}, false)
	if err != nil || missing.Scope != ScopeFull || len(missing.Forced) != 0 || len(missing.Added) != 0 {
		t.Fatalf("missing: %+v %v", missing, err)
	}
	full, err := Resolve(ScopeFull, Options{Mode: ModeFull}, true)
	if err != nil || full.Scope != ScopeFull || fmt.Sprint(full.Forced) != "[transcription diarization]" || len(full.Added) != 0 {
		t.Fatalf("full: %+v %v", full, err)
	}
	voice, err := Resolve(ScopeVoice, Options{Mode: ModeFull}, true)
	if err != nil || voice.Scope != ScopeVoice || fmt.Sprint(voice.Forced) != "[transcription diarization]" {
		t.Fatalf("full voice: %+v %v", voice, err)
	}
}

func TestResolveRefusesWhatIsNotAStep(t *testing.T) {
	for _, opts := range []Options{
		{Mode: "weird"},
		{Mode: ModeSteps},
		{Mode: ModeSteps, Steps: []string{StepMatching}},
		{Mode: ModeSteps, Steps: []string{StepDiarization, StepDiarization}},
		{Mode: ModeSteps, Steps: []string{StepDiarization, "weird"}},
	} {
		if _, err := Resolve(ScopeFull, opts, true); !errors.Is(err, ErrInvalidReanalysis) {
			t.Fatalf("%+v: got %v", opts, err)
		}
	}
}
