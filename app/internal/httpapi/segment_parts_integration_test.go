//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type partView struct {
	SpeakerID  *int64  `json:"local_speaker_id"`
	Label      *string `json:"local_speaker_label"`
	PersonName *string `json:"local_speaker_person_name"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Words      []struct {
		Word string `json:"word"`
	} `json:"words"`
}

func (p partView) text() string {
	words := []string{}
	for _, w := range p.Words {
		words = append(words, w.Word)
	}
	return strings.Join(words, " ")
}

type partsTranscript struct {
	Transcription struct {
		Rule     string `json:"speaker_attribution_rule"`
		Segments []struct {
			Index            int        `json:"index"`
			Start            float64    `json:"start"`
			End              float64    `json:"end"`
			Text             string     `json:"text"`
			Corrected        bool       `json:"corrected"`
			Version          int        `json:"correction_version"`
			SpeakerLabel     *string    `json:"local_speaker_label"`
			SpeakerCorrected bool       `json:"local_speaker_corrected"`
			Parts            []partView `json:"parts"`
		} `json:"segments"`
		Coverage []struct {
			ID       int64   `json:"local_speaker_id"`
			Status   string  `json:"coverage_status"`
			Overlap  float64 `json:"overlapped_seconds"`
			Segments int     `json:"attributed_segments"`
		} `json:"speaker_coverage"`
	} `json:"transcription"`
}

// partsOf renders the parts of a segment as "LABEL: words | LABEL: words"; "-" is a part without speaker.
func partsOf(parts []partView) string {
	out := []string{}
	for _, p := range parts {
		label := "-"
		if p.Label != nil {
			label = *p.Label
		}
		out = append(out, label+": "+p.text())
	}
	return strings.Join(out, " | ")
}

// #398: the words of a segment go to the Local speakers that say them, as parts; segments, bounds and Corrections
// stay put, the Coverage and the Person filter follow the parts, and the per-segment rule is one setting away.
func testSegmentParts(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	admin := h.adminUser()
	h.legacyTranscription(2)
	job, ok := h.claim()
	if !ok || job.Step != "transcription.audio.2" {
		t.Fatalf("expected the transcription task, got %+v", job)
	}
	segment := func(start float64, words ...string) map[string]any {
		list := []map[string]any{}
		for i, w := range words {
			list = append(list, map[string]any{"word": w, "start": start + float64(i), "end": start + float64(i) + 1})
		}
		return map[string]any{"start": start, "end": start + float64(len(words)), "text": strings.Join(words, " "), "words": list}
	}
	h.complete(job, map[string]any{"provider": "fixture", "model": "deterministic-v1", "version": "deterministic-v1", "language": "fr", "audio_stream_index": 2, "segments": []map[string]any{
		segment(0, "bonjour", "toi", "salut", "vous"), // two voices answer each other
		segment(5, "alpha", "beta", "gamma"),          // one word of another voice between two words
		segment(9, "un", "deux", "trois"),             // a turn over the middle word only
		segment(13, "rien"),                           // no turn at all
		segment(15, "quatre", "cinq", "six", "sept"),  // two voices, then a text Correction
		segment(20, "huit", "neuf", "dix", "onze"),    // two voices, then a speaker Correction
	}}, http.StatusOK)
	h.putPipeline(fakePipeline(3))
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	job, ok = h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("expected the diarization task, got %+v", job)
	}
	turn := func(start, end float64) map[string]any { return map[string]any{"start": start, "end": end} }
	h.complete(job, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "audio_stream_index": 2, "speakers": []map[string]any{
		{"label": "SPEAKER_00", "turns": []map[string]any{turn(0, 2.2), turn(5, 6), turn(7, 8), turn(15, 17.2), turn(20, 22.2)}},
		// Never the voice that speaks most over a segment: the per-segment rule gives it no word at all.
		{"label": "SPEAKER_01", "turns": []map[string]any{turn(2.2, 4), turn(6, 7), turn(17.2, 19), turn(22.2, 24)}},
		{"label": "SPEAKER_02", "turns": []map[string]any{turn(10, 11)}},
	}}, http.StatusOK)
	ids := map[string]int64{}
	for _, s := range h.voice().Speakers {
		ids[s.Label] = s.ID
	}
	read := func() (partsTranscript, []byte) {
		t.Helper()
		var v partsTranscript
		body := h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK)
		if err := json.Unmarshal(body, &v); err != nil {
			t.Fatal(err)
		}
		return v, body
	}
	transcriptionOf := func(body []byte) any {
		var v struct {
			Transcription any `json:"transcription"`
		}
		if err := json.Unmarshal(body, &v); err != nil {
			t.Fatal(err)
		}
		return v.Transcription
	}
	expectParts := func(v partsTranscript, want ...string) {
		t.Helper()
		for i, expected := range want {
			if got := partsOf(v.Transcription.Segments[i].Parts); got != expected {
				t.Errorf("segment %d: parts %q, want %q", i, got, expected)
			}
		}
	}
	coverage := func(v partsTranscript, label string) (string, float64, int) {
		t.Helper()
		for _, c := range v.Transcription.Coverage {
			if c.ID == ids[label] {
				return c.Status, c.Overlap, c.Segments
			}
		}
		t.Fatalf("no coverage for %s", label)
		return "", 0, 0
	}
	type rule struct {
		Version int    `json:"version"`
		Rule    string `json:"rule"`
		History []struct {
			Version int     `json:"version"`
			Rule    string  `json:"rule"`
			Author  *string `json:"author"`
		} `json:"history"`
	}
	setRule := func(name string) rule {
		t.Helper()
		var out rule
		if err := json.Unmarshal(h.adminCall(http.MethodPut, "/api/admin/speaker-attribution-rule", map[string]any{"rule": name}, http.StatusOK), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	search := func(params string) personSearchPage {
		t.Helper()
		var page personSearchPage
		if err := json.Unmarshal(h.userCall(admin, http.MethodGet, "/api/search?"+params, nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		return page
	}

	// The word rule is the default one.
	var active rule
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/speaker-attribution-rule", nil, http.StatusOK), &active); err != nil {
		t.Fatal(err)
	}
	if active.Rule != "word" || active.Version != 1 || len(active.History) != 1 || active.History[0].Author != nil {
		t.Fatalf("the word rule is Snitch's default, version 1 without author: %+v", active)
	}
	h.adminCall(http.MethodPut, "/api/admin/speaker-attribution-rule", map[string]any{"rule": "sentence"}, http.StatusBadRequest)
	member := h.newUser("member")
	h.userCall(member, http.MethodGet, "/api/admin/speaker-attribution-rule", nil, http.StatusForbidden)

	byWord, _ := read()
	if byWord.Transcription.Rule != "word" {
		t.Fatalf("the transcription says which rule it was read with: %q", byWord.Transcription.Rule)
	}
	expectParts(byWord,
		"SPEAKER_00: bonjour toi | SPEAKER_01: salut vous",
		"SPEAKER_00: alpha beta gamma",
		"SPEAKER_02: un deux trois",
		"-: rien",
		"SPEAKER_00: quatre cinq | SPEAKER_01: six sept",
		"SPEAKER_00: huit neuf | SPEAKER_01: dix onze",
	)
	if p := byWord.Transcription.Segments[0].Parts; len(p) != 2 || p[0].Start != 0 || p[0].End != 2 || p[1].Start != 2 || p[1].End != 4 {
		t.Errorf("a part spans its words: %+v", p)
	}
	// A voice with words of its own is no longer counted as speaking under the words of others, except where it
	// holds no part (the single word of segment 1).
	if status, overlap, segments := coverage(byWord, "SPEAKER_01"); status != "words" || segments != 3 || math.Abs(overlap-1) > 1e-6 {
		t.Errorf("SPEAKER_01 holds words in three segments: %s, %v s overlapped, %d segments", status, overlap, segments)
	}
	if status, overlap, segments := coverage(byWord, "SPEAKER_00"); status != "words" || segments != 4 || overlap != 0 {
		t.Errorf("SPEAKER_00: %s, %v s overlapped, %d segments", status, overlap, segments)
	}

	// The Person filter keeps a segment as soon as one of its parts is attributed to the Person.
	bob := h.createPerson("Bob")
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", ids["SPEAKER_01"]), map[string]any{"person_id": bob}, http.StatusCreated)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}]}`))
	}))
	defer provider.Close()
	h.api.Config.EmbeddingsURL, h.api.Config.EmbeddingsModel = provider.URL, "fixture"
	var profile, transcription int64
	if err := pool.QueryRow(ctx, `INSERT INTO embedding_profiles(model,dimensions,state) VALUES('fixture',2,'active') RETURNING id`).Scan(&profile); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND status='done'`, h.clipID).Scan(&transcription); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=$1 WHERE id=1`, profile); err != nil {
		t.Fatal(err)
	}
	for _, s := range byWord.Transcription.Segments {
		if _, err := pool.Exec(ctx, `INSERT INTO transcript_embeddings(profile_id,transcription_job_id,clip_id,segment_index,segment_start,segment_end,text,embedding) VALUES($1,$2,$3,$4,$5,$6,$7,'[1,0]')`, profile, transcription, h.clipID, s.Index, s.Start, s.End, s.Text); err != nil {
			t.Fatal(err)
		}
	}
	contexts := func(page personSearchPage) map[string]string {
		out := map[string]string{}
		for _, item := range page.Items {
			name := ""
			if item.Person != nil {
				name = item.Person.DisplayName
			}
			out[item.Context] = name
		}
		return out
	}
	bobSegments := map[string]string{"bonjour toi salut vous": "Bob", "quatre cinq six sept": "Bob", "huit neuf dix onze": "Bob"}
	for _, mode := range []string{"exact", "full_text"} {
		// "bonjour" is said by SPEAKER_00, but its segment has a part of Bob.
		page := search(fmt.Sprintf("mode=%s&q=bonjour&person_id=%d", mode, bob))
		if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Person == nil || page.Items[0].Person.DisplayName != "Bob" {
			t.Errorf("%s: a segment with a part of Bob is kept by his filter and shown as his: %+v", mode, page)
		}
		if page = search(fmt.Sprintf("mode=%s&q=alpha&person_id=%d", mode, bob)); page.Total != 0 {
			t.Errorf("%s: a single word between two words of another voice is no part of Bob: %+v", mode, page)
		}
		// The unknown filter follows the parts too: a segment without speaker, and one with a part of an unknown voice.
		if page = search(fmt.Sprintf("mode=%s&q=rien&person_unknown=true", mode)); page.Total != 1 || page.Items[0].Person != nil {
			t.Errorf("%s: a segment without speaker is unknown: %+v", mode, page)
		}
		if page = search(fmt.Sprintf("mode=%s&q=salut&person_unknown=true", mode)); page.Total != 1 || page.Items[0].Person != nil {
			t.Errorf("%s: a segment with a part of an unknown voice is kept by the unknown filter: %+v", mode, page)
		}
		if page = search(fmt.Sprintf("mode=%s&q=bonjour", mode)); page.Total != 1 || page.Items[0].Person != nil {
			t.Errorf("%s: without filter the segment keeps the voice that speaks most over it: %+v", mode, page)
		}
	}
	if got := contexts(search(fmt.Sprintf("mode=hybrid&q=bonjour&person_id=%d", bob))); !reflect.DeepEqual(got, bobSegments) {
		t.Errorf("hybrid: the semantic candidates of Bob are the segments with a part of his: %v", got)
	}

	// Back to the per-segment rule: one setting, a new traced version, nothing rewritten.
	tables := []string{"jobs", "transcript_revisions", "segment_corrections", "segment_correction_events", "speaker_turns", "local_speakers", "speaker_attributions"}
	before := h.digests(tables...)
	changed := setRule("segment")
	if changed.Rule != "segment" || changed.Version != 2 || len(changed.History) != 2 || changed.History[0].Author == nil {
		t.Fatalf("a change of rule is a new version with its author: %+v", changed)
	}
	if again := setRule("segment"); again.Version != 2 {
		t.Fatalf("the active rule taken again records nothing: %+v", again)
	}
	if after := h.digests(tables...); !reflect.DeepEqual(after, before) {
		t.Fatalf("changing the rule rewrote data: %v -> %v", before, after)
	}
	bySegment, _ := read()
	if bySegment.Transcription.Rule != "segment" {
		t.Fatalf("rule read: %q", bySegment.Transcription.Rule)
	}
	expectParts(bySegment,
		"SPEAKER_00: bonjour toi salut vous",
		"SPEAKER_00: alpha beta gamma",
		"SPEAKER_02: un deux trois",
		"-: rien",
		"SPEAKER_00: quatre cinq six sept",
		"SPEAKER_00: huit neuf dix onze",
	)
	if len(bySegment.Transcription.Segments) != len(byWord.Transcription.Segments) {
		t.Fatal("the rule never changes the number of segments")
	}
	for i, s := range bySegment.Transcription.Segments {
		w := byWord.Transcription.Segments[i]
		if s.Index != w.Index || s.Start != w.Start || s.End != w.End || s.Text != w.Text || !reflect.DeepEqual(s.SpeakerLabel, w.SpeakerLabel) {
			t.Errorf("segment %d: identity, bounds, text and speaker of the segment do not depend on the rule: %+v / %+v", i, s, w)
		}
	}
	// The display of before the word rule: SPEAKER_01 speaks under the words of others, in every segment.
	if status, overlap, segments := coverage(bySegment, "SPEAKER_01"); status != "speech_without_words" || segments != 0 || math.Abs(overlap-6.4) > 1e-6 {
		t.Errorf("per segment, SPEAKER_01 has no word: %s, %v s overlapped, %d segments", status, overlap, segments)
	}
	for _, mode := range []string{"exact", "full_text"} {
		if page := search(fmt.Sprintf("mode=%s&q=bonjour&person_id=%d", mode, bob)); page.Total != 0 {
			t.Errorf("%s: per segment, Bob has no segment: %+v", mode, page)
		}
	}
	if got := contexts(search(fmt.Sprintf("mode=hybrid&q=bonjour&person_id=%d", bob))); len(got) != 0 {
		t.Errorf("hybrid: per segment, Bob has no segment: %v", got)
	}

	// Corrections made under the per-segment rule stay on their segment under the word rule (ADR 0012).
	h.userCall(admin, http.MethodPut, segmentTextPath(h.clipID, 4), map[string]any{"text": "quatre cinq six huit", "expected_version": 0}, http.StatusOK)
	h.userCall(admin, http.MethodPut, fmt.Sprintf("/api/clips/%d/segments/5/speaker", h.clipID), map[string]any{"local_speaker_id": ids["SPEAKER_02"], "expected_version": 0}, http.StatusOK)
	corrected, segmentBody := read()
	corrections := h.digests("segment_corrections", "segment_correction_events")
	if back := setRule("word"); back.Rule != "word" || back.Version != 3 {
		t.Fatalf("back to the word rule: %+v", back)
	}
	if after := h.digests("segment_corrections", "segment_correction_events"); !reflect.DeepEqual(after, corrections) {
		t.Fatal("a Correction was lost or moved by the change of rule")
	}
	final, _ := read()
	expectParts(final,
		"SPEAKER_00: bonjour toi | SPEAKER_01: salut vous",
		"SPEAKER_00: alpha beta gamma",
		"SPEAKER_02: un deux trois",
		"-: rien",
		"SPEAKER_00: ",                   // corrected text: a single speaker, the one of the per-segment rule, and no timed word
		"SPEAKER_02: huit neuf dix onze", // speaker Correction: the whole segment
	)
	for i, s := range final.Transcription.Segments {
		c := corrected.Transcription.Segments[i]
		if s.Text != c.Text || s.Corrected != c.Corrected || s.Version != c.Version || s.SpeakerCorrected != c.SpeakerCorrected || !reflect.DeepEqual(s.SpeakerLabel, c.SpeakerLabel) || s.Start != c.Start || s.End != c.End {
			t.Errorf("segment %d: its Corrections do not depend on the rule: %+v / %+v", i, s, c)
		}
	}
	if s := final.Transcription.Segments[4]; !s.Corrected || s.Text != "quatre cinq six huit" || len(s.Parts) != 1 {
		t.Errorf("a segment with a corrected text has one speaker: %+v", s)
	}
	if s := final.Transcription.Segments[5]; !s.SpeakerCorrected || len(s.Parts) != 1 {
		t.Errorf("a speaker Correction applies to the whole segment: %+v", s)
	}
	// Bob's filter no longer keeps the two corrected segments: each has a single speaker, who is not him.
	if page := search(fmt.Sprintf("mode=exact&q=huit&person_id=%d", bob)); page.Total != 0 {
		t.Errorf("a corrected segment has no part of Bob: %+v", page)
	}

	// And back again: exactly what the per-segment rule showed.
	setRule("segment")
	_, again := read()
	if !reflect.DeepEqual(transcriptionOf(again), transcriptionOf(segmentBody)) {
		t.Fatalf("the per-segment rule must give back exactly its display:\n%s\n%s", again, segmentBody)
	}
}
