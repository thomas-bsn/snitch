//go:build integration

package httpapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type gotEngineOrderResult struct {
	WorkerID string          `json:"worker_id"`
	Status   string          `json:"status"`
	Cause    string          `json:"cause_code"`
	Order    *gotEngineOrder `json:"order"`
}

type gotEngineOrder struct {
	OrderID    string          `json:"order_id"`
	Digest     string          `json:"digest"`
	Engine     string          `json:"engine"`
	Version    string          `json:"version"`
	Intent     string          `json:"intent"`
	Generation int64           `json:"generation"`
	Runtime    json.RawMessage `json:"runtime"`
}

type engineEntry struct {
	Digest        string `json:"digest"`
	Version       string `json:"version"`
	State         string `json:"state"`
	CauseCode     string `json:"cause_code"`
	Generation    int64  `json:"generation"`
	BytesReceived int64  `json:"bytes_received"`
	AuthorName    string `json:"author_name"`
	History       []struct {
		State   string `json:"state"`
		Ignored bool   `json:"ignored"`
		Reason  string `json:"reason"`
	} `json:"history"`
}

type engineOrderHarness struct {
	*voiceHarness
	secondID, secondToken string
	sign                  func(version string) string
}

func newEngineOrderHarness(t *testing.T, pool *pgxpool.Pool) *engineOrderHarness {
	return newEngineOrderHarnessOn(t, newVoiceHarness(t, pool))
}

func newEngineOrderHarnessOn(t *testing.T, h *voiceHarness) *engineOrderHarness {
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.StdEncoding.EncodeToString
	keys, _ := json.Marshal(map[string]string{"publisher": encode(pub)})
	t.Setenv(engineTrustEnv, string(keys))
	e := &engineOrderHarness{voiceHarness: h}
	var created struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/workers", map[string]any{"name": "engine-second", "max_concurrency": 1}, http.StatusCreated), &created); err != nil || created.ID == "" {
		t.Fatalf("second worker: %v", err)
	}
	e.secondID, e.secondToken = created.ID, created.Token
	e.sign = func(version string) string {
		m := testEngineManifest()
		m.Version = version
		raw, _ := json.Marshal(m)
		var release engineRelease
		input := map[string]string{"key_id": "publisher", "payload": encode(raw), "signature": encode(ed25519.Sign(private, raw))}
		if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/voice-engines", input, http.StatusCreated), &release); err != nil {
			t.Fatal(err)
		}
		return release.Digest
	}
	return e
}

func (h *engineOrderHarness) orderOn(digest string, want int, orders ...map[string]string) []gotEngineOrderResult {
	h.t.Helper()
	var out struct {
		Results []gotEngineOrderResult `json:"results"`
	}
	body := h.adminCall(http.MethodPost, "/api/admin/voice-engine-orders", map[string]any{"digest": digest, "intent": "install", "orders": orders}, want)
	if want == http.StatusOK {
		if err := json.Unmarshal(body, &out); err != nil {
			h.t.Fatal(err)
		}
	}
	return out.Results
}

