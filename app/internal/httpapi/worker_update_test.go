//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

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
	h.server.workerCatalogOverride = &workerupdate.Catalog{Client: registry.Client(), Registry: registry.URL, Repo: "thomas-bsn/snitch-worker"}

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

	// A registry without any release, or unreachable, still sends arrays: null would blank the screen.
	for name, handler := range map[string]http.HandlerFunc{
		"no tags": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/token" {
				_ = json.NewEncoder(w).Encode(map[string]string{"token": "anon"})
				return
			}
			_, _ = w.Write([]byte(`{"tags":[]}`))
		},
		"unreachable": func(w http.ResponseWriter, r *http.Request) { http.Error(w, "down", http.StatusBadGateway) },
	} {
		empty := httptest.NewTLSServer(handler)
		h.server.workerCatalogBuilt = &workerupdate.Catalog{Client: empty.Client(), Registry: empty.URL, Repo: "thomas-bsn/snitch-worker"}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(h.do(h.admin, http.MethodGet, h.ts.URL+"/api/admin/worker-updates", nil, "", "", false, http.StatusOK), &raw); err != nil {
			t.Fatal(err)
		}
		if string(raw["catalog"]) != "[]" {
			t.Fatalf("%s: catalog %s", name, raw["catalog"])
		}
		empty.Close()
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

// testWorkerLatestUpdates covers the "latest" strategy (ADR 0036, amended): a standing mode that follows the newest
// published release, written only when a higher one appears, and never lowering the target.
func testWorkerLatestUpdates(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newInstallHarness(t, pool, true)
	var mu sync.Mutex
	tags, down := []string{"latest", "sha-abc", "v1.0.0", "v1.1.0", "v1.10.0", "v2.0.0-rc1"}, false
	registry := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if down {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "anon"})
		case "/v2/thomas-bsn/snitch-worker/tags/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"tags": tags})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(registry.Close)
	// A tiny TTL and no failure backoff window to wait for: each poll sees the registry as it is now.
	newCatalog := func() *workerupdate.Catalog {
		return &workerupdate.Catalog{Client: registry.Client(), Registry: registry.URL, Repo: "thomas-bsn/snitch-worker", TTL: time.Nanosecond}
	}
	h.server.workerCatalogOverride = newCatalog()
	h.server.workerCatalogFor() // consume the Once: the catalogue is replaced directly from here on
	set := func(body string, status int) { h.adminCall(http.MethodPut, "/api/admin/worker-updates", []byte(body), status) }
	_, token, _, _ := h.register("")
	poll := func() (version string, idle bool) {
		var got struct {
			TargetVersion *string `json:"target_version"`
			TargetImage   *string `json:"target_image"`
			Idle          bool    `json:"idle"`
		}
		if err := json.Unmarshal(h.get(h.ts.URL+"/api/worker/update", token, http.StatusOK), &got); err != nil {
			t.Fatal(err)
		}
		if got.TargetVersion != nil {
			if got.TargetImage == nil || *got.TargetImage != "ghcr.io/thomas-bsn/snitch-worker:"+*got.TargetVersion {
				t.Fatalf("image does not match the version: %+v", got)
			}
			version = *got.TargetVersion
		}
		return version, got.Idle
	}
	stamp := func() (mode string, version *string, at time.Time) {
		if err := pool.QueryRow(ctx, `SELECT mode,version,updated_at FROM worker_update_target WHERE id=true`).Scan(&mode, &version, &at); err != nil {
			t.Fatal(err)
		}
		return
	}

	// Bad strategies are refused; pinned needs a version.
	set(`{"mode":"nightly","version":null}`, http.StatusBadRequest)
	set(`{"mode":"pinned","version":null}`, http.StatusBadRequest)
	set(`{"mode":"pinned","version":"latest"}`, http.StatusBadRequest)

	// Pinned stays put whatever the registry holds.
	set(`{"mode":"pinned","version":"v1.1.0"}`, http.StatusOK)
	if v, idle := poll(); v != "v1.1.0" || !idle {
		t.Fatalf("pinned: %q idle=%v", v, idle)
	}

	// Pinned -> latest: the newest stable release (never sha-*, latest or a prerelease) is resolved on the next poll.
	set(`{"mode":"latest","version":"v0.0.1"}`, http.StatusOK) // the sent version is ignored
	if v, _ := poll(); v != "v1.10.0" {
		t.Fatalf("latest resolved to %q", v)
	}
	mode, version, firstWrite := stamp()
	if mode != "latest" || version == nil || *version != "v1.10.0" {
		t.Fatalf("stored: %s %v", mode, version)
	}
	// Nothing new: many polls, many Workers, no write.
	for i := 0; i < 5; i++ {
		poll()
	}
	if _, _, at := stamp(); !at.Equal(firstWrite) {
		t.Fatal("polling without news must not write to the database")
	}
	var listed struct {
		Mode          string  `json:"mode"`
		TargetVersion *string `json:"target_version"`
		LatestVersion *string `json:"latest_version"`
		CheckedAt     *string `json:"checked_at"`
		CheckFailed   bool    `json:"check_failed"`
	}
	read := func() {
		listed = struct {
			Mode          string  `json:"mode"`
			TargetVersion *string `json:"target_version"`
			LatestVersion *string `json:"latest_version"`
			CheckedAt     *string `json:"checked_at"`
			CheckFailed   bool    `json:"check_failed"`
		}{}
		if err := json.Unmarshal(h.do(h.admin, http.MethodGet, h.ts.URL+"/api/admin/worker-updates", nil, "", "", false, http.StatusOK), &listed); err != nil {
			t.Fatal(err)
		}
	}
	read()
	if listed.Mode != "latest" || listed.LatestVersion == nil || *listed.LatestVersion != "v1.10.0" || listed.CheckedAt == nil || listed.CheckFailed {
		t.Fatalf("admin view: %+v", listed)
	}

	// A newer release appears: the effective target follows it, once.
	mu.Lock()
	tags = append(tags, "v1.11.0")
	mu.Unlock()
	if v, _ := poll(); v != "v1.11.0" {
		t.Fatalf("new release not followed: %q", v)
	}

	// The registry goes down: the last valid target is kept and the failure is visible.
	mu.Lock()
	down = true
	mu.Unlock()
	h.server.workerCatalogBuilt = newCatalog() // a restarted server has no cached list
	if v, _ := poll(); v != "v1.11.0" {
		t.Fatalf("registry outage lost the target: %q", v)
	}
	read()
	if !listed.CheckFailed || listed.TargetVersion == nil || *listed.TargetVersion != "v1.11.0" {
		t.Fatalf("outage not visible: %+v", listed)
	}

	// The registry comes back with only older releases (a tag was withdrawn): never a downgrade.
	mu.Lock()
	down, tags = false, []string{"v1.0.0", "v1.1.0"}
	mu.Unlock()
	h.server.workerCatalogBuilt = newCatalog()
	if v, _ := poll(); v != "v1.11.0" {
		t.Fatalf("latest must never go down: %q", v)
	}

	// Latest -> pinned: an explicit older version is a deliberate rollback and sticks across polls.
	set(`{"mode":"pinned","version":"v1.0.0"}`, http.StatusOK)
	if v, _ := poll(); v != "v1.0.0" {
		t.Fatalf("pinned rollback: %q", v)
	}
	// Back to latest from a pinned version above the registry: the pinned version is the floor.
	set(`{"mode":"pinned","version":"v3.0.0"}`, http.StatusOK)
	set(`{"mode":"latest","version":null}`, http.StatusOK)
	if v, _ := poll(); v != "v3.0.0" {
		t.Fatalf("floor lost: %q", v)
	}

	// Disabled clears everything; a request without mode keeps working (derived from the version).
	set(`{"mode":"disabled","version":"v1.0.0"}`, http.StatusOK)
	if v, _ := poll(); v != "" {
		t.Fatalf("disabled: %q", v)
	}
	set(`{"version":"v1.1.0"}`, http.StatusOK)
	if mode, _, _ := stamp(); mode != "pinned" {
		t.Fatalf("legacy request mode: %s", mode)
	}
	set(`{"version":null}`, http.StatusOK)
	if mode, version, _ := stamp(); mode != "disabled" || version != nil {
		t.Fatalf("legacy disable: %s %v", mode, version)
	}

	// Latest with nothing known yet and an unreachable registry: no target, no error.
	mu.Lock()
	down = true
	mu.Unlock()
	h.server.workerCatalogBuilt = newCatalog()
	set(`{"mode":"latest","version":null}`, http.StatusOK)
	if v, _ := poll(); v != "" {
		t.Fatalf("no target expected: %q", v)
	}
}
