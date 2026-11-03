//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Disposition of a local speaker (ignored or random) and refusal of an identification
// (ADR 0031). The rules are read through the public API only: verification, groups,
// clip voices, transcript and matching.

type dispositionSpeaker struct {
	ID          int64   `json:"id"`
	Disposition *string `json:"disposition"`
	State       string  `json:"state"`
}

func (h *voiceHarness) dispositionOf(sp fixtureSpeaker) *string {
	h.t.Helper()
	var view struct {
		Speakers []dispositionSpeaker `json:"speakers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", sp.ClipID), nil, http.StatusOK), &view); err != nil {
		h.t.Fatal(err)
	}
	for _, s := range view.Speakers {
		if s.ID == sp.ID {
			return s.Disposition
		}
	}
	h.t.Fatalf("speaker %d not listed", sp.ID)
	return nil
}

func (h *voiceHarness) setDisposition(sp fixtureSpeaker, disposition string, want int) []byte {
	h.t.Helper()
	return h.adminCall(http.MethodPut, fmt.Sprintf("/api/local-speakers/%d/disposition", sp.ID), map[string]any{"disposition": disposition}, want)
}

func (h *voiceHarness) restoreDisposition(sp fixtureSpeaker) {
	h.t.Helper()
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/local-speakers/%d/disposition", sp.ID), nil, http.StatusOK)
}

// inVerification reports whether the voice is listed in the review queue of the caller.
func (h *voiceHarness) inVerification(u *voiceUser, sp fixtureSpeaker) bool {
	h.t.Helper()
	var page struct {
		Items []struct {
			LocalSpeakerID int64 `json:"local_speaker_id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/voice-verification?page_size=100", nil, http.StatusOK), &page); err != nil {
		h.t.Fatal(err)
	}
	for _, c := range page.Items {
		if c.LocalSpeakerID == sp.ID {
			return true
		}
	}
	return false
}

func (h *voiceHarness) journalCount(sp fixtureSpeaker) int {
	h.t.Helper()
	return h.count(`SELECT count(*) FROM voice_disposition_events WHERE local_speaker_id=$1`, sp.ID)
}

