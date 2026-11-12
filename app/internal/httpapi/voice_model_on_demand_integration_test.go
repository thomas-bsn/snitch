//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"snitch/internal/jobs"

	"github.com/jackc/pgx/v5/pgxpool"
)

type gotEnginePreview struct {
	ModelID          string `json:"model_id"`
	EngineRequired   bool   `json:"engine_required"`
	Family           string `json:"family"`
	Engine           string `json:"engine"`
	Version          string `json:"version"`
	Kind             string `json:"kind"`
	State            string `json:"state"`
	DiskBytes        int64  `json:"disk_bytes"`
	EstimatedSeconds int64  `json:"estimated_seconds"`
	Blocker          string `json:"blocker"`
	Active           bool   `json:"active"`
}

func (h *engineOrderHarness) previewEngine(worker, model string, want int) gotEnginePreview {
	h.t.Helper()
	var out gotEnginePreview
	raw := h.adminCall(http.MethodGet, "/api/admin/workers/"+worker+"/voice-model-orders/preview?model_id="+model, nil, want)
	if want == http.StatusOK {
		if err := json.Unmarshal(raw, &out); err != nil {
			h.t.Fatal(err)
		}
	}
	return out
}

// runtimeReady evaluates the dispatch clause of a ready model on a Worker, as the job claim does.
func (h *engineOrderHarness) runtimeReady(worker, model string) bool {
	h.t.Helper()
	return h.count(`SELECT count(*) FROM worker_voice_models wvm JOIN voice_model_catalog m ON m.id=wvm.model_id WHERE wvm.worker_id=$1 AND wvm.model_id=$2 AND wvm.state='ready' AND `+jobs.RuntimeReadyClause("wvm", "m"), worker, model) == 1
}

