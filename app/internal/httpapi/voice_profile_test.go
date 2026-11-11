//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type profilePerson struct {
	ID                   int64   `json:"id"`
	DisplayName          string  `json:"display_name"`
	UserID               *int64  `json:"user_id"`
	ValidatedSampleCount int     `json:"validated_sample_count"`
	VoiceProfileDeleted  *string `json:"voice_profile_deleted_at"`
}

type profileClips struct {
	Items []struct {
		ID int64 `json:"id"`
	} `json:"items"`
}

func (h *voiceHarness) visibleClips(u *voiceUser, query string) map[int64]bool {
	h.t.Helper()
	var list profileClips
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/clips?"+query, nil, http.StatusOK), &list); err != nil {
		h.t.Fatal(err)
	}
	out := map[int64]bool{}
	for _, c := range list.Items {
		out[c.ID] = true
	}
	return out
}

func testVoiceProfileDeletion(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 3)
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	admin := h.adminUser()

	// Synthetic library: Alice and Bob each have one validated reference; one close voice
	// is only suggested for Alice, one for Bob; Alice also owns the microphone track.
	refAlice := h.fixtureSpeaker(clipA, "REF_ALICE", usableAt(0))
	refBob := h.fixtureSpeaker(clipC, "REF_BOB", usableAt(90))
	nearAlice := h.fixtureSpeaker(clipB, "NEAR_ALICE", usableAt(10))
	nearBob := h.fixtureSpeaker(clipB, "NEAR_BOB", usableAt(80))
	h.validate(refAlice, "Alice")
	h.validate(refBob, "Bob")
	alice, bob := h.personOf(refAlice), h.personOf(refBob)
	if h.matched(nearAlice).Person == nil || h.matched(nearBob).Person == nil {
		t.Fatal("close voices must be suggested before the deletion")
	}
	var sourceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clip_sources ORDER BY id LIMIT 1`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	rulePath := fmt.Sprintf("/api/admin/sources/%d/microphone-track", sourceID)
	h.adminCall(http.MethodPut, rulePath, map[string]any{"title": "Micro", "person_id": alice}, http.StatusOK)
	var micID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM local_speakers WHERE clip_id=$1 AND kind='microphone'`, clipC).Scan(&micID); err != nil {
		t.Fatalf("the microphone track must be attributed before the deletion: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO voice_fingerprints(local_speaker_id,clip_id,fingerprint_job_id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,contributive,model,model_version,dimensions,embedding) SELECT $1,$2,fingerprint_job_id,'speaker',0,2,2,0.9,true,true,model,model_version,dimensions,embedding FROM voice_fingerprints WHERE local_speaker_id=$3`, micID, clipC, nearAlice.ID); err != nil {
		t.Fatal(err)
	}
	// clipB is private to the administrator: Alice's account must never see it.
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private',owner_id=(SELECT id FROM users WHERE username='administrator') WHERE id=$1`, clipB); err != nil {
		t.Fatal(err)
	}

	// Linking a Person to an account is an administrator decision.
	aliceUser, eve := h.newUser("alice-account"), h.newUser("eve")
	link := fmt.Sprintf("/api/admin/persons/%d/account", alice)
	h.userCall(eve, http.MethodPut, link, map[string]any{"username": "eve"}, http.StatusForbidden)
	h.adminCall(http.MethodPut, link, map[string]any{"username": ""}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, link, map[string]any{"username": "nobody"}, http.StatusNotFound)
	h.adminCall(http.MethodPut, "/api/admin/persons/999999/account", map[string]any{"username": "eve"}, http.StatusNotFound)
	var linked profilePerson
	if err := json.Unmarshal(h.adminCall(http.MethodPut, link, map[string]any{"username": "alice-account"}, http.StatusOK), &linked); err != nil || linked.UserID == nil {
		t.Fatalf("the Person must be linked: %+v %v", linked, err)
	}
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/persons/%d/account", bob), map[string]any{"username": "alice-account"}, http.StatusConflict)
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/persons/%d/account", bob), map[string]any{"username": "eve"}, http.StatusOK)
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/persons/%d/account", bob), nil, http.StatusOK)
	h.userCall(eve, http.MethodDelete, link, nil, http.StatusForbidden)

	var mine profilePerson
	if err := json.Unmarshal(h.userCall(aliceUser, http.MethodGet, "/api/me/person", nil, http.StatusOK), &mine); err != nil || mine.ID != alice || mine.ValidatedSampleCount != 1 {
		t.Fatalf("the account must find its Person: %+v %v", mine, err)
	}
	h.userCall(eve, http.MethodGet, "/api/me/person", nil, http.StatusNotFound)

	// The account sees the clips where its Person is identified, within its visibility.
	query := fmt.Sprintf("person_id=%d", alice)
	if seen := h.visibleClips(aliceUser, query); !seen[clipA] || !seen[clipC] || seen[clipB] || len(seen) != 2 {
		t.Fatalf("Alice must see her visible clips only: %+v", seen)
	}
	if seen := h.visibleClips(admin, query); !seen[clipB] {
		t.Fatalf("the administrator sees every clip: %+v", seen)
	}

	// Only the linked account (or an administrator) deletes a profile.
	deletion := fmt.Sprintf("/api/persons/%d/voice-profile", alice)
	h.userCall(eve, http.MethodDelete, deletion, nil, http.StatusForbidden)
	h.userCall(aliceUser, http.MethodDelete, fmt.Sprintf("/api/persons/%d/voice-profile", bob), nil, http.StatusForbidden)
	h.userCall(aliceUser, http.MethodDelete, "/api/persons/999999/voice-profile", nil, http.StatusNotFound)
	if h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1`, alice) != 1 {
		t.Fatal("a refused deletion must change nothing")
	}

	var report struct {
		Samples      int    `json:"samples"`
		Fingerprints int    `json:"fingerprints"`
		Attributions int    `json:"attributions"`
		DeletedAt    string `json:"deleted_at"`
	}
	if err := json.Unmarshal(h.userCall(aliceUser, http.MethodDelete, deletion, nil, http.StatusOK), &report); err != nil {
		t.Fatal(err)
	}
	if report.Samples != 1 || report.Fingerprints != 3 || report.Attributions != 5 || report.DeletedAt == "" {
		t.Fatalf("deletion report: %+v", report)
	}
	if h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1`, alice) != 0 {
		t.Fatal("samples must be erased")
	}
	if h.count(`SELECT count(*) FROM voice_fingerprints WHERE local_speaker_id=ANY($1)`, []int64{refAlice.ID, nearAlice.ID, micID}) != 0 {
		t.Fatal("the fingerprints of her voices must be erased")
	}
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE person_id=$1`, alice) != 0 || h.count(`SELECT count(*) FROM speaker_attribution_candidates WHERE person_id=$1`, alice) != 0 {
		t.Fatal("attributions to her must be erased")
	}
	for _, sp := range []fixtureSpeaker{refAlice, nearAlice} {
		if got := h.matched(sp); got.DecisionLevel != "unknown" || got.Person != nil || got.Attribution != nil || got.State != "unidentified" {
			t.Fatalf("her voices must be unknown again: %+v", got)
		}
	}
	if h.count(`SELECT count(*) FROM local_speakers WHERE kind='microphone'`) != 0 {
		t.Fatal("a microphone track owned by her is no longer attributed")
	}
	if got := h.matched(nearBob); got.Person == nil || got.Person.ID != bob || h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1 AND retired_at IS NULL`, bob) != 1 || h.count(`SELECT count(*) FROM voice_fingerprints WHERE local_speaker_id=$1`, refBob.ID) != 1 {
		t.Fatalf("other Persons must be untouched: %+v", got)
	}
	if h.count(`SELECT count(*) FROM clips`) != 3 || h.count(`SELECT count(*) FROM persons WHERE id=$1 AND voice_profile_deleted_at IS NOT NULL`, alice) != 1 {
		t.Fatal("the deletion must keep clips and the Person, and mark the opt-out")
	}
	var gone profilePerson
	if err := json.Unmarshal(h.userCall(aliceUser, http.MethodGet, "/api/me/person", nil, http.StatusOK), &gone); err != nil || gone.VoiceProfileDeleted == nil || gone.ValidatedSampleCount != 0 {
		t.Fatalf("the Person must show the deleted profile: %+v %v", gone, err)
	}

	// Group links (unknown voice groups) never point at voices whose fingerprints were erased, and
	// the accuracy measure keeps working over the remaining references.
	if h.count(`SELECT count(*) FROM voice_group_links WHERE speaker_a_id=ANY($1) OR speaker_b_id=ANY($1)`, []int64{refAlice.ID, nearAlice.ID, micID}) != 0 {
		t.Fatal("group links must not reference erased fingerprints")
	}
	h.adminCall(http.MethodGet, "/api/admin/voice-accuracy", nil, http.StatusOK)

	// Trace: who, which Person, how many rows, and no voice data at all.
	var traced struct {
		person, by                     string
		samples, fingerprints, attribs int
	}
	if err := pool.QueryRow(ctx, `SELECT person_name,deleted_by_username,sample_count,fingerprint_count,attribution_count FROM voice_profile_deletions WHERE person_id=$1`, alice).Scan(&traced.person, &traced.by, &traced.samples, &traced.fingerprints, &traced.attribs); err != nil {
		t.Fatalf("the deletion must be traced: %v", err)
	}
	if traced.person != "Alice" || traced.by != "alice-account" || traced.samples != 1 || traced.fingerprints != 3 || traced.attribs != 5 {
		t.Fatalf("trace: %+v", traced)
	}
	rows, err := pool.Query(ctx, `SELECT column_name,data_type,udt_name FROM information_schema.columns WHERE table_name='voice_profile_deletions'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name, dataType, udt string
		if err = rows.Scan(&name, &dataType, &udt); err != nil {
			t.Fatal(err)
		}
		if udt == "vector" || strings.Contains(name, "embedding") || strings.Contains(name, "vector") || dataType == "bytea" {
			t.Fatalf("the audit trail must hold no voice data: %s %s", name, udt)
		}
	}
	rows.Close()
	if _, err = pool.Exec(ctx, `UPDATE voice_profile_deletions SET person_name='tampered'`); err == nil {
		t.Fatal("the audit trail must be immutable")
	}
	if _, err = pool.Exec(ctx, `DELETE FROM voice_profile_deletions`); err == nil {
		t.Fatal("the audit trail must not be deletable")
	}
	var history struct {
		Items []struct {
			Action           string `json:"action"`
			NewDecisionLevel string `json:"new_decision_level"`
		} `json:"items"`
	}
	if err = json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/local-speakers/%d/history", refAlice.ID), nil, http.StatusOK), &history); err != nil || len(history.Items) == 0 || history.Items[0].Action != "delete_profile" || history.Items[0].NewDecisionLevel != "unknown" {
		t.Fatalf("the speaker history must record the deletion: %+v %v", history, err)
	}

	// No reconstitution: matching, a new close voice, the microphone rule and even a
	// reference inserted behind the API's back cannot bring the profile back.
	later := h.fixtureSpeaker(clipC, "LATER", usableAt(1))
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	if got := h.matched(later); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("a voice close to the deleted profile must stay unknown: %+v", got)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO voice_samples(person_id,local_speaker_id,clip_id,attribution_id) SELECT $1,$2,$3,id FROM speaker_attributions WHERE local_speaker_id=$4`, alice, nearBob.ID, clipB, nearBob.ID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	if got := h.matched(later); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("matching must ignore the references of an opted-out Person: %+v", got)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM voice_samples WHERE person_id=$1`, alice); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, rulePath, map[string]any{"title": "Micro", "person_id": alice}, http.StatusOK)
	var mic micTrackView
	if err = json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/microphone-track", clipC), nil, http.StatusOK), &mic); err != nil || mic.Status != "owner_profile_deleted" || mic.Applied {
		t.Fatalf("the microphone rule must not attribute an opted-out owner: %+v %v", mic, err)
	}
	if h.count(`SELECT count(*) FROM local_speakers WHERE kind='microphone'`) != 0 || h.count(`SELECT count(*) FROM speaker_attributions WHERE person_id=$1`, alice) != 0 {
		t.Fatal("the microphone rule must not recreate an attribution")
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", later.ID), map[string]any{"person_id": alice}, http.StatusConflict)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/correction", refBob.ID), map[string]any{"person_id": alice, "expected_person_id": bob}, http.StatusConflict)
	if h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1`, alice) != 0 {
		t.Fatal("no reference may be created for an opted-out Person")
	}

	// Deleting twice is harmless and not traced twice.
	if err = json.Unmarshal(h.adminCall(http.MethodDelete, deletion, nil, http.StatusOK), &report); err != nil || report.Samples != 0 || report.Fingerprints != 0 || report.Attributions != 0 {
		t.Fatalf("second deletion: %+v %v", report, err)
	}
	if h.count(`SELECT count(*) FROM voice_profile_deletions WHERE person_id=$1`, alice) != 1 {
		t.Fatal("a repeated deletion must not be traced again")
	}

	// Deleting a clip erases its fingerprints and derived attributions, keeps the
	// original file and the Persons.
	var original string
	if err = pool.QueryRow(ctx, `SELECT s.root_path||'/'||f.relative_path FROM source_files f JOIN clip_sources s ON s.id=f.source_id WHERE f.clip_id=$1`, clipC).Scan(&original); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM source_files WHERE clip_id=$1`, clipC); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM clips WHERE id=$1`, clipC); err != nil {
		t.Fatalf("clip deletion must succeed: %v", err)
	}
	if h.count(`SELECT count(*) FROM voice_fingerprints WHERE clip_id=$1`, clipC)+h.count(`SELECT count(*) FROM local_speakers WHERE clip_id=$1`, clipC)+h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id IN (SELECT id FROM local_speakers WHERE clip_id=$1)`, clipC) != 0 {
		t.Fatal("clip deletion must erase fingerprints and derived attributions")
	}
	if _, err = os.Stat(filepath.Clean(original)); err != nil {
		t.Fatalf("clip deletion must not touch the original file: %v", err)
	}
	if h.count(`SELECT count(*) FROM persons`) != 2 || h.count(`SELECT count(*) FROM clips`) != 2 {
		t.Fatal("clip deletion must keep Persons and other clips")
	}
}
