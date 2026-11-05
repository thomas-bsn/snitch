package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Administration: the Discord Bot card of Admin > Sign ups ----

type botView struct {
	Status        string  `json:"status"` // connected | not_configured | error
	BotName       *string `json:"bot_name"`
	LastError     *string `json:"last_error"`
	LastSyncAt    *string `json:"last_sync_at"`
	LastRefreshed int     `json:"last_refreshed"`
	LastFailed    int     `json:"last_failed"`
	Running       bool    `json:"running"`
}

func (s *Server) readBot(r *http.Request) (botView, error) {
	token, err := s.botToken(r.Context())
	if err != nil {
		return botView{}, err
	}
	var v botView
	var lastSync *time.Time
	var status string
	if err := s.Pool.QueryRow(r.Context(), `SELECT status,bot_name,last_error,last_sync_at,last_refreshed,last_failed,COALESCE(locked_until>now(),false) FROM discord_bot_state WHERE id=1`).Scan(&status, &v.BotName, &v.LastError, &lastSync, &v.LastRefreshed, &v.LastFailed, &v.Running); err != nil {
		return botView{}, err
	}
	if lastSync != nil {
		formatted := lastSync.UTC().Format(time.RFC3339)
		v.LastSyncAt = &formatted
	}
	v.Status = status
	if token == "" {
		// Removed, or unreadable with the current master key: either way nothing can run.
		v.Status, v.BotName, v.LastError = "not_configured", nil, nil
	}
	return v, nil
}

func (s *Server) getDiscordBot(w http.ResponseWriter, r *http.Request) {
	v, err := s.readBot(r)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, v)
}

// botFailure maps a Discord failure to the public error vocabulary.
func botFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errBotRejected):
		fail(w, 400, "invalid_discord_bot_token")
	case errors.Is(err, errDiscordLimited):
		fail(w, 429, "discord_rate_limited")
	default:
		fail(w, 502, "discord_unavailable")
	}
}

