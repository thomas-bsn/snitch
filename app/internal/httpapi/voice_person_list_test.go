//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type personListPage struct {
	Items []struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"items"`
	Total *int `json:"total"`
}

// testPersonListPagination covers the additive pagination of GET /api/persons: bounds,
// stable name order, `q` combined with paging, and the unchanged unpaged response.
func testPersonListPagination(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	user := h.newUser("alice")
	if _, err := pool.Exec(ctx, `INSERT INTO persons(display_name) SELECT 'Personne '||lpad(g::text,4,'0') FROM generate_series(1,650) g`); err != nil {
		t.Fatal(err)
	}
	// Ties on the name are broken by id, so paging never repeats or skips a Person.
	if _, err := pool.Exec(ctx, `INSERT INTO persons(display_name) VALUES ('Doublon'),('Doublon'),('Doublon'),('Cent%_pour\cent'),('Cent pour cent')`); err != nil {
		t.Fatal(err)
	}
	get := func(query string, want int) personListPage {
		t.Helper()
		var page personListPage
		body := h.userCall(user, http.MethodGet, "/api/persons"+query, nil, want)
		if want == http.StatusOK {
			if err := json.Unmarshal(body, &page); err != nil {
				t.Fatal(err)
			}
		}
		return page
	}

	// Without paging parameters: unchanged, capped at 500, no total.
	legacy := get("", http.StatusOK)
	if len(legacy.Items) != 500 || legacy.Total != nil {
		t.Fatalf("unpaged: %d items, total %v", len(legacy.Items), legacy.Total)
	}

	first := get("?page=1&page_size=100", http.StatusOK)
	if len(first.Items) != 100 || first.Total == nil || *first.Total != 655 {
		t.Fatalf("page 1: %d items, total %v", len(first.Items), first.Total)
	}
	// Every Person exactly once across pages, in name order.
	seen := map[int64]bool{}
	last := ""
	for page := 1; page <= 7; page++ {
		got := get(fmt.Sprintf("?page=%d&page_size=100", page), http.StatusOK)
		want := 100
		if page == 7 {
			want = 55
		}
		if len(got.Items) != want {
			t.Fatalf("page %d: %d items, want %d", page, len(got.Items), want)
		}
		for _, item := range got.Items {
			if seen[item.ID] {
				t.Fatalf("person %d repeated across pages", item.ID)
			}
			seen[item.ID] = true
			if strings.ToLower(item.DisplayName) < last {
				t.Fatalf("order broken: %q after %q", item.DisplayName, last)
			}
			last = strings.ToLower(item.DisplayName)
		}
	}
	if len(seen) != 655 {
		t.Fatalf("saw %d persons, want 655", len(seen))
	}
	if beyond := get("?page=8&page_size=100", http.StatusOK); len(beyond.Items) != 0 || beyond.Total == nil || *beyond.Total != 655 {
		t.Fatalf("beyond the last page: %d items, total %v", len(beyond.Items), beyond.Total)
	}
	if size := get("?page_size=3", http.StatusOK); len(size.Items) != 3 || size.Total == nil {
		t.Fatalf("page_size alone: %d items", len(size.Items))
	}

	// q combined with paging: the total counts the matches only.
	filtered := get("?q=personne+06&page=2&page_size=20", http.StatusOK)
	if filtered.Total == nil || *filtered.Total != 51 || len(filtered.Items) != 20 || filtered.Items[0].DisplayName != "Personne 0620" {
		t.Fatalf("q+page: %+v", filtered)
	}
	// q wildcards stay literal.
	literal := get("?q="+url.QueryEscape("%_")+"&page=1&page_size=10", http.StatusOK)
	if literal.Total == nil || *literal.Total != 1 || literal.Items[0].DisplayName != `Cent%_pour\cent` {
		t.Fatalf("literal q: %+v", literal)
	}
	if none := get("?q=introuvable&page=1&page_size=10", http.StatusOK); none.Total == nil || *none.Total != 0 || len(none.Items) != 0 {
		t.Fatalf("no match: %+v", none)
	}

	// Bounds.
	for _, bad := range []string{"?page=0", "?page=-1", "?page=abc", "?page=100001", "?page_size=0", "?page_size=101", "?page_size=x"} {
		get(bad, http.StatusBadRequest)
	}
	get("?page=100000&page_size=100", http.StatusOK)
}
