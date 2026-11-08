//go:build integration

package httpapi

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The tickets #351 (cancellation), #353 (removal), #352 (runtime update) and #354 (resource checks) were written
// in parallel and each rewrote the same CHECK constraints from the same base. This test inserts a row for every
// state, cause, intent and status any of them introduced: a legitimate order must never be rejected by the database.
var engineCauseUnion = []string{"", "disk_full", "runtime_incompatible", "hf_network", "invalid_order", "runtime_missing",
	"cancelled", "runtime_in_use", "platform_incompatible", "cuda_incompatible", "insufficient_ram", "insufficient_vram"}

var engineInventoryStates = []string{"requested", "downloading_runtime", "installing", "verifying", "ready", "failed", "removing", "removed"}

func testVoiceEngineConstraints(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	worker := h.fixtureWorkerID()
	digest := h.sign("1.0.0")
	n := 0
	var manifest []byte
	if err := pool.QueryRow(t.Context(), `SELECT manifest FROM voice_engine_registry WHERE digest=$1`, digest).Scan(&manifest); err != nil {
		t.Fatal(err)
	}
	h.installReady(digest, worker, "constraint-install-0001")
	// Orders: each intent with each status and each cause.
	for _, intent := range []string{"install", "uninstall"} {
		for _, status := range []string{"pending", "acknowledged", "done", "failed", "refused", "cancelled"} {
			for _, cause := range engineCauseUnion {
				n++
				if _, err := pool.Exec(t.Context(), `INSERT INTO worker_engine_orders(id,worker_id,digest,engine,version,intent,manifest,generation,idempotency_key,author_name,status,cause_code)
					VALUES($1,$2,$3,'nemo-titanet','1.0.0',$4,$5,$6,$7,'synthetic',$8,$9)`,
					fmt.Sprintf("ord_constraint_%04d_padding", n), worker, digest, intent, manifest, 1000+n, fmt.Sprintf("constraint-key-%04d", n), status, cause); err != nil {
					t.Fatalf("order %s/%s/%q rejected: %v", intent, status, cause, err)
				}
			}
		}
	}
	// Inventory: each state with each cause, on one row.
	for _, state := range engineInventoryStates {
		for _, cause := range engineCauseUnion {
			if _, err := pool.Exec(t.Context(), `UPDATE worker_engine_inventory SET state=$3,cause_code=$4 WHERE worker_id=$1 AND digest=$2`, worker, digest, state, cause); err != nil {
				t.Fatalf("inventory %s/%q rejected: %v", state, cause, err)
			}
		}
	}
	// Every cause a Worker can report is accepted by the report endpoint's own list.
	for _, cause := range engineCauseUnion {
		if cause != "" && !engineFailureCauses[cause] {
			t.Errorf("cause %q is stored but not accepted from a Worker report", cause)
		}
	}
	// Every value the database rejects stays rejected.
	if _, err := pool.Exec(t.Context(), `UPDATE worker_engine_inventory SET cause_code='unknown_cause' WHERE worker_id=$1`, worker); err == nil {
		t.Fatal("an unknown cause was accepted")
	}
}
