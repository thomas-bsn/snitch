package jobs

import (
	"context"
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
	ID             int64      `json:"id"`
	ClipID         int64      `json:"clip_id"`
	Step           string     `json:"step"`
	InputVersion   string     `json:"input_version"`
	Status         Status     `json:"status"`
	Priority       int        `json:"priority"`
	Attempts       int        `json:"attempts"`
	MaxAttempts    int        `json:"max_attempts"`
	LeaseOwner     *string    `json:"lease_owner"`
	LeaseExpiresAt *time.Time `json:"lease_expires_at"`
	NextAttemptAt  time.Time  `json:"next_attempt_at"`
	LastError      string     `json:"last_error"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
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
		job, err := scanJob(tx.QueryRow(ctx, `SELECT id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at FROM jobs WHERE clip_id=$1 AND step=$2 AND input_version=$3 AND status IN ('pending','leased') FOR UPDATE`, input.ClipID, input.Step, input.InputVersion))
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
		job, err = scanJob(tx.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,priority) VALUES($1,$2,$3,$4) ON CONFLICT (clip_id,step,input_version) WHERE status IN ('pending','leased') DO NOTHING RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at`, input.ClipID, input.Step, input.InputVersion, input.Priority))
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
	rows, err := s.pool.Query(ctx, `SELECT id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at FROM jobs WHERE ($1='' OR status=$1) AND ($2=0 OR clip_id=$2) AND ($3='' OR step=$3) ORDER BY priority DESC,created_at,id LIMIT $4 OFFSET $5`, args...)
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

type scanner interface{ Scan(...any) error }

func scanJob(row scanner) (Job, error) {
	var job Job
	err := row.Scan(&job.ID, &job.ClipID, &job.Step, &job.InputVersion, &job.Status, &job.Priority, &job.Attempts, &job.MaxAttempts, &job.LeaseOwner, &job.LeaseExpiresAt, &job.NextAttemptAt, &job.LastError, &job.CreatedAt, &job.UpdatedAt)
	return job, err
}
