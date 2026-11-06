//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"snitch/internal/auth"

	"github.com/jackc/pgx/v5/pgxpool"
)

type modelOrder struct {
	OrderID        string `json:"order_id"`
	WorkerID       string `json:"worker_id"`
	Kind           string `json:"kind"`
	Action         string `json:"action"`
	ModelID        string `json:"model_id"`
	Engine         string `json:"engine"`
	Repo           string `json:"repo"`
	Revision       string `json:"revision"`
	IdempotencyKey string `json:"idempotency_key"`
	HFToken        string `json:"hf_token"`
}

func (h *voiceHarness) workerOrders() []modelOrder {
	h.t.Helper()
	var out struct {
		WorkerID string       `json:"worker_id"`
		Orders   []modelOrder `json:"orders"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, "/api/worker/voice-models", nil, "", http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	if out.WorkerID != h.fixtureWorkerID() {
		h.t.Fatalf("the Worker is told its own identity, got %q", out.WorkerID)
	}
	return out.Orders
}

func (h *voiceHarness) order(workerID, modelID, key string, want int) modelOrder {
	h.t.Helper()
	var o modelOrder
	body := h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-model-orders", map[string]any{"model_id": modelID, "idempotency_key": key}, want)
	if want < 300 {
		if err := json.Unmarshal(body, &o); err != nil {
			h.t.Fatal(err)
		}
	}
	return o
}

func (h *voiceHarness) ack(orderID string, accepted bool, cause string, want int) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/voice-model-orders/"+orderID+"/ack", map[string]any{"accepted": accepted, "cause_code": cause}, "", want)
}

func (h *voiceHarness) reportOrder(orderID, modelID, state, cause string, received, total, occupied int64, want int) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": orderID, "model_id": modelID, "revision": secondRevision, "state": state, "cause_code": cause, "bytes_received": received, "bytes_total": total, "occupied_bytes": occupied}, "", want)
}

type viewCell struct {
	Kind            string `json:"kind"`
	ModelID         string `json:"model_id"`
	Repo            string `json:"repo"`
	Revision        string `json:"revision"`
	State           string `json:"state"`
	CauseCode       string `json:"cause_code"`
	BytesReceived   int64  `json:"bytes_received"`
	BytesTotal      int64  `json:"bytes_total"`
	OccupiedBytes   int64  `json:"occupied_bytes"`
	OrderID         string `json:"order_id"`
	Supported       bool   `json:"supported"`
	ReplacesModelID string `json:"replaces_model_id"`
	WaitingForJobs  bool   `json:"waiting_for_jobs"`
	Blocked         string `json:"blocked"`
}

func (h *voiceHarness) cells(workerID string) map[string]viewCell {
	h.t.Helper()
	var out struct {
		Workers []struct {
			WorkerID string     `json:"worker_id"`
			Models   []viewCell `json:"models"`
		} `json:"workers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-models/workers", nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	for _, w := range out.Workers {
		if w.WorkerID == workerID {
			cells := map[string]viewCell{}
			for _, c := range w.Models {
				cells[c.Kind] = c
			}
			return cells
		}
	}
	h.t.Fatalf("worker %s missing from the view", workerID)
	return nil
}

func (h *voiceHarness) fixtureWorkerID() string {
	h.t.Helper()
	var id string
	if err := h.pool.QueryRow(h.t.Context(), `SELECT id FROM workers WHERE name='voice-fixture'`).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

func (h *voiceHarness) otherWorker() string {
	h.t.Helper()
	const id = "wk_other_worker_identifier_00001"
	if _, err := h.pool.Exec(h.t.Context(), `INSERT INTO workers(id,name,token_hash,max_concurrency) VALUES($1,'other-worker',decode(repeat('ab',32),'hex'),1) ON CONFLICT DO NOTHING`, id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

// An Administrator orders a Worker to install a catalogue model; Snitch records the order and only that Worker
// receives it. Replaying the same key has no second effect.
func testVoiceModelOrderCreation(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t, pool)
	workerID := h.fixtureWorkerID()
	if orders := h.workerOrders(); len(orders) != 0 {
		t.Fatalf("no order yet: %+v", orders)
	}
	cells := h.cells(workerID)
	if len(cells) != 3 || cells["diarization"].State != "not_installed" || cells["fingerprint"].State != "not_installed" || cells["separation"].State != "not_installed" || cells["separation"].Supported {
		t.Fatalf("a Worker shows its three types: %+v", cells)
	}

	o := h.order(workerID, secondID, "install-b-0001", http.StatusCreated)
	if o.OrderID == "" || o.Action != "install" || o.WorkerID != workerID || o.Kind != "diarization" || o.ModelID != secondID || o.Repo != secondRepo || o.Revision != secondRevision || o.Engine != "pyannote" || o.IdempotencyKey != "install-b-0001" {
		t.Fatalf("unexpected order %+v", o)
	}
	if c := h.cells(workerID)["diarization"]; c.State != "requested" || c.ModelID != secondID || c.OrderID != o.OrderID {
		t.Fatalf("an order makes the model requested: %+v", c)
	}
	// Recorded with its author and time: the audit.
	var author string
	var created bool
	if err := pool.QueryRow(t.Context(), `SELECT author_name, created_at IS NOT NULL FROM worker_model_orders WHERE id=$1`, o.OrderID).Scan(&author, &created); err != nil || author != "administrator" || !created {
		t.Fatalf("order not audited: %q %v %v", author, created, err)
	}

	// Replaying the key returns the same order, creates nothing more.
	if again := h.order(workerID, secondID, "install-b-0001", http.StatusOK); again.OrderID != o.OrderID {
		t.Fatalf("replay created another order: %+v", again)
	}
	var count int
	_ = pool.QueryRow(t.Context(), `SELECT count(*) FROM worker_model_orders`).Scan(&count)
	if count != 1 {
		t.Fatalf("one order expected, got %d", count)
	}
	// The same key for something else is a conflict.
	h.order(workerID, "pyannote-wespeaker-resnet34", "install-b-0001", http.StatusConflict)

	if orders := h.workerOrders(); len(orders) != 1 || orders[0].OrderID != o.OrderID || orders[0].Repo != secondRepo {
		t.Fatalf("the Worker must receive its order: %+v", orders)
	}

	// An order of another Worker is not distributed to this one, and cannot be acknowledged by it.
	other := h.otherWorker()
	foreign := h.order(other, "pyannote-wespeaker-resnet34", "install-fp-0001", http.StatusCreated)
	if orders := h.workerOrders(); len(orders) != 1 || orders[0].OrderID != o.OrderID {
		t.Fatalf("the Worker received an order that is not its own: %+v", orders)
	}
	h.ack(foreign.OrderID, true, "", http.StatusNotFound)

	// What cannot be installed is refused: unknown, unsupported, unknown Worker, occupied type, bad key.
	h.order(workerID, "someone/else", "install-x-0001", http.StatusBadRequest)
	h.order("wk_unknown_worker_identifier_000", secondID, "install-x-0002", http.StatusNotFound)
	h.order(workerID, secondID, "x", http.StatusBadRequest)
	if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,supported,unsupported_reason) VALUES('test-separation','separation','','synthetic/separation','c00000000000000000000000000000000000000c',false,'no adapter')`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(t.Context(), `DELETE FROM voice_model_catalog WHERE id='test-separation'`) })
	h.order(workerID, "test-separation", "install-s-0001", http.StatusBadRequest)
	h.order(workerID, "nvidia-sortformer-4spk-v2.1", "install-n-0001", http.StatusConflict) // the diarization slot is taken (replacement is another ticket)

	// Only an Administrator may order.
	if _, err := pool.Exec(t.Context(), `INSERT INTO users(username,password_hash,role,approved) VALUES('member',$1,'user',true)`, auth.Password("synthetic-password")); err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	member := &voiceHarness{t: t, api: h.api, pool: pool, ts: h.ts, admin: &http.Client{Jar: jar}}
	member.adminCall(http.MethodPost, "/api/auth/login", map[string]string{"username": "member", "password": "synthetic-password"}, http.StatusOK)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(member.adminCall(http.MethodGet, "/api/auth/me", nil, http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	member.csrf = me.CSRF
	member.order(other, "pyannote-wespeaker-resnet34", "install-m-0001", http.StatusForbidden)
	anonymous := &voiceHarness{t: t, api: h.api, pool: pool, ts: h.ts, admin: &http.Client{}}
	anonymous.order(other, "pyannote-wespeaker-resnet34", "install-m-0002", http.StatusUnauthorized)
}

func (h *voiceHarness) retry(workerID, modelID, key string, want int) modelOrder {
	h.t.Helper()
	var o modelOrder
	body := h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-models/retry", map[string]any{"model_id": modelID, "idempotency_key": key}, want)
	if want < 300 {
		if err := json.Unmarshal(body, &o); err != nil {
			h.t.Fatal(err)
		}
	}
	return o
}

const fingerprintID = "pyannote-wespeaker-resnet34"

// The Worker acknowledges, downloads, verifies and reports; it is ready only on its own report, a failure
// carries an actionable cause and Retry orders again. The central token travels with every delivery of an unfinished gated download.
func testVoiceModelOrderExecution(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t, pool)
	workerID := h.fixtureWorkerID()
	const token = "hf_synthetic0123456789abcdef"
	h.adminCall(http.MethodPut, "/api/admin/hf-token", map[string]any{"token": token}, http.StatusNoContent)

	// A gated model downloads with the token, delivered with that order only.
	gated := h.order(workerID, communityID, "install-gated-01", http.StatusCreated)
	first := h.workerOrders()
	if len(first) != 1 || first[0].HFToken != token {
		t.Fatalf("a gated download needs the token: %+v", first)
	}
	h.ack(gated.OrderID, true, "", http.StatusNoContent)
	h.ack(gated.OrderID, true, "", http.StatusNoContent) // a replayed acknowledgement has no second effect
	// The Worker may have crashed before writing its secret file: until the order ends, every delivery carries it.
	for i := 0; i < 2; i++ {
		if again := h.workerOrders(); len(again) != 1 || again[0].HFToken != token {
			t.Fatalf("an unfinished gated order is redelivered with the token (delivery %d): %+v", i, again)
		}
	}
	// A report for an order that is not this Worker's, or for another model, is refused.
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": "ord_unknown_unknown_unknown", "model_id": communityID, "state": "downloading"}, "", http.StatusNotFound)
	h.reportOrder(gated.OrderID, secondID, "downloading", "", 1, 2, 0, http.StatusBadRequest)
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": gated.OrderID, "model_id": communityID, "revision": communityRevision, "state": "failed", "cause_code": "hf_token_missing"}, "", http.StatusNoContent)
	if c := h.cells(workerID)["diarization"]; c.State != "failed" || c.CauseCode != "hf_token_missing" || c.OrderID != gated.OrderID {
		t.Fatalf("a failure keeps its cause: %+v", c)
	}
	if orders := h.workerOrders(); len(orders) != 0 {
		t.Fatalf("a finished order is not distributed again: %+v", orders)
	}
	// A report for a finished order cannot reopen it.
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"order_id": gated.OrderID, "model_id": communityID, "revision": communityRevision, "state": "downloading"}, "", http.StatusConflict)

	// A model that needs no token never receives it, and the full life of an order.
	o := h.order(workerID, secondID, "install-b-0001", http.StatusCreated)
	if orders := h.workerOrders(); len(orders) != 1 || orders[0].HFToken != "" {
		t.Fatalf("the token only goes with a gated download: %+v", orders)
	}
	h.ack(o.OrderID, true, "", http.StatusNoContent)
	h.reportOrder(o.OrderID, secondID, "downloading", "", 10_000_000, 30_000_000, 10_000_000, http.StatusNoContent)
	if c := h.cells(workerID)["diarization"]; c.State != "downloading" || c.BytesReceived != 10_000_000 || c.BytesTotal != 30_000_000 || c.OccupiedBytes != 10_000_000 || c.Revision != secondRevision || c.Repo != secondRepo {
		t.Fatalf("progress, volume and revision are reported: %+v", c)
	}
	h.reportOrder(o.OrderID, secondID, "verifying", "", 30_000_000, 30_000_000, 30_000_000, http.StatusNoContent)
	if c := h.cells(workerID)["diarization"]; c.State != "verifying" {
		t.Fatalf("verifying expected: %+v", c)
	}
	h.reportOrder(o.OrderID, secondID, "ready", "", 30_000_000, 30_000_000, 30_000_000, http.StatusNoContent)
	if c := h.cells(workerID)["diarization"]; c.State != "ready" || c.OccupiedBytes != 30_000_000 {
		t.Fatalf("ready expected: %+v", c)
	}
	h.reportOrder(o.OrderID, secondID, "ready", "", 30_000_000, 30_000_000, 30_000_000, http.StatusNoContent) // replay
	h.reportOrder(o.OrderID, secondID, "downloading", "", 0, 30_000_000, 0, http.StatusConflict)
	var status string
	_ = pool.QueryRow(t.Context(), `SELECT status FROM worker_model_orders WHERE id=$1`, o.OrderID).Scan(&status)
	if status != "done" {
		t.Fatalf("the order is done once the model is ready: %s", status)
	}
	h.order(workerID, secondID, "install-b-0002", http.StatusConflict) // already installed

	// A Worker that refuses an order after its own checks makes it failed; Retry orders again.
	fp := h.order(workerID, fingerprintID, "install-fp-0001", http.StatusCreated)
	h.ack(fp.OrderID, false, "hf_network", http.StatusBadRequest) // only the cause of a refusal
	h.ack(fp.OrderID, false, "invalid_order", http.StatusNoContent)
	if c := h.cells(workerID)["fingerprint"]; c.State != "failed" || c.CauseCode != "invalid_order" {
		t.Fatalf("a refused order is a failure with its cause: %+v", c)
	}
	h.retry(workerID, secondID, "retry-key-0001", http.StatusConflict) // only a failed model can be retried
	retried := h.retry(workerID, fingerprintID, "retry-key-0002", http.StatusCreated)
	if retried.OrderID == fp.OrderID || retried.ModelID != fingerprintID {
		t.Fatalf("Retry is a new order: %+v", retried)
	}
	if c := h.cells(workerID)["fingerprint"]; c.State != "requested" || c.CauseCode != "" {
		t.Fatalf("Retry makes the model requested again: %+v", c)
	}
	if orders := h.workerOrders(); len(orders) != 1 || orders[0].OrderID != retried.OrderID {
		t.Fatalf("the Worker receives the new order: %+v", orders)
	}
	// Without a body, Retry keeps its former meaning (a failed candidate).
	h.adminCall(http.MethodPost, "/api/admin/workers/"+workerID+"/voice-models/retry", nil, http.StatusConflict)
}

// The Worker's pull names the kind of each model it lists and the models Snitch already knows as ready on it:
// a model held on its volume and absent from "held" (installed before orders existed) is reported by the Worker
// once proven, and is then shown in the view and eligible for dispatch (#327).
func testVoiceModelPullHeldAndKind(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	workerID := h.fixtureWorkerID()
	pull := func() (models []struct {
		ModelID string `json:"model_id"`
		Kind    string `json:"kind"`
	}, held []string) {
		var out struct {
			Models []struct {
				ModelID string `json:"model_id"`
				Kind    string `json:"kind"`
			} `json:"models"`
			Held []string `json:"held"`
		}
		if err := json.Unmarshal(h.workerCall(http.MethodGet, "/api/worker/voice-models", nil, "", http.StatusOK), &out); err != nil {
			t.Fatal(err)
		}
		return out.Models, out.Held
	}
	if _, err := pool.Exec(t.Context(), `DELETE FROM worker_voice_models WHERE worker_id=$1`, workerID); err != nil {
		t.Fatal(err)
	}
	models, held := pull()
	kinds := map[string]string{}
	for _, m := range models {
		kinds[m.ModelID] = m.Kind
	}
	if kinds["pyannote-wespeaker-resnet34"] != "fingerprint" || len(held) != 0 || held == nil {
		t.Fatalf("the active fingerprint is listed with its kind and nothing is held yet: %+v %+v", models, held)
	}
	if cell := h.cells(workerID)["fingerprint"]; cell.State != "not_installed" {
		t.Fatalf("before the report the view shows nothing: %+v", cell)
	}

	// The Worker proves the model it already holds and reports it without any order.
	h.workerCall(http.MethodPost, "/api/worker/voice-models/status", map[string]any{"model_id": "pyannote-wespeaker-resnet34", "revision": "837717ddb9ff5507820346191109dc79c958d614", "state": "ready", "cause_code": "", "bytes_received": 0, "bytes_total": 0, "occupied_bytes": 0}, "", http.StatusNoContent)
	if _, held = pull(); len(held) != 1 || held[0] != "pyannote-wespeaker-resnet34" {
		t.Fatalf("a reported ready model is held: %+v", held)
	}
	if cell := h.cells(workerID)["fingerprint"]; cell.State != "ready" || cell.ModelID != "pyannote-wespeaker-resnet34" || cell.Revision != "837717ddb9ff5507820346191109dc79c958d614" {
		t.Fatalf("the Workers view shows the fingerprint with model, version and state: %+v", cell)
	}
}

// An install order that nobody answers is never silent: after two minutes the view says why (#327).
func testVoiceModelOrderStalled(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t, pool)
	workerID := h.fixtureWorkerID()
	o := h.order(workerID, secondID, "install-stalled-0001", http.StatusCreated)
	if cell := h.cells(workerID)["diarization"]; cell.State != "requested" || cell.Blocked != "" {
		t.Fatalf("a fresh order is simply requested: %+v", cell)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE worker_model_orders SET created_at=now()-interval '3 minutes' WHERE id=$1`, o.OrderID); err != nil {
		t.Fatal(err)
	}
	if cell := h.cells(workerID)["diarization"]; cell.Blocked != "worker_offline" && cell.Blocked != "order_not_pulled" {
		t.Fatalf("an order unanswered for three minutes says why: %+v", cell)
	}
	h.ack(o.OrderID, true, "", http.StatusNoContent)
	h.reportOrder(o.OrderID, secondID, "failed", "disk_full", 0, 0, 0, http.StatusNoContent)
	if cell := h.cells(workerID)["diarization"]; cell.Blocked != "" {
		t.Fatalf("an answered order is not blocked: %+v", cell)
	}
}
