//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The queue stream keeps sending whole snapshots; a client that asks for delta=1 gets one whole
// snapshot, then only the runs that changed, with the order of every active run.
func testAdminQueueDelta(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.api.Events.Poll, h.api.Events.Heartbeat = 50*time.Millisecond, time.Minute
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `WITH new_clips AS (
 INSERT INTO clips(owner_id, original_name, size_bytes, modified_at, recorded_at)
 SELECT owner_id, 'delta-synthetic-' || n, 1, now(), now() FROM clips, generate_series(1, 5) n WHERE id=$1 RETURNING id),
 new_runs AS (INSERT INTO clip_analysis_runs(clip_id, scope) SELECT id, 'full' FROM new_clips RETURNING id)
 INSERT INTO clip_analysis_steps(run_id, step, status, reason) SELECT id, 'preparation', 'waiting', 'awaiting_previous_step' FROM new_runs`, h.clipID); err != nil {
		t.Fatal(err)
	}
	open := func(query string) <-chan sseFrame {
		req, _ := http.NewRequest(http.MethodGet, h.ts.URL+"/api/admin/analysis/queue/events"+query, nil)
		resp, err := h.admin.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { resp.Body.Close() })
		return readFrames(resp)
	}
	touch := func() {
		if _, err := pool.Exec(ctx, `UPDATE clip_analysis_steps SET status='running', reason='running_on_worker', updated_at=now() WHERE run_id=(SELECT min(run_id) FROM clip_analysis_steps)`); err != nil {
			t.Fatal(err)
		}
	}

	whole := open("")
	if f := nextFrame(t, whole); f.Event != "snapshot" {
		t.Fatalf("expected a snapshot, got %s", f.Event)
	}
	delta := open("?delta=1")
	first := nextFrame(t, delta)
	var snapshot struct {
		Seq      int64     `json:"seq"`
		Snapshot queueView `json:"snapshot"`
	}
	if first.Event != "snapshot" || json.Unmarshal([]byte(first.Data), &snapshot) != nil || len(snapshot.Snapshot.Active) != 5 {
		t.Fatalf("the first frame is a whole snapshot: %s %s", first.Event, first.Data)
	}

	touch()
	if f := nextFrame(t, whole); f.Event != "snapshot" {
		t.Fatalf("a client that did not ask for deltas keeps whole snapshots, got %s", f.Event)
	}
	f := nextFrame(t, delta)
	var got struct {
		Seq   int64 `json:"seq"`
		Base  int64 `json:"base"`
		Delta struct {
			ActiveIDs []int64 `json:"active_ids"`
			Active    []struct {
				ID int64 `json:"id"`
			} `json:"active"`
			Summary *struct{} `json:"summary"`
		} `json:"delta"`
	}
	if f.Event != "delta" || json.Unmarshal([]byte(f.Data), &got) != nil {
		t.Fatalf("a change comes as a delta: %s %s", f.Event, f.Data)
	}
	if got.Base != snapshot.Seq || got.Seq <= got.Base {
		t.Fatalf("a delta says which state it follows: %+v after %d", got, snapshot.Seq)
	}
	if len(got.Delta.ActiveIDs) != 5 || len(got.Delta.Active) != 1 || got.Delta.Active[0].ID != snapshot.Snapshot.Active[0].ID || got.Delta.Summary == nil {
		t.Fatalf("every run keeps its place, only the changed one is sent: %+v", got.Delta)
	}
	for i, run := range snapshot.Snapshot.Active {
		if got.Delta.ActiveIDs[i] != run.ID {
			t.Fatalf("order of the active runs: %v", got.Delta.ActiveIDs)
		}
	}
}
