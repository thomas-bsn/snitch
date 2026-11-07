//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Human work created after the administrator confirmed what a reanalysis replaces is never made
// obsolete silently: the run keeps the instant of the confirmed stake, and the replacement only
// touches what existed then.
func testClipReanalysisStakeRace(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.useSingleAudioTrack()
	first, second := h.diarizeClip()
	text := func(index int, value string) {
		t.Helper()
		h.adminCall(http.MethodPut, fmt.Sprintf("/api/clips/%d/segments/%d/text", h.clipID, index), map[string]any{"text": value, "expected_version": 0}, http.StatusOK)
	}
	active := func(index int) int {
		return h.count(`SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND segment_index=$2 AND text IS NOT NULL AND stale_at IS NULL`, h.clipID, index)
	}
	text(0, "corrigé avant")
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", first), map[string]any{"new_person_name": "Alice"}, http.StatusCreated)

	// The refusal shows the stake and the instant it was read at.
	var refusal struct {
		Error   string `json:"error"`
		StakeAt string `json:"stake_at"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{"scope": "full", "mode": "full"}, http.StatusConflict), &refusal); err != nil || refusal.StakeAt == "" {
		t.Fatalf("the refusal must say when the stake was read: %v %+v", err, refusal)
	}
	// Someone corrects another segment while the dialog is open: it was never shown.
	text(1, "corrigé pendant la confirmation")
	run := h.startAnalysis(map[string]any{"scope": "full", "mode": "full", "confirm_replace_corrections": true, "confirm_replace_validations": true, "confirmed_stake_at": refusal.StakeAt}, http.StatusCreated)
	if run.step("transcription").Status != "waiting" || run.ConfirmedStakeAt == nil {
		t.Fatalf("the run must keep the confirmed stake instant: %+v", run)
	}
	// A voice is validated after the run started, too.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", second), map[string]any{"new_person_name": "Bob"}, http.StatusCreated)

	job, ok := h.claim()
	if !ok || job.Step != "transcription.audio.2" {
		t.Fatalf("expected the retranscription, got %+v", job)
	}
	h.complete(job, analysisTranscript, http.StatusOK)
	if active(0) != 0 || h.count(`SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND segment_index=0 AND stale_at IS NOT NULL`, h.clipID) != 1 {
		t.Fatal("the correction that was confirmed must become obsolete, history kept")
	}
	if active(1) != 1 {
		t.Fatal("a correction made after the confirmation must not be made obsolete")
	}
	diar, ok := h.claim()
	if !ok || diar.Step != "voice.diarization.2" {
		t.Fatalf("expected the new diarization, got %+v", diar)
	}
	h.complete(diar, diarizationPayload("NEW_A"), http.StatusConflict)
	if h.count(`SELECT count(*) FROM local_speakers WHERE id=ANY($1) AND kind='diarized'`, []int64{first, second}) != 2 {
		t.Fatal("a validation made after the confirmation must stop the replacement of the speakers")
	}
}
