//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
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
	avatarPath := fmt.Sprintf("/api/admin/members/%d/avatar", aliceID)
	t.Run("selected account avatar", func(t *testing.T) {
		h.userCall(alice, "GET", avatarPath, nil, http.StatusForbidden)
		req, _ := http.NewRequest("GET", h.ts.URL+avatarPath, nil)
		h.do(http.DefaultClient, req, http.StatusUnauthorized)
		h.adminCall("GET", "/api/admin/members/0/avatar", nil, http.StatusBadRequest)
		h.adminCall("GET", "/api/admin/members/999999/avatar", nil, http.StatusNotFound)
		h.userCall(operator, "GET", avatarPath, nil, http.StatusNotFound)
		custom := solid(color.NRGBA{R: 180, A: 255}, 16, 16)
		discord := solid(color.NRGBA{B: 180, A: 255}, 16, 16)
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `INSERT INTO avatar_preferences(user_id,source) VALUES($1,'custom') ON CONFLICT(user_id) DO UPDATE SET source='custom';`, aliceID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO account_avatars(user_id,source,png) VALUES($1,'custom',$2),($1,'discord',$3)`, aliceID, custom, discord); err != nil {
			t.Fatal(err)
		}
		read := func(etag string, want int) (string, []byte) {
			t.Helper()
			req, _ := http.NewRequest("GET", h.ts.URL+avatarPath, nil)
			req.Header.Set("If-None-Match", etag)
			resp, err := operator.client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			if err != nil || resp.StatusCode != want {
				t.Fatalf("member avatar: status %d, want %d: %v", resp.StatusCode, want, err)
			}
			if resp.Header.Get("Cache-Control") != "private, no-cache" || resp.Header.Get("ETag") == "" {
				t.Fatalf("avatar cache headers: %v", resp.Header)
			}
			if want == http.StatusOK && resp.Header.Get("Content-Type") != "image/png" {
				t.Fatal("avatar must be served as PNG")
			}
			return resp.Header.Get("ETag"), data
		}
		etag, data := read("", http.StatusOK)
		if !bytes.Equal(data, custom) {
			t.Fatal("member must show the selected personal photo")
		}
		_, data = read(etag, http.StatusNotModified)
		if len(data) != 0 {
			t.Fatal("unchanged photo should have no response body")
		}
		if _, err := pool.Exec(ctx, `UPDATE avatar_preferences SET source='discord',version=version+1 WHERE user_id=$1`, aliceID); err != nil {
			t.Fatal(err)
		}
		nextETag, data := read(etag, http.StatusOK)
		if nextETag == etag || !bytes.Equal(data, discord) {
			t.Fatal("switching to Discord must update the photo and its validator")
		}
		if _, err := pool.Exec(ctx, `UPDATE avatar_preferences SET source='none',version=version+1 WHERE user_id=$1`, aliceID); err != nil {
			t.Fatal(err)
		}
		h.userCall(operator, "GET", avatarPath, nil, http.StatusNotFound)
		if _, err := pool.Exec(ctx, `UPDATE avatar_preferences SET source='custom',version=version+1 WHERE user_id=$1`, aliceID); err != nil {
			t.Fatal(err)
		}
	})
	h.adminCall("PUT", fmt.Sprintf("/api/admin/members/%d/role", otherAdminID), map[string]string{"role": "admin"}, http.StatusOK)
	h.userCall(operator, "PUT", fmt.Sprintf("/api/admin/members/%d/role", otherAdminID), map[string]string{"role": "user"}, http.StatusForbidden)
	h.userCall(operator, "DELETE", fmt.Sprintf("/api/admin/members/%d", otherAdminID), map[string]bool{"hide_clips": false}, http.StatusForbidden)
	h.adminCall("PUT", fmt.Sprintf("/api/admin/members/%d/role", otherAdminID), map[string]string{"role": "user"}, http.StatusOK)
	h.userCall(operator, "PUT", fmt.Sprintf("/api/admin/members/%d/role", aliceID), map[string]string{"role": "admin"}, http.StatusForbidden)
	h.adminCall("PUT", fmt.Sprintf("/api/admin/members/%d/role", owner.User.ID), map[string]string{"role": "user"}, http.StatusForbidden)
	h.userCall(operator, "DELETE", fmt.Sprintf("/api/admin/members/%d", opID), map[string]bool{"hide_clips": false}, http.StatusForbidden)
	h.userCall(operator, "DELETE", fmt.Sprintf("/api/admin/members/%d", aliceID), map[string]bool{"hide_clips": false}, http.StatusOK)
	h.userCall(operator, "GET", avatarPath, nil, http.StatusNotFound)
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
