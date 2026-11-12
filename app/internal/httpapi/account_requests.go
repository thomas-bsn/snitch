package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// A Demande d'accès (#283) is announced to the administrators by one persistent notification, raised in the
// transaction that creates the Inscription en attente. It resolves itself (database trigger) when the account is
// approved by any path or retired, which is how a request is refused.

const kindAccountRequest = "account_request"

// raiseAccountRequest notifies every approved administrator of the pending account. Idempotent: an account has
// one request notification at most, so a retried or repeated sign-up never raises a second one.
func raiseAccountRequest(ctx context.Context, tx pgx.Tx, accountID int64) error {
	_, err := tx.Exec(ctx, `WITH n AS (
 INSERT INTO notifications(kind,account_id,account_name)
 SELECT $1,id,username FROM users WHERE id=$2 AND NOT approved AND deleted_at IS NULL
 ON CONFLICT (account_id) WHERE kind='account_request' DO NOTHING RETURNING id)
INSERT INTO notification_recipients(notification_id,user_id)
SELECT n.id,u.id FROM n,users u WHERE u.approved AND u.deleted_at IS NULL AND u.role IN ('admin','super_admin')`, kindAccountRequest, accountID)
	return err
}

// refuse retires a pending member account: the request is refused, the name freed and the trail anonymized.
// An approved account, or an administrator's, is not a request: 404 not_pending.
func (s *Server) refuse(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_id")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err = tx.QueryRow(r.Context(), `SELECT id FROM users WHERE id=$1 AND NOT approved AND deleted_at IS NULL AND role='user' FOR UPDATE`, id).Scan(&id); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "not_pending")
		return
	} else if err != nil {
		s.internal(w, err)
		return
	}
	if err = retireMember(r.Context(), tx, id, false); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]string{"status": "ok"})
}