// putDiscordBot validates the token with Discord, then seals it. A token Discord refuses is never stored.
func (s *Server) putDiscordBot(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !discordBotTokenFormat.MatchString(in.Token) {
		fail(w, 400, "invalid_discord_bot_token")
		return
	}
	name, err := s.discordBotIdentity(r.Context(), in.Token)
	if err != nil {
		botFailure(w, err)
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err := s.Store.WithTx(tx).Set(r.Context(), discordBotTokenKey, in.Token); err != nil {
		s.internal(w, err)
		return
	}
	// A new token restarts the schedule: the next pass is due at once.
	if _, err := tx.Exec(r.Context(), `UPDATE discord_bot_state SET bot_name=$1,status='connected',last_error=NULL,last_sync_started_at=NULL WHERE id=1`, name); err != nil {
		s.internal(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	s.Log.Info("discord bot connected", "user_id", ss.User.ID)
	s.getDiscordBot(w, r)
}

// deleteDiscordBot disconnects the bot. Existing photos stay; only the synchronization stops.
func (s *Server) deleteDiscordBot(w http.ResponseWriter, r *http.Request) {
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err := s.Store.WithTx(tx).Set(r.Context(), discordBotTokenKey, ""); err != nil {
		s.internal(w, err)
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE discord_bot_state SET bot_name=NULL,status='connected',last_error=NULL,locked_until=NULL WHERE id=1`); err != nil {
		s.internal(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	s.Log.Info("discord bot disconnected", "user_id", ss.User.ID)
	w.WriteHeader(http.StatusNoContent)
}

// testDiscordBot checks the stored token with Discord and records the outcome on the card.
func (s *Server) testDiscordBot(w http.ResponseWriter, r *http.Request) {
	token, err := s.botToken(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	if token == "" {
		fail(w, 409, "discord_bot_not_configured")
		return
	}
	name, err := s.discordBotIdentity(r.Context(), token)
	switch {
	case err == nil:
		if _, err = s.Pool.Exec(r.Context(), `UPDATE discord_bot_state SET bot_name=$1,status='connected',last_error=NULL WHERE id=1`, name); err != nil {
			s.internal(w, err)
			return
		}
	case errors.Is(err, errBotRejected):
		s.markBotError(r.Context(), "discord_bot_rejected")
		botFailure(w, err)
		return
	default:
		botFailure(w, err)
		return
	}
	s.getDiscordBot(w, r)
}

// syncDiscordBot starts a global pass now, in the background. 409 when one is already running anywhere.
func (s *Server) syncDiscordBot(w http.ResponseWriter, r *http.Request) {
	token, err := s.botToken(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	if token == "" {
		fail(w, 409, "discord_bot_not_configured")
		return
	}
	ok, err := s.acquireAvatarPass(r.Context(), true)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !ok {
		fail(w, 409, "discord_sync_running")
		return
	}
	// The pass outlives the request: it follows the server's lifetime (or, in tests that never call Start, nothing).
	parent := s.backgroundContext
	if parent == nil {
		parent = context.WithoutCancel(r.Context())
	}
	s.avatarWG.Add(1)
	go func() {
		defer s.avatarWG.Done()
		s.runAvatarPass(parent, token)
	}()
	w.WriteHeader(http.StatusAccepted)
}

// ---- Account: the photo source choice ----

type avatarSettings struct {
	Source        string  `json:"source"` // discord | custom | none
	DiscordLinked bool    `json:"discord_linked"`
	HasCustom     bool    `json:"has_custom"`
	HasDiscord    bool    `json:"has_discord"`
	BotConfigured bool    `json:"bot_configured"`
	AutoSync      bool    `json:"auto_sync"`
	LastSyncedAt  *string `json:"last_synced_at"`
	LastError     *string `json:"last_error"`
}

func (s *Server) readAvatarSettings(r *http.Request, id int64) (avatarSettings, error) {
	var a avatarSettings
	var checked *time.Time
	err := s.Pool.QueryRow(r.Context(), `SELECT COALESCE(p.source,'none'),u.discord_id IS NOT NULL,
 EXISTS (SELECT 1 FROM account_avatars WHERE user_id=u.id AND source='custom'),
 EXISTS (SELECT 1 FROM account_avatars WHERE user_id=u.id AND source='discord'),
 p.discord_checked_at,p.discord_last_error
FROM users u LEFT JOIN avatar_preferences p ON p.user_id=u.id WHERE u.id=$1`, id).Scan(&a.Source, &a.DiscordLinked, &a.HasCustom, &a.HasDiscord, &checked, &a.LastError)
	if err != nil {
		return a, err
	}
	token, err := s.botToken(r.Context())
	if err != nil {
		return a, err
	}
	a.BotConfigured = token != ""
	a.AutoSync = a.DiscordLinked && a.BotConfigured && a.Source == avatarDiscord
	if checked != nil {
		formatted := checked.UTC().Format(time.RFC3339)
		a.LastSyncedAt = &formatted
	}
	return a, nil
}

func (s *Server) getAvatarSettings(w http.ResponseWriter, r *http.Request) {
	a, err := s.readAvatarSettings(r, r.Context().Value(sessionKey{}).(session).User.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, a)
}

// putAvatarSettings is the explicit source choice. Only the account's own images can be chosen.
func (s *Server) putAvatarSettings(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(sessionKey{}).(session).User.ID
	var in struct {
		Source string `json:"source"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Source != avatarDiscord && in.Source != avatarCustom && in.Source != avatarNone {
		fail(w, 400, "invalid_avatar_source")
		return
	}
	code := ""
	err := s.avatarTx(r.Context(), id, func(tx pgx.Tx) error {
		switch in.Source {
		case avatarCustom:
			if ok, err := hasAvatarImage(r.Context(), tx, id, avatarCustom); err != nil || !ok {
				code = "no_custom_avatar"
				return err
			}
		case avatarDiscord:
			var linked bool
			if err := tx.QueryRow(r.Context(), `SELECT discord_id IS NOT NULL FROM users WHERE id=$1`, id).Scan(&linked); err != nil || !linked {
				code = "discord_not_linked"
				return err
			}
		}
		return setAvatarSource(r.Context(), tx, id, in.Source)
	})
	if err != nil {
		s.internal(w, err)
		return
	}
	if code != "" {
		fail(w, 409, code)
		return
	}
	if in.Source == avatarDiscord {
		// Nothing to show yet: fetch it now when the bot allows it. A failure is only reported in the state.
		if a, err := s.readAvatarSettings(r, id); err == nil && !a.HasDiscord && a.BotConfigured {
			if token, err := s.botToken(r.Context()); err == nil && token != "" {
				_, _ = s.syncDiscordAvatar(r.Context(), id, token)
			}
		}
	}
	s.getAvatarSettings(w, r)
}

// refreshDiscordAvatar is the "Refresh Discord Picture" button.
func (s *Server) refreshDiscordAvatar(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(sessionKey{}).(session).User.ID
	a, err := s.readAvatarSettings(r, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !a.DiscordLinked {
		fail(w, 409, "discord_not_linked")
		return
	}
	token, err := s.botToken(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	if token == "" {
		fail(w, 409, "discord_bot_not_configured")
		return
	}
	// One Discord call per account every few seconds at most, whatever the number of clicks.
	cooldown := s.avatarCooldown
	if cooldown == 0 {
		cooldown = 30 * time.Second
	}
	// Compared by the database against its own clock: the application and PostgreSQL may not agree on the time.
	var tooSoon bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT COALESCE(discord_checked_at > now() - $2 * interval '1 microsecond',false) FROM avatar_preferences WHERE user_id=$1`, id, cooldown.Microseconds()).Scan(&tooSoon); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	if tooSoon {
		fail(w, 429, "avatar_sync_too_soon")
		return
	}
	if _, err := s.syncDiscordAvatar(r.Context(), id, token); err != nil {
		if errors.Is(err, errBotRejected) {
			s.markBotError(r.Context(), "discord_bot_rejected")
			fail(w, 502, "discord_unavailable")
			return
		}
		botFailure(w, err)
		return
	}
	s.getAvatarSettings(w, r)
}
