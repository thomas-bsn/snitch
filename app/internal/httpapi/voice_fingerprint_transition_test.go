//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Changing the fingerprint model is a transition (ADR 0044): preview, confirmation, recomputation of the
// references in a staging space, then an atomic switch. These tests drive it through the administrator API.

type transitionPreview struct {
	FromModelID       string `json:"from_model_id"`
	ToModelID         string `json:"to_model_id"`
	References        int    `json:"references"`
	Clips             int    `json:"clips"`
	Persons           int    `json:"persons"`
	OtherFingerprints int    `json:"other_fingerprints"`
	WorkersReady      int    `json:"workers_ready"`
	ValidatedKept     int    `json:"validated_attributions"`
}

type transitionState struct {
	Open       bool `json:"open"`
	Transition *struct {
		ID                  int64  `json:"id"`
		FromModelID         string `json:"from_model_id"`
		ToModelID           string `json:"to_model_id"`
		Phase               string `json:"phase"`
		AnnouncedReferences int    `json:"announced_references"`
		ReferencesTotal     int    `json:"references_total"`
		ReferencesDone      int    `json:"references_done"`
		ReferencesRemaining int    `json:"references_remaining"`
		JobsActive          int    `json:"jobs_active"`
		JobsFailed          int    `json:"jobs_failed"`
		CatchUpRemaining    int    `json:"catch_up_remaining"`
		WorkersReady        int    `json:"workers_ready"`
	} `json:"transition"`
}

const transitionPath = "/api/admin/voice-models/fingerprint-transition"

func (h *voiceHarness) transitionPreview(to string, want int) transitionPreview {
	h.t.Helper()
	var out transitionPreview
	body := h.adminCall(http.MethodGet, transitionPath+"/preview?to="+url.QueryEscape(to), nil, want)
	if want == http.StatusOK {
		if err := json.Unmarshal(body, &out); err != nil {
			h.t.Fatal(err)
		}
	}
	return out
}

func (h *voiceHarness) transitionState() transitionState {
	h.t.Helper()
	var out transitionState
	if err := json.Unmarshal(h.adminCall(http.MethodGet, transitionPath, nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out
}

func (h *voiceHarness) startTransition(to string, references, want int) []byte {
	h.t.Helper()
	return h.adminCall(http.MethodPost, transitionPath, map[string]any{"to_model_id": to, "confirm_references": references}, want)
}

// transitionLibrary is a synthetic library: Alice's reference voice (clip 1) and an unknown voice close to it
// (clip 2), both in the production space of the active fingerprint model.
type transitionLibrary struct {
	h                *voiceHarness
	reference, other fixtureSpeaker
	alice            int64
}

func newTransitionLibrary(t *testing.T, pool *pgxpool.Pool) *transitionLibrary {
	h := newVoiceHarnessClips(t, pool, 2)
	h.addBenchmarkCatalogModel()
	h.putPipeline(fakePipeline(2))
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=true,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	inActive := func(degrees float64) fixtureFingerprint {
		fp := usableAt(degrees)
		fp.model, fp.version = activeEmbedding, "837717ddb9ff"
		return fp
	}
	// The group indexes of these models are not left behind for the tests that count them.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DO $$ DECLARE r record; BEGIN FOR r IN SELECT indexname FROM pg_indexes WHERE indexname LIKE 'voice_fingerprints_hnsw_%' AND (indexdef LIKE '%fixture/bench-embedding%' OR indexdef LIKE '%wespeaker%') LOOP EXECUTE 'DROP INDEX ' || quote_ident(r.indexname); END LOOP; END $$`)
	})
	lib := &transitionLibrary{h: h}
	lib.reference = h.fixtureSpeaker(h.clipIDs[0], "REF", inActive(0))
	lib.other = h.fixtureSpeaker(h.clipIDs[1], "OTHER", inActive(10))
	// The fingerprint jobs carry the version a real one has, so that a recomputation can be leased and completed.
	for _, sp := range []fixtureSpeaker{lib.reference, lib.other} {
		var diarization int64
		if err := pool.QueryRow(context.Background(), `SELECT diarization_job_id FROM local_speakers WHERE id=$1`, sp.ID).Scan(&diarization); err != nil {
			t.Fatal(err)
		}
		version := fmt.Sprintf("f5-a9-p%s-d%d", h.pipelineVersion(), diarization)
		if _, err := pool.Exec(context.Background(), `UPDATE jobs SET input_version=$2 WHERE id=(SELECT fingerprint_job_id FROM voice_fingerprints WHERE local_speaker_id=$1)`, sp.ID, version); err != nil {
			t.Fatal(err)
		}
	}
	h.validate(lib.reference, "Alice")
	lib.alice = h.personOf(lib.reference)
	return lib
}

func testVoiceFingerprintTransitionPreview(t *testing.T, pool *pgxpool.Pool) {
	lib := newTransitionLibrary(t, pool)
	h := lib.h

	// The impact is known before anything is confirmed: references to recompute, and what is kept.
	p := h.transitionPreview(benchModelID, http.StatusOK)
	if p.FromModelID != fingerprintID || p.ToModelID != benchModelID || p.References != 1 || p.Clips != 1 || p.Persons != 1 || p.OtherFingerprints != 1 || p.WorkersReady != 0 {
		t.Fatalf("the preview must count the references, clips, Persons and other fingerprints: %+v", p)
	}
	if n := h.count(`SELECT count(*) FROM voice_fingerprint_transitions`); n != 0 {
		t.Fatalf("a preview changes nothing, got %d transitions", n)
	}
	// Only another, enabled, supported fingerprint model can be the target.
	h.transitionPreview(fingerprintID, http.StatusBadRequest)
	h.transitionPreview("no-such-model", http.StatusBadRequest)
	h.transitionPreview(communityID, http.StatusBadRequest)
}
