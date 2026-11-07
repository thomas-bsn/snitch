package workers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/auth"
)

var ErrInvalidRestartTransition = errors.New("invalid worker restart transition")

// RestartTransitionAllowed keeps the command lifecycle monotonic. Online is a
// success state only after the supervisor reports a fresh worker pulse.
func RestartTransitionAllowed(from, to string) bool {
	switch from {
	case "requested":
		return to == "accepted" || to == "expired" || to == "failed"
	case "accepted":
		return to == "stopping" || to == "expired" || to == "failed"
	case "stopping":
		return to == "starting" || to == "failed" || to == "expired"
	case "starting":
		return to == "online" || to == "failed" || to == "expired"
	default:
		return false
	}
}

func RestartExpired(now, expires time.Time) bool { return !now.Before(expires) }

type RestartCommand struct {
	ID             string    `json:"id"`
	WorkerID       string    `json:"worker_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	ActorID        string    `json:"actor_id"`
	Reason         string    `json:"reason"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Result         string    `json:"result"`
}

var (
	ErrSupervisorUnavailable = errors.New("managed restart supervisor unavailable")
	ErrRestartPending        = errors.New("worker restart already pending")
)

// RequestRestart creates an auditable, idempotent command only when the opted-in
// supervisor has checked in recently. Worker presence is deliberately irrelevant.
func RequestRestart(ctx context.Context, pool *pgxpool.Pool, workerID, actorID, key, reason string, now time.Time) (RestartCommand, error) {
	key, reason = strings.TrimSpace(key), strings.TrimSpace(reason)
	if len(key) < 1 || len(key) > 128 || len(reason) < 1 || len(reason) > 500 {
		return RestartCommand{}, ErrInvalidInput
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return RestartCommand{}, fmt.Errorf("begin restart request: %w", err)
	}
	defer tx.Rollback(ctx)
	var revoked bool
	err = tx.QueryRow(ctx, `SELECT revoked_at IS NOT NULL FROM workers WHERE id=$1 FOR UPDATE`, workerID).Scan(&revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return RestartCommand{}, ErrNotFound
	}
	if err != nil {
		return RestartCommand{}, fmt.Errorf("read restart supervisor: %w", err)
	}
	var command RestartCommand
	err = tx.QueryRow(ctx, `SELECT id,worker_id,idempotency_key,actor_id,reason,status,created_at,expires_at,updated_at,result FROM worker_restart_commands WHERE worker_id=$1 AND idempotency_key=$2`, workerID, key).Scan(&command.ID, &command.WorkerID, &command.IdempotencyKey, &command.ActorID, &command.Reason, &command.Status, &command.CreatedAt, &command.ExpiresAt, &command.UpdatedAt, &command.Result)
	if err == nil {
		return command, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return RestartCommand{}, fmt.Errorf("read idempotent restart: %w", err)
	}
	var lastSupervisorSeen *time.Time
	if err = tx.QueryRow(ctx, `SELECT last_seen_at FROM worker_supervisors WHERE worker_id=$1 AND revoked_at IS NULL`, workerID).Scan(&lastSupervisorSeen); errors.Is(err, pgx.ErrNoRows) {
		return RestartCommand{}, ErrSupervisorUnavailable
	} else if err != nil {
		return RestartCommand{}, fmt.Errorf("read supervisor heartbeat: %w", err)
	}
	if revoked || lastSupervisorSeen == nil || now.Sub(*lastSupervisorSeen) > 30*time.Second {
		return RestartCommand{}, ErrSupervisorUnavailable
	}
	var pending bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_restart_commands WHERE worker_id=$1 AND status IN ('requested','accepted','stopping','starting') AND expires_at>$2)`, workerID, now).Scan(&pending); err != nil {
		return RestartCommand{}, fmt.Errorf("check pending restart: %w", err)
	}
	if pending {
		return RestartCommand{}, ErrRestartPending
	}
	id := "wr_" + auth.Token()[:28]
	expires := now.Add(5 * time.Minute)
	err = tx.QueryRow(ctx, `INSERT INTO worker_restart_commands(id,worker_id,idempotency_key,actor_id,reason,status,created_at,expires_at,updated_at) VALUES($1,$2,$3,$4,$5,'requested',$6,$7,$6) RETURNING id,worker_id,idempotency_key,actor_id,reason,status,created_at,expires_at,updated_at,result`, id, workerID, key, actorID, reason, now, expires).Scan(&command.ID, &command.WorkerID, &command.IdempotencyKey, &command.ActorID, &command.Reason, &command.Status, &command.CreatedAt, &command.ExpiresAt, &command.UpdatedAt, &command.Result)
	if err != nil {
		return RestartCommand{}, fmt.Errorf("insert restart command: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO worker_restart_events(command_id,actor,status,occurred_at) VALUES($1,$2,'requested',$3)`, id, actorID, now); err != nil {
		return RestartCommand{}, fmt.Errorf("audit restart request: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return RestartCommand{}, fmt.Errorf("commit restart request: %w", err)
	}
	return command, nil
}

// RestartTransitionsForSupervisor expires stale commands before returning one
// actionable command. It never updates the Worker presence timestamp.
func RestartTransitionsForSupervisor(ctx context.Context, pool *pgxpool.Pool, workerID string, now time.Time) (*RestartCommand, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin supervisor poll: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `WITH expired AS (UPDATE worker_restart_commands SET status='expired',updated_at=$2,result='command_expired' WHERE worker_id=$1 AND status IN ('requested','accepted') AND expires_at<=$2 RETURNING id,result) INSERT INTO worker_restart_events(command_id,actor,status,occurred_at,result) SELECT id,'supervisor','expired',$2,result FROM expired`, workerID, now)
	if err != nil {
		return nil, fmt.Errorf("expire restart commands: %w", err)
	}
	var command RestartCommand
	err = tx.QueryRow(ctx, `UPDATE worker_restart_commands SET status='accepted',updated_at=$2 WHERE id=(SELECT id FROM worker_restart_commands WHERE worker_id=$1 AND status='requested' AND expires_at>$2 ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED) RETURNING id,worker_id,idempotency_key,actor_id,reason,status,created_at,expires_at,updated_at,result`, workerID, now).Scan(&command.ID, &command.WorkerID, &command.IdempotencyKey, &command.ActorID, &command.Reason, &command.Status, &command.CreatedAt, &command.ExpiresAt, &command.UpdatedAt, &command.Result)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit supervisor poll: %w", err)
		}
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("read requested restart: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO worker_restart_events(command_id,actor,status,occurred_at) VALUES($1,'supervisor','accepted',$2)`, command.ID, now); err != nil {
		return nil, fmt.Errorf("audit accepted restart: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit supervisor acceptance: %w", err)
	}
	return &command, nil
}

func AuthenticateSupervisor(ctx context.Context, pool *pgxpool.Pool, token string, now time.Time) (string, error) {
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", ErrUnauthorized
	}
	var workerID string
	err := pool.QueryRow(ctx, `UPDATE worker_supervisors s SET last_seen_at=$2 FROM workers w WHERE s.worker_id=w.id AND s.token_hash=$1 AND s.revoked_at IS NULL AND w.revoked_at IS NULL RETURNING s.worker_id`, auth.Hash(token), now).Scan(&workerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUnauthorized
	}
	if err != nil {
		return "", fmt.Errorf("authenticate supervisor: %w", err)
	}
	return workerID, nil
}
