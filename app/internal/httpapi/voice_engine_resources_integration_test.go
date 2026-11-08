//go:build integration

package httpapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// signRelease imports a release with its own manifest: the harness signer only varies the version.
func (h *engineOrderHarness) signRelease(m engineManifest) string {
	h.t.Helper()
	raw, _ := json.Marshal(m)
	encode := base64.StdEncoding.EncodeToString
	pub, private, _ := ed25519.GenerateKey(rand.Reader)
	keys, _ := json.Marshal(map[string]string{"resources": encode(pub)})
	h.t.Setenv(engineTrustEnv, string(keys))
	var release engineRelease
	body := h.adminCall(http.MethodPost, "/api/admin/voice-engines", map[string]string{"key_id": "resources", "payload": encode(raw), "signature": encode(ed25519.Sign(private, raw))}, http.StatusCreated)
	if err := json.Unmarshal(body, &release); err != nil {
		h.t.Fatal(err)
	}
	return release.Digest
}

func (h *engineOrderHarness) setHardware(workerID string, hw workerHardware) {
	h.t.Helper()
	if _, err := h.pool.Exec(h.t.Context(), `INSERT INTO worker_hardware(worker_id,python,platform,cuda,ram_bytes,vram_bytes,disk_free_bytes,cpu_fallback) VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(worker_id) DO UPDATE SET python=EXCLUDED.python,platform=EXCLUDED.platform,cuda=EXCLUDED.cuda,ram_bytes=EXCLUDED.ram_bytes,vram_bytes=EXCLUDED.vram_bytes,disk_free_bytes=EXCLUDED.disk_free_bytes,cpu_fallback=EXCLUDED.cpu_fallback`,
		workerID, hw.Python, hw.Platform, hw.CUDA, hw.RAMBytes, hw.VRAMBytes, hw.DiskFreeBytes, hw.CPUFallback); err != nil {
		h.t.Fatal(err)
	}
}

func (h *engineOrderHarness) only(digest, workerID, key string) gotEngineOrderResult {
	h.t.Helper()
	return h.orderOn(digest, http.StatusOK, map[string]string{"worker_id": workerID, "idempotency_key": key})[0]
}

// ecapaManifest is a second engine, so that two orders can be open on one Worker at once.
func ecapaManifest(disk int64) engineManifest {
	m := testEngineManifest()
	m.Engine, m.Architecture, m.DiskBytes = "speechbrain-ecapa", "ecapa-tdnn", disk
	m.Wheels[0].Size = 100
	return m
}

