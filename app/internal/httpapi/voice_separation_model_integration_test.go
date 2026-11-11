//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testVoiceSeparationModel covers #401 (ADR 0065): a separation model of the catalogue is installed on a Worker by
// an explicit order only, after the resource checks of ADR 0055, with the engine of its family; it is ready only
// when the Worker reports a successful test inference, a failure shows a stable cause, and a model that is not
// there is an explicit state. Being ready opens no production job: the analysis of a clip never separates.
// The Worker is played through its real API; the model is one of the two catalogue candidates, never downloaded.
func testVoiceSeparationModel(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	const model = "speechbrain-sepformer-whamr16k"
	worker := h.fixtureWorkerID()
	family := engineFamilies["speechbrain"]
	rows := func() [4]int {
		return [4]int{h.count(`SELECT count(*) FROM worker_engine_orders`), h.count(`SELECT count(*) FROM worker_model_orders`),
			h.count(`SELECT count(*) FROM worker_voice_models WHERE model_id=$1`, model), h.count(`SELECT count(*) FROM voice_engine_registry`)}
	}
	// stateOn is what the catalogue says of the model on a Worker: its state and its cause.
	stateOn := func(m catalogModel, id string) (string, string) {
		for _, w := range m.Workers {
			if w.WorkerID == id {
				return w.State, w.CauseCode
			}
		}
		return "not_installed", ""
	}
	separationJobs := func() int {
		return h.count(`SELECT count(*) FROM jobs WHERE step LIKE '%separation%'`)
	}

	// Without an order: an explicit "not installed", and nothing reaches the Worker, whatever it pulls.
	if c := h.cells(worker)["separation"]; c.State != "not_installed" || !c.Supported || c.ModelID != "" {
		t.Fatalf("a Worker without a separation model: %+v", c)
	}
	if m := h.voiceModels().model(t, model); m.Installed {
		t.Fatalf("the catalogue reference is installed nowhere: %+v", m)
	} else if state, _ := stateOn(m, worker); state != "not_installed" {
		t.Fatalf("the catalogue says the model is absent from the Worker: %+v", m)
	}
	if orders := h.workerOrders(); len(orders) != 0 || len(h.delivered()) != 0 || rows() != [4]int{} {
		t.Fatalf("something was ordered implicitly: %v %v", orders, rows())
	}

	// Resource checks before installation (ADR 0055): the refusal names its cause and writes nothing.
	for cause, hw := range map[string]workerHardware{
		"platform_incompatible": {Python: "3.12", Platform: "darwin-arm64", RAMBytes: 64 << 30, DiskFreeBytes: 200 << 30},
		"insufficient_ram":      {Python: "3.12", Platform: "linux-x86_64", RAMBytes: 1 << 30, DiskFreeBytes: 200 << 30},
		"insufficient_disk":     {Python: "3.12", Platform: "linux-x86_64", RAMBytes: 64 << 30, DiskFreeBytes: 1 << 20},
	} {
		h.setHardware(worker, hw)
		if p := h.previewEngine(worker, model, http.StatusOK); p.Blocker != cause || !p.EngineRequired || p.Family != "speechbrain" || p.Kind != "separation" || !p.Active {
			t.Fatalf("%s: preview %+v", cause, p)
		}
		if raw := h.adminCall(http.MethodPost, "/api/admin/workers/"+worker+"/voice-model-orders", map[string]any{"model_id": model, "idempotency_key": "separation-refused-" + cause[:8]}, http.StatusConflict); !strings.Contains(string(raw), cause) {
			t.Fatalf("%s: refusal %s", cause, raw)
		}
		if rows() != [4]int{} {
			t.Fatalf("%s: a refusal left rows: %v", cause, rows())
		}
	}

	// The explicit order: the engine of the family and of the kind first, the model waits for it.
	h.setHardware(worker, workerHardware{Python: "3.12", Platform: "linux-x86_64", RAMBytes: 64 << 30, DiskFreeBytes: 200 << 30})
	if p := h.previewEngine(worker, model, http.StatusOK); p.Blocker != "" || p.State != "missing" || p.Engine != family.Engine || p.Version != family.DefaultVer || p.DiskBytes != family.DiskBytes {
		t.Fatalf("preview: %+v", p)
	}
	first := h.order(worker, model, "separation-install-0001", http.StatusCreated)
	engines := h.delivered()
	var runtime map[string]any
	if len(engines) != 1 || json.Unmarshal(engines[0].Runtime, &runtime) != nil || runtime["source"] != "on_demand" || runtime["family"] != "speechbrain" || runtime["kind"] != "separation" {
		t.Fatalf("engine order of a separation model: %+v", engines)
	}
	if len(h.workerOrders()) != 0 {
		t.Fatal("the model was delivered before its engine")
	}
	if c := h.cells(worker)["separation"]; c.State != "requested" || c.ModelID != model {
		t.Fatalf("an ordered model: %+v", c)
	}
	h.ack(engines[0].OrderID)
	h.report(engines[0].OrderID, engines[0].Digest, "ready", "", http.StatusNoContent)
	orders := h.workerOrders()
	if len(orders) != 1 || orders[0].OrderID != first.OrderID || orders[0].Kind != "separation" || orders[0].Engine != "speechbrain-sepformer" ||
		orders[0].Repo != "speechbrain/sepformer-whamr16k" || orders[0].Revision != "21a5b500c6f52fddc387c5d9e5fb13ffd6f039c5" || orders[0].Action != "install" {
		t.Fatalf("model order delivered to the Worker: %+v", orders)
	}

	// The test inference fails: never ready, the stable cause is shown, and the model is eligible for nothing.
	h.modelAck(first.OrderID, true, "", http.StatusNoContent)
	h.modelReport(first.OrderID, model, "verifying", "", http.StatusNoContent)
	if c := h.cells(worker)["separation"]; c.State != "verifying" {
		t.Fatalf("a model being proven is not ready: %+v", c)
	}
	if h.runtimeReady(worker, model) {
		t.Fatal("a model is not usable before its test inference")
	}
	h.modelReport(first.OrderID, model, "failed", "inference_failed", http.StatusNoContent)
	if c := h.cells(worker)["separation"]; c.State != "failed" || c.CauseCode != "inference_failed" {
		t.Fatalf("a failed test inference: %+v", c)
	}
	if m := h.voiceModels().model(t, model); m.Installed {
		t.Fatalf("a failed model is not installed: %+v", m)
	} else if state, cause := stateOn(m, worker); state != "failed" || cause != "inference_failed" {
		t.Fatalf("the catalogue shows the failure and its cause: %+v", m)
	}
	if h.runtimeReady(worker, model) {
		t.Fatal("a failed model is not usable")
	}

	// A new explicit order reuses the ready engine; the successful test inference makes the model ready.
	second := h.order(worker, model, "separation-install-0002", http.StatusCreated)
	if second.OrderID == first.OrderID || len(h.delivered()) != 0 {
		t.Fatalf("a retry is a new model order on the same engine: %+v", second)
	}
	h.modelAck(second.OrderID, true, "", http.StatusNoContent)
	h.modelReport(second.OrderID, model, "ready", "", http.StatusNoContent)
	if c := h.cells(worker)["separation"]; c.State != "ready" || c.CauseCode != "" {
		t.Fatalf("a proven model: %+v", c)
	}
	if m := h.voiceModels().model(t, model); !m.Installed || m.Active || m.SeparatedVoices != 2 || m.SampleRateHz != 16000 {
		t.Fatalf("the catalogue shows the ready model and its limits: %+v", m)
	}
	if !h.runtimeReady(worker, model) {
		t.Fatal("a ready model bound to its ready engine is usable by a job that asks for it")
	}
	// The other Worker holds nothing: its state stays explicit, and it is asked nothing.
	if c := h.cells(h.secondID)["separation"]; c.State != "not_installed" {
		t.Fatalf("the other Worker: %+v", c)
	}
	h.asSecond(func() {
		if orders, _ := h.pulled(); len(orders) != 0 || len(h.delivered()) != 0 {
			t.Fatal("a Worker that was not ordered anything received an order")
		}
	})

	// Being ready opens no production job. No separation model is active, and a whole analysis of a clip
	// (audio plan, recognizer, then the voice scope) plans no separation, with the ready Worker claiming.
	if n := h.count(`SELECT count(*) FROM voice_active_model WHERE kind='separation'`); n != 0 || separationJobs() != 0 {
		t.Fatalf("a ready separation model activated something: %d active, %d jobs", n, separationJobs())
	}
	h.readyAudioWorker()
	h.startMultitrackAnalysis()
	steps := h.driveAudio(&audioScript{words: map[int][]scriptSegment{1: {{1.0, 2.0, "bonjour tout le monde"}}, 2: {{1.2, 2.4, "salut"}}}})
	if !hasStep(steps, "transcription.audio.") || hasStep(steps, "voice.separation.") {
		t.Fatalf("the analysis ran %v", steps)
	}
	if run := h.analysis(); run == nil || run.Status != "completed" {
		t.Fatalf("the analysis must complete: %+v", run)
	}
	h.startAnalysis(map[string]any{"scope": "voice"}, http.StatusCreated)
	for range 10 {
		job, ok := h.claim() // the Worker declares every voice step
		if !ok {
			break
		}
		if strings.Contains(job.Step, "separation") {
			t.Fatalf("the analysis of a clip dispatched %s", job.Step)
		}
		h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", job.ID), map[string]any{"error": "synthetic_stop"}, job.Lease, http.StatusOK)
	}
	if separationJobs() != 0 {
		t.Fatalf("the analysis of a clip planned %d separation jobs", separationJobs())
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.diarization.%'`, h.clipID); n == 0 {
		t.Fatal("the voice scope must have planned its diarization: the guard above would prove nothing")
	}
}
