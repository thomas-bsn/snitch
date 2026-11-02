package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Stable codes explaining why a pending job is not leased yet. The empty cause is a normal wait: a compatible
// Worker is connected and eligible, only busy.
const (
	CauseQueuePaused              = "queue_paused"
	CauseNoCompatibleWorker       = "no_compatible_worker"
	CauseTranscriptionMismatch    = "transcription_version_mismatch"
	CauseTranscriptionNotVerified = "transcription_not_validated"
)

// DurableCause reports whether a cause lasts until someone acts, as opposed to a normal wait or a deliberate pause.
func DurableCause(cause string) bool {
	return cause == CauseTranscriptionMismatch || cause == CauseTranscriptionNotVerified
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WaitCause explains why a pending job has not been leased, using the same eligibility rules as Claim.
// connected is how recently a Worker must have been seen to count as connected.
func (s *Service) WaitCause(ctx context.Context, jobID int64, connected time.Duration) (string, error) {
	var step string
	if err := s.pool.QueryRow(ctx, `SELECT step FROM jobs WHERE id=$1`, jobID).Scan(&step); err != nil {
		return "", fmt.Errorf("read job step: %w", err)
	}
	return StepWaitCause(ctx, s.pool, step, connected)
}

// StepWaitCause is WaitCause for a step, which is all the rules depend on.
func StepWaitCause(ctx context.Context, q rowQuerier, step string, connected time.Duration) (string, error) {
	var paused bool
	if err := q.QueryRow(ctx, `SELECT paused FROM job_control WHERE id=true`).Scan(&paused); err != nil {
		return "", fmt.Errorf("read queue control: %w", err)
	}
	if paused {
		return CauseQueuePaused, nil
	}
	var capable, eligible, failed int
	err := q.QueryRow(ctx, `
		SELECT count(*)::int,
		       count(*) FILTER (WHERE $3 = false OR COALESCE(c.active_version,'')='' OR c.active_version=(SELECT ts.version FROM transcription_settings ts WHERE ts.id=true))::int,
		       count(*) FILTER (WHERE COALESCE(c.error_code,'')<>'')::int
		FROM workers w LEFT JOIN worker_transcription_config c ON c.worker_id=w.id
		WHERE w.revoked_at IS NULL AND w.last_seen_at >= now() - ($2 * interval '1 second')
		  AND EXISTS (
		      SELECT 1 FROM unnest(w.supported_steps) AS capability
		      WHERE CASE WHEN right(capability,2)='.*'
		            THEN left($1::text,length(capability)-1)=left(capability,length(capability)-1)
		            ELSE $1::text=capability END)`,
		step, int64(connected.Seconds()), strings.HasPrefix(step, "transcription.")).Scan(&capable, &eligible, &failed)
	if err != nil {
		return "", fmt.Errorf("diagnose pending step: %w", err)
	}
	switch {
	case capable == 0:
		return CauseNoCompatibleWorker, nil
	case eligible > 0:
		return "", nil
	case failed > 0:
		return CauseTranscriptionNotVerified, nil
	}
	return CauseTranscriptionMismatch, nil
}
