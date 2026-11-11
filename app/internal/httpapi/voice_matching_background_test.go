//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type matchingStatusView struct {
	Pending        bool       `json:"pending"`
	Running        bool       `json:"running"`
	LastFinishedAt *time.Time `json:"last_finished_at"`
	LastDurationMS *int64     `json:"last_duration_ms"`
	LastRequests   *int       `json:"last_requests"`
	LastFailedAt   *time.Time `json:"last_failed_at"`
}

func (h *voiceHarness) matchingStatus() matchingStatusView {
	h.t.Helper()
	var v matchingStatusView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/voice-matching/status", nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

// waitMatching waits until no matching request is left and no pass runs, as the interface sees it: the
// helper of the suites that run matching in the background (newVoiceHarnessBackground).
func (h *voiceHarness) waitMatching() matchingStatusView {
	h.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		v := h.matchingStatus()
		if !v.Pending && !v.Running {
			return v
		}
		if time.Now().After(deadline) {
			h.t.Fatalf("voice matching still in progress: %+v", v)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// postQuietly sends an administrator request without failing the test, so it can run in a goroutine.
func (h *voiceHarness) postQuietly(path string, body any) (int, string, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return 0, "", err
	}
	req, _ := http.NewRequest(http.MethodPost, h.ts.URL+path, bytes.NewReader(payload))
	req.Header.Set("Origin", h.ts.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", h.csrf)
	resp, err := h.admin.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(data), nil
}

// validateWithin validates a voice and fails when the answer takes longer than limit.
func (h *voiceHarness) validateWithin(sp fixtureSpeaker, name string, limit time.Duration) {
	h.t.Helper()
	type answer struct {
		code int
		body string
		err  error
	}
	done := make(chan answer, 1)
	go func() {
		code, body, err := h.postQuietly(fmt.Sprintf("/api/local-speakers/%d/validation", sp.ID), map[string]any{"new_person_name": name})
		done <- answer{code, body, err}
	}()
	select {
	case a := <-done:
		if a.err != nil || a.code != http.StatusCreated {
			h.t.Fatalf("validation of %d: %d %s %v", sp.ID, a.code, a.body, a.err)
		}
	case <-time.After(limit):
		h.t.Fatalf("the validation of %d waited for the matching pass", sp.ID)
	}
}

func (h *voiceHarness) expectSuggestion(sp fixtureSpeaker, person string) {
	h.t.Helper()
	m := h.matched(sp)
	if m.DecisionLevel != string(LevelSuggestion) || m.Person == nil || m.Person.DisplayName != person {
		h.t.Fatalf("speaker %d: want a suggestion of %s, got %+v", sp.ID, person, m)
	}
}

func (h *voiceHarness) expectUnmatched(sp fixtureSpeaker) {
	h.t.Helper()
	if m := h.matched(sp); m.Person != nil || m.DecisionLevel != string(LevelUnknown) {
		h.t.Fatalf("speaker %d matched before the pass: %+v", sp.ID, m)
	}
}

// testVoiceMatchingBackground covers #392: a validation answers as soon as it is recorded, matching follows in
// the background, close decisions make one pass reading the latest state, and an interrupted or failed pass is
// resumed without any user action. Holding the matching lock stands for a long pass in progress.
func testVoiceMatchingBackground(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessBackground(t, pool, 3)
	h.api.Events.Poll, h.api.Events.Heartbeat = 50*time.Millisecond, 300*time.Millisecond
	h.putEligibility(1, 0.5, 0.05)
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	refAlice := h.fixtureSpeaker(clipA, "REF_ALICE", usableAt(0))
	nearAlice := h.fixtureSpeaker(clipB, "NEAR_ALICE", usableAt(10))
	refBob := h.fixtureSpeaker(clipA, "REF_BOB", usableAt(90))
	nearBob := h.fixtureSpeaker(clipC, "NEAR_BOB", usableAt(100))
	refCarol := h.fixtureSpeaker(clipB, "REF_CAROL", usableAt(200))
	nearCarol := h.fixtureSpeaker(clipC, "NEAR_CAROL", usableAt(190))
	before := h.waitMatching()

	// The interface follows the state through a live stream as well.
	req, _ := http.NewRequest(http.MethodGet, h.ts.URL+"/api/voice-matching/status/events", nil)
	stream, err := h.admin.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Body.Close()
	if stream.StatusCode != http.StatusOK || !strings.HasPrefix(stream.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status stream: %d", stream.StatusCode)
	}
	frames := readFrames(stream)
	untilStatus := func(ok func(matchingStatusView) bool) matchingStatusView {
		t.Helper()
		for {
			f := nextFrame(t, frames)
			if f.Event != "snapshot" {
				continue
			}
			var got struct {
				Snapshot matchingStatusView `json:"snapshot"`
			}
			if err := json.Unmarshal([]byte(f.Data), &got); err != nil {
				t.Fatal(err)
			}
			if ok(got.Snapshot) {
				return got.Snapshot
			}
		}
	}
	untilStatus(func(v matchingStatusView) bool { return !v.Pending && !v.Running })

	// A validation answers while a long pass holds the matching: it no longer depends on its duration.
	voiceMatchingLock.Lock()
	h.validateWithin(refAlice, "Alice", 3*time.Second)
	if v := h.matchingStatus(); !v.Pending {
		voiceMatchingLock.Unlock()
		t.Fatalf("the validation left no pending matching: %+v", v)
	}
	untilStatus(func(v matchingStatusView) bool { return v.Pending })
	h.expectUnmatched(nearAlice)
	voiceMatchingLock.Unlock()
	after := h.waitMatching()
	h.expectSuggestion(nearAlice, "Alice")
	if after.LastFinishedAt == nil || (before.LastFinishedAt != nil && !after.LastFinishedAt.After(*before.LastFinishedAt)) || after.LastFailedAt != nil || after.LastDurationMS == nil {
		t.Fatalf("finished pass not reported: before %+v after %+v", before, after)
	}
	untilStatus(func(v matchingStatusView) bool { return !v.Pending && !v.Running && v.LastFinishedAt != nil })

	// Two simultaneous validations lose nothing, and the requests made during a pass make a single pass.
	voiceMatchingLock.Lock()
	var wg sync.WaitGroup
	failures := make(chan string, 2)
	for sp, name := range map[fixtureSpeaker]string{refBob: "Bob", refCarol: "Carol"} {
		wg.Add(1)
		go func(sp fixtureSpeaker, name string) {
			defer wg.Done()
			code, body, err := h.postQuietly(fmt.Sprintf("/api/local-speakers/%d/validation", sp.ID), map[string]any{"new_person_name": name})
			if err != nil || code != http.StatusCreated {
				failures <- fmt.Sprintf("%s: %d %s %v", name, code, body, err)
			}
		}(sp, name)
	}
	wg.Wait()
	close(failures)
	voiceMatchingLock.Unlock()
	for f := range failures {
		t.Fatal(f)
	}
	coalesced := h.waitMatching()
	if coalesced.LastRequests == nil || *coalesced.LastRequests != 2 {
		t.Fatalf("two close validations should make one pass: %+v", coalesced)
	}
	if h.matched(refBob).Person.DisplayName != "Bob" || h.matched(refCarol).Person.DisplayName != "Carol" {
		t.Fatal("a simultaneous validation was lost")
	}
	h.expectSuggestion(nearBob, "Bob")
	h.expectSuggestion(nearCarol, "Carol")
	h.expectSuggestion(nearAlice, "Alice")

	// A failed pass is retried by itself: its requests stay until a pass succeeds.
	refDana := h.fixtureSpeaker(clipA, "REF_DANA", usableAt(300))
	nearDana := h.fixtureSpeaker(clipB, "NEAR_DANA", usableAt(310))
	if _, err = pool.Exec(ctx, `CREATE FUNCTION snitch_test_refuse_candidates() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic matching failure'; END $$;
CREATE TRIGGER snitch_test_refuse_candidates BEFORE INSERT ON speaker_attribution_candidates FOR EACH ROW EXECUTE FUNCTION snitch_test_refuse_candidates()`); err != nil {
		t.Fatal(err)
	}
	dropped := false
	drop := func() {
		if !dropped {
			dropped = true
			if _, e := pool.Exec(ctx, `DROP TRIGGER snitch_test_refuse_candidates ON speaker_attribution_candidates; DROP FUNCTION snitch_test_refuse_candidates()`); e != nil {
				t.Fatal(e)
			}
		}
	}
	defer drop()
	h.validateWithin(refDana, "Dana", 3*time.Second)
	failed := untilStatus(func(v matchingStatusView) bool { return v.LastFailedAt != nil })
	if !failed.Pending {
		t.Fatalf("a failed pass dropped its requests: %+v", failed)
	}
	h.expectUnmatched(nearDana)
	drop()
	retried := h.waitMatching()
	if retried.LastFailedAt != nil {
		t.Fatalf("the retried pass still reports a failure: %+v", retried)
	}
	h.expectSuggestion(nearDana, "Dana")

	// A pass cut by a stop leaves its requests; the next start resumes them without any user action.
	refEve := h.fixtureSpeaker(clipA, "REF_EVE", usableAt(45))
	nearEve := h.fixtureSpeaker(clipC, "NEAR_EVE", usableAt(50))
	voiceMatchingLock.Lock()
	h.validateWithin(refEve, "Eve", 3*time.Second)
	h.api.backgroundCancel() // the server stops while the pass waits
	voiceMatchingLock.Unlock()
	h.api.backgroundWG.Wait()
	if v := h.matchingStatus(); !v.Pending || v.Running {
		t.Fatalf("a stopped pass should leave its requests: %+v", v)
	}
	h.expectUnmatched(nearEve)
	restarted := New(pool, h.api.Config, h.api.Log)
	restarted.matchingRetry = time.Hour // only the start resumes the requests
	if err = restarted.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	h.waitMatching()
	m := h.matched(nearEve)
	if m.Person == nil || m.Person.DisplayName != "Eve" {
		t.Fatalf("the restarted server did not resume matching: %+v", m)
	}
}
