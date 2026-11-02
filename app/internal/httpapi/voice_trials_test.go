//go:build integration

package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const thirdID = "test-diarization-c"

type trialView struct {
	ID          int64  `json:"id"`
	ModelID     string `json:"model_id"`
	NumSpeakers *int   `json:"num_speakers"`
	Status      string `json:"status"`
	Speakers    []struct {
		Label           string  `json:"label"`
		SpeakingSeconds float64 `json:"speaking_seconds"`
		TurnCount       int     `json:"turn_count"`
	} `json:"speakers"`
}

func (h *voiceHarness) trials() []trialView {
	h.t.Helper()
	var out struct {
		Trials []trialView `json:"trials"`
	}
	path := fmt.Sprintf("/api/admin/clips/%d/voice-trials", h.clipID)
	if err := json.Unmarshal(h.adminCall(http.MethodGet, path, nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out.Trials
}

// claimTrial pulses as a Worker that accepts trial steps, then tries to claim one job.
func (h *voiceHarness) claimTrial() (claimedJob, bool) {
	h.t.Helper()
	h.workerCall(http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"transcription.audio.*", "voice.*", "voicetrial.*"}}, "", http.StatusOK)
	var a struct {
		Job        *claimedJob `json:"job"`
		LeaseToken string      `json:"lease_token"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodPost, "/api/worker/jobs/claim", nil, "", http.StatusOK), &a); err != nil {
		h.t.Fatal(err)
	}
	if a.Job == nil {
		return claimedJob{}, false
	}
	a.Job.Lease = a.LeaseToken
	return *a.Job, true
}

func testVoiceTrials(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t)
	saved := voiceCatalog
	voiceCatalog = append(append([]voiceCatalogEntry{}, saved...), voiceCatalogEntry{
		ID: thirdID, Kind: "diarization", Engine: "synthetic", Repo: "synthetic/diarization-c", Revision: "c00000000000000000000000000000000000000c",
		License: "test", MaxSpeakers: 4, SizeBytes: 1, SupportsNumSpeakers: false,
	})
	t.Cleanup(func() { voiceCatalog = saved })

	// A validated result that no trial may ever change.
	left, _ := h.diarizeClip()
	h.validate(fixtureSpeaker{ID: left, ClipID: h.clipID}, "Alice")
	protected := []string{"local_speakers", "speaker_turns", "speaker_attributions", "attribution_events", "voice_fingerprints", "voice_samples", "speaker_attribution_candidates", "person_events", "segment_corrections", "persons"}
	snapshot := func() []int {
		counts := make([]int, len(protected))
		for i, table := range protected {
			counts[i] = h.count("SELECT count(*) FROM " + table)
		}
		return counts
	}
	before := snapshot()
	pipelineBefore := h.pipelineVersion()
	voiceBefore := h.voice()
	path := fmt.Sprintf("/api/admin/clips/%d/voice-trials", h.clipID)
	trial := func(model string, options map[string]any, want int) []byte {
		body := map[string]any{"model_id": model, "audio_stream_index": 2}
		for k, v := range options {
			body[k] = v
		}
		return h.adminCall(http.MethodPost, path, body, want)
	}

	// Only administrators run or read trials.
	member := h.newUser("trial-member")
	if status, _ := h.userStatus(member, http.MethodGet, path, nil); status != http.StatusForbidden {
		t.Fatalf("a member must not read trials: %d", status)
	}

	// Only the candidate or the active model, from the catalogue, and only when a Worker holds it ready.
	trial("someone/else", nil, http.StatusBadRequest)
	trial(secondID, nil, http.StatusBadRequest) // neither candidate nor active yet
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": thirdID}, http.StatusOK)
	if body := trial(thirdID, nil, http.StatusConflict); !bytes.Contains(body, []byte("voice_model_not_ready")) {
		t.Fatalf("expected voice_model_not_ready: %s", body)
	}
	h.reportModel(thirdID, "ready", "", 1, 1, http.StatusNoContent)

	// An option the engine cannot honour is refused, never silently dropped.
	if body := trial(thirdID, map[string]any{"num_speakers": 3}, http.StatusBadRequest); !bytes.Contains(body, []byte("voice_trial_option_unsupported")) {
		t.Fatalf("expected voice_trial_option_unsupported: %s", body)
	}
	trial(thirdID, map[string]any{"min_speakers": 2}, http.StatusBadRequest)

	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": secondID}, http.StatusOK)
	h.reportModel(secondID, "ready", "", 1, 1, http.StatusNoContent)
	trial(secondID, map[string]any{"num_speakers": 0}, http.StatusBadRequest)
	trial(secondID, map[string]any{"num_speakers": 3, "min_speakers": 2}, http.StatusBadRequest)
	trial(secondID, map[string]any{"min_speakers": 4, "max_speakers": 2}, http.StatusBadRequest)

	// The trial is queued under its own step type and targets the requested model.
	trial(secondID, map[string]any{"num_speakers": 3}, http.StatusCreated)
	trial(secondID, map[string]any{"num_speakers": 3}, http.StatusConflict) // already running
	if list := h.trials(); len(list) != 1 || list[0].Status != "pending" || list[0].NumSpeakers == nil || *list[0].NumSpeakers != 3 {
		t.Fatalf("unexpected trials %+v", list)
	}
	// A Worker that does not hold the model ready receives nothing.
	if _, err := pool.Exec(t.Context(), `UPDATE worker_voice_models SET state='verifying' WHERE model_id=$1`, secondID); err != nil {
		t.Fatal(err)
	}
	if job, ok := h.claimTrial(); ok {
		t.Fatalf("a Worker without the model ready claimed %s", job.Step)
	}
	// A Worker that did not declare trial steps never gets one either.
	if _, err := pool.Exec(t.Context(), `UPDATE worker_voice_models SET state='ready' WHERE model_id=$1`, secondID); err != nil {
		t.Fatal(err)
	}
	if job, ok := h.claimDeclaring(nil); ok {
		t.Fatalf("a Worker without trial support claimed %s", job.Step)
	}
	job, ok := h.claimTrial()
	if !ok || job.Step != fmt.Sprintf("voicetrial.diarization.%d", h.trials()[0].ID) {
		t.Fatalf("expected the trial job, got %+v", job)
	}
	var input struct {
		Voice struct {
			Config map[string]any `json:"config"`
			Trial  map[string]any `json:"trial"`
		} `json:"voice"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, http.StatusOK), &input); err != nil {
		t.Fatal(err)
	}
	if input.Voice.Config["model"] != secondRepo || input.Voice.Config["model_version"] != secondRevision || input.Voice.Config["num_speakers"] != float64(3) || input.Voice.Trial["audio_stream_index"] != float64(2) {
		t.Fatalf("the trial must carry its model and options: %+v", input.Voice)
	}
	// A result for another model is refused; the right one is recorded in the trial only.
	wrong := diarizationPayload("A", "B", "C")
	h.complete(job, wrong, http.StatusBadRequest)
	result := diarizationPayload("A", "B", "C")
	result["provider"], result["model"], result["model_version"] = "pyannote", secondRepo, secondRevision
	h.complete(job, result, http.StatusOK)
	list := h.trials()
	if len(list) != 1 || list[0].Status != "done" || len(list[0].Speakers) != 3 || list[0].Speakers[0].TurnCount != 1 || list[0].Speakers[0].SpeakingSeconds != 1.5 {
		t.Fatalf("the trial result is not kept: %+v", list)
	}
	if _, ok := h.claimTrial(); ok {
		t.Fatal("no further job expected, and never a fingerprint step for a trial")
	}

	// The validated result is untouched: same speakers, attributions, fingerprints, pipeline and view.
	after := snapshot()
	for i, table := range protected {
		if after[i] != before[i] {
			t.Fatalf("%s changed from %d to %d rows", table, before[i], after[i])
		}
	}
	if h.pipelineVersion() != pipelineBefore {
		t.Fatal("a trial must not change the active pipeline")
	}
	if voiceAfter := h.voice(); len(voiceAfter.Speakers) != len(voiceBefore.Speakers) || voiceAfter.Status != voiceBefore.Status || voiceAfter.Speakers[0].Attribution == nil || voiceAfter.Speakers[0].Attribution.ValidationState != "validated" {
		t.Fatalf("the clip voice result changed: %+v", voiceAfter)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.%'`, h.clipID); n != 2 {
		t.Fatalf("a trial must not create regular voice jobs: %d", n)
	}

	// Kept once per (clip, model, options): running it again replaces the result; another option is another trial.
	trial(secondID, map[string]any{"num_speakers": 3}, http.StatusCreated)
	trial(secondID, map[string]any{"num_speakers": 2}, http.StatusCreated)
	if list = h.trials(); len(list) != 2 {
		t.Fatalf("expected two trials: %+v", list)
	}
	for _, item := range list {
		if item.NumSpeakers != nil && *item.NumSpeakers == 3 && (item.Status != "pending" || len(item.Speakers) != 3) {
			t.Fatalf("a rerun keeps the previous result visible until the new one succeeds: %+v", item)
		}
	}

	// A trial is deletable, and so is the work still waiting for it.
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", path, list[0].ID), nil, http.StatusNoContent)
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", path, list[0].ID), nil, http.StatusNotFound)
	h.adminCall(http.MethodDelete, fmt.Sprintf("%s/%d", path, list[1].ID), nil, http.StatusNoContent)
	if list = h.trials(); len(list) != 0 {
		t.Fatalf("deleted trials still listed: %+v", list)
	}
	if _, ok := h.claimTrial(); ok {
		t.Fatal("the work of a deleted trial must not be claimable")
	}
	if got := snapshot(); fmt.Sprint(got) != fmt.Sprint(before) {
		t.Fatalf("deleting a trial changed validated data: %v vs %v", got, before)
	}
}
