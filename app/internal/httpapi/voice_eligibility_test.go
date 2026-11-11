//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Eligibility of a fingerprint (ADR 0064, #390): Snitch decides it from the stored measures and a versioned
// rule. Every case is synthetic: fingerprints and turns are inserted the way a fingerprint result projects them.

// measured describes the stored state of one speaker-scope fingerprint. A nil net means a fingerprint stored
// without measures (older Worker or row before the measures): the net speech then comes from the turns.
type measured struct {
	net        *float64
	overlapped bool
	clipped    *float64
	duration   float64
	turns      [][2]float64 // stored turns of the speaker
	others     [][2]float64 // turns of another local speaker of the same diarization
	benchmark  bool
}

// measuredSpeaker inserts a local speaker and its fingerprint, initially decided as unusable: only the rule
// decides afterwards.
func (h *voiceHarness) measuredSpeaker(clipID int64, label string, degrees float64, m measured) fixtureSpeaker {
	h.t.Helper()
	ctx := context.Background()
	var diarization, fingerprint, speaker int64
	if err := h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.9',$2,'done') RETURNING id`, clipID, "elig-d-"+label).Scan(&diarization); err != nil {
		h.t.Fatal(err)
	}
	if err := h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.fingerprint.9',$2,'done') RETURNING id`, clipID, "elig-f-"+label).Scan(&fingerprint); err != nil {
		h.t.Fatal(err)
	}
	insertSpeaker := func(label string, turns [][2]float64) int64 {
		var id int64
		if err := h.pool.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,9,$2,$3,0,2,$4) RETURNING id`, clipID, diarization, label, len(turns)).Scan(&id); err != nil {
			h.t.Fatal(err)
		}
		for i, turn := range turns {
			if _, err := h.pool.Exec(ctx, `INSERT INTO speaker_turns(local_speaker_id,clip_id,turn_index,start_seconds,end_seconds) VALUES($1,$2,$3,$4,$5)`, id, clipID, i, turn[0], turn[1]); err != nil {
				h.t.Fatal(err)
			}
		}
		return id
	}
	speaker = insertSpeaker(label, m.turns)
	if m.others != nil {
		insertSpeaker(label+"_OTHER", m.others)
	}
	vector, _ := json.Marshal(angle(degrees))
	var version *int
	var overlapped *bool
	if m.net != nil {
		one := 1
		version, overlapped = &one, &m.overlapped
	}
	if _, err := h.pool.Exec(ctx, `INSERT INTO voice_fingerprints(local_speaker_id,clip_id,fingerprint_job_id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,unusable_reason,contributive,model,model_version,dimensions,embedding,benchmark,measures_version,net_speech_seconds,overlap_ratio,overlapped,clipped_ratio,net_speech_source)
 VALUES($1,$2,$3,'speaker',0,$4,$4,0.2,false,'too_short',false,'fixture-embedding','emb-v1',4,$5::vector,$6,$7,$8,CASE WHEN $7::int IS NULL THEN NULL ELSE 0 END,$9,$10,CASE WHEN $7::int IS NULL THEN NULL ELSE 'worker' END)`,
		speaker, clipID, fingerprint, m.duration, string(vector), m.benchmark, version, m.net, overlapped, m.clipped); err != nil {
		h.t.Fatal(err)
	}
	return fixtureSpeaker{ID: speaker, ClipID: clipID}
}

type eligibilityView struct {
	Usable                 bool     `json:"usable"`
	Contributive           bool     `json:"contributive"`
	UnusableReason         string   `json:"unusable_reason"`
	EligibilityRuleVersion *int     `json:"eligibility_rule_version"`
	NetSpeechSeconds       *float64 `json:"net_speech_seconds"`
	NetSpeechSource        *string  `json:"net_speech_source"`
}

// eligibility reads the speaker-scope fingerprint of a speaker through the clip voice API.
func (h *voiceHarness) eligibility(sp fixtureSpeaker) eligibilityView {
	h.t.Helper()
	var view struct {
		Speakers []struct {
			ID           int64 `json:"id"`
			Fingerprints []struct {
				Scope string `json:"scope"`
				eligibilityView
			} `json:"fingerprints"`
		} `json:"speakers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", sp.ClipID), nil, http.StatusOK), &view); err != nil {
		h.t.Fatal(err)
	}
	for _, s := range view.Speakers {
		for _, f := range s.Fingerprints {
			if s.ID == sp.ID && f.Scope == "speaker" {
				return f.eligibilityView
			}
		}
	}
	h.t.Fatalf("no speaker fingerprint for speaker %d", sp.ID)
	return eligibilityView{}
}

