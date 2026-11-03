//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type deletionRefusal struct {
	Error string `json:"error"`
	Stake struct {
		Segments       int                 `json:"segments"`
		Corrections    *correctionsAtStake `json:"corrections"`
		Voices         int                 `json:"voices"`
		ValidatedVoice int                 `json:"validated_voices"`
		VoiceSamples   int                 `json:"voice_samples"`
		Persons        int                 `json:"persons"`
	} `json:"stake"`
}

// testClipAnalysisDeletion covers the deletion of analysis steps: the super-administrator
// only, refused during an active analysis, the stake shown before any loss, the cascade of
// each step, the controlled lift of the validated-voice guard, the clip-level audit, and
// the Persons kept.
func testClipAnalysisDeletion(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, _ := h.diarizeClip()
	// Human work: a corrected segment text and a validated voice.
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/clips/%d/segments/0/text", h.clipID), map[string]any{"text": "corrigé", "expected_version": 0}, http.StatusOK)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", first), map[string]any{"new_person_name": "Alice"}, http.StatusCreated)
	deletePath := h.analysisPath()
	deleteAnalysis := func(body map[string]any, want int) []byte {
		t.Helper()
		return h.adminCall(http.MethodDelete, deletePath, body, want)
	}
	refuse := func(body map[string]any) deletionRefusal {
		t.Helper()
		var out deletionRefusal
		if err := json.Unmarshal(deleteAnalysis(body, http.StatusConflict), &out); err != nil {
			t.Fatal(err)
		}
		if out.Error != "analysis_deletion_confirmation_required" {
			t.Fatalf("unexpected refusal: %+v", out)
		}
		return out
	}
	localSpeakers := func() int {
		return h.count(`SELECT count(*) FROM local_speakers WHERE clip_id=$1`, h.clipID)
	}
	transcriptionJobs := func() int {
		return h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%'`, h.clipID)
	}
	persons := func() int { return h.count(`SELECT count(*) FROM persons`) }

	// Only the super-administrator may delete: an administrator and a member are refused.
	admin := h.newUser("deletion-admin")
	h.pool.Exec(t.Context(), `UPDATE users SET role='admin' WHERE username=$1`, "deletion-admin")
	h.userCall(admin, http.MethodDelete, deletePath, map[string]any{"steps": []string{"matching"}}, http.StatusForbidden)
	member := h.newUser("deletion-member")
	h.userCall(member, http.MethodDelete, deletePath, map[string]any{"steps": []string{"matching"}}, http.StatusForbidden)

	// Invalid choices are refused before anything happens.
	for _, body := range []map[string]any{
		{},
		{"steps": []string{}},
		{"steps": []string{"preparation"}},
		{"steps": []string{"indexing"}},
		{"steps": []string{"matching", "weird"}},
	} {
		deleteAnalysis(body, http.StatusBadRequest)
	}

	// Stake: the validated voice and its Person are at stake; nothing is deleted without confirmation.
	stake := refuse(map[string]any{"steps": []string{"diarization"}})
	if stake.Stake.Voices != 2 || stake.Stake.ValidatedVoice != 1 || stake.Stake.VoiceSamples != 1 || stake.Stake.Persons != 1 || stake.Stake.Corrections != nil {
		t.Fatalf("unexpected voice stake: %+v", stake)
	}
	stake = refuse(map[string]any{"steps": []string{"transcription"}})
	if stake.Stake.Corrections == nil || stake.Stake.Corrections.TextSegments != 1 || stake.Stake.Segments == 0 {
		t.Fatalf("unexpected transcription stake: %+v", stake)
	}
	var preview deletionRefusal
	if err := json.Unmarshal(h.adminCall(http.MethodGet, deletePath+"/deletion-preview?steps=diarization", nil, http.StatusOK), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Stake.ValidatedVoice != 1 {
		t.Fatalf("the preview must show the stake: %+v", preview)
	}
	if localSpeakers() != 2 || transcriptionJobs() == 0 {
		t.Fatal("a refused deletion must delete nothing")
	}

	// The guard holds outside the deletion path: a validated voice cannot be deleted directly.
	if _, err := pool.Exec(t.Context(), `DELETE FROM local_speakers WHERE id=$1`, first); err == nil {
		t.Fatal("the guard must refuse a direct deletion of a validated voice")
	}

	// During an active analysis the deletion is refused with 409 analysis_active.
	h.startAnalysis(map[string]any{"scope": "transcription", "mode": "missing"}, http.StatusCreated)
	if h.analysis() != nil && h.analysis().Status == "active" {
		deleteAnalysis(map[string]any{"steps": []string{"transcription"}, "confirm_loss": true}, http.StatusConflict)
		h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusOK)
	}

	// Transcription: its jobs, indexing and corrections go; the voices stay.
	if transcriptionJobs() == 0 || h.count(`SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND text IS NOT NULL`, h.clipID) != 1 {
		t.Fatal("the fixture must hold a transcription and a correction")
	}
	deleteAnalysis(map[string]any{"steps": []string{"transcription"}, "confirm_loss": true}, http.StatusOK)
	if transcriptionJobs() != 0 || h.count(`SELECT count(*) FROM segment_corrections WHERE clip_id=$1`, h.clipID) != 0 {
		t.Fatal("deleting the transcription must delete its jobs and corrections")
	}
	if localSpeakers() != 2 {
		t.Fatal("the voices must not depend on the transcription")
	}
	if h.analysis() != nil {
		if s := h.analysis().step("transcription"); s.Status != "waiting" || s.Reason != "analysis_deleted" {
			t.Fatalf("the deleted step must wait for a new run: %+v", s)
		}
	}

	// Diarization: the validated voice goes with its samples and its attribution, the Person stays.
	deleteAnalysis(map[string]any{"steps": []string{"diarization"}}, http.StatusConflict)
	before := persons()
	deleteAnalysis(map[string]any{"steps": []string{"diarization"}, "confirm_loss": true}, http.StatusOK)
	if localSpeakers() != 0 || h.count(`SELECT count(*) FROM voice_samples WHERE clip_id=$1 AND retired_at IS NULL`, h.clipID) != 0 {
		t.Fatal("deleting the diarization must remove the voices and their samples from the reference")
	}
	if persons() != before {
		t.Fatal("the Person must be kept")
	}
	if h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.%'`, h.clipID) != 0 {
		t.Fatal("deleting the diarization must delete its fingerprint jobs")
	}

	// The clip-level audit survives and names the steps, the author and what was lost.
	if h.count(`SELECT count(*) FROM clip_analysis_deletions WHERE clip_id=$1 AND deleted_by IS NOT NULL AND deleted_by_username <> ''`, h.clipID) != 2 {
		t.Fatal("the audit must record the author of each deletion")
	}
	if h.count(`SELECT count(*) FROM clip_analysis_deletions WHERE clip_id=$1 AND steps='{transcription}' AND lost_corrections=1`, h.clipID) != 1 ||
		h.count(`SELECT count(*) FROM clip_analysis_deletions WHERE clip_id=$1 AND steps='{diarization}' AND lost_validated_voices=1 AND lost_persons=1`, h.clipID) != 1 {
		t.Fatal("each deletion must leave an audit row with its losses")
	}
}

