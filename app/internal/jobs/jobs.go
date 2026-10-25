package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/auth"
)

type Status string

const (
	Pending   Status = "pending"
	Leased    Status = "leased"
	Done      Status = "done"
	Failed    Status = "failed"
	RetryWait Status = "retry_wait"
	Cancelled Status = "cancelled"
	Dead      Status = "dead"
)

var (
	ErrInvalidInput    = errors.New("invalid job input")
	ErrClipNotFound    = errors.New("clip not found")
	ErrJobNotFound     = errors.New("job not found")
	ErrNotCancellable  = errors.New("job cannot be cancelled")
	ErrNotRelaunchable = errors.New("job is not dead")
	stepPattern        = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)
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
	Cycle          int             `json:"cycle"`
	LeaseOwner     *string         `json:"lease_owner"`
	LeaseExpiresAt *time.Time      `json:"lease_expires_at"`
	NextAttemptAt  time.Time       `json:"next_attempt_at"`
	LastError      string          `json:"last_error"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Result         json.RawMessage `json:"result,omitempty"`
}

type Assignment struct {
	Job        Job    `json:"job"`
	LeaseToken string `json:"lease_token"`
}

type ScheduleInput struct {
	ClipID       int64
	Step         string
	InputVersion string
	Priority     int
	MaxAttempts  int
}

type Event struct {
	ID        int64     `json:"id"`
	Event     string    `json:"event"`
	Cycle     int       `json:"cycle"`
	Attempt   *int      `json:"attempt"`
	Actor     string    `json:"actor"`
	ActorID   string    `json:"actor_id"`
	ErrorCode string    `json:"error_code"`
	CreatedAt time.Time `json:"created_at"`
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
	Job             Job              `json:"job"`
	OriginalName    string           `json:"original_name"`
	SizeBytes       int64            `json:"size_bytes"`
	DurationSeconds *float64         `json:"duration_seconds"`
	VideoCodec      string           `json:"video_codec"`
	AudioCodec      string           `json:"audio_codec"`
	Tracks          json.RawMessage  `json:"tracks"`
	MediaURL        string           `json:"media_url"`
	Vocabulary      []VocabularyTerm `json:"vocabulary"`
	Pipeline        json.RawMessage  `json:"pipeline"`
	PipelineVersion string           `json:"pipeline_version"`
	Voice           json.RawMessage  `json:"voice,omitempty"`
}

type VocabularyTerm struct {
	Term              string   `json:"term"`
	Variants          []string `json:"variants"`
	PreferredSpelling string   `json:"preferred_spelling"`
}

type Service struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// Schedule creates one active job per clip, step, and input version. Concurrent
// callers receive the same persisted job rather than creating duplicate work.
func (s *Service) Schedule(ctx context.Context, input ScheduleInput) (Job, bool, error) {
	if input.MaxAttempts == 0 {
		input.MaxAttempts = 3
	}
	if input.ClipID < 1 || !stepPattern.MatchString(input.Step) || input.InputVersion == "" || len(input.InputVersion) > 128 || strings.TrimSpace(input.InputVersion) != input.InputVersion || input.Priority < -100 || input.Priority > 100 || input.MaxAttempts < 1 || input.MaxAttempts > 20 {
		return Job{}, false, ErrInvalidInput
	}
	for attempt := 0; attempt < 5; attempt++ {
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return Job{}, false, fmt.Errorf("begin job schedule: %w", err)
		}
		var lockedClipID int64
		err = tx.QueryRow(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, input.ClipID).Scan(&lockedClipID)
		if errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			return Job{}, false, ErrClipNotFound
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return Job{}, false, fmt.Errorf("lock clip before scheduling step: %w", err)
		}
		if _, err = tx.Exec(ctx, `UPDATE jobs SET status='failed',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,last_error='input_version_superseded',updated_at=now() WHERE clip_id=$1 AND step=$2 AND input_version<>$3 AND status IN ('pending','leased','retry_wait')`, input.ClipID, input.Step, input.InputVersion); err != nil {
			_ = tx.Rollback(ctx)
			return Job{}, false, fmt.Errorf("invalidate superseded step version: %w", err)
		}
		job, err := scanJob(tx.QueryRow(ctx, `SELECT id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result,cycle FROM jobs WHERE clip_id=$1 AND step=$2 AND input_version=$3 AND status IN ('pending','leased','retry_wait') FOR UPDATE`, input.ClipID, input.Step, input.InputVersion))
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
		job, err = scanJob(tx.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,priority,max_attempts) VALUES($1,$2,$3,$4,$5) ON CONFLICT (clip_id,step,input_version) WHERE status IN ('pending','leased','retry_wait') DO NOTHING RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result,cycle`, input.ClipID, input.Step, input.InputVersion, input.Priority, input.MaxAttempts))
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
	if filter.Page < 1 || filter.Page > 100000 || filter.PageSize < 1 || filter.PageSize > 100 || (filter.Status != "" && filter.Status != Pending && filter.Status != Leased && filter.Status != Done && filter.Status != Failed && filter.Status != RetryWait && filter.Status != Cancelled && filter.Status != Dead) || filter.ClipID < 0 || (filter.Step != "" && !stepPattern.MatchString(filter.Step)) {
		return Page{}, ErrInvalidInput
	}
	args := []any{filter.Status, filter.ClipID, filter.Step}
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE ($1='' OR status=$1) AND ($2=0 OR clip_id=$2) AND ($3='' OR step=$3)`, args...).Scan(&total); err != nil {
		return Page{}, fmt.Errorf("count jobs: %w", err)
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.pool.Query(ctx, `SELECT id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,CASE WHEN step LIKE 'voice.fingerprint.%' THEN NULL ELSE result END,cycle FROM jobs WHERE ($1='' OR status=$1) AND ($2=0 OR clip_id=$2) AND ($3='' OR step=$3) ORDER BY priority DESC,created_at,id LIMIT $4 OFFSET $5`, args...)
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

func (s *Service) History(ctx context.Context, jobID int64) ([]Event, error) {
	if jobID < 1 {
		return nil, ErrJobNotFound
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE id=$1)`, jobID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check job history: %w", err)
	}
	if !exists {
		return nil, ErrJobNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT id,event,cycle,attempt,actor,actor_id,error_code,created_at FROM job_events WHERE job_id=$1 ORDER BY id`, jobID)
	if err != nil {
		return nil, fmt.Errorf("read job history: %w", err)
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var event Event
		if err = rows.Scan(&event.ID, &event.Event, &event.Cycle, &event.Attempt, &event.Actor, &event.ActorID, &event.ErrorCode, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan job history: %w", err)
		}
		events = append(events, event)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read job history rows: %w", err)
	}
	return events, nil
}

func (s *Service) Relaunch(ctx context.Context, jobID, adminID int64) (Job, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Job{}, fmt.Errorf("begin job relaunch: %w", err)
	}
	defer tx.Rollback(ctx)
	var previousError string
	err = tx.QueryRow(ctx, `SELECT last_error FROM jobs WHERE id=$1 AND status='dead' FOR UPDATE`, jobID).Scan(&previousError)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if checkErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE id=$1)`, jobID).Scan(&exists); checkErr != nil {
			return Job{}, fmt.Errorf("check job relaunch target: %w", checkErr)
		}
		if !exists {
			return Job{}, ErrJobNotFound
		}
		return Job{}, ErrNotRelaunchable
	}
	if err != nil {
		return Job{}, fmt.Errorf("lock dead job: %w", err)
	}
	job, err := scanJob(tx.QueryRow(ctx, `UPDATE jobs SET status='pending',attempts=0,cycle=cycle+1,next_attempt_at=now(),last_error='',updated_at=now() WHERE id=$1 RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result,cycle`, jobID))
	if err != nil {
		return Job{}, fmt.Errorf("relaunch dead job: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,actor,actor_id,error_code) VALUES($1,'admin_relaunched',$2,'admin',$3,$4)`, jobID, job.Cycle, strconv.FormatInt(adminID, 10), previousError); err != nil {
		return Job{}, fmt.Errorf("record job relaunch: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Job{}, fmt.Errorf("commit job relaunch: %w", err)
	}
	return job, nil
}

func (s *Service) Cancel(ctx context.Context, jobID, adminID int64) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin job cancellation: %w", err)
	}
	defer tx.Rollback(ctx)
	var cycle, attempt int
	var status Status
	err = tx.QueryRow(ctx, `SELECT cycle,attempts,status FROM jobs WHERE id=$1 FOR UPDATE`, jobID).Scan(&cycle, &attempt, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrJobNotFound
	}
	if err != nil {
		return fmt.Errorf("lock job to cancel: %w", err)
	}
	if status != Pending && status != RetryWait && status != Leased {
		return ErrNotCancellable
	}
	if _, err = tx.Exec(ctx, `UPDATE jobs SET status='cancelled',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,updated_at=now() WHERE id=$1`, jobID); err != nil {
		return fmt.Errorf("cancel job: %w", err)
	}
	var attemptValue any
	if status == Leased && attempt > 0 {
		attemptValue = attempt
	}
	if _, err = tx.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,attempt,actor,actor_id,error_code) VALUES($1,'cancelled',$2,$3,'admin',$4,'cancelled_by_admin')`, jobID, cycle, attemptValue, strconv.FormatInt(adminID, 10)); err != nil {
		return fmt.Errorf("record job cancellation: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit job cancellation: %w", err)
	}
	return nil
}

// Claim leases the highest-priority available task without exceeding the
// worker's configured or currently reported concurrency.
func (s *Service) Claim(ctx context.Context, workerID string) (*Assignment, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin job claim: %w", err)
	}
	defer tx.Rollback(ctx)
	// Serialize claims against control updates. An administrator's pause takes
	// effect atomically: when the update returns, no claim can still cross it.
	var paused bool
	var globalCapacity int
	if err = tx.QueryRow(ctx, `SELECT paused,max_concurrency FROM job_control WHERE id=true FOR UPDATE`).Scan(&paused, &globalCapacity); err != nil {
		return nil, fmt.Errorf("read global job controls for claim: %w", err)
	}
	if paused {
		return nil, tx.Commit(ctx)
	}
	if _, err = tx.Exec(ctx, `WITH expired AS (
		UPDATE jobs SET status=CASE WHEN attempts>=max_attempts THEN 'dead' ELSE 'pending' END,
		 lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,last_error='lease_expired',updated_at=now()
		 WHERE status='leased' AND lease_expires_at<=now()
		 RETURNING id,attempts,cycle,status
	) INSERT INTO job_events(job_id,event,cycle,attempt,actor,error_code)
	  SELECT id,CASE WHEN status='dead' THEN 'marked_dead' ELSE 'lease_expired' END,cycle,attempts,'system',CASE WHEN status='dead' THEN 'max_attempts_exhausted' ELSE 'lease_expired' END FROM expired`); err != nil {
		return nil, fmt.Errorf("recover expired job leases: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE jobs SET status='pending',updated_at=now() WHERE status='retry_wait' AND next_attempt_at<=now()`); err != nil {
		return nil, fmt.Errorf("activate due job retries: %w", err)
	}
	var maximum, reported int
	var supportedSteps []string
	err = tx.QueryRow(ctx, `SELECT max_concurrency,reported_concurrency,supported_steps FROM workers WHERE id=$1 AND revoked_at IS NULL FOR UPDATE`, workerID).Scan(&maximum, &reported, &supportedSteps)
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
	var globalActive int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE status='leased' AND lease_expires_at>now()`).Scan(&globalActive); err != nil {
		return nil, fmt.Errorf("count active global leases: %w", err)
	}
	if capacity < 1 || active >= capacity || globalActive >= globalCapacity {
		return nil, tx.Commit(ctx)
	}
	job, err := scanJob(tx.QueryRow(ctx, `SELECT j.id,j.clip_id,j.step,j.input_version,j.status,j.priority,j.attempts,j.max_attempts,j.lease_owner,j.lease_expires_at,j.next_attempt_at,j.last_error,j.created_at,j.updated_at,j.result,j.cycle FROM jobs j WHERE j.status='pending' AND j.next_attempt_at<=now() AND EXISTS (SELECT 1 FROM unnest($1::text[]) AS capability WHERE CASE WHEN right(capability,2)='.*' THEN left(j.step,length(capability)-1)=left(capability,length(capability)-1) ELSE j.step=capability END) ORDER BY j.priority DESC,j.created_at,j.id FOR UPDATE OF j SKIP LOCKED LIMIT 1`, supportedSteps))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("select job for claim: %w", err)
	}
	leaseToken := "lt_" + auth.Token()
	job, err = scanJob(tx.QueryRow(ctx, `UPDATE jobs SET status='leased',lease_owner=$2,lease_token_hash=$3,completed_by_worker_id=NULL,completed_lease_token_hash=NULL,lease_expires_at=now()+interval '2 minutes',attempts=attempts+1,updated_at=now() WHERE id=$1 RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result,cycle`, job.ID, workerID, auth.Hash(leaseToken)))
	if err != nil {
		return nil, fmt.Errorf("lease claimed job: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,attempt,actor,actor_id) SELECT id,'attempt_started',cycle,attempts,'worker',$2 FROM jobs WHERE id=$1`, job.ID, workerID); err != nil {
		return nil, fmt.Errorf("record job attempt: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit job claim: %w", err)
	}
	return &Assignment{Job: job, LeaseToken: leaseToken}, nil
}

