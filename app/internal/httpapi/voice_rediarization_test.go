//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testVoiceRediarization covers the relaunch of the diarization of a clip already
// processed: explicit confirmation when human work is at stake, the old result kept
// until the new one succeeds, then kept in the history with its provenance; speaker
// corrections obsolete, validated speakers, attributions and Voice samples kept.
func testVoiceRediarization(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip()
	schedule := fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID)
	pending := `SELECT count(*) FROM jobs WHERE clip_id=$1 AND step='voice.diarization.2' AND status IN ('pending','leased','retry_wait')`

	// Activating a model never relaunches anything.
	h.putPipeline(fakePipeline(3))
	h.putPipeline(fakePipeline(2))
	if h.count(pending, h.clipID) != 0 {
		t.Fatal("no automatic rediarization after an activation")
	}

	// Human work: a validated speaker (one Voice sample) and a speaker correction.
	var alice struct {
		Person struct {
			ID int64 `json:"id"`
		} `json:"person"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", first), map[string]any{"new_person_name": "Alice"}, http.StatusCreated), &alice); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/clips/%d/segments/1/speaker", h.clipID), map[string]any{"local_speaker_id": first, "expected_version": 0}, http.StatusOK)
	oldJob := int64(h.count(`SELECT id FROM jobs WHERE clip_id=$1 AND step='voice.diarization.2' AND status='done'`, h.clipID))
	attributionID := h.count(`SELECT id FROM speaker_attributions WHERE local_speaker_id=$1 AND validation_state='validated'`, first)
	sampleID := h.count(`SELECT id FROM voice_samples WHERE local_speaker_id=$1 AND retired_at IS NULL`, first)

	// Without confirmation: refused with what is at stake, nothing scheduled.
	var refusal struct {
		Error string                  `json:"error"`
		Stake voiceRediarizationStake `json:"stake"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, schedule, map[string]any{"audio_stream_index": 2}, http.StatusConflict), &refusal); err != nil {
		t.Fatal(err)
	}
	if refusal.Error != "voice_rediarization_confirmation_required" || refusal.Stake != (voiceRediarizationStake{ValidatedSpeakers: 1, VoiceSamples: 1, SpeakerCorrections: 1}) {
		t.Fatalf("unexpected confirmation request: %+v", refusal)
	}
	if h.count(pending, h.clipID) != 0 {
		t.Fatal("a refused relaunch must schedule nothing")
	}

	// Confirmed: scheduled; the visible result stays the old one while it runs.
	h.adminCall(http.MethodPost, schedule, map[string]any{"audio_stream_index": 2, "confirm_replace_validations": true}, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("expected the relaunched diarization: %+v", job)
	}
	during := h.voice()
	if during.Status != "running" || len(during.Speakers) != 2 || during.Speakers[0].ID != first || during.Speakers[1].ID != second || len(during.History) != 0 {
		t.Fatalf("the old result must stay visible until success: %+v", during)
	}
	if h.count(`SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND speaker_override_id=$2`, h.clipID, first) != 1 {
		t.Fatal("speaker corrections must survive until the relaunch succeeds")
	}

	// Success: new speakers, old result in the history with its provenance.
	h.complete(job, diarizationPayload("NEW_A", "NEW_B", "NEW_C"), http.StatusOK)
	after := h.voice()
	if len(after.Speakers) != 3 || after.Speakers[0].Label != "NEW_A" {
		t.Fatalf("the new result must replace the visible one: %+v", after)
	}
	for _, s := range after.Speakers {
		if s.ID == first || s.ID == second || s.DecisionLevel == "human" {
			t.Fatalf("the archived or replaced speakers must not stay visible: %+v", s)
		}
	}
	if len(after.History) != 1 || after.History[0].JobID != oldJob || after.History[0].Model != "fixture-diarization" || after.History[0].ModelVersion != "fixture-v1" || after.History[0].SpeakerCount != 2 || after.History[0].ValidatedSpeaker != 1 {
		t.Fatalf("the replaced result must be kept with its provenance: %+v", after.History)
	}
	if h.count(`SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND speaker_override_id IS NOT NULL`, h.clipID) != 0 ||
		h.count(`SELECT count(*) FROM segment_correction_events WHERE clip_id=$1 AND old_speaker_id=$2 AND new_speaker_id IS NULL`, h.clipID, first) != 1 {
		t.Fatal("speaker corrections must become obsolete, with a trace")
	}

	// Validated speaker, attribution, sample and fingerprint are kept, linked to the audio.
	if h.count(`SELECT count(*) FROM local_speakers WHERE id=$1 AND kind='archived' AND diarization_job_id=$2`, first, oldJob) != 1 ||
		h.count(`SELECT count(*) FROM local_speakers WHERE id=$1`, second) != 0 ||
		h.count(`SELECT id FROM speaker_attributions WHERE local_speaker_id=$1 AND validation_state='validated'`, first) != attributionID ||
		h.count(`SELECT id FROM voice_samples WHERE local_speaker_id=$1 AND retired_at IS NULL`, first) != sampleID ||
		h.count(`SELECT count(*) FROM voice_fingerprints WHERE local_speaker_id=$1 AND scope='speaker'`, first) != 1 ||
		h.count(`SELECT count(*) FROM speaker_turns WHERE local_speaker_id=$1`, first) != 0 {
		t.Fatal("the validated speaker, its attribution, sample and fingerprint must be kept")
	}
	// The person keeps its reference, so matching still uses it (non-regression on profiles).
	var samples int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM voice_samples WHERE person_id=$1 AND retired_at IS NULL`, alice.Person.ID).Scan(&samples); err != nil || samples != 1 {
		t.Fatalf("the person must keep its validated sample: %d %v", samples, err)
	}

	// Fingerprints of the new result are computed; the archived ones are untouched.
	fp, ok := h.claim()
	if !ok || fp.Step != "voice.fingerprint.2" {
		t.Fatalf("expected the fingerprints of the new diarization: %+v", fp)
	}
	items := []map[string]any{}
	for i, label := range []string{"NEW_A", "NEW_B", "NEW_C"} {
		items = append(items, map[string]any{"speaker_label": label, "scope": "speaker", "start": float64(i), "end": float64(i) + 1.5, "quality": 0.9, "embedding": []float64{0.5, 0.5, 0.5, 0.5}})
	}
	h.complete(fp, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": 2, "diarization_job_id": job.ID, "dimensions": 4, "fingerprints": items}, http.StatusOK)
	if h.count(`SELECT count(*) FROM voice_fingerprints WHERE local_speaker_id=$1`, first) != 1 {
		t.Fatal("the archived fingerprint must survive the new fingerprints")
	}

	// The new speakers carry no validation: relaunching with the same active model needs
	// no confirmation and produces a new cycle when the administrator asks for it.
	h.adminCall(http.MethodPost, schedule, map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	if again, ok := h.claim(); !ok || again.Step != "voice.diarization.2" || again.ID == job.ID {
		t.Fatalf("a new cycle with the same model was expected: %+v", again)
	}
	// An identical request while one runs is idempotent.
	h.adminCall(http.MethodPost, schedule, map[string]any{"audio_stream_index": 2}, http.StatusOK)
}