// testVoiceDisposition covers the ignored and random dispositions: their effects on
// verification, groups, rapprochement, clip voices and transcript, reversibility, the
// journal, the group disposition and the rights.
func testVoiceDisposition(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	h.groupSettings()
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	alice := h.newUser("alice")

	reference := h.fixtureSpeaker(clipA, "REF", usableAt(0))
	h.validate(reference, "Alice")
	aliceID := h.personOf(reference)
	// A voice close to the reference of Alice (a suggestion), an unknown voice and its neighbour.
	near := h.fixtureSpeaker(clipB, "NEAR", usableAt(10))
	noise := h.fixtureSpeaker(clipC, "NOISE", usableAt(200))
	neighbour := h.fixtureSpeaker(clipC, "NEIGHBOUR", usableAt(205))
	h.rematch()
	if got := h.matched(near); got.DecisionLevel != "suggestion" {
		t.Fatalf("the close voice starts as a suggestion: %+v", got)
	}
	if !h.inVerification(alice, noise) || h.groups(alice, "").containing(noise.ID).MemberCount != 2 {
		t.Fatalf("the noise voice starts in verification and in a group of two")
	}

	// Ignoring a voice removes it from verification, groups and rapprochement at once.
	h.setDisposition(noise, "ignored", http.StatusOK)
	if got := h.dispositionOf(noise); got == nil || *got != "ignored" {
		t.Fatalf("the disposition is listed on the clip voices: %v", got)
	}
	if h.inVerification(alice, noise) {
		t.Fatal("an ignored voice must leave the verification queue")
	}
	if g := h.groups(alice, "").containing(noise.ID); g != nil {
		t.Fatalf("an ignored voice must leave the groups: %+v", g)
	}
	if g := h.groups(alice, "").containing(neighbour.ID); g == nil || g.MemberCount != 1 {
		t.Fatalf("its former neighbour is alone in its group: %+v", g)
	}
	if h.journalCount(noise) != 1 {
		t.Fatalf("the disposition is journaled once")
	}
	// Ignoring the same voice again changes nothing and writes nothing.
	h.setDisposition(noise, "ignored", http.StatusOK)
	if h.journalCount(noise) != 1 {
		t.Fatal("an unchanged disposition must not be journaled again")
	}

	// Rapprochement skips an ignored voice and its segments have no speaker.
	ignored := fixtureSpeaker{ID: near.ID, ClipID: near.ClipID}
	h.setDisposition(ignored, "ignored", http.StatusOK)
	h.rematch()
	if got := h.matched(ignored); got.DecisionLevel != "unknown" || got.Attribution != nil || got.Person != nil {
		t.Fatalf("an ignored voice receives no suggestion: %+v", got)
	}

	// Restoring the voice gives it back to verification and, once matched again, its suggestion.
	h.restoreDisposition(noise)
	if d := h.dispositionOf(noise); d != nil {
		t.Fatalf("restoring clears the disposition: %v", *d)
	}
	if !h.inVerification(alice, noise) {
		t.Fatal("a restored voice returns to verification")
	}
	if h.journalCount(noise) != 2 {
		t.Fatalf("restoring is journaled as well: %d", h.journalCount(noise))
	}
	h.restoreDisposition(ignored)
	h.rematch()
	if got := h.matched(ignored); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != aliceID {
		t.Fatalf("a restored voice is matched again: %+v", got)
	}

	// Random: the voice leaves verification, groups and rapprochement, but keeps its label.
	h.setDisposition(noise, "random", http.StatusOK)
	if got := h.dispositionOf(noise); got == nil || *got != "random" {
		t.Fatalf("random is listed on the clip voices: %v", got)
	}
	if h.inVerification(alice, noise) || h.groups(alice, "").containing(noise.ID) != nil {
		t.Fatal("a random voice leaves verification and groups")
	}
	h.setDisposition(noise, "ignored", http.StatusOK)
	h.restoreDisposition(noise)

	// A validated voice cannot be ignored or made random: it must be refused first.
	h.setDisposition(reference, "ignored", http.StatusConflict)
	h.setDisposition(reference, "random", http.StatusConflict)
	h.setDisposition(noise, "unknown", http.StatusBadRequest)
	h.adminCall(http.MethodPut, "/api/local-speakers/999999/disposition", map[string]any{"disposition": "ignored"}, http.StatusNotFound)

	// Another user who can see the clip has the same right as for a validation.
	h.userCall(alice, http.MethodPut, fmt.Sprintf("/api/local-speakers/%d/disposition", noise.ID), map[string]any{"disposition": "random"}, http.StatusOK)
	if d := h.dispositionOf(noise); d == nil || *d != "random" {
		t.Fatalf("a signed-in user can set the disposition: %v", d)
	}
	h.restoreDisposition(noise)

	// A group disposition applies to each of its voices.
	group := h.groups(alice, "").containing(neighbour.ID)
	if group == nil || group.MemberCount != 2 {
		t.Fatalf("the two unknown voices form a group: %+v", group)
	}
	h.userCall(alice, http.MethodPut, fmt.Sprintf("/api/voice-groups/%d/disposition", group.ID), map[string]any{"disposition": "random"}, http.StatusOK)
	for _, sp := range []fixtureSpeaker{noise, neighbour} {
		if d := h.dispositionOf(sp); d == nil || *d != "random" {
			t.Fatalf("each voice of the group is random: %v", d)
		}
	}
	h.userCall(alice, http.MethodPut, "/api/voice-groups/999999/disposition", map[string]any{"disposition": "ignored"}, http.StatusNotFound)
	h.userCall(alice, http.MethodPut, fmt.Sprintf("/api/voice-groups/%d/disposition", group.ID), map[string]any{"disposition": "bogus"}, http.StatusBadRequest)
	for _, sp := range []fixtureSpeaker{noise, neighbour} {
		h.restoreDisposition(sp)
	}
	if h.dispositionOf(noise) != nil || h.dispositionOf(neighbour) != nil {
		t.Fatal("restoring the voices of a group clears their dispositions")
	}
}

