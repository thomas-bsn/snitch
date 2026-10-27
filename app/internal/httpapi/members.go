package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type memberView struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ClipCount int64     `json:"clip_count"`
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT u.id,u.username,u.role,u.created_at,count(c.id) FROM users u LEFT JOIN clips c ON c.owner_id=u.id WHERE u.approved AND u.deleted_at IS NULL GROUP BY u.id ORDER BY lower(u.username),u.id`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	members := []memberView{}
	for rows.Next() {
		var v memberView
		if err = rows.Scan(&v.ID, &v.Username, &v.Role, &v.CreatedAt, &v.ClipCount); err != nil {
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
	// Lock both accounts in a deterministic order and recheck the actor after locking.
	rows, err := tx.Query(r.Context(), `SELECT id,role,approved,deleted_at IS NOT NULL FROM users WHERE id IN ($1,$2) ORDER BY id FOR UPDATE`, ss.User.ID, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	var actorRole, targetRole string
	var actorActive, targetActive bool
	for rows.Next() {
		var uid int64
		var current string
		var approved, deleted bool
		if err = rows.Scan(&uid, &current, &approved, &deleted); err != nil {
			break
		}
		if uid == ss.User.ID {
			actorRole = current
			actorActive = approved && !deleted
		} else {
			targetRole = current
			targetActive = approved && !deleted
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		s.internal(w, err)
		return
	}
	if !actorActive || !isAdministrator(actorRole) {
		fail(w, 403, "forbidden")
		return
	}
	if !targetActive {
		fail(w, 404, "member_not_found")
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
		if hide {
			_, err = tx.Exec(r.Context(), `UPDATE clips SET visibility='private' WHERE owner_id=$1`, id)
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `UPDATE users SET username='Utilisateur supprimé',password_hash=NULL,discord_id=NULL,approved=false,role='user',deleted_at=now() WHERE id=$1`, id)
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `DELETE FROM sessions WHERE user_id=$1`, id)
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `DELETE FROM oauth_states WHERE user_id=$1`, id)
		}
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
