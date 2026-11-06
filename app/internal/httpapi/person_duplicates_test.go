//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPersonDuplicates covers #269: an exact duplicate (same name up to case and spaces)
// is refused with the existing Persons, an explicit homonym is allowed, and concurrent
// creations of the same name produce exactly one Person.
func testPersonDuplicates(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	user := h.newUser("alice")
	first := h.createPerson("Marie Curie")

	var refusal struct {
		Error      string `json:"error"`
		Candidates []struct {
			ID int64 `json:"id"`
		} `json:"candidates"`
	}
	for _, name := range []string{"Marie Curie", "  marie   CURIE "} {
		body := h.userCall(user, http.MethodPost, "/api/persons", map[string]any{"display_name": name}, http.StatusConflict)
		if err := json.Unmarshal(body, &refusal); err != nil {
			t.Fatal(err)
		}
		if refusal.Error != "person_exists" || len(refusal.Candidates) != 1 || refusal.Candidates[0].ID != first {
			t.Fatalf("%q: %s", name, body)
		}
	}
	h.userCall(user, http.MethodPost, "/api/persons", map[string]any{"display_name": "Marie Curie", "allow_homonym": true}, http.StatusCreated)

	// Concurrent creations of a new name: one winner, the others see it.
	var wg sync.WaitGroup
	var mu sync.Mutex
	statuses := map[int]int{}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, _ := h.userStatus(user, http.MethodPost, "/api/persons", map[string]any{"display_name": "Ada Lovelace"})
			mu.Lock()
			statuses[status]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if statuses[http.StatusCreated] != 1 || statuses[http.StatusConflict] != 7 {
		t.Fatalf("concurrent creations: %v", statuses)
	}
	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM persons WHERE display_name='Ada Lovelace'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("persons named Ada Lovelace: %d, %v", count, err)
	}
}
