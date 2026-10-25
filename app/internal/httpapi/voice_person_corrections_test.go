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

type correctionSummary struct {
	Person *struct {
		ID                   int64  `json:"id"`
		DisplayName          string `json:"display_name"`
		ValidatedSampleCount int    `json:"validated_sample_count"`
		LocalSpeakerCount    int    `json:"local_speaker_count"`
	} `json:"person"`
	Attributions    int `json:"attributions"`
	Samples         int `json:"samples"`
	MicrophoneRules int `json:"microphone_rules"`
}

type personEventRow struct {
	Action string `json:"action"`
	Person struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"person"`
	Target *struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"target"`
	Attributions int `json:"attribution_count"`
	Samples      int `json:"sample_count"`
	Author       *struct {
		Username string `json:"username"`
	} `json:"author"`
}

func (h *voiceHarness) personEvents(query string) []personEventRow {
	h.t.Helper()
	var v struct {
		Items []personEventRow `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/person-events"+query, nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v.Items
}

type speakerEvent struct {
	Action    string `json:"action"`
	OldPerson *struct {
		DisplayName string `json:"display_name"`
	} `json:"old_person"`
	NewPerson *struct {
		DisplayName string `json:"display_name"`
	} `json:"new_person"`
	NewLevel string `json:"new_decision_level"`
	Author   *struct {
		Username string `json:"username"`
	} `json:"author"`
}

func (h *voiceHarness) speakerHistory(sp fixtureSpeaker) []speakerEvent {
	h.t.Helper()
	var v struct {
		Items []speakerEvent `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/local-speakers/%d/history", sp.ID), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v.Items
}

func (h *voiceHarness) personExists(id int64) bool {
	return h.count(`SELECT count(*) FROM persons WHERE id=$1`, id) == 1
}

func testPersonCorrections(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE person_events`); err != nil {
		t.Fatal(err)
	}
	clipA, clipB := h.clipIDs[0], h.clipIDs[1]
	member := h.newUser("member")
	path := func(id int64, leaf string) string { return fmt.Sprintf("/api/persons/%d/%s", id, leaf) }

	// Merge: two Persons created by mistake for one identity.
	alice := h.fixtureSpeaker(clipA, "ALICE_VOICE", usableAt(0))
	alicia := h.fixtureSpeaker(clipA, "ALICIA_VOICE", usableAt(90))
	nearAlicia := h.fixtureSpeaker(clipB, "NEAR_ALICIA", usableAt(80))
	h.validate(alice, "Alice")
	h.validate(alicia, "Alicia")
	aliceID, aliciaID := h.personOf(alice), h.personOf(alicia)
	if got := h.matched(nearAlicia); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != aliciaID {
		t.Fatalf("before the merge the close voice is suggested to Alicia: %+v", got)
	}

	anon := &voiceUser{name: "anonymous", client: &http.Client{}}
	if status, _ := h.userStatus(anon, http.MethodPost, path(aliciaID, "merge"), map[string]any{"target_person_id": aliceID}); status != http.StatusUnauthorized {
		t.Fatalf("anonymous merge: %d", status)
	}
	h.userCall(member, http.MethodPost, path(aliciaID, "merge"), map[string]any{}, http.StatusBadRequest)
	h.userCall(member, http.MethodPost, path(aliciaID, "merge"), map[string]any{"target_person_id": 0}, http.StatusBadRequest)
	h.userCall(member, http.MethodPost, path(aliciaID, "merge"), map[string]any{"target_person_id": aliciaID}, http.StatusConflict)
	h.userCall(member, http.MethodPost, path(aliciaID, "merge"), map[string]any{"target_person_id": 999999}, http.StatusNotFound)
	h.userCall(member, http.MethodPost, path(999999, "merge"), map[string]any{"target_person_id": aliceID}, http.StatusNotFound)
	// A Person tied to an account is never absorbed silently.
	if _, err := pool.Exec(ctx, `UPDATE persons SET user_id=(SELECT id FROM users WHERE username='member') WHERE id=$1`, aliciaID); err != nil {
		t.Fatal(err)
	}
	h.userCall(member, http.MethodPost, path(aliciaID, "merge"), map[string]any{"target_person_id": aliceID}, http.StatusConflict)
	if _, err := pool.Exec(ctx, `UPDATE persons SET user_id=NULL WHERE id=$1`, aliciaID); err != nil {
		t.Fatal(err)
	}

	var merged correctionSummary
	if err := json.Unmarshal(h.userCall(member, http.MethodPost, path(aliciaID, "merge"), map[string]any{"target_person_id": aliceID}, http.StatusOK), &merged); err != nil {
		t.Fatal(err)
	}
	if merged.Person == nil || merged.Person.ID != aliceID || merged.Person.ValidatedSampleCount != 2 || merged.Person.LocalSpeakerCount != 2 || merged.Attributions != 1 || merged.Samples != 1 {
		t.Fatalf("the merge keeps the target and moves attributions and samples: %+v", merged)
	}
	if h.personExists(aliciaID) {
		t.Fatal("the merged Person must disappear")
	}
	if got := h.matched(alicia); got.State != "identified" || got.Person == nil || got.Person.ID != aliceID || got.Attribution.ValidationState != "validated" {
		t.Fatalf("the merged attribution must point to the target: %+v", got)
	}
	if h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1 AND retired_at IS NULL`, aliceID) != 2 || h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1`, aliciaID) != 0 {
		t.Fatal("both samples must now be references of the target")
	}
	// Matching was recomputed without any worker: the close voice now follows Alice.
	if got := h.matched(nearAlicia); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != aliceID || len(got.Attribution.Candidates) == 0 {
		t.Fatalf("matching must be recomputed after the merge: %+v", got)
	}
	// The history keeps who, when and the old value; the first validator stays the author.
	history := h.speakerHistory(alicia)
	if len(history) != 2 || history[0].Action != "merge" || history[0].OldPerson == nil || history[0].OldPerson.DisplayName != "Alicia" || history[0].NewPerson == nil || history[0].NewPerson.DisplayName != "Alice" || history[0].Author == nil || history[0].Author.Username != "member" || history[1].Action != "validate" {
		t.Fatalf("merge history: %+v", history)
	}
	events := h.personEvents("")
	if len(events) != 1 || events[0].Action != "merge" || events[0].Person.DisplayName != "Alicia" || events[0].Target == nil || events[0].Target.ID != aliceID || events[0].Attributions != 1 || events[0].Author == nil || events[0].Author.Username != "member" {
		t.Fatalf("person history: %+v", events)
	}
	// The first validation still stands: nobody can validate the voice a second time.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", alicia.ID), map[string]any{"new_person_name": "Someone else"}, http.StatusConflict)

	testPersonSeparation(t, h, member)
	testPersonDeletion(t, h, member)
	testPersonMergeMicrophone(t, h, member)
}

func personPath(id int64, leaf string) string {
	if leaf == "" {
		return fmt.Sprintf("/api/persons/%d", id)
	}
	return fmt.Sprintf("/api/persons/%d/%s", id, leaf)
}

// Separation: a voice validated by mistake leaves the Person, which stays.
func testPersonSeparation(t *testing.T, h *voiceHarness, member *voiceUser) {
	clipA, clipB := h.clipIDs[0], h.clipIDs[1]
	bobVoice := h.fixtureSpeaker(clipA, "BOB_VOICE", usableAt(180))
	mistaken := h.fixtureSpeaker(clipB, "BOB_MISTAKE", usableAt(270))
	nearBob := h.fixtureSpeaker(clipB, "NEAR_BOB", usableAt(185))
	h.validate(bobVoice, "Bob")
	bob := h.personOf(bobVoice)
	h.validate(mistaken, bob)
	if got := h.matched(nearBob); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != bob {
		t.Fatalf("the close voice is suggested to Bob: %+v", got)
	}
	body := func(ids ...int64) map[string]any { return map[string]any{"local_speaker_ids": ids} }
	anon := &voiceUser{name: "anonymous", client: &http.Client{}}
	if status, _ := h.userStatus(anon, http.MethodPost, personPath(bob, "separation"), body(mistaken.ID)); status != http.StatusUnauthorized {
		t.Fatalf("anonymous separation: %d", status)
	}
	h.userCall(member, http.MethodPost, personPath(bob, "separation"), body(), http.StatusBadRequest)
	h.userCall(member, http.MethodPost, personPath(bob, "separation"), body(0), http.StatusBadRequest)
	h.userCall(member, http.MethodPost, personPath(999999, "separation"), body(mistaken.ID), http.StatusNotFound)
	h.userCall(member, http.MethodPost, personPath(bob, "separation"), body(999999), http.StatusNotFound)
	// A voice that is not validated for this Person cannot be separated from it.
	free := h.fixtureSpeaker(clipB, "FREE_VOICE", usableAt(300))
	h.userCall(member, http.MethodPost, personPath(bob, "separation"), body(free.ID), http.StatusConflict)
	h.userCall(member, http.MethodPost, personPath(bob, "separation"), body(mistaken.ID, free.ID), http.StatusConflict)
	if h.matched(mistaken).State != "identified" {
		t.Fatal("a refused separation must change nothing")
	}
	// Voices of clips the caller cannot see are reported as not found.
	if _, err := h.pool.Exec(context.Background(), `UPDATE clips SET visibility='private' WHERE id=$1`, clipB); err != nil {
		t.Fatal(err)
	}
	h.userCall(member, http.MethodPost, personPath(bob, "separation"), body(mistaken.ID), http.StatusNotFound)
	if _, err := h.pool.Exec(context.Background(), `UPDATE clips SET visibility='shared' WHERE id=$1`, clipB); err != nil {
		t.Fatal(err)
	}

	var split correctionSummary
	if err := json.Unmarshal(h.userCall(member, http.MethodPost, personPath(bob, "separation"), body(mistaken.ID, mistaken.ID), http.StatusOK), &split); err != nil {
		t.Fatal(err)
	}
	if split.Person == nil || split.Person.ID != bob || split.Attributions != 1 || split.Samples != 1 || split.Person.ValidatedSampleCount != 1 || split.Person.LocalSpeakerCount != 1 {
		t.Fatalf("separation summary: %+v", split)
	}
	if got := h.matched(mistaken); got.State != "unidentified" || got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("a separated voice is unknown again: %+v", got)
	}
	history := h.speakerHistory(mistaken)
	if len(history) != 2 || history[0].Action != "split" || history[0].OldPerson == nil || history[0].OldPerson.DisplayName != "Bob" || history[0].NewLevel != "unknown" || history[0].Author == nil || history[0].Author.Username != "member" {
		t.Fatalf("separation history: %+v", history)
	}
	if got := h.matched(nearBob); got.Person == nil || got.Person.ID != bob {
		t.Fatalf("the remaining reference still matches: %+v", got)
	}
	// The voice can be validated again: the first validation of the new state stands.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", mistaken.ID), map[string]any{"new_person_name": "Carol"}, http.StatusCreated)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", mistaken.ID), map[string]any{"person_id": bob}, http.StatusConflict)

	// Separating the last reference empties the Person and the matching follows.
	h.userCall(member, http.MethodPost, personPath(bob, "separation"), body(bobVoice.ID), http.StatusOK)
	if got := h.matched(nearBob); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("without references nothing is matched: %+v", got)
	}
	if !h.personExists(bob) {
		t.Fatal("a separation keeps the Person")
	}
	events := h.personEvents("?person_id=" + strconv64(bob))
	if len(events) != 2 || events[0].Action != "split" || events[0].Attributions != 1 {
		t.Fatalf("person history after separations: %+v", events)
	}
}

type micByClip struct {
	Status string `json:"status"`
	Person *struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"person"`
	History []struct {
		Scope  string `json:"scope"`
		Action string `json:"action"`
	} `json:"history"`
}

