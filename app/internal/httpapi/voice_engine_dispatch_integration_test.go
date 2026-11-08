//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	titanetModelID   = "dispatch-titanet"
	titanetModelRepo = "fixture/dispatch-titanet"
	titanetRevision  = "89abcdef0123456789abcdef0123456789abcdef"
)

// asWorker runs fn with the calls made as the Worker that owns the token.
func (h *engineOrderHarness) asWorker(token string, fn func()) {
	h.t.Helper()
	previous := h.token
	h.token = token
	defer func() { h.token = previous }()
	fn()
}

// installRuntime orders the release on the Worker and walks the order to ready, as that Worker would.
func (h *engineOrderHarness) installRuntime(workerID, token, digest, key string) {
	h.t.Helper()
	order := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": workerID, "idempotency_key": key})[0].Order
	h.asWorker(token, func() {
		h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+order.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
		h.report(order.OrderID, digest, "ready", "", http.StatusNoContent)
	})
}

// weights marks the model as ready on the Worker, at the revision it pinned, as a model order would.
func (h *engineOrderHarness) weights(workerID, revision string) {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), `INSERT INTO worker_voice_models(worker_id,model_id,state,revision) VALUES($1,$2,'ready',$3) ON CONFLICT (worker_id,model_id) DO UPDATE SET state='ready',revision=EXCLUDED.revision`, workerID, titanetModelID, revision); err != nil {
		h.t.Fatal(err)
	}
}

func (h *engineOrderHarness) claimAs(token string) (claimedJob, bool) {
	h.t.Helper()
	var job claimedJob
	var ok bool
	h.asWorker(token, func() {
		h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"transcription.audio.*", "voice.*"}}, "", http.StatusOK)
		var a struct {
			Job        *claimedJob `json:"job"`
			LeaseToken string      `json:"lease_token"`
		}
		if err := json.Unmarshal(h.workerCall(http.MethodPost, "/api/worker/jobs/claim", nil, "", http.StatusOK), &a); err != nil {
			h.t.Fatal(err)
		}
		if a.Job != nil {
			job, ok = *a.Job, true
			job.Lease = a.LeaseToken
		}
	})
	return job, ok
}

