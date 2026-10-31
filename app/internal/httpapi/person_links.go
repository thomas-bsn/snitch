package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// registerPersonLinkRoutes declares the account ↔ Person link request routes (issue
// #96). A signed-in user asks to link its account to a Person; an Administrator
// approves or refuses. The link itself stays on persons.user_id, unique in both
// directions (ADR 0011, ADR 0018).
func (s *Server) registerPersonLinkRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/me/person/request", s.protected(false, http.HandlerFunc(s.requestPersonLink)))
	mux.Handle("GET /api/me/person/request", s.protected(false, http.HandlerFunc(s.getMyPersonLinkRequest)))
	mux.Handle("DELETE /api/me/person/request", s.protected(false, http.HandlerFunc(s.cancelMyPersonLinkRequest)))
	mux.Handle("DELETE /api/me/person", s.protected(false, http.HandlerFunc(s.unlinkMyPerson)))
	mux.Handle("GET /api/admin/person-link-requests", s.protected(true, http.HandlerFunc(s.listPersonLinkRequests)))
	mux.Handle("POST /api/admin/person-link-requests/{id}/approve", s.protected(true, http.HandlerFunc(s.approvePersonLinkRequest)))
	mux.Handle("POST /api/admin/person-link-requests/{id}/refuse", s.protected(true, http.HandlerFunc(s.refusePersonLinkRequest)))
}

type accountRef struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type personLinkRequestView struct {
	ID        int64       `json:"id"`
	Person    personRef   `json:"person"`
	Account   accountRef  `json:"account"`
	Status    string      `json:"status"`
	Message   string      `json:"message"`
	CreatedAt time.Time   `json:"created_at"`
	DecidedAt *time.Time  `json:"decided_at"`
	DecidedBy *accountRef `json:"decided_by"`
}

const personLinkRequestColumns = `r.id,r.status,r.message,r.created_at,r.decided_at,
 p.id,p.display_name,u.id,u.username,r.decided_by,r.decided_by_username`
const personLinkRequestFrom = `person_link_requests r JOIN persons p ON p.id=r.person_id JOIN users u ON u.id=r.user_id`

func scanPersonLinkRequest(row pgx.Row) (personLinkRequestView, error) {
	var v personLinkRequestView
	var decidedBy *int64
	var decidedByUsername string
	err := row.Scan(&v.ID, &v.Status, &v.Message, &v.CreatedAt, &v.DecidedAt,
		&v.Person.ID, &v.Person.DisplayName, &v.Account.ID, &v.Account.Username,
		&decidedBy, &decidedByUsername)
	if err != nil {
		return v, err
	}
	if decidedBy != nil {
		v.DecidedBy = &accountRef{ID: *decidedBy, Username: decidedByUsername}
	}
	return v, nil
}

func (s *Server) personLinkRequestView(ctx context.Context, id int64) (personLinkRequestView, error) {
	return scanPersonLinkRequest(s.Pool.QueryRow(ctx, `SELECT `+personLinkRequestColumns+` FROM `+personLinkRequestFrom+` WHERE r.id=$1`, id))
}

// personLinkEvent is one row of the immutable trail. Person, accounts and authors are
// kept by id and by name at the time, without foreign keys; no voice data is stored.
type personLinkEvent struct {
	Action           string
	PersonID         int64
	PersonName       string
	UserID           *int64
	Username         string
	PreviousUserID   *int64
	PreviousUsername string
	ActorID          *int64
	ActorUsername    string
}

