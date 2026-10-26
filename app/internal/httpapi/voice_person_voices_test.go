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

type personVoiceRow struct {
	LocalSpeakerID  int64   `json:"local_speaker_id"`
	ClipID          int64   `json:"clip_id"`
	ClipName        string  `json:"clip_name"`
	Label           string  `json:"label"`
	SpeakingSeconds float64 `json:"speaking_seconds"`
}

func (h *voiceHarness) personVoices(u *voiceUser, personID int64) []personVoiceRow {
	h.t.Helper()
	var v struct {
		Items []personVoiceRow `json:"items"`
	}
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, fmt.Sprintf("/api/persons/%d/voices", personID), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v.Items
}

// The voices validated for a Person are listed so that a human can separate one of them.
func testPersonVoices(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	visible, private := h.clipIDs[0], h.clipIDs[1]
	alice := h.newUser("alice")
	shown := h.fixtureSpeaker(visible, "SHOWN", usableAt(0))
	hidden := h.fixtureSpeaker(private, "HIDDEN", usableAt(90))
	h.fixtureSpeaker(visible, "UNVALIDATED", usableAt(45))
	h.validate(shown, "Carol")
	carol := h.personOf(shown)
	h.validate(hidden, carol)
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='shared'; UPDATE clips SET visibility='private',owner_id=(SELECT id FROM users WHERE username='administrator') WHERE id=`+fmt.Sprint(private)); err != nil {
		t.Fatal(err)
	}

	anon := &voiceUser{name: "anonymous", client: &http.Client{}}
	if status, _ := h.userStatus(anon, http.MethodGet, personPath(carol, "voices"), nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous list: %d", status)
	}
	h.userCall(alice, http.MethodGet, personPath(999999, "voices"), nil, http.StatusNotFound)

	// Only the validated voices of clips the caller can see are listed.
	got := h.personVoices(alice, carol)
	if len(got) != 1 || got[0].LocalSpeakerID != shown.ID || got[0].ClipID != visible || got[0].Label != "SHOWN" || got[0].ClipName == "" {
		t.Fatalf("a member sees the visible validated voice only: %+v", got)
	}
	// The administrator sees every clip.
	if all := h.personVoices(h.adminUser(), carol); len(all) != 2 {
		t.Fatalf("the administrator sees both voices: %+v", all)
	}
	// A separated voice leaves the list.
	h.userCall(alice, http.MethodPost, personPath(carol, "separation"), map[string]any{"local_speaker_ids": []int64{shown.ID}}, http.StatusOK)
	if got = h.personVoices(alice, carol); len(got) != 0 {
		t.Fatalf("a separated voice is no longer listed: %+v", got)
	}
}
