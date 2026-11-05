package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type memberView struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"display_name"`
	DiscordLinked bool      `json:"discord_linked"`
	Role          string    `json:"role"`
	CreatedAt     time.Time `json:"created_at"`
	ClipCount     int64     `json:"clip_count"`
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT u.id,u.username,COALESCE(u.display_name,CASE WHEN u.password_hash IS NULL AND starts_with(u.username,'discord_') THEN u.discord_name END,u.username),u.discord_id IS NOT NULL,u.role,u.created_at,count(c.id) FROM users u LEFT JOIN clips c ON c.owner_id=u.id WHERE u.approved AND u.deleted_at IS NULL GROUP BY u.id ORDER BY lower(u.username),u.id`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	members := []memberView{}
	for rows.Next() {
		var v memberView
		if err = rows.Scan(&v.ID, &v.Username, &v.DisplayName, &v.DiscordLinked, &v.Role, &v.CreatedAt, &v.ClipCount); err != nil {
			s.internal(w, err)
			return
		}
		members = append(members, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": members})
}
func (s *Server) changeMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_id")
		return
	}
	var role string
	var hide bool
	if r.Method == http.MethodPut {
		var body struct {
			Role string `json:"role"`
		}
		if !decode(w, r, &body) {
			return
		}
		role = body.Role
		if role != "user" && role != "admin" {
			fail(w, 400, "invalid_role")
			return
		}
	} else {
		var body struct {
			HideClips bool `json:"hide_clips"`
		}
		if !decode(w, r, &body) {
			return
		}
		hide = body.HideClips
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
	if isSuperAdministrator(targetRole) {
		fail(w, 403, "member_protected")
		return
	}
	if !isSuperAdministrator(actorRole) && (targetRole == "admin" || role == "admin") {
		fail(w, 403, "super_administrator_required")
		return
	}
	if r.Method == http.MethodPut {
		_, err = tx.Exec(r.Context(), `UPDATE users SET role=$2 WHERE id=$1`, id, role)
	} else {
		err = retireMember(r.Context(), tx, id, hide)

	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]string{"status": "ok"})
}

// retireMember keeps ownership and correction history while invalidating credentials and sessions.
func retireMember(ctx context.Context, tx pgx.Tx, id int64, hide bool) error {
	if hide {
		if _, err := tx.Exec(ctx, `UPDATE clips SET visibility='private' WHERE owner_id=$1`, id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET username='Utilisateur supprimé',password_hash=NULL,discord_id=NULL,discord_name=NULL,approved=false,role='user',deleted_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	if err := dropDiscordAvatar(ctx, tx, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, id); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM oauth_states WHERE user_id=$1`, id)
	return err
}

func (s *Server) closeAccount(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var role string
	var approved bool
	var deleted bool
	err = tx.QueryRow(r.Context(), `SELECT role,approved,deleted_at IS NOT NULL FROM users WHERE id=$1 FOR UPDATE`, ss.User.ID).Scan(&role, &approved, &deleted)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !approved || deleted {
		fail(w, 401, "unauthorized")
		return
	}
	if isSuperAdministrator(role) {
		fail(w, 403, "member_protected")
		return
	}
	if err = retireMember(r.Context(), tx, ss.User.ID, true); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	s.cookie(w, r, "snitch_session", "", -1)
	reply(w, http.StatusOK, map[string]string{"status": "ok"})
}

// lockMembers locks the acting administrator and the target in a deterministic order and rechecks the actor
// after locking. A non-zero status is the refusal to send. When the target is the actor, both roles are the actor's.
func lockMembers(ctx context.Context, tx pgx.Tx, actorID, id int64) (actorRole, targetRole string, status int, code string, err error) {
	rows, err := tx.Query(ctx, `SELECT id,role,approved,deleted_at IS NOT NULL FROM users WHERE id IN ($1,$2) ORDER BY id FOR UPDATE`, actorID, id)
	if err != nil {
		return "", "", 0, "", err
	}
	var actorActive, targetActive bool
	for rows.Next() {
		var uid int64
		var current string
		var approved, deleted bool
		if err = rows.Scan(&uid, &current, &approved, &deleted); err != nil {
			break
		}
		if uid == actorID {
			actorRole, actorActive = current, approved && !deleted
		}
		if uid == id {
			targetRole, targetActive = current, approved && !deleted
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return "", "", 0, "", err
	}
	if !actorActive || !isAdministrator(actorRole) {
		return "", "", 403, "forbidden", nil
	}
	if !targetActive {
		return "", "", 404, "member_not_found", nil
	}
	return actorRole, targetRole, 0, "", nil
}
