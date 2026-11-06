//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// createPerson makes a Person through the API and returns its id.
func (h *voiceHarness) createPerson(name string) int64 {
	h.t.Helper()
	var p struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/persons", map[string]any{"display_name": name}, http.StatusCreated), &p); err != nil {
		h.t.Fatal(err)
	}
	return p.ID
}

type linkRequest struct {
	ID      int64  `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Person  struct {
		ID int64 `json:"id"`
	} `json:"person"`
	Account struct {
		Username string `json:"username"`
	} `json:"account"`
	CreatedAt string  `json:"created_at"`
	DecidedAt *string `json:"decided_at"`
	DecidedBy *struct {
		Username string `json:"username"`
	} `json:"decided_by"`
}

// testPersonLinkRequests covers the account↔Person link request: a user asks, an
// administrator approves or refuses and can replace an existing link, the account
// withdraws or unlinks itself, and every decision is traced without voice data.
func testPersonLinkRequests(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)

	personA := h.createPerson("Alice")
	personB := h.createPerson("Bob")

	alice := h.newUser("alice-account")
	bob := h.newUser("bob-account")
	eve := h.newUser("eve")

	// The queue of requests is reserved to administrators.
	h.userCall(eve, http.MethodGet, "/api/admin/person-link-requests", nil, http.StatusForbidden)
	h.userCall(eve, http.MethodPost, fmt.Sprintf("/api/admin/person-link-requests/%d/approve", 1), nil, http.StatusForbidden)
	h.userCall(eve, http.MethodPost, fmt.Sprintf("/api/admin/person-link-requests/%d/refuse", 1), nil, http.StatusForbidden)

	// No request and no linked person at first.
	h.userCall(alice, http.MethodGet, "/api/me/person/request", nil, http.StatusNotFound)
	h.userCall(alice, http.MethodGet, "/api/me/person", nil, http.StatusNotFound)

	// A request needs an existing person, a valid id and a bounded message.
	h.userCall(alice, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": 0}, http.StatusBadRequest)
	h.userCall(alice, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": 999999}, http.StatusNotFound)
	h.userCall(alice, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": personA, "message": strings.Repeat("x", 501)}, http.StatusBadRequest)

	// Alice asks for the link.
	var req linkRequest
	if err := json.Unmarshal(h.userCall(alice, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": personA, "message": "C'est moi."}, http.StatusCreated), &req); err != nil {
		t.Fatal(err)
	}
	if req.Status != "pending" || req.Person.ID != personA || req.Account.Username != "alice-account" || req.Message != "C'est moi." || req.DecidedAt != nil || req.DecidedBy != nil {
		t.Fatalf("created request: %+v", req)
	}

	// One request at a time per account, and per person.
	h.userCall(alice, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": personB}, http.StatusConflict)
	h.userCall(bob, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": personA}, http.StatusConflict)

	// The administrator sees exactly this pending request.
	var list struct {
		Items []linkRequest `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/person-link-requests", nil, http.StatusOK), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != req.ID || list.Items[0].Person.ID != personA || list.Items[0].Account.Username != "alice-account" || list.Items[0].Message != "C'est moi." {
		t.Fatalf("pending queue: %+v", list.Items)
	}
	h.adminCall(http.MethodGet, "/api/admin/person-link-requests?status=bogus", nil, http.StatusBadRequest)

	// Alice withdraws, then asks again and is refused.
	h.userCall(alice, http.MethodDelete, "/api/me/person/request", nil, http.StatusOK)
	h.userCall(alice, http.MethodGet, "/api/me/person/request", nil, http.StatusNotFound)
	h.userCall(alice, http.MethodDelete, "/api/me/person/request", nil, http.StatusNotFound)
	if err := json.Unmarshal(h.userCall(alice, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": personA}, http.StatusCreated), &req); err != nil {
		t.Fatal(err)
	}
	var decided linkRequest
	if err := json.Unmarshal(h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/person-link-requests/%d/refuse", req.ID), nil, http.StatusOK), &decided); err != nil {
		t.Fatal(err)
	}
	if decided.Status != "refused" || decided.DecidedAt == nil || decided.DecidedBy == nil || decided.DecidedBy.Username != "administrator" {
		t.Fatalf("refused request: %+v", decided)
	}
	// A decided request cannot be decided again; a missing one is not found.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/person-link-requests/%d/refuse", req.ID), nil, http.StatusConflict)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/person-link-requests/%d/approve", req.ID), nil, http.StatusConflict)
	h.adminCall(http.MethodPost, "/api/admin/person-link-requests/999999/approve", nil, http.StatusNotFound)
	h.adminCall(http.MethodPost, "/api/admin/person-link-requests/999999/refuse", nil, http.StatusNotFound)
	if err := json.Unmarshal(h.userCall(alice, http.MethodGet, "/api/me/person/request", nil, http.StatusOK), &decided); err != nil || decided.Status != "refused" {
		t.Fatalf("the account must see the refusal: %+v %v", decided, err)
	}

	// Alice asks once more and the administrator approves.
	if err := json.Unmarshal(h.userCall(alice, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": personA}, http.StatusCreated), &req); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/person-link-requests/%d/approve", req.ID), nil, http.StatusOK), &decided); err != nil || decided.Status != "approved" {
		t.Fatalf("approved request: %+v %v", decided, err)
	}
	var mine profilePerson
	if err := json.Unmarshal(h.userCall(alice, http.MethodGet, "/api/me/person", nil, http.StatusOK), &mine); err != nil || mine.ID != personA || mine.UserID == nil {
		t.Fatalf("the account must find its person: %+v %v", mine, err)
	}
	// A linked account no longer asks for another person.
	h.userCall(alice, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": personB}, http.StatusConflict)

	// A replacement: Bob asks for the same person; approving takes the link from Alice.
	if err := json.Unmarshal(h.userCall(bob, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": personA}, http.StatusCreated), &req); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/person-link-requests/%d/approve", req.ID), nil, http.StatusOK), &decided); err != nil || decided.Status != "approved" {
		t.Fatalf("replacement approval: %+v %v", decided, err)
	}
	h.userCall(alice, http.MethodGet, "/api/me/person", nil, http.StatusNotFound)
	if err := json.Unmarshal(h.userCall(bob, http.MethodGet, "/api/me/person", nil, http.StatusOK), &mine); err != nil || mine.ID != personA {
		t.Fatalf("Bob must hold the link after the replacement: %+v %v", mine, err)
	}

	// The linked account withdraws its own link; Person, clips and attributions stay.
	var clipsBefore int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM clips`).Scan(&clipsBefore); err != nil {
		t.Fatal(err)
	}
	h.userCall(bob, http.MethodDelete, "/api/me/person", nil, http.StatusOK)
	h.userCall(bob, http.MethodGet, "/api/me/person", nil, http.StatusNotFound)
	h.userCall(bob, http.MethodDelete, "/api/me/person", nil, http.StatusNotFound)
	if h.count(`SELECT count(*) FROM clips`) != clipsBefore || h.count(`SELECT count(*) FROM persons`) != 2 {
		t.Fatal("unlinking must keep persons and clips")
	}
	// The person still exists and is free to be claimed again.
	if h.count(`SELECT count(*) FROM persons WHERE id=$1 AND user_id IS NULL`, personA) != 1 {
		t.Fatal("the person must be unlinked but kept")
	}

	// An account that gained another link between its request and the decision cannot be
	// approved for this person.
	carol := h.newUser("carol-account")
	personC := h.createPerson("Carol")
	personD := h.createPerson("Dana")
	if err := json.Unmarshal(h.userCall(carol, http.MethodPost, "/api/me/person/request", map[string]any{"person_id": personC}, http.StatusCreated), &req); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/persons/%d/account", personD), map[string]any{"username": "carol-account"}, http.StatusOK)
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/person-link-requests/%d/approve", req.ID), nil, http.StatusConflict)
	if err := json.Unmarshal(h.userCall(carol, http.MethodGet, "/api/me/person", nil, http.StatusOK), &mine); err != nil || mine.ID != personD {
		t.Fatalf("carol must keep her direct link: %+v %v", mine, err)
	}

	// The direct administrator link and unlink write the same trace.
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/persons/%d/account", personD), nil, http.StatusOK)
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/persons/%d/account", personA), map[string]any{"username": "eve"}, http.StatusOK)
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/persons/%d/account", personA), nil, http.StatusOK)

	// Trace: every kind of decision is kept, a replacement names the account it left,
	// and no voice data is stored.
	var previousUserID *int64
	var previousUsername string
	if err := pool.QueryRow(ctx, `SELECT previous_user_id, previous_username FROM person_link_events WHERE action='approve' AND person_id=$1 AND previous_user_id IS NOT NULL ORDER BY id LIMIT 1`, personA).Scan(&previousUserID, &previousUsername); err != nil {
		t.Fatalf("the replacement must be traced: %v", err)
	}
	var aliceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='alice-account'`).Scan(&aliceID); err != nil {
		t.Fatal(err)
	}
	if previousUserID == nil || *previousUserID != aliceID || previousUsername != "alice-account" {
		t.Fatalf("replacement trace: %v %q", previousUserID, previousUsername)
	}
	for _, action := range []string{"request", "cancel", "refuse", "approve", "unlink", "link"} {
		if h.count(`SELECT count(*) FROM person_link_events WHERE action=$1`, action) == 0 {
			t.Fatalf("the trace must hold at least one %q event", action)
		}
	}
	if h.count(`SELECT count(*) FROM person_link_events WHERE person_name=''`) != 0 {
		t.Fatal("the trace must keep the person name at the time")
	}
	rows, err := pool.Query(ctx, `SELECT column_name,data_type,udt_name FROM information_schema.columns WHERE table_name IN ('person_link_events','person_link_requests')`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name, dataType, udt string
		if err = rows.Scan(&name, &dataType, &udt); err != nil {
			t.Fatal(err)
		}
		if udt == "vector" || strings.Contains(name, "embedding") || strings.Contains(name, "vector") || dataType == "bytea" {
			t.Fatalf("the link trail must hold no voice data: %s %s", name, udt)
		}
	}
	rows.Close()
	if _, err = pool.Exec(ctx, `UPDATE person_link_events SET action='tampered'`); err == nil {
		t.Fatal("the link trail must be immutable")
	}
	if _, err = pool.Exec(ctx, `DELETE FROM person_link_events`); err == nil {
		t.Fatal("the link trail must not be deletable")
	}
}