func testVoiceEngineResources(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	a, b := h.fixtureWorkerID(), h.secondID
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/workers", map[string]any{"name": "engine-legacy", "max_concurrency": 1}, http.StatusCreated), &created); err != nil {
		t.Fatal(err)
	}
	legacy := created.ID

	cuda := func(version string, mutate func(*engineManifest)) string {
		m := testEngineManifest()
		m.Version, m.CUDA, m.VRAMBytes, m.DiskBytes = version, "12.4", 4*gib, 6000
		if mutate != nil {
			mutate(&m)
		}
		return h.signRelease(m)
	}

	// A Worker that never reported its hardware is not blocked: legacy installations keep working.
	if r := h.only(cuda("legacy", nil), legacy, "res-legacy-00001"); r.Status != "created" {
		t.Fatalf("a Worker without hardware report must keep being served: %+v", r)
	}

	// The report is validated and is an observation only: it orders nothing.
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/hardware", map[string]any{"python": "3.12", "platform": "Linux!", "cuda": "", "ram_bytes": 1, "vram_bytes": 0, "disk_free_bytes": 1, "cpu_fallback": false}, "", http.StatusBadRequest)
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/hardware", map[string]any{"python": "3.12", "platform": "linux-x86_64", "cuda": "12.4", "ram_bytes": 16 * gib, "vram_bytes": 8 * gib, "disk_free_bytes": 10000, "cpu_fallback": false}, "", http.StatusNoContent)
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE worker_id=$1`, a); n != 0 {
		t.Fatalf("a hardware report created %d orders", n)
	}
	var view struct {
		Workers []struct {
			WorkerID string `json:"worker_id"`
			Hardware *struct {
				CUDA      string `json:"cuda"`
				VRAMBytes int64  `json:"vram_bytes"`
			} `json:"hardware"`
			DiskReserved int64 `json:"disk_reserved_bytes"`
		} `json:"workers"`
	}
	read := func() {
		view.Workers = nil
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines/workers", nil, http.StatusOK), &view); err != nil {
			t.Fatal(err)
		}
	}
	read()
	for _, w := range view.Workers {
		switch w.WorkerID {
		case a:
			if w.Hardware == nil || w.Hardware.CUDA != "12.4" || w.Hardware.VRAMBytes != 8*gib {
				t.Fatalf("the screen must see the reported machine: %+v", w.Hardware)
			}
		case legacy:
			if w.Hardware != nil {
				t.Fatal("a Worker that never reported must show no hardware")
			}
		}
	}

	// Cumulated disk and concurrent reservations: 10000 free, 6000 reserved by the open order of another engine.
	first := cuda("1.0.0", nil)
	r := h.only(first, a, "res-a-first-0001")
	if r.Status != "created" {
		t.Fatalf("first order: %+v", r)
	}
	second := h.signRelease(func() engineManifest {
		m := ecapaManifest(6000)
		m.CUDA, m.VRAMBytes, m.Version = "12.4", 4*gib, "1.0.0"
		return m
	}())
	if r := h.only(second, a, "res-a-second-001"); r.Status != "refused" || r.Cause != "insufficient_disk" {
		t.Fatalf("the second order must not take the room already reserved: %+v", r)
	}
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE worker_id=$1`, a); n != 1 {
		t.Fatalf("a refusal recorded an order: %d", n)
	}
	read()
	for _, w := range view.Workers {
		if w.WorkerID == a && w.DiskReserved != 6000 {
			t.Fatalf("reserved disk %d, want 6000", w.DiskReserved)
		}
	}
	// A failed order releases its reservation: the room can be used again.
	h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+r.Order.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
	h.report(r.Order.OrderID, first, "failed", "insufficient_vram", http.StatusNoContent)
	if e, _ := h.entry(a, first); e.State != "failed" || e.CauseCode != "insufficient_vram" {
		t.Fatalf("a Worker-side resource failure must keep its stable cause: %+v", e)
	}
	again := h.only(second, a, "res-a-second-002")
	if again.Status != "created" {
		t.Fatalf("the room was released: %+v", again)
	}

	// Stable causes, with nothing recorded for a refusal.
	h.setHardware(b, workerHardware{Python: "3.12", Platform: "linux-x86_64", CUDA: "12.4", RAMBytes: 16 * gib, VRAMBytes: 8 * gib, DiskFreeBytes: 10000})
	before := h.count(`SELECT count(*) FROM worker_engine_orders`)
	for name, tc := range map[string]struct{ digest, cause string }{
		"other-CUDA": {cuda("cuda-118", func(m *engineManifest) { m.CUDA = "11.8" }), "cuda_incompatible"},
		"VRAM":       {cuda("vram-16", func(m *engineManifest) { m.VRAMBytes = 16 * gib }), "insufficient_vram"},
		"RAM":        {cuda("ram-64", func(m *engineManifest) { m.RAMBytes = 64 * gib }), "insufficient_ram"},
		"disk":       {cuda("disk-big", func(m *engineManifest) { m.DiskBytes = 20000 }), "insufficient_disk"},
	} {
		if r := h.only(tc.digest, b, "res-"+name+"-0001"); r.Status != "refused" || r.Cause != tc.cause {
			t.Fatalf("%s: %+v", name, r)
		}
	}
	if h.count(`SELECT count(*) FROM worker_engine_orders`) != before {
		t.Fatal("a refusal recorded an order")
	}

	// Incompatible platform and CPU fallback, on Worker B.
	h.setHardware(b, workerHardware{Python: "3.12", Platform: "linux-aarch64", CUDA: "12.4", RAMBytes: 16 * gib, VRAMBytes: 8 * gib, DiskFreeBytes: 1 << 40})
	if r := h.only(cuda("platform", nil), b, "res-platform-0001"); r.Status != "refused" || r.Cause != "platform_incompatible" {
		t.Fatalf("platform: %+v", r)
	}
	h.setHardware(b, workerHardware{Python: "3.12", Platform: "linux-x86_64", CUDA: "", RAMBytes: 16 * gib, VRAMBytes: 0, DiskFreeBytes: 1 << 40, CPUFallback: true})
	// The Worker's configuration alone is not enough: the contract must allow the CPU too.
	if r := h.only(cuda("nofallback", nil), b, "res-nofallback-001"); r.Status != "refused" || r.Cause != "cuda_incompatible" {
		t.Fatalf("CUDA release without contract on a GPU-less Worker: %+v", r)
	}
	// Nothing is installed in its place: a CPU runtime is its own approved release, ordered on purpose.
	if n := h.count(`SELECT count(*) FROM worker_engine_orders WHERE worker_id=$1`, b); n != 0 {
		t.Fatalf("a refusal installed something else: %d orders", n)
	}
	withContract := cuda("fallback-ok", func(m *engineManifest) { m.CPUFallback = true })
	if r := h.only(withContract, b, "res-fallback-0001"); r.Status != "created" {
		t.Fatalf("contract and configuration both allow the CPU: %+v", r)
	}
	h.setHardware(b, workerHardware{Python: "3.12", Platform: "linux-x86_64", CUDA: "", RAMBytes: 16 * gib, VRAMBytes: 0, DiskFreeBytes: 1 << 40, CPUFallback: false})
	if r := h.only(cuda("contract-only", func(m *engineManifest) {
		m.CPUFallback = true
		m.Engine, m.Architecture = "speechbrain-ecapa", "ecapa-tdnn"
	}), b, "res-contract-0001"); r.Status != "refused" || r.Cause != "cuda_incompatible" {
		t.Fatalf("the contract alone is not enough: %+v", r)
	}

	// An engine that works is never touched by a later, poorer hardware report.
	h.workerCall(http.MethodPost, "/api/worker/voice-engine-orders/"+again.Order.OrderID+"/ack", map[string]any{"accepted": true, "cause_code": ""}, "", http.StatusNoContent)
	h.report(again.Order.OrderID, second, "ready", "", http.StatusNoContent)
	h.workerCall(http.MethodPost, "/api/worker/voice-engines/hardware", map[string]any{"python": "3.12", "platform": "linux-x86_64", "cuda": "", "ram_bytes": 1 * gib, "vram_bytes": 0, "disk_free_bytes": 1, "cpu_fallback": false}, "", http.StatusNoContent)
	if e, ok := h.entry(a, second); !ok || e.State != "ready" {
		t.Fatalf("a hardware report must not change an installation: %+v", e)
	}
}
