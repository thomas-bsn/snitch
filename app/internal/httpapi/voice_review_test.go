//go:build integration

package httpapi

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testVoiceVectorsNeverPersistedInJobs: a fingerprint result is projected into
// voice_fingerprints only. The job keeps a summary without any vector, so erasing a
// voice profile leaves nothing from which a voice could be rebuilt.
func testVoiceVectorsNeverPersistedInJobs(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, _ := h.diarizeClip()
	sp := fixtureSpeaker{ID: first, ClipID: h.clipID}
	leaks := `SELECT count(*) FROM jobs WHERE step LIKE 'voice.%' AND result::text LIKE '%"embedding":%'`
	if n := h.count(leaks); n != 0 {
		t.Fatalf("a fingerprint result must not keep vectors in jobs.result: %d", n)
	}
	h.validate(sp, "Alice")
	alice := h.personOf(sp)
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/persons/%d/voice-profile", alice), nil, http.StatusOK)
	if n := h.count(leaks); n != 0 {
		t.Fatalf("vectors remain in jobs.result after the profile deletion: %d", n)
	}
	if body := h.adminCall(http.MethodGet, fmt.Sprintf("/api/admin/jobs?clip_id=%d", h.clipID), nil, http.StatusOK); bytes.Contains(body, []byte(`"embedding":`)) {
		t.Fatalf("job list leaked a vector: %s", body)
	}
	if body := h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", h.clipID), nil, http.StatusOK); bytes.Contains(body, []byte(`"embedding":`)) {
		t.Fatalf("voice view leaked a vector: %s", body)
	}
}
