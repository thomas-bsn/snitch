//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type reanalysisRefusal struct {
	Error string `json:"error"`
	Stake struct {
		Corrections *struct {
			TextSegments int `json:"text_segments"`
		} `json:"corrections"`
		Voice *voiceRediarizationStake `json:"voice"`
	} `json:"stake"`
}

// testClipReanalysis covers the reanalysis of a processed clip: the choice of what runs again,
// the explicit confirmation of the consequences on human work, the protection of that work
// until success, and the versions kept in the history.
func testClipReanalysis(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, _ := h.diarizeClip()
	// Human work: a corrected segment text, a validated voice.
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/clips/%d/segments/0/text", h.clipID), map[string]any{"text": "corrigé", "expected_version": 0}, http.StatusOK)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", first), map[string]any{"new_person_name": "Alice"}, http.StatusCreated)
	jobs := func() int { return h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1`, h.clipID) }
	refuse := func(body map[string]any) reanalysisRefusal {
		t.Helper()
		var out reanalysisRefusal
		if err := json.Unmarshal(h.adminCall(http.MethodPost, h.analysisPath(), body, http.StatusConflict), &out); err != nil {
			t.Fatal(err)
		}
		if out.Error != "reanalysis_confirmation_required" {
			t.Fatalf("unexpected refusal: %+v", out)
		}
		return out
	}

	// Invalid choices are refused before anything happens.
	for _, body := range []map[string]any{
		{"scope": "full", "mode": "weird"},
		{"scope": "full", "mode": "steps"},
		{"scope": "full", "mode": "steps", "steps": []string{"matching"}},
		{"scope": "full", "mode": "steps", "steps": []string{"diarization", "diarization"}},
	} {
		h.adminCall(http.MethodPost, h.analysisPath(), body, http.StatusBadRequest)
	}

	// Only the missing steps: valid results are reused, so no confirmation is needed.
	before := jobs()
	reused := h.startAnalysis(map[string]any{"scope": "voice", "mode": "missing"}, http.StatusCreated)
	if reused.step("transcription").Reason != "reused_existing" || reused.step("diarization").Reason != "reused_existing" || jobs() != before {
		t.Fatalf("missing steps only must reuse valid results: %+v", reused)
	}
	if h.analysis().Status == "active" {
		h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusOK)
	}

	// Run chosen steps: only the voice stake is asked, and nothing starts without it.
	refusal := refuse(map[string]any{"scope": "voice", "mode": "steps", "steps": []string{"diarization"}})
	if refusal.Stake.Corrections != nil || refusal.Stake.Voice == nil || refusal.Stake.Voice.ValidatedSpeakers != 1 {
		t.Fatalf("unexpected voice stake: %+v", refusal)
	}
	if jobs() != before {
		t.Fatal("a refused reanalysis must plan nothing")
	}

	// Everything again: corrections and validations are both at stake; one confirmation is not enough.
	refusal = refuse(map[string]any{"scope": "full", "mode": "full"})
	if refusal.Stake.Corrections == nil || refusal.Stake.Corrections.TextSegments != 1 || refusal.Stake.Voice == nil {
		t.Fatalf("unexpected stake: %+v", refusal)
	}
	refusal = refuse(map[string]any{"scope": "full", "mode": "full", "confirm_replace_corrections": true})
	if refusal.Stake.Corrections != nil || refusal.Stake.Voice == nil {
		t.Fatalf("only the voice stake must remain: %+v", refusal)
	}
	member := h.newUser("reanalysis-member")
	h.userCall(member, http.MethodPost, h.analysisPath(), map[string]any{"scope": "full", "mode": "full"}, http.StatusForbidden)
	h.userCall(member, http.MethodGet, h.analysisPath()+"/history", nil, http.StatusForbidden)
	if jobs() != before || h.analysis() == nil || h.analysis().Status == "active" {
		t.Fatal("refusals must leave no active run")
	}

	// Confirmed: the transcription runs again while the current result and the corrections stay visible.
	run := h.startAnalysis(map[string]any{"scope": "full", "mode": "full", "confirm_replace_corrections": true, "confirm_replace_validations": true}, http.StatusCreated)
	if run.step("transcription").Reason == "reused_existing" || run.step("transcription").Status != "waiting" || run.step("diarization").Status != "waiting" {
		t.Fatalf("forced steps must not reuse their results: %+v", run)
	}
	if h.count(`SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND text IS NOT NULL AND stale_at IS NULL`, h.clipID) != 1 {
		t.Fatal("corrections must stay until the retranscription succeeds")
	}
	job, ok := h.claim()
	if !ok || job.Step != "transcription.audio.2" {
		t.Fatalf("expected the retranscription, got %+v", job)
	}
	h.complete(job, analysisTranscript, http.StatusOK)
	if h.count(`SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND text IS NOT NULL AND stale_at IS NULL`, h.clipID) != 0 ||
		h.count(`SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND text IS NOT NULL AND stale_at IS NOT NULL`, h.clipID) != 1 {
		t.Fatal("corrections must become obsolete after success, history kept")
	}
	diar, ok := h.claim()
	if !ok || diar.Step != "voice.diarization.2" {
		t.Fatalf("expected the new diarization, got %+v", diar)
	}
	h.complete(diar, diarizationPayload("NEW_A"), http.StatusOK)
	if h.count(`SELECT count(*) FROM local_speakers WHERE id=$1 AND kind='archived'`, first) != 1 {
		t.Fatal("the validated speaker must be kept (archived) after the new diarization")
	}

	// The history exposes what was forced, what was confirmed and the versions followed.
	var history struct {
		Runs []struct {
			analysisRunView
			Mode                        string   `json:"mode"`
			ForcedSteps                 []string `json:"forced_steps"`
			ConfirmedReplaceCorrections bool     `json:"confirmed_replace_corrections"`
			ConfirmedReplaceValidations bool     `json:"confirmed_replace_validations"`
			Steps                       []struct {
				Step         string  `json:"step"`
				Forced       bool    `json:"forced"`
				InputVersion *string `json:"input_version"`
				Profile      *string `json:"profile"`
			} `json:"steps"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, h.analysisPath()+"/history", nil, http.StatusOK), &history); err != nil {
		t.Fatal(err)
	}
	if len(history.Runs) < 2 || history.Runs[0].Mode != "full" || fmt.Sprint(history.Runs[0].ForcedSteps) != "[transcription diarization]" || !history.Runs[0].ConfirmedReplaceCorrections || !history.Runs[0].ConfirmedReplaceValidations || history.Runs[1].Mode != "missing" {
		t.Fatalf("unexpected history: %+v", history.Runs)
	}
	for _, st := range history.Runs[0].Steps {
		switch st.Step {
		case "transcription", "diarization":
			if !st.Forced || st.InputVersion == nil || st.Profile == nil || *st.Profile == "" {
				t.Fatalf("a forced step must expose its version and profile: %+v", st)
			}
		}
	}
	if st := history.Runs[1].Steps[1]; st.Forced {
		t.Fatalf("a reused step is not forced: %+v", st)
	}
}
