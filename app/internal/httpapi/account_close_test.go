//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testAccountClose(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.transcribe()
	user := h.newUser("fixture-closing")
	var me struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(h.userCall(user, "GET", "/api/auth/me", nil, http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET owner_id=$1,visibility='shared' WHERE id=$2`, me.User.ID, h.clipID); err != nil {
		t.Fatal(err)
	}
	h.userCall(user, "PUT", segmentTextPath(h.clipID, 0), map[string]any{"text": "correction synthétique", "expected_version": 0}, http.StatusOK)
	jar, _ := cookiejar.New(nil)
	second := &voiceUser{name: "second session", client: &http.Client{Jar: jar}}
	h.userCall(second, "POST", "/api/auth/login", map[string]string{"username": user.name, "password": "synthetic-password"}, http.StatusOK)
	h.adminCall("DELETE", "/api/account", nil, http.StatusForbidden)
	h.userCall(user, "DELETE", "/api/account", nil, http.StatusOK)
	h.userCall(user, "GET", "/api/auth/me", nil, http.StatusUnauthorized)
	h.userCall(second, "GET", "/api/auth/me", nil, http.StatusUnauthorized)
	h.userCall(second, "POST", "/api/auth/login", map[string]string{"username": user.name, "password": "synthetic-password"}, http.StatusUnauthorized)
	var clip struct {
		Visibility    string `json:"visibility"`
		Transcription struct {
			Segments []segmentView `json:"segments"`
		} `json:"transcription"`
	}
	if err := json.Unmarshal(h.adminCall("GET", fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &clip); err != nil {
		t.Fatal(err)
	}
	if clip.Visibility != "private" || len(clip.Transcription.Segments) == 0 || clip.Transcription.Segments[0].Text != "correction synthétique" {
		t.Fatalf("clip ou correction perdus : %+v", clip)
	}
	h.adminCall("GET", fmt.Sprintf("/api/clips/%d/segments/0/history", h.clipID), nil, http.StatusOK)
}
