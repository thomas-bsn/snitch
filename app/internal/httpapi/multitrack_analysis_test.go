//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/analysis"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
	"snitch/internal/multitrack"
)

type speechDiagnostics struct{}

func (speechDiagnostics) DiagnoseAudio(_ context.Context, p audioplans.Plan, track contract.AudioTrackInventory) (analysis.AudioObservation, error) {
	return analysis.AudioObservation{Status: "speech", Cause: "synthetic_speech", Intervals: []contract.AudioSpeechInterval{{Start: 0, End: 1}}}, nil
}

// scriptedDiagnostics replaces the real detector by contract-shaped observations.
// A track mapped to an error simulates a detector crash.
type scriptedDiagnostics struct {
	observations map[int]analysis.AudioObservation
	failures     map[int]error
	calls        map[int]int
}

func (d *scriptedDiagnostics) DiagnoseAudio(_ context.Context, _ audioplans.Plan, track contract.AudioTrackInventory) (analysis.AudioObservation, error) {
	if d.calls == nil {
		d.calls = map[int]int{}
	}
	d.calls[track.Index]++
	if err := d.failures[track.Index]; err != nil {
		return analysis.AudioObservation{}, err
	}
	return d.observations[track.Index], nil
}

var (
	speechObservation   = analysis.AudioObservation{Status: "speech", Cause: "synthetic_speech", Intervals: []contract.AudioSpeechInterval{{Start: 0, End: 1}}}
	silentObservation   = analysis.AudioObservation{Status: "no_speech", Cause: "synthetic_silence"}
	ambiguousObervation = analysis.AudioObservation{Status: "ambiguous", Cause: "synthetic_ambiguity"}
)

