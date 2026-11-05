//go:build integration

package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// diarizationOf is the diarization job a fixture speaker belongs to.
func (h *voiceHarness) diarizationOf(sp fixtureSpeaker) int64 {
	h.t.Helper()
	var id int64
	if err := h.pool.QueryRow(context.Background(), `SELECT diarization_job_id FROM local_speakers WHERE id=$1`, sp.ID).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

// completeNext leases the next job and completes it with a fingerprint of the model.
func (h *voiceHarness) completeNext(sp fixtureSpeaker, label, model string, degrees float64) {
	h.t.Helper()
	job, ok := h.claim()
	if !ok || job.Step != "voice.fingerprint.9" {
		h.t.Fatalf("a fingerprint job must be leased, got %+v", job)
	}
	h.complete(job, benchmarkResult(h.diarizationOf(sp), label, model, degrees), http.StatusOK)
}

func (h *voiceHarness) activeFingerprintModel() string {
	h.t.Helper()
	var id string
	if err := h.pool.QueryRow(context.Background(), `SELECT model_id FROM voice_active_model WHERE kind='fingerprint'`).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

func testVoiceFingerprintTransitionFlow(t *testing.T, pool *pgxpool.Pool) {
	lib := newTransitionLibrary(t, pool)
	h := lib.h
	h.rematch()
	if got := h.matched(lib.other); got.Person == nil || got.Person.ID != lib.alice {
		t.Fatalf("before the change the close voice is identified by the current model: %+v", got)
	}

	// The administrator confirms the number he saw, or nothing starts.
	h.startTransition(benchModelID, 5, http.StatusConflict)
	if n := h.count(`SELECT count(*) FROM voice_fingerprint_transitions`); n != 0 {
		t.Fatalf("a stale confirmation starts nothing, got %d", n)
	}
	h.startTransition(benchModelID, 1, http.StatusCreated)
	h.startTransition(benchModelID, 1, http.StatusConflict)
	st := h.transitionState()
	if !st.Open || st.Transition == nil || st.Transition.Phase != "preparing" || st.Transition.AnnouncedReferences != 1 || st.Transition.ReferencesTotal != 1 || st.Transition.ReferencesRemaining != 1 {
		t.Fatalf("the transition is observable from its start: %+v", st.Transition)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE requested_model_id IS NOT NULL`); n != 0 {
		t.Fatalf("nothing is recomputed before a Worker holds the target ready, got %d jobs", n)
	}

	// The source and the target stay installed during the whole transition.
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/"+fingerprintID+"/installation", nil, http.StatusConflict)
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/"+benchModelID+"/installation", nil, http.StatusConflict)

	// The target is ready on a Worker: the references are recomputed with it, in the staging space.
	h.reportModel(benchModelID, "ready", "", 1, 1, http.StatusNoContent)
	if n := h.count(`SELECT count(*) FROM jobs WHERE requested_model_id=$1 AND status='pending'`, benchModelID); n != 1 {
		t.Fatalf("one recomputation per reference expected, got %d", n)
	}
	st = h.transitionState()
	if st.Transition.Phase != "recomputing" || st.Transition.JobsActive != 1 || st.Transition.ReferencesDone != 0 {
		t.Fatalf("recomputing is visible: %+v", st.Transition)
	}
	if h.activeFingerprintModel() != fingerprintID {
		t.Fatal("the production keeps the current model until the essential migration is complete")
	}
	h.completeNext(lib.reference, "REF", benchModelRepo, 0)

	// The switch: the target is active, its fingerprints are the production space, the previous ones are kept aside.
	if h.activeFingerprintModel() != benchModelID {
		t.Fatal("the target must become the active model once every reference is recomputed")
	}
	st = h.transitionState()
	if st.Open || st.Transition.Phase != "switched" || st.Transition.CatchUpRemaining != 1 {
		t.Fatalf("the transition is switched with one voice left to catch up: %+v", st.Transition)
	}
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE NOT benchmark AND model=$1`, benchModelRepo); n != 1 {
		t.Fatalf("the recomputed reference enters the production space, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM voice_fingerprints WHERE benchmark AND model=$1`, activeEmbedding); n != 2 {
		t.Fatalf("the previous fingerprints are kept outside the production space, got %d", n)
	}
	// No silent comparison: the close voice only has an old-model fingerprint, so it is not identified any more.
	if got := h.matched(lib.other); got.Person != nil {
		t.Fatalf("a voice must never be compared across models: %+v", got)
	}
	if h.personOf(lib.reference) != lib.alice {
		t.Fatal("the validated attribution (human decision) is untouched")
	}

	// The voices left are computed with the new model and identified again as the results arrive.
	if n := h.count(`SELECT count(*) FROM jobs WHERE requested_model_id IS NULL AND step='voice.fingerprint.9' AND status='pending'`); n != 1 {
		t.Fatalf("one catch-up job per remaining voice, got %d", n)
	}
	h.completeNext(lib.other, "OTHER", benchModelRepo, 10)
	if got := h.matched(lib.other); got.Person == nil || got.Person.ID != lib.alice {
		t.Fatalf("the voice is identified in the new model: %+v", got)
	}
	if st = h.transitionState(); st.Transition.CatchUpRemaining != 0 {
		t.Fatalf("the catch-up is complete: %+v", st.Transition)
	}
	h.adminCall(http.MethodDelete, transitionPath, nil, http.StatusConflict)
}
