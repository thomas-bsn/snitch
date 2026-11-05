package httpapi

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"snitch/internal/auth"
)

// supportedLanguages lists the interface languages an account can choose.
var supportedLanguages = []string{"en", "fr"}

type accountPreferences struct {
	Language *string `json:"language"`
}

func (s *Server) getAccountPreferences(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var language *string
	if err := s.Pool.QueryRow(r.Context(), `SELECT language FROM users WHERE id=$1`, ss.User.ID).Scan(&language); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, accountPreferences{Language: language})
}

func (s *Server) putAccountPreferences(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var in accountPreferences
	if !decode(w, r, &in) {
		return
	}
	if in.Language == nil {
		fail(w, 400, "invalid_request")
		return
	}
	if !slices.Contains(supportedLanguages, *in.Language) {
		fail(w, 400, "invalid_language")
		return
	}
	if _, err := s.Pool.Exec(r.Context(), `UPDATE users SET language=$1 WHERE id=$2`, *in.Language, ss.User.ID); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, accountPreferences{Language: in.Language})
}

// unlinkDiscord removes the identity only when the account already has a local password.
// The predicate is repeated in the UPDATE so a stale UI state cannot lock out an account.
func (s *Server) unlinkDiscord(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `UPDATE users SET discord_id=NULL,discord_name=NULL,discord_username=NULL WHERE id=$1 AND discord_id IS NOT NULL AND password_hash IS NOT NULL`, ss.User.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 1 {
		// Synchronization stops with the identity; the downloaded photo is dropped, the personal one is kept.
		if err := dropDiscordAvatar(r.Context(), tx, ss.User.ID); err != nil {
			s.internal(w, err)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			s.internal(w, err)
			return
		}
		reply(w, 200, map[string]string{"status": "ok"})
		return
	}
	tx.Rollback(r.Context())
	var linked, canUnlink bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT discord_id IS NOT NULL,password_hash IS NOT NULL FROM users WHERE id=$1`, ss.User.ID).Scan(&linked, &canUnlink); err != nil {
		s.internal(w, err)
		return
	}
	if !linked {
		fail(w, 409, "discord_not_linked")
		return
	}
	if !canUnlink {
		fail(w, 409, "discord_local_login_required")
		return
	}
	fail(w, 409, "discord_unlink_unavailable")
}

// putAccountProfile changes only this account's chosen display name and local login.
func (s *Server) putAccountProfile(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var in struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if !validAccountUsername(in.Username) || utf8.RuneCountInString(in.DisplayName) < 1 || utf8.RuneCountInString(in.DisplayName) > 100 || strings.ContainsFunc(in.DisplayName, unicode.IsControl) {
		fail(w, 400, "invalid_profile")
		return
	}
	_, err := s.Pool.Exec(r.Context(), `UPDATE users SET username=$2,display_name=$3 WHERE id=$1 AND approved AND deleted_at IS NULL`, ss.User.ID, in.Username, in.DisplayName)
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		fail(w, 409, "username_unavailable")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]string{"status": "ok"})
}

// Password changes are serialized with account closure, Discord unlink and each other.
// Existing local credentials must be proved; a Discord-only session can establish its first password.
func (s *Server) putAccountPassword(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decode(w, r, &in) {
		return
	}
	n := utf8.RuneCountInString(in.NewPassword)
	if n < 12 || n > 256 || utf8.RuneCountInString(in.CurrentPassword) > 256 {
		fail(w, 400, "invalid_password_format")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var hash *string
	var approved bool
	if err = tx.QueryRow(r.Context(), `SELECT password_hash,approved FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, ss.User.ID).Scan(&hash, &approved); err != nil {
		s.internal(w, err)
		return
	}
	cookie, _ := sessionCookie(r)
	var live bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>now())`, auth.Hash(cookie.Value), ss.User.ID).Scan(&live); err != nil {
		s.internal(w, err)
		return
	}
	if !approved || !live {
		fail(w, 401, "unauthorized")
		return
	}
	if hash != nil && !auth.Verify(*hash, in.CurrentPassword) {
		fail(w, 403, "current_password_incorrect")
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE users SET password_hash=$2,display_name=COALESCE(display_name,CASE WHEN starts_with(username,'discord_') THEN left(discord_name,100) END) WHERE id=$1`, ss.User.ID, auth.Password(in.NewPassword)); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), `DELETE FROM sessions WHERE user_id=$1 AND token_hash<>$2`, ss.User.ID, auth.Hash(cookie.Value)); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]string{"status": "ok"})
}

func validAccountUsername(name string) bool {
	n := utf8.RuneCountInString(name)
	return n >= 3 && n <= 64 && strings.TrimSpace(name) == name && !strings.ContainsFunc(name, unicode.IsControl)
}
