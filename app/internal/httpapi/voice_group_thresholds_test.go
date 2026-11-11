//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"snitch/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Thresholds of a fingerprint group (ADR 0064, #393): matching thresholds belong to one (model, version,
// dimensions) group, are applied only by an administrator, and a group without applied thresholds only ever
// produces suggestions.

type groupView struct {
	Model        string `json:"model"`
	ModelVersion string `json:"model_version"`
	Dimensions   int    `json:"dimensions"`
}

type matchingView struct {
	AutoEnabled         bool       `json:"auto_enabled"`
	AutomaticThreshold  float64    `json:"automatic_threshold"`
	SuggestionThreshold float64    `json:"suggestion_threshold"`
	AmbiguityMargin     float64    `json:"ambiguity_margin"`
	Group               *groupView `json:"group"`
	ThresholdsApplied   bool       `json:"thresholds_applied"`
}

type thresholdsVersionView struct {
	Version             int     `json:"version"`
	AutomaticThreshold  float64 `json:"automatic_threshold"`
	SuggestionThreshold float64 `json:"suggestion_threshold"`
	AmbiguityMargin     float64 `json:"ambiguity_margin"`
	Source              string  `json:"source"`
	ScoreMode           string  `json:"score_mode"`
	Author              *string `json:"author"`
}

type calibrationModeView struct {
	Mode              string `json:"mode"`
	Tests             int    `json:"tests"`
	Persons           int    `json:"persons"`
	SamePersonScores  int    `json:"same_person_scores"`
	OtherPersonScores int    `json:"other_person_scores"`
	Distribution      []struct {
		SamePerson  int `json:"same_person"`
		OtherPerson int `json:"other_person"`
	} `json:"distribution"`
	Sweep []struct {
		AutomaticThreshold float64 `json:"automatic_threshold"`
		Automatic          int     `json:"automatic"`
		Errors             int     `json:"errors"`
		ToVerify           int     `json:"to_verify"`
		UnknownAutomatic   int     `json:"unknown_automatic"`
	} `json:"sweep"`
	Current  *struct{ Automatic int } `json:"current"`
	Proposal *struct {
		AutomaticThreshold  float64 `json:"automatic_threshold"`
		SuggestionThreshold float64 `json:"suggestion_threshold"`
		AmbiguityMargin     float64 `json:"ambiguity_margin"`
	} `json:"proposal"`
	Withheld string `json:"withheld"`
}

type calibrationView struct {
	Calibration struct {
		Group      *groupView              `json:"group"`
		Thresholds *thresholdsVersionView  `json:"thresholds"`
		History    []thresholdsVersionView `json:"history"`
		ActiveMode string                  `json:"active_mode"`
		MinTests   int                     `json:"min_tests"`
		MinPersons int                     `json:"min_persons"`
		Modes      []calibrationModeView   `json:"modes"`
	} `json:"calibration"`
	Evaluation struct {
		TestSamples        int `json:"test_samples"`
		ExcludedSamples    int `json:"excluded_samples"`
		EvaluatedAutomatic int `json:"evaluated_automatic"`
	} `json:"evaluation"`
}

