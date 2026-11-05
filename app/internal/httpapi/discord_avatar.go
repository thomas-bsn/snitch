package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// Discord avatar synchronization (the "Discord Bot - Profile Pictures" feature). The bot token is a sealed
// setting; the bot only reads public profiles (GET /users/{id}), it never acts in a server. Nothing here is
// ever allowed to block a sign-in: callers run it in the background or report its error separately.

const (
	discordBotTokenKey = "discord_bot_token"
	// avatarSyncEvery is how often the periodic pass runs over every linked account.
	avatarSyncEvery = 24 * time.Hour
	// avatarLease bounds how long an instance owns a pass; it is renewed at every account, so only a dead
	// instance's lease ever expires.
	avatarLease = 5 * time.Minute
)

var (
	discordBotTokenFormat = regexp.MustCompile(`^[A-Za-z0-9._-]{30,128}$`)
	discordSnowflake      = regexp.MustCompile(`^[0-9]{1,24}$`)
	discordAvatarHash     = regexp.MustCompile(`^(a_)?[0-9a-f]{32}$`)

	errBotNotConfigured = errors.New("discord bot not configured")
	errBotRejected      = errors.New("discord bot token rejected")
	errDiscordLimited   = errors.New("discord rate limited")
	errDiscordDown      = errors.New("discord unavailable")
	errAccountGone      = errors.New("discord account not found")
)

// discordCall performs one authenticated Bot API request. The path is built by the caller from validated
// identifiers only; the base address comes from the configuration, never from a client.
func (s *Server) discordCall(ctx context.Context, token, rawURL string, limit int64) (int, []byte, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return 0, nil, 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bot "+token)
	}
	req.Header.Set("User-Agent", "DiscordBot (https://github.com/thomas-bsn/snitch, 1)")
	resp, err := s.Client.Do(req)
	if err != nil {
		return 0, nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return resp.StatusCode, nil, 0, err
	}
	var wait time.Duration
	if resp.StatusCode == http.StatusTooManyRequests {
		if v, e := strconv.ParseFloat(resp.Header.Get("Retry-After"), 64); e == nil {
			wait = time.Duration(v * float64(time.Second))
		} else {
			var in struct {
				RetryAfter float64 `json:"retry_after"`
			}
			if json.Unmarshal(body, &in) == nil {
				wait = time.Duration(in.RetryAfter * float64(time.Second))
			}
		}
		if wait <= 0 {
			wait = time.Second
		}
	}
	return resp.StatusCode, body, wait, nil
}

