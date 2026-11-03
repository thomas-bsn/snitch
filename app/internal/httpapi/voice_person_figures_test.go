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

// testPersonFigures checks the figures a Person carries on the directory: the clips where
// a human validated one of its voices and the speaking time of those voices, counted only
// on clips the caller can see.
func testPersonFigures(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, _ := h.diarizeClip()
	admin := h.adminUser()
	alice := h.newUser("alice")
	h.userCall(admin, http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", first), map[string]any{"new_person_name": "Figures"}, http.StatusCreated)
	var seconds float64
	if err := pool.QueryRow(context.Background(), `SELECT speaking_seconds FROM local_speakers WHERE id=$1`, first).Scan(&seconds); err != nil {
		t.Fatal(err)
	}

	type figures struct {
		ID              int64   `json:"id"`
		ClipCount       int     `json:"clip_count"`
		SpeakingSeconds float64 `json:"speaking_seconds"`
	}
	list := func(u *voiceUser) figures {
		t.Helper()
		var people struct {
			Items []figures `json:"items"`
		}
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/persons", nil, http.StatusOK), &people); err != nil || len(people.Items) != 1 {
			t.Fatalf("expected one person for %s: %v %+v", u.name, err, people)
		}
		return people.Items[0]
	}

	got := list(admin)
	if got.ClipCount != 1 || got.SpeakingSeconds != seconds {
		t.Fatalf("admin figures: %+v, want 1 clip and %v seconds", got, seconds)
	}
	var single figures
	if err := json.Unmarshal(h.userCall(admin, http.MethodGet, fmt.Sprintf("/api/persons/%d", got.ID), nil, http.StatusOK), &single); err != nil {
		t.Fatal(err)
	}
	if single.ClipCount != 1 || single.SpeakingSeconds != seconds {
		t.Fatalf("person figures: %+v", single)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='private' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if hidden := list(alice); hidden.ClipCount != 0 || hidden.SpeakingSeconds != 0 {
		t.Fatalf("a hidden clip must not count for another user: %+v", hidden)
	}
}
