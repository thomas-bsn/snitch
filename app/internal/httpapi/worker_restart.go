package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/workers"
)

func (s *Server) requestWorkerRestart(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IdempotencyKey string `json:"idempotency_key"`
		Reason         string `json:"reason"`
	}
	if !decode(w, r, &input) {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	command, err := workers.RequestRestart(r.Context(), s.Pool, r.PathValue("id"), fmt.Sprint(ss.User.ID), input.IdempotencyKey, input.Reason, time.Now().UTC())
	switch {
	case errors.Is(err, workers.ErrNotFound):
		fail(w, http.StatusNotFound, "worker_not_found")
	case errors.Is(err, workers.ErrSupervisorUnavailable):
		fail(w, http.StatusConflict, "managed_supervisor_unavailable")
	case errors.Is(err, workers.ErrRestartPending):
		fail(w, http.StatusConflict, "restart_already_pending")
	case errors.Is(err, workers.ErrInvalidInput):
		fail(w, http.StatusBadRequest, "invalid_restart_request")
	case err != nil:
		s.internal(w, err)
	default:
		reply(w, http.StatusAccepted, command)
	}
}

func (s *Server) workerRestartHistory(w http.ResponseWriter, r *http.Request) {
	var enrolled bool
	var supervisorSeen *time.Time
	if err := s.Pool.QueryRow(r.Context(), `SELECT last_seen_at FROM worker_supervisors WHERE worker_id=$1 AND revoked_at IS NULL`, r.PathValue("id")).Scan(&supervisorSeen); err == nil {
		enrolled = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT c.id,c.worker_id,c.actor_id,c.reason,c.status,c.created_at,c.updated_at,c.result,COALESCE(u.display_name,u.username,'Administrateur') FROM worker_restart_commands c LEFT JOIN users u ON u.id::text=c.actor_id WHERE c.worker_id=$1 ORDER BY c.created_at DESC LIMIT 20`, r.PathValue("id"))
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, workerID, actorID, reason, status, result, actor string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &workerID, &actorID, &reason, &status, &createdAt, &updatedAt, &result, &actor); err != nil {
			s.internal(w, err)
			return
		}
		items = append(items, map[string]any{"id": id, "worker_id": workerID, "actor_id": actorID, "actor": actor, "reason": reason, "status": status, "created_at": createdAt, "updated_at": updatedAt, "result": result})
	}
	if err := rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items, "supervisor_enrolled": enrolled, "supervisor_last_seen_at": supervisorSeen, "supervisor_available": enrolled && supervisorSeen != nil && time.Since(*supervisorSeen) <= 30*time.Second})
}

func supervisorToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}

func (s *Server) pollWorkerRestart(w http.ResponseWriter, r *http.Request) {
	workerID, err := workers.AuthenticateSupervisor(r.Context(), s.Pool, supervisorToken(r), time.Now().UTC())
	if errors.Is(err, workers.ErrUnauthorized) {
		fail(w, http.StatusUnauthorized, "supervisor_token_invalid")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	command, err := workers.RestartTransitionsForSupervisor(r.Context(), s.Pool, workerID, time.Now().UTC())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"command": command})
}

func (s *Server) updateWorkerRestart(w http.ResponseWriter, r *http.Request) {
	workerID, err := workers.AuthenticateSupervisor(r.Context(), s.Pool, supervisorToken(r), time.Now().UTC())
	if errors.Is(err, workers.ErrUnauthorized) {
		fail(w, http.StatusUnauthorized, "supervisor_token_invalid")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	var input struct {
		Status string `json:"status"`
		Result string `json:"result"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Result != "" && (len(input.Result) > 200 || strings.ContainsAny(input.Result, "\r\n")) {
		fail(w, http.StatusBadRequest, "invalid_restart_result")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var current string
	err = tx.QueryRow(r.Context(), `SELECT status FROM worker_restart_commands WHERE id=$1 AND worker_id=$2 FOR UPDATE`, r.PathValue("id"), workerID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "restart_command_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if !workers.RestartTransitionAllowed(current, input.Status) {
		fail(w, http.StatusConflict, "invalid_restart_transition")
		return
	}
	if input.Status == "online" {
		var confirmed bool
		err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workers w JOIN worker_restart_commands c ON c.worker_id=w.id WHERE c.id=$1 AND w.last_seen_at>c.created_at)`, r.PathValue("id")).Scan(&confirmed)
		if err != nil {
			s.internal(w, err)
			return
		}
		if !confirmed {
			fail(w, http.StatusConflict, "worker_pulse_not_confirmed")
			return
		}
	}
	if (current == "requested" || current == "accepted") && input.Status != "expired" {
		var expires time.Time
		if err = tx.QueryRow(r.Context(), `SELECT expires_at FROM worker_restart_commands WHERE id=$1`, r.PathValue("id")).Scan(&expires); err != nil {
			s.internal(w, err)
			return
		}
		if !time.Now().Before(expires) {
			fail(w, http.StatusConflict, "restart_command_expired")
			return
		}
	}
	_, err = tx.Exec(r.Context(), `UPDATE worker_restart_commands SET status=$2,result=$3,updated_at=now() WHERE id=$1`, r.PathValue("id"), input.Status, input.Result)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO worker_restart_events(command_id,actor,status,occurred_at,result) VALUES($1,'supervisor',$2,now(),$3)`, r.PathValue("id"), input.Status, input.Result)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
