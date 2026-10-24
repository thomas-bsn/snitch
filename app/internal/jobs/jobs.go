package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Status string

const (
	Pending Status = "pending"
	Leased  Status = "leased"
	Done    Status = "done"
	Failed  Status = "failed"
)

var (
	ErrInvalidInput = errors.New("invalid job input")
	ErrClipNotFound = errors.New("clip not found")
	stepPattern     = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)
)

type Job struct {
	ID             int64           `json:"id"`
	ClipID         int64           `json:"clip_id"`
	Step           string          `json:"step"`
	InputVersion   string          `json:"input_version"`
	Status         Status          `json:"status"`
	Priority       int             `json:"priority"`
	Attempts       int             `json:"attempts"`
	MaxAttempts    int             `json:"max_attempts"`
	LeaseOwner     *string         `json:"lease_owner"`
	LeaseExpiresAt *time.Time      `json:"lease_expires_at"`
	NextAttemptAt  time.Time       `json:"next_attempt_at"`
	LastError      string          `json:"last_error"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Result         json.RawMessage `json:"result,omitempty"`
}

type ScheduleInput struct {
	ClipID       int64
	Step         string
	InputVersion string
	Priority     int
}

type Filter struct {
	Status   Status
	ClipID   int64
	Step     string
	Page     int
	PageSize int
}

type Page struct {
	Items    []Job `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type Input struct {
	Job             Job             `json:"job"`
	OriginalName    string          `json:"original_name"`
	SizeBytes       int64           `json:"size_bytes"`
	DurationSeconds *float64        `json:"duration_seconds"`
	VideoCodec      string          `json:"video_codec"`
	AudioCodec      string          `json:"audio_codec"`
	Tracks          json.RawMessage `json:"tracks"`
	MediaURL        string          `json:"media_url"`
}

type Service struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// Schedule creates one active job per clip, step, and input version. Concurrent
// callers receive the same persisted job rather than creating duplicate work.
func (s *Service) Schedule(ctx context.Context, input ScheduleInput) (Job, bool, error) {
	if input.ClipID < 1 || !stepPattern.MatchString(input.Step) || input.InputVersion == "" || len(input.InputVersion) > 128 || strings.TrimSpace(input.InputVersion) != input.InputVersion || input.Priority < -100 || input.Priority > 100 {
		return Job{}, false, ErrInvalidInput
	}
	for attempt := 0; attempt < 5; attempt++ {
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return Job{}, false, fmt.Errorf("begin job schedule: %w", err)
		}
		job, err := scanJob(tx.QueryRow(ctx, `SELECT id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result FROM jobs WHERE clip_id=$1 AND step=$2 AND input_version=$3 AND status IN ('pending','leased') FOR UPDATE`, input.ClipID, input.Step, input.InputVersion))
		if err == nil {
			if err = tx.Commit(ctx); err != nil {
				return Job{}, false, fmt.Errorf("commit existing job schedule: %w", err)
			}
			return job, false, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			return Job{}, false, fmt.Errorf("find active job: %w", err)
		}
		job, err = scanJob(tx.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,priority) VALUES($1,$2,$3,$4) ON CONFLICT (clip_id,step,input_version) WHERE status IN ('pending','leased') DO NOTHING RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result`, input.ClipID, input.Step, input.InputVersion, input.Priority))
		if errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			continue
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return Job{}, false, ErrClipNotFound
			}
			return Job{}, false, fmt.Errorf("insert job: %w", err)
		}
		if err = tx.Commit(ctx); err != nil {
			return Job{}, false, fmt.Errorf("commit new job: %w", err)
		}
		return job, true, nil
	}
	return Job{}, false, errors.New("could not resolve concurrent job schedule")
}

func (s *Service) List(ctx context.Context, filter Filter) (Page, error) {
	if filter.Page < 1 || filter.Page > 100000 || filter.PageSize < 1 || filter.PageSize > 100 || (filter.Status != "" && filter.Status != Pending && filter.Status != Leased && filter.Status != Done && filter.Status != Failed) || filter.ClipID < 0 || (filter.Step != "" && !stepPattern.MatchString(filter.Step)) {
		return Page{}, ErrInvalidInput
	}
	args := []any{filter.Status, filter.ClipID, filter.Step}
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE ($1='' OR status=$1) AND ($2=0 OR clip_id=$2) AND ($3='' OR step=$3)`, args...).Scan(&total); err != nil {
		return Page{}, fmt.Errorf("count jobs: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.pool.Query(ctx, `SELECT id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result FROM jobs WHERE ($1='' OR status=$1) AND ($2=0 OR clip_id=$2) AND ($3='' OR step=$3) ORDER BY priority DESC,created_at,id LIMIT $4 OFFSET $5`, args...)
	if err != nil {
		return Page{}, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()
	items := []Job{}
	for rows.Next() {
		job, scanErr := scanJob(rows)
		if scanErr != nil {
			return Page{}, fmt.Errorf("scan job: %w", scanErr)
		}
		items = append(items, job)
	}
	if err = rows.Err(); err != nil {
		return Page{}, fmt.Errorf("read jobs: %w", err)
	}
	return Page{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, nil
}

// Claim leases the highest-priority available task without exceeding the
// worker's configured or currently reported concurrency.
func (s *Service) Claim(ctx context.Context, workerID string) (*Job, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin job claim: %w", err)
	}
	defer tx.Rollback(ctx)
	var maximum, reported int
	err = tx.QueryRow(ctx, `SELECT max_concurrency,reported_concurrency FROM workers WHERE id=$1 AND revoked_at IS NULL FOR UPDATE`, workerID).Scan(&maximum, &reported)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("read worker capacity for claim: %w", err)
	}
	capacity := min(maximum, reported)
	var active int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE status='leased' AND lease_owner=$1 AND lease_expires_at>now()`, workerID).Scan(&active); err != nil {
		return nil, fmt.Errorf("count active worker leases: %w", err)
	}
	if capacity < 1 || active >= capacity {
		return nil, tx.Commit(ctx)
	}
	job, err := scanJob(tx.QueryRow(ctx, `SELECT id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result FROM jobs WHERE status='pending' AND next_attempt_at<=now() ORDER BY priority DESC,created_at,id FOR UPDATE SKIP LOCKED LIMIT 1`))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("select job for claim: %w", err)
	}
	job, err = scanJob(tx.QueryRow(ctx, `UPDATE jobs SET status='leased',lease_owner=$2,lease_expires_at=now()+interval '2 minutes',attempts=attempts+1,updated_at=now() WHERE id=$1 RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result`, job.ID, workerID))
	if err != nil {
		return nil, fmt.Errorf("lease claimed job: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit job claim: %w", err)
	}
	return &job, nil
}

func (s *Service) Input(ctx context.Context, id int64, workerID string) (Input, error) {
	var input Input
	err := s.pool.QueryRow(ctx, `SELECT j.id,j.clip_id,j.step,j.input_version,j.status,j.priority,j.attempts,j.max_attempts,j.lease_owner,j.lease_expires_at,j.next_attempt_at,j.last_error,j.created_at,j.updated_at,j.result,c.original_name,c.size_bytes,c.duration_seconds,c.video_codec,c.audio_codec,c.tracks FROM jobs j JOIN clips c ON c.id=j.clip_id WHERE j.id=$1 AND j.status='leased' AND j.lease_owner=$2 AND j.lease_expires_at>now()`, id, workerID).Scan(&input.Job.ID, &input.Job.ClipID, &input.Job.Step, &input.Job.InputVersion, &input.Job.Status, &input.Job.Priority, &input.Job.Attempts, &input.Job.MaxAttempts, &input.Job.LeaseOwner, &input.Job.LeaseExpiresAt, &input.Job.NextAttemptAt, &input.Job.LastError, &input.Job.CreatedAt, &input.Job.UpdatedAt, &input.Job.Result, &input.OriginalName, &input.SizeBytes, &input.DurationSeconds, &input.VideoCodec, &input.AudioCodec, &input.Tracks)
	if errors.Is(err, pgx.ErrNoRows) {
		return Input{}, ErrLeaseNotOwned
	}
	if err != nil {
		return Input{}, fmt.Errorf("read worker job input: %w", err)
	}
	input.MediaURL = fmt.Sprintf("/api/worker/jobs/%d/media", id)
	return input, nil
}

func (s *Service) Heartbeat(ctx context.Context, id int64, workerID string) error {
	result, err := s.pool.Exec(ctx, `UPDATE jobs SET lease_expires_at=now()+interval '2 minutes',updated_at=now() WHERE id=$1 AND status='leased' AND lease_owner=$2 AND lease_expires_at>now()`, id, workerID)
	if err != nil {
		return fmt.Errorf("renew job lease: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLeaseNotOwned
	}
	return nil
}

func (s *Service) Complete(ctx context.Context, id int64, workerID, inputVersion string, result json.RawMessage) (Job, error) {
	if !json.Valid(result) || len(result) == 0 || len(result) > 12*1024 {
		return Job{}, ErrInvalidInput
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(result, &object); err != nil || object == nil {
		return Job{}, ErrInvalidInput
	}
	job, err := scanJob(s.pool.QueryRow(ctx, `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,result=$4::jsonb,last_error='',updated_at=now() WHERE id=$1 AND lease_owner=$2 AND status='leased' AND lease_expires_at>now() AND input_version=$3 RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result`, id, workerID, inputVersion, []byte(result)))
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrLeaseNotOwned
	}
	if err != nil {
		return Job{}, fmt.Errorf("complete worker job: %w", err)
	}
	return job, nil
}

func (s *Service) Fail(ctx context.Context, id int64, workerID, message string) (Job, error) {
	if len(message) > 256 || strings.TrimSpace(message) != message {
		return Job{}, ErrInvalidInput
	}
	// Never persist worker-provided stack traces, paths, or arbitrary details.
	code := "worker_failed"
	if message == "provider_unavailable" || message == "invalid_input" {
		code = message
	}
	job, err := scanJob(s.pool.QueryRow(ctx, `UPDATE jobs SET status='failed',lease_owner=NULL,lease_expires_at=NULL,last_error=$3,updated_at=now() WHERE id=$1 AND lease_owner=$2 AND status='leased' AND lease_expires_at>now() RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result`, id, workerID, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrLeaseNotOwned
	}
	if err != nil {
		return Job{}, fmt.Errorf("fail worker job: %w", err)
	}
	return job, nil
}

var ErrLeaseNotOwned = errors.New("worker does not own active job lease")

type scanner interface{ Scan(...any) error }

func scanJob(row scanner) (Job, error) {
	var job Job
	err := row.Scan(&job.ID, &job.ClipID, &job.Step, &job.InputVersion, &job.Status, &job.Priority, &job.Attempts, &job.MaxAttempts, &job.LeaseOwner, &job.LeaseExpiresAt, &job.NextAttemptAt, &job.LastError, &job.CreatedAt, &job.UpdatedAt, &job.Result)
	return job, err
}
