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

type speakerSegment struct {
	Index       int    `json:"index"`
	Text        string `json:"text"`
	SpeakerID   int64  `json:"local_speaker_id"`
	Label       string `json:"local_speaker_label"`
	Level       string `json:"local_speaker_decision_level"`
	PersonName  string `json:"local_speaker_person_name"`
	Corrected   bool   `json:"local_speaker_corrected"`
	Version     int    `json:"correction_version"`
	TextChanged bool   `json:"corrected"`
}

type speakerTranscript struct {
	Transcription struct {
		JobID     int64            `json:"transcription_job_id"`
		Speakers  []map[string]any `json:"local_speakers"`
		Segments  []speakerSegment `json:"segments"`
		AudioStrm int              `json:"audio_stream_index"`
	} `json:"transcription"`
}

type speakerResponse struct {
	Error   string  `json:"error"`
	Version int     `json:"version"`
	ID      *int64  `json:"local_speaker_id"`
	Label   *string `json:"local_speaker_label"`
	Fixed   bool    `json:"corrected"`
	Current *struct {
		Version int `json:"version"`
	} `json:"current"`
}

// testSegmentSpeaker covers the speaker correction of a segment: override and return to
// the deduced speaker, immutability of the voice data, search and library filters,
// refusals, permissions, history and version conflicts.
func testSegmentSpeaker(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip() // SPEAKER_00 speaks 0-1.5 s, SPEAKER_01 speaks 1-2.5 s
	admin := h.adminUser()
	alice := h.newUser("alice")
	reader := h.newUser("reader")
	outsider := h.newUser("outsider")
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	var aliceP, bobP struct {
		ID int64 `json:"id"`
	}
	for name, dest := range map[string]*struct {
		ID int64 `json:"id"`
	}{"Alice": &aliceP, "Bob": &bobP} {
		if err := json.Unmarshal(h.userCall(alice, http.MethodPost, "/api/persons", map[string]any{"display_name": name}, http.StatusCreated), dest); err != nil {
			t.Fatal(err)
		}
	}
	h.userCall(alice, http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", first), map[string]any{"person_id": aliceP.ID}, http.StatusCreated)
	h.userCall(alice, http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", second), map[string]any{"person_id": bobP.ID}, http.StatusCreated)

	transcript := func(u *voiceUser) speakerTranscript {
		t.Helper()
		var v speakerTranscript
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	path := func(index int) string { return fmt.Sprintf("/api/clips/%d/segments/%d/speaker", h.clipID, index) }
	put := func(u *voiceUser, index int, body map[string]any, status int) speakerResponse {
		t.Helper()
		var v speakerResponse
		if err := json.Unmarshal(h.userCall(u, http.MethodPut, path(index), body, status), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	snapshot := func() string {
		t.Helper()
		var out string
		for _, table := range []string{"speaker_turns", "voice_fingerprints", "speaker_attributions", "voice_samples", "attribution_events", "local_speakers"} {
			var digest string
			if err := pool.QueryRow(ctx, fmt.Sprintf(`SELECT md5(COALESCE(string_agg(x::text,'|' ORDER BY x::text),'')) FROM %s x`, table)).Scan(&digest); err != nil {
				t.Fatal(err)
			}
			out += table + "=" + digest + ";"
		}
		return out
	}
	searchTotal := func(q string, person int64) int {
		t.Helper()
		totals := []int{}
		for _, mode := range []string{"exact", "full_text"} {
			var page struct {
				Total int `json:"total"`
			}
			if err := json.Unmarshal(h.userCall(reader, http.MethodGet, fmt.Sprintf("/api/search?mode=%s&q=%s&person_id=%d", mode, q, person), nil, http.StatusOK), &page); err != nil {
				t.Fatal(err)
			}
			totals = append(totals, page.Total)
		}
		if totals[0] != totals[1] {
			t.Fatalf("search modes disagree: %v", totals)
		}
		return totals[0]
	}

	// Deduced speakers: segment 0 is SPEAKER_00, segment 1 SPEAKER_01, segment 2 nobody.
	before := transcript(admin)
	segs := before.Transcription.Segments
	if len(before.Transcription.Speakers) != 2 || segs[0].SpeakerID != first || segs[1].SpeakerID != second || segs[2].SpeakerID != 0 || segs[0].Corrected || segs[1].Corrected {
		t.Fatalf("deduced speakers and roster expected: %+v", before.Transcription)
	}
	if searchTotal("premier", aliceP.ID) != 1 || searchTotal("premier", bobP.ID) != 0 {
		t.Fatal("segment 0 starts as Alice's")
	}

	// Refusals write nothing.
	anon := &voiceUser{name: "anonymous", client: &http.Client{}}
	if status, _ := h.userStatus(anon, http.MethodPut, path(0), map[string]any{"local_speaker_id": second, "expected_version": 0}); status != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", status)
	}
	var otherClipID int64
	if err := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) SELECT owner_id,'other-synthetic.mp4',1,now(),now() FROM clips WHERE id=$1 RETURNING id`, h.clipID).Scan(&otherClipID); err != nil {
		t.Fatal(err)
	}
	foreign := h.fixtureSpeaker(otherClipID, "SPEAKER_00", usableAt(0))
	otherStream := h.fixtureSpeaker(h.clipID, "SPEAKER_09", usableAt(10)) // same clip, other audio stream
	voiceData, jobs := snapshot(), h.count(`SELECT count(*) FROM jobs`)
	put(admin, 0, map[string]any{"local_speaker_id": foreign.ID, "expected_version": 0}, http.StatusBadRequest)
	put(admin, 0, map[string]any{"local_speaker_id": otherStream.ID, "expected_version": 0}, http.StatusBadRequest)
	put(admin, 0, map[string]any{"local_speaker_id": 999999, "expected_version": 0}, http.StatusBadRequest)
	put(admin, 0, map[string]any{"expected_version": 0}, http.StatusBadRequest)
	put(admin, 0, map[string]any{"local_speaker_id": 0, "expected_version": 0}, http.StatusBadRequest)
	put(admin, 0, map[string]any{"local_speaker_id": second}, http.StatusBadRequest)
	put(admin, 0, map[string]any{"local_speaker_id": second, "expected_version": -1}, http.StatusBadRequest)
	put(admin, 0, map[string]any{"local_speaker_id": second, "expected_version": 0, "reason": fmt.Sprintf("%0501d", 0)}, http.StatusBadRequest)
	put(admin, 0, map[string]any{"local_speaker_id": nil, "expected_version": 0}, http.StatusBadRequest) // already deduced: unchanged
	put(admin, 9, map[string]any{"local_speaker_id": second, "expected_version": 0}, http.StatusNotFound)
	if n := h.count(`SELECT count(*) FROM segment_corrections`); n != 0 || h.count(`SELECT count(*) FROM segment_correction_events`) != 0 {
		t.Fatalf("refused corrections must write nothing, got %d rows", n)
	}
	// An invisible clip is not found.
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	put(outsider, 0, map[string]any{"local_speaker_id": second, "expected_version": 0}, http.StatusNotFound)
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}

	// Designate SPEAKER_01 (Bob) for segment 0.
	done := put(reader, 0, map[string]any{"local_speaker_id": second, "expected_version": 0, "reason": "mauvaise diarisation", "transcription_job_id": before.Transcription.JobID}, http.StatusOK)
	if done.Version != 1 || done.ID == nil || *done.ID != second || !done.Fixed || done.Label == nil || *done.Label != "SPEAKER_01" {
		t.Fatalf("speaker correction: %+v", done)
	}
	after := transcript(admin).Transcription.Segments
	if after[0].SpeakerID != second || !after[0].Corrected || after[0].PersonName != "Bob" || after[0].Level != "human" || after[0].Version != 1 || after[0].Text != segs[0].Text || after[1].SpeakerID != second || after[1].Corrected {
		t.Fatalf("display must use the corrected speaker: %+v", after)
	}
	if searchTotal("premier", bobP.ID) != 1 || searchTotal("premier", aliceP.ID) != 0 {
		t.Fatal("search filter must use the corrected speaker")
	}
	if snapshot() != voiceData || h.count(`SELECT count(*) FROM jobs`) != jobs {
		t.Fatal("turns, fingerprints, attributions, samples and jobs must be unchanged and nothing relaunched")
	}
	if h.count(`SELECT count(*) FROM segment_corrections WHERE reindex_pending`) != 0 {
		t.Fatal("a speaker correction must not request a semantic reindexing")
	}
	// The library still lists the clip for both Persons (clip-level reading) and the
	// unknown filter follows the corrected segment rule in search.
	var library libraryPage
	if err := json.Unmarshal(h.userCall(reader, http.MethodGet, fmt.Sprintf("/api/clips?person_id=%d", bobP.ID), nil, http.StatusOK), &library); err != nil || library.Total != 1 {
		t.Fatalf("library filter: %v %+v", err, library)
	}

	// Conflicts: a stale version is refused with the current state; a retranscription
	// is detected through the job id.
	conflict := put(admin, 0, map[string]any{"local_speaker_id": first, "expected_version": 0}, http.StatusConflict)
	if conflict.Error != "segment_conflict" || conflict.Current == nil || conflict.Current.Version != 1 {
		t.Fatalf("conflict: %+v", conflict)
	}
	if after = transcript(admin).Transcription.Segments; after[0].SpeakerID != second {
		t.Fatal("a refused correction must change nothing")
	}
	if c := put(admin, 0, map[string]any{"local_speaker_id": first, "expected_version": 1, "transcription_job_id": before.Transcription.JobID + 1000}, http.StatusConflict); c.Error != "transcription_changed" {
		t.Fatalf("transcription_changed expected: %+v", c)
	}
	put(admin, 0, map[string]any{"local_speaker_id": second, "expected_version": 1}, http.StatusBadRequest) // unchanged

	// A text correction keeps the speaker; the version continues across both kinds.
	h.userCall(admin, http.MethodPut, segmentTextPath(h.clipID, 0), map[string]any{"text": "premier corrigé", "expected_version": 1}, http.StatusOK)
	if s := transcript(admin).Transcription.Segments[0]; s.SpeakerID != second || !s.Corrected || !s.TextChanged || s.Version != 2 {
		t.Fatalf("text correction must keep the speaker override: %+v", s)
	}

	// Return to the deduced speaker.
	back := put(admin, 0, map[string]any{"local_speaker_id": nil, "expected_version": 2}, http.StatusOK)
	if back.Version != 3 || back.Fixed || back.ID == nil || *back.ID != first {
		t.Fatalf("return to the deduced speaker: %+v", back)
	}
	if s := transcript(admin).Transcription.Segments[0]; s.SpeakerID != first || s.Corrected || s.PersonName != "Alice" || !s.TextChanged {
		t.Fatalf("deduced speaker expected, text kept: %+v", s)
	}
	if searchTotal("premier", aliceP.ID) != 1 || searchTotal("premier", bobP.ID) != 0 {
		t.Fatal("search filter must follow the return to the deduced speaker")
	}
	// A segment without deduced speaker can be given one.
	put(admin, 2, map[string]any{"local_speaker_id": first, "expected_version": 0}, http.StatusOK)
	if s := transcript(admin).Transcription.Segments[2]; s.SpeakerID != first || !s.Corrected {
		t.Fatalf("a segment without deduced speaker can be corrected: %+v", s)
	}
	if snapshot() != voiceData || h.count(`SELECT count(*) FROM jobs`) != jobs {
		t.Fatal("voice data must stay unchanged after every correction")
	}

	// The history keeps old and new values, author and date; speaker entries are not
	// restorable.
	var history historyResponse
	if err := json.Unmarshal(h.userCall(reader, http.MethodGet, fmt.Sprintf("/api/clips/%d/segments/0/history", h.clipID), nil, http.StatusOK), &history); err != nil {
		t.Fatal(err)
	}
	if len(history.Items) != 3 || history.Items[0].Kind != "speaker" || history.Items[1].Kind != "text" || history.Items[2].Kind != "speaker" {
		t.Fatalf("history: %+v", history.Items)
	}
	var labels struct {
		Items []struct {
			Kind     string `json:"kind"`
			OldLabel string `json:"old_speaker_label"`
			NewLabel string `json:"new_speaker_label"`
			Reason   string `json:"reason"`
			Author   struct {
				Username string `json:"username"`
			} `json:"author"`
			CreatedAt string `json:"created_at"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.userCall(reader, http.MethodGet, fmt.Sprintf("/api/clips/%d/segments/0/history", h.clipID), nil, http.StatusOK), &labels); err != nil {
		t.Fatal(err)
	}
	oldest, newest := labels.Items[2], labels.Items[0]
	if oldest.OldLabel != "SPEAKER_00" || oldest.NewLabel != "SPEAKER_01" || oldest.Author.Username != "reader" || oldest.Reason != "mauvaise diarisation" || oldest.CreatedAt == "" {
		t.Fatalf("first speaker entry: %+v", oldest)
	}
	if newest.OldLabel != "SPEAKER_01" || newest.NewLabel != "SPEAKER_00" || newest.Author.Username != "administrator" {
		t.Fatalf("return entry: %+v", newest)
	}
	h.userCall(admin, http.MethodPost, fmt.Sprintf("/api/clips/%d/segments/0/restore", h.clipID), map[string]any{"event_id": history.Items[2].ID, "expected_version": 3}, http.StatusBadRequest)
	if _, err := pool.Exec(ctx, `UPDATE segment_correction_events SET reason='x'`); err == nil {
		t.Fatal("history entries are immutable")
	}
}
