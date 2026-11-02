//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type queueView struct {
	Active []struct {
		analysisRunView
		ClipID   int64  `json:"clip_id"`
		ClipName string `json:"clip_name"`
	} `json:"active"`
	Completed []struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	} `json:"completed"`
	Tasks []struct {
		ID       int64  `json:"id"`
		ClipID   int64  `json:"clip_id"`
		Step     string `json:"step"`
		Status   string `json:"status"`
		ClipName string `json:"clip_name"`
	} `json:"tasks"`
}

func (h *voiceHarness) queue() queueView {
	h.t.Helper()
	var q queueView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/analysis/queue", nil, http.StatusOK), &q); err != nil {
		h.t.Fatal(err)
	}
	return q
}

func testAdminQueue(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.api.Events.Poll, h.api.Events.Heartbeat = 50*time.Millisecond, 300*time.Millisecond
	ctx := t.Context()

	// Administrators only, for the snapshot and for the stream.
	member := h.newUser("queue-member")
	h.userCall(member, http.MethodGet, "/api/admin/analysis/queue", nil, http.StatusForbidden)
	h.userCall(member, http.MethodGet, "/api/admin/analysis/queue/events", nil, http.StatusForbidden)
	anon, err := http.Get(h.ts.URL + "/api/admin/analysis/queue/events")
	if err != nil {
		t.Fatal(err)
	}
	anon.Body.Close()
	if anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous stream: %d", anon.StatusCode)
	}

	q := h.queue()
	if len(q.Active) != 0 || len(q.Completed) != 0 || len(q.Tasks) != 0 {
		t.Fatalf("an idle queue is empty: %+v", q)
	}

	// The stream starts from a snapshot and pushes the change of a run, live.
	req, _ := http.NewRequest(http.MethodGet, h.ts.URL+"/api/admin/analysis/queue/events", nil)
	resp, err := h.admin.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("stream content type: %s", resp.Header.Get("Content-Type"))
	}
	frames := readFrames(resp)
	if f := nextFrame(t, frames); f.Event != "snapshot" {
		t.Fatalf("expected a snapshot, got %+v", f)
	}
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	for {
		f := nextFrame(t, frames)
		var got struct {
			Snapshot queueView `json:"snapshot"`
		}
		if err := json.Unmarshal([]byte(f.Data), &got); err != nil {
			t.Fatal(err)
		}
		if len(got.Snapshot.Active) == 1 {
			if got.Snapshot.Active[0].step("preparation").Status != "blocked" {
				t.Fatalf("the card shows the blocked preparation: %+v", got.Snapshot.Active[0])
			}
			break
		}
	}

	// No hidden pagination: far more than a page of active runs all show up, one card each.
	if _, err := pool.Exec(ctx, `WITH new_clips AS (
 INSERT INTO clips(owner_id, original_name, size_bytes, modified_at, recorded_at)
 SELECT owner_id, 'queue-synthetic-' || n, 1, now(), now() FROM clips, generate_series(1, 30) n WHERE id=$1 RETURNING id)
 INSERT INTO clip_analysis_runs(clip_id, scope) SELECT id, 'full' FROM new_clips`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if got := len(h.queue().Active); got != 31 {
		t.Fatalf("all active runs are listed: %d", got)
	}

	// A technical task that belongs to no run stays visible, with its clip.
	var jobID int64
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id, step, input_version) SELECT id, 'transcription.audio.9', 'queue-v1' FROM clips WHERE original_name='queue-synthetic-1' RETURNING id`).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	q = h.queue()
	if len(q.Tasks) != 1 || q.Tasks[0].ID != jobID || q.Tasks[0].ClipName != "queue-synthetic-1" {
		t.Fatalf("a loose task stays visible: %+v", q.Tasks)
	}
	// A job a run follows is shown by its run, not twice.
	if _, err := pool.Exec(ctx, `INSERT INTO clip_analysis_steps(run_id, step, status, job_id) SELECT id, 'transcription', 'waiting', $1 FROM clip_analysis_runs WHERE clip_id=(SELECT clip_id FROM jobs WHERE id=$1)`, jobID); err != nil {
		t.Fatal(err)
	}
	if got := len(h.queue().Tasks); got != 0 {
		t.Fatalf("a followed job is not a loose task: %d", got)
	}

	// Completed keeps exactly the last three clips; failed runs are never pushed out.
	if _, err := pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='completed', finished_at=now() + id * interval '1 second' WHERE clip_id IN (SELECT id FROM clips WHERE original_name ~ 'queue-synthetic-([1-5])$')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='failed', finished_at=now() WHERE clip_id IN (SELECT id FROM clips WHERE original_name ~ 'queue-synthetic-([6-9])$')`); err != nil {
		t.Fatal(err)
	}
	q = h.queue()
	if len(q.Completed) != 3 || len(q.Active) != 31-5 {
		t.Fatalf("three completed clips, the rest still listed: %d completed, %d active", len(q.Completed), len(q.Active))
	}
}
