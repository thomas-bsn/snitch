package workerupdate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidVersionAcceptsOnlyReleaseTags(t *testing.T) {
	for _, v := range []string{"v0.1.0", "v1.10.2"} {
		if !ValidVersion(v) {
			t.Fatalf("%s should be valid", v)
		}
	}
	for _, v := range []string{"", "latest", "1.2.3", "v1.2", "v1.2.3-rc1", "v01.2.3", "v1.2.3\n", "v1.2.3@sha256:abc", "2026-11-03"} {
		if ValidVersion(v) {
			t.Fatalf("%q should be refused", v)
		}
	}
}

func TestImageForKeepsRepositoryOnly(t *testing.T) {
	cases := map[string]string{
		"ghcr.io/thomas-bsn/snitch-worker":        "ghcr.io/thomas-bsn/snitch-worker:v1.2.3",
		"ghcr.io/thomas-bsn/snitch-worker:latest": "ghcr.io/thomas-bsn/snitch-worker:v1.2.3",
		"registry.local:5000/snitch-worker":       "registry.local:5000/snitch-worker:v1.2.3",
	}
	for configured, want := range cases {
		got, err := ImageFor(configured, "v1.2.3")
		if err != nil || got != want {
			t.Fatalf("ImageFor(%q) = %q, %v; want %q", configured, got, err, want)
		}
	}
}

func TestImageForRefusesInvalidInputs(t *testing.T) {
	if _, err := ImageFor("ghcr.io/x/worker", "latest"); !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("latest must be refused, got %v", err)
	}
	if _, err := ImageFor("ghcr.io/x/worker@sha256:abc", "v1.0.0"); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("digest reference must be refused, got %v", err)
	}
}

func TestLessOrdersSemver(t *testing.T) {
	if !Less("v1.9.0", "v1.10.0") || Less("v1.10.0", "v1.9.0") || Less("v2.0.0", "v2.0.0") {
		t.Fatal("semver ordering is wrong: 1.10.0 must be newer than 1.9.0")
	}
}

// fakeRegistry serves a token and a tag list over TLS, like ghcr.io does for public packages.
func fakeRegistry(t *testing.T, tags []string, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("scope") != "repository:thomas-bsn/snitch-worker:pull" {
			http.Error(w, "bad scope", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "anon"})
	})
	mux.HandleFunc("/v2/thomas-bsn/snitch-worker/tags/list", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer anon" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "thomas-bsn/snitch-worker", "tags": tags})
	})
	ts := httptest.NewTLSServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func TestCatalogListsReleasesNewestFirstAndSkipsOtherTags(t *testing.T) {
	var calls atomic.Int32
	ts := fakeRegistry(t, []string{"latest", "v1.2.0", "v1.10.0", "sha-abc", "v1.9.9", "v2.0.0-rc1"}, &calls)
	c := &Catalog{Client: ts.Client(), Registry: ts.URL, Repo: "thomas-bsn/snitch-worker"}
	got, err := c.Versions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"v1.10.0", "v1.9.9", "v1.2.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCatalogCachesWithinTTLAndKeepsLastGoodAnswerOnFailure(t *testing.T) {
	var calls atomic.Int32
	ts := fakeRegistry(t, []string{"v1.0.0"}, &calls)
	now := time.Now()
	c := &Catalog{Client: ts.Client(), Registry: ts.URL, Repo: "thomas-bsn/snitch-worker", TTL: time.Minute, Now: func() time.Time { return now }}
	if _, err := c.Versions(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := calls.Load()
	if _, err := c.Versions(context.Background()); err != nil || calls.Load() != first {
		t.Fatalf("second call inside TTL must use the cache (calls %d -> %d, err %v)", first, calls.Load(), err)
	}
	// Registry becomes unreachable after the TTL: the last good list is still served.
	now = now.Add(2 * time.Minute)
	ts.Close()
	got, err := c.Versions(context.Background())
	if err != nil || !reflect.DeepEqual(got, []string{"v1.0.0"}) {
		t.Fatalf("expected last good list, got %v, %v", got, err)
	}
}

func TestCatalogWithoutCacheReportsUnavailable(t *testing.T) {
	var calls atomic.Int32
	ts := fakeRegistry(t, nil, &calls)
	ts.Close()
	c := &Catalog{Client: ts.Client(), Registry: ts.URL, Repo: "thomas-bsn/snitch-worker"}
	got, err := c.Versions(context.Background())
	if !errors.Is(err, ErrCatalog) || len(got) != 0 {
		t.Fatalf("want ErrCatalog and empty list, got %v, %v", got, err)
	}
}

func TestCatalogRefusesPlainHTTPRegistry(t *testing.T) {
	c := &Catalog{Registry: "http://ghcr.io", Repo: "thomas-bsn/snitch-worker"}
	if _, err := c.Versions(context.Background()); !errors.Is(err, ErrCatalog) || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("plain HTTP registry must be refused, got %v", err)
	}
}