// testVoiceDispositionTranscript checks the transcript: an ignored voice has no speaker
// on its segments, a random voice is labelled "Random" and carries no identity.
func testVoiceDispositionTranscript(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip()
	h.setDisposition(fixtureSpeaker{ID: first, ClipID: h.clipID}, "ignored", http.StatusOK)
	h.setDisposition(fixtureSpeaker{ID: second, ClipID: h.clipID}, "random", http.StatusOK)
	var v speakerTranscript
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &v); err != nil {
		t.Fatal(err)
	}
	labelled := 0
	for _, seg := range v.Transcription.Segments {
		if seg.SpeakerID == first {
			t.Fatalf("an ignored voice must not label its segments: %+v", seg)
		}
		if seg.SpeakerID == second {
			labelled++
			if seg.Label != "Random" || seg.PersonName != "" {
				t.Fatalf("a random voice is labelled Random without identity: %+v", seg)
			}
		}
	}
	if labelled == 0 {
		t.Fatalf("the random voice must still label its segments: %+v", v.Transcription.Segments)
	}
	for _, speaker := range v.Transcription.Speakers {
		if id, _ := speaker["id"].(float64); int64(id) == first {
			t.Fatal("an ignored voice is not offered as a speaker of a correction")
		}
	}
}

// testVoiceRefusal covers "ce n'est pas X": the attribution is removed, the voice goes
// back to unknown, its validated samples leave the references, the refusal is journaled,
// and no later rapprochement proposes that Person for that voice.
func testVoiceRefusal(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	h.groupSettings()
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	reference := h.fixtureSpeaker(clipA, "REF", usableAt(0))
	h.validate(reference, "Alice")
	alice := h.personOf(reference)
	near := h.fixtureSpeaker(clipB, "NEAR", usableAt(10))
	validated := h.fixtureSpeaker(clipC, "VALIDATED", usableAt(0))
	h.rematch()
	if got := h.matched(near); got.Person == nil || got.Person.ID != alice {
		t.Fatalf("the close voice is suggested to Alice before the refusal: %+v", got)
	}

	// Refusing a suggestion removes it, and no later rematch proposes Alice again.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", near.ID), map[string]any{"person_id": alice}, http.StatusOK)
	if got := h.matched(near); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("a refused suggestion is removed: %+v", got)
	}
	h.rematch()
	if got := h.matched(near); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("a refused Person is never proposed again for the voice: %+v", got)
	}
	if n := h.count(`SELECT count(*) FROM voice_refusals WHERE local_speaker_id=$1 AND person_id=$2`, near.ID, alice); n != 1 {
		t.Fatalf("the refusal is recorded once: %d", n)
	}

	// Refusing a validated attribution also takes its samples out of the references.
	h.validate(validated, alice)
	if n := h.count(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=$1 AND retired_at IS NULL`, validated.ID); n != 1 {
		t.Fatalf("the validated voice has its reference sample: %d", n)
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", validated.ID), map[string]any{"person_id": alice}, http.StatusOK)
	if got := h.matched(validated); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("a refused validation leaves the voice unknown: %+v", got)
	}
	if n := h.count(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=$1`, validated.ID); n != 0 {
		t.Fatalf("the samples of a refused validation leave the references: %d", n)
	}
	h.rematch()
	if got := h.matched(validated); got.Person != nil {
		t.Fatalf("the refused Person is not proposed after the validation was refused: %+v", got)
	}

	// The refusal is in the history of the voice.
	var history struct {
		Items []struct {
			Action string `json:"action"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/local-speakers/%d/history", validated.ID), nil, http.StatusOK), &history); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range history.Items {
		found = found || e.Action == "refuse"
	}
	if !found {
		t.Fatalf("the refusal is journaled in the history: %+v", history.Items)
	}

	// Refusing an unknown Person is a not found; a refusal does not block a later
	// explicit validation by a human.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", near.ID), map[string]any{"person_id": 999999}, http.StatusNotFound)
	h.validate(near, alice)
	if got := h.matched(near); got.DecisionLevel != "human" || got.Person == nil || got.Person.ID != alice {
		t.Fatalf("a human may still validate a refused Person: %+v", got)
	}
}