type eligibilityRuleView struct {
	Version                int     `json:"version"`
	ComparableMinSeconds   float64 `json:"comparable_min_seconds"`
	ContributiveMinSeconds float64 `json:"contributive_min_seconds"`
	MaxClippedRatio        float64 `json:"max_clipped_ratio"`
	Author                 *string `json:"author"`
	Provisional            bool    `json:"provisional"`
}

type eligibilitySettingsView struct {
	Rule        eligibilityRuleView   `json:"rule"`
	History     []eligibilityRuleView `json:"history"`
	Requalified *int                  `json:"requalified"`
	Summary     struct {
		Comparable       int `json:"comparable"`
		ContributiveOnly int `json:"contributive_only"`
		TooShort         int `json:"too_short"`
		OverlapOnly      int `json:"overlap_only"`
		Clipped          int `json:"clipped"`
		Stale            int `json:"stale"`
	} `json:"summary"`
}

func (h *voiceHarness) putEligibility(comparable, contributive, clipped float64) eligibilitySettingsView {
	h.t.Helper()
	var out eligibilitySettingsView
	body := map[string]any{"comparable_min_seconds": comparable, "contributive_min_seconds": contributive, "max_clipped_ratio": clipped}
	if err := json.Unmarshal(h.adminCall(http.MethodPut, "/api/admin/voice-eligibility", body, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out
}

func testVoiceEligibility(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]

	// The installation starts from a provisional rule set by Snitch, not by an administrator.
	var initial eligibilitySettingsView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-eligibility", nil, http.StatusOK), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.Rule.Version != 1 || initial.Rule.Author != nil || !initial.Rule.Provisional || initial.Rule.ComparableMinSeconds <= initial.Rule.ContributiveMinSeconds {
		t.Fatalf("unexpected initial rule: %+v", initial.Rule)
	}

	reference := h.measuredSpeaker(clipA, "REF", 0, measured{net: ptr(6.0), duration: 6})
	short := h.measuredSpeaker(clipB, "SHORT", 10, measured{net: ptr(2.0), duration: 2})
	overlapped := h.measuredSpeaker(clipB, "OVERLAP", 5, measured{net: ptr(0.0), overlapped: true, duration: 4})
	clipped := h.measuredSpeaker(clipC, "CLIPPED", 5, measured{net: ptr(5.0), clipped: ptr(0.2), duration: 5})
	// Without measures: net speech from the stored turns (3 s of turns, 1 s spoken over by another speaker)...
	fromTurns := h.measuredSpeaker(clipC, "TURNS", 170, measured{duration: 3, turns: [][2]float64{{0, 3}}, others: [][2]float64{{2, 3}}})
	// ...or, for a validated speaker whose turns were dropped by a later diarization, the stored duration.
	archived := h.measuredSpeaker(clipC, "ARCHIVED", 175, measured{duration: 4})
	bench := h.measuredSpeaker(clipC, "BENCH", 5, measured{net: ptr(6.0), duration: 6, benchmark: true})
	h.validate(reference, "Alice")

	// Changing the rule requalifies every production fingerprint in the database, without any job.
	jobs := h.count(`SELECT count(*) FROM jobs`)
	strict := h.putEligibility(3, 1.5, 0.05)
	if strict.Rule.Version != 2 || strict.Rule.Author == nil || *strict.Rule.Author != "administrator" || strict.Rule.Provisional {
		t.Fatalf("the change is not versioned and traced: %+v", strict.Rule)
	}
	if h.count(`SELECT count(*) FROM jobs`) != jobs {
		t.Fatal("requalifying the fingerprints created jobs")
	}
	got := h.eligibility(short)
	if got.Usable || !got.Contributive || got.UnusableReason != "too_short" || got.EligibilityRuleVersion == nil || *got.EligibilityRuleVersion != 2 {
		t.Fatalf("2 s of net speech under a 3 s rule must be contributive only: %+v", got)
	}
	if m := h.matched(short); m.Attribution != nil {
		t.Fatalf("a fingerprint that is not comparable is never matched: %+v", m)
	}

	// A 2 s local speaker is comparable under a 1.5 s rule, with the same score thresholds.
	lenient := h.putEligibility(1.5, 1, 0.05)
	if lenient.Rule.Version != 3 || len(lenient.History) != 3 || lenient.History[0].Version != 3 {
		t.Fatalf("each change is a new version in the history: %+v", lenient)
	}
	got = h.eligibility(short)
	if !got.Usable || !got.Contributive || got.UnusableReason != "" || *got.EligibilityRuleVersion != 3 || got.NetSpeechSource == nil || *got.NetSpeechSource != "worker" {
		t.Fatalf("2 s of net speech under a 1.5 s rule must be comparable: %+v", got)
	}
	if m := h.matched(short); m.Person == nil || m.DecisionLevel != "suggestion" {
		t.Fatalf("the requalified fingerprint is matched again: %+v", m)
	}

	// An overlapped fingerprint is never contributive, whatever the durations; a clipped one neither.
	if got = h.eligibility(overlapped); got.Usable || got.Contributive || got.UnusableReason != "overlap_only" {
		t.Fatalf("overlapped fingerprint: %+v", got)
	}
	if got = h.eligibility(clipped); got.Usable || got.Contributive || got.UnusableReason != "clipped" {
		t.Fatalf("clipped fingerprint: %+v", got)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE voice_fingerprints SET contributive=true WHERE local_speaker_id=$1`, overlapped.ID); err == nil {
		t.Fatal("the database must refuse a contributive overlapped fingerprint")
	}

	// Without measures, the net speech comes from the stored turns, and is marked as such.
	got = h.eligibility(fromTurns)
	if got.NetSpeechSource == nil || *got.NetSpeechSource != "turns" || got.NetSpeechSeconds == nil || *got.NetSpeechSeconds != 2 || !got.Usable {
		t.Fatalf("net speech from the turns: %+v", got)
	}
	// Without turns, the stored duration is an estimate, visible as one: a validated reference is never
	// rejected for the mere lack of turns.
	got = h.eligibility(archived)
	if got.NetSpeechSource == nil || *got.NetSpeechSource != "estimated" || got.NetSpeechSeconds == nil || *got.NetSpeechSeconds != 4 || !got.Usable {
		t.Fatalf("estimated net speech: %+v", got)
	}

	// Benchmark fingerprints keep the decision they were stored with.
	var benchUsable bool
	var benchVersion *int
	if err := pool.QueryRow(context.Background(), `SELECT usable,eligibility_rule_version FROM voice_fingerprints WHERE local_speaker_id=$1`, bench.ID).Scan(&benchUsable, &benchVersion); err != nil || benchUsable || benchVersion != nil {
		t.Fatalf("benchmark fingerprint requalified: usable=%v version=%v err=%v", benchUsable, benchVersion, err)
	}

	// Saving the same rule creates no version; the summary counts the production speaker fingerprints.
	same := h.putEligibility(1.5, 1, 0.05)
	if same.Rule.Version != 3 || same.Requalified == nil || *same.Requalified != 0 {
		t.Fatalf("an unchanged rule is not a new version: %+v", same.Rule)
	}
	if s := same.Summary; s.Comparable != 4 || s.OverlapOnly != 1 || s.Clipped != 1 || s.TooShort != 0 || s.Stale != 0 {
		t.Fatalf("summary: %+v", s)
	}

	// Only an administrator changes the rule, within bounds.
	peer := h.newUser("peer")
	h.userCall(peer, http.MethodGet, "/api/admin/voice-eligibility", nil, http.StatusForbidden)
	h.userCall(peer, http.MethodPut, "/api/admin/voice-eligibility", map[string]any{"comparable_min_seconds": 1, "contributive_min_seconds": 0.5, "max_clipped_ratio": 0.05}, http.StatusForbidden)
	for _, bad := range []map[string]any{
		{"comparable_min_seconds": 1, "contributive_min_seconds": 2, "max_clipped_ratio": 0.05},
		{"comparable_min_seconds": 0.2, "contributive_min_seconds": 0.2, "max_clipped_ratio": 0.05},
		{"comparable_min_seconds": 2, "contributive_min_seconds": 1, "max_clipped_ratio": 1.5},
		{"comparable_min_seconds": 2, "contributive_min_seconds": 1},
	} {
		h.adminCall(http.MethodPut, "/api/admin/voice-eligibility", bad, http.StatusBadRequest)
	}
}
