// Package analysis keeps the durable analysis runs of clips. A run lists the steps of a
// scope; each step holds a derived status and a stable reason code. Jobs stay the unit of
// work (leases, retries, delays): a step only follows its job. The run advances whenever a
// job changes and on a ticker, so it survives a restart.
package analysis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/jobs"
)

// Scopes: what the administrator asks for. transcription stops after the transcription;
// voice runs the voice detection on a valid transcription (it never transcribes); full is
// the transcription, its indexing and the voices.
const (
	ScopeTranscription = "transcription"
	ScopeVoice         = "voice"
	ScopeFull          = "full"
)

// Steps of a run, in dependency order. Every run lists all of them; the ones outside its
// scope are not applicable.
const (
	StepPreparation   = "preparation"
	StepTranscription = "transcription"
	StepIndexing      = "indexing"
	StepDiarization   = "diarization"
	StepFingerprints  = "fingerprints"
	StepMatching      = "matching"
)

var stepOrder = []string{StepPreparation, StepTranscription, StepIndexing, StepDiarization, StepFingerprints, StepMatching}

// laterSteps are the steps that follow the transcription. Their orchestration is not wired
// yet (indexing, voices): they show not_yet_supported rather than a false success.
var laterSteps = []string{StepIndexing, StepDiarization, StepFingerprints, StepMatching}

func applicable(scope, step string) bool {
	switch step {
	case StepPreparation:
		return true
	case StepTranscription:
		return true
	case StepIndexing:
		return scope == ScopeFull
	default:
		return scope == ScopeVoice || scope == ScopeFull
	}
}

// Step statuses. Reasons are stable codes the interface translates.
const (
	StatusWaiting         = "waiting"
	StatusRunning         = "running"
	StatusRetry           = "retry"
	StatusBlocked         = "blocked"
	StatusFailed          = "failed"
	StatusDone            = "done"
	StatusNotApplicable   = "not_applicable"
	StatusNotYetSupported = "not_yet_supported"
)

// Run statuses.
const (
	RunActive    = "active"
	RunCompleted = "completed"
	// RunIncomplete: every step that can run is done, the others are not yet supported.
	RunIncomplete = "incomplete"
	RunFailed     = "failed"
	RunCancelled  = "cancelled"
)

var (
	ErrClipNotFound          = errors.New("clip not found")
	ErrInvalidScope          = errors.New("invalid analysis scope")
	ErrScopeConflict         = errors.New("another analysis scope is active")
	ErrTranscriptionRequired = errors.New("a valid transcription is required")
)

type Step struct {
	Step       string     `json:"step"`
	Status     string     `json:"status"`
	Reason     string     `json:"reason"`
	JobID      *int64     `json:"job_id"`
	Attempt    int        `json:"attempt"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Run struct {
	ID         int64      `json:"id"`
	ClipID     int64      `json:"clip_id"`
	Scope      string     `json:"scope"`
	Status     string     `json:"status"`
	StartedBy  *int64     `json:"started_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Steps      []Step     `json:"steps"`
}

// Preparation is what the clip lets a run do: the media must be ready and one audio track
// resolved. Status is waiting, blocked or done; Reason is a stable code.
type Preparation struct{ Status, Reason string }

// Transcription is the job a run follows for the transcription step.
type Transcription struct {
	Job jobs.Job
	// Kept: a transcription made by an older input version stays visible because human
	// corrections would be replaced by a new one.
	Kept bool
}

// Hooks are the clip-specific decisions the service does not own.
type Hooks interface {
	Prepare(ctx context.Context, clipID int64) (Preparation, error)
	// Transcribe finds the job to follow, or schedules it. Jobs created before since that
	// ended without success are ignored (a new run starts afresh); valid results are reused.
	Transcribe(ctx context.Context, clipID int64, since time.Time) (Transcription, error)
	// HasTranscription tells whether the clip has a valid transcription to build on.
	HasTranscription(ctx context.Context, clipID int64) (bool, error)
}

type Service struct {
	pool  *pgxpool.Pool
	hooks Hooks
	log   *slog.Logger
}

func New(pool *pgxpool.Pool, hooks Hooks, log *slog.Logger) *Service {
	return &Service{pool: pool, hooks: hooks, log: log}
}

func validScope(scope string) bool {
	return scope == ScopeTranscription || scope == ScopeVoice || scope == ScopeFull
}

const runColumns = `id,clip_id,scope,status,started_by,created_at,updated_at,finished_at`

