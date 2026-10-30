package workers

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/auth"
)

var (
	ErrInvalidInput       = errors.New("invalid worker input")
	ErrNotFound           = errors.New("worker not found")
	ErrUnauthorized       = errors.New("worker unauthorized")
	namePattern           = regexp.MustCompile(`^[\pL\pN][\pL\pN _.-]{0,63}$`)
	stepCapabilityPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}(\.\*)?$`)
)

// ConnectedWindow is six five-second worker poll intervals.
const ConnectedWindow = 30 * time.Second

type Worker struct {
	SupportedSteps      []string   `json:"supported_steps"`
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	MaxConcurrency      int        `json:"max_concurrency"`
	ReportedConcurrency int        `json:"reported_concurrency"`
	LastSeenAt          *time.Time `json:"last_seen_at"`
	CreatedAt           time.Time  `json:"created_at"`
	RevokedAt           *time.Time `json:"revoked_at"`
	Connected           bool       `json:"connected"`
	Models              Models     `json:"models"`
	TranscriptionModel  string     `json:"transcription_model"`
	Machine             *Machine   `json:"machine,omitempty"`
	RemoteAddress       string     `json:"remote_address,omitempty"`
}

type Registration struct {
	Worker
	Token  string `json:"token"`
	Bundle Bundle `json:"bundle"`
}

type Input struct {
	Name           string
	MaxConcurrency int
}

type Service struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) Register(ctx context.Context, input Input) (Registration, error) {
	registration, _, err := s.RegisterWith(ctx, input, nil)
	return registration, err
}

// RegisterWith creates a worker in one transaction with `before`, which runs first and learns
// whether this is the first worker ever registered. An error from `before` aborts everything.
func (s *Service) RegisterWith(ctx context.Context, input Input, before func(ctx context.Context, tx pgx.Tx, first bool) error) (Registration, bool, error) {
	input.Name = strings.TrimSpace(input.Name)
	if !namePattern.MatchString(input.Name) || input.MaxConcurrency < 1 || input.MaxConcurrency > 4 {
		return Registration{}, false, ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Registration{}, false, fmt.Errorf("begin worker registration: %w", err)
	}
	defer tx.Rollback(ctx)
	// Serialize registrations so that exactly one of two simultaneous requests is the first.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(7310073)`); err != nil {
		return Registration{}, false, fmt.Errorf("lock worker registration: %w", err)
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM workers`).Scan(&count); err != nil {
		return Registration{}, false, fmt.Errorf("count workers: %w", err)
	}
	first := count == 0
	if before != nil {
		if err = before(ctx, tx, first); err != nil {
			return Registration{}, first, err
		}
	}
	id := "wk_" + auth.Token()[:28]
	token := "rw_" + auth.Token()
	var worker Worker
	err = tx.QueryRow(ctx, `INSERT INTO workers(id,name,token_hash,max_concurrency) VALUES($1,$2,$3,$4) RETURNING id,name,max_concurrency,reported_concurrency,last_seen_at,created_at,revoked_at`, id, input.Name, auth.Hash(token), input.MaxConcurrency).Scan(&worker.ID, &worker.Name, &worker.MaxConcurrency, &worker.ReportedConcurrency, &worker.LastSeenAt, &worker.CreatedAt, &worker.RevokedAt)
	if err != nil {
		return Registration{}, first, fmt.Errorf("register worker: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Registration{}, first, fmt.Errorf("commit worker registration: %w", err)
	}
	return Registration{Worker: worker, Token: token}, first, nil
}

// First reports whether no worker has ever been registered.
func (s *Service) First(ctx context.Context) (bool, error) {
	var any bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workers)`).Scan(&any); err != nil {
		return false, fmt.Errorf("check first worker: %w", err)
	}
	return !any, nil
}

