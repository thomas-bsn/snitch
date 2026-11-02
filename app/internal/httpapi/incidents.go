package httpapi

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"

	"snitch/internal/workers"
)

// Technical incidents are raised for administrators, grouped by cause: a retry refreshes
// the active notification of its cause instead of adding one. A notification resolves itself
// when its cause disappears (a Worker came back, the job was retried and succeeded). A new
// cause (another kind, or another failing step) is a distinct notification. Payloads carry
// codes only, never the raw worker error, a path or a stack trace. The owner of the clip
// concerned may also be a recipient, but only ever sees the generic message.

const (
	kindWorkerUnavailable   = "worker_unavailable"
	kindProviderUnavailable = "provider_unavailable"
	kindModelMissing        = "model_missing"
	kindJobFailed           = "job_failed"
	// kindTechnicalIncident is what a non-administrator sees whatever the incident.
	kindTechnicalIncident = "technical_incident"

	// workerGrace is how long a job may wait with no compatible Worker before it is an incident.
	workerGrace = 2 * time.Minute
	// incidentThrottle bounds how often Worker traffic triggers a reconciliation.
	incidentThrottle = 5 * time.Second
)

// incidentSync throttles the reconciliations triggered by Worker traffic.
type incidentSync struct{ at atomic.Int64 }

type incident struct {
	kind, cause, step, code string
	count                   int
	jobID, clipID           int64
}

func (s *Server) currentIncidents(ctx context.Context, tx pgx.Tx) ([]incident, error) {
	var out []incident
	// Waiting for a Worker: a job no connected Worker is able to run, for too long.
	var n int
	var id, clip int64
	var step string
	err := tx.QueryRow(ctx, `SELECT count(*)::int,COALESCE(max(id),0),COALESCE(max(clip_id),0),COALESCE(min(step),'') FROM jobs j
WHERE j.status IN ('pending','retry_wait') AND j.created_at < now()-($1 * interval '1 second') AND NOT EXISTS (
 SELECT 1 FROM workers w, unnest(w.supported_steps) AS capability WHERE w.revoked_at IS NULL AND w.last_seen_at >= now() - ($2 * interval '1 second')
 AND CASE WHEN right(capability,2)='.*' THEN left(j.step,length(capability)-1)=left(capability,length(capability)-1) ELSE j.step=capability END)`,
		int64(workerGrace.Seconds()), int64(workers.ConnectedWindow.Seconds())).Scan(&n, &id, &clip, &step)
	if err != nil {
		return nil, err
	}
	if n > 0 {
		out = append(out, incident{kind: kindWorkerUnavailable, cause: kindWorkerUnavailable, count: n, jobID: id, clipID: clip})
	}
	// Failing jobs (dead, or retrying for a provider/model cause), grouped by cause. A failed
	// job superseded by a later successful one of the same step is no longer an incident.
	rows, err := tx.Query(ctx, `SELECT CASE WHEN j.last_error IN ('provider_unavailable','model_missing') THEN j.last_error ELSE 'job_failed' END AS kind,
 CASE WHEN j.last_error IN ('provider_unavailable','model_missing') THEN '' ELSE j.step END AS grp,
 CASE WHEN j.last_error='' THEN 'retries_exhausted' ELSE j.last_error END AS code,
 count(*)::int,max(j.id),max(j.clip_id)
FROM jobs j WHERE NOT EXISTS (SELECT 1 FROM jobs d WHERE d.clip_id=j.clip_id AND d.step=j.step AND d.status='done' AND d.id>j.id)
AND (j.status='dead' OR (j.status='retry_wait' AND j.last_error IN ('provider_unavailable','model_missing')))
GROUP BY 1,2,3 ORDER BY 1,2,3`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var in incident
		var grp string
		if err = rows.Scan(&in.kind, &grp, &in.code, &in.count, &in.jobID, &in.clipID); err != nil {
			return nil, err
		}
		in.cause = in.kind
		if in.kind == kindJobFailed {
			in.cause = kindJobFailed + ":" + grp + ":" + in.code
			in.step = grp
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

// syncIncidents brings the incident notifications in line with the current causes. It is
// idempotent; failures are logged and never undo the triggering change.
func (s *Server) syncIncidents(ctx context.Context) {
	if err := s.trySyncIncidents(context.WithoutCancel(ctx)); err != nil {
		s.Log.Error("incident sync failed", "error", err)
	}
}

// syncIncidentsSoon is the cheap variant for hot paths such as Worker polling.
func (s *Server) syncIncidentsSoon(ctx context.Context) {
	now := time.Now().UnixNano()
	last := s.incidents.at.Load()
	if now-last < int64(incidentThrottle) || !s.incidents.at.CompareAndSwap(last, now) {
		return
	}
	s.syncIncidents(ctx)
}

func (s *Server) trySyncIncidents(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('notification:incidents',0))`); err != nil {
		return err
	}
	current, err := s.currentIncidents(ctx, tx)
	if err != nil {
		return err
	}
	causes := make([]string, 0, len(current))
	for _, in := range current {
		causes = append(causes, in.cause)
		var clip any
		if in.clipID > 0 {
			clip = in.clipID
		}
		var id int64
		// A refresh bumps the notification only when the incident really changed.
		err = tx.QueryRow(ctx, `UPDATE notifications SET pending_count=$3,job_id=$4,step=$5,error_code=$6,clip_id=$7,
updated_at=CASE WHEN pending_count=$3 AND job_id IS NOT DISTINCT FROM $4 THEN updated_at ELSE now() END
WHERE kind=$1 AND cause=$2 AND state='active' RETURNING id`, in.kind, in.cause, in.count, in.jobID, in.step, in.code, clip).Scan(&id)
		if err == pgx.ErrNoRows {
			err = tx.QueryRow(ctx, `INSERT INTO notifications(kind,cause,pending_count,job_id,step,error_code,clip_id) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, in.kind, in.cause, in.count, in.jobID, in.step, in.code, clip).Scan(&id)
		}
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO notification_recipients(notification_id,user_id)
SELECT $1,u.id FROM users u WHERE u.approved AND u.deleted_at IS NULL AND (u.role IN ('admin','super_admin') OR u.id=(SELECT owner_id FROM clips WHERE id=$2))
ON CONFLICT DO NOTHING`, id, clip); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE notifications SET state='resolved',resolved_at=now(),updated_at=now(),pending_count=0
WHERE cause<>'' AND state='active' AND NOT (cause = ANY($1::text[]))`, causes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// serveIncidents reconciles at once, then on every tick, until ctx ends.
func (s *Server) serveIncidents(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		s.syncIncidents(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
