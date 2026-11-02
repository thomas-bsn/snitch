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
	VoiceSamples    int  `json:"voice_samples"`
	Fingerprints    int  `json:"fingerprints"`
	Attributions    int  `json:"attributions"`
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

	t.Run("samples per voice", func(t *testing.T) {
		spk := h.fixtureSpeaker(visible, "SAMPLED", usableAt(30))
		h.validate(spk, "Fanny")
		fanny := h.personOf(spk)
		voices := func() []struct {
			ID      int64 `json:"local_speaker_id"`
			Samples int   `json:"sample_count"`
		} {
			var v struct {
				Items []struct {
					ID      int64 `json:"local_speaker_id"`
					Samples int   `json:"sample_count"`
				} `json:"items"`
			}
			if err := json.Unmarshal(h.userCall(alice, http.MethodGet, personPath(fanny, "voices"), nil, http.StatusOK), &v); err != nil {
				t.Fatal(err)
			}
			return v.Items
		}
		if got := voices(); len(got) != 1 || got[0].ID != spk.ID || got[0].Samples != 1 {
			t.Fatalf("a validated voice carries its sample: %+v", got)
		}
		if _, err := pool.Exec(context.Background(), `UPDATE voice_samples SET retired_at=now(),retired_reason='test' WHERE local_speaker_id=$1`, spk.ID); err != nil {
			t.Fatal(err)
		}
		if got := voices(); len(got) != 1 || got[0].Samples != 0 {
			t.Fatalf("a retired sample is no reference: %+v", got)
		}
	})

	// What the deletion of a voice profile erases is announced with the exact numbers
	// the deletion then reports, so the confirmation never promises a different count.
	t.Run("voice profile counts", func(t *testing.T) {
		first := h.fixtureSpeaker(visible, "PROFILE_A", usableAt(120))
		second := h.fixtureSpeaker(visible, "PROFILE_B", usableAt(150))
		h.validate(first, "Hugo")
		hugo := h.personOf(first)
		h.validate(second, "Hugo")
		before := h.impactOf(alice, hugo)
		if before.VoiceSamples < 1 || before.Attributions < 1 || before.Fingerprints < 1 {
			t.Fatalf("a validated voice: %+v", before)
		}
		var deletion struct {
			Samples      int `json:"samples"`
			Fingerprints int `json:"fingerprints"`
			Attributions int `json:"attributions"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodDelete, fmt.Sprintf("/api/persons/%d/voice-profile", hugo), nil, http.StatusOK), &deletion); err != nil {
			t.Fatal(err)
		}
		if deletion.Samples != before.VoiceSamples || deletion.Fingerprints != before.Fingerprints || deletion.Attributions != before.Attributions {
			t.Fatalf("announced %+v, erased %+v", before, deletion)
		}
		if after := h.impactOf(alice, hugo); after.VoiceSamples != 0 || after.Fingerprints != 0 || after.Attributions != 0 {
			t.Fatalf("nothing left to erase: %+v", after)
		}
	})
}