func audioInput(t *testing.T, pool *pgxpool.Pool, mode string, mutate func(*audioplans.Input)) audioplans.Input {
	t.Helper()
	var input audioplans.Input
	if err := json.Unmarshal(audioplans.ContractFixture(), &input); err != nil {
		t.Fatal(err)
	}
	input.Mode = contract.AudioPlanInputMode(mode)
	if mutate != nil {
		mutate(&input)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO transcription_vocabulary_versions(version,terms) VALUES('vocab-v1','[]') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	return input
}

func childStatuses(t *testing.T, pool *pgxpool.Pool, planID int64) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT c.stage||':'||c.track_index,j.status FROM audio_plan_children c JOIN jobs j ON j.id=c.job_id WHERE c.plan_id=$1`, planID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err = rows.Scan(&k, &v); err != nil {
			t.Fatal(err)
		}
		out[k] = v
	}
	return out
}

type collectedAudio struct{ results []multitrack.Result }

func (c *collectedAudio) PublishAudio(_ context.Context, _ pgx.Tx, r multitrack.Result) error {
	c.results = append(c.results, r)
	return nil
}

func testMultitrackAnalysis(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	var input audioplans.Input
	if err := json.Unmarshal(audioplans.ContractFixture(), &input); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO transcription_vocabulary_versions(version,terms) VALUES('vocab-v1','[]') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	publisher := &collectedAudio{}
	service := analysis.NewAudio(pool, speechDiagnostics{}, publisher)
	plan, err := service.Start(ctx, h.clipID, input)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Reconcile(ctx, plan.Id); err != nil {
		t.Fatal(err)
	}
	state, err := service.State(ctx, plan.Id)
	if err != nil || len(state.Tracks) != 2 || state.Status != "waiting" {
		t.Fatalf("waiting on two tracks: %+v %v", state, err)
	}
	if len(publisher.results) != 0 {
		t.Fatal("published before children")
	}
	// A new service instance must follow the same children after a server restart.
	restarted := analysis.NewAudio(pool, speechDiagnostics{}, publisher)
	if err = restarted.Reconcile(ctx, plan.Id); err != nil {
		t.Fatal(err)
	}
	again, err := restarted.State(ctx, plan.Id)
	if err != nil || again.Tracks[0].JobID != state.Tracks[0].JobID {
		t.Fatal("restart duplicated work", again, err)
	}
	if _, ok := h.claim(); ok {
		t.Fatal("old wildcard Worker claimed a multitrack task")
	}
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := restarted.Reconcile(ctx, plan.Id); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	first := h.claimAudio()
	var task struct {
		AudioPlan       map[string]any `json:"audio_plan"`
		Pipeline        map[string]any `json:"pipeline"`
		PipelineVersion string         `json:"pipeline_version"`
	}
	if e := json.Unmarshal(h.workerCall("GET", fmt.Sprintf("/api/worker/jobs/%d/input", first.ID), nil, first.Lease, 200), &task); e != nil || task.AudioPlan == nil || task.PipelineVersion != "pipeline-v1" {
		t.Fatal("pinned task", task, e)
	}
	h.complete(first, audioOutput(plan, 2), 200)
	if e := restarted.Reconcile(ctx, plan.Id); e != nil {
		t.Fatal(e)
	}
	if len(publisher.results) != 0 {
		t.Fatal("first sibling published alone")
	}
	second := h.claimAudio()
	h.complete(second, audioOutput(plan, 7), 200)
	if e := restarted.Reconcile(ctx, plan.Id); e != nil {
		t.Fatal(e)
	}
	final, e := restarted.State(ctx, plan.Id)
	if e != nil || final.Status != "done" || len(publisher.results) != 1 || len(publisher.results[0].Tracks) != 2 {
		t.Fatal("complete aggregate", final, publisher, e)
	}
	if e = restarted.Reconcile(ctx, plan.Id); e != nil || len(publisher.results) != 1 {
		t.Fatal("duplicate publication", e)
	}

}

func (h *voiceHarness) claimAudio() claimedJob {
	h.t.Helper()
	h.workerCall("POST", "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"transcription.audio.*", multitrack.Capability}}, "", 200)
	var claim struct {
		Job        *claimedJob `json:"job"`
		LeaseToken string      `json:"lease_token"`
	}
	if err := json.Unmarshal(h.workerCall("POST", "/api/worker/jobs/claim", nil, "", 200), &claim); err != nil || claim.Job == nil {
		h.t.Fatal("missing compatible audio claim", err)
	}
	claim.Job.Lease = claim.LeaseToken
	return *claim.Job
}
func audioOutput(p audioplans.Plan, track int) map[string]any {
	return map[string]any{"provider": "synthetic", "model": "synthetic-model", "version": "1", "audio_stream_index": track, "time_base": "clip", "audio_plan_id": p.Id, "audio_plan_version": p.InputVersion, "segments": []any{map[string]any{"start": 1.5, "end": 1.8, "text": "synthetic", "words": []any{map[string]any{"start": 1.5, "end": 1.8, "word": "synthetic"}}}}}
}

// A child that dies after its sibling succeeded must not redo the sibling, and a
// retry of only that child completes the plan with a single publication.
func testMultitrackChildFailureAndRetry(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	publisher := &collectedAudio{}
	service := analysis.NewAudio(pool, &scriptedDiagnostics{observations: map[int]analysis.AudioObservation{2: speechObservation, 7: speechObservation}}, publisher)
	plan, err := service.Start(ctx, h.clipID, audioInput(t, pool, "manual", nil))
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Reconcile(ctx, plan.Id); err != nil {
		t.Fatal(err)
	}
	// No compatible Worker: a legacy wildcard Worker leaves both children queued.
	if _, ok := h.claim(); ok {
		t.Fatal("legacy Worker claimed a multitrack child")
	}
	waiting, err := service.State(ctx, plan.Id)
	if err != nil || waiting.Status != "waiting" || waiting.Tracks[0].Status != "queued" || waiting.Tracks[1].Status != "queued" {
		t.Fatalf("queued without compatible Worker: %+v %v", waiting, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE jobs SET max_attempts=1 WHERE id=$1`, waiting.Tracks[1].JobID); err != nil {
		t.Fatal(err)
	}
	ok := h.claimAudio()
	h.complete(ok, audioOutput(plan, 2), 200)
	bad := h.claimAudio()
	h.workerCall("POST", fmt.Sprintf("/api/worker/jobs/%d/fail", bad.ID), map[string]any{"error": "worker_failed"}, bad.Lease, 200)
	if err = service.Reconcile(ctx, plan.Id); err != nil {
		t.Fatal(err)
	}
	failed, err := service.State(ctx, plan.Id)
	if err != nil || failed.Status != "failed" || failed.Cause != "audio_child_failed" || failed.Tracks[0].Status != "done" || failed.Tracks[1].Status != "dead" || failed.Tracks[1].Cause != "worker_failed" {
		t.Fatalf("failed child is visible beside its successful sibling: %+v %v", failed, err)
	}
	if len(publisher.results) != 0 {
		t.Fatal("partial plan published")
	}
	if err = service.Retry(ctx, plan.Id, 2, "transcription"); err != analysis.ErrNotRetryableAudio {
		t.Fatalf("a successful child must not be retried: %v", err)
	}
	if err = service.Retry(ctx, plan.Id, 7, "transcription"); err != nil {
		t.Fatal(err)
	}
	if err = service.Retry(ctx, plan.Id, 7, "transcription"); err != analysis.ErrNotRetryableAudio {
		t.Fatalf("a queued child must not be retried twice: %v", err)
	}
	retry := h.claimAudio()
	if retry.ID != bad.ID {
		t.Fatalf("retry created another job: %d vs %d", retry.ID, bad.ID)
	}
	h.complete(retry, audioOutput(plan, 7), 200)
	for i := 0; i < 2; i++ {
		if err = service.Reconcile(ctx, plan.Id); err != nil {
			t.Fatal(err)
		}
	}
	done, err := service.State(ctx, plan.Id)
	if err != nil || done.Status != "done" || len(publisher.results) != 1 || len(publisher.results[0].Tracks) != 2 || done.Tracks[0].JobID != failed.Tracks[0].JobID {
		t.Fatalf("retry completes only the missing child: %+v %v", done, err)
	}
	var jobs int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE step LIKE 'transcription.audio.%'`).Scan(&jobs); err != nil || jobs != 2 {
		t.Fatalf("no duplicate transcription jobs: %d %v", jobs, err)
	}
}

// Cancelling stops only the plan's own children, frees the clip for a new
// analysis and rejects the late result of a Worker that held a lease.
func testMultitrackCancel(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 2)
	publisher := &collectedAudio{}
	diagnostics := &scriptedDiagnostics{observations: map[int]analysis.AudioObservation{2: speechObservation, 7: speechObservation}}
	service := analysis.NewAudio(pool, diagnostics, publisher)
	first, err := service.Start(ctx, h.clipIDs[0], audioInput(t, pool, "manual", nil))
	if err != nil {
		t.Fatal(err)
	}
	other, err := service.Start(ctx, h.clipIDs[1], audioInput(t, pool, "manual", nil))
	if err != nil {
		t.Fatal(err)
	}
	// Only one active analysis per clip; the same input is idempotent.
	different := audioInput(t, pool, "manual", func(i *audioplans.Input) { i.TrackIndices = []int{2}; i.PipelineVersion = "pipeline-v2" })
	if _, err = service.Start(ctx, h.clipIDs[0], different); err != analysis.ErrActiveAudio {
		t.Fatalf("second active analysis: %v", err)
	}
	if again, e := service.Start(ctx, h.clipIDs[0], audioInput(t, pool, "manual", nil)); e != nil || again.Id != first.Id {
		t.Fatalf("same plan start must be idempotent: %+v %v", again, e)
	}
	for _, plan := range []audioplans.Plan{first, other} {
		if err = service.Reconcile(ctx, plan.Id); err != nil {
			t.Fatal(err)
		}
	}
	leased := h.claimAudio()
	var leasedPlan int64
	if err = pool.QueryRow(ctx, `SELECT plan_id FROM audio_plan_children WHERE job_id=$1`, leased.ID).Scan(&leasedPlan); err != nil {
		t.Fatal(err)
	}
	cancelled, kept := first, other
	if leasedPlan == other.Id {
		cancelled, kept = other, first
	}
	if err = service.Cancel(ctx, cancelled.Id); err != nil {
		t.Fatal(err)
	}
	if err = service.Cancel(ctx, cancelled.Id); err != nil {
		t.Fatalf("cancel is idempotent: %v", err)
	}
	for key, status := range childStatuses(t, pool, cancelled.Id) {
		if status != "cancelled" && key[:4] != "diag" {
			t.Fatalf("cancelled plan child %s is %s", key, status)
		}
	}
	for key, status := range childStatuses(t, pool, kept.Id) {
		if key[:4] != "diag" && status != "pending" && status != "leased" {
			t.Fatalf("another plan's child %s was touched: %s", key, status)
		}
	}
	h.complete(leased, audioOutput(cancelled, 2), 409)
	if err = service.Reconcile(ctx, cancelled.Id); err != nil {
		t.Fatal(err)
	}
	state, err := service.State(ctx, cancelled.Id)
	if err != nil || state.Status != "blocked" || state.Cause != "run_cancelled" || len(publisher.results) != 0 {
		t.Fatalf("cancelled plan plans nothing more: %+v %v", state, err)
	}
	var cancelledJobs int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE status='cancelled'`).Scan(&cancelledJobs); err != nil || cancelledJobs != 2 {
		t.Fatalf("only the two children of the cancelled plan: %d %v", cancelledJobs, err)
	}
	// The clip is free again for a different analysis.
	cancelledClip := h.clipIDs[0]
	if cancelled.Id == other.Id {
		cancelledClip = h.clipIDs[1]
	}
	if _, err = service.Start(ctx, cancelledClip, different); err != nil {
		t.Fatalf("new analysis after cancellation: %v", err)
	}
}

