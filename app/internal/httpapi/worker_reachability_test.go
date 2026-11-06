//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testWorkerReachabilityReporting: a Worker's last transcription-server result travels in its pulse and is shown
// to administrators with the last failure; malformed results are refused; only administrators read the detail.
func testWorkerReachabilityReporting(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.adminCall(http.MethodPut, "/api/admin/transcription-settings", map[string]any{"mode": "enabled", "endpoint": "http://127.0.0.1:9000", "model": "fixture-model", "language": "fr"}, http.StatusOK)
	now := time.Now().UTC().Format(time.RFC3339)
	failure := map[string]any{
		"failed_at":    now,
		"method":       "GET",
		"url":          "http://127.0.0.1:9000/v1/models",
		"status_code":  503,
		"latency_ms":   12,
		"error_type":   "http_status",
		"body_excerpt": "busy",
	}
	pulse := func(reachability any) {
		t.Helper()
		body := map[string]any{"capacity": 1, "supported_steps": []string{"transcription.audio.*"}}
		if reachability != nil {
			body["transcription_reachability"] = reachability
		}
		h.workerCall(http.MethodPost, "/api/worker/pulse", body, "", http.StatusOK)
	}
	reachabilityOf := func() map[string]any {
		t.Helper()
		var state struct {
			Items []struct {
				Reachability map[string]any `json:"transcription_reachability"`
			} `json:"items"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/workers", nil, http.StatusOK), &state); err != nil {
			t.Fatal(err)
		}
		if len(state.Items) != 1 {
			t.Fatalf("expected one worker, got %d", len(state.Items))
		}
		return state.Items[0].Reachability
	}

	if got := reachabilityOf(); got != nil {
		t.Fatalf("a Worker that never reported a test shows a reachability: %v", got)
	}
	pulse(map[string]any{"checked_at": now, "reachable": false, "latency_ms": 12, "last_failure": failure})
	got := reachabilityOf()
	if got == nil || got["reachable"] != false {
		t.Fatalf("failed test not reported as unreachable: %v", got)
	}
	last, _ := got["last_failure"].(map[string]any)
	if last["url"] != "http://127.0.0.1:9000/v1/models" || last["error_type"] != "http_status" || last["method"] != "GET" || last["status_code"] != float64(503) || last["body_excerpt"] != "busy" {
		t.Fatalf("last failure detail lost: %v", last)
	}

	// A later success keeps the last failure only while the Worker still sends it.
	pulse(map[string]any{"checked_at": now, "reachable": true, "latency_ms": 3})
	got = reachabilityOf()
	if got["reachable"] != true || got["last_failure"] != nil {
		t.Fatalf("success did not replace the state: %v", got)
	}

	// Results outside the contract are refused and nothing is stored.
	bad := []map[string]any{
		{"checked_at": now, "reachable": false, "latency_ms": 1, "last_failure": withField(failure, "error_type", "bogus")},
		{"checked_at": now, "reachable": false, "latency_ms": 1, "last_failure": withField(failure, "url", strings.Repeat("u", 2049))},
		{"checked_at": now, "reachable": false, "latency_ms": 1, "last_failure": withField(failure, "body_excerpt", strings.Repeat("x", 201))},
		{"checked_at": now, "reachable": false, "latency_ms": -1},
	}
	for _, payload := range bad {
		h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"transcription.audio.*"}, "transcription_reachability": payload}, "", http.StatusBadRequest)
	}
	if got = reachabilityOf(); got["reachable"] != true {
		t.Fatalf("refused result was stored: %v", got)
	}

	// The detail is an administrator view: a Worker token cannot read it.
	req, _ := http.NewRequest(http.MethodGet, h.ts.URL+"/api/admin/workers", nil)
	req.Header.Set("Authorization", "Bearer "+h.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("a Worker token read the administrator worker view")
	}

	// A pulse without a result clears the test: the Worker no longer has a target.
	pulse(nil)
	if got = reachabilityOf(); got != nil {
		t.Fatalf("absent result left a stale reachability: %v", got)
	}
}

func withField(source map[string]any, key string, value any) map[string]any {
	copied := map[string]any{}
	for k, v := range source {
		copied[k] = v
	}
	copied[key] = value
	return copied
}
