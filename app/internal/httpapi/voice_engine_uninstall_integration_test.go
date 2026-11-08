//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const wespeakerID = "pyannote-wespeaker-resnet34"

type gotBlockers struct {
	Models []struct {
		ModelID      string `json:"model_id"`
		Busy         bool   `json:"busy"`
		InTransition bool   `json:"in_transition"`
	} `json:"models"`
	OpenOrder bool `json:"open_order"`
}

type gotUninstallEntry struct {
	engineEntry
	OrderIntent    string       `json:"order_intent"`
	OrderStatus    string       `json:"order_status"`
	OrderCauseCode string       `json:"order_cause_code"`
	Waiting        *gotBlockers `json:"waiting"`
}

func (h *engineOrderHarness) uninstall(digest, workerID, key string) gotEngineOrderResultWithBlockers {
	h.t.Helper()
	var out struct {
		Results []gotEngineOrderResultWithBlockers `json:"results"`
	}
	body := h.adminCall(http.MethodPost, "/api/admin/voice-engine-orders", map[string]any{"digest": digest, "intent": "uninstall", "orders": []map[string]string{{"worker_id": workerID, "idempotency_key": key}}}, http.StatusOK)
	if err := json.Unmarshal(body, &out); err != nil || len(out.Results) != 1 {
		h.t.Fatalf("uninstall answer: %v %s", err, body)
	}
	return out.Results[0]
}

type gotEngineOrderResultWithBlockers struct {
	gotEngineOrderResult
	Blockers *gotBlockers `json:"blockers"`
}

