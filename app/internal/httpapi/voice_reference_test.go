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

// The rule of this suite: 3 s of net speech to be comparable, 1 s to be contributive.
const (
	referenceComparable   = 3.0
	referenceContributive = 1.0
)

type shortVoice struct {
	vector  []float64
	net     float64
	version string // fingerprint model version; emb-v1 by default
}

// shortVoice inserts a local speaker whose speaker fingerprint carries Worker measures (net speech, no overlap,
// no clipping), then lets the eligibility rule of the suite decide it through the administrator API.
func (h *voiceHarness) shortVoice(clipID int64, label string, v shortVoice) fixtureSpeaker {
	h.t.Helper()
	ctx := context.Background()
	if v.version == "" {
		v.version = "emb-v1"
	}
	var diarization, fingerprint, speaker int64
	if err := h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.9',$2,'done') RETURNING id`, clipID, "ref-d-"+label).Scan(&diarization); err != nil {
		h.t.Fatal(err)
	}
	if err := h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.fingerprint.9',$2,'done') RETURNING id`, clipID, "ref-f-"+label).Scan(&fingerprint); err != nil {
		h.t.Fatal(err)
	}
	if err := h.pool.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,9,$2,$3,0,$4,1) RETURNING id`, clipID, diarization, label, v.net).Scan(&speaker); err != nil {
		h.t.Fatal(err)
	}
	vector, _ := json.Marshal(v.vector)
	if _, err := h.pool.Exec(ctx, `INSERT INTO voice_fingerprints(local_speaker_id,clip_id,fingerprint_job_id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,unusable_reason,contributive,model,model_version,dimensions,embedding,measures_version,net_speech_seconds,overlap_ratio,overlapped,clipped_ratio,net_speech_source)
 VALUES($1,$2,$3,'speaker',0,$4,$4,0.2,false,'too_short',false,'fixture-embedding',$5,4,$6::vector,1,$4,0,false,0,'worker')`, speaker, clipID, fingerprint, v.net, v.version, string(vector)); err != nil {
		h.t.Fatal(err)
	}
	h.putEligibility(referenceComparable, referenceContributive, 0.05)
	return fixtureSpeaker{ID: speaker, ClipID: clipID}
}

type scoreModeView struct {
	Mode    string  `json:"mode"`
	Version int     `json:"version"`
	Author  *string `json:"author"`
	History []struct {
		Version int     `json:"version"`
		Mode    string  `json:"mode"`
		Author  *string `json:"author"`
	} `json:"history"`
}

func (h *voiceHarness) scoreMode(method string, body any, want int) scoreModeView {
	h.t.Helper()
	var out scoreModeView
	raw := h.adminCall(method, "/api/admin/voice-score-mode", body, want)
	if want == http.StatusOK {
		if err := json.Unmarshal(raw, &out); err != nil {
			h.t.Fatal(err)
		}
	}
	return out
}

// cosineOf is the expected score, computed here from the vectors alone.
func cosineOf(a, b []float64) float64 {
	var dot, na, nb float64
	for i := range a {
		dot, na, nb = dot+a[i]*b[i], na+a[i]*a[i], nb+b[i]*b[i]
	}
	return dot / math.Sqrt(na*nb)
}

func testVoiceReference(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 6)
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clips := h.clipIDs

	// Three 2 s extracts of Dora, each pushed off her voice (the first axis) by a different noise. Under the rule
	// none is comparable on its own; all three are contributive.
	v1, v2, v3 := []float64{1, 1.2, 0, 0}, []float64{1, -0.6, 1.0392305, 0}, []float64{1, -0.6, -1.0392305, 0}
	s1 := h.shortVoice(clips[0], "S1", shortVoice{vector: v1, net: 2})
	s2 := h.shortVoice(clips[1], "S2", shortVoice{vector: v2, net: 2})
	s3 := h.shortVoice(clips[2], "S3", shortVoice{vector: v3, net: 2})
	fourthVector := []float64{1, 0.1, 0, 0}
	fourth := h.shortVoice(clips[3], "FOURTH", shortVoice{vector: fourthVector, net: 3.5})
	for _, sp := range []fixtureSpeaker{s1, s2, s3} {
		if e := h.eligibility(sp); e.Usable || !e.Contributive {
			t.Fatalf("a 2 s extract must be contributive without being comparable: %+v", e)
		}
	}
	if e := h.eligibility(fourth); !e.Usable {
		t.Fatalf("the fourth extract must be comparable: %+v", e)
	}
	full := cosineOf(fourthVector, []float64{1, 0, 0, 0})              // Reference of the three: Dora's true voice
	partial := cosineOf(fourthVector, []float64{2, 0.6, 1.0392305, 0}) // Reference of S1 and S2 only
	confidence := func(sp fixtureSpeaker, person int64, level string, want float64) {
		t.Helper()
		got := h.matched(sp)
		if got.DecisionLevel != level || got.Person == nil || got.Person.ID != person || got.Attribution == nil || got.Attribution.Confidence == nil || math.Abs(*got.Attribution.Confidence-want) > 1e-5 {
			t.Fatalf("expected a %s for Person %d at %.6f: %+v (confidence %v)", level, person, want, got, got.Attribution)
		}
	}
	unknown := func(sp fixtureSpeaker, why string) {
		t.Helper()
		if got := h.matched(sp); got.DecisionLevel != "unknown" || got.Person != nil {
			t.Fatalf("%s: %+v", why, got)
		}
	}

	// One extract is not enough: its own cosine with the fourth is below the suggestion threshold.
	h.validate(s1, "Dora")
	dora := h.personOf(s1)
	unknown(fourth, "a single short extract must not recognise the fourth")
	h.validate(s2, dora)
	h.validate(s3, dora)
	confidence(fourth, dora, "suggestion", full)
	if full < 0.99 || partial > 0.9 || cosineOf(fourthVector, v1) >= 0.75 {
		t.Fatalf("the fixture no longer separates the cases: full %.4f partial %.4f alone %.4f", full, partial, cosineOf(fourthVector, v1))
	}

	// The score mode is a versioned setting: cumulative by default, decided by Snitch.
	mode := h.scoreMode(http.MethodGet, nil, http.StatusOK)
	if mode.Mode != "cumulative" || mode.Version != 1 || mode.Author != nil || len(mode.History) != 1 {
		t.Fatalf("unexpected default score mode: %+v", mode)
	}
	// The best-sample mode compares each comparable sample alone: Dora has none, the fourth is unknown again.
	mode = h.scoreMode(http.MethodPut, map[string]any{"mode": "best_sample"}, http.StatusOK)
	if mode.Mode != "best_sample" || mode.Version != 2 || mode.Author == nil || len(mode.History) != 2 || mode.History[1].Mode != "cumulative" {
		t.Fatalf("a mode change must be a new traced version: %+v", mode)
	}
	unknown(fourth, "the best-sample mode has no comparable reference of Dora")
	if again := h.scoreMode(http.MethodPut, map[string]any{"mode": "best_sample"}, http.StatusOK); again.Version != 2 {
		t.Fatalf("the same mode must record nothing: %+v", again)
	}
	h.scoreMode(http.MethodPut, map[string]any{"mode": "centroid"}, http.StatusBadRequest)
	h.scoreMode(http.MethodPut, map[string]any{}, http.StatusBadRequest)
	member := h.newUser("score-member")
	h.userCall(member, http.MethodGet, "/api/admin/voice-score-mode", nil, http.StatusForbidden)
	h.userCall(member, http.MethodPut, "/api/admin/voice-score-mode", map[string]any{"mode": "cumulative"}, http.StatusForbidden)
	if mode = h.scoreMode(http.MethodPut, map[string]any{"mode": "cumulative"}, http.StatusOK); mode.Version != 3 {
		t.Fatalf("switching back is a new version: %+v", mode)
	}
	confidence(fourth, dora, "suggestion", full)

	// An automatic attribution never enters a Reference: STRAY (cosine 0.91 with Dora) becomes automatic and the
	// score of the fourth stays the one of the three validated extracts.
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=true`); err != nil {
		t.Fatal(err)
	}
	stray := h.shortVoice(clips[4], "STRAY", shortVoice{vector: []float64{1, 0.45, 0, 0}, net: 4})
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	confidence(stray, dora, "automatic", cosineOf([]float64{1, 0.45, 0, 0}, []float64{1, 0, 0, 0}))
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	confidence(fourth, dora, "automatic", full)
	if n := h.count(`SELECT count(*) FROM voice_samples WHERE retired_at IS NULL`); n != 3 {
		t.Fatalf("matching must not create samples, got %d", n)
	}

	// A sample of another fingerprint group is never mixed into this one.
	other := h.shortVoice(clips[4], "OTHER_GROUP", shortVoice{vector: []float64{0, 1, 0, 0}, net: 4, version: "emb-v2"})
	h.validate(other, dora)
	confidence(fourth, dora, "automatic", full)

	// Ignored and random voices are never matched; restoring them matches them again (ADR 0031).
	for _, disposition := range []string{"ignored", "random"} {
		h.setDisposition(fourth, disposition, http.StatusOK)
		if got := h.matched(fourth); got.Attribution != nil {
			t.Fatalf("a %s voice keeps no machine attribution: %+v", disposition, got)
		}
		h.restoreDisposition(fourth)
		confidence(fourth, dora, "automatic", full)
	}
	// A refused Person is never proposed again for that voice, whatever her Reference.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/refusal", fourth.ID), map[string]any{"person_id": dora}, http.StatusOK)
	unknown(fourth, "a refused Person must not be proposed")
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/local-speakers/%d/refusals/%d", fourth.ID, dora), nil, http.StatusOK)
	confidence(fourth, dora, "automatic", full)

	// Correcting a sample removes it from the Reference: Dora keeps S1 and S2.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/correction", s3.ID), map[string]any{"new_person_name": "Eve", "expected_person_id": dora}, http.StatusOK)
	eve := h.personOf(s3)
	confidence(fourth, dora, "suggestion", partial)
	// Merging Eve into Dora brings the sample back.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/persons/%d/merge", eve), map[string]any{"target_person_id": dora}, http.StatusOK)
	confidence(fourth, dora, "automatic", full)
	// Separating it takes it out again; a new validation puts it back.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/persons/%d/separation", dora), map[string]any{"local_speaker_ids": []int64{s3.ID}}, http.StatusOK)
	confidence(fourth, dora, "suggestion", partial)
	h.validate(s3, dora)
	confidence(fourth, dora, "automatic", full)

	// Deleting the clip of a sample takes it out of the Reference.
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/clips/%d", s3.ClipID), nil, http.StatusOK)
	confidence(fourth, dora, "suggestion", partial)
	// So does deleting the fingerprints of a sample's clip: S1 alone does not recognise the fourth.
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/clips/%d/analysis", s2.ClipID), map[string]any{"steps": []string{"fingerprints"}, "confirm_loss": true}, http.StatusOK)
	unknown(fourth, "after the fingerprints of S2 are deleted, S1 alone is not enough")

	// Deleting Dora's voice profile leaves her without Reference.
	h.validate(h.shortVoice(clips[5], "S5", shortVoice{vector: v2, net: 2}), dora)
	confidence(fourth, dora, "suggestion", partial)
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/persons/%d/voice-profile", dora), nil, http.StatusOK)
	if got := h.matched(fourth); got.Attribution != nil || got.Person != nil {
		t.Fatalf("without voice profile Dora has no Reference: %+v", got)
	}
}
