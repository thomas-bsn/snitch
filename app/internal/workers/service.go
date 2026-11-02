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
	ErrNotRevoked         = errors.New("worker is not revoked")
	ErrCleanupPending     = errors.New("worker local cleanup is not confirmed")
	namePattern           = regexp.MustCompile(`^[\pL\pN][\pL\pN _.-]{0,63}$`)
	stepCapabilityPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}(\.\*)?$`)
)

// ConnectedWindow is six five-second worker poll intervals.
const ConnectedWindow = 30 * time.Second

type WorkerTranscriptionConfig struct {
	ReceivedVersion  string `json:"received_version"`
	TestedVersion    string `json:"tested_version"`
	ActiveVersion    string `json:"active_version"`
	ErrorCode        string `json:"error_code"`
	EndpointOverride string `json:"endpoint_override"`
}

type Worker struct {
	SupportedSteps           []string                   `json:"supported_steps"`
	ID                       string                     `json:"id"`
	Name                     string                     `json:"name"`
	MaxConcurrency           int                        `json:"max_concurrency"`
	ReportedConcurrency      int                        `json:"reported_concurrency"`
	LastSeenAt               *time.Time                 `json:"last_seen_at"`
	CreatedAt                time.Time                  `json:"created_at"`
	RevokedAt                *time.Time                 `json:"revoked_at"`
	RevokeReason             string                     `json:"revoke_reason"`
	RevocationAcknowledgedAt *time.Time                 `json:"revocation_acknowledged_at"`
	Connected                bool                       `json:"connected"`
	Models                   Models                     `json:"models"`
	TranscriptionModel       string                     `json:"transcription_model"`
	TranscriptionConfig      *WorkerTranscriptionConfig `json:"transcription_config,omitempty"`
	TranscriptionConfigReady bool                       `json:"transcription_config_ready"`
	Machine                  *Machine                   `json:"machine,omitempty"`
	RemoteAddress            string                     `json:"remote_address,omitempty"`
	Install                  *InstallProgress           `json:"install,omitempty"`
}

type Registration struct {
	Worker
	Token          string `json:"token"`
	InstallCommand string `json:"install_command,omitempty"`
	// InstallCommandUnavailable says why no command was generated (insecure_origin, invalid_origin).
	InstallCommandUnavailable string `json:"install_command_unavailable,omitempty"`
	Bundle                    Bundle `json:"bundle"`
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
	var receivedVersion, testedVersion, activeVersion, errorCode, endpointOverride string
	var supportedSteps []string
	err := s.pool.QueryRow(ctx, `SELECT w.id,w.name,w.max_concurrency,w.reported_concurrency,w.last_seen_at,w.created_at,w.revoked_at,w.revoke_reason,w.revocation_acknowledged_at,COALESCE(w.revoked_at IS NULL AND w.last_seen_at >= now() - ($2 * interval '1 second'),false),w.models,w.transcription_model,w.supported_steps,w.machine,w.remote_address,COALESCE(wtc.received_version,''),COALESCE(wtc.tested_version,''),COALESCE(wtc.active_version,''),COALESCE(wtc.error_code,''),COALESCE(wtc.endpoint_override,'') FROM workers w LEFT JOIN worker_transcription_config wtc ON wtc.worker_id=w.id WHERE w.id=$1`, id, int64(ConnectedWindow.Seconds())).Scan(&d.ID, &d.Name, &d.MaxConcurrency, &d.ReportedConcurrency, &d.LastSeenAt, &d.CreatedAt, &d.RevokedAt, &d.RevokeReason, &d.RevocationAcknowledgedAt, &d.Connected, &d.Models, &d.TranscriptionModel, &supportedSteps, &d.Machine, &d.RemoteAddress, &receivedVersion, &testedVersion, &activeVersion, &errorCode, &endpointOverride)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("read worker: %w", err)
	}
	if receivedVersion != "" || testedVersion != "" || activeVersion != "" || errorCode != "" || endpointOverride != "" {
		d.TranscriptionConfig = &WorkerTranscriptionConfig{
			ReceivedVersion:  receivedVersion,
			TestedVersion:    testedVersion,
			ActiveVersion:    activeVersion,
			ErrorCode:        errorCode,
			EndpointOverride: endpointOverride,
		}
	}
	// Check if worker supports centralized transcription config
	for _, step := range supportedSteps {
		if step == "transcription.config.v1" {
			d.TranscriptionConfigReady = true
			break
		}
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

// RenewReason is the audit reason kept on an identity replaced after a lost token.
const RenewReason = "token_renewed"

// RenewToken answers a lost token: the token is unrecoverable, so the active identity is revoked
// (running tasks return to the queue, unconsumed invitations are deleted, minimal audit kept) and
// a new identity with the same name and concurrency is created, in one transaction. The caller
// then issues a new single-use invitation for the new identity.
func (s *Service) RenewToken(ctx context.Context, id string) (Registration, error) {
	if !validID(id) {
		return Registration{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Registration{}, fmt.Errorf("begin worker renewal: %w", err)
	}
	defer tx.Rollback(ctx)
	var name string
	var maxConcurrency int
	var revoked bool
	err = tx.QueryRow(ctx, `SELECT name,max_concurrency,revoked_at IS NOT NULL FROM workers WHERE id=$1 FOR UPDATE`, id).Scan(&name, &maxConcurrency, &revoked)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && revoked) {
		return Registration{}, ErrNotFound
	}
	if err != nil {
		return Registration{}, fmt.Errorf("lock worker for renewal: %w", err)
	}
	if err = revokeTx(ctx, tx, id, RenewReason); err != nil {
		return Registration{}, err
	}
	newID := "wk_" + auth.Token()[:28]
	token := "rw_" + auth.Token()
	var worker Worker
	err = tx.QueryRow(ctx, `INSERT INTO workers(id,name,token_hash,max_concurrency) VALUES($1,$2,$3,$4) RETURNING id,name,max_concurrency,reported_concurrency,last_seen_at,created_at,revoked_at`, newID, name, auth.Hash(token), maxConcurrency).Scan(&worker.ID, &worker.Name, &worker.MaxConcurrency, &worker.ReportedConcurrency, &worker.LastSeenAt, &worker.CreatedAt, &worker.RevokedAt)
	if err != nil {
		return Registration{}, fmt.Errorf("create replacement worker: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Registration{}, fmt.Errorf("commit worker renewal: %w", err)
	}
	return Registration{Worker: worker, Token: token}, nil
}

func (s *Service) List(ctx context.Context) ([]Worker, error) {
	rows, err := s.pool.Query(ctx, `SELECT w.id,w.name,w.max_concurrency,w.reported_concurrency,w.last_seen_at,w.created_at,w.revoked_at,w.revoke_reason,w.revocation_acknowledged_at,COALESCE(w.revoked_at IS NULL AND w.last_seen_at >= now() - ($1 * interval '1 second'),false),w.models,w.transcription_model,w.supported_steps,COALESCE(wtc.received_version,''),COALESCE(wtc.tested_version,''),COALESCE(wtc.active_version,''),COALESCE(wtc.error_code,''),COALESCE(wtc.endpoint_override,'') FROM workers w LEFT JOIN worker_transcription_config wtc ON wtc.worker_id=w.id ORDER BY w.created_at,w.id`, int64(ConnectedWindow.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	defer rows.Close()
	items := []Worker{}
	for rows.Next() {
		var worker Worker
		var receivedVersion, testedVersion, activeVersion, errorCode, endpointOverride string
		if err = rows.Scan(&worker.ID, &worker.Name, &worker.MaxConcurrency, &worker.ReportedConcurrency, &worker.LastSeenAt, &worker.CreatedAt, &worker.RevokedAt, &worker.RevokeReason, &worker.RevocationAcknowledgedAt, &worker.Connected, &worker.Models, &worker.TranscriptionModel, &worker.SupportedSteps, &receivedVersion, &testedVersion, &activeVersion, &errorCode, &endpointOverride); err != nil {
			return nil, fmt.Errorf("scan worker: %w", err)
		}
		if receivedVersion != "" || testedVersion != "" || activeVersion != "" || errorCode != "" || endpointOverride != "" {
			worker.TranscriptionConfig = &WorkerTranscriptionConfig{
				ReceivedVersion:  receivedVersion,
				TestedVersion:    testedVersion,
				ActiveVersion:    activeVersion,
				ErrorCode:        errorCode,
				EndpointOverride: endpointOverride,
			}
		}
		// Check if worker supports centralized transcription config
		for _, step := range worker.SupportedSteps {
			if step == "transcription.config.v1" {
				worker.TranscriptionConfigReady = true
				break
			}
		}
		items = append(items, worker)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read workers: %w", err)
	}
	return items, nil
}

// MaxRevokeReason bounds the optional reason kept in the deletion audit.
const MaxRevokeReason = 200

// Revoke removes a worker for good, in one transaction: the identity stops authenticating, its
// running tasks return to the queue at once, and everything but the minimal audit (name, date,
// optional reason) is deleted. Revoking twice keeps the first audit. A revoked identity is never
// reactivated: only an installation invitation creates a new one.
func (s *Service) Revoke(ctx context.Context, id, reason string) error {
	reason = strings.TrimSpace(reason)
	if !validID(id) {
		return ErrNotFound
	}
	if len([]rune(reason)) > MaxRevokeReason || strings.ContainsAny(reason, "\r\n\x00") {
		return ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin worker revocation: %w", err)
	}
	defer tx.Rollback(ctx)
	if err = revokeTx(ctx, tx, id, reason); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit worker revocation: %w", err)
	}
	return nil
}

// Forget deletes the row of a revoked Worker so it leaves the list. Only a revoked identity whose
// machine confirmed the revocation can go: until then the launcher still needs the row to learn
// it must clean up. Jobs and events keep their history; the Worker's name is no longer resolved.
func (s *Service) Forget(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin worker removal: %w", err)
	}
	defer tx.Rollback(ctx)
	var revoked, acknowledged bool
	if err = tx.QueryRow(ctx, `SELECT revoked_at IS NOT NULL,revocation_acknowledged_at IS NOT NULL FROM workers WHERE id=$1 FOR UPDATE`, id).Scan(&revoked, &acknowledged); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("lock worker for removal: %w", err)
	}
	if !revoked {
		return ErrNotRevoked
	}
	if !acknowledged {
		return ErrCleanupPending
	}
	if _, err = tx.Exec(ctx, `UPDATE jobs SET lease_owner=NULL WHERE lease_owner=$1`, id); err != nil {
		return fmt.Errorf("release worker references: %w", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM workers WHERE id=$1`, id); err != nil {
		return fmt.Errorf("delete worker: %w", err)
	}
	return tx.Commit(ctx)
}

