//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type accuracyReport struct {
	AutoEnabled        bool    `json:"auto_enabled"`
	PrecisionThreshold float64 `json:"precision_threshold"`
	MinConfirmations   int     `json:"min_confirmations"`
	Evaluation         struct {
		ReferenceSamples   int      `json:"reference_samples"`
		TestSamples        int      `json:"test_samples"`
		ExcludedSamples    int      `json:"excluded_samples"`
		EvaluatedAutomatic int      `json:"evaluated_automatic"`
		Confirmed          int      `json:"confirmed"`
		Precision          *float64 `json:"precision"`
		Coverage           *float64 `json:"coverage"`
	} `json:"evaluation"`
	Persons []struct {
		Person struct {
			DisplayName string `json:"display_name"`
		} `json:"person"`
		ReferenceSamples     int      `json:"reference_samples"`
		TestSamples          int      `json:"test_samples"`
		AutomaticPredictions int      `json:"automatic_predictions"`
		Confirmed            int      `json:"confirmed"`
		Precision            *float64 `json:"precision"`
	} `json:"persons"`
	Library struct {
		SpeakersTotal      int      `json:"speakers_total"`
		Validated          int      `json:"validated"`
		Automatic          int      `json:"automatic"`
		Suggestion         int      `json:"suggestion"`
		Ambiguous          int      `json:"ambiguous"`
		Unknown            int      `json:"unknown"`
		ToVerify           int      `json:"to_verify"`
		Coverage           *float64 `json:"coverage"`
		MicrophoneAssigned int      `json:"microphone_assigned"`
	} `json:"library"`
	Criteria struct {
		Met              bool `json:"met"`
		PrecisionMet     bool `json:"precision_met"`
		ConfirmationsMet bool `json:"confirmations_met"`
	} `json:"criteria"`
	ActivationProposed bool `json:"activation_proposed"`
	Events             []struct {
		Action      string `json:"action"`
		Author      string `json:"author"`
		CriteriaMet bool   `json:"criteria_met"`
	} `json:"events"`
}

func (h *voiceHarness) accuracy() accuracyReport {
	h.t.Helper()
	var r accuracyReport
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-accuracy", nil, http.StatusOK), &r); err != nil {
		h.t.Fatal(err)
	}
	return r
}

func approx(a *float64, b float64) bool { return a != nil && math.Abs(*a-b) < 0.001 }

func (h *voiceHarness) matchingEnabled() bool {
	h.t.Helper()
	var on bool
	if err := h.pool.QueryRow(context.Background(), `SELECT auto_enabled FROM voice_matching_settings WHERE id=true`).Scan(&on); err != nil {
		h.t.Fatal(err)
	}
	return on
}

