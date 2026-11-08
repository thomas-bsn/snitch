//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testWorkerEngineGroups(t *testing.T, pool *pgxpool.Pool) {
	h := newEngineOrderHarness(t, pool)
	id := h.fixtureWorkerID()
	path := "/api/admin/workers/" + id + "/engine-group"
	read := func() map[string]string {
		t.Helper()
		var view struct {
			Workers []struct {
				ID    string `json:"worker_id"`
				Group string `json:"group_name"`
			} `json:"workers"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines/workers", nil, http.StatusOK), &view); err != nil {
			t.Fatal(err)
		}
		result := map[string]string{}
		for _, worker := range view.Workers {
			result[worker.ID] = worker.Group
		}
		return result
	}
	h.adminCall(http.MethodPut, path, map[string]string{"group_name": "Atelier"}, http.StatusNoContent)
	h.adminCall(http.MethodPut, "/api/admin/workers/"+h.secondID+"/engine-group", map[string]string{"group_name": "Atelier"}, http.StatusNoContent)
	groups := read()
	if groups[id] != "Atelier" || groups[h.secondID] != "Atelier" {
		t.Fatalf("group not persisted: %+v", groups)
	}
	h.adminCall(http.MethodPut, path, map[string]string{"group_name": "Studio"}, http.StatusNoContent)
	if read()[h.secondID] != "Atelier" || read()[id] != "Studio" {
		t.Fatal("changing one assignment changed another")
	}
	h.adminCall(http.MethodPut, path, map[string]string{"group_name": ""}, http.StatusNoContent)
	if read()[id] != "" {
		t.Fatal("group not cleared")
	}
	h.adminCall(http.MethodPut, path, map[string]string{}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, path, map[string]string{"group_name": strings.Repeat("x", 81)}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, path, map[string]string{"group_name": "bad\nname"}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, "/api/admin/workers/wk_missing_synthetic/engine-group", map[string]string{"group_name": "Atelier"}, http.StatusNotFound)
	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM worker_engine_orders`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("grouping must not order engines: %d %v", count, err)
	}
}
