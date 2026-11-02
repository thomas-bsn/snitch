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

type personImpact struct {
	MicrophoneRules int  `json:"microphone_rules"`
	HasHiddenVoices bool `json:"has_hidden_voices"`
}

func (h *voiceHarness) impactOf(u *voiceUser, personID int64) personImpact {
	h.t.Helper()
	var v personImpact
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, personPath(personID, "impact"), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

// The impact of a correction on one Person, counted before it is confirmed: the microphone
// rules it would remove or move, and whether a voice sits on a clip the caller cannot see.
func testPersonImpact(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	visible, private := h.clipIDs[0], h.clipIDs[1]
	alice := h.newUser("alice")

	t.Run("microphone rules", func(t *testing.T) {
		dora := h.newPerson("Dora")
		if got := h.impactOf(alice, dora); got.MicrophoneRules != 0 {
			t.Fatalf("no rule yet: %+v", got)
		}
		h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/sources/%d/microphone-track", h.sourceID()), map[string]any{"title": "Micro", "person_id": dora}, http.StatusOK)
		h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/microphone-track", private), map[string]any{"audio_stream_index": 1, "person_id": dora}, http.StatusOK)
		if got := h.impactOf(alice, dora); got.MicrophoneRules != 2 {
			t.Fatalf("one source rule and one clip override: %+v", got)
		}
		h.userCall(alice, http.MethodGet, personPath(999999, "impact"), nil, http.StatusNotFound)
	})

	t.Run("hidden voices", func(t *testing.T) {
		hidden := h.fixtureSpeaker(private, "HIDDEN", usableAt(0))
		shown := h.fixtureSpeaker(visible, "SHOWN", usableAt(90))
		h.validate(hidden, "Carol")
		carol := h.personOf(hidden)
		h.validate(shown, "Erin")
		erin := h.personOf(shown)
		if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='shared'; UPDATE clips SET visibility='private',owner_id=(SELECT id FROM users WHERE username='administrator') WHERE id=`+fmt.Sprint(private)); err != nil {
			t.Fatal(err)
		}
		if got := h.impactOf(alice, carol); !got.HasHiddenVoices {
			t.Fatalf("a voice on a private clip is hidden from alice: %+v", got)
		}
		if got := h.impactOf(alice, erin); got.HasHiddenVoices {
			t.Fatalf("a voice on a shared clip is not hidden: %+v", got)
		}
		var admin personImpact
		if err := json.Unmarshal(h.adminCall(http.MethodGet, personPath(carol, "impact"), nil, http.StatusOK), &admin); err != nil {
			t.Fatal(err)
		}
		if admin.HasHiddenVoices {
			t.Fatalf("the administrator sees every clip: %+v", admin)
		}
	})
}
