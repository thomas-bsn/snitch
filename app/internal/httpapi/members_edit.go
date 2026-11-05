package httpapi

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"snitch/internal/auth"
)

// memberPermissions tells the interface what the acting administrator may do with this account, so the
// rules live in one place. Edit covers the profile (login, name, language); Manage covers everything that
// changes access or credentials, which nobody does to their own account here.
type memberPermissions struct {
	Edit    bool `json:"edit"`
	Manage  bool `json:"manage"`
	SetRole bool `json:"set_role"`
}

func permissionsOn(actorID int64, actorRole string, id int64, targetRole string) memberPermissions {
	self := actorID == id
	var p memberPermissions
	switch {
	case self:
		p.Edit = true
	case isSuperAdministrator(targetRole):
	case targetRole == roleAdministrator:
		p.Edit = isSuperAdministrator(actorRole)
	default:
		p.Edit = true
	}
	p.Manage = p.Edit && !self
	p.SetRole = p.Manage && isSuperAdministrator(actorRole)
	return p
}

type memberDetail struct {
	ID              int64             `json:"id"`
	Username        string            `json:"username"`
	DisplayName     *string           `json:"display_name"`
	Language        *string           `json:"language"`
	Role            string            `json:"role"`
	Approved        bool              `json:"approved"`
	DiscordLinked   bool              `json:"discord_linked"`
	DiscordUsername *string           `json:"discord_username"`
	HasPassword     bool              `json:"has_password"`
	CreatedAt       time.Time         `json:"created_at"`
	ClipCount       int64             `json:"clip_count"`
	SessionCount    int64             `json:"session_count"`
	Permissions     memberPermissions `json:"permissions"`
}

func memberID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_id")
		return 0, false
	}
	return id, true
}

func (s *Server) getMember(w http.ResponseWriter, r *http.Request) {
	id, ok := memberID(w, r)
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	var m memberDetail
	err := s.Pool.QueryRow(r.Context(), `SELECT u.id,u.username,u.display_name,u.language,u.role,u.approved,u.discord_id IS NOT NULL,u.discord_username,u.password_hash IS NOT NULL,u.created_at,
 (SELECT count(*) FROM clips c WHERE c.owner_id=u.id),(SELECT count(*) FROM sessions s WHERE s.user_id=u.id AND s.expires_at>now())
 FROM users u WHERE u.id=$1 AND u.deleted_at IS NULL`, id).Scan(&m.ID, &m.Username, &m.DisplayName, &m.Language, &m.Role, &m.Approved, &m.DiscordLinked, &m.DiscordUsername, &m.HasPassword, &m.CreatedAt, &m.ClipCount, &m.SessionCount)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "member_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	m.Permissions = permissionsOn(ss.User.ID, ss.User.Role, m.ID, m.Role)
	reply(w, 200, m)
}

// optional is a JSON field that can be absent (keep), null (clear) or a value, which a plain pointer cannot tell apart.
type optional[T any] struct {
	Set   bool
	Null  bool
	Value T
}

func (o *optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Null = true
		return nil
	}
	return json.Unmarshal(b, &o.Value)
}

type memberPatch struct {
	Username    optional[string] `json:"username"`
	DisplayName optional[string] `json:"display_name"`
	Language    optional[string] `json:"language"`
	Role        optional[string] `json:"role"`
	Approved    optional[bool]   `json:"approved"`
}

// patchMember edits any field of an account in one transaction. Fields left out are untouched; null clears
// the display name and the language (the account then falls back to its login and to the browser's language).
func (s *Server) patchMember(w http.ResponseWriter, r *http.Request) {
	id, ok := memberID(w, r)
	if !ok {
		return
	}
	var in memberPatch
	if !decode(w, r, &in) {
		return
	}
	if (in.Username.Set && in.Username.Null) || (in.Role.Set && in.Role.Null) || (in.Approved.Set && in.Approved.Null) {
		fail(w, 400, "invalid_profile")
		return
	}
	if !in.Username.Set && !in.DisplayName.Set && !in.Language.Set && !in.Role.Set && !in.Approved.Set {
		fail(w, 400, "invalid_profile")
		return
	}
	var display *string
	if in.Username.Set && !validAccountUsername(in.Username.Value) {
		fail(w, 400, "invalid_profile")
		return
	}
	if in.DisplayName.Set && !in.DisplayName.Null {
		name := strings.TrimSpace(in.DisplayName.Value)
		if n := utf8.RuneCountInString(name); n > 100 || strings.ContainsFunc(name, unicode.IsControl) {
			fail(w, 400, "invalid_profile")
			return
		}
		if name != "" {
			display = &name
		}
	}
	if in.Language.Set && !in.Language.Null && !slices.Contains(supportedLanguages, in.Language.Value) {
		fail(w, 400, "invalid_language")
		return
	}
	if in.Role.Set && in.Role.Value != "user" && in.Role.Value != "admin" {
		fail(w, 400, "invalid_role")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	actorRole, targetRole, status, code, err := lockMembers(r.Context(), tx, ss.User.ID, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	if status != 0 {
		fail(w, status, code)
		return
	}
	if id == ss.User.ID {
		targetRole = actorRole
	}
	perm := permissionsOn(ss.User.ID, actorRole, id, targetRole)
	if !perm.Edit {
		fail(w, 403, "member_protected")
		return
	}
	if (in.Role.Set || in.Approved.Set) && !perm.Manage {
		fail(w, 403, "member_protected")
		return
	}
	if in.Role.Set && !perm.SetRole {
		fail(w, 403, "super_administrator_required")
		return
	}
	// Each column is written only when sent, so two administrators editing different fields do not overwrite each other.
	var sets []string
	args := []any{id}
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, column+"=$"+strconv.Itoa(len(args)))
	}
	if in.Username.Set {
		add("username", in.Username.Value)
	}
	if in.DisplayName.Set {
		add("display_name", display)
	}
	if in.Language.Set {
		var language *string
		if !in.Language.Null {
			language = &in.Language.Value
		}
		add("language", language)
	}
	if in.Role.Set {
		add("role", in.Role.Value)
	}
	if in.Approved.Set {
		add("approved", in.Approved.Value)
	}
	if _, err = tx.Exec(r.Context(), `UPDATE users SET `+strings.Join(sets, ",")+` WHERE id=$1`, args...); err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "23505" {
			fail(w, 409, "username_unavailable")
			return
		}
		s.internal(w, err)
		return
	}
	// A suspended account must not keep a live session.
	if in.Approved.Set && !in.Approved.Value {
		if _, err = tx.Exec(r.Context(), `DELETE FROM sessions WHERE user_id=$1`, id); err != nil {
			s.internal(w, err)
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]string{"status": "ok"})
}

