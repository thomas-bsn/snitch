//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

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
	if n := h.count(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=$1 AND retired_at IS NULL`, validated.ID); n != 0 {
		t.Fatalf("the samples of a refused validation leave the references: %d", n)
	}
	// They are retired, like a corrected sample, not deleted: the refusal in the history
	// still points to an existing sample.
	if n := h.count(`SELECT count(*) FROM voice_samples vs JOIN attribution_events e ON e.old_sample_id=vs.id
 WHERE e.local_speaker_id=$1 AND e.action='refuse' AND vs.retired_at IS NOT NULL AND vs.retired_reason='refus_identification' AND vs.retired_by IS NOT NULL`, validated.ID); n != 1 {
		t.Fatalf("the sample of a refused validation is retired, and the refusal keeps it: %d", n)
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

// rediarize relaunches the diarization of the harness clip and completes it with speakers
// whose turns are given as [start,end] pairs, then returns the new voices by label.
func (h *voiceHarness) rediarize(speakers map[string][][2]float64, order ...string) map[string]fixtureSpeaker {
	h.t.Helper()
	return h.rediarizeWith(map[string]any{"audio_stream_index": 2}, speakers, order...)
}

// rediarizeWith is rediarize with the body of the relaunch (a confirmation, for instance).
func (h *voiceHarness) rediarizeWith(relaunch map[string]any, speakers map[string][][2]float64, order ...string) map[string]fixtureSpeaker {
	h.t.Helper()
	h.dropPendingFingerprints()
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), relaunch, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		h.t.Fatalf("expected the relaunched diarization: %+v", job)
	}
	items := []map[string]any{}
	for _, label := range order {
		turns := []map[string]any{}
		for _, t := range speakers[label] {
			turns = append(turns, map[string]any{"start": t[0], "end": t[1]})
		}
		items = append(items, map[string]any{"label": label, "turns": turns})
	}
	h.complete(job, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "audio_stream_index": 2, "speakers": items}, http.StatusOK)
	voices := map[string]fixtureSpeaker{}
	for _, sp := range h.voice().Speakers {
		voices[sp.Label] = fixtureSpeaker{ID: sp.ID, ClipID: h.clipID}
	}
	return voices
}

// dropPendingFingerprints takes the fingerprints queued by a rediarization out of the queue:
// the fixtures do not need them, and a pending voice job keeps the analysis of the clip active.
func (h *voiceHarness) dropPendingFingerprints() {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), `UPDATE jobs SET status='failed',last_error='fixture' WHERE clip_id=$1 AND step LIKE 'voice.fingerprint.%' AND status='pending'`, h.clipID); err != nil {
		h.t.Fatal(err)
	}
}

// refusalEvents counts the refusal journal rows of a clip with an action, whose reason
// contains a text.
func (h *voiceHarness) refusalEvents(action, reason string) int {
	h.t.Helper()
	return h.count(`SELECT count(*) FROM voice_refusal_events WHERE clip_id=$1 AND action=$2 AND author_username='system' AND author_id IS NULL AND strpos(reason,$3)>0`, h.clipID, action, reason)
}

// testRediarizationCarriesDispositions: a new diarization carries the disposition and the
// refusals of each old voice to the new voice whose turns cover at least half of its
// speaking time. Every carry is journaled by the system, on both voices. The diarizeClip
// voices speak over [0,1.5] (SPEAKER_00) and [1,2.5] (SPEAKER_01).
func testRediarizationCarriesDispositions(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip()
	bob := h.createPerson("Bob")
	h.setDisposition(fixtureSpeaker{ID: first, ClipID: h.clipID}, "ignored", http.StatusOK)
	h.setDisposition(fixtureSpeaker{ID: second, ClipID: h.clipID}, "random", http.StatusOK)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", second), map[string]any{"person_id": bob}, http.StatusOK)

	// NEW_A covers all of SPEAKER_00 [0,1.5]; NEW_B two thirds of SPEAKER_01 [1,2.5].
	voices := h.rediarize(map[string][][2]float64{"NEW_A": {{0, 1.5}}, "NEW_B": {{1.5, 2.5}}}, "NEW_A", "NEW_B")
	if d := h.dispositionOf(voices["NEW_A"]); d == nil || *d != "ignored" {
		t.Fatalf("the ignored voice is carried to the new voice that covers it: %v", d)
	}
	if d := h.dispositionOf(voices["NEW_B"]); d == nil || *d != "random" {
		t.Fatalf("the random voice is carried to the new voice covering 67%% of it: %v", d)
	}
	if got := h.refusedPersonsOf(voices["NEW_B"]); len(got) != 1 || got[0] != bob {
		t.Fatalf("the refusal follows the voice: %v", got)
	}
	if got := h.refusedPersonsOf(voices["NEW_A"]); len(got) != 0 {
		t.Fatalf("the refusal stays with its own voice: %v", got)
	}
	// Journal: the old voice gives its disposition up, the new one receives it, by the system.
	for _, voice := range []fixtureSpeaker{voices["NEW_A"], voices["NEW_B"]} {
		if n := h.count(`SELECT count(*) FROM voice_disposition_events WHERE local_speaker_id=$1 AND old_disposition='' AND author_username='system' AND author_id IS NULL AND strpos(reason,'Reprise de la voix SPEAKER_0')>0`, voice.ID); n != 1 {
			t.Fatalf("the carried disposition is journaled on the new voice: %d", n)
		}
	}
	if n := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1 AND new_disposition='' AND author_username='system' AND strpos(reason,'Reportée sur la voix NEW_')>0`, h.clipID); n != 2 {
		t.Fatalf("each carried disposition is journaled on its old voice: %d", n)
	}
	if n := h.refusalEvents("carry", "Repris de la voix SPEAKER_01"); n != 1 {
		t.Fatalf("the carried refusal is journaled: %d", n)
	}
	if n := h.refusalEvents("drop", ""); n != 0 {
		t.Fatalf("nothing is dropped: %d", n)
	}
}

