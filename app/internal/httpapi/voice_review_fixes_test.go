//go:build integration

package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func realPipeline(model, revision string) map[string]any {
	return map[string]any{"provider": "pyannote", "model": model, "model_version": revision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5}
}

// The typed pipeline and "activate" share one rule: a real model needs a Worker that is connected and ready.
func testVoicePipelineActivationRule(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t, pool)
	real := realPipeline(secondRepo, secondRevision)

	h.adminCall(http.MethodPut, "/api/admin/voice-pipeline", real, http.StatusConflict)
	h.adminCall(http.MethodPut, "/api/admin/voice-pipeline", fakePipeline(2), http.StatusOK) // the synthetic provider stays free

	// Ready but not connected: refused by both paths.
	h.reportModel(secondID, "ready", "", 1, 1, http.StatusNoContent)
	if _, err := pool.Exec(t.Context(), `UPDATE workers SET last_seen_at=now() - interval '10 minutes'`); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-pipeline", real, http.StatusConflict)
	h.adminCall(http.MethodPost, "/api/admin/voice-models/"+secondID+"/activate", nil, http.StatusConflict)

	// Ready and connected: accepted, and the model stops being a candidate like with "activate".
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": secondID}, http.StatusOK)
	h.reportModel(secondID, "ready", "", 1, 1, http.StatusNoContent) // reporting refreshes last_seen_at
	h.adminCall(http.MethodPut, "/api/admin/voice-pipeline", real, http.StatusOK)
	if m := h.voiceModels().model(t, secondID); !m.Active || m.Candidate {
		t.Fatalf("a typed pipeline activates through the same path: %+v", m)
	}
}

// A pending job is pinned to the pipeline version it was scheduled with: the claim guard follows that
// version, not the pipeline that is active now.
func testVoiceClaimPinnedVersion(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t, pool)
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent)
	h.putPipeline(realPipeline(activeDiarization, communityRevision))
	h.transcribe()
	h.adminCall(http.MethodPost, "/api/admin/clips/"+strconv64(h.clipID)+"/voice", map[string]any{"audio_stream_index": 2}, http.StatusCreated)

	// Another model becomes active while the job (pinned to the previous one) is still waiting.
	h.reportModel(secondID, "ready", "", 1, 1, http.StatusNoContent)
	h.adminCall(http.MethodPost, "/api/admin/voice-models/"+secondID+"/activate", nil, http.StatusOK)
	if _, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel(secondRepo, "b00000000000")}); ok {
		t.Fatal("a Worker holding only the new active model claimed a job pinned to the previous one")
	}
	if _, ok := h.claimDeclaring(map[string]any{"diarization": voiceModel(activeDiarization, "3533c8cf8e36")}); !ok {
		t.Fatal("the Worker holding the pinned model must claim the job")
	}
}

// Choosing a candidate and removing the same model at the same time never leaves a candidate whose
// installation is marked for removal.
func testVoiceModelConcurrency(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t, pool)
	h.reportModel(secondID, "ready", "", 1, 1, http.StatusNoContent)
	for i := 0; i < 15; i++ {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			h.adminCallStatus(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": secondID})
		}()
		go func() {
			defer wg.Done()
			h.adminCallStatus(http.MethodDelete, "/api/admin/voice-models/"+secondID+"/installation", nil)
		}()
		wg.Wait()
		if n := h.count(`SELECT count(*) FROM voice_model_candidate c JOIN worker_voice_models m ON m.model_id=c.model_id WHERE m.removal_requested`); n != 0 {
			t.Fatalf("round %d: a candidate is marked for removal", i)
		}
		h.adminCall(http.MethodDelete, "/api/admin/voice-models/candidate", nil, http.StatusNoContent)
		h.reportModel(secondID, "ready", "", 1, 1, http.StatusNoContent)
		if _, err := pool.Exec(t.Context(), `UPDATE worker_voice_models SET removal_requested=false`); err != nil {
			t.Fatal(err)
		}
	}
}

// After a relaunch the validated speaker is archived: it no longer makes the clip match its person.
func testVoiceArchivedSpeakersFilter(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, _ := h.diarizeClip()
	var alice struct {
		Person struct {
			ID int64 `json:"id"`
		} `json:"person"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", first), map[string]any{"new_person_name": "Alice"}, http.StatusCreated), &alice); err != nil {
		t.Fatal(err)
	}
	clips := func(query string) (int, int) {
		var page struct {
			Total int `json:"total"`
			Items []struct {
				Persons []struct{} `json:"persons"`
			} `json:"items"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/clips?"+query, nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		persons := 0
		for _, item := range page.Items {
			persons += len(item.Persons)
		}
		return page.Total, persons
	}
	byPerson := fmt.Sprintf("person_id=%d", alice.Person.ID)
	if total, persons := clips(byPerson); total != 1 || persons != 1 {
		t.Fatalf("before the relaunch the clip matches Alice once: %d clips, %d persons", total, persons)
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2, "confirm_replace_validations": true}, http.StatusCreated)
	job, ok := h.claim()
	if !ok {
		t.Fatal("expected the relaunched diarization")
	}
	h.complete(job, diarizationPayload("NEW_A", "NEW_B"), http.StatusOK)
	if h.count(`SELECT count(*) FROM local_speakers WHERE id=$1 AND kind='archived'`, first) != 1 {
		t.Fatal("the validated speaker must be archived")
	}
	if total, persons := clips(byPerson); total != 0 || persons != 0 {
		t.Fatalf("an archived speaker must not make the clip match its person: %d clips, %d persons", total, persons)
	}
	if total, _ := clips("person_unknown=true"); total != 1 {
		t.Fatalf("the clip is now unknown: %d", total)
	}
}

// adminCallStatus is adminCall for goroutines: it reports nothing and fails nothing.
func (h *voiceHarness) adminCallStatus(method, path string, body any) {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, h.ts.URL+path, bytes.NewReader(payload))
	req.Header.Set("Origin", h.ts.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", h.csrf)
	if resp, err := h.admin.Do(req); err == nil {
		resp.Body.Close()
	}
}
