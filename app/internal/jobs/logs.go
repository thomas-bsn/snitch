package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"snitch/internal/auth"
)

// KeptLogLines is how many of its most recent log lines a job keeps (ADR 0028).
const KeptLogLines = 50

// JobLogLine is one line of a job's log, as the Worker sent it (already masked there).
type JobLogLine struct {
	Seq  int64  `json:"seq"`
	Text string `json:"text"`
}

// AppendLogs stores a batch of log lines sent by the Worker holding the lease, keeps only the
// KeptLogLines most recent lines of the job and signals the change on the job's event topic.
// Lines are masked again here (ScrubLogLine) before storage: the Worker masks them too, and the
// server never trusts it to have done so.
func (s *Service) AppendLogs(ctx context.Context, id int64, workerID, leaseToken string, lines []string) (int, error) {
	if !validLeaseToken(leaseToken) {
		return 0, ErrLeaseNotOwned
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin job logs: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var owned int64
	err = tx.QueryRow(ctx, `SELECT id FROM jobs WHERE id=$1 AND status='leased' AND lease_owner=$2 AND lease_token_hash=$3 AND lease_expires_at>now() FOR UPDATE`, id, workerID, auth.Hash(leaseToken)).Scan(&owned)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrLeaseNotOwned
	}
	if err != nil {
		return 0, fmt.Errorf("check job lease for logs: %w", err)
	}
	masked := make([]string, len(lines))
	for i, line := range lines {
		masked[i] = ScrubLogLine(line)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO job_log_lines (job_id, text) SELECT $1, unnest($2::text[])`, id, masked); err != nil {
		return 0, fmt.Errorf("store job logs: %w", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM job_log_lines WHERE job_id=$1 AND id <= COALESCE((SELECT id FROM job_log_lines WHERE job_id=$1 ORDER BY id DESC OFFSET $2 LIMIT 1), 0)`, id, KeptLogLines); err != nil {
		return 0, fmt.Errorf("trim job logs: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO events (topic, kind) VALUES ($1, 'lines')`, JobLogsTopic(id)); err != nil {
		return 0, fmt.Errorf("signal job logs: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit job logs: %w", err)
	}
	return len(lines), nil
}

// JobLogsTopic is the event topic of a job's log lines.
func JobLogsTopic(id int64) string {
	return fmt.Sprintf("job_logs:%d", id)
}

// LogSnapshot returns the kept lines of a job, oldest first. An empty list is a valid answer.
func (s *Service) LogSnapshot(ctx context.Context, id int64) ([]JobLogLine, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, text FROM (SELECT id, text FROM job_log_lines WHERE job_id=$1 ORDER BY id DESC LIMIT $2) recent ORDER BY id`, id, KeptLogLines)
	if err != nil {
		return nil, fmt.Errorf("read job logs: %w", err)
	}
	defer rows.Close()
	lines := []JobLogLine{}
	for rows.Next() {
		var line JobLogLine
		if err := rows.Scan(&line.Seq, &line.Text); err != nil {
			return nil, fmt.Errorf("scan job logs: %w", err)
		}
		lines = append(lines, line)
	}
	return lines, rows.Err()
}