// manageMember runs one credential action on another account, under the same locks and rules as patchMember.
func (s *Server) manageMember(w http.ResponseWriter, r *http.Request, act func(tx pgx.Tx, id int64) (int, string, any, error)) {
	id, ok := memberID(w, r)
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	if id == ss.User.ID {
		fail(w, 403, "member_protected")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	actorRole, targetRole, status, code, err := lockMembers(r.Context(), tx, ss.User.ID, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	if status != 0 {
		fail(w, status, code)
		return
	}
	if !permissionsOn(ss.User.ID, actorRole, id, targetRole).Manage {
		fail(w, 403, "member_protected")
		return
	}
	status, code, body, err := act(tx, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	if status != 0 {
		fail(w, status, code)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, body)
}

const temporaryPasswordAlphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func temporaryPassword() (string, error) {
	out := make([]byte, 20)
	limit := big.NewInt(int64(len(temporaryPasswordAlphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		out[i] = temporaryPasswordAlphabet[n.Int64()]
	}
	return string(out), nil
}

// resetMemberPassword sets a random password, shown once, and ends the account's sessions.
func (s *Server) resetMemberPassword(w http.ResponseWriter, r *http.Request) {
	// The answer carries a secret: no cache may keep it.
	w.Header().Set("Cache-Control", "no-store")
	password, err := temporaryPassword()
	if err != nil {
		s.internal(w, err)
		return
	}
	hash := auth.Password(password)
	s.manageMember(w, r, func(tx pgx.Tx, id int64) (int, string, any, error) {
		if _, err := tx.Exec(r.Context(), `UPDATE users SET password_hash=$2,display_name=COALESCE(display_name,CASE WHEN password_hash IS NULL AND starts_with(username,'discord_') THEN left(discord_name,100) END) WHERE id=$1`, id, hash); err != nil {
			return 0, "", nil, err
		}
		_, err := tx.Exec(r.Context(), `DELETE FROM sessions WHERE user_id=$1`, id)
		return 0, "", map[string]string{"temporary_password": password}, err
	})
}

func (s *Server) revokeMemberSessions(w http.ResponseWriter, r *http.Request) {
	s.manageMember(w, r, func(tx pgx.Tx, id int64) (int, string, any, error) {
		_, err := tx.Exec(r.Context(), `DELETE FROM sessions WHERE user_id=$1`, id)
		return 0, "", map[string]string{"status": "ok"}, err
	})
}

// unlinkMemberDiscord needs a local password on the account, as for the member's own unlink: otherwise it could not sign in again.
func (s *Server) unlinkMemberDiscord(w http.ResponseWriter, r *http.Request) {
	s.manageMember(w, r, func(tx pgx.Tx, id int64) (int, string, any, error) {
		tag, err := tx.Exec(r.Context(), `UPDATE users SET discord_id=NULL,discord_name=NULL,discord_username=NULL WHERE id=$1 AND discord_id IS NOT NULL AND password_hash IS NOT NULL`, id)
		if err != nil {
			return 0, "", nil, err
		}
		if tag.RowsAffected() == 1 {
			return 0, "", map[string]string{"status": "ok"}, dropDiscordAvatar(r.Context(), tx, id)
		}
		var linked, local bool
		if err := tx.QueryRow(r.Context(), `SELECT discord_id IS NOT NULL,password_hash IS NOT NULL FROM users WHERE id=$1`, id).Scan(&linked, &local); err != nil {
			return 0, "", nil, err
		}
		if !linked {
			return 409, "discord_not_linked", nil, nil
		}
		return 409, "discord_local_login_required", nil, nil
	})
}
