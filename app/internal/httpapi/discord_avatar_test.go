//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

// A synthetic bot token, assembled at run time so that no literal in the source looks like a real credential.
var botToken = strings.Join([]string{strings.Repeat("A", 24), "Fixt", strings.Repeat("z", 27)}, ".")

const (
	hashRed   = "11111111111111111111111111111111"
	hashBlue  = "22222222222222222222222222222222"
	hashGreen = "a_33333333333333333333333333333333"
)

// fakeDiscord is the whole Discord surface Snitch talks to: OAuth, the Bot API and the image CDN.
type fakeDiscord struct {
	mu          sync.Mutex
	avatars     map[string]string // discord id -> avatar hash ("" = none, absent = unknown account)
	oauthID     string
	botStatus   int // 0 = normal; otherwise the status every Bot API call answers
	retryAfter  string
	limitedNext atomic.Int64 // that many Bot API calls answer 429 (Retry-After retryAfter), then normally
	profileHits atomic.Int64
	downloads   atomic.Int64
}

func (d *fakeDiscord) set(id, hash string) { d.mu.Lock(); d.avatars[id] = hash; d.mu.Unlock() }
func (d *fakeDiscord) mode(status int, retry string) {
	d.mu.Lock()
	d.botStatus, d.retryAfter = status, retry
	d.mu.Unlock()
}

func colourOf(hash string) color.NRGBA {
	switch strings.TrimPrefix(hash, "a_")[0] {
	case '1':
		return color.NRGBA{R: 255, A: 255}
	case '2':
		return color.NRGBA{B: 255, A: 255}
	}
	return color.NRGBA{G: 255, A: 255}
}

