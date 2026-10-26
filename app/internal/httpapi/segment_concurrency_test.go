//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type concurrentResult struct {
	status int
	body   correctionResponse
}

type concurrentWrite struct {
	user    *voiceUser
	index   int
	version int
	text    string
}

// raceWrites fires every PUT at the same instant and returns the answers in order.
func raceWrites(h *voiceHarness, writes []concurrentWrite) []concurrentResult {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]concurrentResult, len(writes))
	for i, w := range writes {
		wg.Add(1)
		go func(i int, w concurrentWrite) {
			defer wg.Done()
			<-start
			status, data := h.userStatus(w.user, http.MethodPut, segmentTextPath(h.clipID, w.index), map[string]any{"text": w.text, "expected_version": w.version})
			var body correctionResponse
			_ = json.Unmarshal(data, &body)
			results[i] = concurrentResult{status, body}
		}(i, w)
	}
	close(start)
	wg.Wait()
	return results
}

func testSegmentConcurrency(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.transcribe()
	alice, bob := h.newUser("alice"), h.newUser("bob")

	// Two writers read the same version. Round 0 is the first write of a never corrected
	// segment (no row yet), round 1 a write on an existing row: each has one winner.
	for round := 0; round < 2; round++ {
		results := raceWrites(h, []concurrentWrite{
			{alice, 0, round, fmt.Sprintf("version d'Alice %d", round)},
			{bob, 0, round, fmt.Sprintf("version de Bob %d", round)},
		})
		names := []string{"alice", "bob"}
		won, lost := -1, -1
		for i, r := range results {
			switch r.status {
			case http.StatusOK:
				if won >= 0 {
					t.Fatalf("round %d: two winners: %+v", round, results)
				}
				won = i
			case http.StatusConflict:
				lost = i
			default:
				t.Fatalf("round %d: unexpected answer: %+v", round, results)
			}
		}
		if won < 0 || lost < 0 {
			t.Fatalf("round %d: one winner and one conflict expected: %+v", round, results)
		}
		winner, loser := results[won], results[lost]
		if winner.body.Version != round+1 {
			t.Fatalf("round %d: winner version: %+v", round, winner)
		}
		c := loser.body.Current
		if loser.body.Error != "segment_conflict" || c == nil || c.Version != round+1 || c.Text != winner.body.Text || !c.Corrected || c.Author == nil || c.Author.Username != names[won] {
			t.Fatalf("round %d: the loser must receive the winner's version and author: %+v", round, loser)
		}
		// Nothing disappears silently: the winner is the effective text and the history has one entry per write.
		if got := h.clipTranscript(alice).Transcription.Segments[0]; got.Text != winner.body.Text || got.Version != round+1 || got.CorrectedBy != names[won] {
			t.Fatalf("round %d: effective segment: %+v", round, got)
		}
		if n := h.count(`SELECT count(*) FROM segment_correction_events WHERE segment_index=0`); n != round+1 {
			t.Fatalf("round %d: history entries: %d", round, n)
		}
	}

	// Many simultaneous writers on one segment still give a single winner.
	const writers = 8
	var writes []concurrentWrite
	for i := 0; i < writers; i++ {
		user := alice
		if i%2 == 1 {
			user = bob
		}
		writes = append(writes, concurrentWrite{user, 1, 0, fmt.Sprintf("course %d", i)})
	}
	wins, conflicts := 0, 0
	for _, r := range raceWrites(h, writes) {
		switch r.status {
		case http.StatusOK:
			wins++
		case http.StatusConflict:
			conflicts++
		}
	}
	if wins != 1 || conflicts != writers-1 {
		t.Fatalf("one winner among %d writers expected, got %d wins and %d conflicts", writers, wins, conflicts)
	}
	if n := h.count(`SELECT count(*) FROM segment_correction_events WHERE segment_index=1`); n != 1 {
		t.Fatalf("history entries: %d", n)
	}

	// Two different segments corrected at the same time never conflict.
	for round := 0; round < 5; round++ {
		if _, err := pool.Exec(ctx, `DELETE FROM segment_corrections WHERE segment_index=2`); err != nil {
			t.Fatal(err)
		}
		results := raceWrites(h, []concurrentWrite{
			{alice, 2, 0, fmt.Sprintf("parallèle A%d", round)},
			{bob, 1, 1 + round, fmt.Sprintf("parallèle B%d", round)},
		})
		if results[0].status != http.StatusOK || results[1].status != http.StatusOK {
			t.Fatalf("round %d: different segments must not conflict: %+v", round, results)
		}
	}
}
