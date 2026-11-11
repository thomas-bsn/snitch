//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type personReferenceView struct {
	Level               string  `json:"level"`
	Basis               string  `json:"basis"`
	ContributiveSamples int     `json:"contributive_samples"`
	Clips               int     `json:"clips"`
	NetSpeechSeconds    float64 `json:"net_speech_seconds"`
	Discarded           []struct {
		Reason string `json:"reason"`
		Count  int    `json:"count"`
	} `json:"discarded"`
}

type personReferenceRead struct {
	ID                   int64                `json:"id"`
	DisplayName          string               `json:"display_name"`
	ValidatedSampleCount int                  `json:"validated_sample_count"`
	Reference            *personReferenceView `json:"reference"`
}

// bareVoice inserts a local speaker with no fingerprint at all: a validated voice the Reference cannot use.
func (h *voiceHarness) bareVoice(clipID int64, label string) fixtureSpeaker {
	h.t.Helper()
	ctx := context.Background()
	var diarization, speaker int64
	if err := h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.9',$2,'done') RETURNING id`, clipID, "bare-d-"+label).Scan(&diarization); err != nil {
		h.t.Fatal(err)
	}
	if err := h.pool.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,9,$2,$3,0,3,1) RETURNING id`, clipID, diarization, label).Scan(&speaker); err != nil {
		h.t.Fatal(err)
	}
	return fixtureSpeaker{ID: speaker, ClipID: clipID}
}

func testPersonReferenceLevel(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 6)
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	c := h.clipIDs
	axis := func(i int) []float64 {
		v := []float64{0.05, 0.05, 0.05, 0.05}
		v[i%4] = 1
		return v
	}
	// person validates the voices of one Person, created by the first one.
	person := func(name string, voices ...fixtureSpeaker) int64 {
		t.Helper()
		h.validate(voices[0], name)
		id := h.personOf(voices[0])
		for _, v := range voices[1:] {
			h.validate(v, id)
		}
		return id
	}
	list := func(u *voiceUser, query string) map[string]personReferenceRead {
		t.Helper()
		var raw []byte
		if u == nil {
			raw = h.adminCall(http.MethodGet, "/api/persons"+query, nil, http.StatusOK)
		} else {
			raw = h.userCall(u, http.MethodGet, "/api/persons"+query, nil, http.StatusOK)
		}
		var out struct {
			Items []personReferenceRead `json:"items"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		byName := map[string]personReferenceRead{}
		for _, p := range out.Items {
			byName[p.DisplayName] = p
		}
		return byName
	}
	record := func(id int64) personReferenceRead {
		t.Helper()
		var out personReferenceRead
		if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/persons/%d", id), nil, http.StatusOK), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	expect := func(p personReferenceRead, level, basis string, samples, clips int, speech float64) {
		t.Helper()
		r := p.Reference
		if r == nil || r.Level != level || r.Basis != basis || r.ContributiveSamples != samples || r.Clips != clips || math.Abs(r.NetSpeechSeconds-speech) > 1e-6 {
			t.Fatalf("%s: want %s (%s) %d samples on %d clips, %.1f s; got %+v", p.DisplayName, level, basis, samples, clips, speech, r)
		}
	}

	// Two References only: no comparison, two distinct clips are enough.
	ann := person("Ann", h.shortVoice(c[0], "ANN1", shortVoice{vector: axis(0), net: 2}), h.shortVoice(c[1], "ANN2", shortVoice{vector: axis(0), net: 2}))
	ben := person("Ben",
		h.shortVoice(c[0], "BEN1", shortVoice{vector: axis(1), net: 4}),
		h.shortVoice(c[1], "BEN2", shortVoice{vector: axis(1), net: 4}),
		h.shortVoice(c[2], "BEN3", shortVoice{vector: axis(1), net: 4}))
	expect(list(nil, "")["Ann"], "good", "few_references", 2, 2, 4)

	// From three References, each one is compared with the median net speech of the group. Cid's 15 s clips
	// count 10 s each (cap per sample); Dee has one clip of 30 s; Eve has three validations, one contributive.
	cid := person("Cid", h.shortVoice(c[2], "CID1", shortVoice{vector: axis(2), net: 15}), h.shortVoice(c[3], "CID2", shortVoice{vector: axis(2), net: 15}))
	dee := person("Dee", h.shortVoice(c[3], "DEE1", shortVoice{vector: axis(3), net: 30}))
	eve := person("Eve",
		h.shortVoice(c[4], "EVE1", shortVoice{vector: []float64{1, 1, 0, 0}, net: 2}),
		h.shortVoice(c[4], "EVE2", shortVoice{vector: []float64{1, 1, 0, 0}, net: 0.5}),
		h.shortVoice(c[5], "EVE3", shortVoice{vector: []float64{1, 1, 0, 0}, net: 0.5}))
	// Fay: a voice too short to contribute, and a long one in another fingerprint group only.
	fay := person("Fay",
		h.shortVoice(c[5], "FAY1", shortVoice{vector: []float64{1, 0, 1, 0}, net: 0.5}),
		h.shortVoice(c[5], "FAY2", shortVoice{vector: []float64{1, 0, 1, 0}, net: 9, version: "emb-v2"}))
	// Gus: a validated voice without any fingerprint.
	gus := person("Gus", h.bareVoice(c[5], "GUS1"))

	// References: Eve 2, Ann 4, Dee 10, Ben 12, Cid 20 s: median 10 s.
	all := list(nil, "")
	expect(all["Ann"], "weak", "below_median", 2, 2, 4)
	expect(all["Ben"], "good", "median_or_more", 3, 3, 12)
	expect(all["Cid"], "good", "median_or_more", 2, 2, 20)
	expect(all["Dee"], "weak", "single_clip", 1, 1, 10)
	expect(all["Eve"], "weak", "single_clip", 1, 1, 2)
	expect(all["Fay"], "none", "no_sample", 0, 0, 0)
	expect(all["Gus"], "none", "no_sample", 0, 0, 0)
	if all["Eve"].ValidatedSampleCount != 3 {
		t.Fatalf("Eve has three validations: %+v", all["Eve"])
	}

	// The median does not depend on the page, nor on the search.
	for _, query := range []string{"?page=1&page_size=1&q=Ann", "?q=" + url.QueryEscape("Ann")} {
		expect(list(nil, query)["Ann"], "weak", "below_median", 2, 2, 4)
	}
	// Nor on what the reader can see: a member who sees none of the clips gets the same levels.
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='private'`); err != nil {
		t.Fatal(err)
	}
	member := list(h.newUser("reference-reader"), "")
	expect(member["Ann"], "weak", "below_median", 2, 2, 4)
	expect(member["Ben"], "good", "median_or_more", 3, 3, 12)
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='shared'`); err != nil {
		t.Fatal(err)
	}

	// The record gives the same level, and why samples are set aside.
	discarded := func(p personReferenceRead) map[string]int {
		out := map[string]int{}
		for _, d := range p.Reference.Discarded {
			out[d.Reason] = d.Count
		}
		return out
	}
	expect(record(ben), "good", "median_or_more", 3, 3, 12)
	expect(record(cid), "good", "median_or_more", 2, 2, 20)
	expect(record(dee), "weak", "single_clip", 1, 1, 10)
	e := record(eve)
	expect(e, "weak", "single_clip", 1, 1, 2)
	if got := discarded(e); len(got) != 1 || got["too_short"] != 2 {
		t.Fatalf("Eve's two short voices are set aside as too short: %+v", got)
	}
	if got := discarded(record(fay)); len(got) != 2 || got["too_short"] != 1 || got["no_fingerprint"] != 1 {
		t.Fatalf("Fay: one too short, one without a fingerprint of the active group: %+v", got)
	}
	if got := discarded(record(gus)); len(got) != 1 || got["no_fingerprint"] != 1 {
		t.Fatalf("Gus: no fingerprint: %+v", got)
	}

	// Each voice of the record says what the Reference makes of it.
	var voices struct {
		Items []struct {
			Label        string  `json:"label"`
			ReferenceUse *string `json:"reference_use"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/persons/%d/voices", eve), nil, http.StatusOK), &voices); err != nil {
		t.Fatal(err)
	}
	uses := map[string]string{}
	for _, v := range voices.Items {
		if v.ReferenceUse == nil {
			t.Fatalf("a validated voice with a sample has a use: %+v", v)
		}
		uses[v.Label] = *v.ReferenceUse
	}
	if uses["EVE1"] != "contributive" || uses["EVE2"] != "too_short" || uses["EVE3"] != "too_short" {
		t.Fatalf("unexpected uses of Eve's voices: %+v", uses)
	}

	// An automatic attribution changes no Reference, hence no level.
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=true`); err != nil {
		t.Fatal(err)
	}
	h.applyThresholds(fixtureGroup, 0.90, 0.75) // a group decides automatically only with applied thresholds (#393)
	stray := h.shortVoice(c[5], "STRAY", shortVoice{vector: axis(0), net: 8})
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)
	if got := h.matched(stray); got.DecisionLevel != "automatic" || got.Person == nil || got.Person.ID != ann {
		t.Fatalf("STRAY must be attributed automatically to Ann: %+v", got)
	}
	expect(record(ann), "weak", "below_median", 2, 2, 4)
}