func scanRun(row pgx.Row) (Run, error) {
	var r Run
	err := row.Scan(&r.ID, &r.ClipID, &r.Scope, &r.Status, &r.StartedBy, &r.CreatedAt, &r.UpdatedAt, &r.FinishedAt)
	return r, err
}

// Start begins the analysis of a clip, or returns the one already active (created=false).
// An active run of another scope is refused with ErrScopeConflict (the active run is returned).
func (s *Service) Start(ctx context.Context, clipID int64, scope string, adminID int64) (Run, bool, error) {
	if !validScope(scope) {
		return Run{}, false, ErrInvalidScope
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Run{}, false, err
	}
	defer tx.Rollback(ctx)
	run, err := scanRun(tx.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,started_by) VALUES($1,$2,$3) ON CONFLICT (clip_id) WHERE status='active' DO NOTHING RETURNING `+runColumns, clipID, scope, adminID))
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		run, err = scanRun(tx.QueryRow(ctx, `SELECT `+runColumns+` FROM clip_analysis_runs WHERE clip_id=$1 AND status='active'`, clipID))
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return Run{}, false, ErrClipNotFound
		}
		return Run{}, false, err
	}
	if created && scope == ScopeVoice {
		has, hasErr := s.hooks.HasTranscription(ctx, clipID)
		if hasErr != nil {
			return Run{}, false, hasErr
		}
		if !has {
			return Run{}, false, ErrTranscriptionRequired
		}
	}
	if created {
		for _, step := range stepOrder {
			status, reason := StatusWaiting, "awaiting_previous_step"
			switch {
			case !applicable(scope, step):
				status, reason = StatusNotApplicable, "outside_scope"
			case step == StepPreparation:
				reason = "awaiting_preparation"
			}
			if _, err = tx.Exec(ctx, `INSERT INTO clip_analysis_steps(run_id,step,status,reason,finished_at) VALUES($1,$2,$3,$4,CASE WHEN $3='not_applicable' THEN now() END)`, run.ID, step, status, reason); err != nil {
				return Run{}, false, err
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Run{}, false, err
	}
	if created || run.Scope == scope {
		if err = s.advance(ctx, run); err != nil {
			s.log.Warn("advance new analysis run", "run", run.ID, "error", err)
		}
	}
	full, err := s.Latest(ctx, clipID)
	if err != nil || full == nil {
		return run, created, err
	}
	if !created && run.Scope != scope {
		return *full, false, ErrScopeConflict
	}
	return *full, created, nil
}

// Latest returns the active run of the clip, or else the most recent one; nil when the
// clip was never analyzed.
func (s *Service) Latest(ctx context.Context, clipID int64) (*Run, error) {
	run, err := scanRun(s.pool.QueryRow(ctx, `SELECT `+runColumns+` FROM clip_analysis_runs WHERE clip_id=$1 ORDER BY (status='active') DESC,id DESC LIMIT 1`, clipID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT step,status,reason,job_id,attempt,started_at,finished_at,updated_at FROM clip_analysis_steps WHERE run_id=$1 ORDER BY array_position($2::text[],step)`, run.ID, stepOrder)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	run.Steps = []Step{}
	for rows.Next() {
		var st Step
		if err = rows.Scan(&st.Step, &st.Status, &st.Reason, &st.JobID, &st.Attempt, &st.StartedAt, &st.FinishedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		run.Steps = append(run.Steps, st)
	}
	return &run, rows.Err()
}