// testRediarizationDropsUncoveredDispositions: an old voice no new voice covers for at
// least half of its speaking time loses its disposition and its refusals, journaled.
func testRediarizationDropsUncoveredDispositions(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, _ := h.diarizeClip()
	bob := h.createPerson("Bob")
	h.setDisposition(fixtureSpeaker{ID: first, ClipID: h.clipID}, "ignored", http.StatusOK)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", first), map[string]any{"person_id": bob}, http.StatusOK)

	// SPEAKER_00 [0,1.5] is split in two thirds: 40 % each, below half.
	voices := h.rediarize(map[string][][2]float64{"NEW_A": {{0, 0.6}}, "NEW_B": {{0.6, 1.2}}, "NEW_C": {{1.2, 2.5}}}, "NEW_A", "NEW_B", "NEW_C")
	for _, voice := range voices {
		if d := h.dispositionOf(voice); d != nil {
			t.Fatalf("no new voice covers half of the old one: %v", *d)
		}
		if got := h.refusedPersonsOf(voice); len(got) != 0 {
			t.Fatalf("the refusal is dropped: %v", got)
		}
	}
	if n := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1 AND old_disposition='ignored' AND new_disposition='' AND author_username='system' AND strpos(reason,'moins de la moitié')>0`, h.clipID); n != 1 {
		t.Fatalf("the dropped disposition is journaled with its reason: %d", n)
	}
	if n := h.refusalEvents("drop", "moins de la moitié"); n != 1 {
		t.Fatalf("the dropped refusal is journaled with its reason: %d", n)
	}
	if n := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1 AND local_speaker_id IS NULL`, h.clipID); n != 2 {
		t.Fatalf("the journal of the dropped voice outlives it: %d", n)
	}
}