// discordGet retries transient failures (network, 5xx, short rate limits) with a growing delay; a long rate
// limit is not waited for, it is reported so the caller reschedules instead of holding a worker.
func (s *Server) discordGet(ctx context.Context, step, token, rawURL string, limit int64) (int, []byte, error) {
	base := s.avatarRetryBase
	if base == 0 {
		base = time.Second
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		status, body, wait, err := s.discordCall(ctx, token, rawURL, limit)
		delay := base << attempt
		switch {
		case err != nil:
			if ctx.Err() != nil {
				return 0, nil, ctx.Err()
			}
			s.discordStepFailed(step, "request_failed", 0, nil, err)
			lastErr = errDiscordDown
		case status == http.StatusTooManyRequests:
			s.discordStepFailed(step, "rate_limited", status, body, nil)
			if wait > 30*time.Second {
				return status, nil, errDiscordLimited
			}
			delay, lastErr = max(wait, delay), errDiscordLimited
		case status >= 500:
			s.discordStepFailed(step, "http_status", status, body, nil)
			lastErr = errDiscordDown
		default:
			return status, body, nil
		}
		select {
		case <-ctx.Done():
			return 0, nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return 0, nil, lastErr
}

// discordBotIdentity asks Discord who the token belongs to. errBotRejected means the token is not valid.
func (s *Server) discordBotIdentity(ctx context.Context, token string) (string, error) {
	status, body, err := s.discordGet(ctx, "bot_identity", token, s.Config.DiscordAPIURL+"/users/@me", 16384)
	if err != nil {
		return "", err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		s.discordStepFailed("bot_identity", "http_status", status, body, nil)
		return "", errBotRejected
	}
	var me struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		GlobalName string `json:"global_name"`
	}
	if status != 200 || json.Unmarshal(body, &me) != nil || me.ID == "" {
		s.discordStepFailed("bot_identity", "invalid_response", status, body, nil)
		return "", errDiscordDown
	}
	name := me.GlobalName
	if name == "" {
		name = me.Username
	}
	return truncateDisplayName(name, 128), nil
}

func (s *Server) botToken(ctx context.Context) (string, error) {
	return s.Store.Get(ctx, discordBotTokenKey)
}

// syncOutcome tells what a synchronization of one account did.
type syncOutcome string

const (
	syncUpdated   syncOutcome = "updated"
	syncUnchanged syncOutcome = "unchanged"
	syncNoAvatar  syncOutcome = "no_avatar"
)

// syncDiscordAvatar refreshes the Discord image of one account. It never touches the personal photo or the
// displayed source: it only maintains the Discord image, under the account lock, and only when the account is
// still linked to the identity it asked about.
func (s *Server) syncDiscordAvatar(ctx context.Context, id int64, token string) (syncOutcome, error) {
	var discordID *string
	if err := s.Pool.QueryRow(ctx, `SELECT discord_id FROM users WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&discordID); err != nil {
		return "", err
	}
	if discordID == nil || !discordSnowflake.MatchString(*discordID) {
		return "", pgx.ErrNoRows
	}
	outcome, err := s.fetchAndStoreAvatar(ctx, id, *discordID, token)
	s.recordAvatarCheck(ctx, id, err)
	return outcome, err
}

func (s *Server) fetchAndStoreAvatar(ctx context.Context, id int64, discordID, token string) (syncOutcome, error) {
	status, body, err := s.discordGet(ctx, "avatar_profile", token, s.Config.DiscordAPIURL+"/users/"+discordID, 65536)
	if err != nil {
		return "", err
	}
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		s.discordStepFailed("avatar_profile", "http_status", status, body, nil)
		return "", errBotRejected
	case http.StatusNotFound:
		s.discordStepFailed("avatar_profile", "http_status", status, body, nil)
		return "", errAccountGone
	}
	var profile struct {
		Avatar     *string `json:"avatar"`
		Username   string  `json:"username"`
		GlobalName string  `json:"global_name"`
	}
	if status != 200 || json.Unmarshal(body, &profile) != nil {
		s.discordStepFailed("avatar_profile", "invalid_response", status, body, nil)
		return "", errDiscordDown
	}
	// Profile names refresh even when the avatar hash is unchanged. Never change a chosen Snitch name.
	if profile.Username != "" {
		name := profile.GlobalName
		if name == "" {
			name = profile.Username
		}
		_, err = s.Pool.Exec(ctx, `UPDATE users SET discord_username=$3,discord_name=$4 WHERE id=$1 AND discord_id=$2 AND deleted_at IS NULL`, id, discordID, truncateDisplayName(profile.Username, 128), truncateDisplayName(name, 128))
		if err != nil {
			return "", err
		}
	}
	if profile.Avatar == nil || *profile.Avatar == "" {
		return syncNoAvatar, s.storeDiscordAvatar(ctx, id, discordID, "", nil)
	}
	hash := *profile.Avatar
	if !discordAvatarHash.MatchString(hash) {
		s.discordStepFailed("avatar_profile", "invalid_response", status, nil, nil)
		return "", errDiscordDown
	}
	var known *string
	_ = s.Pool.QueryRow(ctx, `SELECT discord_hash FROM account_avatars WHERE user_id=$1 AND source='discord'`, id).Scan(&known)
	if known != nil && *known == hash {
		return syncUnchanged, nil
	}
	status, image, err := s.discordGet(ctx, "avatar_download", "", s.Config.DiscordCDNURL+"/avatars/"+discordID+"/"+hash+".png?size=256", maxAvatarBytes)
	if err != nil {
		return "", err
	}
	if status != 200 || len(image) > maxAvatarBytes {
		s.discordStepFailed("avatar_download", "http_status", status, nil, nil)
		return "", errDiscordDown
	}
	encoded, err := normalizeAvatar(image)
	if err != nil {
		s.discordStepFailed("avatar_download", "invalid_response", status, nil, nil)
		return "", errDiscordDown
	}
	return syncUpdated, s.storeDiscordAvatar(ctx, id, discordID, hash, encoded)
}

// storeDiscordAvatar writes (or, with an empty hash, removes) the Discord image. It does nothing when the
// account was unlinked or relinked to another identity while Discord was being asked.
func (s *Server) storeDiscordAvatar(ctx context.Context, id int64, discordID, hash string, png []byte) error {
	return s.avatarTx(ctx, id, func(tx pgx.Tx) error {
		var current *string
		if err := tx.QueryRow(ctx, `SELECT discord_id FROM users WHERE id=$1`, id).Scan(&current); err != nil {
			return err
		}
		if current == nil || *current != discordID {
			return nil
		}
		if hash == "" {
			if _, err := tx.Exec(ctx, `DELETE FROM account_avatars WHERE user_id=$1 AND source='discord'`, id); err != nil {
				return err
			}
		} else if _, err := tx.Exec(ctx, `INSERT INTO account_avatars(user_id,source,png,discord_hash) VALUES($1,'discord',$2,$3) ON CONFLICT(user_id,source) DO UPDATE SET png=EXCLUDED.png,discord_hash=EXCLUDED.discord_hash`, id, png, hash); err != nil {
			return err
		}
		// Only what is displayed needs a new validator: a Discord change never alters a personal photo.
		_, err := tx.Exec(ctx, `UPDATE avatar_preferences SET version=version+1 WHERE user_id=$1 AND source='discord'`, id)
		return err
	})
}

// recordAvatarCheck keeps the per-account state: success resets the backoff, a failure doubles it (5 minutes up
// to a day), so a broken account is not retried at every pass.
func (s *Server) recordAvatarCheck(ctx context.Context, id int64, syncErr error) {
	if ctx.Err() != nil {
		return
	}
	code := ""
	if syncErr != nil {
		code = avatarErrorCode(syncErr)
	}
	var err error
	if syncErr == nil {
		_, err = s.Pool.Exec(ctx, `INSERT INTO avatar_preferences(user_id,discord_checked_at) VALUES($1,now()) ON CONFLICT(user_id) DO UPDATE SET discord_checked_at=now(),discord_next_attempt_at=NULL,discord_failures=0,discord_last_error=NULL`, id)
	} else {
		_, err = s.Pool.Exec(ctx, `INSERT INTO avatar_preferences(user_id,discord_checked_at,discord_failures,discord_last_error,discord_next_attempt_at) VALUES($1,now(),1,$2,now()+interval '5 minutes')
ON CONFLICT(user_id) DO UPDATE SET discord_checked_at=now(),discord_failures=avatar_preferences.discord_failures+1,discord_last_error=$2,
 discord_next_attempt_at=now()+least(interval '24 hours', interval '5 minutes' * power(2, least(avatar_preferences.discord_failures, 8)))`, id, code)
	}
	if err != nil {
		s.Log.Error("avatar sync state not recorded", "kind", fmtError(err))
	}
}

func avatarErrorCode(err error) string {
	switch {
	case errors.Is(err, errBotRejected):
		return "discord_bot_rejected"
	case errors.Is(err, errDiscordLimited):
		return "discord_rate_limited"
	case errors.Is(err, errAccountGone):
		return "discord_account_not_found"
	case errors.Is(err, pgx.ErrNoRows):
		return "discord_not_linked"
	}
	return "discord_unavailable"
}

// triggerAvatarSync refreshes one account in the background, right after it was created or linked. The sign-in
// has already been answered: whatever happens here is only logged.
func (s *Server) triggerAvatarSync(id int64) {
	ctx := s.backgroundContext
	if ctx == nil {
		ctx = context.Background()
	}
	s.avatarWG.Add(1)
	go func() {
		defer s.avatarWG.Done()
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		token, err := s.botToken(ctx)
		if err != nil || token == "" {
			return
		}
		if _, err := s.syncDiscordAvatar(ctx, id, token); err != nil {
			s.Log.Warn("discord avatar sync failed", "user_id", id, "code", avatarErrorCode(err))
			if errors.Is(err, errBotRejected) {
				s.markBotError(ctx, "discord_bot_rejected")
			}
		}
	}()
}

func (s *Server) markBotError(ctx context.Context, code string) {
	_, _ = s.Pool.Exec(ctx, `UPDATE discord_bot_state SET status='error',last_error=$1 WHERE id=1`, code)
}

// acquireAvatarPass takes the pass lease for this instance. Without force it also requires the previous pass to be
// older than avatarSyncEvery, which is read from the database, so a restart or a second instance neither loses
// nor repeats the schedule.
func (s *Server) acquireAvatarPass(ctx context.Context, force bool) (bool, error) {
	tag, err := s.Pool.Exec(ctx, `UPDATE discord_bot_state SET locked_until=now()+$2 * interval '1 second',last_sync_started_at=now()
WHERE id=1 AND (locked_until IS NULL OR locked_until<now()) AND ($1 OR last_sync_started_at IS NULL OR last_sync_started_at<now()-$3 * interval '1 second')`, force, int64(avatarLease.Seconds()), int64(avatarSyncEvery.Seconds()))
	return err == nil && tag.RowsAffected() == 1, err
}

// runAvatarPass walks the linked accounts that are due. The caller holds the lease.
func (s *Server) runAvatarPass(ctx context.Context, token string) {
	refreshed, failed := 0, 0
	defer func() {
		// The final write must survive a cancelled context (shutdown): the lease is released either way.
		wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_, err := s.Pool.Exec(wctx, `UPDATE discord_bot_state SET locked_until=NULL,last_sync_at=now(),last_refreshed=$1,last_failed=$2 WHERE id=1`, refreshed, failed)
		if err != nil {
			s.Log.Error("avatar pass state not recorded", "kind", fmtError(err))
		}
	}()
	rows, err := s.Pool.Query(ctx, `SELECT u.id FROM users u LEFT JOIN avatar_preferences p ON p.user_id=u.id
WHERE u.discord_id IS NOT NULL AND u.deleted_at IS NULL AND (p.discord_next_attempt_at IS NULL OR p.discord_next_attempt_at<=now())
ORDER BY p.discord_checked_at NULLS FIRST,u.id`)
	if err != nil {
		s.Log.Error("avatar pass failed", "kind", fmtError(err))
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		_, _ = s.Pool.Exec(ctx, `UPDATE discord_bot_state SET locked_until=now()+$1 * interval '1 second' WHERE id=1`, int64(avatarLease.Seconds()))
		outcome, err := s.syncDiscordAvatar(ctx, id, token)
		switch {
		case err == nil:
			if outcome == syncUpdated {
				refreshed++
			}
		case errors.Is(err, errBotRejected):
			// Every following account would fail the same way: stop and surface it to the administrator.
			failed++
			s.markBotError(ctx, "discord_bot_rejected")
			s.Log.Warn("discord avatar pass stopped", "code", "discord_bot_rejected")
			return
		case errors.Is(err, errDiscordLimited):
			failed++
			s.Log.Warn("discord avatar pass stopped", "code", "discord_rate_limited")
			return
		default:
			failed++
		}
		if pause := s.avatarPause; pause > 0 {
			time.Sleep(pause)
		}
	}
	_, _ = s.Pool.Exec(ctx, `UPDATE discord_bot_state SET status='connected',last_error=NULL WHERE id=1 AND status='error' AND last_error='discord_bot_rejected'`)
}

// serveDiscordAvatars is the scheduler. It wakes up regularly and starts a pass only when the persisted
// schedule says one is due and no other instance holds the lease.
func (s *Server) serveDiscordAvatars(ctx context.Context) {
	tick := s.avatarTick
	if tick == 0 {
		tick = 10 * time.Minute
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		s.scheduledAvatarPass(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) scheduledAvatarPass(ctx context.Context) {
	token, err := s.botToken(ctx)
	if err != nil || token == "" {
		return
	}
	if ok, err := s.acquireAvatarPass(ctx, false); err != nil || !ok {
		return
	}
	s.runAvatarPass(ctx, token)
}
