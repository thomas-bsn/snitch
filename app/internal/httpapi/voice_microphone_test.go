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

type micSpeaker struct {
	ID            int64  `json:"id"`
	Label         string `json:"label"`
	Kind          string `json:"kind"`
	State         string `json:"state"`
	DecisionLevel string `json:"decision_level"`
	Person        *struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"person"`
	Attribution *struct {
		Origin          string   `json:"origin"`
		ValidationState string   `json:"validation_state"`
		Confidence      *float64 `json:"confidence"`
	} `json:"attribution"`
}

func (h *voiceHarness) speakers() []micSpeaker {
	h.t.Helper()
	var v struct {
		Speakers []micSpeaker `json:"speakers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", h.clipID), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v.Speakers
}

type micTrackView struct {
	Status           string `json:"status"`
	SelectionSource  string `json:"selection_source"`
	AudioStreamIndex *int   `json:"audio_stream_index"`
	Applied          bool   `json:"applied"`
	LocalSpeakerID   *int64 `json:"local_speaker_id"`
	Person           *struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"person"`
	Candidates []int `json:"candidates"`
	Tracks     []struct {
		Index int    `json:"index"`
		Title string `json:"title"`
		Role  string `json:"role"`
	} `json:"tracks"`
	History []struct {
		Scope  string          `json:"scope"`
		Action string          `json:"action"`
		Old    json.RawMessage `json:"old_rule"`
		New    json.RawMessage `json:"new_rule"`
		Author *struct {
			Username string `json:"username"`
		} `json:"author"`
	} `json:"history"`
}

func (h *voiceHarness) micTrack() micTrackView {
	h.t.Helper()
	var v micTrackView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/microphone-track", h.clipID), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

func (h *voiceHarness) newPerson(name string) int64 {
	h.t.Helper()
	var p struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/persons", map[string]any{"display_name": name}, http.StatusCreated), &p); err != nil {
		h.t.Fatal(err)
	}
	return p.ID
}

// retitleTrack renames one detected track of a synthetic clip (to build a duplicate title).
const retitleTrack = `UPDATE clips SET tracks=(SELECT jsonb_agg(CASE WHEN x.t->>'index'=$2 THEN jsonb_set(x.t,'{title}',to_jsonb($3::text)) ELSE x.t END ORDER BY x.ord) FROM jsonb_array_elements(tracks) WITH ORDINALITY x(t,ord)) WHERE id=$1`

func testMicrophoneTrack(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	var sourceID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM clip_sources ORDER BY id LIMIT 1`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	rulePath := fmt.Sprintf("/api/admin/sources/%d/microphone-track", sourceID)
	alice, bob := h.newPerson("Alice Micro"), h.newPerson("Bob Micro")
	_ = bob

	// Nothing is designated by default and the tracks offered are the ones detected.
	var rule struct {
		Rule   *struct{} `json:"rule"`
		Detect []struct {
			Index int    `json:"audio_stream_index"`
			Title string `json:"title"`
			Clips int    `json:"clip_count"`
		} `json:"detected_tracks"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, rulePath, nil, http.StatusOK), &rule); err != nil {
		t.Fatal(err)
	}
	if rule.Rule != nil || len(rule.Detect) != 2 || rule.Detect[0].Index != 1 || rule.Detect[0].Title != "Micro" || rule.Detect[1].Title != "Voix" || rule.Detect[0].Clips != 1 {
		t.Fatalf("detected tracks must come from the clips of the source: %+v", rule)
	}
	if got := h.micTrack(); got.Status != "none" || len(got.Tracks) != 2 || got.Tracks[0].Role != "other" {
		t.Fatalf("an undesignated clip reports no microphone track: %+v", got)
	}

	// Only administrators designate; invalid rules and unknown Persons are refused.
	member := h.newUser("member")
	h.userCall(member, http.MethodPut, rulePath, map[string]any{"title": "Micro", "person_id": alice}, http.StatusForbidden)
	h.adminCall(http.MethodPut, rulePath, map[string]any{"person_id": alice}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, rulePath, map[string]any{"title": "  ", "person_id": alice}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, rulePath, map[string]any{"title": "Micro", "person_id": 999999}, http.StatusNotFound)
	h.adminCall(http.MethodPut, "/api/admin/sources/999999/microphone-track", map[string]any{"title": "Micro", "person_id": alice}, http.StatusNotFound)

	// A designated track is attributed to its owner without any voice task.
	h.adminCall(http.MethodPut, rulePath, map[string]any{"title": "Micro", "person_id": alice}, http.StatusOK)
	if n := h.count(`SELECT count(*) FROM jobs WHERE step LIKE 'voice.%'`); n != 0 {
		t.Fatalf("a microphone track must not need a diarization task: %d", n)
	}
	speakers := h.speakers()
	if len(speakers) != 1 || speakers[0].Kind != "microphone" || speakers[0].Person == nil || speakers[0].Person.ID != alice || speakers[0].DecisionLevel != "automatic" || speakers[0].State != "identified" {
		t.Fatalf("microphone track must carry its owner: %+v", speakers)
	}
	if a := speakers[0].Attribution; a == nil || a.Origin != "microphone_track" || a.ValidationState != "unvalidated" || a.Confidence == nil || *a.Confidence != 1 {
		t.Fatalf("designation is stored as a distinct, unvalidated attribution: %+v", a)
	}
	if n := h.count(`SELECT count(*) FROM voice_samples`); n != 0 {
		t.Fatal("a designation never becomes a validated sample")
	}
	got := h.micTrack()
	if got.Status != "designated" || got.SelectionSource != "source_rule" || got.AudioStreamIndex == nil || *got.AudioStreamIndex != 1 || !got.Applied || got.Tracks[0].Role != "microphone" || got.Tracks[1].Role != "other" {
		t.Fatalf("clip view: %+v", got)
	}

	// The other tracks are processed normally: track 2 is transcribed and diarized,
	// and the microphone speaker stays next to the diarized ones.
	h.transcribe()
	h.putPipeline(fakePipeline(2))
	clipVoice := fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID)
	h.adminCall(http.MethodPost, clipVoice, map[string]any{"audio_stream_index": 1}, http.StatusConflict)
	if n := h.count(`SELECT count(*) FROM jobs WHERE step LIKE 'voice.%'`); n != 0 {
		t.Fatalf("scheduling the microphone track must create nothing: %d", n)
	}
	h.adminCall(http.MethodPost, clipVoice, map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("expected diarization of the other track: %+v", job)
	}
	h.complete(job, diarizationPayload("SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	speakers = h.speakers()
	if len(speakers) != 3 || speakers[0].Kind != "microphone" || speakers[1].Kind != "diarized" || speakers[1].DecisionLevel != "unknown" || speakers[2].Label != "SPEAKER_01" {
		t.Fatalf("diarized tracks and microphone track must coexist: %+v", speakers)
	}

	// A designation that matches nothing is reported, never dropped silently.
	var counts struct {
		StatusCounts map[string]int `json:"status_counts"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPut, rulePath, map[string]any{"title": "Absente", "person_id": alice}, http.StatusOK), &counts); err != nil {
		t.Fatal(err)
	}
	if counts.StatusCounts["track_not_found"] != 1 {
		t.Fatalf("unmatched designation must be counted: %+v", counts)
	}
	got = h.micTrack()
	if got.Status != "track_not_found" || got.SelectionSource != "source_rule" || got.Applied || got.Tracks[0].Role != "other" || got.Tracks[1].Role != "other" {
		t.Fatalf("unmatched designation must be explicit: %+v", got)
	}
	if speakers = h.speakers(); len(speakers) != 2 || speakers[0].Kind != "diarized" {
		t.Fatalf("a designation that does not apply attributes nothing: %+v", speakers)
	}
	if _, err := pool.Exec(context.Background(), retitleTrack, h.clipID, "2", "Micro"); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, rulePath, map[string]any{"title": "Micro", "person_id": alice}, http.StatusOK)
	if got = h.micTrack(); got.Status != "ambiguous_track" || len(got.Candidates) != 2 || got.Tracks[0].Role != "candidate" || got.Applied {
		t.Fatalf("several matching tracks must be reported as ambiguous: %+v", got)
	}
	h.adminCall(http.MethodPut, rulePath, map[string]any{"audio_stream_index": 1, "title": "Micro", "person_id": alice}, http.StatusOK)
	if got = h.micTrack(); got.Status != "designated" || !got.Applied || got.Tracks[0].Role != "microphone" {
		t.Fatalf("the index disambiguates: %+v", got)
	}
	if _, err := pool.Exec(context.Background(), retitleTrack, h.clipID, "2", "Voix"); err != nil {
		t.Fatal(err)
	}

	// Replacement for one clip: no microphone, another owner, then back to the rule.
	clipPath := fmt.Sprintf("/api/admin/clips/%d/microphone-track", h.clipID)
	h.userCall(member, http.MethodPut, clipPath, map[string]any{"disabled": true}, http.StatusForbidden)
	h.adminCall(http.MethodPut, clipPath, map[string]any{}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, clipPath, map[string]any{"disabled": true, "audio_stream_index": 1, "person_id": bob}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, clipPath, map[string]any{"audio_stream_index": 9, "person_id": bob}, http.StatusBadRequest)
	h.adminCall(http.MethodPut, clipPath, map[string]any{"audio_stream_index": 1, "person_id": 999999}, http.StatusNotFound)
	h.adminCall(http.MethodPut, "/api/admin/clips/999999/microphone-track", map[string]any{"disabled": true}, http.StatusNotFound)
	h.adminCall(http.MethodPut, clipPath, map[string]any{"disabled": true}, http.StatusOK)
	if got = h.micTrack(); got.Status != "disabled" || got.SelectionSource != "clip_override" || len(h.speakers()) != 2 {
		t.Fatalf("a clip can say it has no microphone track: %+v", got)
	}
	h.adminCall(http.MethodPut, clipPath, map[string]any{"disabled": true}, http.StatusOK)
	h.adminCall(http.MethodPut, clipPath, map[string]any{"audio_stream_index": 1, "person_id": bob}, http.StatusOK)
	got = h.micTrack()
	if got.Status != "designated" || got.SelectionSource != "clip_override" || got.Person == nil || got.Person.ID != bob {
		t.Fatalf("a clip can replace the owner: %+v", got)
	}
	if speakers = h.speakers(); speakers[0].Kind != "microphone" || speakers[0].Person.ID != bob {
		t.Fatalf("replacement must reach the attribution: %+v", speakers)
	}
	h.adminCall(http.MethodDelete, clipPath, nil, http.StatusOK)
	if got = h.micTrack(); got.Status != "designated" || got.SelectionSource != "source_rule" || got.Person.ID != alice {
		t.Fatalf("removing the replacement restores the source rule: %+v", got)
	}
	// Changes are recorded with before/after, author and scope; repeated saves are not.
	var scopes []string
	for _, e := range got.History {
		scopes = append(scopes, e.Scope+"/"+e.Action)
		if e.Author == nil || e.Author.Username != "administrator" {
			t.Fatalf("history must name the author: %+v", e)
		}
	}
	want := []string{"clip/remove", "clip/set", "clip/disable"}
	if len(scopes) < 3 || fmt.Sprint(scopes[:3]) != fmt.Sprint(want) {
		t.Fatalf("clip history, newest first: %v", scopes)
	}
	if string(got.History[1].Old) == "null" || string(got.History[1].New) == "null" || string(got.History[2].Old) != "null" {
		t.Fatalf("history keeps old and new values: %+v", got.History[:3])
	}
	if h.count(`SELECT count(*) FROM attribution_events WHERE action='microphone_assign' AND new_person_name='Bob Micro'`) != 1 {
		t.Fatal("an attribution change is recorded in the speaker history")
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM microphone_track_events`); err == nil {
		t.Fatal("the designation history must not be deletable")
	}

	// A human validation outranks the designation and is never lost.
	speakers = h.speakers()
	speakerPath := fmt.Sprintf("/api/local-speakers/%d", speakers[0].ID)
	h.adminCall(http.MethodPost, speakerPath+"/validation", map[string]any{"person_id": bob}, http.StatusCreated)
	if speakers = h.speakers(); speakers[0].DecisionLevel != "human" || speakers[0].Person.ID != bob {
		t.Fatalf("a human validation outranks the microphone designation: %+v", speakers[0])
	}
	h.adminCall(http.MethodDelete, rulePath, nil, http.StatusOK)
	if speakers = h.speakers(); len(speakers) != 3 || speakers[0].DecisionLevel != "human" || speakers[0].Person.ID != bob {
		t.Fatalf("removing the rule must keep the validated speaker: %+v", speakers)
	}
	if got = h.micTrack(); got.Status != "none" || got.Applied {
		t.Fatalf("after removal no track is designated: %+v", got)
	}
	// Validated voices on a stream also stop a designation from replacing them.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", speakers[1].ID), map[string]any{"person_id": alice}, http.StatusCreated)
	h.adminCall(http.MethodPut, clipPath, map[string]any{"audio_stream_index": 2, "person_id": bob}, http.StatusOK)
	if got = h.micTrack(); got.Status != "blocked_by_validation" || len(h.speakers()) != 3 {
		t.Fatalf("a stream with human validations is never replaced: %+v", got)
	}

	// Anonymous callers read nothing.
	resp, err := http.Get(h.ts.URL + fmt.Sprintf("/api/clips/%d/microphone-track", h.clipID))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous read: %d", resp.StatusCode)
	}

	// Deleting the clip erases its microphone speaker and designation history.
	if _, err = pool.Exec(context.Background(), `DELETE FROM source_files WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `DELETE FROM clips WHERE id=$1`, h.clipID); err != nil {
		t.Fatalf("clip deletion must cascade: %v", err)
	}
	if h.count(`SELECT count(*) FROM local_speakers`)+h.count(`SELECT count(*) FROM clip_microphone_overrides`)+h.count(`SELECT count(*) FROM microphone_track_events WHERE scope='clip'`) != 0 {
		t.Fatal("clip deletion must erase microphone data")
	}
}
