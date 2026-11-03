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
	// The panel sent what it displayed: a group that is no longer exactly that set is refused
	// (group_changed) and no voice is touched.
	groupPath := fmt.Sprintf("/api/voice-groups/%d/disposition", group.ID)
	h.userCall(alice, http.MethodPut, groupPath, map[string]any{"disposition": "random", "local_speaker_ids": []int64{noise.ID}}, http.StatusConflict)
	h.userCall(alice, http.MethodPut, groupPath, map[string]any{"disposition": "random", "local_speaker_ids": []int64{noise.ID, neighbour.ID, 999999}}, http.StatusConflict)
	if h.dispositionOf(noise) != nil || h.dispositionOf(neighbour) != nil {
		t.Fatal("a stale group disposition touches no voice")
	}
	h.userCall(alice, http.MethodPut, groupPath, map[string]any{"disposition": "random"}, http.StatusBadRequest)
	both := []int64{noise.ID, neighbour.ID}
	h.userCall(alice, http.MethodPut, groupPath, map[string]any{"disposition": "random", "local_speaker_ids": both}, http.StatusOK)
	for _, sp := range []fixtureSpeaker{noise, neighbour} {
		if d := h.dispositionOf(sp); d == nil || *d != "random" {
			t.Fatalf("each voice of the group is random: %v", d)
		}
	}
	h.userCall(alice, http.MethodPut, "/api/voice-groups/999999/disposition", map[string]any{"disposition": "ignored", "local_speaker_ids": both}, http.StatusNotFound)
	h.userCall(alice, http.MethodPut, groupPath, map[string]any{"disposition": "bogus", "local_speaker_ids": both}, http.StatusBadRequest)
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

// testVoiceDispositionSurvivesRediarization: a new diarization replaces the local speakers
// and drops their dispositions and refusals with them, but the journal stays. Each dropped
// disposition is recorded once, by the system, and its event keeps the clip.
func testVoiceDispositionSurvivesRediarization(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip()
	h.setDisposition(fixtureSpeaker{ID: first, ClipID: h.clipID}, "ignored", http.StatusOK)
	h.setDisposition(fixtureSpeaker{ID: second, ClipID: h.clipID}, "random", http.StatusOK)
	if got := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1`, h.clipID); got != 2 {
		t.Fatalf("two journal events before the rediarization, got %d", got)
	}

	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("expected the relaunched diarization: %+v", job)
	}
	h.complete(job, diarizationPayload("NEW_A", "NEW_B"), http.StatusOK)

	if got := h.count(`SELECT count(*) FROM local_speaker_dispositions`); got != 0 {
		t.Fatalf("dispositions go with the dropped voices, got %d", got)
	}
	if got := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1`, h.clipID); got != 4 {
		t.Fatalf("the journal must survive the rediarization (2 before, 2 system events), got %d", got)
	}
	if got := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1 AND local_speaker_id IS NULL`, h.clipID); got != 4 {
		t.Fatalf("the journal rows of dropped voices keep their clip with a NULL speaker, got %d", got)
	}
	if got := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1 AND author_username='system' AND author_id IS NULL AND new_disposition='' AND old_disposition IN ('ignored','random')`, h.clipID); got != 2 {
		t.Fatalf("one system event per dropped disposition, got %d", got)
	}
}

// testRandomCountsInVerificationCounters: a random voice stays in the counters of the
// verification (ADR 0031 excludes it from verification, groups and rapprochement only); an
// ignored voice leaves them.
func testRandomCountsInVerificationCounters(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip()
	admin := h.adminUser()
	total := func() int {
		t.Helper()
		var page struct {
			Counts map[string]int `json:"counts"`
		}
		if err := json.Unmarshal(h.userCall(admin, http.MethodGet, "/api/voice-verification", nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		sum := 0
		for _, n := range page.Counts {
			sum += n
		}
		return sum
	}
	before := total()
	if before != 2 {
		t.Fatalf("two unknown voices start in the counters, got %d", before)
	}
	h.setDisposition(fixtureSpeaker{ID: first, ClipID: h.clipID}, "random", http.StatusOK)
	if got := total(); got != before {
		t.Fatalf("a random voice stays in the counters: %d, want %d", got, before)
	}
	h.setDisposition(fixtureSpeaker{ID: second, ClipID: h.clipID}, "ignored", http.StatusOK)
	if got := total(); got != before-1 {
		t.Fatalf("an ignored voice leaves the counters: %d, want %d", got, before-1)
	}
}

// testIgnoredVoiceNeverDeduced: "retour à l'auto" must not attribute a segment to an
// ignored voice, even when that voice is the one speaking most over it.
func testIgnoredVoiceNeverDeduced(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip() // segment 0 is spoken by first only
	path := fmt.Sprintf("/api/clips/%d/segments/0/speaker", h.clipID)
	var designated speakerResponse
	if err := json.Unmarshal(h.adminCall(http.MethodPut, path, map[string]any{"local_speaker_id": second, "expected_version": 0}, http.StatusOK), &designated); err != nil {
		t.Fatal(err)
	}
	h.setDisposition(fixtureSpeaker{ID: first, ClipID: h.clipID}, "ignored", http.StatusOK)
	var back speakerResponse
	if err := json.Unmarshal(h.adminCall(http.MethodPut, path, map[string]any{"local_speaker_id": nil, "expected_version": designated.Version}, http.StatusOK), &back); err != nil {
		t.Fatal(err)
	}
	if back.ID != nil {
		t.Fatalf("the deduced speaker of segment 0 is ignored, so the segment has none: %+v", back)
	}
}

// testRefusalDuringActiveAnalysis proves that a refusal, like the deletion of analysis steps,
// waits for the analysis: during an active run it is refused with 409 analysis_active and
// writes nothing; once the run is cancelled it is recorded.
func testRefusalDuringActiveAnalysis(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	_, second := h.diarizeClip()
	bob := h.createPerson("Bob")
	h.startAnalysis(map[string]any{"scope": "voice", "mode": "full"}, http.StatusCreated)
	if a := h.analysis(); a == nil || a.Status != "active" {
		t.Fatalf("the fixture needs an active analysis: %+v", a)
	}
	var refused struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", second), map[string]any{"person_id": bob}, http.StatusConflict), &refused); err != nil {
		t.Fatal(err)
	}
	if refused.Error != "analysis_active" {
		t.Fatalf("a refusal during an active analysis must say so: %+v", refused)
	}
	if n := h.count(`SELECT count(*) FROM voice_refusals WHERE local_speaker_id=$1`, second); n != 0 {
		t.Fatalf("a refused refusal must write nothing: %d", n)
	}
	h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusOK)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", second), map[string]any{"person_id": bob}, http.StatusOK)
	if n := h.count(`SELECT count(*) FROM voice_refusals WHERE local_speaker_id=$1`, second); n != 1 {
		t.Fatalf("the refusal is recorded once the analysis is over: %d", n)
	}
}
