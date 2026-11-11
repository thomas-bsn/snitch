//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Preview and requalification of the fingerprints already stored (#394, spec voix-breves section 6, steps 1
// and 2). Every case is synthetic.

type eligibilityOutcomeView struct {
	Present              int `json:"present"`
	Comparable           int `json:"comparable"`
	Contributive         int `json:"contributive"`
	TooShort             int `json:"too_short"`
	OverlapOnly          int `json:"overlap_only"`
	Clipped              int `json:"clipped"`
	LowQuality           int `json:"low_quality"`
	PersonsWithReference int `json:"persons_with_reference"`
}

type eligibilityScenarioView struct {
	Rule struct {
		ComparableMinSeconds   float64 `json:"comparable_min_seconds"`
		ContributiveMinSeconds float64 `json:"contributive_min_seconds"`
		MaxClippedRatio        float64 `json:"max_clipped_ratio"`
	} `json:"rule"`
	eligibilityOutcomeView
	RecoveredComparable   int `json:"recovered_comparable"`
	RecoveredContributive int `json:"recovered_contributive"`
	LostComparable        int `json:"lost_comparable"`
	LostContributive      int `json:"lost_contributive"`
}

type eligibilityPersonView struct {
	Person struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"person"`
	Present        int  `json:"present"`
	Comparable     int  `json:"comparable"`
	Contributive   int  `json:"contributive"`
	TooShort       int  `json:"too_short"`
	OverlapOnly    int  `json:"overlap_only"`
	Clipped        int  `json:"clipped"`
	MissingSources int  `json:"missing_sources"`
	Estimated      int  `json:"estimated"`
	Reference      bool `json:"reference"`
	ReferenceNow   bool `json:"reference_now"`
}

type eligibilityPreviewView struct {
	PersonsWithSamples int `json:"persons_with_samples"`
	Groups             []struct {
		Model           string                    `json:"model"`
		ModelVersion    string                    `json:"model_version"`
		Dimensions      int                       `json:"dimensions"`
		MissingSources  int                       `json:"missing_sources"`
		Estimated       int                       `json:"estimated"`
		OriginalVectors int                       `json:"original_vectors"`
		OverlapShare    *float64                  `json:"overlap_share"`
		Current         eligibilityOutcomeView    `json:"current"`
		Candidate       eligibilityScenarioView   `json:"candidate"`
		Scenarios       []eligibilityScenarioView `json:"scenarios"`
		Persons         []eligibilityPersonView   `json:"persons"`
	} `json:"groups"`
}

type eligibilityRequalificationView struct {
	RuleVersion     int                    `json:"rule_version"`
	Author          *string                `json:"author"`
	Changed         int                    `json:"changed"`
	MissingSources  int                    `json:"missing_sources"`
	Estimated       int                    `json:"estimated"`
	OriginalVectors int                    `json:"original_vectors"`
	Before          eligibilityOutcomeView `json:"before"`
	After           eligibilityOutcomeView `json:"after"`
}

func (h *voiceHarness) eligibilityPreview(query string) eligibilityPreviewView {
	h.t.Helper()
	var out eligibilityPreviewView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-eligibility/preview"+query, nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	if len(out.Groups) != 1 {
		h.t.Fatalf("one production fingerprint group expected: %+v", out)
	}
	return out
}

// requalify saves a rule (which requalifies every production fingerprint) and returns the traced report.
func (h *voiceHarness) requalify(comparable, contributive, clipped float64) eligibilityRequalificationView {
	h.t.Helper()
	var out struct {
		Requalifications []eligibilityRequalificationView `json:"requalifications"`
	}
	body := map[string]any{"comparable_min_seconds": comparable, "contributive_min_seconds": contributive, "max_clipped_ratio": clipped}
	if err := json.Unmarshal(h.adminCall(http.MethodPut, "/api/admin/voice-eligibility", body, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	if len(out.Requalifications) == 0 {
		h.t.Fatal("the requalification is not traced")
	}
	return out.Requalifications[0]
}

// digest fingerprints the whole content of a table, to prove that nothing in it changed.
func (h *voiceHarness) digest(table string) string {
	h.t.Helper()
	var out *string
	if err := h.pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT md5(string_agg(t::text, '|' ORDER BY t::text)) FROM %s t`, table)).Scan(&out); err != nil {
		h.t.Fatal(err)
	}
	if out == nil {
		return ""
	}
	return *out
}

func (h *voiceHarness) digests(tables ...string) map[string]string {
	out := map[string]string{}
	for _, table := range tables {
		out[table] = h.digest(table)
	}
	return out
}

