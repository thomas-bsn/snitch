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

type verificationCaseT struct {
	LocalSpeakerID int64  `json:"local_speaker_id"`
	ClipID         int64  `json:"clip_id"`
	Label          string `json:"label"`
	DecisionLevel  string `json:"decision_level"`
	GroupID        int64  `json:"group_id"`
	GroupVoices    int    `json:"group_voice_count"`
	GroupClips     int    `json:"group_clip_count"`
	Candidates     []struct {
		Score  float64 `json:"score"`
		Person struct {
			DisplayName string `json:"display_name"`
		} `json:"person"`
	} `json:"candidates"`
}

type verificationT struct {
	Items  []verificationCaseT `json:"items"`
	Total  int                 `json:"total"`
	Counts map[string]int      `json:"counts"`
}

func (h *voiceHarness) verification(u *voiceUser, query string) verificationT {
	h.t.Helper()
	var v verificationT
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/voice-verification"+query, nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

func testVoiceVerification(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	admin := h.adminUser()
	alice := h.newUser("alice")

	aliceRef := h.fixtureSpeaker(clipA, "REF_ALICE", usableAt(0))
	bobRef := h.fixtureSpeaker(clipA, "REF_BOB", usableAt(20))
	h.validate(aliceRef, "Alice")
	h.validate(bobRef, "Bob")
	ambiguous := h.fixtureSpeaker(clipB, "AMBIGUOUS", usableAt(10)) // as close to Alice as to Bob
	suggested := h.fixtureSpeaker(clipC, "SUGGESTED", usableAt(40)) // only Bob is close
	u1 := h.fixtureSpeaker(clipB, "U1", usableAt(120))
	u2 := h.fixtureSpeaker(clipC, "U2", usableAt(125))
	u4 := h.fixtureSpeaker(clipA, "U4", usableAt(122))
	u3 := h.fixtureSpeaker(clipA, "U3", usableAt(200))
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)

	// Only unknown and ambiguous voices are cases; the suggestion and the two human
	// validations are not, but all five states are counted by the database.
	v := h.verification(admin, "")
	if v.Total != 5 || len(v.Items) != 5 {
		t.Fatalf("the review must list the 4 unknown voices and the ambiguous one: %+v", v)
	}
	for _, c := range v.Items {
		if c.LocalSpeakerID == suggested.ID || c.LocalSpeakerID == aliceRef.ID || c.LocalSpeakerID == bobRef.ID {
			t.Fatalf("a suggestion or a validated voice is not a case: %+v", c)
		}
	}
	want := map[string]int{"human": 2, "automatic": 0, "suggestion": 1, "ambiguous": 1, "unknown": 4}
	for level, n := range want {
		if v.Counts[level] != n {
			t.Fatalf("count %s = %d, want %d (%+v)", level, v.Counts[level], n, v.Counts)
		}
	}

	// Most frequent first: the proposed group of three clips, then the ambiguous voice
	// whose group spans two clips, then the lone voice.
	order := []int64{u1.ID, u2.ID, u4.ID, ambiguous.ID, u3.ID}
	for i, id := range order {
		if v.Items[i].LocalSpeakerID != id {
			t.Fatalf("case %d should be speaker %d, got %+v", i, id, v.Items)
		}
	}
	if first := v.Items[0]; first.DecisionLevel != "unknown" || first.GroupVoices != 3 || first.GroupClips != 3 || first.GroupID != u1.ID || len(first.Candidates) != 0 {
		t.Fatalf("unknown case: %+v", first)
	}
	amb := v.Items[3]
	if amb.DecisionLevel != "ambiguous" || len(amb.Candidates) != 2 || amb.GroupClips != 2 || amb.GroupVoices != 2 || amb.Candidates[0].Score < 0.98 {
		t.Fatalf("ambiguous case with its candidates: %+v", amb)
	}
	if last := v.Items[4]; last.GroupVoices != 1 || last.GroupClips != 1 {
		t.Fatalf("lone voice: %+v", last)
	}

	// Filters and pagination are applied by the database.
	if only := h.verification(admin, "?level=ambiguous"); only.Total != 1 || only.Items[0].LocalSpeakerID != ambiguous.ID {
		t.Fatalf("ambiguous filter: %+v", only)
	}
	if only := h.verification(admin, "?level=unknown"); only.Total != 4 {
		t.Fatalf("unknown filter: %+v", only)
	}
	if page := h.verification(admin, "?page=2&page_size=2"); page.Total != 5 || len(page.Items) != 2 || page.Items[0].LocalSpeakerID != u4.ID {
		t.Fatalf("pagination: %+v", page)
	}
	for _, query := range []string{"?level=suggestion", "?page=0", "?page_size=101"} {
		if status, _ := h.userStatus(admin, http.MethodGet, "/api/voice-verification"+query, nil); status != http.StatusBadRequest {
			t.Fatalf("%s must be refused, got %d", query, status)
		}
	}
	if n := h.count(`SELECT count(*) FROM persons`); n != 2 {
		t.Fatalf("reading the review must not create anything, persons = %d", n)
	}

	// A private clip is neither listed nor counted for another user.
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='private' WHERE id=$1`, clipC); err != nil {
		t.Fatal(err)
	}
	seen := h.verification(alice, "")
	if seen.Total != 4 || seen.Counts["ambiguous"] != 1 || seen.Counts["suggestion"] != 0 || seen.Counts["human"] != 2 || seen.Counts["unknown"] != 3 {
		t.Fatalf("hidden clips must not leak into cases or counters: %+v", seen)
	}
	for _, c := range seen.Items {
		if c.ClipID == clipC {
			t.Fatalf("hidden clip case: %+v", c)
		}
	}
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='shared' WHERE id=$1`, clipC); err != nil {
		t.Fatal(err)
	}

	// Settling a case is a human validation (same mechanism as a single validation): the
	// case leaves the review and the counters come back from the database.
	h.userCall(alice, http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", ambiguous.ID), map[string]any{"person_id": h.personOf(aliceRef)}, http.StatusCreated)
	after := h.verification(admin, "")
	if after.Counts["human"] != 3 || after.Counts["ambiguous"] != 0 || after.Total != 4 {
		t.Fatalf("settling a case must update the counters: %+v", after)
	}
	for _, c := range after.Items {
		if c.LocalSpeakerID == ambiguous.ID {
			t.Fatalf("a settled case must leave the review: %+v", c)
		}
	}
	if n := h.count(fmt.Sprintf(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=%d`, ambiguous.ID)); n != 1 {
		t.Fatalf("settling creates the validated sample, got %d", n)
	}
}
