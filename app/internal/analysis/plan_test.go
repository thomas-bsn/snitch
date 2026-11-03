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
			plan, err := Resolve(tc.scope, Options{Mode: ModeSteps, Steps: tc.steps}, Existing{Transcription: tc.hasTranscription, Voices: true})
			if err != nil {
				t.Fatal(err)
			}
			if plan.Scope != tc.wantScope || fmt.Sprint(plan.Forced) != tc.wantForced || fmt.Sprint(plan.Added) != tc.wantAdded {
				t.Fatalf("got scope %s, forced %v, added %v", plan.Scope, plan.Forced, plan.Added)
			}
		})
	}
}

// A scope widened to the full analysis also runs the voice steps that never ran: they are
// listed as added too, so the launch shows everything that will run.
func TestResolveListsNeverRunStepsOfAWidenedScope(t *testing.T) {
	for _, tc := range []struct {
		name      string
		scope     string
		steps     []string
		existing  Existing
		wantScope string
		wantAdded string
	}{
		{"retranscription widens to the full scope", ScopeTranscription, []string{StepTranscription}, Existing{Transcription: true}, ScopeFull,
			"[{indexing dependent transcription} {diarization missing } {fingerprints missing } {matching missing }]"},
		{"indexing widens to the full scope", ScopeTranscription, []string{StepIndexing}, Existing{Transcription: true}, ScopeFull,
			"[{diarization missing } {fingerprints missing } {matching missing }]"},
		{"voices already detected run only if missing", ScopeTranscription, []string{StepTranscription}, Existing{Transcription: true, Voices: true}, ScopeFull,
			"[{indexing dependent transcription}]"},
		{"a chosen voice detection is not missing", ScopeTranscription, []string{StepDiarization}, Existing{Transcription: true}, ScopeFull,
			"[{fingerprints dependent diarization} {matching dependent fingerprints}]"},
		{"the requested full scope adds nothing it already runs", ScopeFull, []string{StepTranscription}, Existing{Transcription: true}, ScopeFull,
			"[{indexing dependent transcription}]"},
		{"a voice scope widened to full", ScopeVoice, []string{StepTranscription}, Existing{Transcription: true}, ScopeFull,
			"[{indexing dependent transcription}]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := Resolve(tc.scope, Options{Mode: ModeSteps, Steps: tc.steps}, tc.existing)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Scope != tc.wantScope || fmt.Sprint(plan.Added) != tc.wantAdded {
				t.Fatalf("got scope %s, added %v", plan.Scope, plan.Added)
			}
		})
	}
}

// The shortcuts keep their meaning: nothing is added, the scope stays the chosen one.
func TestResolveKeepsTheShortcuts(t *testing.T) {
	missing, err := Resolve(ScopeFull, Options{Mode: ModeMissing}, Existing{})
	if err != nil || missing.Scope != ScopeFull || len(missing.Forced) != 0 || len(missing.Added) != 0 {
		t.Fatalf("missing: %+v %v", missing, err)
	}
	full, err := Resolve(ScopeFull, Options{Mode: ModeFull}, Existing{Transcription: true, Voices: true})
	if err != nil || full.Scope != ScopeFull || fmt.Sprint(full.Forced) != "[transcription diarization]" || len(full.Added) != 0 {
		t.Fatalf("full: %+v %v", full, err)
	}
	voice, err := Resolve(ScopeVoice, Options{Mode: ModeFull}, Existing{Transcription: true, Voices: true})
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
		if _, err := Resolve(ScopeFull, opts, Existing{Transcription: true, Voices: true}); !errors.Is(err, ErrInvalidReanalysis) {
			t.Fatalf("%+v: got %v", opts, err)
		}
	}
}
