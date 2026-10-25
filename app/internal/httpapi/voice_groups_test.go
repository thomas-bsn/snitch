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

type groupMemberT struct {
	LocalSpeakerID int64  `json:"local_speaker_id"`
	ClipID         int64  `json:"clip_id"`
	Label          string `json:"label"`
	Excluded       bool   `json:"excluded"`
}

type groupViewT struct {
	ID          int64          `json:"id"`
	MemberCount int            `json:"member_count"`
	ClipCount   int            `json:"clip_count"`
	Proposed    bool           `json:"proposed"`
	Members     []groupMemberT `json:"members"`
}

type groupListT struct {
	Items []groupViewT `json:"items"`
	Total int          `json:"total"`
}

func (h *voiceHarness) groups(u *voiceUser, query string) groupListT {
	h.t.Helper()
	var list groupListT
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/voice-groups"+query, nil, http.StatusOK), &list); err != nil {
		h.t.Fatal(err)
	}
	return list
}

func (l groupListT) containing(id int64) *groupViewT {
	for i := range l.Items {
		for _, m := range l.Items[i].Members {
			if m.LocalSpeakerID == id {
				return &l.Items[i]
			}
		}
	}
	return nil
}

func testVoiceGroups(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	admin := h.adminUser()
	alice := h.newUser("alice")

	s1 := h.fixtureSpeaker(clipA, "G1", usableAt(0))
	s2 := h.fixtureSpeaker(clipB, "G2", usableAt(10))
	s3 := h.fixtureSpeaker(clipC, "G3", usableAt(20))
	s7 := h.fixtureSpeaker(clipA, "G7", usableAt(5))
	orthogonal := func(axis int) fixtureFingerprint {
		fp := usableAt(0)
		fp.vector = make([]float64, 4)
		fp.vector[axis] = 1
		return fp
	}
	s4 := h.fixtureSpeaker(clipA, "G4", orthogonal(2))
	s5 := h.fixtureSpeaker(clipB, "G5", usableAt(180))
	s6 := h.fixtureSpeaker(clipC, "G6", orthogonal(3))

	// Without any proximity link every unknown voice is its own group.
	list := h.groups(admin, "")
	if list.Total != 7 || len(list.Items) != 7 {
		t.Fatalf("every unknown voice must start as a singleton group: %+v", list)
	}
	for _, g := range list.Items {
		if g.MemberCount != 1 || g.ClipCount != 1 || g.Proposed || len(g.Members) != 1 || g.ID != g.Members[0].LocalSpeakerID {
			t.Fatalf("unexpected singleton group: %+v", g)
		}
	}

	// Matching proposes links between close voices; nothing is confirmed.
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	list = h.groups(admin, "")
	group := list.containing(s1.ID)
	if group == nil || group.MemberCount != 4 || group.ClipCount != 3 || !group.Proposed || group.ID != s1.ID {
		t.Fatalf("close voices must form one proposed group of four voices over three clips: %+v", group)
	}
	for _, other := range []fixtureSpeaker{s2, s3, s7} {
		if list.containing(other.ID) != group {
			t.Fatalf("speaker %d must belong to the proposed group", other.ID)
		}
	}
	for name, far := range map[string]fixtureSpeaker{"third axis": s4, "opposite": s5, "fourth axis": s6} {
		if g := list.containing(far.ID); g == nil || g.MemberCount != 1 || g.Proposed {
			t.Fatalf("a distant voice (%s) must stay alone: %+v", name, g)
		}
	}
	if list.Total != 4 || list.Items[0].ID != s1.ID {
		t.Fatalf("groups must be sorted by number of clips, the proposed group first: %+v", list)
	}
	for _, query := range []string{"count(*) FROM persons", "count(*) FROM voice_samples", "count(*) FROM speaker_attributions WHERE validation_state='validated'", "count(*) FROM attribution_events WHERE action <> 'match'"} {
		if n := h.count(`SELECT ` + query); n != 0 {
			t.Fatalf("a proposed group must never be confirmed (%s = %d)", query, n)
		}
	}
	if proposed := h.groups(admin, "?proposed=true"); proposed.Total != 1 || proposed.Items[0].ID != s1.ID {
		t.Fatalf("proposed filter: %+v", proposed)
	}
	if by := h.groups(admin, "?sort=members"); by.Items[0].ID != s1.ID {
		t.Fatalf("sort by members: %+v", by)
	}
	if status, _ := h.userStatus(admin, http.MethodGet, "/api/voice-groups?sort=age", nil); status != http.StatusBadRequest {
		t.Fatalf("an unknown sort must be refused, got %d", status)
	}
	if paged := h.groups(admin, "?page_size=2&page=2"); paged.Total != 4 || len(paged.Items) != 2 {
		t.Fatalf("pagination: %+v", paged)
	}
	single := h.userCall(admin, http.MethodGet, fmt.Sprintf("/api/voice-groups/%d", s1.ID), nil, http.StatusOK)
	var one groupViewT
	if err := json.Unmarshal(single, &one); err != nil || one.MemberCount != 4 {
		t.Fatalf("group detail: %s %v", single, err)
	}
	h.userCall(admin, http.MethodGet, "/api/voice-groups/999999", nil, http.StatusNotFound)

	// A group is only visible through the clips the caller may see; counts follow.
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='private' WHERE id=$1`, clipC); err != nil {
		t.Fatal(err)
	}
	seen := h.groups(alice, "")
	aliceGroup := seen.containing(s1.ID)
	if aliceGroup == nil || aliceGroup.MemberCount != 3 || aliceGroup.ClipCount != 2 || seen.containing(s3.ID) != nil || seen.containing(s6.ID) != nil {
		t.Fatalf("a private clip must not appear in another user's group or counts: %+v", aliceGroup)
	}
	if got := h.groups(admin, "").containing(s1.ID); got.MemberCount != 4 {
		t.Fatalf("the owner still sees every member: %+v", got)
	}
	h.userCall(alice, http.MethodGet, fmt.Sprintf("/api/voice-groups/%d", s3.ID), nil, http.StatusNotFound)

	// Excluding one member leaves the others untouched.
	h.userCall(alice, http.MethodPut, fmt.Sprintf("/api/local-speakers/%d/group-exclusion", s3.ID), nil, http.StatusNotFound)
	h.userCall(admin, http.MethodPut, fmt.Sprintf("/api/local-speakers/%d/group-exclusion", s2.ID), nil, http.StatusOK)
	after := h.groups(admin, "")
	rest := after.containing(s1.ID)
	if rest == nil || rest.MemberCount != 3 || after.containing(s7.ID) != rest || after.containing(s3.ID) != rest {
		t.Fatalf("the other members must stay together: %+v", rest)
	}
	excluded := after.containing(s2.ID)
	if excluded == nil || excluded.MemberCount != 1 || excluded.Proposed || !excluded.Members[0].Excluded {
		t.Fatalf("the excluded member must be a flagged singleton: %+v", excluded)
	}
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE validation_state='validated'`) != 0 {
		t.Fatal("excluding a member must not validate anything")
	}
	// A refreshed proximity never brings an excluded member back.
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	if g := h.groups(admin, "").containing(s2.ID); g.MemberCount != 1 || !g.Members[0].Excluded {
		t.Fatalf("an exclusion must survive a new proximity computation: %+v", g)
	}

	// Naming requires the voices the user listened to: a stale list is refused.
	name := func(u *voiceUser, id int64, body map[string]any, want int) []byte {
		return h.userCall(u, http.MethodPost, fmt.Sprintf("/api/voice-groups/%d/naming", id), body, want)
	}
	name(admin, s1.ID, map[string]any{"new_person_name": "Dana", "local_speaker_ids": []int64{s1.ID, s2.ID, s3.ID, s7.ID}}, http.StatusConflict)
	name(admin, s1.ID, map[string]any{"new_person_name": "Dana", "local_speaker_ids": []int64{s1.ID}}, http.StatusConflict)
	name(admin, s1.ID, map[string]any{"new_person_name": "Dana"}, http.StatusBadRequest)
	name(admin, s1.ID, map[string]any{"new_person_name": "Dana", "person_id": 1, "local_speaker_ids": []int64{s1.ID, s3.ID, s7.ID}}, http.StatusBadRequest)
	name(admin, s1.ID, map[string]any{"person_id": 999999, "local_speaker_ids": []int64{s1.ID, s3.ID, s7.ID}}, http.StatusNotFound)
	name(alice, s3.ID, map[string]any{"new_person_name": "Dana", "local_speaker_ids": []int64{s3.ID}}, http.StatusNotFound)
	if h.count(`SELECT count(*) FROM persons`) != 0 {
		t.Fatal("a refused naming must not create a Person")
	}

	// Alice names what she can see: the private voice of the group is left alone.
	var named struct {
		Person struct {
			ID                   int64 `json:"id"`
			ValidatedSampleCount int   `json:"validated_sample_count"`
		} `json:"person"`
		Validated []int64 `json:"validated"`
		Skipped   []int64 `json:"skipped"`
	}
	if err := json.Unmarshal(name(alice, s1.ID, map[string]any{"new_person_name": "Dana", "local_speaker_ids": []int64{s1.ID, s7.ID}}, http.StatusOK), &named); err != nil {
		t.Fatal(err)
	}
	if len(named.Validated) != 2 || len(named.Skipped) != 0 || named.Person.ValidatedSampleCount != 2 {
		t.Fatalf("naming result: %+v", named)
	}
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE validation_state='validated' AND person_id=$1 AND author_username='alice'`, named.Person.ID) != 2 ||
		h.count(`SELECT count(*) FROM voice_samples WHERE person_id=$1 AND retired_at IS NULL`, named.Person.ID) != 2 ||
		h.count(`SELECT count(*) FROM attribution_events WHERE action='group_validate' AND new_person_id=$1 AND author_username='alice'`, named.Person.ID) != 2 {
		t.Fatal("naming a group must validate every member with its sample and its history")
	}
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=$1 AND validation_state='validated'`, s3.ID) != 0 {
		t.Fatal("the member Alice cannot see must not be validated")
	}
	var history struct {
		Items []struct {
			Action string `json:"action"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.userCall(alice, http.MethodGet, fmt.Sprintf("/api/local-speakers/%d/history", s1.ID), nil, http.StatusOK), &history); err != nil || len(history.Items) == 0 || history.Items[0].Action != "group_validate" {
		t.Fatalf("history of a named member: %+v %v", history, err)
	}
	// The named voices leave the unknown groups; the remaining voices are now matched
	// against the new references (a suggestion, never a validation).
	final := h.groups(admin, "")
	if final.containing(s1.ID) != nil || final.containing(s7.ID) != nil {
		t.Fatal("validated voices are no longer unknown")
	}
	if got := h.matched(s3); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != named.Person.ID {
		t.Fatalf("the remaining voice is suggested, not validated: %+v", got)
	}
	if g := final.containing(s3.ID); g == nil || g.MemberCount != 1 {
		t.Fatalf("a suggested voice is still an unknown group member: %+v", g)
	}

	// A singleton can be named too, with an existing Person; a voice that is no longer
	// unknown has no group left to name.
	h.validate(s4, named.Person.ID)
	name(admin, s6.ID, map[string]any{"person_id": named.Person.ID, "local_speaker_ids": []int64{s6.ID}}, http.StatusOK)
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=$1 AND validation_state='validated'`, s6.ID) != 1 {
		t.Fatal("a singleton group can be named")
	}
	name(admin, s4.ID, map[string]any{"person_id": named.Person.ID, "local_speaker_ids": []int64{s4.ID}}, http.StatusNotFound)

	// Restoring an excluded member, anonymous access and invalid ids.
	h.userCall(admin, http.MethodDelete, fmt.Sprintf("/api/local-speakers/%d/group-exclusion", s2.ID), nil, http.StatusOK)
	if h.count(`SELECT count(*) FROM voice_group_exclusions`) != 0 {
		t.Fatal("restoring must remove the exclusion")
	}
	h.userCall(admin, http.MethodPut, fmt.Sprintf("/api/local-speakers/%d/group-exclusion", s4.ID), nil, http.StatusConflict)
	h.userCall(admin, http.MethodPut, "/api/local-speakers/0/group-exclusion", nil, http.StatusBadRequest)
	anonymous := &voiceUser{name: "anonymous", client: &http.Client{}}
	for _, path := range []string{"/api/voice-groups", "/api/voice-groups/1"} {
		if status, _ := h.userStatus(anonymous, http.MethodGet, path, nil); status != http.StatusUnauthorized {
			t.Fatalf("anonymous %s: %d", path, status)
		}
	}
}