// Ambiguous tracks block Auto until an explicit, recorded decision is given in a
// new plan; failed detectors are visible and retryable; silence alone ends empty.
func testMultitrackDiagnosticDecisions(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	publisher := &collectedAudio{}
	diagnostics := &scriptedDiagnostics{observations: map[int]analysis.AudioObservation{2: speechObservation, 7: ambiguousObervation}}
	service := analysis.NewAudio(pool, diagnostics, publisher)
	auto, err := service.Start(ctx, h.clipID, audioInput(t, pool, "auto", nil))
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Reconcile(ctx, auto.Id); err != nil {
		t.Fatal(err)
	}
	state, err := service.State(ctx, auto.Id)
	if err != nil || state.Status != "blocked" || state.Cause != "diagnostic_ambiguous" || state.Tracks[1].Status != "ambiguous" {
		t.Fatalf("ambiguity awaits a decision: %+v %v", state, err)
	}
	if got := childStatuses(t, pool, auto.Id); len(got) != 2 || got["transcription:2"] != "" {
		t.Fatalf("no ASR may start before every diagnostic is settled: %v", got)
	}
	if err = service.Cancel(ctx, auto.Id); err != nil {
		t.Fatal(err)
	}
	forced := audioInput(t, pool, "manual", func(i *audioplans.Input) {
		i.Decisions = []contract.AudioTrackDecision{
			{TrackIndex: 2, Action: "include", ActorId: 1, DecidedAt: time.Unix(1, 0).UTC(), Reason: "operator listened"},
			{TrackIndex: 7, Action: "include", Forced: true, ActorId: 1, DecidedAt: time.Unix(1, 0).UTC(), Reason: "operator forced the ambiguous track"},
		}
	})
	manual, err := service.Start(ctx, h.clipID, forced)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Reconcile(ctx, manual.Id); err != nil {
		t.Fatal(err)
	}
	if got := childStatuses(t, pool, manual.Id); got["transcription:2"] != "pending" || got["transcription:7"] != "pending" {
		t.Fatalf("forced decision uses the exact indices: %v", got)
	}
	if err = service.Cancel(ctx, manual.Id); err != nil {
		t.Fatal(err)
	}

	// A detector crash is a visible failure that Retry recomputes as attempt 2.
	diagnostics = &scriptedDiagnostics{observations: map[int]analysis.AudioObservation{2: speechObservation, 7: speechObservation}, failures: map[int]error{7: errors.New("detector_crashed")}}
	service = analysis.NewAudio(pool, diagnostics, publisher)
	crashed, err := service.Start(ctx, h.clipID, audioInput(t, pool, "auto", func(i *audioplans.Input) { i.DetectorConfigVersion = "cfg-v2" }))
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Reconcile(ctx, crashed.Id); err != nil {
		t.Fatal(err)
	}
	state, _ = service.State(ctx, crashed.Id)
	if state.Cause != "diagnostic_failed" || state.Tracks[1].Status != "failed" || state.Tracks[1].Cause != "detector_crashed" {
		t.Fatalf("diagnostic failure is visible: %+v", state)
	}
	delete(diagnostics.failures, 7)
	if err = service.Retry(ctx, crashed.Id, 7, "diagnostic"); err != nil {
		t.Fatal(err)
	}
	if err = service.Reconcile(ctx, crashed.Id); err != nil {
		t.Fatal(err)
	}
	state, _ = service.State(ctx, crashed.Id)
	if state.Tracks[1].Diagnostic.Attempt != 2 || state.Tracks[1].Diagnostic.Status != "speech" || diagnostics.calls[2] != 1 || diagnostics.calls[7] != 2 {
		t.Fatalf("only the failed diagnostic reruns: %+v calls=%v", state, diagnostics.calls)
	}
	if got := childStatuses(t, pool, crashed.Id); got["transcription:2"] != "pending" || got["transcription:7"] != "pending" {
		t.Fatalf("diagnosed speech tracks are transcribed: %v", got)
	}
	if err = service.Cancel(ctx, crashed.Id); err != nil {
		t.Fatal(err)
	}

	// Every track without speech: no ASR job, the empty result is published.
	diagnostics = &scriptedDiagnostics{observations: map[int]analysis.AudioObservation{2: silentObservation, 7: silentObservation}}
	service = analysis.NewAudio(pool, diagnostics, publisher)
	silent, err := service.Start(ctx, h.clipID, audioInput(t, pool, "auto", func(i *audioplans.Input) { i.DetectorConfigVersion = "cfg-v3" }))
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Reconcile(ctx, silent.Id); err != nil {
		t.Fatal(err)
	}
	state, _ = service.State(ctx, silent.Id)
	if state.Status != "done" || len(publisher.results) != 1 || !publisher.results[0].Empty || len(publisher.results[0].Tracks) != 0 {
		t.Fatalf("all tracks without speech: %+v %+v", state, publisher.results)
	}
	var asr int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audio_plan_children WHERE plan_id=$1 AND stage='transcription'`, silent.Id).Scan(&asr); err != nil || asr != 0 {
		t.Fatalf("no ASR for silent tracks: %d %v", asr, err)
	}
}
