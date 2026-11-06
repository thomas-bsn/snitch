//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type correctedSearchPage struct {
	Items []struct {
		searchItem
	} `json:"items"`
	Total    int  `json:"total"`
	Degraded bool `json:"degraded"`
}

// testSearchCorrections checks that the four search modes read the effective text of
// a segment: the corrected text is found and flagged, the replaced text is not found.
func testSearchCorrections(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	_, second := h.diarizeClip() // SPEAKER_01 speaks 1-2.5 s, over the second segment
	alice, reader := h.newUser("alice"), h.newUser("reader")
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}]}`))
	}))
	defer provider.Close()
	h.api.Config.EmbeddingsURL, h.api.Config.EmbeddingsModel = provider.URL, "fixture"
	var profile, job int64
	if err := pool.QueryRow(ctx, `INSERT INTO embedding_profiles(model,dimensions,state) VALUES('fixture',2,'active') RETURNING id`).Scan(&profile); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND status='done'`, h.clipID).Scan(&job); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=$1 WHERE id=1`, profile); err != nil {
		t.Fatal(err)
	}
	for i, seg := range []struct {
		start, end float64
		text       string
	}{{0.2, 0.9, "premier"}, {1.25, 2.1, "second"}, {3.9, 4.1, "ailleurs"}} {
		if _, err := pool.Exec(ctx, `INSERT INTO transcript_embeddings(profile_id,transcription_job_id,clip_id,segment_index,segment_start,segment_end,text,embedding) VALUES($1,$2,$3,$4,$5,$6,$7,'[1,0]')`, profile, job, h.clipID, i, seg.start, seg.end, seg.text); err != nil {
			t.Fatal(err)
		}
	}
	search := func(u *voiceUser, mode, q, extra string) correctedSearchPage {
		t.Helper()
		var page correctedSearchPage
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/search?mode="+mode+"&q="+url.QueryEscape(q)+extra, nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		return page
	}
	contexts := func(page correctedSearchPage) []string {
		out := []string{}
		for _, it := range page.Items {
			out = append(out, it.Context)
		}
		return out
	}
	modes := []string{"exact", "full_text", "hybrid"}
	corrected := "« troisième chose ! »"

	// Before any correction the original text is found and not flagged.
	for _, mode := range modes {
		page := search(reader, mode, "second", "")
		if page.Total < 1 || page.Items[0].Context != "second" || page.Items[0].Corrected || page.Items[0].Timestamp != 1.25 {
			t.Fatalf("%s: original text before correction: %+v", mode, page)
		}
	}

	h.userCall(alice, http.MethodPut, segmentTextPath(h.clipID, 1), map[string]any{"text": corrected, "expected_version": 0}, http.StatusOK)

	for _, mode := range modes {
		// The replaced text is no longer found, whatever the mode.
		page := search(reader, mode, "second", "")
		for _, c := range contexts(page) {
			if c == "second" {
				t.Fatalf("%s: the replaced text must not be found: %+v", mode, contexts(page))
			}
		}
		if mode != "hybrid" && page.Total != 0 {
			t.Fatalf("%s: the replaced word must produce no result: %+v", mode, page)
		}
		// The corrected text is found (accent-insensitive), flagged, at the segment start.
		page = search(reader, mode, "troisieme", "")
		hit := -1
		for i, it := range page.Items {
			if it.Context == corrected {
				hit = i
			}
		}
		if hit < 0 || !page.Items[hit].Corrected || page.Items[hit].Timestamp != 1.25 {
			t.Fatalf("%s: corrected text: %+v", mode, page)
		}
		if mode == "exact" && (page.Total != 1 || page.Items[hit].MatchedText != "troisième" || page.Items[hit].MatchType != "exact") {
			t.Fatalf("exact: the word is read from the corrected text, punctuation stripped: %+v", page)
		}
	}
	// Exact search takes a phrase: its words follow each other, punctuation is ignored.
	if page := search(reader, "exact", "troisieme chose", ""); page.Total != 1 || page.Items[0].MatchType != "exact" || !page.Items[0].Corrected || page.Items[0].Timestamp != 1.25 {
		t.Fatalf("exact phrase: %+v", page)
	}
	// Phrase (full text) and approximate matches use the corrected text only.
	if page := search(reader, "full_text", "troisieme chose", ""); page.Total != 1 || page.Items[0].MatchType != "full_text" || !page.Items[0].Corrected {
		t.Fatalf("full text phrase: %+v", page)
	}
	if page := search(reader, "full_text", "troisiemes", ""); page.Total != 1 || page.Items[0].MatchType != "approximate" || !page.Items[0].Corrected {
		t.Fatalf("approximate: %+v", page)
	}
	if page := search(reader, "full_text", "seconds", ""); page.Total != 0 {
		t.Fatalf("approximate must not match the replaced word: %+v", page)
	}
	// Other segments stay uncorrected.
	if page := search(reader, "exact", "premier", ""); page.Total != 1 || page.Items[0].Corrected {
		t.Fatalf("uncorrected segment: %+v", page)
	}

	// Semantic voice: the stored embedding text is the old one. While the segment waits
	// for reindexing it is not offered; once flagged as indexed it shows the effective text.
	page := search(reader, "hybrid", "zzz", "")
	for _, it := range page.Items {
		if it.Context == "second" || it.Context == corrected {
			t.Fatalf("a segment awaiting reindexing must not surface through its old embedding: %+v", contexts(page))
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE segment_corrections SET reindex_pending=false WHERE transcription_job_id=$1`, job); err != nil {
		t.Fatal(err)
	}
	page = search(reader, "hybrid", "zzz", "")
	found := false
	for _, it := range page.Items {
		if it.Context == "second" {
			t.Fatalf("the old text must never be displayed: %+v", contexts(page))
		}
		if it.Context == corrected && it.Corrected && it.MatchType == "semantic" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the semantic result must display the effective text: %+v", contexts(page))
	}

	// Person filter and other filters keep working with corrected segments.
	var person struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(h.userCall(alice, http.MethodPost, "/api/persons", map[string]any{"display_name": "Alice"}, http.StatusCreated), &person); err != nil {
		t.Fatal(err)
	}
	h.userCall(alice, http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", second), map[string]any{"person_id": person.ID}, http.StatusCreated)
	for _, mode := range modes {
		page = search(reader, mode, "troisieme", fmt.Sprintf("&person_id=%d", person.ID))
		if page.Total != 1 || page.Items[0].Person == nil || page.Items[0].Person.DisplayName != "Alice" || !page.Items[0].Corrected {
			t.Fatalf("%s: person filter on a corrected segment: %+v", mode, page)
		}
	}
	if page = search(reader, "exact", "troisieme", "&status=missing"); page.Total != 0 {
		t.Fatalf("status filter must apply: %+v", page)
	}

	// Permissions apply before the count: a private clip reveals nothing to others,
	// corrected text included.
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	for _, mode := range modes {
		if page = search(reader, mode, "troisieme", ""); page.Total != 0 || len(page.Items) != 0 {
			t.Fatalf("%s: a private corrected segment leaked: %+v", mode, page)
		}
	}
	if page = search(h.adminUser(), "exact", "troisieme", ""); page.Total != 1 {
		t.Fatalf("the admin sees the private clip: %+v", page)
	}
}