func recordPersonLinkEvent(ctx context.Context, tx pgx.Tx, e personLinkEvent) error {
	_, err := tx.Exec(ctx, `INSERT INTO person_link_events(action,person_id,person_name,user_id,username,previous_user_id,previous_username,actor_id,actor_username) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.Action, e.PersonID, e.PersonName, e.UserID, e.Username, e.PreviousUserID, e.PreviousUsername, e.ActorID, e.ActorUsername)
	return err
}

// requestPersonLink lets a signed-in user ask to link its account to a Person. The
// account must not already hold a link, and neither the account nor the Person may
// already have a pending request.
func (s *Server) requestPersonLink(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PersonID int64  `json:"person_id"`
		Message  string `json:"message"`
	}
	if !decode(w, r, &input) {
		return
	}
	message := strings.TrimSpace(input.Message)
	if input.PersonID < 1 || len([]rune(message)) > 500 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var linked bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM persons WHERE user_id=$1)`, ss.User.ID).Scan(&linked); err != nil {
		s.internal(w, err)
		return
	}
	if linked {
		// One account holds one Person: unlink before asking for another.
		fail(w, http.StatusConflict, "account_already_linked")
		return
	}
	var pending bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM person_link_requests WHERE user_id=$1 AND status='pending')`, ss.User.ID).Scan(&pending); err != nil {
		s.internal(w, err)
		return
	}
	if pending {
		fail(w, http.StatusConflict, "request_pending")
		return
	}
	var personName string
	if err = tx.QueryRow(ctx, `SELECT display_name FROM persons WHERE id=$1 FOR SHARE`, input.PersonID).Scan(&personName); errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	} else if err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM person_link_requests WHERE person_id=$1 AND status='pending')`, input.PersonID).Scan(&pending); err != nil {
		s.internal(w, err)
		return
	}
	if pending {
		fail(w, http.StatusConflict, "person_request_pending")
		return
	}
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO person_link_requests(person_id,user_id,message) VALUES($1,$2,$3) RETURNING id`, input.PersonID, ss.User.ID, message).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// A concurrent request on the same Person or from the same account won the race.
			fail(w, http.StatusConflict, "request_pending")
			return
		}
		s.internal(w, err)
		return
	}
	if err = recordPersonLinkEvent(ctx, tx, personLinkEvent{Action: "request", PersonID: input.PersonID, PersonName: personName, UserID: &ss.User.ID, Username: ss.User.Username, ActorID: &ss.User.ID, ActorUsername: ss.User.Username}); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	v, err := s.personLinkRequestView(ctx, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusCreated, v)
}

// getMyPersonLinkRequest returns the caller's most recent request, whatever its state,
// so the account screen can show a pending, approved or refused request.
func (s *Server) getMyPersonLinkRequest(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	v, err := scanPersonLinkRequest(s.Pool.QueryRow(r.Context(), `SELECT `+personLinkRequestColumns+` FROM `+personLinkRequestFrom+` WHERE r.user_id=$1 ORDER BY r.id DESC LIMIT 1`, ss.User.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "no_link_request")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

// cancelMyPersonLinkRequest withdraws the caller's pending request. The row is removed
// (freeing the pending uniqueness) and the fact is kept in the immutable trail.
func (s *Server) cancelMyPersonLinkRequest(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var id, personID int64
	var personName string
	err = tx.QueryRow(ctx, `SELECT r.id,r.person_id,p.display_name FROM person_link_requests r JOIN persons p ON p.id=r.person_id WHERE r.user_id=$1 AND r.status='pending' FOR UPDATE OF r`, ss.User.ID).Scan(&id, &personID, &personName)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "no_pending_request")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM person_link_requests WHERE id=$1`, id); err != nil {
		s.internal(w, err)
		return
	}
	if err = recordPersonLinkEvent(ctx, tx, personLinkEvent{Action: "cancel", PersonID: personID, PersonName: personName, UserID: &ss.User.ID, Username: ss.User.Username, ActorID: &ss.User.ID, ActorUsername: ss.User.Username}); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]string{"status": "ok"})
}