// revokeTx applies the revocation inside the caller's transaction.
func revokeTx(ctx context.Context, tx pgx.Tx, id, reason string) error {
	var err error
	var already bool
	if err = tx.QueryRow(ctx, `SELECT revoked_at IS NOT NULL FROM workers WHERE id=$1 FOR UPDATE`, id).Scan(&already); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("lock worker for revocation: %w", err)
	}
	if already {
		return nil
	}
	if _, err = tx.Exec(ctx, `UPDATE workers SET revoked_at=now(),revoke_reason=$2,reported_concurrency=0,supported_steps='{}',models='{}',machine='{}',remote_address='' WHERE id=$1`, id, reason); err != nil {
		return fmt.Errorf("revoke worker: %w", err)
	}
	// A revocation is not the task's fault: the interrupted attempt is not counted.
	if _, err = tx.Exec(ctx, `WITH released AS (
		UPDATE jobs SET status='pending',attempts=GREATEST(attempts-1,0),lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,last_error='worker_revoked',updated_at=now()
		 WHERE status='leased' AND lease_owner=$1
		 RETURNING id,cycle
	) INSERT INTO job_events(job_id,event,cycle,actor,actor_id,error_code) SELECT id,'worker_revoked',cycle,'system',$1,'worker_revoked' FROM released`, id); err != nil {
		return fmt.Errorf("requeue revoked worker tasks: %w", err)
	}
	for _, statement := range []string{
		`DELETE FROM worker_install_events WHERE worker_id=$1`,
		`DELETE FROM worker_install_invites WHERE worker_id=$1`,
		`DELETE FROM worker_transcription_config WHERE worker_id=$1`,
	} {
		if _, err = tx.Exec(ctx, statement, id); err != nil {
			return fmt.Errorf("delete revoked worker details: %w", err)
		}
	}
	return nil
}