// testClipAnalysisDeletionCascades covers the dependencies that are not reached by the
// full path: fingerprints and matching alone, and the whole analysis at once.
func testClipAnalysisDeletionCascades(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	_, second := h.diarizeClip()
	// A validated reference lets the other voice be matched automatically.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", second), map[string]any{"new_person_name": "Bob"}, http.StatusCreated)
	deletePath := h.analysisPath()
	count := func(query string) int { return h.count(query, h.clipID) }
	automatic := func() int {
		return count(`SELECT count(*) FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND a.validation_state='unvalidated'`)
	}
	if count(`SELECT count(*) FROM voice_fingerprints WHERE clip_id=$1`) == 0 || automatic() == 0 {
		t.Fatal("the fixture must hold fingerprints and an automatic matching")
	}

	// Fingerprints: their jobs and vectors go, with the matching; the diarization stays.
	h.adminCall(http.MethodDelete, deletePath, map[string]any{"steps": []string{"fingerprints"}}, http.StatusOK)
	if count(`SELECT count(*) FROM voice_fingerprints WHERE clip_id=$1`) != 0 || count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.fingerprint.%'`) != 0 || automatic() != 0 {
		t.Fatal("deleting the fingerprints must delete their vectors and the matching")
	}
	if count(`SELECT count(*) FROM local_speakers WHERE clip_id=$1`) != 2 || count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.diarization.%'`) != 1 {
		t.Fatal("deleting the fingerprints must keep the diarization")
	}

	// Matching alone: no validated voice is at stake, the diarization stays.
	h.adminCall(http.MethodDelete, deletePath, map[string]any{"steps": []string{"matching"}}, http.StatusOK)
	if count(`SELECT count(*) FROM local_speakers WHERE clip_id=$1`) != 2 {
		t.Fatal("deleting the matching must keep the voices")
	}

	// Everything: the history, the trials and the notifications of the clip go too.
	if count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%'`) == 0 {
		t.Fatal("the fixture must still hold the transcription")
	}
	h.adminCall(http.MethodDelete, deletePath, map[string]any{"all": true, "confirm_loss": true}, http.StatusOK)
	if count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND (step LIKE 'transcription.audio.%' OR step LIKE 'voice.%')`) != 0 ||
		count(`SELECT count(*) FROM local_speakers WHERE clip_id=$1`) != 0 ||
		count(`SELECT count(*) FROM clip_analysis_runs WHERE clip_id=$1`) != 0 ||
		count(`SELECT count(*) FROM clip_analysis_deletions WHERE clip_id=$1 AND steps='{transcription,diarization,fingerprints,matching}'`) != 1 {
		t.Fatal("deleting the whole analysis must delete every step, the history, and leave one audit row")
	}
	if h.analysis() != nil {
		t.Fatal("the analysis history must be gone")
	}
}
