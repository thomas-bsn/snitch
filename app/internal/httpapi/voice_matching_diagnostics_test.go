//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/logging"
)

type matchingSummary struct {
	Event   string         `json:"event"`
	ClipID  int64          `json:"clip_id"`
	Levels  map[string]int `json:"levels"`
	Reasons map[string]int `json:"reasons"`
}

// lastSummary returns the latest per-clip matching summary of the captured logs.
func lastSummary(t *testing.T, logs *lockedBuffer, clip int64) (matchingSummary, bool) {
	t.Helper()
	var found matchingSummary
	ok := false
	for _, line := range strings.Split(logs.String(), "\n") {
		var entry matchingSummary
		if json.Unmarshal([]byte(line), &entry) == nil && entry.Event == "voice_matching_summary" && entry.ClipID == clip {
			found, ok = entry, true
		}
	}
	return found, ok
}

func testVoiceMatchingDiagnostics(t *testing.T, pool *pgxpool.Pool) {
	logs := &lockedBuffer{}
	h := newVoiceHarnessLogged(t, pool, 2, logging.New(logs, "snitch", "DEBUG"))
	if _, err := pool.Exec(t.Context(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clip, other := h.clipIDs[0], h.clipIDs[1]

	// Without any reference the voices cannot be compared: the reason is no_reference.
	h.fixtureSpeaker(clip, "LONELY", usableAt(0))
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	first, ok := lastSummary(t, logs, clip)
	if !ok || first.Reasons["no_reference"] != 1 || first.Levels["unknown"] != 1 {
		t.Fatalf("a clip without reference must report no_reference: %+v", first)
	}

	// Two references, then voices that are ambiguous, too far, suggested or on the microphone track.
	h.validate(h.fixtureSpeaker(other, "REF_ALICE", usableAt(0)), "Alice")
	bobReference := h.fixtureSpeaker(other, "REF_BOB", usableAt(20))
	h.validate(bobReference, "Bob")
	bob := h.personOf(bobReference)
	h.fixtureSpeaker(clip, "TIED", usableAt(10))
	h.fixtureSpeaker(clip, "FAR", usableAt(100))
	mic := h.fixtureSpeaker(clip, "MIC", usableAt(5))
	if _, err := pool.Exec(t.Context(), `UPDATE local_speakers SET kind='microphone',diarization_job_id=NULL WHERE id=$1`, mic.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,confidence,origin) VALUES($1,$2,'automatic','unvalidated',1,'microphone_track')`, mic.ID, bob); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	got, ok := lastSummary(t, logs, clip)
	wantLevels := map[string]int{"suggestion": 1, "ambiguous": 1, "unknown": 1, "automatic": 1}
	wantReasons := map[string]int{"tied_candidates": 1, "below_threshold": 1, "micro_track_direct": 1}
	if !ok || !sameCounts(got.Levels, wantLevels) || !sameCounts(got.Reasons, wantReasons) {
		t.Fatalf("unexpected clip summary: %+v", got)
	}

	// No Person name, fingerprint or individual score ever reaches the logs.
	for _, forbidden := range []string{"Alice", "Bob", "embedding", "score", "confidence", "0.98"} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("logs must not contain %q:\n%s", forbidden, logs.String())
		}
	}
}

func sameCounts(got, want map[string]int) bool {
	for key, value := range want {
		if got[key] != value {
			return false
		}
	}
	for key, value := range got {
		if value != 0 && want[key] != value {
			return false
		}
	}
	return true
}
