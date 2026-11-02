//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type localSpeakerReadT struct {
	LocalSpeakerID int64  `json:"local_speaker_id"`
	ClipID         int64  `json:"clip_id"`
	ClipName       string `json:"clip_name"`
	Label          string `json:"label"`
	DecisionLevel  string `json:"decision_level"`
	GroupID        *int64 `json:"group_id"`
	Attribution    *struct {
		Person *struct {
			ID          int64  `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"person"`
		Author *struct {
			Username string `json:"username"`
		} `json:"author"`
		DecidedAt string `json:"decided_at"`
	} `json:"attribution"`
}

// testLocalSpeakerRead covers the direct link target of the Verification review: the
// read of one Local speaker answers with the same visibility rules as the rest, and a
// hidden voice cannot be told from a missing one.
func testLocalSpeakerRead(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	visible, private := h.clipIDs[0], h.clipIDs[1]
	admin := h.adminUser()
	alice := h.newUser("alice")
	open := h.fixtureSpeaker(visible, "OPEN", usableAt(0))
	named := h.fixtureSpeaker(visible, "NAMED", usableAt(90))
	hidden := h.fixtureSpeaker(private, "HIDDEN", usableAt(180))
	h.validate(named, "Carol")
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='shared'; UPDATE clips SET visibility='private',owner_id=(SELECT id FROM users WHERE username='administrator') WHERE id=`+fmt.Sprint(private)); err != nil {
		t.Fatal(err)
	}
	read := func(u *voiceUser, id int64, want int) []byte {
		return h.userCall(u, http.MethodGet, fmt.Sprintf("/api/local-speakers/%d", id), nil, want)
	}

	var got localSpeakerReadT
	if err := json.Unmarshal(read(alice, open.ID, http.StatusOK), &got); err != nil {
		t.Fatal(err)
	}
	if got.LocalSpeakerID != open.ID || got.Label != "OPEN" || got.DecisionLevel != "unknown" || got.Attribution != nil || got.GroupID == nil || got.ClipName == "" {
		t.Fatalf("an open voice is read as unknown with its clip and group: %+v", got)
	}
	got = localSpeakerReadT{}
	if err := json.Unmarshal(read(alice, named.ID, http.StatusOK), &got); err != nil {
		t.Fatal(err)
	}
	if got.DecisionLevel != "human" || got.Attribution == nil || got.Attribution.Person == nil || got.Attribution.Person.DisplayName != "Carol" || got.Attribution.Author == nil || got.Attribution.DecidedAt == "" || got.GroupID != nil {
		t.Fatalf("a settled voice tells who attributed it to whom and when: %+v", got)
	}
	// The administrator sees the private clip, another user does not.
	read(admin, hidden.ID, http.StatusOK)
	hiddenBody := read(alice, hidden.ID, http.StatusNotFound)
	missingBody := read(alice, 999999, http.StatusNotFound)
	if !bytes.Equal(hiddenBody, missingBody) {
		t.Fatalf("a hidden voice and a missing one must answer the same body: %s / %s", hiddenBody, missingBody)
	}
	var clipName string
	if err := pool.QueryRow(context.Background(), `SELECT original_name FROM clips WHERE id=$1`, private).Scan(&clipName); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(hiddenBody, []byte(clipName)) || bytes.Contains(hiddenBody, []byte("HIDDEN")) {
		t.Fatalf("the name of a hidden clip leaks: %s", hiddenBody)
	}
}

// testVoiceGroupByVoice covers the additive voice filter of the groups list.
func testVoiceGroupByVoice(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	alice := h.newUser("alice")
	a := h.fixtureSpeaker(h.clipIDs[0], "A", usableAt(0))
	b := h.fixtureSpeaker(h.clipIDs[1], "B", usableAt(1))
	c := h.fixtureSpeaker(h.clipIDs[0], "C", usableAt(150))
	named := h.fixtureSpeaker(h.clipIDs[1], "N", usableAt(250))
	h.validate(named, "Dora")
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	for _, sp := range []fixtureSpeaker{a, b, c} {
		list := h.groups(alice, fmt.Sprintf("?voice=%d", sp.ID))
		if len(list.Items) != 1 || list.Total != 1 || list.containing(sp.ID) == nil {
			t.Fatalf("voice %d: exactly its group is returned: %+v", sp.ID, list)
		}
	}
	if list := h.groups(alice, fmt.Sprintf("?voice=%d", named.ID)); len(list.Items) != 0 || list.Total != 0 {
		t.Fatalf("a named voice is in no group: %+v", list)
	}
	if list := h.groups(alice, "?voice=999999"); len(list.Items) != 0 {
		t.Fatalf("an unknown voice is in no group: %+v", list)
	}
	h.userCall(alice, http.MethodGet, "/api/voice-groups?voice=abc", nil, http.StatusBadRequest)
}