func solid(c color.NRGBA, w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func (d *fakeDiscord) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	jsonReply := func(status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	switch {
	case r.URL.Path == "/token":
		jsonReply(200, map[string]string{"access_token": "fixture-token"})
	case r.URL.Path == "/me":
		d.mu.Lock()
		id := d.oauthID
		d.mu.Unlock()
		jsonReply(200, map[string]string{"id": id, "username": "user" + id, "global_name": "User " + id})
	case strings.HasPrefix(r.URL.Path, "/cdn/avatars/"):
		d.downloads.Add(1)
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/cdn/avatars/"), "/")
		hash := strings.TrimSuffix(parts[len(parts)-1], ".png")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(solid(colourOf(hash), 128, 128))
	case strings.HasPrefix(r.URL.Path, "/api/users/"):
		d.mu.Lock()
		status, retry := d.botStatus, d.retryAfter
		d.mu.Unlock()
		if r.Header.Get("Authorization") != "Bot "+botToken {
			jsonReply(401, map[string]any{"code": 0, "message": "401: Unauthorized"})
			return
		}
		if status == 0 && d.limitedNext.Load() <= 0 {
			d.limitedNext.Store(0)
		}
		if status == 429 || (status == 0 && d.limitedNext.Add(-1) >= 0) {
			w.Header().Set("Retry-After", retry)
			jsonReply(429, map[string]any{"message": "You are being rate limited.", "retry_after": 1.5, "global": false})
			return
		}
		if status != 0 {
			jsonReply(status, map[string]any{"message": "boom"})
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/users/")
		if id == "@me" {
			jsonReply(200, map[string]any{"id": "999", "username": "snitch-bot", "bot": true})
			return
		}
		d.profileHits.Add(1)
		d.mu.Lock()
		hash, known := d.avatars[id]
		d.mu.Unlock()
		if !known {
			jsonReply(404, map[string]any{"code": 10013, "message": "Unknown User"})
			return
		}
		var avatar any
		if hash != "" {
			avatar = hash
		}
		jsonReply(200, map[string]any{"id": id, "username": "user" + id, "avatar": avatar})
	default:
		http.NotFound(w, r)
	}
}

func testDiscordAvatars(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	var logs bytes.Buffer
	var logsMu sync.Mutex
	logger := slog.New(slog.NewJSONHandler(syncWriter{&logs, &logsMu}, nil))
	if _, err := pool.Exec(ctx, `UPDATE discord_bot_state SET bot_name=NULL,status='connected',last_error=NULL,last_sync_started_at=NULL,last_sync_at=NULL,last_refreshed=0,last_failed=0,locked_until=NULL WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	fake := &fakeDiscord{avatars: map[string]string{}, oauthID: "100"}
	provider := httptest.NewServer(fake)
	defer provider.Close()
	key := bytes.Repeat([]byte{7}, 32)
	newServer := func() (*Server, *httptest.Server) {
		s := New(pool, config.Config{MasterKey: key}, logger)
		s.avatarRetryBase, s.avatarCooldown = time.Millisecond, time.Nanosecond
		ts := httptest.NewUnstartedServer(s.Handler())
		s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
		ts.Start()
		s.Config.DiscordTokenURL, s.Config.DiscordUserURL, s.Config.DiscordAuthorizeURL = provider.URL+"/token", provider.URL+"/me", provider.URL+"/authorize"
		s.Config.DiscordAPIURL, s.Config.DiscordCDNURL = provider.URL+"/api", provider.URL+"/cdn"
		return s, ts
	}
	s, ts := newServer()
	defer ts.Close()

	newClient := func() *http.Client {
		j, _ := cookiejar.New(nil)
		return &http.Client{Jar: j, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	type who struct {
		c    *http.Client
		csrf string
	}
	do := func(base string, w who, method, path, body string, want int) (*http.Response, []byte) {
		t.Helper()
		r, _ := http.NewRequest(method, base+path, strings.NewReader(body))
		r.Header.Set("Origin", base)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Token", w.csrf)
		resp, err := w.c.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("%s %s: %d %s want %d", method, path, resp.StatusCode, b, want)
		}
		return resp, b
	}
	call := func(w who, method, path, body string, want int) (*http.Response, []byte) {
		t.Helper()
		s.rates.Clear()
		return do(ts.URL, w, method, path, body, want)
	}
	refreshCSRF := func(w *who) {
		_, b := call(*w, "GET", "/api/auth/me", "", 200)
		var m struct {
			CSRF string `json:"csrf_token"`
		}
		json.Unmarshal(b, &m)
		w.csrf = m.CSRF
	}
	// The owner.
	owner := who{c: newClient()}
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]string{"code": code, "username": "owner", "password": "synthetic-password"})
	call(owner, "POST", "/api/setup", string(b), 201)
	refreshCSRF(&owner)
	call(owner, "PUT", "/api/admin/settings", `{"client_id":"fixture-app","client_secret":"fixture-secret","registration_approval":false}`, 200)

	// Signs in (or up) through the fake Discord as the given identity.
	oauth := func(w who, identity string, link bool) *http.Response {
		t.Helper()
		fake.mu.Lock()
		fake.oauthID = identity
		fake.mu.Unlock()
		path, csrf := "/api/auth/discord/start", ""
		if link {
			path, csrf = "/api/auth/discord/link", w.csrf
		}
		_, b := call(who{c: w.c, csrf: csrf}, "POST", path, "", 200)
		var out struct{ URL string }
		json.Unmarshal(b, &out)
		u, _ := url.Parse(out.URL)
		resp, _ := call(w, "GET", "/api/auth/discord/callback?code=fixture-code&state="+u.Query().Get("state"), "", 303)
		s.avatarWG.Wait()
		return resp
	}
	discordUser := func(identity string) who {
		t.Helper()
		w := who{c: newClient()}
		oauth(w, identity, false)
		refreshCSRF(&w)
		return w
	}
	localUser := func(name string) who {
		t.Helper()
		w := who{c: newClient()}
		call(w, "POST", "/api/auth/register", fmt.Sprintf(`{"username":%q,"password":"synthetic-password"}`, name), 201)
		call(w, "POST", "/api/auth/login", fmt.Sprintf(`{"username":%q,"password":"synthetic-password"}`, name), 200)
		refreshCSRF(&w)
		return w
	}
	type settings struct {
		Source        string  `json:"source"`
		DiscordLinked bool    `json:"discord_linked"`
		HasCustom     bool    `json:"has_custom"`
		HasDiscord    bool    `json:"has_discord"`
		BotConfigured bool    `json:"bot_configured"`
		AutoSync      bool    `json:"auto_sync"`
		LastError     *string `json:"last_error"`
	}
	state := func(w who) settings {
		t.Helper()
		_, b := call(w, "GET", "/api/account/avatar-settings", "", 200)
		var a settings
		if err := json.Unmarshal(b, &a); err != nil {
			t.Fatal(err)
		}
		return a
	}
	shown := func(w who) (color.NRGBA, bool) { // the colour of the displayed photo, false when initials show
		t.Helper()
		r, _ := http.NewRequest("GET", ts.URL+"/api/account/avatar", nil)
		resp, err := w.c.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == 404 {
			return color.NRGBA{}, false
		}
		if resp.StatusCode != 200 || resp.Header.Get("Cache-Control") != "private, no-cache" || resp.Header.Get("ETag") == "" {
			t.Fatalf("avatar: %d %v", resp.StatusCode, resp.Header)
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil || img.Bounds().Dx() != 256 || img.Bounds().Dy() != 256 {
			t.Fatalf("stored avatar: %v", err)
		}
		return color.NRGBAModel.Convert(img.At(128, 128)).(color.NRGBA), true
	}
	etag := func(w who) string {
		r, _ := http.NewRequest("GET", ts.URL+"/api/account/avatar", nil)
		resp, _ := w.c.Do(r)
		resp.Body.Close()
		return resp.Header.Get("ETag")
	}
	wantShown := func(w who, want color.NRGBA) {
		t.Helper()
		if got, ok := shown(w); !ok || got != want {
			t.Fatalf("displayed %v (photo=%v), want %v", got, ok, want)
		}
	}
	wantInitials := func(w who) {
		t.Helper()
		if _, ok := shown(w); ok {
			t.Fatal("a photo is shown where initials were expected")
		}
	}
	red, blue, green := colourOf(hashRed), colourOf(hashBlue), colourOf(hashGreen)
	upload := func(w who, c color.NRGBA) {
		t.Helper()
		call(w, "PUT", "/api/account/avatar", string(solid(c, 64, 64)), 204)
	}
	userID := func(name string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	connect := func() {
		t.Helper()
		call(owner, "PUT", "/api/admin/discord-bot", fmt.Sprintf(`{"token":%q}`, botToken), 200)
	}

	t.Run("bot endpoints are restricted and the token is validated before it is stored", func(t *testing.T) {
		call(who{c: newClient()}, "GET", "/api/admin/discord-bot", "", 401)
		member := localUser("plain-member")
		call(member, "GET", "/api/admin/discord-bot", "", 403)
		call(member, "PUT", "/api/admin/discord-bot", fmt.Sprintf(`{"token":%q}`, botToken), 403)
		// An Administrator reads the card and starts a pass, but cannot configure the token.
		if _, err := pool.Exec(ctx, `UPDATE users SET role=$1 WHERE username='plain-member'`, roleAdministrator); err != nil {
			t.Fatal(err)
		}
		call(member, "GET", "/api/admin/discord-bot", "", 200)
		for _, op := range [][2]string{{"PUT", fmt.Sprintf(`{"token":%q}`, botToken)}, {"DELETE", ""}} {
			_, b := call(member, op[0], "/api/admin/discord-bot", op[1], 403)
			if !bytes.Contains(b, []byte("super_administrator_required")) {
				t.Fatal(string(b))
			}
		}
		call(member, "POST", "/api/admin/discord-bot/test", "", 403)
		call(member, "POST", "/api/admin/discord-bot/sync", "", 409) // allowed, but no bot yet
		if _, err := pool.Exec(ctx, `UPDATE users SET role='user' WHERE username='plain-member'`); err != nil {
			t.Fatal(err)
		}
		call(owner, "PUT", "/api/admin/discord-bot", `{"token":"short"}`, 400)
		bad := strings.Repeat("x", 59)
		call(owner, "PUT", "/api/admin/discord-bot", fmt.Sprintf(`{"token":%q}`, bad), 400) // Discord answers 401
		_, b := call(owner, "GET", "/api/admin/discord-bot", "", 200)
		if !bytes.Contains(b, []byte(`"not_configured"`)) {
			t.Fatal(string(b))
		}
		var stored int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM settings WHERE key=$1`, discordBotTokenKey).Scan(&stored)
		if stored != 0 {
			t.Fatal("a refused token was stored")
		}
		fake.mode(500, "")
		call(owner, "PUT", "/api/admin/discord-bot", fmt.Sprintf(`{"token":%q}`, botToken), 502)
		fake.mode(429, "120")
		call(owner, "PUT", "/api/admin/discord-bot", fmt.Sprintf(`{"token":%q}`, botToken), 429)
		fake.mode(0, "")
		call(owner, "POST", "/api/admin/discord-bot/test", "", 409)
		connect()
		_, b = call(owner, "GET", "/api/admin/discord-bot", "", 200)
		if !bytes.Contains(b, []byte(`"connected"`)) || !bytes.Contains(b, []byte(`snitch-bot`)) || bytes.Contains(b, []byte(botToken)) {
			t.Fatal(string(b))
		}
		var sealed []byte
		_ = pool.QueryRow(ctx, `SELECT value FROM settings WHERE key=$1`, discordBotTokenKey).Scan(&sealed)
		if len(sealed) == 0 || bytes.Contains(sealed, []byte(botToken)) {
			t.Fatal("the token is not sealed")
		}
		call(owner, "POST", "/api/admin/discord-bot/test", "", 200)
	})

	t.Run("an account created through Discord shows its Discord photo", func(t *testing.T) {
		fake.set("201", hashRed)
		w := discordUser("201")
		a := state(w)
		if a.Source != "discord" || !a.HasDiscord || a.HasCustom || !a.AutoSync || !a.BotConfigured || !a.DiscordLinked {
			t.Fatalf("%+v", a)
		}
		wantShown(w, red)
	})

	t.Run("a changed Discord avatar replaces the image and an unchanged one costs no download", func(t *testing.T) {
		w := discordUser("202")
		fake.set("202", hashRed)
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 200)
		wantShown(w, red)
		first, downloads := etag(w), fake.downloads.Load()
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 200)
		if fake.downloads.Load() != downloads || etag(w) != first {
			t.Fatal("an unchanged avatar was downloaded again")
		}
		fake.set("202", hashBlue)
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 200)
		wantShown(w, blue)
		if etag(w) == first {
			t.Fatal("the validator did not change with the photo")
		}
		// The browser revalidates cheaply: same validator, no body.
		r, _ := http.NewRequest("GET", ts.URL+"/api/account/avatar", nil)
		r.Header.Set("If-None-Match", etag(w))
		resp, _ := w.c.Do(r)
		resp.Body.Close()
		if resp.StatusCode != 304 {
			t.Fatal(resp.StatusCode)
		}
		// Animated avatars (a_ hash) are fetched as still PNG too.
		fake.set("202", hashGreen)
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 200)
		wantShown(w, green)
		// Discord removes the avatar: initials, no error.
		fake.set("202", "")
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 200)
		wantInitials(w)
		if a := state(w); a.HasDiscord || a.Source != "discord" || a.LastError != nil {
			t.Fatalf("%+v", a)
		}
	})

	t.Run("a Discord user without an avatar shows initials and signs in normally", func(t *testing.T) {
		fake.set("203", "")
		w := discordUser("203")
		wantInitials(w)
		if a := state(w); a.Source != "discord" || a.HasDiscord {
			t.Fatalf("%+v", a)
		}
	})

	t.Run("linking Discord to a local account without a photo adopts the Discord photo", func(t *testing.T) {
		fake.set("301", hashBlue)
		w := localUser("local-no-photo")
		wantInitials(w)
		oauth(w, "301", true)
		if a := state(w); a.Source != "discord" || !a.HasDiscord {
			t.Fatalf("%+v", a)
		}
		wantShown(w, blue)
	})

	t.Run("a personal photo survives linking, uploads and every synchronization", func(t *testing.T) {
		fake.set("302", hashRed)
		w := localUser("local-with-photo")
		upload(w, green)
		oauth(w, "302", true)
		a := state(w)
		if a.Source != "custom" || !a.HasCustom || !a.HasDiscord {
			t.Fatalf("%+v", a)
		}
		wantShown(w, green)
		fake.set("302", hashBlue)
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 200)
		call(owner, "POST", "/api/admin/discord-bot/sync", "", 202)
		s.avatarWG.Wait()
		wantShown(w, green)
		// Back to Discord on purpose, then to the personal photo again: both images were kept.
		call(w, "PUT", "/api/account/avatar-settings", `{"source":"discord"}`, 200)
		wantShown(w, blue)
		call(w, "PUT", "/api/account/avatar-settings", `{"source":"custom"}`, 200)
		wantShown(w, green)
		// A new upload makes custom the active mode at once.
		call(w, "PUT", "/api/account/avatar-settings", `{"source":"discord"}`, 200)
		upload(w, red)
		if a := state(w); a.Source != "custom" {
			t.Fatalf("%+v", a)
		}
		wantShown(w, red)
		// Removing it falls back to the Discord photo; the choice is not lost.
		call(w, "DELETE", "/api/account/avatar", "", 204)
		if a := state(w); a.Source != "discord" || a.HasCustom {
			t.Fatalf("%+v", a)
		}
		wantShown(w, blue)
		call(w, "PUT", "/api/account/avatar-settings", `{"source":"custom"}`, 409)
		call(w, "PUT", "/api/account/avatar-settings", `{"source":"bogus"}`, 400)
		localOnly := localUser("local-never-linked")
		call(localOnly, "PUT", "/api/account/avatar-settings", `{"source":"discord"}`, 409)
		call(localOnly, "POST", "/api/account/avatar/discord-refresh", "", 409)
	})

	t.Run("a personal upload racing the synchronization always wins", func(t *testing.T) {
		fake.set("303", hashRed)
		w := localUser("racer")
		oauth(w, "303", true)
		custom := solid(blue, 64, 64)
		for i := 0; i < 8; i++ {
			fake.set("303", []string{hashRed, hashBlue, hashGreen}[i%3])
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); do(ts.URL, w, "POST", "/api/account/avatar/discord-refresh", "", 200) }()
			go func() { defer wg.Done(); do(ts.URL, w, "PUT", "/api/account/avatar", string(custom), 204) }()
			wg.Wait()
			if a := state(w); a.Source != "custom" || !a.HasCustom {
				t.Fatalf("round %d: %+v", i, a)
			}
			wantShown(w, blue)
			call(w, "PUT", "/api/account/avatar-settings", `{"source":"discord"}`, 200)
		}
	})

	t.Run("unlinking Discord stops synchronization and keeps the personal photo", func(t *testing.T) {
		fake.set("401", hashRed)
		w := discordUser("401")
		call(w, "POST", "/api/auth/logout", "", 200)
		// A Discord-only account cannot unlink (no password): link Discord to a local account instead.
		fake.set("402", hashRed)
		local := localUser("unlinker")
		oauth(local, "402", true)
		wantShown(local, red)
		call(local, "DELETE", "/api/account/discord", "", 200)
		a := state(local)
		if a.Source != "none" || a.HasDiscord || a.DiscordLinked {
			t.Fatalf("%+v", a)
		}
		wantInitials(local)
		call(owner, "POST", "/api/admin/discord-bot/sync", "", 202)
		s.avatarWG.Wait()
		wantInitials(local)
		// With a personal photo, unlinking leaves it displayed.
		fake.set("403", hashBlue)
		other := localUser("unlinker-custom")
		upload(other, green)
		oauth(other, "403", true)
		call(other, "DELETE", "/api/account/discord", "", 200)
		if a := state(other); a.Source != "custom" || !a.HasCustom || a.HasDiscord {
			t.Fatalf("%+v", a)
		}
		wantShown(other, green)
	})

	t.Run("Discord failures never block sign-in or linking", func(t *testing.T) {
		fake.set("501", hashRed)
		fake.mode(500, "")
		w := discordUser("501") // the helper fails the test unless the callback answers 303
		call(w, "GET", "/api/auth/me", "", 200)
		wantInitials(w)
		var failures int
		_ = pool.QueryRow(ctx, `SELECT discord_failures FROM avatar_preferences WHERE user_id=$1`, userID("discord_501")).Scan(&failures)
		if failures != 1 {
			t.Fatal("the failure was not recorded", failures)
		}
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 502)
		fake.mode(429, "120")
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 429)
		// Recovery: the next refresh works and resets the backoff.
		fake.mode(0, "")
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 200)
		wantShown(w, red)
		if err := pool.QueryRow(ctx, `SELECT discord_failures FROM avatar_preferences WHERE user_id=$1`, userID("discord_501")).Scan(&failures); err != nil || failures != 0 {
			t.Fatal(err, failures)
		}
	})

	t.Run("a short rate limit is waited for and a long one stops the pass", func(t *testing.T) {
		fake.set("601", hashRed)
		fake.mode(0, "")
		w := discordUser("601")
		wantShown(w, red)
		// Two short 429s (Retry-After in seconds) are waited for, then the refresh succeeds.
		fake.set("601", hashBlue)
		fake.mode(0, "0.01")
		fake.limitedNext.Store(2)
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 200)
		wantShown(w, blue)
		// A long limit stops the pass, counts the failure and keeps what was there.
		fake.mode(429, "120")
		fake.set("601", hashRed)
		call(owner, "POST", "/api/admin/discord-bot/sync", "", 202)
		s.avatarWG.Wait()
		_, b := call(owner, "GET", "/api/admin/discord-bot", "", 200)
		var bot struct {
			Failed int  `json:"last_failed"`
			Run    bool `json:"running"`
		}
		json.Unmarshal(b, &bot)
		if bot.Failed < 1 || bot.Run {
			t.Fatal(string(b))
		}
		fake.mode(0, "")
		wantShown(w, blue)
	})

	t.Run("a global pass reports its counts, never overlaps and honours the persisted schedule", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE avatar_preferences SET discord_next_attempt_at=NULL,discord_failures=0`); err != nil {
			t.Fatal(err)
		}
		fake.mode(0, "")
		call(owner, "POST", "/api/admin/discord-bot/sync", "", 202)
		s.avatarWG.Wait()
		_, b := call(owner, "GET", "/api/admin/discord-bot", "", 200)
		var bot struct {
			Refreshed int     `json:"last_refreshed"`
			Failed    int     `json:"last_failed"`
			At        *string `json:"last_sync_at"`
		}
		json.Unmarshal(b, &bot)
		if bot.At == nil || bot.Refreshed < 1 {
			t.Fatal(string(b))
		}
		// Another instance (or a previous run) holds the lease: no second pass.
		if _, err := pool.Exec(ctx, `UPDATE discord_bot_state SET locked_until=now()+interval '5 minutes'`); err != nil {
			t.Fatal(err)
		}
		call(owner, "POST", "/api/admin/discord-bot/sync", "", 409)
		if _, err := pool.Exec(ctx, `UPDATE discord_bot_state SET locked_until=NULL`); err != nil {
			t.Fatal(err)
		}
		// The scheduler does nothing while the last pass is recent, and runs when it is a day old.
		hits := fake.profileHits.Load()
		s.scheduledAvatarPass(ctx)
		if fake.profileHits.Load() != hits {
			t.Fatal("the scheduler ran before its time")
		}
		if _, err := pool.Exec(ctx, `UPDATE discord_bot_state SET last_sync_started_at=now()-interval '25 hours'; UPDATE avatar_preferences SET discord_next_attempt_at=NULL`); err != nil {
			t.Fatal(err)
		}
		s.scheduledAvatarPass(ctx)
		if fake.profileHits.Load() == hits {
			t.Fatal("the scheduler did not run when due")
		}
		// Two instances racing for the same due pass: exactly one gets the lease.
		if _, err := pool.Exec(ctx, `UPDATE discord_bot_state SET last_sync_started_at=now()-interval '25 hours',locked_until=NULL`); err != nil {
			t.Fatal(err)
		}
		var won atomic.Int64
		var wg sync.WaitGroup
		for i := 0; i < 6; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if ok, err := s.acquireAvatarPass(ctx, false); err == nil && ok {
					won.Add(1)
				}
			}()
		}
		wg.Wait()
		if won.Load() != 1 {
			t.Fatalf("%d instances took the lease", won.Load())
		}
		pool.Exec(ctx, `UPDATE discord_bot_state SET locked_until=NULL`)
	})

	t.Run("a Discord account that vanished is recorded and backed off", func(t *testing.T) {
		fake.set("701", hashRed)
		w := discordUser("701")
		fake.mu.Lock()
		delete(fake.avatars, "701")
		fake.mu.Unlock()
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 502)
		a := state(w)
		if a.LastError == nil || *a.LastError != "discord_account_not_found" || !a.HasDiscord {
			t.Fatalf("%+v", a)
		}
		wantShown(w, red) // the last known photo stays
	})

	t.Run("disconnecting the bot and restarting keep photos, secrets and schedule", func(t *testing.T) {
		fake.set("801", hashBlue)
		w := discordUser("801")
		wantShown(w, blue)
		call(owner, "DELETE", "/api/admin/discord-bot", "", 204)
		_, b := call(owner, "GET", "/api/admin/discord-bot", "", 200)
		if !bytes.Contains(b, []byte(`"not_configured"`)) {
			t.Fatal(string(b))
		}
		wantShown(w, blue)
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 409)
		call(owner, "POST", "/api/admin/discord-bot/sync", "", 409)
		if a := state(w); a.BotConfigured || a.AutoSync {
			t.Fatalf("%+v", a)
		}
		hits := fake.profileHits.Load()
		s.scheduledAvatarPass(ctx)
		if fake.profileHits.Load() != hits {
			t.Fatal("a disconnected bot synchronized")
		}
		// A new server on the same database and master key: the redeployment case.
		connect()
		ts.Close()
		s.Close()
		s, ts = newServer()
		owner = who{c: owner.c}
		refreshCSRF(&owner)
		_, b = call(owner, "GET", "/api/admin/discord-bot", "", 200)
		if !bytes.Contains(b, []byte(`"connected"`)) || !bytes.Contains(b, []byte(`snitch-bot`)) {
			t.Fatal(string(b))
		}
		w.csrf = ""
		refreshCSRF(&w)
		wantShown(w, blue)
		fake.set("801", hashRed)
		call(w, "POST", "/api/account/avatar/discord-refresh", "", 200)
		wantShown(w, red)
		// A restart after the disconnection keeps it disconnected.
		call(owner, "DELETE", "/api/admin/discord-bot", "", 204)
		ts.Close()
		s.Close()
		s, ts = newServer()
		refreshCSRF(&owner)
		_, b = call(owner, "GET", "/api/admin/discord-bot", "", 200)
		if !bytes.Contains(b, []byte(`"not_configured"`)) {
			t.Fatal(string(b))
		}
	})

	t.Run("no secret reaches the logs", func(t *testing.T) {
		logsMu.Lock()
		defer logsMu.Unlock()
		for _, secret := range []string{botToken, "fixture-token", "fixture-secret"} {
			if strings.Contains(logs.String(), secret) {
				t.Fatalf("log leaks %q", secret)
			}
		}
	})
}

type syncWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

func (s syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
