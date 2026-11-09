//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type gotLogPage struct {
	SessionID string `json:"session_id"`
	HasMore   bool   `json:"has_more"`
	Lines     []struct {
		ID   int64  `json:"id"`
		Seq  int64  `json:"seq"`
		Line string `json:"line"`
	} `json:"lines"`
}

type gotControl struct {
	Paused        bool  `json:"paused"`
	AppliedPaused *bool `json:"applied_paused"`
}

func testWorkerControl(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	id := h.fixtureWorkerID()
	logs := "/api/admin/workers/" + id + "/logs"
	control := "/api/admin/workers/" + id + "/control"
	batch := func(session string, from, to int) map[string]any {
		lines := []map[string]any{}
		for i := from; i < to; i++ {
			lines = append(lines, map[string]any{"seq": i, "line": fmt.Sprintf(`{"event":"line_%d"}`, i)})
		}
		return map[string]any{"session_id": session, "lines": lines}
	}
	page := func(query string) gotLogPage {
		var p gotLogPage
		if err := json.Unmarshal(h.adminCall(http.MethodGet, logs+query, nil, http.StatusOK), &p); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// Rights: only an Administrator reads the logs or drives the Worker; nothing changes for a refusal.
	member := h.newUser("worker-control-member")
	h.userCall(member, http.MethodGet, logs, nil, http.StatusForbidden)
	h.userCall(member, http.MethodPost, control, map[string]string{"action": "pause"}, http.StatusForbidden)
	h.adminCall(http.MethodGet, "/api/admin/workers/wk_unknown_worker_identifier_01/logs", nil, http.StatusNotFound)
	h.adminCall(http.MethodPost, control, map[string]string{"action": "format-disk"}, http.StatusBadRequest)
	// Restarting the container is the managed restart, not an action of this endpoint.
	h.adminCall(http.MethodPost, control, map[string]string{"action": "restart"}, http.StatusBadRequest)

	// No log yet: an empty page, not an error.
	if p := page(""); len(p.Lines) != 0 || p.SessionID != "" {
		t.Fatalf("no line was pushed: %+v", p)
	}

	// Validation of what a Worker pushes.
	h.workerCall(http.MethodPost, "/api/worker/logs", batch("bad", 0, 1), "", http.StatusBadRequest)
	h.workerCall(http.MethodPost, "/api/worker/logs", map[string]any{"session_id": "session-aaaa", "lines": []any{}}, "", http.StatusBadRequest)
	h.workerCall(http.MethodPost, "/api/worker/logs", batch("session-aaaa", 0, maxLogBatch+1), "", http.StatusBadRequest)
	h.workerCall(http.MethodPost, "/api/worker/logs", map[string]any{"session_id": "session-aaaa", "lines": []map[string]any{{"seq": 0, "line": string(make([]byte, maxLogLineLength+1))}}}, "", http.StatusBadRequest)

	// A batch is stored once, even when it is resent after a lost answer.
	h.workerCall(http.MethodPost, "/api/worker/logs", batch("session-aaaa", 0, 5), "", http.StatusNoContent)
	h.workerCall(http.MethodPost, "/api/worker/logs", batch("session-aaaa", 0, 5), "", http.StatusNoContent)
	if p := page(""); len(p.Lines) != 5 || p.SessionID != "session-aaaa" || p.HasMore {
		t.Fatalf("a resent batch must not duplicate lines: %+v", p)
	}

	// The viewer follows the live end from a cursor, in order.
	first := page("")
	h.workerCall(http.MethodPost, "/api/worker/logs", batch("session-aaaa", 5, 8), "", http.StatusNoContent)
	follow := page(fmt.Sprintf("?after=%d", first.Lines[len(first.Lines)-1].ID))
	if len(follow.Lines) != 3 || follow.Lines[0].Seq != 5 || follow.Lines[2].Seq != 7 {
		t.Fatalf("following the live end: %+v", follow)
	}

	// Paging back from the newest page reaches the start of the session.
	recent := page("?limit=3")
	if len(recent.Lines) != 3 || !recent.HasMore || recent.Lines[0].Seq != 5 {
		t.Fatalf("newest page: %+v", recent)
	}
	older := page(fmt.Sprintf("?limit=3&before=%d", recent.Lines[0].ID))
	if len(older.Lines) != 3 || older.Lines[0].Seq != 2 || older.Lines[2].Seq != 4 || !older.HasMore {
		t.Fatalf("older page: %+v", older)
	}
	h.adminCall(http.MethodGet, logs+"?limit=0", nil, http.StatusBadRequest)
	h.adminCall(http.MethodGet, logs+"?after=1&before=2", nil, http.StatusBadRequest)

	// A new process of the Worker starts a new session: the viewer shows it, not the previous one.
	h.workerCall(http.MethodPost, "/api/worker/logs", batch("session-bbbb", 0, 2), "", http.StatusNoContent)
	if p := page(""); p.SessionID != "session-bbbb" || len(p.Lines) != 2 {
		t.Fatalf("the current session is the latest one: %+v", p)
	}

	// The window is bounded: the oldest lines go first.
	for from := 2; from < workerLogRetention+300; from += maxLogBatch {
		to := min(from+maxLogBatch, workerLogRetention+300)
		h.workerCall(http.MethodPost, "/api/worker/logs", batch("session-bbbb", from, to), "", http.StatusNoContent)
	}
	var kept int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM worker_log_lines WHERE worker_id=$1`, id).Scan(&kept); err != nil || kept != workerLogRetention {
		t.Fatalf("kept %d lines (err %v), want %d", kept, err, workerLogRetention)
	}

	// A batch of real lines is far over the default 16 KiB request limit and must still be taken: 200 lines of
	// about 1 KiB with quotes to escape. A body beyond the log limit is refused, not truncated.
	long := make([]map[string]any, 0, maxLogBatch)
	for i := 0; i < maxLogBatch; i++ {
		long = append(long, map[string]any{"seq": i, "line": fmt.Sprintf(`{"event":"long_%d","stderr_tail":"%s"}`, i, strings.Repeat(`a"b `, 250))})
	}
	h.workerCall(http.MethodPost, "/api/worker/logs", map[string]any{"session_id": "session-long1", "lines": long}, "", http.StatusNoContent)
	if p := page("?limit=1000"); p.SessionID != "session-long1" || len(p.Lines) != maxLogBatch {
		t.Fatalf("a batch of real lines must be stored: %d lines in %q", len(p.Lines), p.SessionID)
	}
	h.workerCall(http.MethodPost, "/api/worker/logs", map[string]any{"session_id": "session-huge1", "lines": []map[string]any{{"seq": 0, "line": strings.Repeat("x", maxWorkerLogBody)}}}, "", http.StatusBadRequest)

	// Control: nothing asked yet.
	var state gotControl
	if err := json.Unmarshal(h.adminCall(http.MethodGet, control, nil, http.StatusOK), &state); err != nil || state.Paused || state.AppliedPaused != nil {
		t.Fatalf("initial control state: %+v (%v)", state, err)
	}
	var polled struct {
		Paused bool `json:"paused"`
	}
	poll := func() {
		polled.Paused = false
		if err := json.Unmarshal(h.workerCall(http.MethodGet, "/api/worker/control", nil, "", http.StatusOK), &polled); err != nil {
			t.Fatal(err)
		}
	}
	poll()
	if polled.Paused {
		t.Fatalf("a Worker nobody drove must run: %+v", polled)
	}

	// Pause is a state: asked, then confirmed by the Worker.
	if err := json.Unmarshal(h.adminCall(http.MethodPost, control, map[string]string{"action": "pause"}, http.StatusOK), &state); err != nil || !state.Paused || state.AppliedPaused != nil {
		t.Fatalf("pause requested: %+v (%v)", state, err)
	}
	poll()
	if !polled.Paused {
		t.Fatalf("the Worker must read the pause: %+v", polled)
	}
	h.workerCall(http.MethodPost, "/api/worker/control/ack", map[string]any{"paused": true}, "", http.StatusNoContent)
	if err := json.Unmarshal(h.adminCall(http.MethodGet, control, nil, http.StatusOK), &state); err != nil || !state.Paused || state.AppliedPaused == nil || !*state.AppliedPaused {
		t.Fatalf("pause confirmed: %+v (%v)", state, err)
	}
	// Asking twice is the same as asking once.
	h.adminCall(http.MethodPost, control, map[string]string{"action": "pause"}, http.StatusOK)
	h.adminCall(http.MethodPost, control, map[string]string{"action": "resume"}, http.StatusOK)
	poll()
	if polled.Paused {
		t.Fatalf("resume must lift the pause: %+v", polled)
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, control, nil, http.StatusOK), &state); err != nil || state.Paused || state.AppliedPaused == nil || !*state.AppliedPaused {
		t.Fatalf("resume asked, not yet confirmed: %+v (%v)", state, err)
	}
	h.workerCall(http.MethodPost, "/api/worker/control/ack", map[string]any{"paused": false}, "", http.StatusNoContent)
	if err := json.Unmarshal(h.adminCall(http.MethodGet, control, nil, http.StatusOK), &state); err != nil || state.Paused || *state.AppliedPaused {
		t.Fatalf("resume confirmed: %+v (%v)", state, err)
	}
	// A deleted Worker's control row goes with it.
	var rows int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM worker_control WHERE worker_id=$1`, id).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("control rows: %d (%v)", rows, err)
	}

	// A Worker without its token reads nothing and pushes nothing.
	for _, call := range []struct{ method, path string }{{http.MethodGet, "/api/worker/control"}, {http.MethodPost, "/api/worker/logs"}} {
		req, _ := http.NewRequest(call.method, h.ts.URL+call.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s without token: %v %v", call.method, call.path, resp, err)
		}
		resp.Body.Close()
	}
}