func (s *Service) Input(ctx context.Context, id int64, workerID, leaseToken string) (Input, error) {
	if !validLeaseToken(leaseToken) {
		return Input{}, ErrLeaseNotOwned
	}
	var input Input
	err := s.pool.QueryRow(ctx, `SELECT j.id,j.clip_id,j.step,j.input_version,j.status,j.priority,j.attempts,j.max_attempts,j.lease_owner,j.lease_expires_at,j.next_attempt_at,j.last_error,j.created_at,j.updated_at,j.result,c.original_name,c.size_bytes,c.duration_seconds,c.video_codec,c.audio_codec,c.tracks FROM jobs j JOIN clips c ON c.id=j.clip_id WHERE j.id=$1 AND j.status='leased' AND j.lease_owner=$2 AND j.lease_token_hash=$3 AND j.lease_expires_at>now()`, id, workerID, auth.Hash(leaseToken)).Scan(&input.Job.ID, &input.Job.ClipID, &input.Job.Step, &input.Job.InputVersion, &input.Job.Status, &input.Job.Priority, &input.Job.Attempts, &input.Job.MaxAttempts, &input.Job.LeaseOwner, &input.Job.LeaseExpiresAt, &input.Job.NextAttemptAt, &input.Job.LastError, &input.Job.CreatedAt, &input.Job.UpdatedAt, &input.Job.Result, &input.OriginalName, &input.SizeBytes, &input.DurationSeconds, &input.VideoCodec, &input.AudioCodec, &input.Tracks)
	if errors.Is(err, pgx.ErrNoRows) {
		return Input{}, ErrLeaseNotOwned
	}
	if err != nil {
		return Input{}, fmt.Errorf("read worker job input: %w", err)
	}
	input.MediaURL = fmt.Sprintf("/api/worker/jobs/%d/media", id)
	input.Vocabulary = []VocabularyTerm{}
	input.Pipeline = json.RawMessage(`{"provider":"openai-compatible","model":"default","language":"fr","preparation":"wav-16khz-mono-v1","vad":{"mode":"off","version":"vad-off-v1"}}`)
	if strings.HasPrefix(input.Job.Step, "transcription.audio.") {
		vocabMarker := strings.Index(input.Job.InputVersion, "-v")
		if vocabMarker >= 0 {
			vocabVersion := input.Job.InputVersion[vocabMarker+len("-v"):]
			marker := strings.Index(vocabVersion, "-p")
			if marker < 0 {
				return Input{}, fmt.Errorf("transcription input has no pipeline separator")
			}
			vocabVersion = vocabVersion[:marker]
			var rawVocabulary []byte
			if queryErr := s.pool.QueryRow(ctx, `SELECT terms FROM transcription_vocabulary_versions WHERE version=$1`, vocabVersion).Scan(&rawVocabulary); queryErr != nil {
				return Input{}, fmt.Errorf("read transcription vocabulary snapshot: %w", queryErr)
			}
			if queryErr := json.Unmarshal(rawVocabulary, &input.Vocabulary); queryErr != nil {
				return Input{}, fmt.Errorf("decode transcription vocabulary snapshot: %w", queryErr)
			}
		}
		pipelineMarker := strings.LastIndex(input.Job.InputVersion, "-p")
		if pipelineMarker < 0 {
			return Input{}, fmt.Errorf("transcription input has no pipeline version")
		}
		pipelineVersion := input.Job.InputVersion[pipelineMarker+len("-p"):]
		input.PipelineVersion = pipelineVersion
		if queryErr := s.pool.QueryRow(ctx, `SELECT config FROM transcription_pipeline_versions WHERE version=$1`, pipelineVersion).Scan(&input.Pipeline); queryErr != nil {
			return Input{}, fmt.Errorf("read transcription pipeline snapshot: %w", queryErr)
		}
	}
	if strings.HasPrefix(input.Job.Step, "voice.") {
		if err = s.voiceInput(ctx, &input); err != nil {
			return Input{}, err
		}
	}
	return input, nil
}

