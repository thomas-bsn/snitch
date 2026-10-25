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

type personSearchItem struct {
	searchItem
	Person *struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"person"`
	DecisionLevel string `json:"decision_level"`
}

type personSearchPage struct {
	Items    []personSearchItem `json:"items"`
	Total    int                `json:"total"`
	Degraded bool               `json:"degraded"`
}

type libraryPage struct {
	Items []struct {
		ID      int64 `json:"id"`
		Persons []struct {
			ID            int64  `json:"id"`
			DisplayName   string `json:"display_name"`
			DecisionLevel string `json:"decision_level"`
		} `json:"persons"`
	} `json:"items"`
	Total int `json:"total"`
}

// testPersonFilter covers ticket 39: the Person filter of search (four modes) and
// of the library, driven only by the traced attributions of local speakers.
func testPersonFilter(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip() // SPEAKER_00 speaks 0-1.5 s, SPEAKER_01 speaks 1-2.5 s
	alice := h.newUser("alice")
	reader := h.newUser("reader")
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}

	// Hybrid mode needs an embedding index: a synthetic provider and indexed segments.
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
	for i, seg := range []struct {
		start, end float64
		text       string
	}{{0.2, 0.9, "premier"}, {1.25, 2.1, "second"}, {3.9, 4.1, "ailleurs"}} {
		if _, err := pool.Exec(ctx, `INSERT INTO transcript_embeddings(profile_id,transcription_job_id,clip_id,segment_index,segment_start,segment_end,text,embedding) VALUES($1,$2,$3,$4,$5,$6,$7,'[1,0]')`, profile, transcription, h.clipID, i, seg.start, seg.end, seg.text); err != nil {
			t.Fatal(err)
		}
	}

	search := func(u *voiceUser, want int, params string) personSearchPage {
		t.Helper()
		var page personSearchPage
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/search?"+params, nil, want), &page); err != nil {
			t.Fatal(err)
		}
		return page
	}
	var alicePerson, bobPerson, carolPerson struct {
		ID int64 `json:"id"`
	}
	for name, dest := range map[string]*struct {
		ID int64 `json:"id"`
	}{"Alice": &alicePerson, "Bob": &bobPerson, "Carol": &carolPerson} {
		if err := json.Unmarshal(h.userCall(alice, http.MethodPost, "/api/persons", map[string]any{"display_name": name}, http.StatusCreated), dest); err != nil {
			t.Fatal(err)
		}
	}
	modes := []string{"exact", "full_text", "hybrid"}
	broad := func(mode, extra string) string { return "mode=" + mode + "&q=" + url.QueryEscape("e") + extra }

	// Nothing is attributed yet: every segment is unknown and no Person matches.
	for _, mode := range modes {
		page := search(reader, 200, "mode="+mode+"&q=premier")
		if len(page.Items) == 0 || page.Items[0].Person != nil || page.Items[0].DecisionLevel != "unknown" {
			t.Fatalf("%s: an unattributed segment must be unknown: %+v", mode, page)
		}
		if got := search(reader, 200, fmt.Sprintf("mode=%s&q=premier&person_id=%d", mode, alicePerson.ID)); got.Total != 0 || len(got.Items) != 0 {
			t.Fatalf("%s: the filter must not recognise a voice without a traced attribution: %+v", mode, got)
		}
	}

	// Human validation of SPEAKER_00, automatic machine attribution of SPEAKER_01.
	h.userCall(alice, http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", first), map[string]any{"person_id": alicePerson.ID}, http.StatusCreated)
	if _, err := pool.Exec(ctx, `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,confidence,origin) VALUES($1,$2,'automatic','unvalidated',0.97,'synthetic_origin') ON CONFLICT (local_speaker_id) WHERE validation_state='unvalidated' DO UPDATE SET person_id=EXCLUDED.person_id,decision_level=EXCLUDED.decision_level,confidence=EXCLUDED.confidence,origin=EXCLUDED.origin`, second, bobPerson.ID); err != nil {
		t.Fatal(err)
	}

	for _, mode := range modes {
		// Word queries: exact, full-text and hybrid all honour the filter.
		page := search(reader, 200, fmt.Sprintf("mode=%s&q=premier&person_id=%d", mode, alicePerson.ID))
		if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Person == nil || page.Items[0].Person.DisplayName != "Alice" || page.Items[0].DecisionLevel != "human" || page.Items[0].Timestamp != 0.2 {
			t.Fatalf("%s: Alice's segment must be found with its person, level and timestamp: %+v", mode, page)
		}
		page = search(reader, 200, fmt.Sprintf("mode=%s&q=second&person_id=%d", mode, bobPerson.ID))
		if page.Total != 1 || page.Items[0].Person == nil || page.Items[0].Person.DisplayName != "Bob" || page.Items[0].DecisionLevel != "automatic" || page.Items[0].Timestamp != 1.25 {
			t.Fatalf("%s: an automatic attribution must be found and traced as such: %+v", mode, page)
		}
		// Semantic candidates need no word match, so hybrid may return other segments of
		// Alice, but never a segment of another voice.
		page = search(reader, 200, fmt.Sprintf("mode=%s&q=second&person_id=%d", mode, alicePerson.ID))
		for _, item := range page.Items {
			if item.Person == nil || item.Person.DisplayName != "Alice" || item.Context == "second" {
				t.Fatalf("%s: another voice's segment leaked: %+v", mode, page)
			}
		}
		if mode != "hybrid" && page.Total != 0 {
			t.Fatalf("%s: another Person's segment counted: %+v", mode, page)
		}
		if page = search(reader, 200, fmt.Sprintf("mode=%s&q=premier&person_id=%d", mode, carolPerson.ID)); page.Total != 0 {
			t.Fatalf("%s: a Person without voice matched: %+v", mode, page)
		}
	}
	for _, mode := range []string{"full_text", "hybrid"} {
		// A broad query matches all three segments; counts and pages follow the filter.
		all := search(reader, 200, broad(mode, ""))
		if all.Total != 3 {
			t.Fatalf("%s: three segments expected without filter: %+v", mode, all)
		}
		onlyAlice := search(reader, 200, broad(mode, fmt.Sprintf("&person_id=%d&page_size=1", alicePerson.ID)))
		if onlyAlice.Total != 1 || len(onlyAlice.Items) != 1 || onlyAlice.Items[0].Context != "premier" {
			t.Fatalf("%s: total and page must be computed after the Person filter: %+v", mode, onlyAlice)
		}
		if page2 := search(reader, 200, broad(mode, fmt.Sprintf("&person_id=%d&page_size=1&page=2", alicePerson.ID))); page2.Total != 1 || len(page2.Items) != 0 {
			t.Fatalf("%s: second page must be empty after filtering: %+v", mode, page2)
		}
		unknown := search(reader, 200, broad(mode, "&person_unknown=true"))
		if unknown.Total != 1 || unknown.Items[0].Context != "ailleurs" || unknown.Items[0].Person != nil || unknown.Items[0].DecisionLevel != "unknown" {
			t.Fatalf("%s: the unknown state must select the segment outside any known voice: %+v", mode, unknown)
		}
	}

	// A suggestion is not an identification: shown with its level but no Person, and
	// the segment counts as unknown for the filter.
	if _, err := pool.Exec(ctx, `UPDATE speaker_attributions SET decision_level='suggestion' WHERE local_speaker_id=$1`, second); err != nil {
		t.Fatal(err)
	}
	for _, mode := range modes {
		if page := search(reader, 200, fmt.Sprintf("mode=%s&q=second&person_id=%d", mode, bobPerson.ID)); page.Total != 0 {
			t.Fatalf("%s: a suggestion must not match a Person filter: %+v", mode, page)
		}
		page := search(reader, 200, "mode="+mode+"&q=second&person_unknown=true")
		found := false
		for _, item := range page.Items {
			if item.Context == "second" {
				found = item.Person == nil && item.DecisionLevel == "suggestion"
			}
		}
		if !found {
			t.Fatalf("%s: a suggestion is unknown but its level stays visible: %+v", mode, page)
		}
	}
	// A human validation overrides the machine attribution of the same voice.
	h.userCall(alice, http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", second), map[string]any{"person_id": carolPerson.ID}, http.StatusCreated)
	if page := search(reader, 200, fmt.Sprintf("mode=full_text&q=second&person_id=%d", carolPerson.ID)); page.Total != 1 || page.Items[0].DecisionLevel != "human" {
		t.Fatalf("validated attribution must take precedence: %+v", page)
	}

	// Validation of the parameters.
	for _, params := range []string{"q=e&person_id=0", "q=e&person_id=abc", "q=e&person_unknown=maybe", fmt.Sprintf("q=e&person_id=%d&person_unknown=true", alicePerson.ID)} {
		h.userCall(reader, http.MethodGet, "/api/search?"+params, nil, http.StatusBadRequest)
		h.userCall(reader, http.MethodGet, "/api/clips?"+params[4:], nil, http.StatusBadRequest)
	}

	// Permissions apply before the filter, count and pagination: a private clip never
	// leaks, not even through its Person.
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='private' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	for _, mode := range modes {
		if page := search(reader, 200, fmt.Sprintf("mode=%s&q=premier&person_id=%d", mode, alicePerson.ID)); page.Total != 0 || len(page.Items) != 0 {
			t.Fatalf("%s: private clip leaked through the Person filter: %+v", mode, page)
		}
		if page := search(reader, 200, "mode="+mode+"&q=premier&person_unknown=true"); page.Total != 0 {
			t.Fatalf("%s: private clip leaked through the unknown filter: %+v", mode, page)
		}
		if page := search(h.adminUser(), 200, fmt.Sprintf("mode=%s&q=premier&person_id=%d", mode, alicePerson.ID)); page.Total != 1 {
			t.Fatalf("%s: administrator must still see the clip: %+v", mode, page)
		}
	}
	var hidden libraryPage
	if err := json.Unmarshal(h.userCall(reader, http.MethodGet, fmt.Sprintf("/api/clips?person_id=%d", alicePerson.ID), nil, http.StatusOK), &hidden); err != nil || hidden.Total != 0 || len(hidden.Items) != 0 {
		t.Fatalf("private clip leaked through the library Person filter: %+v %v", hidden, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}

	// Library: filter by Person and identified Persons shown on each clip.
	library := func(params string) libraryPage {
		t.Helper()
		var page libraryPage
		if err := json.Unmarshal(h.userCall(reader, http.MethodGet, "/api/clips?"+params, nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		return page
	}
	page := library(fmt.Sprintf("person_id=%d", alicePerson.ID))
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != h.clipID || len(page.Items[0].Persons) != 2 {
		t.Fatalf("the library must filter by Person and list the identified Persons: %+v", page)
	}
	if p := page.Items[0].Persons; p[0].DisplayName != "Alice" || p[0].DecisionLevel != "human" || p[1].DisplayName != "Carol" {
		t.Fatalf("clip Persons must be sorted with their level: %+v", p)
	}
	if page = library(fmt.Sprintf("person_id=%d", bobPerson.ID)); page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("a Person without voice must give an empty library: %+v", page)
	}
	if page = library("person_unknown=true"); page.Total != 0 {
		t.Fatalf("every voice is identified, no clip has an unknown voice: %+v", page)
	}
	if page = library(""); page.Total != 1 || len(page.Items[0].Persons) != 2 {
		t.Fatalf("unfiltered library must still show Persons: %+v", page)
	}
	// Dropping the machine layer is not enough: a validated attribution cannot be dropped
	// here, so reset the second voice to unknown by removing its validated row directly.
	if _, err := pool.Exec(ctx, `DELETE FROM voice_samples WHERE local_speaker_id=$1`, second); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM speaker_attributions WHERE local_speaker_id=$1`, second); err != nil {
		t.Fatal(err)
	}
	if page = library("person_unknown=true"); page.Total != 1 || len(page.Items[0].Persons) != 1 {
		t.Fatalf("a clip with an unattributed voice must match the unknown filter: %+v", page)
	}
	var detail struct {
		Persons []struct {
			DisplayName string `json:"display_name"`
		} `json:"persons"`
	}
	if err := json.Unmarshal(h.userCall(reader, http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &detail); err != nil || len(detail.Persons) != 1 || detail.Persons[0].DisplayName != "Alice" {
		t.Fatalf("the clip page must show identified Persons: %+v %v", detail, err)
	}
}
