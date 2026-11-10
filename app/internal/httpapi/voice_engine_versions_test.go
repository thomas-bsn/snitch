package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestLatestStableKeepsTenNewestFinalReleases(t *testing.T) {
	file, yanked := []pypiFile{{}}, []pypiFile{{Yanked: true}}
	releases := map[string][]pypiFile{
		"3.1.0rc1": file, "3.1.0.dev1": file, "3.0.0.post1": file, // not final releases
		"9.0.0": yanked, "9.1.0": nil, // withdrawn or without any file
	}
	for _, v := range []string{"3.0.0", "2.10.0", "2.9.0", "2.7.3", "2.7.2", "2.7.1", "2.7.0", "2.6.2", "2.6.1", "2.6.0", "2.5.3", "1.0.0"} {
		releases[v] = file
	}
	got := latestStable(releases)
	want := []string{"3.0.0", "2.10.0", "2.9.0", "2.7.3", "2.7.2", "2.7.1", "2.7.0", "2.6.2", "2.6.1", "2.6.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPyPIIndexCachesAndSurvivesAnOutage(t *testing.T) {
	var calls atomic.Int32
	healthy := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !healthy {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		if r.URL.Path != "/pypi/nemo-toolkit/json" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"releases":{"3.0.0":[{"yanked":false}],"2.7.3":[{"yanked":false}]}}`))
	}))
	defer server.Close()
	index := &pypiIndex{base: server.URL, timeout: time.Second, cache: map[string]pypiEntry{}}

	if got := index.recent(context.Background(), "nemo-toolkit"); !reflect.DeepEqual(got, []string{"3.0.0", "2.7.3"}) {
		t.Fatalf("got %v", got)
	}
	index.recent(context.Background(), "nemo-toolkit")
	if calls.Load() != 1 {
		t.Fatalf("a fresh list must be served from the cache, got %d calls", calls.Load())
	}
	// The index goes down once the entry is stale: the last known list is kept, never emptied.
	healthy = false
	index.mu.Lock()
	entry := index.cache["nemo-toolkit"]
	entry.until = time.Now().Add(-time.Second)
	index.cache["nemo-toolkit"] = entry
	index.mu.Unlock()
	if got := index.recent(context.Background(), "nemo-toolkit"); !reflect.DeepEqual(got, []string{"3.0.0", "2.7.3"}) {
		t.Fatalf("an outage must keep the last known list, got %v", got)
	}
	if got := index.recent(context.Background(), "speechbrain"); got != nil {
		t.Fatalf("an unknown outage with no history must give no suggestion, got %v", got)
	}
}
