package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
)

// Resource checks of an engine release against what a Worker reports of its machine (#354, ADR 0055).
//
// The check is a refusal with a stable cause, never a repair: it installs, downloads and falls back to nothing.
// A Worker that never reported its hardware is not blocked here (older Workers keep working) and checks by
// itself before installing. The Worker's own check stays the last word.

const (
	causePlatform     = "platform_incompatible"
	causeCUDA         = "cuda_incompatible"
	causeInsufficient = "insufficient_ram"
	causeVRAM         = "insufficient_vram"
	causeDisk         = "insufficient_disk"
)

var (
	hardwareVersion  = regexp.MustCompile(`^([0-9]{1,2}\.[0-9]{1,2})?$`)
	hardwarePlatform = regexp.MustCompile(`^[a-z0-9_-]{0,32}$`)
)

const maxHardwareBytes = int64(1) << 50

type workerHardware struct {
	Python        string    `json:"python"`
	Platform      string    `json:"platform"`
	CUDA          string    `json:"cuda"`
	RAMBytes      int64     `json:"ram_bytes"`
	VRAMBytes     int64     `json:"vram_bytes"`
	DiskFreeBytes int64     `json:"disk_free_bytes"`
	CPUFallback   bool      `json:"cpu_fallback"`
	ReportedAt    time.Time `json:"reported_at"`
}

// engineVerdict is the exact reason an engine cannot be installed on a Worker, with the numbers behind it.
type engineVerdict struct {
	Cause     string
	Required  int64
	Available int64
}

// checkEngineResources decides one release on one Worker. reservedDisk is what the open orders of that Worker
// already claim. skipDisk is set when the runtime is already on the Worker and needs no new space.
func checkEngineResources(m engineManifest, hw *workerHardware, reservedDisk int64, skipDisk bool) engineVerdict {
	if hw == nil {
		return engineVerdict{}
	}
	if (hw.Python != "" && hw.Python != m.Python) || (hw.Platform != "" && hw.Platform != m.Platform) {
		return engineVerdict{Cause: causePlatform}
	}
	// Running on the CPU is never automatic: a CPU runtime is its own approved release. A CUDA release only runs
	// without a GPU when its contract and the Worker's configuration both allow it, and still installs nothing else.
	onCPU := m.CUDA == "cpu"
	if !onCPU {
		if hw.CUDA == "" {
			if !(m.CPUFallback && hw.CPUFallback) {
				return engineVerdict{Cause: causeCUDA}
			}
			onCPU = true
		} else if hw.CUDA != m.CUDA {
			return engineVerdict{Cause: causeCUDA}
		}
	}
	if !onCPU && m.VRAMBytes > hw.VRAMBytes {
		return engineVerdict{Cause: causeVRAM, Required: m.VRAMBytes, Available: hw.VRAMBytes}
	}
	if m.RAMBytes > 0 && hw.RAMBytes > 0 && m.RAMBytes > hw.RAMBytes {
		return engineVerdict{Cause: causeInsufficient, Required: m.RAMBytes, Available: hw.RAMBytes}
	}
	if !skipDisk {
		available := hw.DiskFreeBytes - reservedDisk
		if available < 0 {
			available = 0
		}
		if m.DiskBytes > available {
			return engineVerdict{Cause: causeDisk, Required: m.DiskBytes, Available: available}
		}
	}
	return engineVerdict{}
}

func loadWorkerHardware(ctx context.Context, db pgQuerier, workerID string) (*workerHardware, error) {
	var hw workerHardware
	err := db.QueryRow(ctx, `SELECT python,platform,cuda,ram_bytes,vram_bytes,disk_free_bytes,cpu_fallback,reported_at FROM worker_hardware WHERE worker_id=$1`, workerID).
		Scan(&hw.Python, &hw.Platform, &hw.CUDA, &hw.RAMBytes, &hw.VRAMBytes, &hw.DiskFreeBytes, &hw.CPUFallback, &hw.ReportedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &hw, err
}

// reservedDisk is the disk that the orders still open on a Worker claim: engine orders and model orders that
// carry a runtime. Counted whole (an acknowledged order may already have used part of its space), which can only
// refuse early and never over-commit. Callers hold the voice model lock, so two orders cannot both take the same room.
func reservedDisk(ctx context.Context, db pgQuerier, workerID string) (int64, error) {
	var total int64
	err := db.QueryRow(ctx, `SELECT
		COALESCE((SELECT sum((manifest->>'disk_bytes')::bigint) FROM worker_engine_orders WHERE worker_id=$1 AND status IN ('pending','acknowledged')),0)
		+ COALESCE((SELECT sum((runtime->>'disk_bytes')::bigint) FROM worker_model_orders WHERE worker_id=$1 AND status IN ('pending','acknowledged') AND action='install' AND runtime IS NOT NULL),0)`, workerID).Scan(&total)
	return total, err
}

// checkWorkerEngine runs the check for a manifest stored in an order or the registry. The refusal is an
// orderFailure carrying the stable cause.
func checkWorkerEngine(ctx context.Context, db pgQuerier, workerID string, manifest json.RawMessage, skipDisk bool) error {
	var m engineManifest
	if json.Unmarshal(manifest, &m) != nil {
		return orderFailure{http.StatusConflict, "runtime_incompatible"}
	}
	hw, err := loadWorkerHardware(ctx, db, workerID)
	if err != nil || hw == nil {
		return err
	}
	reserved, err := reservedDisk(ctx, db, workerID)
	if err != nil {
		return err
	}
	if v := checkEngineResources(m, hw, reserved, skipDisk); v.Cause != "" {
		return orderFailure{http.StatusConflict, v.Cause}
	}
	return nil
}

// postWorkerHardware records what the Worker reports of its machine. It is an observation: it orders nothing.
func (s *Server) postWorkerHardware(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Python        string `json:"python"`
		Platform      string `json:"platform"`
		CUDA          string `json:"cuda"`
		RAMBytes      int64  `json:"ram_bytes"`
		VRAMBytes     int64  `json:"vram_bytes"`
		DiskFreeBytes int64  `json:"disk_free_bytes"`
		CPUFallback   bool   `json:"cpu_fallback"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !hardwareVersion.MatchString(in.Python) || !hardwarePlatform.MatchString(in.Platform) || !hardwareVersion.MatchString(in.CUDA) ||
		in.RAMBytes < 0 || in.VRAMBytes < 0 || in.DiskFreeBytes < 0 || in.RAMBytes > maxHardwareBytes || in.VRAMBytes > maxHardwareBytes || in.DiskFreeBytes > maxHardwareBytes {
		fail(w, http.StatusBadRequest, "invalid_worker_hardware")
		return
	}
	worker := currentWorker(r)
	if _, err := s.Pool.Exec(r.Context(), `INSERT INTO worker_hardware(worker_id,python,platform,cuda,ram_bytes,vram_bytes,disk_free_bytes,cpu_fallback,reported_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,now())
		ON CONFLICT(worker_id) DO UPDATE SET python=EXCLUDED.python,platform=EXCLUDED.platform,cuda=EXCLUDED.cuda,ram_bytes=EXCLUDED.ram_bytes,vram_bytes=EXCLUDED.vram_bytes,disk_free_bytes=EXCLUDED.disk_free_bytes,cpu_fallback=EXCLUDED.cpu_fallback,reported_at=now()`,
		worker.ID, in.Python, in.Platform, in.CUDA, in.RAMBytes, in.VRAMBytes, in.DiskFreeBytes, in.CPUFallback); err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