func (h *engineOrderHarness) preview(workerID, digest string) (removable bool, b gotBlockers) {
	h.t.Helper()
	var out struct {
		Removable bool        `json:"removable"`
		Blockers  gotBlockers `json:"blockers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines/uninstall-preview?worker_id="+workerID+"&digest="+digest, nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Removable, out.Blockers
}

func (h *engineOrderHarness) uninstallEntry(workerID, digest string) gotUninstallEntry {
	h.t.Helper()
	var out struct {
		Workers []struct {
			WorkerID string              `json:"worker_id"`
			Engines  []gotUninstallEntry `json:"engines"`
		} `json:"workers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines/workers", nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	for _, w := range out.Workers {
		if w.WorkerID == workerID {
			for _, e := range w.Engines {
				if e.Digest == digest {
					return e
				}
			}
		}
	}
	h.t.Fatalf("no entry for %s", digest)
	return gotUninstallEntry{}
}

// installReady runs an explicit install order to its end, as a Worker would.
func (h *engineOrderHarness) installReady(digest, workerID, key string) {
	h.t.Helper()
	r := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": workerID, "idempotency_key": key})
	if r[0].Status != "created" {
		h.t.Fatalf("install: %+v", r)
	}
	h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+r[0].Order.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
	h.report(r[0].Order.OrderID, digest, "ready", "", http.StatusNoContent)
}

// bindModel makes a model of the Worker depend on the release: its current installation order carried the
// manifest of the release. The model weights are synthetic rows, nothing is read or removed.
func (h *engineOrderHarness) bindModel(workerID, digest, modelID, orderID string) {
	h.t.Helper()
	if _, err := h.pool.Exec(h.t.Context(), `INSERT INTO worker_model_orders(id,worker_id,kind,action,model_id,revision,idempotency_key,author_name,status,runtime)
		SELECT $1,$2,c.kind,'install',c.id,c.revision,'bind-'||$1,'synthetic','done',g.manifest FROM voice_model_catalog c, voice_engine_registry g WHERE c.id=$3 AND g.digest=$4`, orderID, workerID, modelID, digest); err != nil {
		h.t.Fatal(err)
	}
	if _, err := h.pool.Exec(h.t.Context(), `INSERT INTO worker_voice_models(worker_id,model_id,state,order_id) VALUES($1,$2,'ready',$3)
		ON CONFLICT(worker_id,model_id) DO UPDATE SET state='ready',cause_code='',order_id=EXCLUDED.order_id`, workerID, modelID, orderID); err != nil {
		h.t.Fatal(err)
	}
}

func (h *engineOrderHarness) unbindModel(workerID, modelID string) {
	h.t.Helper()
	if _, err := h.pool.Exec(h.t.Context(), `UPDATE worker_voice_models SET state='failed',cause_code='runtime_incompatible' WHERE worker_id=$1 AND model_id=$2`, workerID, modelID); err != nil {
		h.t.Fatal(err)
	}
}

func testVoiceEngineUninstall(t *testing.T, pool *pgxpool.Pool) {
	base, fixture, job := startActiveCommunity(t, pool)
	h := newEngineOrderHarnessOn(t, base)
	digest, spare := h.sign("1.0.0"), h.sign("2.0.0")
	h.installReady(digest, fixture, "engine-un-install-01")
	h.installReady(spare, fixture, "engine-un-install-02")

	// Rights, and a release that is not installed: nothing is recorded.
	member := h.newUser("engine-uninstall-member")
	h.userCall(member, http.MethodPost, "/api/admin/voice-engine-orders", map[string]any{"digest": digest, "intent": "uninstall", "orders": []map[string]string{{"worker_id": fixture, "idempotency_key": "engine-un-denied-01"}}}, http.StatusForbidden)
	if r := h.uninstall(h.sign("3.0.0"), fixture, "engine-un-absent-01"); r.Status != "refused" || r.Cause != "engine_not_installed" {
		t.Fatalf("absent release: %+v", r)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE intent='uninstall'`); n != 0 {
		t.Fatalf("a refusal recorded %d removals", n)
	}

	// A version shared by two models, one of them busy with a leased job, one in an open transition: the
	// removal is refused with each reason, and nothing changes.
	h.bindModel(fixture, digest, communityID, "ord_bind_community_000001")
	h.bindModel(fixture, digest, wespeakerID, "ord_bind_wespeaker_0000001")
	if _, err := pool.Exec(t.Context(), `INSERT INTO voice_fingerprint_transitions(from_model_id,to_model_id,phase,announced_references,author_name) VALUES($1,'synthetic-target','preparing',0,'synthetic')`, wespeakerID); err != nil {
		t.Fatal(err)
	}
	r := h.uninstall(digest, fixture, "engine-un-shared-01")
	if r.Status != "refused" || r.Cause != "engine_in_use" || r.Blockers == nil || len(r.Blockers.Models) != 2 {
		t.Fatalf("shared release: %+v", r)
	}
	for _, m := range r.Blockers.Models {
		if (m.ModelID == communityID && !m.Busy) || (m.ModelID == wespeakerID && !m.InTransition) {
			t.Fatalf("blocker reasons: %+v", r.Blockers.Models)
		}
	}
	if ok, b := h.preview(fixture, digest); ok || len(b.Models) != 2 {
		t.Fatalf("preview: %v %+v", ok, b)
	}
	if e := h.uninstallEntry(fixture, digest); e.State != "ready" {
		t.Fatalf("a refusal changed the state: %s", e.State)
	}
	// The other release of the engine is not bound: it can be removed on its own while this one stays.
	if ok, _ := h.preview(fixture, spare); !ok {
		t.Fatal("an unbound release is removable")
	}

	// Released: the job ends, the transition is over, the models are no longer bound to the release.
	if _, err := pool.Exec(t.Context(), `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE voice_fingerprint_transitions SET phase='cancelled'`); err != nil {
		t.Fatal(err)
	}
	if r = h.uninstall(digest, fixture, "engine-un-shared-02"); r.Cause != "engine_in_use" || len(r.Blockers.Models) != 2 || r.Blockers.Models[0].Busy || r.Blockers.Models[1].InTransition {
		t.Fatalf("models stay in the way after the job and the transition ended: %+v", r.Blockers)
	}
	h.unbindModel(fixture, communityID)
	h.unbindModel(fixture, wespeakerID)
	modelsBefore := h.count(`SELECT count(*) FROM worker_voice_models`)

	// Accepted: recorded under the key with its author, the inventory shows the removal in progress.
	first := h.uninstall(digest, fixture, "engine-un-shared-03")
	if first.Status != "created" || first.Order.Intent != "uninstall" || first.Order.Generation != 3 {
		t.Fatalf("uninstall: %+v", first)
	}
	if e := h.uninstallEntry(fixture, digest); e.State != "removing" || e.AuthorName == "" || e.OrderIntent != "uninstall" || e.OrderStatus != "pending" {
		t.Fatalf("progress: %+v", e)
	}
	// Idempotent: the same key answers the same order, a new key or an install meanwhile is refused.
	if again := h.uninstall(digest, fixture, "engine-un-shared-03"); again.Status != "replayed" || again.Order.OrderID != first.Order.OrderID {
		t.Fatalf("replay: %+v", again)
	}
	if again := h.uninstall(digest, fixture, "engine-un-shared-04"); again.Cause != "engine_order_in_progress" {
		t.Fatalf("second removal: %+v", again)
	}
	if again := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "engine-un-race-0001"}); again[0].Cause != "engine_order_in_progress" {
		t.Fatalf("install during removal: %+v", again)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE intent='uninstall' AND status='pending'`); n != 1 {
		t.Fatalf("removals recorded: %d", n)
	}
	// A revocation forbids installing, it does not cancel the removal.
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+digest, nil, http.StatusOK)
	if e := h.uninstallEntry(fixture, digest); e.State != "removing" || e.OrderStatus != "pending" {
		t.Fatalf("revocation touched the removal: %+v", e)
	}

	// The drain: a dependency that appears before delivery holds the order back, without losing it. Once free
	// it is delivered, on every pull, with the frozen manifest.
	h.bindModel(fixture, digest, wespeakerID, "ord_bind_wespeaker_0000002")
	if got := h.delivered(); len(got) != 0 {
		t.Fatalf("a removal is delivered while a model depends on the release: %+v", got)
	}
	if e := h.uninstallEntry(fixture, digest); e.Waiting == nil || len(e.Waiting.Models) != 1 {
		t.Fatalf("what the removal waits for: %+v", e.Waiting)
	}
	h.unbindModel(fixture, wespeakerID)
	for i := 0; i < 2; i++ {
		got := h.delivered()
		if len(got) != 1 || got[0].OrderID != first.Order.OrderID || got[0].Intent != "uninstall" || string(got[0].Runtime) != string(first.Order.Runtime) {
			t.Fatalf("delivery %d: %+v", i, got)
		}
	}
	if e := h.uninstallEntry(fixture, digest); e.Waiting != nil {
		t.Fatalf("nothing left to wait for: %+v", e.Waiting)
	}

	// The Worker refuses because a model it holds is still bound: the runtime stays, the result is readable.
	h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+first.Order.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
	h.report(first.Order.OrderID, digest, "installing", "", http.StatusBadRequest)
	h.report(first.Order.OrderID, digest, "ready", "", http.StatusBadRequest)
	h.report(first.Order.OrderID, digest, "failed", "runtime_in_use", http.StatusNoContent)
	if e := h.uninstallEntry(fixture, digest); e.State != "ready" || e.OrderStatus != "failed" || e.OrderCauseCode != "runtime_in_use" || e.AuthorName == "" {
		t.Fatalf("refused removal: %+v", e)
	}
	h.report(first.Order.OrderID, digest, "removed", "", http.StatusNoContent) // late: the order has ended
	if e := h.uninstallEntry(fixture, digest); e.State != "ready" {
		t.Fatalf("a late report after the end moved the state: %s", e.State)
	}

	// Retry as a new generation, resumed from the data alone, until the Worker reports the removal.
	second := h.uninstall(digest, fixture, "engine-un-shared-05")
	if second.Status != "created" || second.Order.Generation != 4 {
		t.Fatalf("retry: %+v", second)
	}
	h.report(first.Order.OrderID, digest, "failed", "runtime_in_use", http.StatusNoContent) // older generation
	if e := h.uninstallEntry(fixture, digest); e.State != "removing" {
		t.Fatalf("a late report of an older generation applied: %+v", e)
	}
	h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+second.Order.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
	h.report(second.Order.OrderID, digest, "removed", "", http.StatusNoContent)
	h.report(second.Order.OrderID, digest, "removed", "", http.StatusNoContent) // repeated: no second effect
	e := h.uninstallEntry(fixture, digest)
	if e.State != "removed" || e.OrderStatus != "done" || e.AuthorName == "" {
		t.Fatalf("final result: %+v", e)
	}
	if got := h.delivered(); len(got) != 0 {
		t.Fatalf("a finished removal is delivered again: %+v", got)
	}
	// Only the designated release went: the other one is still ready, and no model row was touched.
	if o := h.uninstallEntry(fixture, spare); o.State != "ready" {
		t.Fatalf("the other release of the engine changed: %+v", o)
	}
	if n := h.count(`SELECT count(*) FROM worker_voice_models`); n != modelsBefore {
		t.Fatalf("model rows: %d -> %d", modelsBefore, n)
	}
	if r = h.uninstall(digest, fixture, "engine-un-shared-06"); r.Cause != "engine_not_installed" {
		t.Fatalf("removing twice: %+v", r)
	}
}
