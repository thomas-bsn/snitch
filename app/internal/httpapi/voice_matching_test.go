//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Voice matching runs in the database from deterministic synthetic fingerprints: no
// worker, no media. Every vector lies in the plane of the first two axes, so the cosine
// similarity between two voices is exactly the cosine of the angle between them.
type fixtureFingerprint struct {
	model, version string
	dimensions     int
	vector         []float64
	quality        float64
	usable         bool
}

// angle returns the unit vector at the given angle in degrees, padded to four dimensions.
func angle(degrees float64) []float64 {
	radians := degrees * math.Pi / 180
	return []float64{math.Cos(radians), math.Sin(radians), 0, 0}
}

func usableAt(degrees float64) fixtureFingerprint {
	return fixtureFingerprint{model: "fixture-embedding", version: "emb-v1", dimensions: 4, vector: angle(degrees), quality: 0.9, usable: true}
}

type fixtureSpeaker struct{ ID, ClipID int64 }

// fixtureSpeaker inserts a local speaker with one speaker-scope fingerprint, the way
// the worker results would have projected them.
func (h *voiceHarness) fixtureSpeaker(clipID int64, label string, fp fixtureFingerprint) fixtureSpeaker {
	h.t.Helper()
	ctx := context.Background()
	var diarization, fingerprint, speaker int64
	if err := h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.9',$2,'done') RETURNING id`, clipID, "fixture-d-"+label).Scan(&diarization); err != nil {
		h.t.Fatal(err)
	}
	if err := h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.fingerprint.9',$2,'done') RETURNING id`, clipID, "fixture-f-"+label).Scan(&fingerprint); err != nil {
		h.t.Fatal(err)
	}
	if err := h.pool.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,9,$2,$3,0,2,1) RETURNING id`, clipID, diarization, label).Scan(&speaker); err != nil {
		h.t.Fatal(err)
	}
	reason := ""
	if !fp.usable {
		reason = "too_short"
	}
	vector, _ := json.Marshal(fp.vector)
	if _, err := h.pool.Exec(ctx, `INSERT INTO voice_fingerprints(local_speaker_id,clip_id,fingerprint_job_id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,unusable_reason,contributive,model,model_version,dimensions,embedding) VALUES($1,$2,$3,'speaker',0,2,2,$4,$5,$6,$5,$7,$8,$9,$10::vector)`, speaker, clipID, fingerprint, fp.quality, fp.usable, reason, fp.model, fp.version, fp.dimensions, string(vector)); err != nil {
		h.t.Fatal(err)
	}
	return fixtureSpeaker{ID: speaker, ClipID: clipID}
}

type matchedSpeaker struct {
	ID            int64  `json:"id"`
	State         string `json:"state"`
	DecisionLevel string `json:"decision_level"`
	Person        *struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"person"`
	Attribution *struct {
		ValidationState string   `json:"validation_state"`
		Origin          string   `json:"origin"`
		Confidence      *float64 `json:"confidence"`
		Candidates      []struct {
			Rank   int     `json:"rank"`
			Score  float64 `json:"score"`
			Person struct {
				ID          int64  `json:"id"`
				DisplayName string `json:"display_name"`
			} `json:"person"`
		} `json:"candidates"`
	} `json:"attribution"`
}

