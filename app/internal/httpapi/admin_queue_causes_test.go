//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type causeStep struct {
	Step              string `json:"step"`
	Status            string `json:"status"`
	Reason            string `json:"reason"`
	FailureCode       string `json:"failure_code"`
	RecommendedAction string `json:"recommended_action"`
}

type causeQueue struct {
	Active []struct {
		ClipID int64       `json:"clip_id"`
		Status string      `json:"status"`
		Steps  []causeStep `json:"steps"`
	} `json:"active"`
	Tasks []struct {
		ID                int64  `json:"id"`
		Step              string `json:"step"`
		Status            string `json:"status"`
		LastError         string `json:"last_error"`
		FailureCode       string `json:"failure_code"`
		RecommendedAction string `json:"recommended_action"`
		ClipName          string `json:"clip_name"`
	} `json:"tasks"`
	Summary *struct {
		RunningTasks   int  `json:"running_tasks"`
		WaitingClips   int  `json:"waiting_clips"`
		MaxConcurrency int  `json:"max_concurrency"`
		Paused         bool `json:"paused"`
	} `json:"summary"`
}

func (h *voiceHarness) causeQueue() (causeQueue, []byte) {
	h.t.Helper()
	raw := h.adminCall(http.MethodGet, "/api/admin/analysis/queue", nil, http.StatusOK)
	var q causeQueue
	if err := json.Unmarshal(raw, &q); err != nil {
		h.t.Fatal(err)
	}
	return q, raw
}

func (q causeQueue) step(clipID int64, name string) causeStep {
	for _, r := range q.Active {
		if r.ClipID == clipID {
			for _, s := range r.Steps {
				if s.Step == name {
					return s
				}
			}
		}
	}
	return causeStep{}
}

