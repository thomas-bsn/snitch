package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"snitch/internal/analysis"
	"snitch/internal/events"
)

// Notifications are persistent and per recipient. A notification is one shared action: its
// resolution is the same for everybody, while reading and deleting are individual. Voices
// that need a human decision are grouped per clip (one active needs_speaker_review per
// clip) and the notification resolves itself when every voice of the clip is handled.

const kindSpeakerReview = "needs_speaker_review"

// notificationVisible is the access rule, re-checked on every read ($1 user, $2 administrator):
// a recipient who lost access to the clip no longer sees its notifications, history included.
// An incident is administrators' business; the owner of the clip concerned keeps a generic view.
const notificationVisible = `(CASE WHEN n.kind='needs_speaker_review' THEN (c.visibility='shared' OR c.owner_id=$1 OR $2) ELSE ($2 OR c.owner_id=$1) END)`

// pendingVoicesSQL counts the voices of clip $1 that still need a decision: unknown or
// ambiguous, neither validated nor automatically attributed (the Verification list).
const pendingVoicesSQL = `SELECT count(*)::int FROM local_speakers ls WHERE ls.clip_id=$1 AND ` + analysis.PendingVoiceCondition

func (s *Server) registerNotificationRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/notifications", s.protected(false, http.HandlerFunc(s.listNotifications)))
	mux.Handle("GET /api/notifications/events", s.protected(false, http.HandlerFunc(s.notificationEvents)))
	mux.Handle("POST /api/notifications/read", s.protected(false, http.HandlerFunc(s.readAllNotifications)))
	mux.Handle("PUT /api/notifications/{id}/read", s.protected(false, http.HandlerFunc(s.markNotificationRead)))
	mux.Handle("DELETE /api/notifications/{id}/read", s.protected(false, http.HandlerFunc(s.markNotificationUnread)))
	mux.Handle("DELETE /api/notifications/{id}", s.protected(false, http.HandlerFunc(s.deleteNotification)))
}

// syncSpeakerReview brings the clip's notification in line with its voices: raised or
// refreshed while some voice awaits a decision, resolved when none does. Idempotent and
// safe to call after any change; failures are logged and never undo the triggering change.
func (s *Server) syncSpeakerReview(ctx context.Context, clipID int64) {
	if err := s.trySyncSpeakerReview(context.WithoutCancel(ctx), clipID); err != nil {
		s.Log.Error("notification sync failed", "clip_id", clipID, "error", err)
	}
}

func (s *Server) trySyncSpeakerReview(ctx context.Context, clipID int64) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Serializes concurrent syncs of the same clip; the count is read under the lock.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('notification:'||($1::bigint)::text,0))`, clipID); err != nil {
		return err
	}
	var n int
	if err = tx.QueryRow(ctx, pendingVoicesSQL, clipID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		if _, err = tx.Exec(ctx, `UPDATE notifications SET state='resolved',resolved_at=now(),updated_at=now(),pending_count=0 WHERE kind=$1 AND clip_id=$2 AND state='active'`, kindSpeakerReview, clipID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	var id int64
	var previous int
	err = tx.QueryRow(ctx, `SELECT id,pending_count FROM notifications WHERE kind=$1 AND clip_id=$2 AND state='active' FOR UPDATE`, kindSpeakerReview, clipID).Scan(&id, &previous)
	exists := err == nil
	switch {
	case err == nil:
		if previous != n {
			if _, err = tx.Exec(ctx, `UPDATE notifications SET pending_count=$2,updated_at=now() WHERE id=$1`, id, n); err != nil {
				return err
			}
		}
	case err == pgx.ErrNoRows:
		if err = tx.QueryRow(ctx, `INSERT INTO notifications(kind,clip_id,pending_count) VALUES($1,$2,$3) RETURNING id`, kindSpeakerReview, clipID, n).Scan(&id); err != nil {
			return err
		}
	default:
		return err
	}
	// Recipients: administrators and the clip's owner (added later too, never removed).
	if _, err = tx.Exec(ctx, `INSERT INTO notification_recipients(notification_id,user_id)