// unlinkMyPerson lets a linked account withdraw its own link. The Person, its
// attributions and its clips are untouched; only the association goes.
func (s *Server) unlinkMyPerson(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var personID int64
	var personName string
	err = tx.QueryRow(ctx, `SELECT id,display_name FROM persons WHERE user_id=$1 FOR UPDATE`, ss.User.ID).Scan(&personID, &personName)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "no_linked_person")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE persons SET user_id=NULL,updated_at=now() WHERE id=$1`, personID); err != nil {
		s.internal(w, err)
		return
	}
	if err = recordPersonLinkEvent(ctx, tx, personLinkEvent{Action: "unlink", PersonID: personID, PersonName: personName, UserID: &ss.User.ID, Username: ss.User.Username, ActorID: &ss.User.ID, ActorUsername: ss.User.Username}); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	s.replyPerson(w, r, personID)
}

func (s *Server) listPersonLinkRequests(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "approved" && status != "refused" {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT `+personLinkRequestColumns+` FROM `+personLinkRequestFrom+` WHERE r.status=$1 ORDER BY r.created_at,r.id LIMIT 500`, status)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []personLinkRequestView{}
	for rows.Next() {
		v, scanErr := scanPersonLinkRequest(rows)
		if scanErr != nil {
			s.internal(w, scanErr)
			return
		}
		items = append(items, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items})
}

// approvePersonLinkRequest links the requesting account to the Person, replacing an
// existing link when there is one (the previous account loses it). Both outcomes are
// the Administrator's explicit decision and are recorded.
func (s *Server) approvePersonLinkRequest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "invalid_id")
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var personID, userID int64
	var status string
	err = tx.QueryRow(ctx, `SELECT person_id,user_id,status FROM person_link_requests WHERE id=$1 FOR UPDATE`, id).Scan(&personID, &userID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "request_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if status != "pending" {
		fail(w, http.StatusConflict, "request_not_pending")
		return
	}
	var personName string
	var current *int64
	if err = tx.QueryRow(ctx, `SELECT display_name,user_id FROM persons WHERE id=$1 FOR UPDATE`, personID).Scan(&personName, &current); err != nil {
		s.internal(w, err)
		return
	}
	var otherLinked bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM persons WHERE user_id=$1 AND id<>$2)`, userID, personID).Scan(&otherLinked); err != nil {
		s.internal(w, err)
		return
	}
	if otherLinked {
		// The account gained another link since it asked: the decision can no longer stand.
		fail(w, http.StatusConflict, "account_already_linked")
		return
	}
	var requesterName string
	if err = tx.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, userID).Scan(&requesterName); err != nil {
		s.internal(w, err)
		return
	}
	var prevID *int64
	var prevName string
	if current != nil && *current != userID {
		prevID = current
		if err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT username FROM users WHERE id=$1),'')`, *current).Scan(&prevName); err != nil {
			s.internal(w, err)
			return
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE persons SET user_id=$2,updated_at=now() WHERE id=$1`, personID, userID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(w, http.StatusConflict, "account_already_linked")
			return
		}
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE person_link_requests SET status='approved',decided_at=now(),decided_by=$2,decided_by_username=$3 WHERE id=$1`, id, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = recordPersonLinkEvent(ctx, tx, personLinkEvent{Action: "approve", PersonID: personID, PersonName: personName, UserID: &userID, Username: requesterName, PreviousUserID: prevID, PreviousUsername: prevName, ActorID: &ss.User.ID, ActorUsername: ss.User.Username}); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	v, err := s.personLinkRequestView(ctx, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

func (s *Server) refusePersonLinkRequest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "invalid_id")
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var personID, userID int64
	var status string
	err = tx.QueryRow(ctx, `SELECT person_id,user_id,status FROM person_link_requests WHERE id=$1 FOR UPDATE`, id).Scan(&personID, &userID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "request_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if status != "pending" {
		fail(w, http.StatusConflict, "request_not_pending")
		return
	}
	var personName, requesterName string
	if err = tx.QueryRow(ctx, `SELECT display_name FROM persons WHERE id=$1`, personID).Scan(&personName); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, userID).Scan(&requesterName); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE person_link_requests SET status='refused',decided_at=now(),decided_by=$2,decided_by_username=$3 WHERE id=$1`, id, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = recordPersonLinkEvent(ctx, tx, personLinkEvent{Action: "refuse", PersonID: personID, PersonName: personName, UserID: &userID, Username: requesterName, ActorID: &ss.User.ID, ActorUsername: ss.User.Username}); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	v, err := s.personLinkRequestView(ctx, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}
