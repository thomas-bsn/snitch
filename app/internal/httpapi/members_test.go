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

func testMembers(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	operator := h.newUser("fixture-operator")
	alice := h.newUser("fixture-alice")
	bob := h.newUser("fixture-bob")
	otherAdmin := h.newUser("fixture-admin")
	readID := func(user *voiceUser) int64 {
		var me struct {
			User struct {
				ID int64 `json:"id"`
			} `json:"user"`
		}
		if err := json.Unmarshal(h.userCall(user, "GET", "/api/auth/me", nil, http.StatusOK), &me); err != nil {
			t.Fatal(err)
		}
		return me.User.ID
	}
	opID, aliceID, bobID := readID(operator), readID(alice), readID(bob)
	otherAdminID := readID(otherAdmin)
	var owner struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(h.adminCall("GET", "/api/auth/me", nil, http.StatusOK), &owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET owner_id=CASE id WHEN $2::bigint THEN $1::bigint ELSE $3::bigint END,visibility='shared' WHERE id IN ($2,$4)`, aliceID, h.clipIDs[0], bobID, h.clipIDs[1]); err != nil {
		t.Fatal(err)
	}
	h.userCall(alice, "GET", "/api/admin/members", nil, http.StatusForbidden)
	h.adminCall("PUT", fmt.Sprintf("/api/admin/members/%d/role", opID), map[string]string{"role": "admin"}, http.StatusOK)
	h.userCall(operator, "GET", "/api/admin/members", nil, http.StatusOK)
	h.adminCall("PUT", fmt.Sprintf("/api/admin/members/%d/role", otherAdminID), map[string]string{"role": "admin"}, http.StatusOK)
	h.userCall(operator, "PUT", fmt.Sprintf("/api/admin/members/%d/role", otherAdminID), map[string]string{"role": "user"}, http.StatusForbidden)
	h.userCall(operator, "DELETE", fmt.Sprintf("/api/admin/members/%d", otherAdminID), map[string]bool{"hide_clips": false}, http.StatusForbidden)
	h.adminCall("PUT", fmt.Sprintf("/api/admin/members/%d/role", otherAdminID), map[string]string{"role": "user"}, http.StatusOK)
	h.userCall(operator, "PUT", fmt.Sprintf("/api/admin/members/%d/role", aliceID), map[string]string{"role": "admin"}, http.StatusForbidden)
	h.adminCall("PUT", fmt.Sprintf("/api/admin/members/%d/role", owner.User.ID), map[string]string{"role": "user"}, http.StatusForbidden)
	h.userCall(operator, "DELETE", fmt.Sprintf("/api/admin/members/%d", opID), map[string]bool{"hide_clips": false}, http.StatusForbidden)
	h.userCall(operator, "DELETE", fmt.Sprintf("/api/admin/members/%d", aliceID), map[string]bool{"hide_clips": false}, http.StatusOK)
	h.adminCall("DELETE", fmt.Sprintf("/api/admin/members/%d", bobID), map[string]bool{"hide_clips": true}, http.StatusOK)
	h.userCall(alice, "GET", "/api/auth/me", nil, http.StatusUnauthorized)
	h.userCall(bob, "GET", "/api/auth/me", nil, http.StatusUnauthorized)
	h.adminCall("POST", fmt.Sprintf("/api/admin/users/%d/approve", aliceID), nil, http.StatusNotFound)
	var clips struct {
		Items []struct {
			ID         int64  `json:"id"`
			Owner      string `json:"owner"`
			Visibility string `json:"visibility"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall("GET", "/api/clips", nil, http.StatusOK), &clips); err != nil {
		t.Fatal(err)
	}
	if len(clips.Items) != 2 {
		t.Fatalf("les clips doivent être conservés : %+v", clips.Items)
	}
	for _, c := range clips.Items {
		if c.ID == h.clipIDs[0] && (c.Visibility != "shared" || c.Owner != "Utilisateur supprimé") {
			t.Fatalf("clip partagé : %+v", c)
		}
		if c.ID == h.clipIDs[1] && c.Visibility != "private" {
			t.Fatalf("clip masqué : %+v", c)
		}
	}
}