// The dispatch of a job needs the model, its revision and its approved runtime ready on the same Worker (#350):
// weights alone never open it, a lost runtime closes it without touching the jobs already leased, and the
// requested-model (benchmark), production and fingerprint-transition paths share the same rule.
func testVoiceEngineDispatch(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,'fingerprint','nemo-titanet',$2,$3)`, titanetModelID, titanetModelRepo, titanetRevision); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM voice_model_catalog WHERE id=$1`, titanetModelID) })
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent)
	h.putPipeline(map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5})
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 3}, http.StatusOK)
	digest := h.sign("1.0.0")
	fixture, fixtureToken := h.fixtureWorkerID(), h.token
	other, otherToken := h.secondID, h.secondToken

	// Benchmark: the job asks for the TitaNet model. Weights alone, on both Workers, open nothing.
	h.scheduleRequested(h.clipID, "voice.fingerprint.1", "dispatch-bench-1", titanetModelID, http.StatusCreated)
	h.weights(fixture, titanetRevision)
	h.weights(other, titanetRevision)
	if job, ok := h.claimAs(fixtureToken); ok {
		t.Fatalf("weights without the runtime opened %s", job.Step)
	}
	if job, ok := h.claimAs(otherToken); ok {
		t.Fatalf("weights without the runtime opened %s on the other Worker", job.Step)
	}
	// A runtime that is only requested or installing is not ready either.
	order := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": other, "idempotency_key": "dispatch-other-0001"})[0].Order
	h.asWorker(otherToken, func() {
		h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+order.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
		h.report(order.OrderID, digest, "installing", "", http.StatusNoContent)
	})
	if job, ok := h.claimAs(otherToken); ok {
		t.Fatalf("an installing runtime opened %s", job.Step)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders`); n != 1 {
		t.Fatalf("a claim ordered a runtime: %d orders", n)
	}

	// The runtime is ready on the fixture Worker only: that Worker takes the job, the other one waits.
	h.installRuntime(fixture, fixtureToken, digest, "dispatch-fixture-001")
	if job, ok := h.claimAs(otherToken); ok {
		t.Fatalf("the runtime of another Worker opened %s here", job.Step)
	}
	// A stale model (revision other than the catalogue's) is not usable even with the runtime ready.
	h.weights(fixture, strings.Repeat("1", 40))
	if job, ok := h.claimAs(fixtureToken); ok {
		t.Fatalf("a stale model revision opened %s", job.Step)
	}
	h.weights(fixture, titanetRevision)
	bench, ok := h.claimAs(fixtureToken)
	if !ok || bench.Step != "voice.fingerprint.1" {
		t.Fatalf("the Worker with model and runtime must claim the job, got %+v", bench)
	}

	// The runtime disappears while that job is leased: the job is not interrupted, new jobs are not attributed,
	// and nothing is ordered or downloaded.
	h.scheduleRequested(h.clipID, "voice.fingerprint.2", "dispatch-bench-2", titanetModelID, http.StatusCreated)
	h.asWorker(fixtureToken, func() {
		h.workerCall(http.MethodPost, "/api/worker/voice-engines/inventory", map[string]any{"runtimes": []map[string]string{}}, "", http.StatusNoContent)
	})
	if e, _ := h.entry(fixture, digest); e.State != "failed" || e.CauseCode != "runtime_missing" {
		t.Fatalf("lost runtime: %+v", e)
	}
	if h.count(`SELECT count(*) FROM jobs WHERE id=$1 AND status='leased'`, bench.ID) != 1 {
		t.Fatal("losing the runtime interrupted a leased job")
	}
	h.release(bench) // frees the capacity so the next claim is decided by the rule alone
	if job, ok := h.claimAs(fixtureToken); ok {
		t.Fatalf("a lost runtime (weights still present) opened %s", job.Step)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders`); n != 2 {
		t.Fatalf("losing the runtime ordered something: %d orders", n)
	}

	// The release is revoked after a successful install: the existing binding keeps serving.
	h.installRuntime(fixture, fixtureToken, digest, "dispatch-fixture-002")
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+digest, nil, http.StatusOK)
	if job, ok := h.claimAs(fixtureToken); !ok || job.Step != "voice.fingerprint.2" {
		t.Fatalf("a revoked release must keep serving its binding, got %+v %v", job, ok)
	}

	// Production and ADR 0044 transition: the active fingerprint model follows the same rule. The other Worker
	// has weights and no runtime; the fixture Worker has both.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE status='leased'`); err != nil {
		t.Fatal(err)
	}
	if n, err := workersHolding(ctx, pool, titanetModelID); err != nil || n != 1 {
		t.Fatalf("workers holding the model with its runtime: %d %v", n, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE worker_engine_inventory SET state='failed',cause_code='runtime_missing' WHERE worker_id=$1`, fixture); err != nil {
		t.Fatal(err)
	}
	if n, err := workersHolding(ctx, pool, titanetModelID); err != nil || n != 0 {
		t.Fatalf("a Worker without its runtime holds the model for the transition: %d %v", n, err)
	}
	if ready, err := voiceModelReadyIn(ctx, pool, titanetModelID); err != nil || ready {
		t.Fatalf("activation of a model without runtime must be refused: %v %v", ready, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE voice_active_model SET model_id=$1 WHERE kind='fingerprint'`, titanetModelID); err != nil {
		t.Fatal(err)
	}
	h.scheduleRequested(h.clipID, "voice.fingerprint.3", "dispatch-prod-1", "", http.StatusCreated)
	if job, ok := h.claimAs(fixtureToken); ok {
		t.Fatalf("an active model without runtime opened %s", job.Step)
	}
	if _, err := pool.Exec(ctx, `UPDATE worker_engine_inventory SET state='ready',cause_code='' WHERE worker_id=$1`, fixture); err != nil {
		t.Fatal(err)
	}
	if job, ok := h.claimAs(fixtureToken); !ok || job.Step != "voice.fingerprint.3" {
		t.Fatalf("the production job must reach the Worker with model and runtime, got %+v %v", job, ok)
	}
}
