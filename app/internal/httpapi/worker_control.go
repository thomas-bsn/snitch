package httpapi

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// Live logs and pause of a Worker (ADR 0063).
//
// A Worker only calls Snitch. Its log lines are pushed in batches and kept in a bounded window per Worker; the
// pause is a state the Worker reads on its next poll, never a queue of commands, so it survives a Worker that was
// offline and is never applied twice. Restarting the container is the managed restart (worker_restart.go).

const (
	maxLogBatch        = 200
	maxLogLineLength   = 4096
	workerLogRetention = 5000 // lines kept per Worker, the newest ones
	workerLogPageLimit = 1000
	// maxWorkerLogBody bounds one batch on the wire: 200 lines of 4096 bytes, JSON-escaped, with room to spare.
	// The Worker cuts its batches well under it (log.UPLINK_BATCH_BYTES).
	maxWorkerLogBody = 2 << 20
)

var logSessionID = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

func (s *Server) registerWorkerControlRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/worker/logs", s.workerOnly(http.HandlerFunc(s.postWorkerLogs)))
	mux.Handle("GET /api/worker/control", s.workerOnly(http.HandlerFunc(s.getWorkerControlForWorker)))
	mux.Handle("POST /api/worker/control/ack", s.workerOnly(http.HandlerFunc(s.ackWorkerControl)))
	mux.Handle("GET /api/admin/workers/{id}/logs", s.protected(true, http.HandlerFunc(s.listWorkerLogs)))
	mux.Handle("GET /api/admin/workers/{id}/control", s.protected(true, http.HandlerFunc(s.getWorkerControl)))
	mux.Handle("POST /api/admin/workers/{id}/control", s.protected(true, http.HandlerFunc(s.postWorkerControl)))
}

func (s *Server) postWorkerLogs(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SessionID string `json:"session_id"`
		Lines     []struct {
			Seq  int64  `json:"seq"`
			Line string `json:"line"`
		} `json:"lines"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !logSessionID.MatchString(in.SessionID) || len(in.Lines) == 0 || len(in.Lines) > maxLogBatch {
		fail(w, http.StatusBadRequest, "invalid_worker_logs")
		return
	}
	seqs := make([]int64, len(in.Lines))
	lines := make([]string, len(in.Lines))
	for i, l := range in.Lines {
		if l.Seq < 0 || len(l.Line) > maxLogLineLength {
			fail(w, http.StatusBadRequest, "invalid_worker_logs")
			return
		}
		seqs[i], lines[i] = l.Seq, l.Line
	}
	worker := currentWorker(r)
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	// A line already stored under the same session and number is left as it is: a batch resent after a lost answer is harmless.
	if _, err = tx.Exec(r.Context(), `INSERT INTO worker_log_lines(worker_id,session_id,seq,line)
		SELECT $1,$2,t.seq,t.line FROM unnest($3::bigint[],$4::text[]) AS t(seq,line) ON CONFLICT DO NOTHING`, worker.ID, in.SessionID, seqs, lines); err != nil {
		s.internal(w, err)
		return
	}
	// The window is bounded by count: a chatty Worker never grows the table without limit.
	if _, err = tx.Exec(r.Context(), `DELETE FROM worker_log_lines WHERE worker_id=$1 AND id <= (SELECT id FROM worker_log_lines WHERE worker_id=$1 ORDER BY id DESC OFFSET $2 LIMIT 1)`, worker.ID, workerLogRetention); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type workerLogLine struct {
	ID   int64     `json:"id"`
	Seq  int64     `json:"seq"`
	At   time.Time `json:"at"`
	Line string    `json:"line"`
}

type workerLogPage struct {
	SessionID string          `json:"session_id"`
	Lines     []workerLogLine `json:"lines"`
	// HasMore tells that older lines of the session exist beyond the first one returned.
	HasMore bool `json:"has_more"`
}

// listWorkerLogs returns the lines of the Worker's current session (its latest process). Without a cursor it is
// the newest page; "after" follows the live end, "before" walks back to the start of the session.
func (s *Server) listWorkerLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.workerExists(w, r, id) {
		return
	}
	limit := workerLogPageLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > workerLogPageLimit {
			fail(w, http.StatusBadRequest, "invalid_request")
			return
		}
		limit = n
	}
	after, before := int64(0), int64(0)
	for name, target := range map[string]*int64{"after": &after, "before": &before} {
		if v := r.URL.Query().Get(name); v != "" {
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n < 0 {
				fail(w, http.StatusBadRequest, "invalid_request")
				return
			}
			*target = n
		}
	}
	if after > 0 && before > 0 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	page := workerLogPage{Lines: []workerLogLine{}}
	err := s.Pool.QueryRow(r.Context(), `SELECT session_id FROM worker_log_lines WHERE worker_id=$1 ORDER BY id DESC LIMIT 1`, id).Scan(&page.SessionID)
	if err == pgx.ErrNoRows {
		reply(w, http.StatusOK, page)
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	var rows pgx.Rows
	switch {
	case after > 0:
		rows, err = s.Pool.Query(r.Context(), `SELECT id,seq,logged_at,line FROM worker_log_lines WHERE worker_id=$1 AND session_id=$2 AND id>$3 ORDER BY id LIMIT $4`, id, page.SessionID, after, limit)
	default:
		// Newest first, then put back in reading order; one more row tells that the session goes further back.
		rows, err = s.Pool.Query(r.Context(), `SELECT id,seq,logged_at,line FROM worker_log_lines WHERE worker_id=$1 AND session_id=$2 AND ($3=0 OR id<$3) ORDER BY id DESC LIMIT $4`, id, page.SessionID, before, limit+1)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var l workerLogLine
		if err = rows.Scan(&l.ID, &l.Seq, &l.At, &l.Line); err != nil {
			s.internal(w, err)
			return
		}
		page.Lines = append(page.Lines, l)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	if after == 0 {
		if len(page.Lines) > limit {
			page.HasMore = true
			page.Lines = page.Lines[:limit]
		}
		for i, j := 0, len(page.Lines)-1; i < j; i, j = i+1, j-1 {
			page.Lines[i], page.Lines[j] = page.Lines[j], page.Lines[i]
		}
	}
	reply(w, http.StatusOK, page)
}

// workerControlState is what the interface shows: the wish of the administrator and what the Worker applied.
type workerControlState struct {
	Paused bool `json:"paused"`
	// AppliedPaused is null until the Worker confirms; it differs from Paused while the instruction is on its way.
	AppliedPaused *bool `json:"applied_paused"`
}

const controlColumns = `COALESCE(c.paused,false),c.applied_paused`

func (s *Server) readWorkerControl(r *http.Request, id string) (workerControlState, error) {
	var st workerControlState
	err := s.Pool.QueryRow(r.Context(), `SELECT `+controlColumns+` FROM workers w LEFT JOIN worker_control c ON c.worker_id=w.id WHERE w.id=$1`, id).
		Scan(&st.Paused, &st.AppliedPaused)
	return st, err
}

func (s *Server) workerExists(w http.ResponseWriter, r *http.Request, id string) bool {
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND revoked_at IS NULL)`, id).Scan(&exists); err != nil {
		s.internal(w, err)
		return false
	}
	if !exists {
		fail(w, http.StatusNotFound, "worker_not_found")
	}
	return exists
}

