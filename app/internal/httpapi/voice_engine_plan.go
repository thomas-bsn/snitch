package httpapi

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"sync"

	"github.com/jackc/pgx/v5"
)

// Families and plan of on-demand engines (#371, ADR 0056). The family table of voice_engine_families.go is the
// only source: the screen lists it and asks for a plan before any order. A plan reads and creates nothing.

type engineFamilyView struct {
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
	// RecentVersions are the latest stable releases of the library, newest first; empty when the index is unreachable.
	RecentVersions []string `json:"recent_versions"`
	// GitRepository is the repository an engine of this family may be installed from at a pinned commit; empty when
	// the family only installs published releases.
	GitRepository string `json:"git_repository"`
}

func (s *Server) listEngineFamilies(w http.ResponseWriter, r *http.Request) {
	families := make([]engineFamilyView, len(engineFamilies))
	keys := make([]string, 0, len(engineFamilies))
	for key := range engineFamilies {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var wg sync.WaitGroup
	for i, key := range keys {
		f := engineFamilies[key]
		families[i] = engineFamilyView{
			Family: f.Key, Engine: f.Engine, DefaultVersion: f.DefaultVer, Kinds: f.Kinds, Python: f.Python, Platform: f.Platform,
			CUDA: f.CUDA, VRAMBytes: f.VRAMBytes, RAMBytes: f.RAMBytes, DiskBytes: f.DiskBytes, EstimatedSeconds: f.EstimatedSeconds,
			RecentVersions: []string{}, GitRepository: f.GitURL,
		}
		wg.Add(1)
		go func(i int, distribution string) {
			defer wg.Done()
			if versions := s.pypiIndex().recent(r.Context(), distribution); versions != nil {
				families[i].RecentVersions = versions
			}
		}(i, f.Distribution)
	}
	wg.Wait()
	reply(w, http.StatusOK, map[string]any{"families": families})
}

type enginePlanWorker struct {
	WorkerID       string `json:"worker_id"`
	Status         string `json:"status"`
	CauseCode      string `json:"cause_code,omitempty"`
	RequiredBytes  *int64 `json:"required_bytes,omitempty"`
	AvailableBytes *int64 `json:"available_bytes,omitempty"`
}

// planOnDemandWorker decides one Worker the way the order would, without recording anything.
func (s *Server) planOnDemandWorker(ctx context.Context, f engineFamily, version, kind, workerID string) (enginePlanWorker, error) {
	out := enginePlanWorker{WorkerID: workerID, Status: "refused"}
	_, digest, err := onDemandManifest(f, version, kind)
	if err != nil {
		return out, err
	}
	var exists bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, workerID).Scan(&exists); err != nil {
		return out, err
	}
	if !exists {
		out.CauseCode = "worker_not_found"
		return out, nil
	}
	var open bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_engine_orders WHERE worker_id=$1 AND engine=$2 AND status IN ('pending','acknowledged'))`, workerID, f.Engine).Scan(&open); err != nil {
		return out, err
	}
	if open {
		out.CauseCode = "engine_order_in_progress"
		return out, nil
	}
	var state string
	switch err := s.Pool.QueryRow(ctx, `SELECT state FROM worker_engine_inventory WHERE worker_id=$1 AND digest=$2`, workerID, digest).Scan(&state); {
	case errors.Is(err, pgx.ErrNoRows), err == nil && (state == "failed" || state == "removed"):
	case err != nil:
		return out, err
	case state == "ready":
		out.CauseCode = "engine_already_installed"
		return out, nil
	default:
		out.CauseCode = "engine_order_in_progress"
		return out, nil
	}
	m := engineManifest{Python: f.Python, Platform: f.Platform, CUDA: f.CUDA, DiskBytes: f.DiskBytes, VRAMBytes: f.VRAMBytes, RAMBytes: f.RAMBytes}
	hw, err := loadWorkerHardware(ctx, s.Pool, workerID)
	if err != nil {
		return out, err
	}
	if hw != nil {
		reserved, err := reservedDisk(ctx, s.Pool, workerID)
		if err != nil {
			return out, err
		}
		if v := checkEngineResources(m, hw, reserved, false); v.Cause != "" {
			out.CauseCode = v.Cause
			if v.Required > 0 {
				out.RequiredBytes, out.AvailableBytes = &v.Required, &v.Available
			}
			return out, nil
		}
	}
	out.Status = "ok"
	return out, nil
}

// planOnDemandEngine answers what an order of one family and version would do on each Worker: the disk and the
// time it claims, and the refusal of each Worker that cannot host it. The confirmation is the order itself.
func (s *Server) planOnDemandEngine(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Family    string   `json:"family"`
		Version   string   `json:"version"`
		Kind      string   `json:"kind"`
		WorkerIDs []string `json:"worker_ids"`
	}
	if !decode(w, r, &input) {
		return
	}
	family, ok := engineFamilies[input.Family]
	if !ok {
		fail(w, http.StatusBadRequest, "unsupported_engine_family")
		return
	}
	version := input.Version
	if version == "" {
		version = family.DefaultVer
	}
	kind := input.Kind
	if kind == "" {
		kind = family.Kinds[0]
	}
	if !family.covers(kind) || !validEngineVersion(family, version) || len(input.WorkerIDs) == 0 || len(input.WorkerIDs) > 100 {
		fail(w, http.StatusBadRequest, "invalid_engine_order")
		return
	}
	workers := make([]enginePlanWorker, 0, len(input.WorkerIDs))
	accepted := 0
	for _, id := range input.WorkerIDs {
		item, err := s.planOnDemandWorker(r.Context(), family, version, kind, id)
		if err != nil {
			s.internal(w, err)
			return
		}
		if item.Status == "ok" {
			accepted++
		}
		workers = append(workers, item)
	}
	// Workers install in parallel: the disk adds up, the time is that of one installation.
	seconds := int64(0)
	if accepted > 0 {
		seconds = family.EstimatedSeconds
	}
	reply(w, http.StatusOK, map[string]any{
		"family": family.Key, "version": version, "accepted": accepted,
		"disk_bytes": family.DiskBytes * int64(accepted), "estimated_seconds": seconds, "workers": workers,
	})
}
