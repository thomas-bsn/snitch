//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testAutomaticFullAnalysis drives the ready hook the library calls when a managed clip is
// ready. By default (setting absent) that starts a full analysis; with the setting off, only
// the transcription is scheduled and no analysis run is created.
func testAutomaticFullAnalysis(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	ctx := context.Background()
	var other int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clips WHERE id<>$1 ORDER BY id LIMIT 1`, h.clipID).Scan(&other); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{h.clipID, other} {
		h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/audio-track", id), map[string]any{"audio_stream_index": 2}, http.StatusOK)
	}

	if err := h.api.Library.OnManagedReady(ctx, h.clipID); err != nil {
		t.Fatal(err)
	}
	run := h.analysis()
	if run == nil || run.Scope != "full" {
		t.Fatalf("an added clip must start a full analysis by default: %+v", run)
	}

	if err := h.api.Store.Set(ctx, settingAutoFullAnalysis, "false"); err != nil {
		t.Fatal(err)
	}
	if err := h.api.Library.OnManagedReady(ctx, other); err != nil {
		t.Fatal(err)
	}
	if n := h.count(`SELECT count(*) FROM clip_analysis_runs WHERE clip_id=$1`, other); n != 0 {
		t.Fatalf("with the setting off no analysis run may be created, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.%'`, other); n == 0 {
		t.Fatal("the transcription must still be scheduled when the setting is off")
	}
}