func TestNewestIgnoresRegistryOrderAndNonReleases(t *testing.T) {
	if got := Newest([]string{"v1.9.0", "v1.10.0", "v1.2.0", "latest", "sha-abc", "v2.0.0-rc1"}); got != "v1.10.0" {
		t.Fatalf("got %q", got)
	}
	if got := Newest(nil); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestCatalogFollowsPaginationOnTheSameHostOnly(t *testing.T) {
	var host string
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "anon"})
	})
	mux.HandleFunc("/v2/thomas-bsn/snitch-worker/tags/list", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("last") == "" {
			w.Header().Set("Link", `</v2/thomas-bsn/snitch-worker/tags/list?n=1000&last=sha-1>; rel="next"`)
			_ = json.NewEncoder(w).Encode(map[string]any{"tags": []string{"v1.0.0", "sha-1"}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"tags": []string{"v1.1.0"}})
	})
	ts := httptest.NewTLSServer(mux)
	defer ts.Close()
	host = ts.URL
	c := &Catalog{Client: ts.Client(), Registry: host, Repo: "thomas-bsn/snitch-worker"}
	got, err := c.Versions(context.Background())
	if err != nil || !reflect.DeepEqual(got, []string{"v1.1.0", "v1.0.0"}) {
		t.Fatalf("second page lost: %v, %v", got, err)
	}

	// A link towards another host would leak the bearer token: the whole list is refused.
	evil := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "anon"})
			return
		}
		w.Header().Set("Link", `<https://evil.example/steal>; rel="next"`)
		_ = json.NewEncoder(w).Encode(map[string]any{"tags": []string{"v9.9.9"}})
	}))
	defer evil.Close()
	c = &Catalog{Client: evil.Client(), Registry: evil.URL, Repo: "thomas-bsn/snitch-worker"}
	if got, err = c.Versions(context.Background()); !errors.Is(err, ErrCatalog) || len(got) != 0 {
		t.Fatalf("foreign pagination link must fail: %v, %v", got, err)
	}
}

func TestCatalogBacksOffAfterFailureAndReportsStatus(t *testing.T) {
	var calls atomic.Int32
	ts := fakeRegistry(t, []string{"v1.0.0"}, &calls)
	now := time.Now()
	c := &Catalog{Client: ts.Client(), Registry: ts.URL, Repo: "thomas-bsn/snitch-worker", TTL: time.Minute, Now: func() time.Time { return now }}
	if at, failed := c.Status(); !at.IsZero() || failed {
		t.Fatal("no attempt yet")
	}
	if _, err := c.Versions(context.Background()); err != nil {
		t.Fatal(err)
	}
	if at, failed := c.Status(); at.IsZero() || failed {
		t.Fatalf("successful check expected, failed=%v", failed)
	}
	now = now.Add(2 * time.Minute)
	ts.Close()
	if got, err := c.Versions(context.Background()); err != nil || len(got) != 1 {
		t.Fatalf("last good list expected: %v, %v", got, err)
	}
	if _, failed := c.Status(); !failed {
		t.Fatal("the failure must be visible")
	}
	// While backing off, no new registry attempt is made even though the TTL has long expired.
	tried, _ := c.Status()
	now = now.Add(10 * time.Second)
	_, _ = c.Versions(context.Background())
	if again, _ := c.Status(); !again.Equal(tried) {
		t.Fatal("a Worker poll must not hammer an unreachable registry")
	}
}