func (s *Server) getWorkerControl(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.workerExists(w, r, id) {
		return
	}
	st, err := s.readWorkerControl(r, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, st)
}

func (s *Server) postWorkerControl(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Action string `json:"action"`
	}
	if !decode(w, r, &in) {
		return
	}
	var query string
	switch in.Action {
	case "pause":
		query = `INSERT INTO worker_control(worker_id,paused,requested_by) VALUES($1,true,$2) ON CONFLICT(worker_id) DO UPDATE SET paused=true,requested_by=$2,updated_at=now()`
	case "resume":
		query = `INSERT INTO worker_control(worker_id,paused,requested_by) VALUES($1,false,$2) ON CONFLICT(worker_id) DO UPDATE SET paused=false,requested_by=$2,updated_at=now()`
	default:
		fail(w, http.StatusBadRequest, "invalid_worker_control")
		return
	}
	if !s.workerExists(w, r, id) {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	if _, err := s.Pool.Exec(r.Context(), query, id, ss.User.ID); err != nil {
		s.internal(w, err)
		return
	}
	// An instruction to someone's machine is traced: who, which Worker, what.
	s.Log.Info("worker control requested", "worker", id, "action", in.Action, "user", ss.User.ID)
	st, err := s.readWorkerControl(r, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, st)
}

// getWorkerControlForWorker is what the Worker polls: the state to apply.
func (s *Server) getWorkerControlForWorker(w http.ResponseWriter, r *http.Request) {
	worker := currentWorker(r)
	out := struct {
		Paused bool `json:"paused"`
	}{}
	err := s.Pool.QueryRow(r.Context(), `SELECT paused FROM worker_control WHERE worker_id=$1`, worker.ID).Scan(&out.Paused)
	if err != nil && err != pgx.ErrNoRows {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}

func (s *Server) ackWorkerControl(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Paused bool `json:"paused"`
	}
	if !decode(w, r, &in) {
		return
	}
	worker := currentWorker(r)
	if _, err := s.Pool.Exec(r.Context(), `INSERT INTO worker_control(worker_id,applied_paused) VALUES($1,$2)
		ON CONFLICT(worker_id) DO UPDATE SET applied_paused=$2`, worker.ID, in.Paused); err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