// RevocationStatus answers the local launcher. It reveals only whether this token belongs to a
// revoked identity and when; an unknown token is unauthorized. A revoked answer is recorded as
// the launcher's acknowledgement (it then stops the service), never as a local erasure.
func (s *Service) RevocationStatus(ctx context.Context, token string) (bool, time.Time, error) {
	if !strings.HasPrefix(token, "rw_") || len(token) > 128 {
		return false, time.Time{}, ErrUnauthorized
	}
	var revokedAt *time.Time
	err := s.pool.QueryRow(ctx, `UPDATE workers SET revocation_acknowledged_at=CASE WHEN revoked_at IS NOT NULL THEN COALESCE(revocation_acknowledged_at,now()) END WHERE token_hash=$1 RETURNING revoked_at`, auth.Hash(token)).Scan(&revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, time.Time{}, ErrUnauthorized
	}
	if err != nil {
		return false, time.Time{}, fmt.Errorf("read worker revocation: %w", err)
	}
	if revokedAt == nil {
		return false, time.Time{}, nil
	}
	return true, *revokedAt, nil
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
	result, err := s.pool.Exec(ctx, `UPDATE workers SET last_seen_at=now(),first_pulse_at=COALESCE(first_pulse_at,now()),reported_concurrency=$2,supported_steps=$3,models=$4,machine=$5,remote_address=$6 WHERE id=$1 AND revoked_at IS NULL AND max_concurrency >= $2`, id, capacity, steps, models, machine, remoteAddress)
	if err != nil {
		return fmt.Errorf("update worker pulse: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUnauthorized
	}
	return nil
}

func validID(id string) bool { return len(id) >= 23 && len(id) <= 63 && strings.HasPrefix(id, "wk_") }
