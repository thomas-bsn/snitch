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

// testAutomaticFullAnalysis drives the ready hook the library calls when a managed clip is
// ready. By default (setting absent) that starts a full analysis; with the setting off, only
// the transcription is scheduled and no analysis run is created.
func testAutomaticFullAnalysis(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	h.useSingleAudioTrack()
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

// testAutomaticAnalysisSetting drives the administration endpoints of the setting (#215): on by
// default, changed by an Administrator, and stored under analysis_auto_full.
func testAutomaticAnalysisSetting(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	h.useSingleAudioTrack()
	read := func() bool {
		var got struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/settings/automatic-analysis", nil, http.StatusOK), &got); err != nil {
			t.Fatal(err)
		}
		return got.Enabled
	}
	if !read() {
		t.Fatal("the automatic full analysis must be on by default")
	}
	h.adminCall(http.MethodPut, "/api/admin/settings/automatic-analysis", map[string]any{"enabled": false}, http.StatusOK)
	if read() {
		t.Fatal("the setting must stay off once switched off")
	}
	if v, _ := h.api.Store.Get(context.Background(), settingAutoFullAnalysis); v != "false" {
		t.Fatalf("the setting is stored under analysis_auto_full, got %q", v)
	}
	h.adminCall(http.MethodPut, "/api/admin/settings/automatic-analysis", map[string]any{}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, "/api/admin/settings/automatic-analysis", map[string]any{"enabled": true}, http.StatusOK)
	if !read() {
		t.Fatal("the setting must come back on")
	}
	resp, err := http.Get(h.ts.URL + "/api/admin/settings/automatic-analysis")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous read: %d", resp.StatusCode)
	}
}
