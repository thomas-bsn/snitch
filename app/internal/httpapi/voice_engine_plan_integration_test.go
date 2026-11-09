//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type gotFamily struct {
	Family           string   `json:"family"`
	Engine           string   `json:"engine"`
	DefaultVersion   string   `json:"default_version"`
	Kinds            []string `json:"kinds"`
	Python           string   `json:"python"`
	Platform         string   `json:"platform"`
	CUDA             string   `json:"cuda"`
	VRAMBytes        int64    `json:"vram_bytes"`
	RAMBytes         int64    `json:"ram_bytes"`
	DiskBytes        int64    `json:"disk_bytes"`
	EstimatedSeconds int64    `json:"estimated_seconds"`
}

type gotPlan struct {
	Family           string `json:"family"`
	Version          string `json:"version"`
	DiskBytes        int64  `json:"disk_bytes"`
	EstimatedSeconds int64  `json:"estimated_seconds"`
	Accepted         int    `json:"accepted"`
	Workers          []struct {
		WorkerID       string `json:"worker_id"`
		Status         string `json:"status"`
		CauseCode      string `json:"cause_code"`
		RequiredBytes  *int64 `json:"required_bytes"`
		AvailableBytes *int64 `json:"available_bytes"`
	} `json:"workers"`
}

func (h *engineOrderHarness) plan(want int, body map[string]any) gotPlan {
	h.t.Helper()
	raw := h.adminCall(http.MethodPost, "/api/admin/voice-engines/on-demand-plan", body, want)
	var out gotPlan
	if want == http.StatusOK {
		if err := json.Unmarshal(raw, &out); err != nil {
			h.t.Fatal(err)
		}
	}
	return out
}

// The family table is listed and a plan is computed before any order; planning creates nothing (#371).
func testVoiceEngineFamiliesAndPlan(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	fixture, other := h.fixtureWorkerID(), h.secondID

	// The list does not depend on any signed release being imported.
	var listed struct {
		Families []gotFamily `json:"families"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines/families", nil, http.StatusOK), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Families) != 2 {
		t.Fatalf("families: %+v", listed.Families)
	}
	// SpeechBrain (#374) covers fingerprint and separation; its engine differs from the signed ECAPA release.
	if sb := listed.Families[1]; sb.Family != "speechbrain" || sb.Engine != "speechbrain" || sb.DefaultVersion != "1.0.3" || len(sb.Kinds) != 2 || sb.Kinds[0] != "fingerprint" || sb.Kinds[1] != "separation" {
		t.Fatalf("speechbrain family: %+v", sb)
	}
	f := listed.Families[0]
	if f.Family != "nemo" || f.Engine != "nemo-toolkit" || f.DefaultVersion != "3.0.0" || len(f.Kinds) != 2 || f.Kinds[0] != "diarization" ||
		f.Python != "3.12" || f.Platform != "linux-x86_64" || f.DiskBytes != 12<<30 || f.RAMBytes != 8<<30 || f.EstimatedSeconds != 600 {
		t.Fatalf("family: %+v", f)
	}

	h.setHardware(fixture, workerHardware{Python: "3.12", Platform: "linux-x86_64", RAMBytes: 64 << 30, DiskFreeBytes: 200 << 30})
	h.setHardware(other, workerHardware{Python: "3.12", Platform: "linux-x86_64", RAMBytes: 64 << 30, DiskFreeBytes: 1 << 30})

	// The default version applies when none is given; each Worker is judged, the totals count accepted ones.
	p := h.plan(http.StatusOK, map[string]any{"family": "nemo", "worker_ids": []string{fixture, other}})
	if p.Version != "3.0.0" || p.Accepted != 1 || p.DiskBytes != 12<<30 || p.EstimatedSeconds != 600 || len(p.Workers) != 2 {
		t.Fatalf("plan: %+v", p)
	}
	if w := p.Workers[0]; w.WorkerID != fixture || w.Status != "ok" {
		t.Fatalf("first worker: %+v", w)
	}
	if w := p.Workers[1]; w.Status != "refused" || w.CauseCode != "insufficient_disk" || w.RequiredBytes == nil || *w.RequiredBytes != 12<<30 || w.AvailableBytes == nil || *w.AvailableBytes != 1<<30 {
		t.Fatalf("second worker: %+v", w)
	}
	// An edited version is carried back; an open order is reported as the order itself would be refused.
	if p := h.plan(http.StatusOK, map[string]any{"family": "nemo", "version": "2.9.0", "worker_ids": []string{fixture}}); p.Version != "2.9.0" || p.Accepted != 1 {
		t.Fatalf("edited version: %+v", p)
	}
	h.orderFamily("nemo", "", http.StatusOK, map[string]string{"worker_id": fixture, "idempotency_key": "plan-order-000001"})
	orders := h.count(`SELECT count(*) FROM worker_engine_orders`)
	registry := h.count(`SELECT count(*) FROM voice_engine_registry`)
	p = h.plan(http.StatusOK, map[string]any{"family": "nemo", "worker_ids": []string{fixture, "wrk_missing"}})
	if p.Accepted != 0 || p.Workers[0].CauseCode != "engine_order_in_progress" || p.Workers[1].CauseCode != "worker_not_found" {
		t.Fatalf("open order and missing Worker: %+v", p)
	}

	// Planning creates nothing and bad input is a request error.
	h.plan(http.StatusBadRequest, map[string]any{"family": "unknown", "worker_ids": []string{fixture}})
	h.plan(http.StatusBadRequest, map[string]any{"family": "nemo", "version": "--index-url=x", "worker_ids": []string{fixture}})
	h.plan(http.StatusBadRequest, map[string]any{"family": "nemo", "worker_ids": []string{}})
	if h.count(`SELECT count(*) FROM worker_engine_orders`) != orders || h.count(`SELECT count(*) FROM voice_engine_registry`) != registry {
		t.Fatal("planning created a row")
	}
}