SELECT $1,u.id FROM users u WHERE u.approved AND u.deleted_at IS NULL AND (u.role IN ('admin','super_admin') OR u.id=(SELECT owner_id FROM clips WHERE id=$2))
ON CONFLICT DO NOTHING`, id, clipID); err != nil {
		return err
	}
	// More voices to decide than before: the news is unread again for everybody.
	if exists && n > previous {
		if _, err = tx.Exec(ctx, `UPDATE notification_recipients SET read_at=NULL WHERE notification_id=$1`, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// syncAllSpeakerReviews reconciles every clip that has a diarized voice or an active
// notification (after a library-wide rematch).
func (s *Server) syncAllSpeakerReviews(ctx context.Context) {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT clip_id FROM local_speakers WHERE kind='diarized'
UNION SELECT clip_id FROM notifications WHERE kind=$1 AND state='active' AND clip_id IS NOT NULL`, kindSpeakerReview)
	if err != nil {
		s.Log.Error("notification sync failed", "error", err)
		return
	}
	var clips []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			clips = append(clips, id)
		}
	}
	rows.Close()
	for _, id := range clips {
		s.syncSpeakerReview(ctx, id)
	}
}

type notificationView struct {
	ID           int64      `json:"id"`
	Kind         string     `json:"kind"`
	State        string     `json:"state"`
	ClipID       *int64     `json:"clip_id"`
	ClipName     string     `json:"clip_name"`
	PendingCount int        `json:"pending_count"`
	Read         bool       `json:"read"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ResolvedAt   *time.Time `json:"resolved_at"`
	// Incident payload: codes only. Empty for a non-administrator, who gets Generic instead.
	JobID     *int64 `json:"job_id"`
	Step      string `json:"step"`
	ErrorCode string `json:"error_code"`
	Generic   bool   `json:"generic"`
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	admin := isAdministrator(ss.User.Role)
	state := r.URL.Query().Get("state")
	switch state {
	case "", "all":
		state = "all"
	case "active", "resolved":
	default:
		fail(w, http.StatusBadRequest, "invalid_filter")
		return
	}
	filter := r.URL.Query().Get("filter")
	switch filter {
	case "", "all":
		filter = "all"
	case "unread", "actions", "errors":
	default:
		fail(w, http.StatusBadRequest, "invalid_filter")
		return
	}
	limit, offset := 50, 0
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
		limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
		offset = n
	}
	list, err := s.notificationList(r.Context(), ss.User.ID, admin, notificationPage{State: state, Filter: filter, Limit: limit, Offset: offset})
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, list)
}

// notificationPage selects a page of the list: State and Filter narrow it on the server (the
// client never filters a capped copy), Offset pages through the rest.
type notificationPage struct {
	State, Filter string
	Limit, Offset int
}

// notificationFilterSQL is the condition of a page filter (alias n for the notification, x for
// its recipient row); the filter name is validated before it reaches here.
func notificationFilterSQL(filter string) string {
	switch filter {
	case "unread":
		return `x.read_at IS NULL`
	case "actions":
		return `n.state='active'`
	case "errors":
		return `n.kind<>'needs_speaker_review'`
	}
	return `true`
}

type notificationListView struct {
	Items   []notificationView `json:"items"`
	Unread  int                `json:"unread"`
	Pending int                `json:"pending"`
	// Total counts everything the state and filter select, whatever the page.
	Total int `json:"total"`
}

// notificationList reads the notifications of a recipient, the access rule applied at this
// very read. The two counters cover everything visible, whatever the page size or filter.
func (s *Server) notificationList(ctx context.Context, userID int64, admin bool, page notificationPage) (notificationListView, error) {
	where := `x.user_id=$1 AND x.deleted_at IS NULL AND ` + notificationVisible + ` AND ($3='all' OR n.state=$3) AND ` + notificationFilterSQL(page.Filter)
	out := notificationListView{Items: []notificationView{}}
	rows, err := s.Pool.Query(ctx, `SELECT n.id,n.kind,n.state,n.clip_id,COALESCE(c.display_title,c.original_name,''),n.pending_count,x.read_at IS NOT NULL,n.created_at,n.updated_at,n.resolved_at,n.job_id,n.step,n.error_code
FROM notification_recipients x JOIN notifications n ON n.id=x.notification_id LEFT JOIN clips c ON c.id=n.clip_id
WHERE `+where+`
ORDER BY (n.state='active') DESC,n.updated_at DESC,n.id DESC LIMIT $4 OFFSET $5`, userID, admin, page.State, page.Limit, page.Offset)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v notificationView
		if err = rows.Scan(&v.ID, &v.Kind, &v.State, &v.ClipID, &v.ClipName, &v.PendingCount, &v.Read, &v.CreatedAt, &v.UpdatedAt, &v.ResolvedAt, &v.JobID, &v.Step, &v.ErrorCode); err != nil {
			return out, err
		}
		if v.Kind != kindSpeakerReview && !admin {
			// A safe generic message: no cause, step, job or counts.
			v.Kind, v.Generic, v.JobID, v.Step, v.ErrorCode, v.PendingCount = kindTechnicalIncident, true, nil, "", "", 0
		}
		out.Items = append(out.Items, v)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, err
	}
	err = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE x.read_at IS NULL)::int,count(*) FILTER (WHERE n.state='active')::int,count(*) FILTER (WHERE ($3='all' OR n.state=$3) AND `+notificationFilterSQL(page.Filter)+`)::int
FROM notification_recipients x JOIN notifications n ON n.id=x.notification_id LEFT JOIN clips c ON c.id=n.clip_id
WHERE x.user_id=$1 AND x.deleted_at IS NULL AND `+notificationVisible, userID, admin, page.State).Scan(&out.Unread, &out.Pending, &out.Total)
	return out, err
}