func (h *voiceHarness) matchingSettings(u *voiceUser) matchingView {
	h.t.Helper()
	var raw []byte
	if u == nil {
		raw = h.adminCall(http.MethodGet, "/api/voice-matching", nil, http.StatusOK)
	} else {
		raw = h.userCall(u, http.MethodGet, "/api/voice-matching", nil, http.StatusOK)
	}
	var v matchingView
	if err := json.Unmarshal(raw, &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

func (h *voiceHarness) calibration() calibrationView {
	h.t.Helper()
	var v calibrationView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-accuracy", nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

var fixtureGroup = map[string]any{"model": "fixture-embedding", "model_version": "emb-v1", "dimensions": 4}

// applyThresholds applies thresholds to a fingerprint group the way an administrator does.
func (h *voiceHarness) applyThresholds(group map[string]any, automatic, suggestion float64) {
	h.t.Helper()
	h.adminCall(http.MethodPut, "/api/admin/voice-matching", map[string]any{"automatic_threshold": automatic, "suggestion_threshold": suggestion, "ambiguity_margin": 0.05, "max_candidates": 5, "group": group}, http.StatusOK)
}

func testVoiceGroupThresholds(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 10)
	c := h.clipIDs
	h.putEligibility(1, 0.5, 0.05)
	// The automation is on: only the thresholds of the group decide whether a voice can be automatic.
	if _, err := pool.Exec(ctx, `UPDATE voice_matching_settings SET auto_enabled=true,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}

	// A blank installation has no fingerprint group, hence no thresholds.
	if v := h.matchingSettings(nil); v.Group != nil || v.ThresholdsApplied || !v.AutoEnabled {
		t.Fatalf("a blank installation has no active group: %+v", v)
	}
	if cal := h.calibration().Calibration; cal.Group != nil || len(cal.Modes) != 0 || cal.Thresholds != nil {
		t.Fatalf("nothing to calibrate without fingerprints: %+v", cal)
	}

	// The first fingerprints make their group active, without thresholds: a strong match is only a suggestion.
	reference := h.fixtureSpeaker(c[0], "REF_ALICE", usableAt(0))
	near := h.fixtureSpeaker(c[1], "NEAR", usableAt(5))
	h.validate(reference, "Alice")
	alice := h.personOf(reference)
	if got := h.matched(near); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != alice {
		t.Fatalf("a group without applied thresholds only suggests: %+v", got)
	}
	peer := h.newUser("peer")
	if v := h.matchingSettings(peer); v.Group == nil || v.Group.Model != "fixture-embedding" || v.Group.ModelVersion != "emb-v1" || v.Group.Dimensions != 4 || v.ThresholdsApplied {
		t.Fatalf("every user sees the active group and that it has no thresholds: %+v", v)
	}
	if n := h.count(`SELECT count(*) FROM voice_group_thresholds`); n != 0 {
		t.Fatalf("nothing is applied without an administrator: %d", n)
	}

	// Applying is an administrator action, traced with its author; then strong matches become automatic.
	body := map[string]any{"automatic_threshold": 0.9, "suggestion_threshold": 0.75, "ambiguity_margin": 0.05, "max_candidates": 5, "group": fixtureGroup}
	h.userCall(peer, http.MethodPut, "/api/admin/voice-matching", body, http.StatusForbidden)
	unknownGroup := map[string]any{"automatic_threshold": 0.9, "suggestion_threshold": 0.75, "ambiguity_margin": 0.05, "max_candidates": 5, "group": map[string]any{"model": "fixture-embedding", "model_version": "emb-v9", "dimensions": 4}}
	h.adminCall(http.MethodPut, "/api/admin/voice-matching", unknownGroup, http.StatusConflict)
	h.adminCall(http.MethodPut, "/api/admin/voice-matching", map[string]any{"automatic_threshold": 0.9, "suggestion_threshold": 0.75, "ambiguity_margin": 0.05, "max_candidates": 5, "source": "guess"}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, "/api/admin/voice-matching", body, http.StatusOK)
	if got := h.matched(near); got.DecisionLevel != "automatic" || got.Person == nil || got.Person.ID != alice {
		t.Fatalf("applied thresholds allow automatic attributions: %+v", got)
	}
	if v := h.matchingSettings(peer); !v.ThresholdsApplied || v.AutomaticThreshold != 0.9 {
		t.Fatalf("the active group has its thresholds: %+v", v)
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-matching", body, http.StatusOK) // same values: nothing new
	cal := h.calibration().Calibration
	if cal.Thresholds == nil || cal.Thresholds.Version != 1 || cal.Thresholds.Source != "administrator" || cal.Thresholds.Author == nil || *cal.Thresholds.Author != "administrator" || cal.Thresholds.ScoreMode != "cumulative" || len(cal.History) != 1 {
		t.Fatalf("applied thresholds are versioned and traced: %+v", cal)
	}
	// Without a group in the request, the thresholds go to the active group.
	h.adminCall(http.MethodPut, "/api/admin/voice-matching", map[string]any{"automatic_threshold": 0.999, "suggestion_threshold": 0.75, "ambiguity_margin": 0.05, "max_candidates": 5}, http.StatusOK)
	if got := h.matched(near); got.DecisionLevel != "suggestion" {
		t.Fatalf("a stricter automatic threshold of the active group: %+v", got)
	}
	h.applyThresholds(fixtureGroup, 0.9, 0.75)
	if cal = h.calibration().Calibration; len(cal.History) != 3 || cal.Thresholds.Version != 3 || cal.History[1].AutomaticThreshold != 0.999 {
		t.Fatalf("every change is a new version: %+v", cal.History)
	}

	// A new fingerprint model (another version) forms its own group, without thresholds: its voices are only
	// suggested, while the first group keeps its automatic attributions.
	newModel := fixtureFingerprint{model: "fixture-embedding", version: "emb-v2", dimensions: 4, vector: angle(0), quality: 0.9, usable: true}
	h.validate(h.fixtureSpeaker(c[2], "REF_ALICE_V2", newModel), alice)
	newNear := newModel
	newNear.vector = angle(5)
	nearV2 := h.fixtureSpeaker(c[3], "NEAR_V2", newNear)
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	if got := h.matched(nearV2); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != alice {
		t.Fatalf("a group without thresholds only suggests: %+v", got)
	}
	if got := h.matched(near); got.DecisionLevel != "automatic" {
		t.Fatalf("the first group keeps its thresholds: %+v", got)
	}
	if v := h.matchingSettings(peer); v.Group == nil || v.Group.ModelVersion != "emb-v2" || v.ThresholdsApplied {
		t.Fatalf("the latest group is the active one, without thresholds: %+v", v)
	}
	h.applyThresholds(map[string]any{"model": "fixture-embedding", "model_version": "emb-v2", "dimensions": 4}, 0.9, 0.75)
	if got := h.matched(nearV2); got.DecisionLevel != "automatic" {
		t.Fatalf("once applied, the new group attributes automatically: %+v", got)
	}

	// Too few tests: the report shows both score modes and withholds the proposal.
	testV2 := newModel
	testV2.vector = angle(3)
	h.validate(h.fixtureSpeaker(c[4], "TEST_ALICE", testV2), alice)
	cal = h.calibration().Calibration
	if cal.Group == nil || cal.Group.ModelVersion != "emb-v2" || cal.ActiveMode != "cumulative" || cal.MinTests != 20 || cal.MinPersons != 3 || len(cal.Modes) != 2 || cal.Modes[0].Mode != "cumulative" || cal.Modes[1].Mode != "best_sample" {
		t.Fatalf("the calibration covers the active group in both modes: %+v", cal)
	}
	for _, m := range cal.Modes {
		if m.Proposal != nil || m.Withheld != "too_few" || len(m.Sweep) != 20 || len(m.Distribution) != 20 {
			t.Fatalf("too few tests: no proposal: %+v", m)
		}
	}
}

// The calibration replays the production score on the validated samples of the active group, with the rule of
// the measure: no test shares a clip with a reference.
func testVoiceCalibration(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 10)
	c := h.clipIDs
	h.putEligibility(1, 0.5, 0.05)
	if _, err := pool.Exec(ctx, `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5,precision_threshold=0.95`); err != nil {
		t.Fatal(err)
	}
	// Four Persons far apart; each validates one reference voice in a first clip and six test voices in a later
	// clip, 1 to 6 degrees away.
	persons := make([]int64, 4)
	for p := range 4 {
		base := float64(p) * 90
		ref := h.fixtureSpeaker(c[2*p], fmt.Sprintf("REF_%d", p), usableAt(base))
		name := []string{"Alice", "Bob", "Carol", "Dora"}[p]
		h.validate(ref, name)
		person := h.personOf(ref)
		persons[p] = person
		for i := 1; i <= 6; i++ {
			h.validate(h.fixtureSpeaker(c[2*p+1], fmt.Sprintf("TEST_%d_%d", p, i), usableAt(base+float64(i))), person)
		}
	}
	// Eve's test voice shares its clip with the only reference of Fay: it is set aside, never tested. Both voices
	// lie off the plane of the others (cosine 0 with all of them).
	off := func(axis int) fixtureFingerprint {
		v := []float64{0, 0, 0, 0}
		v[axis] = 1
		return fixtureFingerprint{model: "fixture-embedding", version: "emb-v1", dimensions: 4, vector: v, quality: 0.9, usable: true}
	}
	eve := h.fixtureSpeaker(c[8], "REF_EVE", off(2))
	h.validate(eve, "Eve")
	h.validate(h.fixtureSpeaker(c[9], "REF_FAY", off(3)), "Fay")
	h.validate(h.fixtureSpeaker(c[9], "SHARED", off(2)), h.personOf(eve))
	if persons[0] == persons[1] {
		t.Fatal("four distinct Persons expected")
	}

	report := h.calibration()
	cal := report.Calibration
	if report.Evaluation.TestSamples != 24 || report.Evaluation.ExcludedSamples != 1 || report.Evaluation.EvaluatedAutomatic != 0 {
		t.Fatalf("no applied thresholds: the measure attributes nothing automatically: %+v", report.Evaluation)
	}
	if cal.Thresholds != nil || len(cal.Modes) != 2 {
		t.Fatalf("calibration of a group without thresholds: %+v", cal)
	}
	for _, m := range cal.Modes {
		if m.Tests != 24 || m.Persons != 4 || m.SamePersonScores != 24 || m.OtherPersonScores != 24*5 || m.Current != nil {
			t.Fatalf("%s: every test against every Reference, the shared clip left out: %+v", m.Mode, m)
		}
		if m.Proposal == nil || m.Withheld != "" || m.Proposal.AutomaticThreshold != 0.11 || m.Proposal.SuggestionThreshold > m.Proposal.AutomaticThreshold {
			t.Fatalf("%s: a proposal is made on enough tests: %+v", m.Mode, m)
		}
	}

	// Applying the proposal is an explicit administrator action, recorded as such.
	p := cal.Modes[0].Proposal
	h.adminCall(http.MethodPut, "/api/admin/voice-matching", map[string]any{"automatic_threshold": p.AutomaticThreshold, "suggestion_threshold": p.SuggestionThreshold, "ambiguity_margin": p.AmbiguityMargin, "max_candidates": 5, "group": fixtureGroup, "source": "proposal"}, http.StatusOK)
	report = h.calibration()
	if cal = report.Calibration; cal.Thresholds == nil || cal.Thresholds.Source != "proposal" || cal.Thresholds.AutomaticThreshold != 0.11 || cal.Modes[0].Current == nil || cal.Modes[0].Current.Automatic != 24 {
		t.Fatalf("the applied proposal is traced and measured: %+v", cal)
	}
	if report.Evaluation.EvaluatedAutomatic != 24 {
		t.Fatalf("with applied thresholds the measure attributes automatically: %+v", report.Evaluation)
	}
}

// Migration 00147 gives the thresholds of the installation to the active fingerprint group, and to nothing else.
func testVoiceGroupThresholdsMigration(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	raw, err := migrations.Files.ReadFile("00147_voice_group_thresholds.sql")
	if err != nil {
		t.Fatal(err)
	}
	up := strings.SplitN(strings.TrimPrefix(string(raw), "-- +goose Up"), "-- +goose Down", 2)[0]
	seed := up[strings.Index(up, "INSERT INTO voice_group_thresholds"):]
	h := newVoiceHarnessClips(t, pool, 2)
	if _, err = pool.Exec(ctx, `UPDATE voice_matching_settings SET automatic_threshold=0.93,suggestion_threshold=0.7,ambiguity_margin=0.04`); err != nil {
		t.Fatal(err)
	}
	// A blank installation: no group, nothing seeded.
	if _, err = pool.Exec(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if n := h.count(`SELECT count(*) FROM voice_group_thresholds`); n != 0 {
		t.Fatalf("no fingerprint, no group: %d", n)
	}
	// Production fingerprints of an older group, then of the active one; benchmark fingerprints are ignored.
	h.fixtureSpeaker(h.clipIDs[0], "OLD", fixtureFingerprint{model: "fixture-embedding", version: "emb-v0", dimensions: 4, vector: angle(0), quality: 0.9, usable: true})
	h.fixtureSpeaker(h.clipIDs[0], "ACTIVE", usableAt(0))
	bench := h.fixtureSpeaker(h.clipIDs[1], "BENCH", fixtureFingerprint{model: "fixture-embedding", version: "emb-v3", dimensions: 4, vector: angle(0), quality: 0.9, usable: true})
	if _, err = pool.Exec(ctx, `UPDATE voice_fingerprints SET benchmark=true WHERE local_speaker_id=$1`, bench.ID); err != nil {
		t.Fatal(err)
	}
	for range 2 { // idempotent
		if _, err = pool.Exec(ctx, seed); err != nil {
			t.Fatal(err)
		}
	}
	var model, version, source string
	var dims, n int
	var automatic, suggestion, margin float64
	var author *string
	if err = pool.QueryRow(ctx, `SELECT model,model_version,dimensions,automatic_threshold,suggestion_threshold,ambiguity_margin,source,author_username,(SELECT count(*) FROM voice_group_thresholds) FROM voice_group_thresholds`).Scan(&model, &version, &dims, &automatic, &suggestion, &margin, &source, &author, &n); err != nil {
		t.Fatal(err)
	}
	if n != 1 || model != "fixture-embedding" || version != "emb-v1" || dims != 4 || automatic != 0.93 || suggestion != 0.7 || margin != 0.04 || source != "migration" || author != nil {
		t.Fatalf("the current thresholds go to the active group only: %s %s %d %v %v %v %s %v (%d rows)", model, version, dims, automatic, suggestion, margin, source, author, n)
	}
}