func testVoiceEligibilityPreview(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	ctx := context.Background()
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]

	// Fingerprints stored before the rule: all decided "too short, not contributive" by the fixture, so every
	// eligibility the preview announces is a change.
	alice1 := h.measuredSpeaker(clipA, "ALICE1", 0, measured{net: ptr(1.2), duration: 1.2})
	alice2 := h.measuredSpeaker(clipA, "ALICE2", 2, measured{net: ptr(1.7), duration: 1.7})
	// Without measures, the net speech comes from the turns: 3 s of turns, 1 s spoken over by another speaker.
	h.measuredSpeaker(clipA, "TURNS", 90, measured{duration: 3, turns: [][2]float64{{0, 3}}, others: [][2]float64{{2, 3}}})
	bob := h.measuredSpeaker(clipB, "BOB", 40, measured{net: ptr(0.8), duration: 0.8})
	carol := h.measuredSpeaker(clipB, "CAROL", 60, measured{net: ptr(2.5), duration: 2.5})
	h.measuredSpeaker(clipB, "UNKNOWN", 120, measured{net: ptr(1.6), duration: 1.6})
	dan := h.measuredSpeaker(clipB, "DAN", 150, measured{net: ptr(0.0), overlapped: true, duration: 3})
	// Clip C has lost its source file: a validated speaker archived without turns (estimated net speech)...
	eve := h.measuredSpeaker(clipC, "EVE", 170, measured{duration: 1.4})
	h.measuredSpeaker(clipC, "CLIPPED", 30, measured{net: ptr(5.0), clipped: ptr(0.5), duration: 5})
	// ...and a benchmark fingerprint the preview never counts.
	h.measuredSpeaker(clipC, "BENCH", 5, measured{net: ptr(6.0), duration: 6, benchmark: true})
	if _, err := pool.Exec(ctx, `UPDATE source_files SET status='missing' WHERE clip_id=$1`, clipC); err != nil {
		t.Fatal(err)
	}
	h.validate(alice1, "Alice")
	h.validate(alice2, h.personOf(alice1))
	h.validate(bob, "Bob")
	h.validate(carol, "Carol")
	h.validate(dan, "Dan")
	h.validate(eve, "Eve")

	// The preview changes nothing: no fingerprint, rule, trace, job, Person, sample or attribution.
	protected := []string{"voice_fingerprints", "voice_eligibility_rules", "voice_eligibility_requalifications", "jobs", "persons", "voice_samples", "speaker_attributions", "local_speakers"}
	before := h.digests(protected...)
	preview := h.eligibilityPreview("")
	if after := h.digests(protected...); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatal("the preview wrote to the database")
	}

	g := preview.Groups[0]
	if g.Model != "fixture-embedding" || g.ModelVersion != "emb-v1" || g.Dimensions != 4 {
		t.Fatalf("group: %+v", g)
	}
	if preview.PersonsWithSamples != 5 {
		t.Fatalf("five Persons have validated samples: %d", preview.PersonsWithSamples)
	}
	// Two production fingerprints of clip C have no audio source; Eve's net speech is an estimate; the
	// fingerprints without Worker measures keep their original vector.
	if g.MissingSources != 2 || g.Estimated != 1 || g.OriginalVectors != 2 {
		t.Fatalf("missing=%d estimated=%d original=%d", g.MissingSources, g.Estimated, g.OriginalVectors)
	}
	// One third of TURNS is overlapped; every other known share is zero (2 s of speech each): 1/3 over 8.
	if g.OverlapShare == nil || math.Abs(*g.OverlapShare-0.3333/8) > 1e-3 {
		t.Fatalf("overlap share: %v", g.OverlapShare)
	}
	if want := (eligibilityOutcomeView{Present: 9, TooShort: 9}); g.Current != want {
		t.Fatalf("current state: %+v", g.Current)
	}
	// Under the active rule (2 s comparable, 1 s contributive): CAROL and TURNS comparable; ALICE1, ALICE2,
	// UNKNOWN and EVE contributive only; BOB too short; DAN overlapped; CLIPPED clipped.
	candidate := eligibilityOutcomeView{Present: 9, Comparable: 2, Contributive: 6, TooShort: 1, OverlapOnly: 1, Clipped: 1, PersonsWithReference: 3}
	if g.Candidate.eligibilityOutcomeView != candidate || g.Candidate.RecoveredComparable != 2 || g.Candidate.RecoveredContributive != 6 || g.Candidate.Rule.ComparableMinSeconds != 2 {
		t.Fatalf("candidate: %+v", g.Candidate)
	}
	// The default durations: 1, 1.5 and 2 s, each applied to both uses.
	wanted := []struct {
		seconds  float64
		eligible int
		persons  int
	}{{1, 6, 3}, {1.5, 4, 2}, {2, 2, 1}}
	if len(g.Scenarios) != len(wanted) {
		t.Fatalf("scenarios: %+v", g.Scenarios)
	}
	for i, w := range wanted {
		s := g.Scenarios[i]
		if s.Rule.ComparableMinSeconds != w.seconds || s.Rule.ContributiveMinSeconds != w.seconds || s.Comparable != w.eligible || s.Contributive != w.eligible ||
			s.RecoveredComparable != w.eligible || s.PersonsWithReference != w.persons || s.LostComparable != 0 {
			t.Fatalf("scenario %v s: %+v", w.seconds, s)
		}
	}
	// Per Person, under the candidate rule: what their validated samples become.
	persons := map[string]eligibilityPersonView{}
	for _, p := range g.Persons {
		persons[p.Person.DisplayName] = p
	}
	if p := persons["Alice"]; p.Present != 2 || p.Contributive != 2 || p.Comparable != 0 || !p.Reference || p.ReferenceNow {
		t.Fatalf("Alice: %+v", p)
	}
	if p := persons["Bob"]; p.Present != 1 || p.TooShort != 1 || p.Reference {
		t.Fatalf("Bob: %+v", p)
	}
	if p := persons["Dan"]; p.OverlapOnly != 1 || p.Reference {
		t.Fatalf("Dan: %+v", p)
	}
	if p := persons["Eve"]; p.MissingSources != 1 || p.Estimated != 1 || !p.Reference {
		t.Fatalf("Eve: %+v", p)
	}

	// A preview of another rule and other durations; invalid values are refused.
	other := h.eligibilityPreview("?comparable_min_seconds=1.5&contributive_min_seconds=0.5&max_clipped_ratio=0.6&durations=0.5")
	if c := other.Groups[0].Candidate; c.Comparable != 5 || c.Contributive != 8 || c.Clipped != 0 || len(other.Groups[0].Scenarios) != 1 || other.Groups[0].Scenarios[0].Contributive != 8 {
		t.Fatalf("other rule: %+v", other.Groups[0])
	}
	for _, bad := range []string{"?comparable_min_seconds=1&contributive_min_seconds=2", "?durations=0.2", "?max_clipped_ratio=2", "?comparable_min_seconds=abc"} {
		h.adminCall(http.MethodGet, "/api/admin/voice-eligibility/preview"+bad, nil, http.StatusBadRequest)
	}
	peer := h.newUser("peer")
	h.userCall(peer, http.MethodGet, "/api/admin/voice-eligibility/preview", nil, http.StatusForbidden)

	// The figures of the preview for a duration are those obtained by requalifying at that duration.
	humans := []string{"persons", "voice_samples", "(SELECT * FROM speaker_attributions WHERE validation_state='validated') AS"}
	humanBefore := h.digests(humans...)
	jobs := h.count(`SELECT count(*) FROM jobs`)
	for i, w := range wanted {
		report := h.requalify(w.seconds, w.seconds, 0.01)
		now := h.eligibilityPreview("").Groups[0].Current
		if want := g.Scenarios[i].eligibilityOutcomeView; now != want || report.After != want {
			t.Fatalf("requalified at %v s: now %+v, report %+v, preview announced %+v", w.seconds, now, report.After, want)
		}
		if report.Author == nil || *report.Author != "administrator" || report.RuleVersion != i+2 {
			t.Fatalf("the requalification is traced with its author and rule: %+v", report)
		}
		// The fingerprint of a clip without source is requalified, and reported; so are the estimated net speech
		// and the vectors left as they were computed.
		if report.MissingSources != 2 || report.Estimated != 1 || report.OriginalVectors != 2 || h.count(`SELECT count(*) FROM voice_fingerprints WHERE local_speaker_id=$1 AND eligibility_rule_version=$2`, eve.ID, i+2) != 1 {
			t.Fatalf("missing source: %+v", report)
		}
	}
	if h.count(`SELECT count(*) FROM jobs`) != jobs {
		t.Fatal("the requalification created jobs")
	}
	if after := h.digests(humans...); fmt.Sprint(after) != fmt.Sprint(humanBefore) {
		t.Fatal("the requalification changed a Person, a sample or a validation")
	}

	// Requalifying again under the same rule changes nothing, and is traced too.
	fingerprints := h.digest("voice_fingerprints")
	again := h.requalify(2, 2, 0.01)
	if again.Changed != 0 || again.Before != again.After || again.RuleVersion != 4 || h.digest("voice_fingerprints") != fingerprints {
		t.Fatalf("requalifying twice: %+v", again)
	}
	if n := h.count(`SELECT count(*) FROM voice_eligibility_requalifications`); n != 4 {
		t.Fatalf("each requalification is traced: %d", n)
	}
	// The benchmark fingerprint was never touched.
	if h.count(`SELECT count(*) FROM voice_fingerprints WHERE benchmark AND eligibility_rule_version IS NOT NULL`) != 0 {
		t.Fatal("a benchmark fingerprint was requalified")
	}
}