// A simulated Whisper failure shows its persisted cause and advised action on the failed step,
// a Worker wait shows the stable start_worker remedy, and a deferred retry never does.
func testAdminQueueFailureCauses(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.useSingleAudioTrack()
	ctx := context.Background()
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusOK)
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{}, http.StatusOK)

	// Waiting for a Worker: the reason stays, the remedy is a stable action code.
	q, _ := h.causeQueue()
	if st := q.step(h.clipID, "transcription"); st.Reason != "awaiting_compatible_worker" || st.RecommendedAction != "start_worker" || st.FailureCode != "" {
		t.Fatalf("a Worker wait advises starting a Worker: %+v", st)
	}
	// A pending job whose next attempt is in the future is a deferred retry, not a Worker wait.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET next_attempt_at=now()+interval '1 hour' WHERE clip_id=$1 AND status='pending'`, h.clipID); err != nil {
		t.Fatal(err)
	}
	q, _ = h.causeQueue()
	if st := q.step(h.clipID, "transcription"); st.RecommendedAction != "" {
		t.Fatalf("a deferred job is not a Worker wait: %+v", st)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET next_attempt_at=now() WHERE clip_id=$1 AND status='pending'`, h.clipID); err != nil {
		t.Fatal(err)
	}

	// Three failed attempts: the last cause is persisted; the step fails with it.
	codes := []string{"whisper_network", "whisper_timeout", "whisper_model_missing"}
	for i, code := range codes {
		job := h.mustClaim("the transcription is not claimable")
		if err := h.api.Analysis.ReconcileClip(ctx, h.clipID); err != nil {
			t.Fatal(err)
		}
		q, _ = h.causeQueue()
		if st := q.step(h.clipID, "transcription"); st.Status != "running" || st.RecommendedAction != "" {
			t.Fatalf("a running step has no remedy: %+v", st)
		}
		h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", job.ID), map[string]any{"error": code, "provider": "whisper"}, job.Lease, http.StatusOK)
		if i < len(codes)-1 {
			q, _ = h.causeQueue()
			if st := q.step(h.clipID, "transcription"); st.Status != "retry" || st.RecommendedAction != "" {
				t.Fatalf("a deferred retry advises no Worker: %+v", st)
			}
			if _, err := pool.Exec(ctx, `UPDATE jobs SET next_attempt_at=now() WHERE id=$1`, job.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := h.api.Analysis.ReconcileClip(ctx, h.clipID); err != nil {
		t.Fatal(err)
	}
	q, _ = h.causeQueue()
	st := q.step(h.clipID, "transcription")
	if st.Status != "failed" || st.Reason != "job_dead" || st.FailureCode != "whisper_model_missing" || st.RecommendedAction != "install_transcription_model" {
		t.Fatalf("the dead step shows its cause and action: %+v", st)
	}
	if len(q.Active) != 1 {
		t.Fatalf("the failed run stays visible: %+v", q.Active)
	}
}

// Dead, failed and untracked tasks stay visible with a coded error; raw error text never shows.
func testAdminQueueFailedTasks(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := context.Background()
	var dead, failed, bare int64
	insert := func(step, status, lastError string, out *int64) {
		t.Helper()
		if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,last_error) VALUES($1,$2,'v1',$3,$4) RETURNING id`, h.clipID, step, status, lastError).Scan(out); err != nil {
			t.Fatal(err)
		}
	}
	insert("mystery.future.1", "dead", "", &dead)
	if _, err := pool.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,actor,error_code) VALUES($1,'marked_dead',0,'worker','gpu_out_of_memory')`, dead); err != nil {
		t.Fatal(err)
	}
	insert("mystery.future.2", "failed", "Traceback (most recent call last): /home/secret/token=abc", &failed)
	insert("mystery.future.3", "dead", "", &bare)
	q, raw := h.causeQueue()
	type view struct{ code, action, status string }
	got := map[int64]view{}
	for _, tk := range q.Tasks {
		got[tk.ID] = view{tk.FailureCode, tk.RecommendedAction, tk.Status}
		if strings.Contains(tk.LastError, " ") {
			t.Fatalf("last_error must be a code: %q", tk.LastError)
		}
	}
	if g := got[dead]; g != (view{"gpu_out_of_memory", "check_gpu", "dead"}) {
		t.Fatalf("a dead untracked task shows its cause and action: %+v", g)
	}
	if g := got[failed]; g != (view{"worker_failed", "check_worker_logs", "failed"}) {
		t.Fatalf("a failed task stays visible with a generic code: %+v", g)
	}
	if g := got[bare]; g != (view{"", "", "dead"}) {
		t.Fatalf("a dead task without any cause is still listed: %+v", g)
	}
	if strings.Contains(string(raw), "Traceback") || strings.Contains(string(raw), "secret") {
		t.Fatalf("raw error text leaked: %s", raw)
	}
}

// The summary counts live leases like the scheduler: suspended clips take no slot, an expired
// lease is not running, and the pause is reported.
func testAdminQueueSummary(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := context.Background()
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 3}, http.StatusOK)
	if _, err := pool.Exec(ctx, `WITH c AS (INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at)
 SELECT owner_id,'summary-' || n,1,now(),now() FROM clips,generate_series(1,6) n WHERE id=$1 RETURNING id,original_name)
 INSERT INTO clip_analysis_runs(clip_id,scope) SELECT id,'full' FROM c`, h.clipID); err != nil {
		t.Fatal(err)
	}
	lease := func(name, step, status, expires string) {
		t.Helper()
		owner, exp, token := "NULL", "NULL", "NULL"
		if status == "leased" {
			owner, exp, token = "(SELECT id FROM workers LIMIT 1)", expires, "sha256('x')"
		}
		if _, err := pool.Exec(ctx, fmt.Sprintf(`INSERT INTO jobs(clip_id,step,input_version,status,lease_owner,lease_expires_at,lease_token_hash) SELECT id,$1,'v1',$2,%s,%s,%s FROM clips WHERE original_name=$3`, owner, exp, token), step, status, name); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{"summary-1", "summary-2", "summary-3"} {
		lease(n, "transcription.audio.1", "leased", "now()+interval '5 minutes'")
	}
	lease("summary-4", "transcription.audio.1", "leased", "now()-interval '1 minute'") // lost lease
	lease("summary-5", "transcription.audio.1", "pending", "")                         // ready, no free slot
	// summary-6 and the harness clip are suspended: no job at all.
	q, _ := h.causeQueue()
	if q.Summary == nil {
		t.Fatal("the queue carries a summary")
	}
	waiting := len(q.Active) - 3
	if q.Summary.RunningTasks != 3 || q.Summary.MaxConcurrency != 3 || q.Summary.Paused || q.Summary.WaitingClips != waiting {
		t.Fatalf("summary %+v, %d active runs", *q.Summary, len(q.Active))
	}
	// A slot frees up and the queue is paused.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE clip_id=(SELECT id FROM clips WHERE original_name='summary-1')`); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": true, "max_concurrency": 2}, http.StatusOK)
	q, _ = h.causeQueue()
	if q.Summary.RunningTasks != 2 || !q.Summary.Paused || q.Summary.MaxConcurrency != 2 || q.Summary.WaitingClips != waiting+1 {
		t.Fatalf("summary after the change %+v", *q.Summary)
	}
}

func testAdminQueueCausesAccess(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.userCall(h.newUser("causes-member"), http.MethodGet, "/api/admin/analysis/queue", nil, http.StatusForbidden)
	resp, err := http.Get(h.ts.URL + "/api/admin/analysis/queue")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", resp.StatusCode)
	}
}