func testVoiceAutomation(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 10)
	c := h.clipIDs
	if _, err := pool.Exec(ctx, `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}

	// Without any validation nothing is measured and nothing is proposed.
	empty := h.accuracy()
	if empty.Evaluation.Precision != nil || empty.ActivationProposed || empty.Criteria.Met || empty.AutoEnabled {
		t.Fatalf("an empty library measures nothing: %+v", empty)
	}

	// Each Person validates one voice in a first clip (reference) and one in a later clip
	// (test). Voices are angles: the cosine similarity is the cosine of the angle.
	alice := h.fixtureSpeaker(c[0], "REF_ALICE", usableAt(0))
	h.validate(alice, "Alice")
	aliceID := h.personOf(alice)
	h.validate(h.fixtureSpeaker(c[1], "TEST_ALICE", usableAt(3)), aliceID) // correct, automatic
	bob := h.fixtureSpeaker(c[2], "REF_BOB", usableAt(90))
	h.validate(bob, "Bob")
	bobID := h.personOf(bob)
	h.validate(h.fixtureSpeaker(c[3], "TEST_BOB", usableAt(85)), bobID) // correct, automatic
	carol := h.fixtureSpeaker(c[4], "REF_CAROL", usableAt(180))
	h.validate(carol, "Carol")
	carolID := h.personOf(carol)
	h.validate(h.fixtureSpeaker(c[5], "TEST_CAROL", usableAt(88)), carolID) // sounds like Bob: wrong automatic
	frank := h.fixtureSpeaker(c[7], "REF_FRANK", usableAt(270))
	h.validate(frank, "Frank")
	frankID := h.personOf(frank)
	h.validate(h.fixtureSpeaker(c[8], "TEST_FRANK", usableAt(45)), frankID) // between everyone: not automatic
	eve := h.fixtureSpeaker(c[6], "REF_EVE", usableAt(135))
	h.validate(eve, "Eve") // a single clip: reference only, never tested

	// Thresholds belong to the fingerprint group (#393): without applied ones, nothing would be automatic.
	if r := h.accuracy(); r.Evaluation.TestSamples != 4 || r.Evaluation.EvaluatedAutomatic != 0 || r.Evaluation.Precision != nil {
		t.Fatalf("a group without applied thresholds attributes nothing automatically: %+v", r.Evaluation)
	}
	h.applyThresholds(fixtureGroup, 0.90, 0.75)

	r := h.accuracy()
	e := r.Evaluation
	if e.ReferenceSamples != 5 || e.TestSamples != 4 || e.ExcludedSamples != 0 || e.EvaluatedAutomatic != 3 || e.Confirmed != 2 || !approx(e.Precision, 2.0/3) || !approx(e.Coverage, 0.75) {
		t.Fatalf("unexpected evaluation: %+v", e)
	}
	byName := map[string]int{}
	for i, p := range r.Persons {
		byName[p.Person.DisplayName] = i
	}
	if len(r.Persons) != 5 {
		t.Fatalf("every Person is listed: %+v", r.Persons)
	}
	pa, pb, pc, pe := r.Persons[byName["Alice"]], r.Persons[byName["Bob"]], r.Persons[byName["Carol"]], r.Persons[byName["Eve"]]
	if pa.AutomaticPredictions != 1 || pa.Confirmed != 1 || !approx(pa.Precision, 1) || pb.AutomaticPredictions != 2 || pb.Confirmed != 1 || !approx(pb.Precision, 0.5) || pc.AutomaticPredictions != 0 || pc.Precision != nil || pc.TestSamples != 1 || pe.TestSamples != 0 || pe.ReferenceSamples != 1 {
		t.Fatalf("unexpected per-Person measure: %+v", r.Persons)
	}
	// Measuring never depends on the switch and creates no reference.
	if r.AutoEnabled || h.count(`SELECT count(*) FROM voice_samples`) != 9 {
		t.Fatalf("measuring must not change anything: %+v", r)
	}

	// Criteria are configurable, administrators only, and validated.
	peer := h.newUser("peer")
	h.userCall(peer, http.MethodGet, "/api/admin/voice-accuracy", nil, http.StatusForbidden)
	h.userCall(peer, http.MethodPut, "/api/admin/voice-automation/criteria", map[string]any{"precision_threshold": 0.5, "min_confirmations": 1}, http.StatusForbidden)
	h.userCall(peer, http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": true}, http.StatusForbidden)
	h.userCall(peer, http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": true, "confirm_unmet_criteria": true}, http.StatusForbidden)
	for _, bad := range []map[string]any{{"precision_threshold": 1.5, "min_confirmations": 5}, {"precision_threshold": -0.1, "min_confirmations": 5}, {"precision_threshold": 0.9, "min_confirmations": 0}, {"precision_threshold": 0.9}, {"min_confirmations": 3}} {
		h.adminCall(http.MethodPut, "/api/admin/voice-automation/criteria", bad, http.StatusBadRequest)
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-automation/criteria", map[string]any{"precision_threshold": 0.95, "min_confirmations": 50}, http.StatusOK)
	if r = h.accuracy(); r.Criteria.Met || r.ActivationProposed || r.Criteria.PrecisionMet || r.Criteria.ConfirmationsMet {
		t.Fatalf("2 confirmations at 67 percent do not meet 95 percent over 50: %+v", r.Criteria)
	}

	// An activation below the criteria needs the explicit confirmation of the administrator.
	h.adminCall(http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": true}, http.StatusConflict)
	if h.matchingEnabled() {
		t.Fatal("a refused activation must not change anything")
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-automation/criteria", map[string]any{"precision_threshold": 0.6, "min_confirmations": 2}, http.StatusOK)
	if r = h.accuracy(); !r.Criteria.Met || !r.ActivationProposed || r.PrecisionThreshold != 0.6 || r.MinConfirmations != 2 {
		t.Fatalf("criteria reached: Snitch must propose the activation: %+v", r)
	}

	// Unvalidated voices in a clip without any reference: strong, middling, far. In production every validated
	// sample of a Person adds up to her cumulative Reference (ADR 0064): Alice points at 1.5 degrees, Bob at 87.5,
	// Carol at 134 (her test voice sounds like Bob), Eve at 135 and Frank at 337.5.
	strong := h.fixtureSpeaker(c[9], "STRONG", usableAt(2))     // cosine 0.999 to Alice
	middling := h.fixtureSpeaker(c[9], "MIDDLE", usableAt(300)) // 0.793 to Frank: a suggestion
	far := h.fixtureSpeaker(c[9], "FAR", usableAt(225))         // unknown
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	// The microphone speaker is counted apart: it never depends on the switch.
	mic := h.fixtureSpeaker(c[9], "MIC", usableAt(5))
	if _, err := pool.Exec(ctx, `UPDATE local_speakers SET kind='microphone',diarization_job_id=NULL WHERE id=$1`, mic.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,confidence,origin) VALUES($1,$2,'automatic','unvalidated',1,'microphone_track')`, mic.ID, bobID); err != nil {
		t.Fatal(err)
	}
	if got := h.matched(strong); got.DecisionLevel != "suggestion" {
		t.Fatalf("automation off: a strong match is a suggestion: %+v", got)
	}
	l := h.accuracy().Library
	if l.SpeakersTotal != 12 || l.Validated != 9 || l.Automatic != 1 || l.Suggestion != 1 || l.Unknown != 1 || l.Ambiguous != 0 || l.ToVerify != 2 || !approx(l.Coverage, 1.0/3) || l.MicrophoneAssigned != 1 {
		t.Fatalf("library figures must come from the database and keep the microphone apart: %+v", l)
	}

	// Activation: administrators only, recorded, rematch triggered, every user sees it.
	var changed struct {
		Settings struct {
			AutoEnabled bool `json:"auto_enabled"`
		} `json:"settings"`
		Summary struct {
			Automatic int `json:"automatic"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": true}, http.StatusOK), &changed); err != nil {
		t.Fatal(err)
	}
	if !changed.Settings.AutoEnabled || changed.Summary.Automatic != 1 {
		t.Fatalf("activation must enable the switch and rematch: %+v", changed)
	}
	if got := h.matched(strong); got.DecisionLevel != "automatic" || got.Person == nil || got.Person.ID != aliceID {
		t.Fatalf("automation on: the strong match becomes automatic: %+v", got)
	}
	if got := h.matched(middling); got.DecisionLevel != "suggestion" {
		t.Fatalf("a middling match stays a suggestion: %+v", got)
	}
	if got := h.matched(far); got.DecisionLevel != "unknown" {
		t.Fatalf("a distant voice stays unknown: %+v", got)
	}
	var visible struct {
		AutoEnabled bool `json:"auto_enabled"`
	}
	if err := json.Unmarshal(h.userCall(peer, http.MethodGet, "/api/voice-matching", nil, http.StatusOK), &visible); err != nil || !visible.AutoEnabled {
		t.Fatalf("every user must see the automation state: %v %+v", err, visible)
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": true}, http.StatusOK) // idempotent
	if h.count(`SELECT count(*) FROM voice_samples`) != 9 {
		t.Fatal("automatic attributions must never become validated samples")
	}
	if n := h.count(`SELECT count(*) FROM voice_automation_events WHERE action='enable'`); n != 1 {
		t.Fatalf("one enable event expected, got %d", n)
	}

	// Deactivation brings the automatic attributions back to suggestions.
	h.userCall(peer, http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": false}, http.StatusForbidden)
	if err := json.Unmarshal(h.adminCall(http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": false}, http.StatusOK), &changed); err != nil {
		t.Fatal(err)
	}
	if changed.Settings.AutoEnabled || changed.Summary.Automatic != 0 {
		t.Fatalf("deactivation: %+v", changed)
	}
	if got := h.matched(strong); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != aliceID {
		t.Fatalf("deactivation must bring automatic attributions back to suggestions: %+v", got)
	}
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE decision_level='automatic' AND origin='matching'`) != 0 {
		t.Fatal("no matching attribution may stay automatic")
	}
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=$1 AND origin='microphone_track' AND decision_level='automatic'`, mic.ID) != 1 {
		t.Fatal("the microphone attribution does not depend on the switch")
	}
	if r = h.accuracy(); r.AutoEnabled || len(r.Events) != 2 || r.Events[0].Action != "disable" || r.Events[1].Action != "enable" || r.Events[1].Author != "administrator" || !r.Events[1].CriteriaMet {
		t.Fatalf("history of the switch: %+v", r.Events)
	}

	// Forcing an activation below the criteria is possible but recorded as such.
	h.adminCall(http.MethodPut, "/api/admin/voice-automation/criteria", map[string]any{"precision_threshold": 0.99, "min_confirmations": 50}, http.StatusOK)
	h.adminCall(http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": true}, http.StatusConflict)
	h.adminCall(http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": true, "confirm_unmet_criteria": true}, http.StatusOK)
	if r = h.accuracy(); !r.AutoEnabled || r.Events[0].CriteriaMet {
		t.Fatalf("a forced activation records that the criteria were not met: %+v", r.Events)
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-automation", map[string]any{"enabled": false}, http.StatusOK)

	// Evaluation uses distinct clips: a clip that holds a reference of another Person
	// cannot hold a test. Gina validates a voice in Bob's test clip, which becomes a
	// reference clip: Bob's test sample is set aside.
	h.validate(h.fixtureSpeaker(c[3], "REF_GINA", usableAt(200)), "Gina")
	e = h.accuracy().Evaluation
	if e.ReferenceSamples != 6 || e.TestSamples != 3 || e.ExcludedSamples != 1 || e.EvaluatedAutomatic != 2 || e.Confirmed != 1 || !approx(e.Precision, 0.5) {
		t.Fatalf("a shared clip must never serve as reference and test: %+v", e)
	}
}