func (h *engineOrderHarness) delivered() []gotEngineOrder {
	h.t.Helper()
	var out struct {
		EngineOrders []gotEngineOrder `json:"engine_orders"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, "/api/worker/voice-models", nil, "", http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.EngineOrders
}

func (h *engineOrderHarness) report(order, digest, state, cause string, want int) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/status", map[string]any{"order_id": order, "digest": digest, "state": state, "cause_code": cause, "bytes_received": 10, "bytes_total": 100, "local_identity": strings.Repeat("c", 64)}, "", want)
}

func (h *engineOrderHarness) entry(workerID, digest string) (engineEntry, bool) {
	h.t.Helper()
	var out struct {
		Workers []struct {
			WorkerID string        `json:"worker_id"`
			Engines  []engineEntry `json:"engines"`
		} `json:"workers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines/workers", nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	for _, w := range out.Workers {
		if w.WorkerID == workerID {
			for _, e := range w.Engines {
				if e.Digest == digest {
					return e, true
				}
			}
		}
	}
	return engineEntry{}, false
}

func (h *engineOrderHarness) count(query string, args ...any) int {
	h.t.Helper()
	var n int
	if err := h.pool.QueryRow(h.t.Context(), query, args...).Scan(&n); err != nil {
		h.t.Fatal(err)
	}
	return n
}

func testVoiceEngineOrders(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	digest := h.sign("1.0.0")
	fixture, other := h.fixtureWorkerID(), h.secondID

	// Rights: an Administrator orders, a member does not; nothing is recorded for a refusal.
	member := h.newUser("engine-order-member")
	h.userCall(member, http.MethodPost, "/api/admin/voice-engine-orders", map[string]any{"digest": digest, "intent": "install", "orders": []map[string]string{{"worker_id": fixture, "idempotency_key": "engine-denied-0001"}}}, http.StatusForbidden)
	if n := h.count(`SELECT count(*) FROM worker_engine_orders`); n != 0 {
		t.Fatalf("a refused request recorded %d orders", n)
	}

	// One order per chosen Worker, each frozen with the version and the manifest, nothing downloaded yet.
	results := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "engine-fixture-0001"}, map[string]string{"worker_id": other, "idempotency_key": "engine-other-00001"})
	if len(results) != 2 || results[0].Status != "created" || results[1].Status != "created" || results[0].Order.Generation != 1 || results[0].Order.Version != "1.0.0" || results[0].Order.Intent != "install" {
		t.Fatalf("orders: %+v", results)
	}
	first := results[0].Order
	if e, ok := h.entry(fixture, digest); !ok || e.State != "requested" || e.Generation != 1 || e.AuthorName == "" {
		t.Fatalf("inventory after order: %+v %v", e, ok)
	}

	// Replay: same key, same content, no second order. Same key for another Worker is refused.
	if r := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "engine-fixture-0001"}); r[0].Status != "replayed" || r[0].Order.OrderID != first.OrderID {
		t.Fatalf("replay: %+v", r)
	}
	if r := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": other, "idempotency_key": "engine-fixture-0001"}); r[0].Status != "refused" || r[0].Cause != "idempotency_key_reused" {
		t.Fatalf("key reuse: %+v", r)
	}
	// A second order while one is open is refused instead of running twice.
	if r := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "engine-fixture-0002"}); r[0].Status != "refused" || r[0].Cause != "engine_order_in_progress" {
		t.Fatalf("concurrent: %+v", r)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders`); n != 2 {
		t.Fatalf("orders recorded: %d", n)
	}
	if r := h.orderOn(strings.Repeat("e", 64), http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "engine-unknown-001"}); r[0].Cause != "engine_not_found" {
		t.Fatalf("unknown release: %+v", r)
	}

	// Delivery: only this Worker's order, with the frozen manifest. Pulling again (a reconnection) changes nothing.
	for i := 0; i < 2; i++ {
		got := h.delivered()
		if len(got) != 1 || got[0].OrderID != first.OrderID || got[0].Digest != digest || got[0].Generation != 1 || string(got[0].Runtime) != string(first.Runtime) {
			t.Fatalf("delivery %d: %+v", i, got)
		}
	}
	if e, _ := h.entry(fixture, digest); e.State != "requested" {
		t.Fatalf("a pull changed the state: %s", e.State)
	}

	// Progress through the real states; a report older than the current step is ignored and journalled.
	h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+first.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
	h.report(first.OrderID, digest, "downloading_runtime", "", http.StatusNoContent)
	h.report(first.OrderID, digest, "installing", "", http.StatusNoContent)
	h.report(first.OrderID, digest, "downloading_runtime", "", http.StatusNoContent)
	if e, _ := h.entry(fixture, digest); e.State != "installing" {
		t.Fatalf("late step moved the state: %s", e.State)
	}
	h.report(first.OrderID, digest, "verifying", "", http.StatusNoContent)
	h.report(first.OrderID, digest, "failed", "not_a_cause", http.StatusBadRequest)
	h.report(first.OrderID, digest, "ready", "", http.StatusNoContent)
	e, _ := h.entry(fixture, digest)
	ignored := 0
	for _, ev := range e.History {
		if ev.Ignored {
			ignored++
		}
	}
	if e.State != "ready" || len(e.History) < 6 || ignored != 1 {
		t.Fatalf("history: %+v", e)
	}
	if got := h.delivered(); len(got) != 0 {
		t.Fatalf("a finished order is delivered again: %+v", got)
	}
	// A late report after the end is ignored, never a regression.
	h.report(first.OrderID, digest, "downloading_runtime", "", http.StatusNoContent)
	if e, _ := h.entry(fixture, digest); e.State != "ready" {
		t.Fatalf("late report regressed a ready engine: %s", e.State)
	}
	if r := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "engine-fixture-0003"}); r[0].Cause != "engine_already_installed" {
		t.Fatalf("reinstall: %+v", r)
	}

	// Workers diverge: the other one fails without touching the first; a retry is a new generation, and the
	// failed generation's late report is ignored.
	second := results[1].Order
	h.otherWorkerReport(second.OrderID, digest, "failed", "disk_full")
	if e, _ := h.entry(other, digest); e.State != "failed" || e.CauseCode != "disk_full" {
		t.Fatalf("other worker: %+v", e)
	}
	if e, _ := h.entry(fixture, digest); e.State != "ready" {
		t.Fatalf("a failure on one Worker changed another: %s", e.State)
	}
	retry := h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": other, "idempotency_key": "engine-other-00002"})
	if retry[0].Status != "created" || retry[0].Order.Generation != 2 {
		t.Fatalf("retry: %+v", retry)
	}
	h.otherWorkerReport(second.OrderID, digest, "ready", "")
	if e, _ := h.entry(other, digest); e.State != "requested" || e.Generation != 2 {
		t.Fatalf("late report of an older generation applied: %+v", e)
	}

	// A revoked release cannot be ordered; one that is queued is cancelled.
	digest2 := h.sign("2.0.0")
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+digest2, nil, http.StatusOK)
	if r := h.orderOn(digest2, http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "engine-revoked-001"}); r[0].Cause != "engine_release_unavailable" {
		t.Fatalf("revoked: %+v", r)
	}

	// Reconciliation only observes: a Worker that lost its runtime loses the ready state, nothing is ordered.
	before := h.count(`SELECT count(*) FROM worker_engine_orders`)
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/inventory", map[string]any{"runtimes": []map[string]string{{"engine": "nemo-titanet", "local_identity": strings.Repeat("c", 64)}}}, "", http.StatusNoContent)
	if e, _ := h.entry(fixture, digest); e.State != "ready" {
		t.Fatalf("a present runtime was dropped: %s", e.State)
	}
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/inventory", map[string]any{"runtimes": []map[string]string{}}, "", http.StatusNoContent)
	if e, _ := h.entry(fixture, digest); e.State != "failed" || e.CauseCode != "runtime_missing" {
		t.Fatalf("lost runtime: %+v", e)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders`); n != before {
		t.Fatalf("reconciliation created orders: %d -> %d", before, n)
	}
	if got := h.delivered(); len(got) != 0 {
		t.Fatalf("reconciliation caused a delivery: %+v", got)
	}

	// Revoking a release cancels the engine orders still queued for it; the late report is ignored.
	digest3 := h.sign("3.0.0")
	queued := h.orderOn(digest3, http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "engine-queued-0001"})[0].Order
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+digest3, nil, http.StatusOK)
	if got := h.delivered(); len(got) != 0 {
		t.Fatalf("a revoked queued order is delivered: %+v", got)
	}
	h.report(queued.OrderID, digest3, "ready", "", http.StatusNoContent)
	if e, _ := h.entry(fixture, digest3); e.State == "ready" {
		t.Fatal("a cancelled order became ready")
	}
}

// otherWorkerReport reports as the second Worker, with its own token.
func (h *engineOrderHarness) otherWorkerReport(orderID, digest, state, cause string) {
	h.t.Helper()
	token := h.token
	h.token = h.secondToken
	defer func() { h.token = token }()
	h.report(orderID, digest, state, cause, http.StatusNoContent)
}
