//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Merging or deleting a Person rewrites the attributions of all its voices. A user who
// cannot see one of those clips (private to someone else) is refused: they could
// otherwise modify attributions they are not allowed to read. Administrators and
// Persons whose voices are all visible are unaffected.
func testPersonCorrectionsRespectVisibility(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	visible, private, other := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	alice := h.newUser("alice")
	hidden := h.fixtureSpeaker(private, "HIDDEN", usableAt(0))
	shown := h.fixtureSpeaker(visible, "SHOWN", usableAt(90))
	spare := h.fixtureSpeaker(other, "SPARE", usableAt(180))
	h.validate(hidden, "Carol")
	carol := h.personOf(hidden)
	h.validate(shown, carol)
	h.validate(spare, "Dave")
	dave := h.personOf(spare)
	h.validate(h.fixtureSpeaker(visible, "ERIN", usableAt(45)), "Erin")
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='shared'; UPDATE clips SET visibility='private',owner_id=(SELECT id FROM users WHERE username='administrator') WHERE id=`+fmt.Sprint(private)); err != nil {
		t.Fatal(err)
	}
	body := h.userCall(alice, http.MethodPost, fmt.Sprintf("/api/persons/%d/merge", carol), map[string]any{"target_person_id": dave}, http.StatusConflict)
	if !bytes.Contains(body, []byte("person_has_hidden_voices")) {
		t.Fatalf("unexpected error: %s", body)
	}
	body = h.userCall(alice, http.MethodDelete, fmt.Sprintf("/api/persons/%d", carol), nil, http.StatusConflict)
	if !bytes.Contains(body, []byte("person_has_hidden_voices")) {
		t.Fatalf("unexpected error: %s", body)
	}
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE person_id=$1 AND validation_state='validated'`, carol) != 2 {
		t.Fatal("a refused correction must change nothing")
	}
	// A Person whose voices are all visible can be merged by any connected user.
	var erin int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM persons WHERE display_name='Erin'`).Scan(&erin); err != nil {
		t.Fatal(err)
	}
	h.userCall(alice, http.MethodPost, fmt.Sprintf("/api/persons/%d/merge", erin), map[string]any{"target_person_id": dave}, http.StatusOK)
	// The administrator sees every clip and may correct.
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/persons/%d", carol), nil, http.StatusOK)
}
