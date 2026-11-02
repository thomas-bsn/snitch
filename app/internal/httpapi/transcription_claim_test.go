//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"snitch/internal/workers"
)

// testTranscriptionClaimVersionGate: a Worker whose validated transcription configuration is the desired
// one receives a pending transcription job; a Worker validated on a superseded configuration does not.
func testTranscriptionClaimVersionGate(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := t.Context()
	var desired struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPut, "/api/admin/transcription-settings", map[string]any{"mode": "enabled", "endpoint": "http://127.0.0.1:9000", "model": "fixture-model", "language": "fr"}, http.StatusOK), &desired); err != nil {
		t.Fatal(err)
	}
	var jobID int64
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version) VALUES($1,'transcription.audio.1','t4-a1-r1-vabc-ppipeline-0123456789ab') RETURNING id`, h.clipID).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"transcription.audio.*"}}, "", http.StatusOK)
	h.workerCall(http.MethodGet, "/api/worker/transcription-config", nil, "", http.StatusOK)

	claim := func() (id int64) {
		t.Helper()
		var assignment struct {
			Job *struct {
				ID int64 `json:"id"`
			} `json:"job"`
		}
		if err := json.Unmarshal(h.workerCall(http.MethodPost, "/api/worker/jobs/claim", nil, "", http.StatusOK), &assignment); err != nil {
			t.Fatal(err)
		}
		if assignment.Job != nil {
			return assignment.Job.ID
		}
		return 0
	}
	// Stale validation: the Worker validated another configuration than the desired one.
	if _, err := pool.Exec(ctx, `UPDATE worker_transcription_config SET active_version='transcription-stale'`); err != nil {
		t.Fatal(err)
	}
	if got := claim(); got != 0 {
		t.Fatalf("worker on a superseded configuration was given job %d", got)
	}
	// Validated desired configuration: the pending job must be leased without any replanning.
	h.workerCall(http.MethodPost, "/api/worker/transcription-config/status", map[string]any{"version": desired.Version, "active": true}, "", http.StatusOK)
	if got := claim(); got != jobID {
		t.Fatalf("pending transcription job %d was not leased to a worker on the desired configuration (got %d)", jobID, got)
	}
}

// testPendingTranscriptionDiagnosis: a pending transcription job never waits without a diagnosed cause, the
// durable ones become one deduplicated administrator incident, and everything heals when the cause is lifted.
func testPendingTranscriptionDiagnosis(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := t.Context()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	var desired struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPut, "/api/admin/transcription-settings", map[string]any{"mode": "enabled", "endpoint": "http://127.0.0.1:9000", "model": "fixture-model", "language": "fr"}, http.StatusOK), &desired); err != nil {
		t.Fatal(err)
	}
	var jobID int64
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,created_at) VALUES($1,'transcription.audio.1','t4-a1-r1-vabc-ppipeline-0123456789ab',now()-interval '10 minutes') RETURNING id`, h.clipID).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	cause := func() string {
		t.Helper()
		got, err := h.api.Jobs.WaitCause(ctx, jobID, workers.ConnectedWindow)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	admin := h.adminUser()
	blocked := func() []incidentT {
		t.Helper()
		h.api.syncIncidents(ctx)
		items, _ := h.incidents(admin)
		var active []incidentT
		for _, i := range items {
			if i.State == "active" {
				active = append(active, i)
			}
		}
		return active
	}

	exec(`UPDATE workers SET last_seen_at=now()-interval '1 hour'`)
	if got := cause(); got != "no_compatible_worker" {
		t.Fatalf("no connected worker: %q", got)
	}
	h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"transcription.audio.*"}}, "", http.StatusOK)
	h.workerCall(http.MethodGet, "/api/worker/transcription-config", nil, "", http.StatusOK)
	exec(`UPDATE worker_transcription_config SET active_version='transcription-stale'`)
	if got := cause(); got != "transcription_version_mismatch" {
		t.Fatalf("stale validated version: %q", got)
	}
	exec(`UPDATE worker_transcription_config SET error_code='model_missing'`)
	if got := cause(); got != "transcription_not_validated" {
		t.Fatalf("failed validation: %q", got)
	}
	exec(`UPDATE worker_transcription_config SET error_code=''`)

	// Durable cause + past the grace period: a single, repeated-sync-safe incident that names the cause and the job.
	for i := 0; i < 2; i++ {
		active := blocked()
		if len(active) != 1 || active[0].Kind != "job_blocked" || active[0].ErrorCode != "transcription_version_mismatch" || active[0].JobID == nil || *active[0].JobID != jobID || active[0].Count != 1 {
			t.Fatalf("expected one job_blocked incident, got %+v", active)
		}
	}
	// A recent job is only waiting, not an incident.
	exec(`UPDATE jobs SET created_at=now() WHERE id=$1`, jobID)
	if active := blocked(); len(active) != 0 {
		t.Fatalf("a young pending job raised %+v", active)
	}
	exec(`UPDATE jobs SET created_at=now()-interval '10 minutes' WHERE id=$1`, jobID)
	if active := blocked(); len(active) != 1 {
		t.Fatalf("incident did not come back: %+v", active)
	}

	// The queue being paused is a decision, not an incident; it is still explained.
	exec(`UPDATE job_control SET paused=true`)
	if got := cause(); got != "queue_paused" {
		t.Fatalf("paused queue: %q", got)
	}
	if active := blocked(); len(active) != 0 {
		t.Fatalf("a paused queue raised %+v", active)
	}
	exec(`UPDATE job_control SET paused=false`)

	// Lifting the cause: incident resolves, no waiting reason left, and the job is claimed without replanning.
	h.workerCall(http.MethodPost, "/api/worker/transcription-config/status", map[string]any{"version": desired.Version, "active": true}, "", http.StatusOK)
	if got := cause(); got != "" {
		t.Fatalf("healthy wait must have no cause, got %q", got)
	}
	if active := blocked(); len(active) != 0 {
		t.Fatalf("incident not resolved: %+v", active)
	}
	var assignment struct {
		Job *struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodPost, "/api/worker/jobs/claim", nil, "", http.StatusOK), &assignment); err != nil {
		t.Fatal(err)
	}
	if assignment.Job == nil || assignment.Job.ID != jobID {
		t.Fatalf("job not leased after the fix: %+v", assignment)
	}
}
