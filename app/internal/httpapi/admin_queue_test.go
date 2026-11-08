//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
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
	h.useSingleAudioTrack()
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
			if got.Snapshot.Active[0].step("transcription").Reason != "awaiting_compatible_worker" {
				t.Fatalf("the card shows the wait for a Worker: %+v", got.Snapshot.Active[0])
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

func testAdminQueueHistory(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.useSingleAudioTrack()
	ctx := t.Context()

	// An isolated step is refused while an analysis is active on the clip, for every way
	// of scheduling one, and allowed again once the run is over.
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	refused := func(method, path string, body any) {
		t.Helper()
		var got struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(h.adminCall(method, path, body, http.StatusConflict), &got); err != nil || got.Error != "analysis_active" {
			t.Fatalf("%s %s: %s %v", method, path, got.Error, err)
		}
	}
	id := strconv.FormatInt(h.clipID, 10)
	refused(http.MethodPost, "/api/admin/jobs", map[string]any{"clip_id": h.clipID, "step": "transcription.audio.1", "input_version": "v1"})
	refused(http.MethodPost, "/api/admin/clips/"+id+"/transcription", map[string]any{})
	refused(http.MethodPost, "/api/admin/clips/"+id+"/voice", map[string]any{})
	h.userCall(h.newUser("queue-history-member"), http.MethodGet, "/api/admin/analysis/history", nil, http.StatusForbidden)
	h.adminCall(http.MethodGet, "/api/admin/analysis/history?page=0", nil, http.StatusBadRequest)

	// Clips finish: the queue ranks the last three distinct ones by completion time, the
	// history keeps every run (the failed one too) with its steps.
	if _, err := pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='failed', finished_at=now()-interval '1 hour' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE clip_analysis_steps SET status='failed', reason='job_dead' WHERE step='transcription' AND run_id IN (SELECT id FROM clip_analysis_runs WHERE clip_id=$1)`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `WITH c AS (INSERT INTO clips(owner_id, original_name, size_bytes, modified_at, recorded_at)
 SELECT owner_id, 'history-' || n, 1, now(), now() FROM clips, generate_series(1, 4) n WHERE id=$1 RETURNING id, original_name)
 INSERT INTO clip_analysis_runs(clip_id, scope, status, finished_at)
 SELECT id, 'full', 'completed', now() - (substr(original_name, 9)::int * interval '1 minute') FROM c`, h.clipID); err != nil {
		t.Fatal(err)
	}
	// A distinct clip is listed once even with several finished runs.
	if _, err := pool.Exec(ctx, `INSERT INTO clip_analysis_runs(clip_id, scope, status, finished_at) SELECT id,'full','completed',now()-interval '30 seconds' FROM clips WHERE original_name='history-1'`); err != nil {
		t.Fatal(err)
	}
	var q struct {
		Completed []struct {
			ClipName string `json:"clip_name"`
		} `json:"completed"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/analysis/queue", nil, http.StatusOK), &q); err != nil {
		t.Fatal(err)
	}
	if len(q.Completed) != 3 || q.Completed[0].ClipName != "history-1" || q.Completed[2].ClipName != "history-3" {
		t.Fatalf("last three distinct clips, newest first: %+v", q.Completed)
	}
	type history struct {
		Items []struct {
			ClipID int64  `json:"clip_id"`
			Status string `json:"status"`
			Steps  []struct {
				Step   string `json:"step"`
				Status string `json:"status"`
				Reason string `json:"reason"`
			} `json:"steps"`
		} `json:"items"`
		Total int `json:"total"`
	}
	var hist history
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/analysis/history", nil, http.StatusOK), &hist); err != nil {
		t.Fatal(err)
	}
	if hist.Total != 6 || len(hist.Items) != 6 {
		t.Fatalf("every finished run is kept: %+v", hist)
	}
	var failedKept bool
	for _, it := range hist.Items {
		if it.ClipID == h.clipID && it.Status == "failed" {
			for _, st := range it.Steps {
				failedKept = failedKept || (st.Step == "transcription" && st.Status == "failed" && st.Reason == "job_dead")
			}
		}
	}
	if !failedKept {
		t.Fatalf("the failed step is kept: %+v", hist.Items)
	}
	hist = history{}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/analysis/history?q=HISTORY-2", nil, http.StatusOK), &hist); err != nil {
		t.Fatal(err)
	}
	if hist.Total != 1 {
		t.Fatalf("search by clip name: %+v", hist)
	}

	// Once the run is over the isolated step is accepted again.
	h.adminCall(http.MethodPost, "/api/admin/jobs", map[string]any{"clip_id": h.clipID, "step": "transcription.audio.1", "input_version": "v1"}, http.StatusCreated)
}

// The "voices to check" badge follows the voices the clip really has left, not the run status.
func testAdminQueuePendingVoices(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := t.Context()
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	if _, err := pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='completed', finished_at=now() WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	pending := func() int {
		var q struct {
			Completed []struct {
				PendingVoices int `json:"pending_voices"`
			} `json:"completed"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/analysis/queue", nil, http.StatusOK), &q); err != nil || len(q.Completed) != 1 {
			t.Fatalf("%v %+v", err, q)
		}
		if run := h.analysis(); run == nil || run.PendingVoices != q.Completed[0].PendingVoices {
			t.Fatalf("the clip page and the queue must agree on the voices to check: %+v vs %d", run, q.Completed[0].PendingVoices)
		}
		return q.Completed[0].PendingVoices
	}
	if got := pending(); got != 0 {
		t.Fatalf("no voice left, fully identified: %d", got)
	}
	var job, speaker int64
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.2','pv','done') RETURNING id`, h.clipID).Scan(&job); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,2,$2,'SPEAKER_00',0,2,1) RETURNING id`, h.clipID, job).Scan(&speaker); err != nil {
		t.Fatal(err)
	}
	if got := pending(); got != 1 {
		t.Fatalf("an unknown voice leaves the run partially identified: %d", got)
	}
	// An automatic attribution settles the voice, as in the Verification list.
	if _, err := pool.Exec(ctx, `WITH p AS (INSERT INTO persons(display_name) VALUES('Synthetic') RETURNING id) INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,origin) SELECT $1,id,'automatic','unvalidated','test' FROM p`, speaker); err != nil {
		t.Fatal(err)
	}
	if got := pending(); got != 0 {
		t.Fatalf("an attributed voice is no longer pending: %d", got)
	}
}