// testRediarizationConflictDropsDispositions: two old voices merged into one new voice
// with different dispositions (a voice without disposition included) keep none of them;
// their refusals are merged.
func testRediarizationConflictDropsDispositions(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip()
	bob, carol := h.createPerson("Bob"), h.createPerson("Carol")
	h.setDisposition(fixtureSpeaker{ID: first, ClipID: h.clipID}, "ignored", http.StatusOK)
	h.setDisposition(fixtureSpeaker{ID: second, ClipID: h.clipID}, "random", http.StatusOK)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", first), map[string]any{"person_id": bob}, http.StatusOK)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", second), map[string]any{"person_id": carol}, http.StatusOK)

	voices := h.rediarize(map[string][][2]float64{"MERGED": {{0, 2.5}}}, "MERGED")
	if d := h.dispositionOf(voices["MERGED"]); d != nil {
		t.Fatalf("conflicting dispositions are dropped: %v", *d)
	}
	got := h.refusedPersonsOf(voices["MERGED"])
	if len(got) != 2 || !((got[0] == bob && got[1] == carol) || (got[0] == carol && got[1] == bob)) {
		t.Fatalf("the refusals are merged: %v", got)
	}
	if n := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1 AND new_disposition='' AND author_username='system' AND strpos(reason,'conflit')>0`, h.clipID); n != 2 {
		t.Fatalf("each conflicting disposition is dropped with its reason: %d", n)
	}
	if n := h.refusalEvents("carry", ""); n != 2 {
		t.Fatalf("each merged refusal is journaled: %d", n)
	}

	// A voice without disposition merged with an ignored one: ignoring the merged voice
	// would hide the other, so the disposition is dropped too.
	split := h.rediarize(map[string][][2]float64{"P": {{0, 1.5}}, "Q": {{1.5, 2.5}}}, "P", "Q")
	h.dropPendingFingerprints()
	h.setDisposition(split["P"], "ignored", http.StatusOK)
	voices = h.rediarize(map[string][][2]float64{"ONE": {{0, 2.5}}}, "ONE")
	if d := h.dispositionOf(voices["ONE"]); d != nil {
		t.Fatalf("an ignored voice merged with a normal one is not carried: %v", *d)
	}
	if n := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1 AND old_disposition='ignored' AND new_disposition='' AND strpos(reason,'conflit')>0`, h.clipID); n != 2 {
		t.Fatalf("the second conflict is journaled: %d", n)
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

// refusedPersonsOf lists the Persons refused for a voice, as the clip voices show them.
func (h *voiceHarness) refusedPersonsOf(sp fixtureSpeaker) []int64 {
	h.t.Helper()
	var view struct {
		Speakers []struct {
			ID             int64 `json:"id"`
			RefusedPersons []struct {
				ID int64 `json:"id"`
			} `json:"refused_persons"`
		} `json:"speakers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", sp.ClipID), nil, http.StatusOK), &view); err != nil {
		h.t.Fatal(err)
	}
	for _, s := range view.Speakers {
		if s.ID == sp.ID {
			ids := []int64{}
			for _, p := range s.RefusedPersons {
				ids = append(ids, p.ID)
			}
			return ids
		}
	}
	h.t.Fatalf("speaker %d not listed", sp.ID)
	return nil
}

// testVoiceRefusalCancel covers "Annuler le refus": a refusal is reversible with the same
// right as the refusal, the cancellation is journaled, and matching may propose the Person
// for that voice again.
func testVoiceRefusalCancel(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	h.groupSettings()
	reference := h.fixtureSpeaker(h.clipIDs[0], "REF", usableAt(0))
	h.validate(reference, "Alice")
	alice := h.personOf(reference)
	near := h.fixtureSpeaker(h.clipIDs[1], "NEAR", usableAt(10))
	h.rematch()
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", near.ID), map[string]any{"person_id": alice}, http.StatusOK)
	if got := h.refusedPersonsOf(near); len(got) != 1 || got[0] != alice {
		t.Fatalf("the refused Person is listed on the voice: %v", got)
	}
	h.rematch()
	if got := h.matched(near); got.Person != nil {
		t.Fatalf("the refused Person is not proposed: %+v", got)
	}

	// A signed-in user who sees the clip has the right of the refusal.
	user := h.newUser("bob")
	cancel := fmt.Sprintf("/api/local-speakers/%d/refusals/%d", near.ID, alice)
	var view struct {
		LocalSpeakerID int64 `json:"local_speaker_id"`
		PersonID       int64 `json:"person_id"`
		Refused        bool  `json:"refused"`
	}
	if err := json.Unmarshal(h.userCall(user, http.MethodDelete, cancel, nil, http.StatusOK), &view); err != nil {
		t.Fatal(err)
	}
	if view.LocalSpeakerID != near.ID || view.PersonID != alice || view.Refused {
		t.Fatalf("the cancellation answers the voice, the Person and refused=false: %+v", view)
	}
	if got := h.refusedPersonsOf(near); len(got) != 0 {
		t.Fatalf("a cancelled refusal leaves the voice: %v", got)
	}
	h.rematch()
	if got := h.matched(near); got.Person == nil || got.Person.ID != alice {
		t.Fatalf("after the cancellation, matching proposes the Person again: %+v", got)
	}

	// The refusal and its cancellation are journaled, by their authors, in order.
	type event struct{ Action, Author string }
	rows, err := pool.Query(context.Background(), `SELECT action,author_username FROM voice_refusal_events WHERE local_speaker_id=$1 AND person_id=$2 ORDER BY id`, near.ID, alice)
	if err != nil {
		t.Fatal(err)
	}
	events := []event{}
	for rows.Next() {
		var e event
		if err = rows.Scan(&e.Action, &e.Author); err != nil {
			t.Fatal(err)
		}
		events = append(events, e)
	}
	rows.Close()
	if len(events) != 2 || events[0] != (event{"refuse", "administrator"}) || events[1] != (event{"cancel", "bob"}) {
		t.Fatalf("the refusal and its cancellation are journaled: %+v", events)
	}
	// Cancelling a refusal that does not exist writes nothing.
	h.userCall(user, http.MethodDelete, cancel, nil, http.StatusOK)
	if n := h.count(`SELECT count(*) FROM voice_refusal_events WHERE local_speaker_id=$1`, near.ID); n != 2 {
		t.Fatalf("a cancellation without refusal is not journaled: %d", n)
	}
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/local-speakers/999999/refusals/%d", alice), nil, http.StatusNotFound)
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/local-speakers/%d/refusals/0", near.ID), nil, http.StatusBadRequest)
	// The journal is append-only.
	if _, err = pool.Exec(context.Background(), `DELETE FROM voice_refusal_events`); err == nil {
		t.Fatal("the refusal journal must be append-only")
	}
}

// testRefusalCancelDuringActiveAnalysis: like the refusal, its cancellation waits for the
// analysis (409 analysis_active, nothing written).
func testRefusalCancelDuringActiveAnalysis(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	_, second := h.diarizeClip()
	bob := h.createPerson("Bob")
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", second), map[string]any{"person_id": bob}, http.StatusOK)
	h.startAnalysis(map[string]any{"scope": "voice", "mode": "full"}, http.StatusCreated)
	var refused struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodDelete, fmt.Sprintf("/api/local-speakers/%d/refusals/%d", second, bob), nil, http.StatusConflict), &refused); err != nil {
		t.Fatal(err)
	}
	if refused.Error != "analysis_active" {
		t.Fatalf("a cancellation during an active analysis must say so: %+v", refused)
	}
	if n := h.count(`SELECT count(*) FROM voice_refusals WHERE local_speaker_id=$1`, second); n != 1 {
		t.Fatalf("a refused cancellation keeps the refusal: %d", n)
	}
	h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusOK)
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/local-speakers/%d/refusals/%d", second, bob), nil, http.StatusOK)
	if n := h.count(`SELECT count(*) FROM voice_refusals WHERE local_speaker_id=$1`, second); n != 0 {
		t.Fatalf("the refusal is cancelled once the analysis is over: %d", n)
	}
}

// testDispositionDuringActiveAnalysis: like a refusal, a disposition waits for the analysis
// (409 analysis_active, nothing written), for a voice, its restoration and a group.
func testDispositionDuringActiveAnalysis(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip()
	firstVoice := fixtureSpeaker{ID: first, ClipID: h.clipID}
	h.setDisposition(firstVoice, "random", http.StatusOK)
	h.startAnalysis(map[string]any{"scope": "voice", "mode": "full"}, http.StatusCreated)
	errorOf := func(data []byte) string {
		t.Helper()
		var refused struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(data, &refused); err != nil {
			t.Fatal(err)
		}
		return refused.Error
	}
	if got := errorOf(h.setDisposition(fixtureSpeaker{ID: second, ClipID: h.clipID}, "ignored", http.StatusConflict)); got != "analysis_active" {
		t.Fatalf("a disposition during an active analysis must say so: %q", got)
	}
	if got := errorOf(h.adminCall(http.MethodDelete, fmt.Sprintf("/api/local-speakers/%d/disposition", first), nil, http.StatusConflict)); got != "analysis_active" {
		t.Fatalf("a restoration during an active analysis must say so: %q", got)
	}
	if n := h.count(`SELECT count(*) FROM voice_disposition_events WHERE clip_id=$1`, h.clipID); n != 1 {
		t.Fatalf("a refused disposition writes nothing: %d", n)
	}
	h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusOK)
	h.setDisposition(fixtureSpeaker{ID: second, ClipID: h.clipID}, "ignored", http.StatusOK)
	h.restoreDisposition(firstVoice)
}

// testDispositionWaitsForDiarization: a disposition takes the clip row FOR SHARE, like a
// validation or a refusal, so it cannot be committed while a diarization (which takes the row
// FOR UPDATE, plans the carry, then deletes the old voices) is being ingested.
func testDispositionWaitsForDiarization(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, _ := h.diarizeClip()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, h.clipID); err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/local-speakers/%d/disposition", h.ts.URL, first), strings.NewReader(`{"disposition":"ignored"}`))
		req.Header.Set("Origin", h.ts.URL)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", h.csrf)
		resp, err := h.admin.Do(req)
		if err != nil {
			done <- 0
			return
		}
		resp.Body.Close()
		done <- resp.StatusCode
	}()
	select {
	case status := <-done:
		t.Fatalf("the disposition must wait for the clip lock of the diarization, it answered %d", status)
	case <-time.After(500 * time.Millisecond):
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-done:
		if status != http.StatusOK {
			t.Fatalf("once the diarization is over the disposition is recorded: %d", status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the disposition never completed")
	}
}

// testDroppedSpeakerReasonNamesCause: when Local speakers are deleted, what they held (a
// segment's speaker, a disposition, a refusal) is journaled by the system with the actual
// cause: a new diarization, or the designation of a microphone track (ADR 0031).
func testDroppedSpeakerReasonNamesCause(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	_, second := h.diarizeClip()
	segment := fmt.Sprintf("/api/clips/%d/segments/0/speaker", h.clipID)
	h.adminCall(http.MethodPut, segment, map[string]any{"local_speaker_id": second, "expected_version": 0}, http.StatusOK)
	segmentEvents := func(reason string) int {
		t.Helper()
		return h.count(`SELECT count(*) FROM segment_correction_events WHERE clip_id=$1 AND author_username='system' AND reason=$2`, h.clipID, reason)
	}

	// A new diarization: the segment returns to its deduced speaker, for that reason.
	voices := h.rediarizeWith(map[string]any{"audio_stream_index": 2, "confirm_replace_validations": true}, map[string][][2]float64{"NEW_A": {{0, 1.5}}, "NEW_B": {{1.5, 2.5}}}, "NEW_A", "NEW_B")
	if n := segmentEvents("Locuteur supprimé par une nouvelle diarisation"); n != 1 {
		t.Fatalf("the dropped segment speaker names the new diarization: %d", n)
	}

	// The designation of the microphone track: the dropped speaker, disposition and refusal
	// are journaled before the voices go, naming the designation.
	h.dropPendingFingerprints()
	bob := h.createPerson("Bob")
	h.setDisposition(voices["NEW_A"], "ignored", http.StatusOK)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", voices["NEW_B"].ID), map[string]any{"person_id": bob}, http.StatusOK)
	// The designation and the system's return to the deduced speaker each made a version.
	h.adminCall(http.MethodPut, segment, map[string]any{"local_speaker_id": voices["NEW_B"].ID, "expected_version": 2}, http.StatusOK)
	var sourceID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM clip_sources ORDER BY id LIMIT 1`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/sources/%d/microphone-track", sourceID), map[string]any{"title": "Voix", "person_id": h.createPerson("Alice")}, http.StatusOK)
	const designation = "Locuteur supprimé par la désignation de la piste microphone"
	if n := h.count(`SELECT count(*) FROM voice_disposition_events WHERE local_speaker_id IS NULL AND clip_id=$1 AND old_disposition='ignored' AND new_disposition='' AND author_username='system' AND reason=$2`, h.clipID, designation); n != 1 {
		t.Fatalf("the dropped disposition is journaled with the designation as its cause: %d", n)
	}
	if n := h.count(`SELECT count(*) FROM voice_refusal_events WHERE local_speaker_id IS NULL AND clip_id=$1 AND person_id=$2 AND action='drop' AND author_username='system' AND reason=$3`, h.clipID, bob, designation); n != 1 {
		t.Fatalf("the dropped refusal is journaled with the designation as its cause: %d", n)
	}
	if n := segmentEvents(designation); n != 1 {
		t.Fatalf("the dropped segment speaker names the designation: %d", n)
	}
}