func (s *Service) Heartbeat(ctx context.Context, id int64, workerID, leaseToken string) error {
	if !validLeaseToken(leaseToken) {
		return ErrLeaseNotOwned
	}
	result, err := s.pool.Exec(ctx, `UPDATE jobs SET lease_expires_at=now()+interval '2 minutes',updated_at=now() WHERE id=$1 AND status='leased' AND lease_owner=$2 AND lease_token_hash=$3 AND lease_expires_at>now()`, id, workerID, auth.Hash(leaseToken))
	if err != nil {
		return fmt.Errorf("renew job lease: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLeaseNotOwned
	}
	return nil
}

func (s *Service) Complete(ctx context.Context, id int64, workerID, leaseToken, inputVersion string, result json.RawMessage) (Job, error) {
	return s.CompleteWith(ctx, id, workerID, leaseToken, inputVersion, result, CompleteOptions{})
}

// CompleteOptions extends completion for steps that project their result into
// dedicated tables. Project runs inside the completion transaction, only for the
// first successful completion; any error rolls the whole completion back.
type CompleteOptions struct {
	MaxResultBytes int
	// Summarize returns what jobs.result keeps instead of the raw result, for
	// results that must not be stored as received (voice embeddings).
	Summarize func(result json.RawMessage) json.RawMessage
	Project   func(ctx context.Context, tx pgx.Tx, job Job) error
}

func (s *Service) CompleteWith(ctx context.Context, id int64, workerID, leaseToken, inputVersion string, result json.RawMessage, opts CompleteOptions) (Job, error) {
	if !validLeaseToken(leaseToken) {
		return Job{}, ErrLeaseNotOwned
	}
	if opts.MaxResultBytes == 0 {
		opts.MaxResultBytes = 12 * 1024
	}
	if !json.Valid(result) || len(result) == 0 || len(result) > opts.MaxResultBytes {
		return Job{}, ErrInvalidInput
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(result, &object); err != nil || object == nil {
		return Job{}, ErrInvalidInput
	}
	// The digest of the result as received identifies an identical retry, even when
	// only a summary is stored.
	sum := sha256.Sum256(result)
	digest := hex.EncodeToString(sum[:])
	stored := []byte(result)
	if opts.Summarize != nil {
		stored = opts.Summarize(result)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Job{}, fmt.Errorf("begin job completion: %w", err)
	}
	defer tx.Rollback(ctx)
	job, err := scanJob(tx.QueryRow(ctx, `UPDATE jobs SET status='done',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,completed_by_worker_id=$2,completed_lease_token_hash=$3,result=$5::jsonb,result_digest=$6,last_error='',updated_at=now() WHERE id=$1 AND lease_owner=$2 AND lease_token_hash=$3 AND status='leased' AND lease_expires_at>now() AND input_version=$4 RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result,cycle`, id, workerID, auth.Hash(leaseToken), inputVersion, stored, digest))
	newlyCompleted := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		// A worker may lose the successful response and retry the same result.
		// Return the committed row only to the worker that completed it, and only
		// when the step version and JSON result are identical.
		job, err = scanJob(tx.QueryRow(ctx, `SELECT id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result,cycle FROM jobs WHERE id=$1 AND status='done' AND completed_by_worker_id=$2 AND completed_lease_token_hash=$3 AND input_version=$4 AND (result_digest=$6 OR (result_digest IS NULL AND result=$5::jsonb))`, id, workerID, auth.Hash(leaseToken), inputVersion, stored, digest))
		if errors.Is(err, pgx.ErrNoRows) {
			return Job{}, ErrLeaseNotOwned
		}
	}
	if err != nil {
		return Job{}, fmt.Errorf("complete worker job: %w", err)
	}
	if newlyCompleted {
		if strings.HasPrefix(job.Step, "transcription.audio.") {
			var embeddingStateID int
			if err = tx.QueryRow(ctx, `SELECT id FROM embedding_index_state WHERE id=1 FOR SHARE`).Scan(&embeddingStateID); err != nil {
				return Job{}, fmt.Errorf("lock embedding profile state: %w", err)
			}
			if _, err = tx.Exec(ctx, `INSERT INTO embedding_tasks(profile_id,transcription_job_id) SELECT p.id,$1 FROM embedding_profiles p JOIN embedding_index_state s ON s.id=1 WHERE p.id=s.active_profile_id OR p.id=s.candidate_profile_id ON CONFLICT(profile_id,transcription_job_id) DO NOTHING`, id); err != nil {
				return Job{}, fmt.Errorf("queue transcript embeddings: %w", err)
			}
		}
		if opts.Project != nil {
			if err = opts.Project(ctx, tx, job); err != nil {
				return Job{}, err
			}
		}
		if _, err = tx.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,attempt,actor,actor_id) VALUES($1,'completed',$2,$3,'worker',$4)`, id, job.Cycle, job.Attempts, workerID); err != nil {
			return Job{}, fmt.Errorf("record completed job: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Job{}, fmt.Errorf("commit job completion: %w", err)
	}
	return job, nil
}

func (s *Service) Fail(ctx context.Context, id int64, workerID, leaseToken, message string) (Job, error) {
	if !validLeaseToken(leaseToken) {
		return Job{}, ErrLeaseNotOwned
	}
	if len(message) > 256 || strings.TrimSpace(message) != message {
		return Job{}, ErrInvalidInput
	}
	// Never persist worker-provided stack traces, paths, or arbitrary details.
	code := "worker_failed"
	if message == "provider_unavailable" || message == "invalid_input" {
		code = message
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Job{}, fmt.Errorf("begin worker job failure: %w", err)
	}
	defer tx.Rollback(ctx)
	job, err := scanJob(tx.QueryRow(ctx, `SELECT id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result,cycle FROM jobs WHERE id=$1 AND lease_owner=$2 AND lease_token_hash=$3 AND status='leased' AND lease_expires_at>now() FOR UPDATE`, id, workerID, auth.Hash(leaseToken)))
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrLeaseNotOwned
	}
	if err != nil {
		return Job{}, fmt.Errorf("read worker job for failure: %w", err)
	}
	status := RetryWait
	delay := retryDelay(job.Attempts)
	if code == "invalid_input" || job.Attempts >= job.MaxAttempts {
		status = Dead
	}
	job, err = scanJob(tx.QueryRow(ctx, `UPDATE jobs SET status=$4,lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,last_error=$5,next_attempt_at=CASE WHEN $4='retry_wait' THEN now()+$6::interval ELSE next_attempt_at END,updated_at=now() WHERE id=$1 AND lease_owner=$2 AND lease_token_hash=$3 AND status='leased' RETURNING id,clip_id,step,input_version,status,priority,attempts,max_attempts,lease_owner,lease_expires_at,next_attempt_at,last_error,created_at,updated_at,result,cycle`, id, workerID, auth.Hash(leaseToken), status, code, delay.String()))
	if err != nil {
		return Job{}, fmt.Errorf("update failed worker job: %w", err)
	}
	event := "retry_scheduled"
	if status == Dead {
		event = "marked_dead"
	}
	if _, err = tx.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,attempt,actor,actor_id,error_code) VALUES($1,$2,$3,$4,'worker',$5,$6)`, id, event, job.Cycle, job.Attempts, workerID, code); err != nil {
		return Job{}, fmt.Errorf("record worker job failure: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Job{}, fmt.Errorf("commit worker job failure: %w", err)
	}
	return job, nil
}

func retryDelay(attempt int) time.Duration {
	delay := 2 * time.Second
	for i := 1; i < attempt && delay < 15*time.Minute; i++ {
		delay *= 2
	}
	if delay > 15*time.Minute {
		return 15 * time.Minute
	}
	return delay
}

var ErrLeaseNotOwned = errors.New("worker does not own active job lease")

func validLeaseToken(token string) bool {
	return strings.HasPrefix(token, "lt_") && len(token) >= 32 && len(token) <= 128 && !strings.ContainsAny(token, " \t\r\n")
}

type scanner interface{ Scan(...any) error }

func scanJob(row scanner) (Job, error) {
	var job Job
	err := row.Scan(&job.ID, &job.ClipID, &job.Step, &job.InputVersion, &job.Status, &job.Priority, &job.Attempts, &job.MaxAttempts, &job.LeaseOwner, &job.LeaseExpiresAt, &job.NextAttemptAt, &job.LastError, &job.CreatedAt, &job.UpdatedAt, &job.Result, &job.Cycle)
	return job, err
}
