//go:build integration

package httpapi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sseFrame struct {
	Event string
	ID    string
	Data  string
}

// readFrames parses a text/event-stream into frames on a channel until it ends.
func readFrames(resp *http.Response) <-chan sseFrame {
	out := make(chan sseFrame, 64)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		var f sseFrame
		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == "":
				if f.Event != "" {
					out <- f
				}
				f = sseFrame{}
			case strings.HasPrefix(line, "event: "):
				f.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "id: "):
				f.ID = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "data: "):
				f.Data = strings.TrimPrefix(line, "data: ")
			}
		}
	}()
	return out
}

func nextFrame(t *testing.T, frames <-chan sseFrame) sseFrame {
	t.Helper()
	select {
	case f, ok := <-frames:
		if !ok {
			t.Fatal("stream ended")
		}
		return f
	case <-time.After(5 * time.Second):
		t.Fatal("no event received")
	}
	return sseFrame{}
}

type snapshotFrame struct {
	Seq      int64 `json:"seq"`
	Snapshot struct {
		Run *analysisRunView `json:"run"`
	} `json:"snapshot"`
}

func (h *voiceHarness) openStream() (*http.Response, <-chan sseFrame) {
	h.t.Helper()
	req, _ := http.NewRequest(http.MethodGet, h.ts.URL+h.analysisPath()+"/events", nil)
	resp, err := h.admin.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		h.t.Fatalf("stream: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	return resp, readFrames(resp)
}

func testClipAnalysisEvents(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.api.Events.Poll, h.api.Events.Heartbeat = 50*time.Millisecond, 300*time.Millisecond

	// Anonymous viewers and unknown clips get no stream.
	anon, err := http.Get(h.ts.URL + h.analysisPath() + "/events")
	if err != nil {
		t.Fatal(err)
	}
	anon.Body.Close()
	if anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous stream: %d", anon.StatusCode)
	}
	h.adminCall(http.MethodGet, "/api/admin/clips/99999/analysis/events", nil, http.StatusNotFound)

	// Two viewers (tabs) see the initial snapshot, then the change, in order.
	respA, a := h.openStream()
	defer respA.Body.Close()
	respB, b := h.openStream()
	defer respB.Body.Close()
	var first snapshotFrame
	for _, frames := range []<-chan sseFrame{a, b} {
		f := nextFrame(t, frames)
		if f.Event != "snapshot" {
			t.Fatalf("expected a snapshot, got %+v", f)
		}
		if err := json.Unmarshal([]byte(f.Data), &first); err != nil || first.Snapshot.Run != nil {
			t.Fatalf("a never analyzed clip has no run: %s", f.Data)
		}
	}
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	for _, frames := range []<-chan sseFrame{a, b} {
		var got snapshotFrame
		for {
			f := nextFrame(t, frames)
			if f.Event != "snapshot" {
				continue
			}
			if err := json.Unmarshal([]byte(f.Data), &got); err != nil {
				t.Fatal(err)
			}
			if got.Snapshot.Run != nil && got.Snapshot.Run.step("preparation").Status == "blocked" {
				break
			}
		}
		if got.Seq <= first.Seq {
			t.Fatalf("the sequence must grow: %d then %d", first.Seq, got.Seq)
		}
	}

	// A reconnection starts from a complete snapshot, whatever happened meanwhile.
	respA.Body.Close()
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusOK)
	h.adminCall(http.MethodPost, h.analysisPath(), map[string]any{}, http.StatusOK)
	respC, c := h.openStream()
	defer respC.Body.Close()
	var again snapshotFrame
	if err := json.Unmarshal([]byte(nextFrame(t, c).Data), &again); err != nil || again.Snapshot.Run == nil || again.Snapshot.Run.step("preparation").Status != "done" {
		t.Fatalf("reconnection must give the current state: %+v", again)
	}

	// Losing access (here, the session) ends the stream with a notice.
	if _, err := pool.Exec(t.Context(), `DELETE FROM sessions`); err != nil {
		t.Fatal(err)
	}
	for {
		f, ok := <-c
		if !ok {
			t.Fatal("the stream ended without telling the viewer")
		}
		if f.Event == "forbidden" {
			break
		}
	}
}