// matched reads the attribution of a speaker through the public clip voice API.
func (h *voiceHarness) matched(sp fixtureSpeaker) matchedSpeaker {
	h.t.Helper()
	var view struct {
		Speakers []matchedSpeaker `json:"speakers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", sp.ClipID), nil, http.StatusOK), &view); err != nil {
		h.t.Fatal(err)
	}
	for _, s := range view.Speakers {
		if s.ID == sp.ID {
			return s
		}
	}
	h.t.Fatalf("speaker %d not listed for clip %d", sp.ID, sp.ClipID)
	return matchedSpeaker{}
}

func (h *voiceHarness) validate(sp fixtureSpeaker, person any) {
	h.t.Helper()
	body := map[string]any{}
	switch p := person.(type) {
	case string:
		body["new_person_name"] = p
	case int64:
		body["person_id"] = p
	}
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", sp.ID), body, http.StatusCreated)
}

func (h *voiceHarness) personOf(sp fixtureSpeaker) int64 {
	h.t.Helper()
	m := h.matched(sp)
	if m.Person == nil {
		h.t.Fatalf("speaker %d has no person", sp.ID)
	}
	return m.Person.ID
}

func testVoiceMatching(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	// These synthetic voices speak about 1 s: an eligibility rule at 1 s makes them comparable (#390).
	h.putEligibility(1, 0.5, 0.05)
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]

	// Defaults: the automation is off, so matching can only suggest.
	var settings struct {
		AutoEnabled         bool    `json:"auto_enabled"`
		AutomaticThreshold  float64 `json:"automatic_threshold"`
		SuggestionThreshold float64 `json:"suggestion_threshold"`
		AmbiguityMargin     float64 `json:"ambiguity_margin"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/voice-matching", nil, http.StatusOK), &settings); err != nil {
		t.Fatal(err)
	}
	if settings.AutoEnabled || settings.AutomaticThreshold != 0.90 || settings.SuggestionThreshold != 0.75 || settings.AmbiguityMargin != 0.05 {
		t.Fatalf("unexpected default matching settings: %+v", settings)
	}

	// One human validation becomes the reference of Alice; nearby voices of other clips
	// are matched from the stored fingerprints alone (no worker is running).
	reference := h.fixtureSpeaker(clipA, "REF_ALICE", usableAt(0))
	near := h.fixtureSpeaker(clipB, "NEAR", usableAt(10))       // cosine 0.985
	middling := h.fixtureSpeaker(clipB, "MIDDLE", usableAt(35)) // cosine 0.819
	far := h.fixtureSpeaker(clipC, "FAR", usableAt(60))         // cosine 0.5
	weak := h.fixtureSpeaker(clipC, "WEAK", fixtureFingerprint{model: "fixture-embedding", version: "emb-v1", dimensions: 4, vector: angle(0), quality: 0.2, usable: false})
	otherModel := h.fixtureSpeaker(clipC, "OTHER_MODEL", fixtureFingerprint{model: "fixture-embedding", version: "emb-v2", dimensions: 4, vector: angle(0), quality: 0.9, usable: true})
	otherDims := h.fixtureSpeaker(clipC, "OTHER_DIMS", fixtureFingerprint{model: "fixture-embedding", version: "emb-v1", dimensions: 3, vector: []float64{1, 0, 0}, quality: 0.9, usable: true})
	h.validate(reference, "Alice")
	alice := h.personOf(reference)

	got := h.matched(near)
	if got.DecisionLevel != "suggestion" || got.State != "unidentified" || got.Person == nil || got.Person.ID != alice || got.Attribution == nil || got.Attribution.Origin != "matching" || got.Attribution.ValidationState != "unvalidated" || got.Attribution.Confidence == nil || math.Abs(*got.Attribution.Confidence-0.985) > 0.001 {
		t.Fatalf("a close voice must be a suggestion while the automation is off: %+v", got)
	}
	if got = h.matched(middling); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != alice {
		t.Fatalf("a voice between the thresholds is a suggestion: %+v", got)
	}
	if got = h.matched(far); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("a distant voice stays unknown: %+v", got)
	}
	for name, sp := range map[string]fixtureSpeaker{"low quality": weak, "other model version": otherModel, "other dimensions": otherDims} {
		if got = h.matched(sp); got.DecisionLevel != "unknown" || got.Attribution != nil {
			t.Fatalf("%s fingerprints must be neither compared nor propagated: %+v", name, got)
		}
	}
	// Matching never creates references: only the human validation did.
	if n := h.count(`SELECT count(*) FROM voice_samples`); n != 1 {
		t.Fatalf("matching must not create validated samples, got %d", n)
	}

	// A second Person whose voice is as close to NEAR as Alice's makes NEAR ambiguous:
	// adding a reference re-examines the fingerprints already stored.
	bobReference := h.fixtureSpeaker(clipA, "REF_BOB", usableAt(20))
	h.validate(bobReference, "Bob")
	bob := h.personOf(bobReference)
	got = h.matched(near)
	if got.DecisionLevel != "ambiguous" || got.Person != nil || got.State != "unidentified" || got.Attribution == nil || len(got.Attribution.Candidates) != 2 {
		t.Fatalf("a voice close to two Persons must be ambiguous with both candidates: %+v", got)
	}
	candidates := map[int64]bool{}
	for _, c := range got.Attribution.Candidates {
		candidates[c.Person.ID] = true
		if c.Score < 0.98 || c.Score > 0.99 {
			t.Fatalf("candidate score is not the cosine similarity: %+v", c)
		}
	}
	if !candidates[alice] || !candidates[bob] {
		t.Fatalf("candidates must be Alice and Bob: %+v", got.Attribution.Candidates)
	}
	// MIDDLE (35 degrees) is 0.819 from Alice and 0.966 from Bob: only Bob is within the
	// margin of the best, so the voice is a suggestion for Bob.
	if got = h.matched(middling); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != bob {
		t.Fatalf("the nearer Person must win when the other is outside the margin: %+v", got)
	}

	// Two nearly identical unknown voices are never merged: no Person, no reference, no
	// attribution is created for them without a human validation.
	twinOne := h.fixtureSpeaker(clipB, "TWIN_1", usableAt(100))
	twinTwo := h.fixtureSpeaker(clipC, "TWIN_2", usableAt(101))
	h.validate(h.fixtureSpeaker(clipC, "REF_EXTRA", usableAt(180)), "Carol") // any later rematch sees both twins
	for _, twin := range []fixtureSpeaker{twinOne, twinTwo} {
		if got = h.matched(twin); got.DecisionLevel != "unknown" || got.Person != nil {
			t.Fatalf("two close unknown voices must stay unknown: %+v", got)
		}
	}
	if h.count(`SELECT count(*) FROM persons`) != 3 || h.count(`SELECT count(*) FROM voice_samples`) != 3 {
		t.Fatal("matching must neither create Persons nor references")
	}

	// A microphone-track speaker keeps its designated owner: matching never touches it,
	// even when its fingerprint is close to another Person's reference.
	micSpeaker := h.fixtureSpeaker(clipC, "MIC", usableAt(5))
	if _, err := pool.Exec(context.Background(), `UPDATE local_speakers SET kind='microphone',diarization_job_id=NULL WHERE id=$1`, micSpeaker.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,confidence,origin) VALUES($1,$2,'automatic','unvalidated',1,'microphone_track')`, micSpeaker.ID, bob); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=$1 AND origin='microphone_track' AND person_id=$2`, micSpeaker.ID, bob) != 1 || h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=$1`, micSpeaker.ID) != 1 {
		t.Fatal("matching must ignore microphone-track speakers")
	}

	// A correction retires the old reference and creates a new one: the stored
	// fingerprints are matched again without any worker.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/correction", bobReference.ID), map[string]any{"person_id": alice, "expected_person_id": bob}, http.StatusOK)
	if got = h.matched(near); got.DecisionLevel != "suggestion" || got.Person == nil || got.Person.ID != alice {
		t.Fatalf("after the correction Bob has no reference: NEAR is a suggestion for Alice again: %+v", got)
	}
	if got = h.matched(middling); got.Person == nil || got.Person.ID != alice || got.Attribution.Confidence == nil || *got.Attribution.Confidence < 0.96 {
		t.Fatalf("MIDDLE must follow the corrected references (Alice at 20 degrees now): %+v", got)
	}
	if h.count(`SELECT count(*) FROM voice_samples WHERE retired_at IS NULL`) != 3 || h.count(`SELECT count(*) FROM voice_samples`) != 4 {
		t.Fatal("the correction must retire one sample and create one, matching none")
	}

	// A validated speaker keeps its human decision and carries no machine row.
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=$1 AND validation_state='unvalidated'`, reference.ID) != 0 {
		t.Fatal("a validated speaker must not keep a machine attribution")
	}

	// Without the automation switch no voice is ever automatic, whatever its score. Once
	// an administrator enables it (automation switch), strong matches become automatic and weaker
	// ones stay suggestions; automatic attributions still create no reference.
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=true`); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	if got = h.matched(near); got.DecisionLevel != "automatic" || got.State != "identified" || got.Person == nil || got.Person.ID != alice {
		t.Fatalf("a strong match is automatic once the automation is enabled: %+v", got)
	}
	if h.count(`SELECT count(*) FROM voice_samples`) != 4 {
		t.Fatal("automatic attributions must never become validated samples")
	}

	// Thresholds are configured by administrators only, and a change re-examines the
	// stored fingerprints.
	peer := h.newUser("peer")
	thresholds := map[string]any{"automatic_threshold": 0.99, "suggestion_threshold": 0.97, "ambiguity_margin": 0.05, "max_candidates": 5}
	h.userCall(peer, http.MethodPut, "/api/admin/voice-matching", thresholds, http.StatusForbidden)
	h.userCall(peer, http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusForbidden)
	if body := h.userCall(peer, http.MethodGet, "/api/voice-matching", nil, http.StatusOK); !json.Valid(body) {
		t.Fatal("every signed-in user can read the matching settings")
	}
	for _, bad := range []map[string]any{
		{"automatic_threshold": 0.7, "suggestion_threshold": 0.9, "ambiguity_margin": 0.05, "max_candidates": 5},
		{"automatic_threshold": 1.2, "suggestion_threshold": 0.9, "ambiguity_margin": 0.05, "max_candidates": 5},
		{"automatic_threshold": 0.9, "suggestion_threshold": 0.8, "ambiguity_margin": 0.05, "max_candidates": 1},
		{"automatic_threshold": 0.9, "suggestion_threshold": 0.8},
		{"automatic_threshold": 0.9, "suggestion_threshold": 0.8, "ambiguity_margin": 0.05, "max_candidates": 5, "auto_enabled": true},
	} {
		h.adminCall(http.MethodPut, "/api/admin/voice-matching", bad, http.StatusBadRequest)
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-matching", thresholds, http.StatusOK)
	if got = h.matched(near); got.DecisionLevel != "suggestion" {
		t.Fatalf("a stricter automatic threshold turns NEAR back into a suggestion: %+v", got)
	}
	if got = h.matched(middling); got.DecisionLevel != "unknown" || got.Person != nil {
		t.Fatalf("a stricter suggestion threshold turns MIDDLE into unknown: %+v", got)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75`); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)

	// Fingerprints arriving from the worker are matched against the existing references.
	h.transcribe()
	h.putPipeline(fakePipeline(2))
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("expected diarization, got %+v", job)
	}
	h.complete(job, diarizationPayload("SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	fp, ok := h.claim()
	if !ok || fp.Step != "voice.fingerprint.2" {
		t.Fatalf("expected fingerprints, got %+v", fp)
	}
	items := []map[string]any{
		{"speaker_label": "SPEAKER_00", "scope": "speaker", "start": 0, "end": 1.5, "quality": 0.9, "embedding": angle(1)},
		{"speaker_label": "SPEAKER_01", "scope": "speaker", "start": 1, "end": 2.5, "quality": 0.9, "embedding": angle(120)},
	}
	h.complete(fp, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": 2, "diarization_job_id": job.ID, "dimensions": 4, "fingerprints": items}, http.StatusOK)
	var view struct {
		Speakers []matchedSpeaker `json:"speakers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", h.clipID), nil, http.StatusOK), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Speakers) != 2 || view.Speakers[0].DecisionLevel != "suggestion" || view.Speakers[0].Person == nil || view.Speakers[0].Person.ID != alice || view.Speakers[1].DecisionLevel != "unknown" {
		t.Fatalf("worker fingerprints must be matched on arrival: %+v", view.Speakers)
	}

	// Every fingerprint group has its own HNSW index, and the machine decisions leave a
	// trace in the history of the speaker.
	if n := h.count(`SELECT count(*) FROM pg_indexes WHERE indexname LIKE 'voice_fingerprints_hnsw_%'`); n != 3 {
		t.Fatalf("expected one HNSW index per (model, version, dimensions), got %d", n)
	}
	var history struct {
		Items []struct {
			Action string `json:"action"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/local-speakers/%d/history", near.ID), nil, http.StatusOK), &history); err != nil {
		t.Fatal(err)
	}
	if len(history.Items) == 0 || history.Items[0].Action != "match" {
		t.Fatalf("matching changes must be recorded in the history: %+v", history)
	}
}
