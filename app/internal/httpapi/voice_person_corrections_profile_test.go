//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

type verificationSummaryView struct {
	Items []struct {
		LocalSpeakerID int64  `json:"local_speaker_id"`
		DecisionLevel  string `json:"decision_level"`
	} `json:"items"`
	Counts map[string]int `json:"counts"`
}

func (h *voiceHarness) verificationSummary() (verificationSummaryView, int) {
	h.t.Helper()
	var v verificationSummaryView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/voice-verification?page_size=100", nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	sum := 0
	for _, n := range v.Counts {
		sum += n
	}
	return v, sum
}

// Corrections next to the voice profile deletion (voice profile deletion) and the verification view
// (review view): an opt-out is never dissolved or rebuilt by a merge, a deletion removes
// it without touching its immutable trace, and the verification counters stay true.
func testPersonCorrectionsWithProfiles(t *testing.T, h *voiceHarness, member *voiceUser) {
	clipA, clipB := h.clipIDs[0], h.clipIDs[1]
	ginaVoice := h.fixtureSpeaker(clipA, "GINA_VOICE", usableAt(225))
	hugoVoice := h.fixtureSpeaker(clipB, "HUGO_VOICE", usableAt(315))
	h.validate(ginaVoice, "Gina")
	h.validate(hugoVoice, "Hugo")
	gina, hugo := h.personOf(ginaVoice), h.personOf(hugoVoice)
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/sources/%d/microphone-track", h.sourceID()), map[string]any{"title": "Micro", "person_id": gina}, http.StatusOK)
	if m := h.micOf(clipA); m.Status != "designated" {
		t.Fatalf("Gina owns the microphone track: %+v", m)
	}
	_, total := h.verificationSummary()
	if total != h.count(`SELECT count(*) FROM local_speakers`) {
		t.Fatalf("verification counters must cover every voice: %d", total)
	}

	// Gina deletes her voice profile: the opt-out is on the Person.
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/persons/%d/voice-profile", gina), nil, http.StatusOK)
	if m := h.micOf(clipA); m.Status != "owner_profile_deleted" {
		t.Fatalf("a designation naming an opted-out Person is not applied: %+v", m)
	}
	// Neither direction of a merge may dissolve or rebuild the opt-out.
	h.userCall(member, http.MethodPost, personPath(hugo, "merge"), map[string]any{"target_person_id": gina}, http.StatusConflict)
	h.userCall(member, http.MethodPost, personPath(gina, "merge"), map[string]any{"target_person_id": hugo}, http.StatusConflict)
	if !h.personExists(gina) || !h.personExists(hugo) || h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1 AND retired_at IS NULL`, hugo) != 1 || h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1`, gina) != 0 {
		t.Fatal("a refused merge must change nothing and an opted-out Person holds no reference")
	}

	// Separating a voice keeps the verification view true: the voice is a case again.
	before, beforeTotal := h.verificationSummary()
	h.userCall(member, http.MethodPost, personPath(hugo, "separation"), map[string]any{"local_speaker_ids": []int64{hugoVoice.ID}}, http.StatusOK)
	after, afterTotal := h.verificationSummary()
	if after.Counts["human"] != before.Counts["human"]-1 || afterTotal != beforeTotal || afterTotal != h.count(`SELECT count(*) FROM local_speakers`) {
		t.Fatalf("verification counters after separation: %+v then %+v", before.Counts, after.Counts)
	}
	if after.Counts["human"] != h.count(`SELECT count(*) FROM speaker_attributions WHERE validation_state='validated'`) {
		t.Fatalf("the human counter must be the validated voices: %+v", after.Counts)
	}
	listed := false
	for _, c := range after.Items {
		listed = listed || (c.LocalSpeakerID == hugoVoice.ID && c.DecisionLevel == "unknown")
	}
	if !listed {
		t.Fatalf("a separated voice is an unknown case to verify: %+v", after.Items)
	}

	// Deleting the opted-out Person works; its trace stays, without any voice data.
	traces := h.count(`SELECT count(*) FROM voice_profile_deletions WHERE person_id=$1`, gina)
	if traces == 0 {
		t.Fatal("the profile deletion must have left a trace")
	}
	h.userCall(member, http.MethodDelete, personPath(gina, ""), nil, http.StatusOK)
	if h.personExists(gina) || h.count(`SELECT count(*) FROM voice_profile_deletions WHERE person_id=$1`, gina) != traces {
		t.Fatal("the Person goes, the immutable trace of the profile deletion stays")
	}
	if m := h.micOf(clipA); m.Status != "none" {
		t.Fatalf("the rule naming the deleted Person is removed: %+v", m)
	}
	final, finalTotal := h.verificationSummary()
	if finalTotal != h.count(`SELECT count(*) FROM local_speakers`) || final.Counts["human"] != h.count(`SELECT count(*) FROM speaker_attributions WHERE validation_state='validated'`) {
		t.Fatalf("verification counters after deletion: %+v", final.Counts)
	}
}