func (h *engineOrderHarness) modelOrderOf(worker, model string) string {
	h.t.Helper()
	var id string
	if err := h.pool.QueryRow(h.t.Context(), `SELECT id FROM worker_model_orders WHERE worker_id=$1 AND model_id=$2 ORDER BY created_at DESC LIMIT 1`, worker, model).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

// testVoiceModelOnDemandEngine covers #372: choosing a model whose family engine is missing orders the engine,
// then the model, in one request.
func testVoiceModelOnDemandEngine(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	const rev = "0123456789abcdef0123456789abcdef01234567"
	for _, m := range [][]string{
		{"synthetic-nemo-diar", "diarization", "nemo", "synthetic/nemo-diar", "nemo"},
		{"synthetic-nemo-fp", "fingerprint", "nemo", "synthetic/nemo-fp", "nemo"},
		{"synthetic-speechbrain-fp", "fingerprint", "speechbrain-ecapa", "synthetic/sb-fp", "mystery-lib"},
		{"synthetic-embedded-sortformer", "diarization", "nemo", "synthetic/sortformer", ""},
	} {
		if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,family) VALUES($1,$2,$3,$4,$5,$6)`, m[0], m[1], m[2], m[3], rev, m[4]); err != nil {
			t.Fatal(err)
		}
	}
	fixture, other := h.fixtureWorkerID(), h.secondID
	order := func(worker, model, key string, want int) []byte {
		return h.adminCall(http.MethodPost, "/api/admin/workers/"+worker+"/voice-model-orders", map[string]any{"model_id": model, "idempotency_key": key}, want)
	}
	pulledModels := func() int { o, _ := h.pulled(); return len(o) }

	// Preview: the engine, its version, the disk and the time, before anything exists.
	p := h.previewEngine(fixture, "synthetic-nemo-diar", http.StatusOK)
	f := engineFamilies["nemo"]
	if !p.EngineRequired || p.Family != "nemo" || p.Engine != f.Engine || p.Version != f.DefaultVer || p.Kind != "diarization" || p.State != "missing" || p.DiskBytes != f.DiskBytes || p.EstimatedSeconds != f.EstimatedSeconds || p.Blocker != "" {
		t.Fatalf("preview: %+v", p)
	}
	if h.count(`SELECT count(*) FROM worker_engine_orders`)+h.count(`SELECT count(*) FROM worker_model_orders`) != 0 {
		t.Fatal("a preview ordered something")
	}

	// One confirmation, two linked orders: the engine of the model's kind, then the model.
	var modelOrder gotRuntimeOrder
	if err := json.Unmarshal(order(fixture, "synthetic-nemo-diar", "ondemand-model-0001", http.StatusCreated), &modelOrder); err != nil {
		t.Fatal(err)
	}
	engines := h.delivered()
	if len(engines) != 1 || engines[0].Engine != f.Engine || engines[0].Intent != "install" {
		t.Fatalf("engine order: %+v", engines)
	}
	var runtime map[string]any
	if err := json.Unmarshal(engines[0].Runtime, &runtime); err != nil || runtime["source"] != "on_demand" || runtime["kind"] != "diarization" || runtime["family"] != "nemo" {
		t.Fatalf("engine runtime: %s %v", engines[0].Runtime, err)
	}
	if h.count(`SELECT count(*) FROM voice_engine_registry WHERE digest=$1 AND kind='diarization' AND source='on_demand'`, engines[0].Digest) != 1 {
		t.Fatal("the registry row must carry the kind of the model")
	}
	if h.count(`SELECT count(*) FROM worker_model_engine_dependencies d JOIN worker_engine_orders e ON e.id=d.engine_order_id WHERE d.model_order_id=$1 AND e.digest=$2`, modelOrder.OrderID, engines[0].Digest) != 1 {
		t.Fatal("the model order must depend on the engine order")
	}
	if p := h.previewEngine(fixture, "synthetic-nemo-diar", http.StatusOK); p.State != "in_progress" {
		t.Fatalf("preview with an engine in progress: %+v", p)
	}

	// The model waits for its engine and is never ready by itself.
	if n := pulledModels(); n != 0 {
		t.Fatalf("model delivered before its engine: %d", n)
	}
	if state, _ := h.modelState(fixture, "synthetic-nemo-diar"); state != "requested" {
		t.Fatalf("model state %q", state)
	}
	if h.runtimeReady(fixture, "synthetic-nemo-diar") {
		t.Fatal("a model is not eligible without a ready engine")
	}
	// Replay creates nothing more.
	order(fixture, "synthetic-nemo-diar", "ondemand-model-0001", http.StatusOK)
	if h.count(`SELECT count(*) FROM worker_engine_orders`) != 1 {
		t.Fatal("replay duplicated the engine order")
	}
	h.ack(engines[0].OrderID)
	h.report(engines[0].OrderID, engines[0].Digest, "ready", "", http.StatusNoContent)
	if n := pulledModels(); n != 1 {
		t.Fatalf("model not delivered once the engine is ready: %d", n)
	}
	h.modelAck(modelOrder.OrderID, true, "", http.StatusNoContent)
	h.modelReport(modelOrder.OrderID, "synthetic-nemo-diar", "ready", "", http.StatusNoContent)
	if !h.runtimeReady(fixture, "synthetic-nemo-diar") {
		t.Fatal("a model bound to its ready on-demand engine must be eligible")
	}
	if p := h.previewEngine(fixture, "synthetic-nemo-diar", http.StatusOK); p.State != "ready" {
		t.Fatalf("preview with a ready engine: %+v", p)
	}

	// A model of the other kind of the same family needs the engine of its own kind: ordered, not reused.
	order(fixture, "synthetic-nemo-fp", "ondemand-model-0002", http.StatusCreated)
	var kinds []string
	for _, e := range h.delivered() {
		var r map[string]any
		_ = json.Unmarshal(e.Runtime, &r)
		kinds = append(kinds, r["kind"].(string))
	}
	if len(kinds) != 1 || kinds[0] != "fingerprint" {
		t.Fatalf("fingerprint engine order: %v", kinds)
	}
	h.modelReport(h.modelOrderOf(fixture, "synthetic-nemo-fp"), "synthetic-nemo-fp", "ready", "", http.StatusNoContent)
	if h.runtimeReady(fixture, "synthetic-nemo-fp") {
		t.Fatal("the diarization engine must not make a fingerprint model eligible")
	}

	// A pending engine of the other Worker is reused by the model, never ordered twice.
	r := h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": other, "idempotency_key": "ondemand-other-0001"})
	if r[0].Status != "created" {
		t.Fatalf("independent engine order: %+v", r)
	}
	order(other, "synthetic-nemo-diar", "ondemand-model-0003", http.StatusCreated)
	if h.count(`SELECT count(*) FROM worker_engine_orders WHERE worker_id=$1`, other) != 1 {
		t.Fatal("an engine in progress was ordered again")
	}

	// Unsupported family: a distinct refusal, and not a single row.
	count := func() [3]int {
		return [3]int{h.count(`SELECT count(*) FROM worker_engine_orders`), h.count(`SELECT count(*) FROM worker_model_orders`), h.count(`SELECT count(*) FROM voice_engine_registry`)}
	}
	before := count()
	if !json.Valid(order(other, "synthetic-speechbrain-fp", "ondemand-model-0004", http.StatusBadRequest)) {
		t.Fatal("refusal must be an error document")
	}
	h.previewEngine(other, "synthetic-speechbrain-fp", http.StatusBadRequest)
	if after := count(); before != after || h.count(`SELECT count(*) FROM worker_voice_models WHERE model_id='synthetic-speechbrain-fp'`) != 0 {
		t.Fatalf("an unsupported family left rows: %v -> %v", before, after)
	}

	// Embedded engines stay untouched: no engine, no runtime, no engine in the preview.
	engineOrders := h.count(`SELECT count(*) FROM worker_engine_orders`)
	if p := h.previewEngine(fixture, "synthetic-embedded-sortformer", http.StatusOK); p.EngineRequired {
		t.Fatalf("embedded preview: %+v", p)
	}
	h.adminCall(http.MethodPost, "/api/admin/workers/"+fixture+"/voice-model-orders", map[string]any{"model_id": "synthetic-embedded-sortformer", "idempotency_key": "ondemand-embedded-01", "replaces_model_id": "synthetic-nemo-diar"}, http.StatusCreated)
	if h.count(`SELECT count(*) FROM worker_engine_orders`) != engineOrders || h.count(`SELECT count(*) FROM worker_model_orders WHERE model_id='synthetic-embedded-sortformer' AND runtime IS NULL`) != 1 {
		t.Fatal("an embedded model must order no engine and carry no runtime")
	}
	if _, err := pool.Exec(t.Context(), `UPDATE worker_voice_models SET state='ready' WHERE worker_id=$1 AND model_id='synthetic-embedded-sortformer'`, fixture); err != nil {
		t.Fatal(err)
	}
	if !h.runtimeReady(fixture, "synthetic-embedded-sortformer") {
		t.Fatal("embedded models keep bypassing the engine clause")
	}
}

// testVoiceModelOnDemandFamilies covers the SpeechBrain and WeSpeaker families (#374, #375): each model orders
// the engine of its family and kind. The separation models of the SpeechBrain family have their own test
// (testVoiceSeparationModel).
func testVoiceModelOnDemandFamilies(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	const rev = "0123456789abcdef0123456789abcdef01234567"
	for _, m := range [][]string{
		{"synthetic-sb-fp", "fingerprint", "speechbrain-ecapa", "synthetic/fam-sb-fp", "speechbrain"},
		{"synthetic-ws-fp", "fingerprint", "wespeaker", "synthetic/fam-ws-fp", "wespeaker"},
	} {
		if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,family) VALUES($1,$2,$3,$4,$5,$6)`, m[0], m[1], m[2], m[3], rev, m[4]); err != nil {
			t.Fatal(err)
		}
	}
	fixture, other := h.fixtureWorkerID(), h.secondID
	order := func(worker, model, key string) {
		h.adminCall(http.MethodPost, "/api/admin/workers/"+worker+"/voice-model-orders", map[string]any{"model_id": model, "idempotency_key": key}, http.StatusCreated)
	}
	runtimeOf := func(digest string) map[string]any {
		var raw []byte
		if err := pool.QueryRow(t.Context(), `SELECT manifest FROM voice_engine_registry WHERE digest=$1`, digest).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}

	for _, c := range []struct {
		worker, model, family, engine, kind string
		active                              bool
	}{
		{fixture, "synthetic-ws-fp", "wespeaker", "wespeaker", "fingerprint", true},
		{other, "synthetic-sb-fp", "speechbrain", "speechbrain", "fingerprint", true},
	} {
		p := h.previewEngine(c.worker, c.model, http.StatusOK)
		f := engineFamilies[c.family]
		if !p.EngineRequired || p.Family != c.family || p.Engine != c.engine || p.Kind != c.kind || p.Version != f.DefaultVer || p.State != "missing" || p.Active != c.active || p.DiskBytes != f.DiskBytes {
			t.Fatalf("%s preview: %+v", c.model, p)
		}
		order(c.worker, c.model, "ondemand-fam-"+c.model)
	}
	// Each Worker received the engine of its family, with the kind of its model, before any model.
	check := func(delivered []gotEngineOrder, want map[string]string) {
		t.Helper()
		got := map[string]string{}
		for _, e := range delivered {
			m := runtimeOf(e.Digest)
			got[m["family"].(string)+"/"+m["kind"].(string)] = e.Engine
		}
		if len(got) != len(want) {
			t.Fatalf("engine orders: %v, want %v", got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Fatalf("engine orders: %v, want %v", got, want)
			}
		}
	}
	check(h.delivered(), map[string]string{"wespeaker/fingerprint": "wespeaker"})
	h.asSecond(func() {
		check(h.delivered(), map[string]string{"speechbrain/fingerprint": "speechbrain"})
	})

	// Ready engines make the fingerprint models eligible.
	ready := func(worker, model string) {
		t.Helper()
		var digest, order string
		if err := pool.QueryRow(t.Context(), `SELECT e.digest,e.id FROM worker_model_engine_dependencies d JOIN worker_model_orders m ON m.id=d.model_order_id JOIN worker_engine_orders e ON e.id=d.engine_order_id WHERE m.worker_id=$1 AND m.model_id=$2`, worker, model).Scan(&digest, &order); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(t.Context(), `UPDATE worker_engine_inventory SET state='ready' WHERE worker_id=$1 AND digest=$2`, worker, digest); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(t.Context(), `UPDATE worker_voice_models SET state='ready' WHERE worker_id=$1 AND model_id=$2`, worker, model); err != nil {
			t.Fatal(err)
		}
	}
	ready(fixture, "synthetic-ws-fp")
	ready(other, "synthetic-sb-fp")
	if !h.runtimeReady(fixture, "synthetic-ws-fp") || !h.runtimeReady(other, "synthetic-sb-fp") {
		t.Fatal("a fingerprint model bound to its ready family engine must be eligible")
	}
	// The family also installs for separation, as its own engine identity.
	var sep struct {
		Results []gotEngineOrderResult `json:"results"`
	}
	body := map[string]any{"family": "speechbrain", "kind": "separation", "orders": []map[string]string{{"worker_id": fixture, "idempotency_key": "ondemand-sb-sep-0001"}}}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/voice-engines/on-demand-orders", body, http.StatusOK), &sep); err != nil || len(sep.Results) != 1 || sep.Results[0].Status != "created" {
		t.Fatalf("separation engine order: %+v %v", sep, err)
	}
	// The WeSpeaker engine does not serve a SpeechBrain model and the reverse.
	if _, err := pool.Exec(t.Context(), `UPDATE worker_voice_models SET model_id='synthetic-sb-fp' WHERE worker_id=$1 AND model_id='synthetic-ws-fp'`, fixture); err != nil {
		t.Fatal(err)
	}
	if h.runtimeReady(fixture, "synthetic-sb-fp") {
		t.Fatal("an engine of another family must not make the model eligible")
	}
}
