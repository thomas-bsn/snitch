//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"snitch/internal/workerupdate"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testWorkerUpdates covers ADR 0036: the administrator's target version, the image handed to
// Workers, refusal of anything that is not a released tag, and the status a Worker reports.
func testWorkerUpdates(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newInstallHarness(t, pool, true)
	registry := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "anon"})
		case "/v2/thomas-bsn/snitch-worker/tags/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"tags": []string{"latest", "v1.0.0", "v1.1.0", "v1.10.0"}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(registry.Close)
	h.server.workerCatalog = &workerupdate.Catalog{Client: registry.Client(), Registry: registry.URL, Repo: "thomas-bsn/snitch-worker"}

	type listing struct {
		TargetVersion    *string  `json:"target_version"`
		TargetImage      *string  `json:"target_image"`
		ImageRepository  string   `json:"image_repository"`
		Catalog          []string `json:"catalog"`
		CatalogAvailable bool     `json:"catalog_available"`
		Workers          []struct {
			ID, Name, State, Version, Digest, CauseCode string
		} `json:"workers"`
	}
	list := func() listing {
		var l listing
		if err := json.Unmarshal(h.do(h.admin, http.MethodGet, h.ts.URL+"/api/admin/worker-updates", nil, "", "", false, http.StatusOK), &l); err != nil {
			t.Fatal(err)
		}
		return l
	}

	// Nothing chosen: no target, the catalogue is newest first and never contains "latest".
	first := list()
	if first.TargetVersion != nil || first.TargetImage != nil || first.ImageRepository != "ghcr.io/thomas-bsn/snitch-worker" {
		t.Fatalf("initial state: %+v", first)
	}
	if !first.CatalogAvailable || !reflect.DeepEqual(first.Catalog, []string{"v1.10.0", "v1.1.0", "v1.0.0"}) {
		t.Fatalf("catalogue: %v (available %v)", first.Catalog, first.CatalogAvailable)
	}

	// Only released tags are accepted; the target is never a free reference.
	for _, body := range []string{`{"version":"latest"}`, `{"version":"v1.0.0@sha256:` + repeatHex(64) + `"}`, `{"version":"ghcr.io/evil:v1.0.0"}`, `{"version":"v1.0"}`} {
		h.do(h.admin, http.MethodPut, h.ts.URL+"/api/admin/worker-updates", []byte(body), h.csrf, "", true, http.StatusBadRequest)
	}
	h.do(h.anon, http.MethodPut, h.ts.URL+"/api/admin/worker-updates", []byte(`{"version":"v1.1.0"}`), "", "", true, http.StatusUnauthorized)
	h.adminCall(http.MethodPut, "/api/admin/worker-updates", []byte(`{"version":"v1.1.0"}`), http.StatusOK)

	id, token, _, _ := h.register("")
	var target struct {
		TargetVersion *string `json:"target_version"`
		TargetImage   *string `json:"target_image"`
	}
	if err := json.Unmarshal(h.get(h.ts.URL+"/api/worker/update", token, http.StatusOK), &target); err != nil {
		t.Fatal(err)
	}
	if target.TargetImage == nil || *target.TargetImage != "ghcr.io/thomas-bsn/snitch-worker:v1.1.0" || *target.TargetVersion != "v1.1.0" {
		t.Fatalf("worker target: %+v", target)
	}
	h.get(h.ts.URL+"/api/worker/update", "", http.StatusUnauthorized)

	// Status reports: stable codes and digests only; anything else is refused.
	ok := []byte(`{"state":"updated","version":"v1.1.0","digest":"sha256:` + repeatHex(64) + `","cause_code":""}`)
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/update-status", ok, "", token, false, http.StatusNoContent)
	for _, body := range []string{
		`{"state":"exploded","version":"v1.1.0"}`,
		`{"state":"failed","version":"latest"}`,
		`{"state":"failed","version":"v1.1.0","digest":"sha256:XYZ"}`,
		`{"state":"failed","version":"v1.1.0","cause_code":"Free text with spaces"}`,
	} {
		h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/update-status", []byte(body), "", token, false, http.StatusBadRequest)
	}
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/update-status", ok, "", "", false, http.StatusUnauthorized)
	after := list()
	if len(after.Workers) != 1 || after.Workers[0].ID != id || after.Workers[0].State != "updated" || after.Workers[0].Version != "v1.1.0" {
		t.Fatalf("reported status: %+v", after.Workers)
	}

	// Choosing nothing turns automatic updates off for every Worker.
	h.adminCall(http.MethodPut, "/api/admin/worker-updates", []byte(`{"version":null}`), http.StatusOK)
	if err := json.Unmarshal(h.get(h.ts.URL+"/api/worker/update", token, http.StatusOK), &target); err != nil || target.TargetImage != nil {
		t.Fatalf("disabled target: %+v, %v", target, err)
	}

	// A revoked Worker disappears from the list and its status goes with its details.
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+id, []byte(`{"reason":"retired"}`), http.StatusNoContent)
	if left := list(); len(left.Workers) != 0 {
		t.Fatalf("revoked worker still listed: %+v", left.Workers)
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM worker_update_status WHERE worker_id=$1`, id).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("status rows after revocation: %d, %v", rows, err)
	}
}

func repeatHex(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = 'a'
	}
	return string(out)
}
