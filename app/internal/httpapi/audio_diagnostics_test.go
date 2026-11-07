//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/analysis"
	"snitch/internal/audioplans"
	"testing"
)

func testAudioDiagnostics(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.transcribe()
	ctx := context.Background()
	var run int64
	if err := pool.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,status) VALUES($1,'full','completed') RETURNING id`, h.clipID).Scan(&run); err != nil {
		t.Fatal(err)
	}
	var input audioplans.Input
	_ = json.Unmarshal(audioplans.ContractFixture(), &input)
	input.DetectorVersion = "silero-v6.2.1-7e30209a3e901f9842f81b225f3e93d8199902b1"
	input.DetectorConfigVersion = "channels-16k-512-hi050-lo020-min096-v1"
	plan, err := audioplans.New(pool).CreatePlan(ctx, run, input)
	if err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d/diagnostics", h.clipID, plan.Id)
	var state audioplans.State
	raw := h.adminCall("POST", path, map[string]any{}, 200)
	if err = json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Children) != 2 {
		t.Fatalf("all tracks must be queued: %s", raw)
	}
	// An old worker's generic transcription/voice capabilities cannot take diagnostics.
	if job, ok := h.claim(); ok {
		t.Fatalf("legacy worker claimed diagnostic: %+v", job)
	}
	raw = h.adminCall("POST", path, map[string]any{}, 200)
	if h.count(`SELECT count(*) FROM audio_plan_children WHERE plan_id=$1`, plan.Id) != 2 {
		t.Fatal("duplicate jobs")
	}
	for _, index := range []int{2, 7} {
		h.workerCall("POST", "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"audio.diagnostic.*"}}, "", 200)
		var claim struct {
			Job   *claimedJob `json:"job"`
			Lease string      `json:"lease_token"`
		}
		// Capability alone does not prove that the pinned model is installed.
		if index == 2 {
			_ = json.Unmarshal(h.workerCall("POST", "/api/worker/jobs/claim", nil, "", 200), &claim)
			if claim.Job != nil {
				t.Fatal("model-less worker took diagnostic")
			}
			order := h.order(h.fixtureWorkerID(), "silero-vad-v6.2.1", "diagnostic-install-01", 201)
			h.workerCall("POST", "/api/worker/voice-model-orders/"+order.OrderID+"/ack", map[string]any{"accepted": true}, "", 204)
			h.workerCall("POST", "/api/worker/voice-models/status", map[string]any{"order_id": order.OrderID, "model_id": order.ModelID, "revision": order.Revision, "state": "ready", "cause_code": "", "bytes_received": 2272526, "bytes_total": 2272526, "occupied_bytes": 2272526}, "", 204)
		}
		if err = json.Unmarshal(h.workerCall("POST", "/api/worker/jobs/claim", nil, "", 200), &claim); err != nil || claim.Job == nil {
			t.Fatalf("claim %v %+v", err, claim)
		}
		job := *claim.Job
		job.Lease = claim.Lease
		if job.Step != fmt.Sprintf("audio.diagnostic.%d", index) {
			t.Fatal(job)
		}
		raw = h.workerCall("GET", fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, 200)
		var task struct {
			Diagnostic struct {
				PlanID       int64    `json:"plan_id"`
				TrackIndex   int      `json:"track_index"`
				InputVersion string   `json:"input_version"`
				Start        float64  `json:"start_seconds"`
				Duration     *float64 `json:"duration_seconds"`
			} `json:"audio_diagnostic"`
		}
		_ = json.Unmarshal(raw, &task)
		if task.Diagnostic.PlanID != plan.Id || task.Diagnostic.TrackIndex != index || task.Diagnostic.InputVersion != plan.InputVersion {
			t.Fatalf("leased immutable input: %s", raw)
		}
		// The stream's own offset/duration (never the container's) reach the Worker; unknown stays null.
		if index == 7 && (task.Diagnostic.Start != 1.25 || task.Diagnostic.Duration != nil) || index == 2 && (task.Diagnostic.Start != 0 || task.Diagnostic.Duration == nil || *task.Diagnostic.Duration != 12) {
			t.Fatalf("stream timeline: %s", raw)
		}
		payload := map[string]any{"status": "speech", "cause": "neural_speech", "intervals": []map[string]any{{"start": 1.5, "end": 2.0}}, "detector_version": input.DetectorVersion, "detector_config_version": input.DetectorConfigVersion}
		h.complete(job, payload, 200)
		h.complete(job, payload, 200)
	}
	raw = h.adminCall("GET", fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d", h.clipID, plan.Id), nil, 200)
	assertAudioContract(t, "AudioPlanState", raw)
	_ = json.Unmarshal(raw, &state)
	if state.Diagnostics[0].Status != "speech" || state.Diagnostics[1].Status != "speech" || state.Diagnostics[1].JobId == nil {
		t.Fatalf("persisted diagnostics: %s", raw)
	}
	if got := h.clipTranscript(h.adminUser()); got.Transcription.Segments[0].Text != "premier" {
		t.Fatal("diagnostic changed transcript")
	}
}

// Errors, missing evidence and bad reports are never turned into a reliable absence of speech.
func testAudioDiagnosticOutcomes(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.transcribe()
	ctx := context.Background()
	var run int64
	if err := pool.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,status) VALUES($1,'full','completed') RETURNING id`, h.clipID).Scan(&run); err != nil {
		t.Fatal(err)
	}
	var input audioplans.Input
	_ = json.Unmarshal(audioplans.ContractFixture(), &input)
	input.DetectorVersion = "silero-v6.2.1-7e30209a3e901f9842f81b225f3e93d8199902b1"
	input.DetectorConfigVersion = "channels-16k-512-hi050-lo020-min096-v1"
	plan, err := audioplans.New(pool).CreatePlan(ctx, run, input)
	if err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d/diagnostics", h.clipID, plan.Id)
	h.adminCall("POST", "/api/admin/clips/999999/audio-plans/1/diagnostics", map[string]any{}, 404)
	h.adminCall("POST", fmt.Sprintf("/api/admin/clips/%d/audio-plans/999999/diagnostics", h.clipID), map[string]any{}, 404)
	h.adminCall("POST", path, map[string]any{}, 200)
	h.workerCall("POST", "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"audio.diagnostic.*"}}, "", 200)
	order := h.order(h.fixtureWorkerID(), "silero-vad-v6.2.1", "diagnostic-install-02", 201)
	h.workerCall("POST", "/api/worker/voice-model-orders/"+order.OrderID+"/ack", map[string]any{"accepted": true}, "", 204)
	h.workerCall("POST", "/api/worker/voice-models/status", map[string]any{"order_id": order.OrderID, "model_id": order.ModelID, "revision": order.Revision, "state": "ready", "cause_code": "", "bytes_received": 2272526, "bytes_total": 2272526, "occupied_bytes": 2272526}, "", 204)
	if order.Kind != "vad" || order.Engine != "silero-jit" || order.Repo != "snakers4/silero-vad" {
		t.Fatalf("order contract: %+v", order)
	}
	base := map[string]any{"detector_version": input.DetectorVersion, "detector_config_version": input.DetectorConfigVersion}
	with := func(extra map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range base {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	var firstClaim struct {
		Job   *claimedJob `json:"job"`
		Lease string      `json:"lease_token"`
	}
	_ = json.Unmarshal(h.workerCall("POST", "/api/worker/jobs/claim", nil, "", 200), &firstClaim)
	if firstClaim.Job == nil {
		t.Fatal("job not claimable")
	}
	first := *firstClaim.Job
	first.Lease = firstClaim.Lease
	for _, bad := range []map[string]any{
		with(map[string]any{"status": "unexamined", "cause": "x", "intervals": []any{}}),
		with(map[string]any{"status": "no_speech", "cause": "x", "intervals": []map[string]any{{"start": 1, "end": 2}}}),
		with(map[string]any{"status": "speech", "cause": "x", "intervals": []any{}}),
		with(map[string]any{"status": "speech", "cause": "x", "intervals": []map[string]any{{"start": 3, "end": 2}}}),
		with(map[string]any{"status": "no_speech", "cause": "", "intervals": []any{}}),
		{"status": "no_speech", "cause": "x", "intervals": []any{}, "detector_version": "other", "detector_config_version": input.DetectorConfigVersion},
	} {
		h.complete(first, bad, 400)
	}
	// A failed or ambiguous observation keeps its cause and is not a clean exclusion.
	for i, observation := range []map[string]any{with(map[string]any{"status": "failed", "cause": "model_missing", "intervals": []any{}}), with(map[string]any{"status": "ambiguous", "cause": "insufficient_evidence", "intervals": []any{}})} {
		job := first
		if i == 1 {
			var claim struct {
				Job   *claimedJob `json:"job"`
				Lease string      `json:"lease_token"`
			}
			_ = json.Unmarshal(h.workerCall("POST", "/api/worker/jobs/claim", nil, "", 200), &claim)
			if claim.Job == nil {
				t.Fatalf("claim %d", i)
			}
			job = *claim.Job
			job.Lease = claim.Lease
		}
		h.complete(job, observation, 200)
	}
	var state audioplans.State
	raw := h.adminCall("GET", fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d", h.clipID, plan.Id), nil, 200)
	_ = json.Unmarshal(raw, &state)
	got := map[int]string{}
	for _, d := range state.Diagnostics {
		got[d.TrackIndex] = string(d.Status) + "/" + d.Cause
	}
	if got[2] != "failed/model_missing" || got[7] != "ambiguous/insufficient_evidence" {
		t.Fatalf("diagnostics: %s", raw)
	}
	for _, d := range state.Diagnostics {
		if d.Status == "no_speech" {
			t.Fatalf("converted to absence: %s", raw)
		}
	}
}

// The production boundary of the multitrack orchestration: diagnostics run on a Worker, are awaited without
// inference, and a failed observation is retried as the next attempt before any transcription starts.
func testMultitrackWorkerDiagnostics(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	service := analysis.NewAudio(pool, analysis.WorkerAudioDiagnostics{}, &collectedAudio{})
	plan, err := service.Start(ctx, h.clipID, audioInput(t, pool, "auto", nil))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 { // replaying the reconciliation never duplicates a job
		if err = service.Reconcile(ctx, plan.Id); err != nil {
			t.Fatal(err)
		}
	}
	got := childStatuses(t, pool, plan.Id)
	if len(got) != 2 || got["diagnostic:2"] != "pending" || got["diagnostic:7"] != "pending" {
		t.Fatalf("every inventoried track is examined first, no ASR yet: %v", got)
	}
	h.workerCall("POST", "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"audio.diagnostic.*"}}, "", 200)
	order := h.order(h.fixtureWorkerID(), "silero-vad-v6.2.1", "diagnostic-install-03", 201)
	h.workerCall("POST", "/api/worker/voice-model-orders/"+order.OrderID+"/ack", map[string]any{"accepted": true}, "", 204)
	h.workerCall("POST", "/api/worker/voice-models/status", map[string]any{"order_id": order.OrderID, "model_id": order.ModelID, "revision": order.Revision, "state": "ready", "cause_code": "", "bytes_received": 1, "bytes_total": 1, "occupied_bytes": 1}, "", 204)
	claim := func() claimedJob {
		var c struct {
			Job   *claimedJob `json:"job"`
			Lease string      `json:"lease_token"`
		}
		_ = json.Unmarshal(h.workerCall("POST", "/api/worker/jobs/claim", nil, "", 200), &c)
		if c.Job == nil {
			t.Fatalf("diagnostic not claimable: %v", childStatuses(t, pool, plan.Id))
		}
		job := *c.Job
		job.Lease = c.Lease
		return job
	}
	observation := func(status, cause string, intervals []map[string]any) map[string]any {
		return map[string]any{"status": status, "cause": cause, "intervals": intervals, "detector_version": "vad-v1", "detector_config_version": "cfg-v1"}
	}
	for range 2 {
		job := claim()
		if job.Step == "audio.diagnostic.2" {
			h.complete(job, observation("speech", "neural_speech", []map[string]any{{"start": 0.5, "end": 1.5}}), 200)
		} else {
			h.complete(job, observation("failed", "decode_failed", nil), 200)
		}
	}
	if err = service.Reconcile(ctx, plan.Id); err != nil {
		t.Fatal(err)
	}
	state, _ := service.State(ctx, plan.Id)
	if state.Cause != "diagnostic_failed" || state.Tracks[1].Status != "failed" || state.Tracks[1].Cause != "decode_failed" {
		t.Fatalf("a failed examination is visible and blocks, never 'no speech': %+v", state)
	}
	if got = childStatuses(t, pool, plan.Id); got["transcription:2"] != "" {
		t.Fatalf("no ASR before every diagnostic is settled: %v", got)
	}
	if err = service.Retry(ctx, plan.Id, 7, "diagnostic"); err != nil {
		t.Fatal(err)
	}
	retried := claim()
	var attempt int
	_ = pool.QueryRow(ctx, `SELECT attempts FROM jobs WHERE id=$1`, retried.ID).Scan(&attempt)
	if retried.Step != "audio.diagnostic.7" || attempt != 2 {
		t.Fatalf("retry is the next attempt of the same durable job: %+v attempt %d", retried, attempt)
	}
	h.complete(retried, observation("speech", "neural_speech", []map[string]any{{"start": 0.1, "end": 0.9}}), 200)
	if err = service.Reconcile(ctx, plan.Id); err != nil {
		t.Fatal(err)
	}
	if got = childStatuses(t, pool, plan.Id); got["transcription:2"] != "pending" || got["transcription:7"] != "pending" || len(got) != 4 {
		t.Fatalf("transcription follows complete, successful diagnostics: %v", got)
	}
}
