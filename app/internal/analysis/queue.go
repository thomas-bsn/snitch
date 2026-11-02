package analysis

import (
	"context"
	"fmt"

	"snitch/internal/jobs"
)

// CompletedRuns is how many finished clips the queue keeps in view; the rest is history.
const CompletedRuns = 3

type QueueRun struct {
	Run
	ClipName string `json:"clip_name"`
}

type QueueJob struct {
	jobs.Job
	ClipName string `json:"clip_name"`
}

// Queue is the state of the Task Queue: one entry per analysis run, never paginated, plus the
// technical tasks that belong to no run so they are never lost.
type Queue struct {
	// Active lists the runs that still need attention: running, waiting, blocked, or failed
	// and not superseded by a newer run of the same clip.
	Active []QueueRun `json:"active"`
	// Completed lists the last clips whose automatic processing finished.
	Completed []QueueRun `json:"completed"`
	// Tasks lists the unfinished jobs that no listed run follows.
	Tasks []QueueJob `json:"tasks"`
}

func (s *Service) queueRuns(ctx context.Context, where, order string, limit int) ([]QueueRun, error) {
	rows, err := s.pool.Query(ctx, `SELECT r.id,r.clip_id,r.scope,r.status,r.started_by,r.created_at,r.updated_at,r.finished_at,c.original_name
FROM clip_analysis_runs r JOIN clips c ON c.id=r.clip_id
WHERE r.id=(SELECT max(id) FROM clip_analysis_runs WHERE clip_id=r.clip_id) AND (`+where+`) ORDER BY `+order+` LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list queue runs: %w", err)
	}
	defer rows.Close()
	out := []QueueRun{}
	ids := []int64{}
	for rows.Next() {
		var r QueueRun
		if err = rows.Scan(&r.ID, &r.ClipID, &r.Scope, &r.Status, &r.StartedBy, &r.CreatedAt, &r.UpdatedAt, &r.FinishedAt, &r.ClipName); err != nil {
			return nil, err
		}
		r.Steps = []Step{}
		out = append(out, r)
		ids = append(ids, r.ID)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	steps, err := s.pool.Query(ctx, `SELECT run_id,step,status,reason,job_id,attempt,started_at,finished_at,updated_at FROM clip_analysis_steps WHERE run_id=ANY($1) ORDER BY run_id,array_position($2::text[],step)`, ids, stepOrder)
	if err != nil {
		return nil, fmt.Errorf("list queue steps: %w", err)
	}
	defer steps.Close()
	index := map[int64]int{}
	for i, r := range out {
		index[r.ID] = i
	}
	for steps.Next() {
		var runID int64
		var st Step
		if err = steps.Scan(&runID, &st.Step, &st.Status, &st.Reason, &st.JobID, &st.Attempt, &st.StartedAt, &st.FinishedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		out[index[runID]].Steps = append(out[index[runID]].Steps, st)
	}
	return out, steps.Err()
}

// Queue reads the whole queue at once.
func (s *Service) Queue(ctx context.Context) (Queue, error) {
	var q Queue
	var err error
	if q.Active, err = s.queueRuns(ctx, `r.status IN ('active','failed')`, `r.created_at,r.id`, 100000); err != nil {
		return q, err
	}
	if q.Completed, err = s.queueRuns(ctx, `r.status IN ('completed','incomplete')`, `r.finished_at DESC NULLS LAST,r.id DESC`, CompletedRuns); err != nil {
		return q, err
	}
	rows, err := s.pool.Query(ctx, `SELECT j.id,j.clip_id,j.step,j.input_version,j.status,j.priority,j.attempts,j.max_attempts,j.lease_owner,j.lease_expires_at,j.next_attempt_at,j.last_error,j.created_at,j.updated_at,j.cycle,c.original_name
FROM jobs j JOIN clips c ON c.id=j.clip_id
WHERE j.status IN ('pending','leased','retry_wait','dead')
 AND NOT EXISTS (SELECT 1 FROM clip_analysis_steps st JOIN clip_analysis_runs r ON r.id=st.run_id WHERE st.job_id=j.id AND r.status IN ('active','failed') AND r.id=(SELECT max(id) FROM clip_analysis_runs WHERE clip_id=r.clip_id))
 AND NOT EXISTS (SELECT 1 FROM jobs n WHERE n.clip_id=j.clip_id AND n.step=j.step AND n.id>j.id)
ORDER BY j.priority DESC,j.created_at,j.id`)
	if err != nil {
		return q, fmt.Errorf("list queue tasks: %w", err)
	}
	defer rows.Close()
	q.Tasks = []QueueJob{}
	for rows.Next() {
		var j QueueJob
		if err = rows.Scan(&j.ID, &j.ClipID, &j.Step, &j.InputVersion, &j.Status, &j.Priority, &j.Attempts, &j.MaxAttempts, &j.LeaseOwner, &j.LeaseExpiresAt, &j.NextAttemptAt, &j.LastError, &j.CreatedAt, &j.UpdatedAt, &j.Cycle, &j.ClipName); err != nil {
			return q, err
		}
		q.Tasks = append(q.Tasks, j)
	}
	return q, rows.Err()
}