// ReconcileClip advances the active run of a clip, if any.
func (s *Service) ReconcileClip(ctx context.Context, clipID int64) error {
	run, err := scanRun(s.pool.QueryRow(ctx, `SELECT `+runColumns+` FROM clip_analysis_runs WHERE clip_id=$1 AND status='active'`, clipID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.advance(ctx, run)
}

// Reconcile advances every active run: the startup and ticker path that makes runs survive
// a restart or a lost notification.
func (s *Service) Reconcile(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT `+runColumns+` FROM clip_analysis_runs WHERE status='active' ORDER BY id`)
	if err != nil {
		return err
	}
	var active []Run
	for rows.Next() {
		r, scanErr := scanRun(rows)
		if scanErr != nil {
			rows.Close()
			return scanErr
		}
		active = append(active, r)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	var firstErr error
	for _, r := range active {
		if err = s.advance(ctx, r); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("run %d: %w", r.ID, err)
		}
	}
	return firstErr
}

// Serve reconciles at once, then on every tick, until ctx ends.
func (s *Service) Serve(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if err := s.Reconcile(ctx); err != nil && ctx.Err() == nil {
			s.log.Warn("reconcile analysis runs", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type stepState struct {
	status, reason string
	jobID          *int64
	attempt        int
}

// advance derives every step of the run from the current facts and stores the result.
// It is idempotent, so concurrent callers converge on the same state.
func (s *Service) advance(ctx context.Context, run Run) error {
	prep, err := s.hooks.Prepare(ctx, run.ClipID)
	if err != nil {
		return err
	}
	states := map[string]stepState{StepPreparation: {status: prep.Status, reason: prep.Reason}}
	for _, step := range stepOrder[1:] {
		if !applicable(run.Scope, step) {
			states[step] = stepState{status: StatusNotApplicable, reason: "outside_scope"}
		}
	}
	switch {
	case run.Scope == ScopeVoice:
		has, hasErr := s.hooks.HasTranscription(ctx, run.ClipID)
		if hasErr != nil {
			return hasErr
		}
		if has {
			states[StepTranscription] = stepState{status: StatusDone, reason: "reused_existing"}
		} else {
			states[StepTranscription] = stepState{status: StatusBlocked, reason: "transcription_required"}
		}
	case prep.Status != StatusDone:
		states[StepTranscription] = stepState{status: StatusWaiting, reason: "awaiting_preparation"}
	default:
		tr, trErr := s.hooks.Transcribe(ctx, run.ClipID, run.CreatedAt)
		if trErr != nil {
			return trErr
		}
		states[StepTranscription] = transcriptionState(tr, run.CreatedAt)
	}
	for _, step := range laterSteps {
		if !applicable(run.Scope, step) {
			continue
		}
		if states[StepTranscription].status == StatusDone {
			states[step] = stepState{status: StatusNotYetSupported, reason: "orchestration_pending"}
		} else {
			states[step] = stepState{status: StatusWaiting, reason: "awaiting_transcription"}
		}
	}
	failed, finished, unsupported := false, true, false
	for step, st := range states {
		if _, err = s.pool.Exec(ctx, `UPDATE clip_analysis_steps SET status=$3,reason=$4,job_id=$5,attempt=$6,updated_at=now(),
 started_at=COALESCE(started_at,CASE WHEN $3 IN ('running','retry','done','failed') THEN now() END),
 finished_at=CASE WHEN $3 IN ('done','failed','not_applicable') THEN COALESCE(finished_at,now()) END
 WHERE run_id=$1 AND step=$2 AND (status,reason,job_id,attempt) IS DISTINCT FROM ($3::text,$4::text,$5::bigint,$6::integer)`, run.ID, step, st.status, st.reason, st.jobID, st.attempt); err != nil {
			return err
		}
		failed = failed || st.status == StatusFailed
		unsupported = unsupported || st.status == StatusNotYetSupported
		finished = finished && (st.status == StatusDone || st.status == StatusNotApplicable || st.status == StatusNotYetSupported)
	}
	switch {
	case failed:
		_, err = s.pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='failed',updated_at=now(),finished_at=now() WHERE id=$1 AND status='active'`, run.ID)
	case finished && unsupported:
		_, err = s.pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='incomplete',updated_at=now(),finished_at=now() WHERE id=$1 AND status='active'`, run.ID)
	case finished:
		_, err = s.pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='completed',updated_at=now(),finished_at=now() WHERE id=$1 AND status='active'`, run.ID)
	}
	return err
}

func transcriptionState(tr Transcription, runStart time.Time) stepState {
	id := tr.Job.ID
	st := stepState{jobID: &id, attempt: tr.Job.Attempts}
	switch tr.Job.Status {
	case jobs.Pending:
		st.status, st.reason = StatusWaiting, "awaiting_worker"
	case jobs.Leased:
		st.status, st.reason = StatusRunning, "running"
	case jobs.RetryWait:
		st.status, st.reason = StatusRetry, "retry_scheduled"
	case jobs.Done:
		st.status, st.reason = StatusDone, "completed"
		if tr.Kept {
			st.reason = "kept_for_corrections"
		} else if tr.Job.CreatedAt.Before(runStart) {
			st.reason = "reused_existing"
		}
	case jobs.Dead:
		st.status, st.reason = StatusFailed, "job_dead"
	case jobs.Cancelled:
		st.status, st.reason = StatusFailed, "job_cancelled"
	default:
		st.status, st.reason = StatusFailed, "job_failed"
	}
	return st
}
