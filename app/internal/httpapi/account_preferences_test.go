//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

func testAccountPreferences(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)
	server.Config.PublicURL = ts.URL
	newClient := func() *http.Client {
		jar, err := cookiejar.New(nil)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Client{Jar: jar}
	}
	call := func(client *http.Client, csrf, method, path, body string, want int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", ts.URL)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, out)
		}
		return out
	}
	csrfOf := func(client *http.Client) string {
		t.Helper()
		var me struct {
			CSRF string `json:"csrf_token"`
		}
		if err := json.Unmarshal(call(client, "", "GET", "/api/auth/me", "", 200), &me); err != nil {
			t.Fatal(err)
		}
		return me.CSRF
	}
	language := func(client *http.Client) *string {
		t.Helper()
		var prefs struct {
			Language *string `json:"language"`
		}
		if err := json.Unmarshal(call(client, "", "GET", "/api/account/preferences", "", 200), &prefs); err != nil {
			t.Fatal(err)
		}
		return prefs.Language
	}
	admin := newClient()
	call(admin, "", "POST", "/api/setup", `{"code":"`+code+`","username":"administrator","password":"synthetic-password-123"}`, 201)
	adminCSRF := csrfOf(admin)

	t.Run("preferences require a session", func(t *testing.T) {
		call(newClient(), "", "GET", "/api/account/preferences", "", 401)
		call(newClient(), "", "PUT", "/api/account/preferences", `{"language":"fr"}`, 401)
	})
	t.Run("Discord link state and unlink preserve the local sign-in safeguard", func(t *testing.T) {
		var me struct {
			User struct {
				ID int64 `json:"id"`
			} `json:"user"`
			DiscordLinked    bool `json:"discord_linked"`
			CanUnlinkDiscord bool `json:"can_unlink_discord"`
		}
		if err := json.Unmarshal(call(admin, "", "GET", "/api/auth/me", "", 200), &me); err != nil {
			t.Fatal(err)
		}
		userID := me.User.ID
		if _, err := pool.Exec(ctx, `UPDATE users SET discord_id='fixture-discord-user' WHERE id=$1`, userID); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(call(admin, "", "GET", "/api/auth/me", "", 200), &me); err != nil {
			t.Fatal(err)
		}
		if !me.DiscordLinked || !me.CanUnlinkDiscord {
			t.Fatalf("linked account state: linked=%v can_unlink=%v", me.DiscordLinked, me.CanUnlinkDiscord)
		}
		call(admin, "", "DELETE", "/api/account/discord", "", 403)
		call(admin, adminCSRF, "DELETE", "/api/account/discord", "", 200)
		var linked bool
		if err := pool.QueryRow(ctx, `SELECT discord_id IS NOT NULL FROM users WHERE id=$1`, userID).Scan(&linked); err != nil || linked {
			t.Fatalf("Discord identity should be removed: linked=%v err=%v", linked, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET discord_id='fixture-discord-only',password_hash=NULL WHERE id=$1`, userID); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(call(admin, "", "GET", "/api/auth/me", "", 200), &me); err != nil {
			t.Fatal(err)
		}
		if !me.DiscordLinked || me.CanUnlinkDiscord {
			t.Fatalf("Discord-only account state: linked=%v can_unlink=%v", me.DiscordLinked, me.CanUnlinkDiscord)
		}
		out := call(admin, adminCSRF, "DELETE", "/api/account/discord", "", 409)
		if !bytes.Contains(out, []byte(`"error":"discord_local_login_required"`)) {
			t.Fatalf("unexpected safeguard response: %s", out)
		}
		if err := pool.QueryRow(ctx, `SELECT discord_id IS NOT NULL FROM users WHERE id=$1`, userID).Scan(&linked); err != nil || !linked {
			t.Fatalf("Discord-only identity must remain linked: linked=%v err=%v", linked, err)
		}
	})
	t.Run("a new account has no language chosen", func(t *testing.T) {
		if got := language(admin); got != nil {
			t.Fatalf("language %q, want none", *got)
		}
	})
	t.Run("a valid language is read back after being written", func(t *testing.T) {
		out := call(admin, adminCSRF, "PUT", "/api/account/preferences", `{"language":"fr"}`, 200)
		if !bytes.Contains(out, []byte(`"language":"fr"`)) {
			t.Fatalf("write did not return the stored language: %s", out)
		}
		if got := language(admin); got == nil || *got != "fr" {
			t.Fatalf("language %v after writing fr", got)
		}
		var me struct {
			User struct {
				Language *string `json:"language"`
			} `json:"user"`
		}
		if err := json.Unmarshal(call(admin, "", "GET", "/api/auth/me", "", 200), &me); err != nil {
			t.Fatal(err)
		}
		if me.User.Language == nil || *me.User.Language != "fr" {
			t.Fatalf("session does not carry the account language: %+v", me)
		}
		call(admin, adminCSRF, "PUT", "/api/account/preferences", `{"language":"en"}`, 200)
		if got := language(admin); got == nil || *got != "en" {
			t.Fatalf("language %v after writing en", got)
		}
	})
	t.Run("the language survives a new session", func(t *testing.T) {
		call(admin, adminCSRF, "PUT", "/api/account/preferences", `{"language":"fr"}`, 200)
		again := newClient()
		call(again, "", "POST", "/api/auth/login", `{"username":"administrator","password":"synthetic-password-123"}`, 200)
		if got := language(again); got == nil || *got != "fr" {
			t.Fatalf("language %v on a new session", got)
		}
	})
	t.Run("an unsupported or malformed language is refused and nothing changes", func(t *testing.T) {
		for body, want := range map[string]string{
			`{"language":"de"}`:             "invalid_language",
			`{"language":""}`:               "invalid_language",
			`{"language":"FR"}`:             "invalid_language",
			`{"language":null}`:             "invalid_request",
			`{}`:                            "invalid_request",
			`{"language":"en","theme":"x"}`: "invalid_request",
		} {
			out := call(admin, adminCSRF, "PUT", "/api/account/preferences", body, 400)
			if !bytes.Contains(out, []byte(want)) {
				t.Errorf("%s: got %s, want %s", body, out, want)
			}
		}
		if got := language(admin); got == nil || *got != "fr" {
			t.Fatalf("a refused write changed the language to %v", got)
		}
	})
	t.Run("a write needs the CSRF token", func(t *testing.T) {
		call(admin, "", "PUT", "/api/account/preferences", `{"language":"en"}`, 403)
		if got := language(admin); got == nil || *got != "fr" {
			t.Fatalf("language %v after a rejected write", got)
		}
	})
	t.Run("each account keeps its own language", func(t *testing.T) {
		call(newClient(), "", "POST", "/api/auth/register", `{"username":"second-member","password":"synthetic-password-456"}`, 201)
		var id int64
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='second-member'`).Scan(&id); err != nil {
			t.Fatal(err)
		}
		call(admin, adminCSRF, "POST", "/api/admin/users/"+strconv.FormatInt(id, 10)+"/approve", "", 200)
		member := newClient()
		call(member, "", "POST", "/api/auth/login", `{"username":"second-member","password":"synthetic-password-456"}`, 200)
		if got := language(member); got != nil {
			t.Fatalf("second account inherited %q", *got)
		}
		call(member, csrfOf(member), "PUT", "/api/account/preferences", `{"language":"en"}`, 200)
		if got := language(admin); got == nil || *got != "fr" {
			t.Fatalf("administrator language became %v", got)
		}
	})
	t.Run("profile photo is validated persisted and removable", func(t *testing.T) {
		call(newClient(), "", "GET", "/api/account/avatar", "", 401)
		call(admin, "", "GET", "/api/account/avatar", "", 404)
		var imageData bytes.Buffer
		if err := png.Encode(&imageData, image.NewNRGBA(image.Rect(0, 0, 20, 10))); err != nil {
			t.Fatal(err)
		}
		call(admin, "", "PUT", "/api/account/avatar", imageData.String(), 403)
		call(admin, adminCSRF, "PUT", "/api/account/avatar", "not an image", 400)
		call(admin, adminCSRF, "PUT", "/api/account/avatar", imageData.String(), 204)
		data := call(admin, "", "GET", "/api/account/avatar", "", 200)
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || format != "png" || cfg.Width != 256 || cfg.Height != 256 {
			t.Fatalf("invalid stored avatar: %+v %s %v", cfg, format, err)
		}
		call(admin, adminCSRF, "DELETE", "/api/account/avatar", "", 204)
		call(admin, "", "GET", "/api/account/avatar", "", 404)
	})

}