// Task is one recent piece of work a worker did, typed by what it is.
type Task struct {
	JobID     int64     `json:"job_id"`
	ClipID    int64     `json:"clip_id"`
	Kind      string    `json:"kind"`
	Outcome   string    `json:"outcome"`
	ErrorCode string    `json:"error_code"`
	At        time.Time `json:"at"`
}

// Detail is one worker with its latest tasks.
type Detail struct {
	Worker
	RecentTasks []Task `json:"recent_tasks"`
}

// Get returns one worker (revoked ones included) with its ten most recent tasks.
func (s *Service) Get(ctx context.Context, id string) (Detail, error) {
	if !validID(id) {
		return Detail{}, ErrNotFound
	}
	var d Detail
	err := s.pool.QueryRow(ctx, `SELECT id,name,max_concurrency,reported_concurrency,last_seen_at,created_at,revoked_at,COALESCE(revoked_at IS NULL AND last_seen_at >= now() - ($2 * interval '1 second'),false),models,transcription_model,machine,remote_address FROM workers WHERE id=$1`, id, int64(ConnectedWindow.Seconds())).Scan(&d.ID, &d.Name, &d.MaxConcurrency, &d.ReportedConcurrency, &d.LastSeenAt, &d.CreatedAt, &d.RevokedAt, &d.Connected, &d.Models, &d.TranscriptionModel, &d.Machine, &d.RemoteAddress)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("read worker: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT job_id,clip_id,step,event,error_code,created_at FROM (
		SELECT DISTINCT ON (e.job_id) e.job_id,j.clip_id,j.step,e.event,e.error_code,e.created_at,e.id
		FROM job_events e JOIN jobs j ON j.id=e.job_id
		WHERE e.actor='worker' AND e.actor_id=$1 ORDER BY e.job_id,e.id DESC) latest ORDER BY id DESC LIMIT 10`, id)
	if err != nil {
		return Detail{}, fmt.Errorf("read worker tasks: %w", err)
	}
	defer rows.Close()
	d.RecentTasks = []Task{}
	for rows.Next() {
		var task Task
		var step, event string
		if err = rows.Scan(&task.JobID, &task.ClipID, &step, &event, &task.ErrorCode, &task.At); err != nil {
			return Detail{}, fmt.Errorf("scan worker task: %w", err)
		}
		task.Kind, task.Outcome = taskKind(step), taskOutcome(event)
		d.RecentTasks = append(d.RecentTasks, task)
	}
	if err = rows.Err(); err != nil {
		return Detail{}, fmt.Errorf("read worker task rows: %w", err)
	}
	return d, nil
}

func taskKind(step string) string {
	switch {
	case strings.HasPrefix(step, "transcription."):
		return "transcription"
	case strings.HasPrefix(step, "voice.diarization."):
		return "diarization"
	case strings.HasPrefix(step, "voice.fingerprint."):
		return "fingerprint"
	}
	return "other"
}

func taskOutcome(event string) string {
	switch event {
	case "completed":
		return "completed"
	case "retry_scheduled":
		return "retry"
	case "marked_dead":
		return "failed"
	}
	return "started"
}

// RenewToken replaces the token of an active worker; the old one stops working at once.
func (s *Service) RenewToken(ctx context.Context, id string) (Registration, error) {
	if !validID(id) {
		return Registration{}, ErrNotFound
	}
	token := "rw_" + auth.Token()
	var worker Worker
	err := s.pool.QueryRow(ctx, `UPDATE workers SET token_hash=$2 WHERE id=$1 AND revoked_at IS NULL RETURNING id,name,max_concurrency,reported_concurrency,last_seen_at,created_at,revoked_at`, id, auth.Hash(token)).Scan(&worker.ID, &worker.Name, &worker.MaxConcurrency, &worker.ReportedConcurrency, &worker.LastSeenAt, &worker.CreatedAt, &worker.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Registration{}, ErrNotFound
	}
	if err != nil {
		return Registration{}, fmt.Errorf("renew worker token: %w", err)
	}
	return Registration{Worker: worker, Token: token}, nil
}

func (s *Service) List(ctx context.Context) ([]Worker, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,name,max_concurrency,reported_concurrency,last_seen_at,created_at,revoked_at,COALESCE(revoked_at IS NULL AND last_seen_at >= now() - ($1 * interval '1 second'),false),models,transcription_model,supported_steps FROM workers ORDER BY created_at,id`, int64(ConnectedWindow.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	defer rows.Close()
	items := []Worker{}
	for rows.Next() {
		var worker Worker
		if err = rows.Scan(&worker.ID, &worker.Name, &worker.MaxConcurrency, &worker.ReportedConcurrency, &worker.LastSeenAt, &worker.CreatedAt, &worker.RevokedAt, &worker.Connected, &worker.Models, &worker.TranscriptionModel, &worker.SupportedSteps); err != nil {
			return nil, fmt.Errorf("scan worker: %w", err)
		}
		items = append(items, worker)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read workers: %w", err)
	}
	return items, nil
}

func (s *Service) Revoke(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	result, err := s.pool.Exec(ctx, `UPDATE workers SET revoked_at=COALESCE(revoked_at,now()) WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("revoke worker: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (Worker, error) {
	if !strings.HasPrefix(token, "rw_") || len(token) > 128 {
		return Worker{}, ErrUnauthorized
	}
	var worker Worker
	err := s.pool.QueryRow(ctx, `SELECT id,name,max_concurrency,reported_concurrency,last_seen_at,created_at,revoked_at FROM workers WHERE token_hash=$1 AND revoked_at IS NULL`, auth.Hash(token)).Scan(&worker.ID, &worker.Name, &worker.MaxConcurrency, &worker.ReportedConcurrency, &worker.LastSeenAt, &worker.CreatedAt, &worker.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Worker{}, ErrUnauthorized
	}
	if err != nil {
		return Worker{}, fmt.Errorf("authenticate worker: %w", err)
	}
	return worker, nil
}

func (s *Service) Touch(ctx context.Context, id string, capacity int) error {
	if capacity < 0 || capacity > 4 {
		return ErrInvalidInput
	}
	result, err := s.pool.Exec(ctx, `UPDATE workers SET last_seen_at=now(),reported_concurrency=$2 WHERE id=$1 AND revoked_at IS NULL AND max_concurrency >= $2`, id, capacity)
	if err != nil {
		return fmt.Errorf("update worker activity: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUnauthorized
	}
	return nil
}

// Pulse refreshes worker presence, available capacity, and the step families
// this running worker can claim, and the models it declares. Both are replaced
// on every pulse.
func (s *Service) Pulse(ctx context.Context, id string, capacity int, supportedSteps []string, models Models, machine Machine, remoteAddress string) error {
	if capacity < 1 || capacity > 4 || len(supportedSteps) > 64 || !models.Valid() || !machine.Valid() {
		return ErrInvalidInput
	}
	steps := make([]string, 0, len(supportedSteps))
	seen := make(map[string]struct{}, len(supportedSteps))
	for _, step := range supportedSteps {
		if !stepCapabilityPattern.MatchString(step) || (strings.HasSuffix(step, ".*") && strings.Count(step, "*") != 1) {
			return ErrInvalidInput
		}
		if _, exists := seen[step]; exists {
			continue
		}
		seen[step] = struct{}{}
		steps = append(steps, step)
	}
	slices.Sort(steps)
	result, err := s.pool.Exec(ctx, `UPDATE workers SET last_seen_at=now(),reported_concurrency=$2,supported_steps=$3,models=$4,machine=$5,remote_address=$6 WHERE id=$1 AND revoked_at IS NULL AND max_concurrency >= $2`, id, capacity, steps, models, machine, remoteAddress)
	if err != nil {
		return fmt.Errorf("update worker pulse: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUnauthorized
	}
	return nil
}

func validID(id string) bool { return len(id) >= 23 && len(id) <= 63 && strings.HasPrefix(id, "wk_") }
