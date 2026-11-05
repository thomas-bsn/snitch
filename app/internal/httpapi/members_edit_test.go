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
	"snitch/internal/auth"
)

func testMemberEdit(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	ctx := context.Background()
	operator := h.newUser("edit-operator")
	otherAdmin := h.newUser("edit-admin")
	alice := h.newUser("edit-alice")
	idOf := func(u *voiceUser) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, u.name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	opID, adminID, aliceID := idOf(operator), idOf(otherAdmin), idOf(alice)
	var ownerID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE role='super_admin'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	path := func(id int64, suffix string) string { return fmt.Sprintf("/api/admin/members/%d%s", id, suffix) }
	h.adminCall("PUT", path(opID, "/role"), map[string]string{"role": "admin"}, http.StatusOK)
	h.adminCall("PUT", path(adminID, "/role"), map[string]string{"role": "admin"}, http.StatusOK)

	// A plain member reaches nothing.
	h.userCall(alice, "GET", path(aliceID, ""), nil, http.StatusForbidden)
	h.userCall(alice, "PATCH", path(aliceID, ""), map[string]string{"display_name": "x"}, http.StatusForbidden)

	// Full edit by the Super-administrator.
	h.adminCall("PATCH", path(aliceID, ""), map[string]any{"username": "alice-renamed", "display_name": "  Alice A.  ", "language": "fr"}, http.StatusOK)
	var got struct {
		Username    string  `json:"username"`
		DisplayName *string `json:"display_name"`
		Language    *string `json:"language"`
		Permissions struct {
			Edit, Manage bool
			SetRole      bool `json:"set_role"`
		} `json:"permissions"`
	}
	read := func(by *voiceUser, id int64) {
		var data []byte
		if by == nil {
			data = h.adminCall("GET", path(id, ""), nil, http.StatusOK)
		} else {
			data = h.userCall(by, "GET", path(id, ""), nil, http.StatusOK)
		}
		got.DisplayName, got.Language = nil, nil
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
	}
	read(nil, aliceID)
	if got.Username != "alice-renamed" || got.DisplayName == nil || *got.DisplayName != "Alice A." || got.Language == nil || *got.Language != "fr" || !got.Permissions.Manage || !got.Permissions.SetRole {
		t.Fatalf("fiche : %+v", got)
	}
	// Absent keeps, null clears.
	h.adminCall("PATCH", path(aliceID, ""), map[string]any{"display_name": nil, "language": nil}, http.StatusOK)
	read(nil, aliceID)
	if got.Username != "alice-renamed" || got.DisplayName != nil || got.Language != nil {
		t.Fatalf("effacement : %+v", got)
	}

	// Validation and uniqueness.
	h.adminCall("PATCH", path(aliceID, ""), map[string]any{}, http.StatusBadRequest)
	h.adminCall("PATCH", path(aliceID, ""), map[string]any{"username": nil}, http.StatusBadRequest)
	h.adminCall("PATCH", path(aliceID, ""), map[string]any{"username": "ab"}, http.StatusBadRequest)
	h.adminCall("PATCH", path(aliceID, ""), map[string]any{"display_name": strings.Repeat("x", 101)}, http.StatusBadRequest)
	h.adminCall("PATCH", path(aliceID, ""), map[string]any{"language": "de"}, http.StatusBadRequest)
	h.adminCall("PATCH", path(aliceID, ""), map[string]any{"role": "super_admin"}, http.StatusBadRequest)
	h.adminCall("PATCH", path(aliceID, ""), map[string]any{"username": "EDIT-ADMIN"}, http.StatusConflict)
	h.adminCall("PATCH", path(9999999, ""), map[string]any{"language": "fr"}, http.StatusNotFound)

	// An Administrator edits members, not other Administrators nor the Super-administrator, and does not grant roles.
	h.userCall(operator, "PATCH", path(aliceID, ""), map[string]any{"display_name": "Par un admin"}, http.StatusOK)
	h.userCall(operator, "PATCH", path(aliceID, ""), map[string]any{"role": "admin"}, http.StatusForbidden)
	h.userCall(operator, "PATCH", path(adminID, ""), map[string]any{"display_name": "x"}, http.StatusForbidden)
	h.userCall(operator, "PATCH", path(ownerID, ""), map[string]any{"display_name": "x"}, http.StatusForbidden)
	h.userCall(operator, "POST", path(adminID, "/password"), nil, http.StatusForbidden)
	h.userCall(operator, "DELETE", path(adminID, "/sessions"), nil, http.StatusForbidden)
	read(operator, adminID)
	if got.Permissions.Edit || got.Permissions.Manage {
		t.Fatalf("droits sur un administrateur : %+v", got.Permissions)
	}

	// Nobody changes their own role or approval; the profile stays editable.
	h.userCall(operator, "PATCH", path(opID, ""), map[string]any{"display_name": "Moi"}, http.StatusOK)
	h.userCall(operator, "PATCH", path(opID, ""), map[string]any{"role": "user"}, http.StatusForbidden)
	h.userCall(operator, "PATCH", path(opID, ""), map[string]any{"approved": false}, http.StatusForbidden)
	h.adminCall("PATCH", path(ownerID, ""), map[string]any{"role": "user"}, http.StatusForbidden)
	h.adminCall("POST", path(ownerID, "/password"), nil, http.StatusForbidden)
	h.adminCall("PATCH", path(adminID, ""), map[string]any{"role": "user"}, http.StatusOK)

	// Password reset: shown once, usable, ends the sessions.
	body := h.adminCall("POST", path(aliceID, "/password"), nil, http.StatusOK)
	var reset struct {
		Password string `json:"temporary_password"`
	}
	if err := json.Unmarshal(body, &reset); err != nil || len(reset.Password) < 12 {
		t.Fatalf("mot de passe temporaire : %q %v", body, err)
	}
	h.userCall(alice, "GET", "/api/auth/me", nil, http.StatusUnauthorized)
	var hash string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1`, aliceID).Scan(&hash); err != nil || !auth.Verify(hash, reset.Password) {
		t.Fatalf("le mot de passe temporaire doit être celui du compte : %v", err)
	}

	// Closing sessions.
	h.userCall(otherAdmin, "GET", "/api/auth/me", nil, http.StatusOK)
	h.userCall(operator, "DELETE", path(adminID, "/sessions"), nil, http.StatusOK)
	h.userCall(otherAdmin, "GET", "/api/auth/me", nil, http.StatusUnauthorized)

	// Discord: unlink only with a local password.
	h.adminCall("DELETE", path(aliceID, "/discord"), nil, http.StatusConflict)
	if _, err := pool.Exec(ctx, `UPDATE users SET discord_id='42',discord_username='alice#1' WHERE id=$1`, aliceID); err != nil {
		t.Fatal(err)
	}
	h.adminCall("DELETE", path(aliceID, "/discord"), nil, http.StatusOK)
	if _, err := pool.Exec(ctx, `UPDATE users SET discord_id='43',password_hash=NULL WHERE id=$1`, aliceID); err != nil {
		t.Fatal(err)
	}
	h.adminCall("DELETE", path(aliceID, "/discord"), nil, http.StatusConflict)

	// Suspension ends the sessions and moves the account back to the sign-ups.
	h.userCall(operator, "GET", "/api/auth/me", nil, http.StatusOK)
	h.adminCall("PATCH", path(opID, ""), map[string]any{"approved": false}, http.StatusOK)
	h.userCall(operator, "GET", "/api/auth/me", nil, http.StatusUnauthorized)
}