// notificationEvents streams the caller's notifications: a snapshot, then one per change. The
// session (and so the role) is read again before every snapshot, so a user who lost access
// stops receiving what they could see, and a closed session ends the stream.
func (s *Server) notificationEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(sessionKey{}).(session).User.ID
	current := func(ctx context.Context) (session, error) {
		ss, err := s.findSession(r.WithContext(ctx))
		switch {
		case errors.Is(err, pgx.ErrNoRows), errors.Is(err, http.ErrNoCookie):
			return ss, events.ErrForbidden
		case err != nil:
			return ss, err
		case ss.User.ID != userID:
			return ss, events.ErrForbidden
		}
		return ss, nil
	}
	s.Events.Stream(w, r, events.Topic{
		Name: "user:" + strconv.FormatInt(userID, 10),
		Authorize: func(ctx context.Context) error {
			_, err := current(ctx)
			return err
		},
		Snapshot: func(ctx context.Context) (any, error) {
			ss, err := current(ctx)
			if err != nil {
				return nil, err
			}
			return s.notificationList(ctx, userID, isAdministrator(ss.User.Role), notificationPage{State: "all", Filter: "all", Limit: 200})
		},
	})
}

// ownNotification reports the state of a notification the caller may act on.
func (s *Server) ownNotification(ctx context.Context, ss session, id int64) (state string, ok bool, err error) {
	err = s.Pool.QueryRow(ctx, `SELECT n.state FROM notification_recipients x JOIN notifications n ON n.id=x.notification_id LEFT JOIN clips c ON c.id=n.clip_id
WHERE x.user_id=$1 AND n.id=$3 AND x.deleted_at IS NULL AND `+notificationVisible, ss.User.ID, isAdministrator(ss.User.Role), id).Scan(&state)
	if err == pgx.ErrNoRows {
		return "", false, nil
	}
	return state, err == nil, err
}

func notificationID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusNotFound, "notification_not_found")
		return 0, false
	}
	return id, true
}

func (s *Server) setNotificationRead(w http.ResponseWriter, r *http.Request, read bool) {
	ss := r.Context().Value(sessionKey{}).(session)
	id, ok := notificationID(w, r)
	if !ok {
		return
	}
	_, found, err := s.ownNotification(r.Context(), ss, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !found {
		fail(w, http.StatusNotFound, "notification_not_found")
		return
	}
	if _, err = s.Pool.Exec(r.Context(), `UPDATE notification_recipients SET read_at=CASE WHEN $3 THEN COALESCE(read_at,now()) ELSE NULL END WHERE notification_id=$1 AND user_id=$2`, id, ss.User.ID, read); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"id": id, "read": read})
}

func (s *Server) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	s.setNotificationRead(w, r, true)
}

func (s *Server) markNotificationUnread(w http.ResponseWriter, r *http.Request) {
	s.setNotificationRead(w, r, false)
}

func (s *Server) readAllNotifications(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	tag, err := s.Pool.Exec(r.Context(), `UPDATE notification_recipients x SET read_at=now() FROM notifications n LEFT JOIN clips c ON c.id=n.clip_id
WHERE n.id=x.notification_id AND x.user_id=$1 AND x.read_at IS NULL AND x.deleted_at IS NULL AND `+notificationVisible, ss.User.ID, isAdministrator(ss.User.Role))
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"updated": tag.RowsAffected()})
}

// deleteNotification removes a resolved notification from the caller's history only. An
// active action cannot be deleted: it disappears only by being handled.
func (s *Server) deleteNotification(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	id, ok := notificationID(w, r)
	if !ok {
		return
	}
	state, found, err := s.ownNotification(r.Context(), ss, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !found {
		fail(w, http.StatusNotFound, "notification_not_found")
		return
	}
	if state != "resolved" {
		fail(w, http.StatusConflict, "notification_active")
		return
	}
	if _, err = s.Pool.Exec(r.Context(), `UPDATE notification_recipients SET deleted_at=now() WHERE notification_id=$1 AND user_id=$2`, id, ss.User.ID); err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