func (h *voiceHarness) micOf(clipID int64) micByClip {
	h.t.Helper()
	var v micByClip
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/microphone-track", clipID), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

func (h *voiceHarness) sourceID() int64 {
	var id int64
	if err := h.pool.QueryRow(context.Background(), `SELECT id FROM clip_sources ORDER BY id LIMIT 1`).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

// Deletion: validations withdrawn, samples erased, designations removed, clips kept.
func testPersonDeletion(t *testing.T, h *voiceHarness, member *voiceUser) {
	ctx := context.Background()
	clipA, clipB := h.clipIDs[0], h.clipIDs[1]
	dora := h.newPerson("Dora")
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/sources/%d/microphone-track", h.sourceID()), map[string]any{"title": "Micro", "person_id": dora}, http.StatusOK)
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/microphone-track", clipB), map[string]any{"audio_stream_index": 1, "person_id": dora}, http.StatusOK)
	doraVoice := h.fixtureSpeaker(clipA, "DORA_VOICE", usableAt(135))
	nearDora := h.fixtureSpeaker(clipB, "NEAR_DORA", usableAt(140))
	h.validate(doraVoice, dora)
	if got := h.matched(nearDora); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != dora {
		t.Fatalf("the close voice is suggested to Dora: %+v", got)
	}
	if m := h.micOf(clipA); m.Status != "designated" || m.Person == nil || m.Person.ID != dora {
		t.Fatalf("Dora owns the microphone track: %+v", m)
	}
	if h.count(`SELECT count(*) FROM local_speakers WHERE kind='microphone'`) != 2 {
		t.Fatal("both clips must have a microphone speaker")
	}

	anon := &voiceUser{name: "anonymous", client: &http.Client{}}
	if status, _ := h.userStatus(anon, http.MethodDelete, personPath(dora, ""), nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous deletion: %d", status)
	}
	h.userCall(member, http.MethodDelete, personPath(999999, ""), nil, http.StatusNotFound)
	h.userCall(member, http.MethodDelete, "/api/persons/zero", nil, http.StatusBadRequest)
	if _, err := h.pool.Exec(ctx, `UPDATE persons SET user_id=(SELECT id FROM users WHERE username='member') WHERE id=$1`, dora); err != nil {
		t.Fatal(err)
	}
	h.userCall(member, http.MethodDelete, personPath(dora, ""), nil, http.StatusConflict)
	if _, err := h.pool.Exec(ctx, `UPDATE persons SET user_id=NULL WHERE id=$1`, dora); err != nil {
		t.Fatal(err)
	}
	if !h.personExists(dora) || h.matched(doraVoice).State != "identified" {
		t.Fatal("a refused deletion must change nothing")
	}

	var deleted correctionSummary
	if err := json.Unmarshal(h.userCall(member, http.MethodDelete, personPath(dora, ""), nil, http.StatusOK), &deleted); err != nil {
		t.Fatal(err)
	}
	if deleted.Person != nil || deleted.Attributions != 1 || deleted.Samples != 1 || deleted.MicrophoneRules != 2 {
		t.Fatalf("deletion summary: %+v", deleted)
	}
	if h.personExists(dora) || h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1`, dora) != 0 {
		t.Fatal("the Person and its samples must be gone")
	}
	if got := h.matched(doraVoice); got.State != "unidentified" || got.DecisionLevel != "unknown" || got.Person != nil || (got.Attribution != nil && got.Attribution.ValidationState == "validated") {
		t.Fatalf("the voice is unknown again: %+v", got)
	}
	if got := h.matched(nearDora); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("matching is recomputed without Dora: %+v", got)
	}
	history := h.speakerHistory(doraVoice)
	if len(history) != 2 || history[0].Action != "delete_person" || history[0].OldPerson == nil || history[0].OldPerson.DisplayName != "Dora" || history[0].Author == nil || history[0].Author.Username != "member" {
		t.Fatalf("deletion history: %+v", history)
	}
	// The designations that named Dora are removed and said so; the clips stay.
	if m := h.micOf(clipA); m.Status != "none" || len(m.History) == 0 || m.History[0].Action != "remove" {
		t.Fatalf("the microphone rule must be removed with a history entry: %+v", m)
	}
	if m := h.micOf(clipB); m.Status != "none" || len(m.History) == 0 || m.History[0].Action != "remove" {
		t.Fatalf("the clip override must be removed with a history entry: %+v", m)
	}
	if h.count(`SELECT count(*) FROM local_speakers WHERE kind='microphone'`) != 0 || h.count(`SELECT count(*) FROM clips`) != 2 || h.count(`SELECT count(*) FROM voice_fingerprints WHERE local_speaker_id=$1`, doraVoice.ID) != 1 {
		t.Fatal("deleting a Person keeps clips and fingerprints and drops its microphone speakers")
	}
	events := h.personEvents("?person_id=" + strconv64(dora))
	if len(events) != 1 || events[0].Action != "delete" || events[0].Person.DisplayName != "Dora" || events[0].Attributions != 1 || events[0].Samples != 1 {
		t.Fatalf("person history after deletion: %+v", events)
	}
	h.userCall(member, http.MethodDelete, personPath(dora, ""), nil, http.StatusNotFound)
}

// Merging also moves the microphone designations that named the Person.
func testPersonMergeMicrophone(t *testing.T, h *voiceHarness, member *voiceUser) {
	clipA := h.clipIDs[0]
	eve, eva := h.newPerson("Eve"), h.newPerson("Eva")
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/sources/%d/microphone-track", h.sourceID()), map[string]any{"title": "Micro", "person_id": eva}, http.StatusOK)
	if m := h.micOf(clipA); m.Person == nil || m.Person.ID != eva {
		t.Fatalf("Eva owns the microphone track: %+v", m)
	}
	var merged correctionSummary
	if err := json.Unmarshal(h.userCall(member, http.MethodPost, personPath(eva, "merge"), map[string]any{"target_person_id": eve}, http.StatusOK), &merged); err != nil {
		t.Fatal(err)
	}
	if merged.MicrophoneRules != 1 {
		t.Fatalf("merge summary: %+v", merged)
	}
	if m := h.micOf(clipA); m.Status != "designated" || m.Person == nil || m.Person.ID != eve {
		t.Fatalf("the designation must follow the merge: %+v", m)
	}
	var rule struct {
		Rule *struct {
			Person struct {
				ID int64 `json:"id"`
			} `json:"person"`
		} `json:"rule"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/admin/sources/%d/microphone-track", h.sourceID()), nil, http.StatusOK), &rule); err != nil {
		t.Fatal(err)
	}
	if rule.Rule == nil || rule.Rule.Person.ID != eve {
		t.Fatalf("the source rule must name the target: %+v", rule)
	}
}
