//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (h *voiceHarness) resumeTransition() transitionState {
	h.t.Helper()
	var out transitionState
	body := h.adminCall(http.MethodPost, transitionPath+"/resume", nil, http.StatusOK)
	if err := json.Unmarshal(body, &out); err != nil {
		h.t.Fatal(err)
	}
	return out
}

// An interruption anywhere leaves the transition resumable: the state is the data, not a counter.
func testVoiceFingerprintTransitionResume(t *testing.T, pool *pgxpool.Pool) {
	lib := newTransitionLibrary(t, pool)
	h := lib.h
	ctx := context.Background()
	h.startTransition(benchModelID, 1, http.StatusCreated)
	h.reportModel(benchModelID, "ready", "", 1, 1, http.StatusNoContent)
	if n := h.count(`SELECT count(*) FROM jobs WHERE requested_model_id=$1 AND status='pending'`, benchModelID); n != 1 {
		t.Fatalf("one recomputation expected, got %d", n)
	}

	// The job is lost (queue purged, instance restarted): reading the state schedules it again.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='cancelled' WHERE requested_model_id=$1`, benchModelID); err != nil {
		t.Fatal(err)
	}
	st := h.transitionState()
	if st.Transition.JobsActive != 1 || st.Transition.Phase != "recomputing" {
		t.Fatalf("an interrupted transition resumes by itself: %+v", st.Transition)
	}

	// A job that died is not retried in a loop: it is shown as failed until the administrator resumes.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='dead' WHERE requested_model_id=$1 AND status='pending'`, benchModelID); err != nil {
		t.Fatal(err)
	}
	st = h.transitionState()
	if st.Transition.JobsFailed != 1 || st.Transition.JobsActive != 0 || st.Transition.ReferencesRemaining != 1 {
		t.Fatalf("a failed recomputation is observable and not retried alone: %+v", st.Transition)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE requested_model_id=$1 AND status='pending'`, benchModelID); n != 0 {
		t.Fatalf("no automatic retry loop, got %d pending", n)
	}
	st = h.resumeTransition()
	if st.Transition.JobsActive != 1 || st.Transition.JobsFailed != 0 {
		t.Fatalf("resuming schedules the failed reference again: %+v", st.Transition)
	}

	// The remaining work is done after the interruption: the switch happens as if nothing had occurred.
	h.completeNext(lib.reference, "REF", benchModelRepo, 0)
	if h.activeFingerprintModel() != benchModelID {
		t.Fatal("a resumed transition ends with the switch")
	}
}

// Cancelling before the switch leaves the production exactly as it was, and a new transition can start.
func testVoiceFingerprintTransitionCancel(t *testing.T, pool *pgxpool.Pool) {
	lib := newTransitionLibrary(t, pool)
	h := lib.h
	h.adminCall(http.MethodDelete, transitionPath, nil, http.StatusNotFound)
	h.startTransition(benchModelID, 1, http.StatusCreated)
	h.adminCall(http.MethodDelete, transitionPath, nil, http.StatusNoContent)
	st := h.transitionState()
	if st.Open || st.Transition.Phase != "cancelled" || h.activeFingerprintModel() != fingerprintID {
		t.Fatalf("a cancelled transition changes nothing: %+v", st.Transition)
	}
	// The models are free again, and the administrator can start over.
	h.startTransition(benchModelID, 1, http.StatusCreated)
	// A model that was staged for nothing never feeds identification.
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE NOT benchmark AND model=$1`, benchModelRepo); n != 0 {
		t.Fatalf("no target fingerprint in the production space before the switch, got %d", n)
	}
}
