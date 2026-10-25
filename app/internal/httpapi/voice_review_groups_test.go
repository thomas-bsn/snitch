//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (h *voiceHarness) rematch() {
	h.t.Helper()
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
}

func (h *voiceHarness) groupSettings() {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		h.t.Fatal(err)
	}
}

// A private clip must not join two voices the caller sees: groups are computed over
// the voices visible to the caller only.
func testVoiceGroupVisibilityIsNotTransitive(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	h.groupSettings()
	clipA, clipB, clipPrivate := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	alice := h.newUser("alice")
	a := h.fixtureSpeaker(clipA, "A", usableAt(0))
	bridge := h.fixtureSpeaker(clipPrivate, "BRIDGE", usableAt(40))
	b := h.fixtureSpeaker(clipB, "B", usableAt(80))
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='private',owner_id=(SELECT id FROM users WHERE username='administrator') WHERE id=$1`, clipPrivate); err != nil {
		t.Fatal(err)
	}
	h.rematch()
	if g := h.groups(h.adminUser(), "").containing(a.ID); g == nil || g.MemberCount != 3 {
		t.Fatalf("the owner sees the chain as one group: %+v", g)
	}
	seen := h.groups(alice, "")
	ga, gb := seen.containing(a.ID), seen.containing(b.ID)
	if ga == nil || gb == nil || ga == gb || ga.MemberCount != 1 || gb.MemberCount != 1 || seen.containing(bridge.ID) != nil {
		t.Fatalf("a private voice must not link two visible voices: %+v %+v", ga, gb)
	}
}

// A person who deleted their voice profile cannot get new references by naming a group.
func testVoiceGroupNamingRefusesDeletedProfile(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	h.groupSettings()
	ref := h.fixtureSpeaker(h.clipIDs[0], "REF", usableAt(0))
	h.validate(ref, "Alice")
	alice := h.personOf(ref)
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/persons/%d/voice-profile", alice), nil, http.StatusOK)
	u := h.fixtureSpeaker(h.clipIDs[1], "U", usableAt(90))
	body := h.adminCall(http.MethodPost, fmt.Sprintf("/api/voice-groups/%d/naming", u.ID), map[string]any{"person_id": alice, "local_speaker_ids": []int64{u.ID}}, http.StatusConflict)
	if !bytes.Contains(body, []byte("voice_profile_deleted")) {
		t.Fatalf("unexpected error: %s", body)
	}
	if h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1`, alice) != 0 {
		t.Fatal("no reference may be created for an opted-out Person")
	}
}

// The nearest-neighbor limit applies to unknown voices only: voices that are already
// validated must not hide an unknown neighbor.
func testVoiceGroupLinksIgnoreKnownNeighbors(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	h.groupSettings()
	u1 := h.fixtureSpeaker(h.clipIDs[0], "U1", usableAt(0))
	u2 := h.fixtureSpeaker(h.clipIDs[1], "U2", usableAt(40))
	var person int64
	for i := 0; i < 25; i++ {
		known := h.fixtureSpeaker(h.clipIDs[0], fmt.Sprintf("K%d", i), usableAt(20))
		if i == 0 {
			h.validate(known, "Known")
			person = h.personOf(known)
		} else {
			h.validate(known, person)
		}
	}
	h.rematch()
	list := h.groups(h.adminUser(), "")
	g := list.containing(u1.ID)
	if g == nil || g.MemberCount != 2 || list.containing(u2.ID) != g {
		t.Fatalf("close unknown voices must be linked despite 25 closer known voices: %+v", g)
	}
}
