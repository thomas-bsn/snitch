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
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/jobs"
	"snitch/internal/workers"
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

// Reanalysis modes: missing runs only what has no valid result (the default); full runs
// again every step that can replace a result in the scope; steps runs again the chosen ones.
const (
	ModeMissing = "missing"
	ModeFull    = "full"
	ModeSteps   = "steps"
)

// Options are the reanalysis choices of a start. Steps matters only with ModeSteps; the
// confirmations record the consequences the administrator accepted.
type Options struct {
	Mode                      string
	Steps                     []string
	ConfirmReplaceCorrections bool
	ConfirmReplaceValidations bool
	// ConfirmedStakeAt is when the human work at stake was read and shown; zero means now. The
	// replacement never makes obsolete what was created after it.
	ConfirmedStakeAt time.Time
}

// forceable are the steps a reanalysis can run again: the others follow them (the
// fingerprints and the matching come after a diarization).
var forceable = []string{StepTranscription, StepIndexing, StepDiarization}

// ForcedSteps resolves the steps a mode forces in a scope, in dependency order.
func ForcedSteps(scope string, opts Options) ([]string, error) {
	switch opts.Mode {
	case "", ModeMissing:
		return nil, nil
	case ModeFull:
		out := []string{}
		for _, step := range forceable {
			// Indexing follows a new transcription by itself; forcing it alone would be redundant.
			if applicable(scope, step) && step != StepIndexing {
				out = append(out, step)
			}
		}
		return out, nil
	case ModeSteps:
		out := []string{}
		for _, step := range forceable {
			if slices.Contains(opts.Steps, step) {
				if !applicable(scope, step) {
					return nil, ErrInvalidReanalysis
				}
				out = append(out, step)
			}
		}
		if len(out) == 0 || len(out) != len(opts.Steps) {
			return nil, ErrInvalidReanalysis
		}
		return out, nil
	}
	return nil, ErrInvalidReanalysis
}

var stepOrder = []string{StepPreparation, StepTranscription, StepIndexing, StepDiarization, StepFingerprints, StepMatching}

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
	StatusWaiting       = "waiting"
	StatusRunning       = "running"
	StatusRetry         = "retry"
	StatusBlocked       = "blocked"
	StatusFailed        = "failed"
	StatusDone          = "done"
	StatusNotApplicable = "not_applicable"
)

// Run statuses.
const (
	RunActive    = "active"
	RunCompleted = "completed"
	RunFailed    = "failed"
	RunCancelled = "cancelled"
)

var (
	ErrClipNotFound          = errors.New("clip not found")
	ErrInvalidScope          = errors.New("invalid analysis scope")
	ErrScopeConflict         = errors.New("another analysis scope is active")
	ErrTranscriptionRequired = errors.New("a valid transcription is required")
	ErrNoRun                 = errors.New("the clip has no analysis")
	ErrNotActive             = errors.New("no analysis is active")
	ErrNotRetryable          = errors.New("the analysis cannot be retried")
	ErrInvalidStep           = errors.New("invalid analysis step")
	ErrStepNotFailed         = errors.New("the step has not failed")
	ErrInvalidReanalysis     = errors.New("invalid reanalysis choice")
)

type Step struct {
	Step    string `json:"step"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	JobID   *int64 `json:"job_id"`
	Attempt int    `json:"attempt"`
	// InputVersion and Profile say which versions the step followed (history).
	InputVersion *string `json:"input_version"`
	Profile      *string `json:"profile"`
	Forced       bool    `json:"forced"`
	// FailureCode and RecommendedAction are set on the Task Queue only: the persisted cause of
	// the last failure of the followed job (a catalogue code, never raw text) and the action the
	// server advises; a Worker wait advises start_worker.
	FailureCode       string `json:"failure_code,omitempty"`
	RecommendedAction string `json:"recommended_action,omitempty"`
	// Events are the persisted events of the followed job, oldest first; the run history only.
	Events     []StepEvent `json:"events,omitempty"`
	StartedAt  *time.Time  `json:"started_at"`
	FinishedAt *time.Time  `json:"finished_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// StepEvent is one persisted job event as the run history shows it: the cause is a catalogue
// code and the Worker is named, never an error text or a path.
type StepEvent struct {
	jobs.Event
	WorkerName string `json:"worker_name,omitempty"`
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
	Mode       string     `json:"mode"`
	// ForcedSteps run again whatever valid result existed; the confirmations are what the
	// administrator accepted before the run started.
	ForcedSteps                 []string `json:"forced_steps"`
	ConfirmedReplaceCorrections bool     `json:"confirmed_replace_corrections"`
	ConfirmedReplaceValidations bool     `json:"confirmed_replace_validations"`
	// ConfirmedStakeAt is the instant up to which corrections and validations were confirmed
	// replaceable; nil for a run that forces nothing.
	ConfirmedStakeAt *time.Time `json:"confirmed_stake_at"`
	Steps            []Step     `json:"steps"`
	// PendingVoices counts the voices of the clip still unknown or ambiguous (the Verification
	// list): a completed run with some left is only partially identified. It is read live, not stored.
	PendingVoices int `json:"pending_voices"`
}

// PublicStep is a step as every user who can open the clip sees it: its state and the stable
// reason code, never the job, the Worker or an error.
type PublicStep struct {
	Step      string    `json:"step"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PublicRun is the read-only view of a run for users who are not administrators.
type PublicRun struct {
	ClipID        int64        `json:"clip_id"`
	Scope         string       `json:"scope"`
	Status        string       `json:"status"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	FinishedAt    *time.Time   `json:"finished_at"`
	PendingVoices int          `json:"pending_voices"`
	Steps         []PublicStep `json:"steps"`
}

// Public strips a run down to what a non-administrator may see.
func (r *Run) Public() *PublicRun {
	if r == nil {
		return nil
	}
	out := &PublicRun{ClipID: r.ClipID, Scope: r.Scope, Status: r.Status, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, FinishedAt: r.FinishedAt, PendingVoices: r.PendingVoices, Steps: []PublicStep{}}
	for _, st := range r.Steps {
		out.Steps = append(out.Steps, PublicStep{Step: st.Step, Status: st.Status, Reason: st.Reason, UpdatedAt: st.UpdatedAt})
	}
	return out
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
	// With force, only jobs created since are considered: the valid result is replaced.
	Transcribe(ctx context.Context, clipID int64, since time.Time, force bool) (Transcription, error)
	// HasTranscription tells whether the clip has a valid transcription to build on.
	HasTranscription(ctx context.Context, clipID int64) (bool, error)
	// Index tells where the semantic indexing of the visible transcription stands. The text
	// search never depends on it.
	Index(ctx context.Context, clipID int64) (StepResult, error)
	// Voice resolves the audio track, applies the microphone designation, and follows or
	// schedules the diarization and fingerprint jobs (valid results are reused).
	// With force, a diarization created before since is not reused: it is run again.
	Voice(ctx context.Context, clipID int64, since time.Time, force bool) (VoiceSteps, error)
	// Reindex asks for the semantic indexing of the visible transcription to be redone.
	Reindex(ctx context.Context, clipID int64) error
	// RetryIndex asks for the semantic indexing of the visible transcription to be run again.
	RetryIndex(ctx context.Context, clipID int64) error
	// Match matches the fingerprints of the clip with the existing references.
	Match(ctx context.Context, clipID int64) error
}

// StepResult is the state a hook reports for a step. Job is the job it follows, if any.
type StepResult struct {
	Status, Reason string
	Job            *jobs.Job
}

// VoiceSteps are the voice steps of a clip. Matching is set only when it needs no matching
// run (a directly attributed microphone track, nothing to match).
type VoiceSteps struct {
	Diarization, Fingerprints StepResult
	Matching                  *StepResult
}

const maxMatchingAttempts = 3

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

const runColumns = `id,clip_id,scope,status,started_by,created_at,updated_at,finished_at,mode,forced_steps,confirmed_replace_corrections,confirmed_replace_validations,confirmed_stake_at`

func scanRun(row pgx.Row) (Run, error) {
	var r Run
	err := row.Scan(&r.ID, &r.ClipID, &r.Scope, &r.Status, &r.StartedBy, &r.CreatedAt, &r.UpdatedAt, &r.FinishedAt, &r.Mode, &r.ForcedSteps, &r.ConfirmedReplaceCorrections, &r.ConfirmedReplaceValidations, &r.ConfirmedStakeAt)
	return r, err
}

// Start begins the analysis of a clip, or returns the one already active (created=false).
// An active run of another scope is refused with ErrScopeConflict (the active run is returned).
func (s *Service) Start(ctx context.Context, clipID int64, scope string, adminID int64) (Run, bool, error) {
	return s.StartWith(ctx, clipID, scope, adminID, Options{})
}

// StartWith is Start with the reanalysis choices: the forced steps run again whatever valid
// result exists, and the run keeps the confirmations it was given.
func (s *Service) StartWith(ctx context.Context, clipID int64, scope string, adminID int64, opts Options) (Run, bool, error) {
	if !validScope(scope) {
		return Run{}, false, ErrInvalidScope
	}
	forced, err := ForcedSteps(scope, opts)
	if err != nil {
		return Run{}, false, err
	}
	if opts.Mode == "" {
		opts.Mode = ModeMissing
	}
	if forced == nil {
		forced = []string{}
	}
	var stakeAt *time.Time // nil: the database clock decides (now), the same one that stamps the human work
	if !opts.ConfirmedStakeAt.IsZero() {
		stakeAt = &opts.ConfirmedStakeAt
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Run{}, false, err
	}
	defer tx.Rollback(ctx)
	run, err := scanRun(tx.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,started_by,mode,forced_steps,confirmed_replace_corrections,confirmed_replace_validations,confirmed_stake_at) VALUES($1,$2,$3,$4,$5,$6,$7,CASE WHEN cardinality($5::text[])>0 THEN LEAST(COALESCE($8::timestamptz,now()),now()) END) ON CONFLICT (clip_id) WHERE status='active' DO NOTHING RETURNING `+runColumns, clipID, scope, adminID, opts.Mode, forced, opts.ConfirmReplaceCorrections, opts.ConfirmReplaceValidations, stakeAt))
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
	if created && slices.Contains(forced, StepIndexing) && !slices.Contains(forced, StepTranscription) {
		if err = s.hooks.Reindex(ctx, clipID); err != nil {
			s.log.Warn("reindex for the analysis run", "run", run.ID, "error", err)
		}
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
	if err = s.loadSteps(ctx, &run); err != nil {
		return nil, err
	}
	counts, err := s.pendingVoices(ctx, []int64{clipID})
	if err != nil {
		return nil, err
	}
	run.PendingVoices = counts[clipID]
	return &run, nil
}

// loadSteps fills the steps of a run with the versions of the job each one followed.
func (s *Service) loadSteps(ctx context.Context, run *Run) error {
	rows, err := s.pool.Query(ctx, `SELECT st.step,st.status,st.reason,st.job_id,st.attempt,j.input_version,st.started_at,st.finished_at,st.updated_at FROM clip_analysis_steps st LEFT JOIN jobs j ON j.id=st.job_id WHERE st.run_id=$1 ORDER BY array_position($2::text[],st.step)`, run.ID, stepOrder)
	if err != nil {
		return err
	}
	defer rows.Close()
	run.Steps = []Step{}
	for rows.Next() {
		var st Step
		if err = rows.Scan(&st.Step, &st.Status, &st.Reason, &st.JobID, &st.Attempt, &st.InputVersion, &st.StartedAt, &st.FinishedAt, &st.UpdatedAt); err != nil {
			return err
		}
		st.Profile = profileOf(st.InputVersion)
		st.Forced = slices.Contains(run.ForcedSteps, st.Step)
		run.Steps = append(run.Steps, st)
	}
	return rows.Err()
}

// profileOf extracts the pipeline profile from the input version of a job
// ("t4-a2-r1-v<vocabulary>-p<pipeline>", "d5-a2-p<pipeline>", "f5-a2-p<pipeline>-d12").
func profileOf(version *string) *string {
	if version == nil {
		return nil
	}
	if strings.HasPrefix(*version, "t4-") {
		if i := strings.LastIndex(*version, "-p"); i >= 0 {
			pipeline := (*version)[i+2:]
			return &pipeline
		}
		return nil
	}
	if pipeline, _, ok := jobs.VoiceInputVersion(*version); ok {
		return &pipeline
	}
	return nil
}

// History lists the most recent runs of a clip with their steps, versions and the
// confirmations they were started with.
func (s *Service) History(ctx context.Context, clipID int64, limit int) ([]Run, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+runColumns+` FROM clip_analysis_runs WHERE clip_id=$1 ORDER BY id DESC LIMIT $2`, clipID, limit)
	if err != nil {
		return nil, err
	}
	runs := []Run{}
	for rows.Next() {
		r, scanErr := scanRun(rows)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		runs = append(runs, r)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	counts, err := s.pendingVoices(ctx, []int64{clipID})
	if err != nil {
		return nil, err
	}
	for i := range runs {
		if err = s.loadSteps(ctx, &runs[i]); err != nil {
			return nil, err
		}
		runs[i].PendingVoices = counts[clipID]
	}
	return runs, nil
}

// ReconcileClip advances the active run of a clip, if any.
func (s *Service) ReconcileClip(ctx context.Context, clipID int64) error {
	if err := s.settleCancelled(ctx, clipID); err != nil {
		return err
	}
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
	if err := s.settleCancelled(ctx, 0); err != nil {
		return err
	}
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
	// A manual retry makes the hooks ignore the failed jobs of that step only.
	since := map[string]time.Time{}
	retryRows, err := s.pool.Query(ctx, `SELECT step,retry_since FROM clip_analysis_steps WHERE run_id=$1 AND retry_since IS NOT NULL`, run.ID)
	if err != nil {
		return err
	}
	for retryRows.Next() {
		var step string
		var at time.Time
		if err = retryRows.Scan(&step, &at); err != nil {
			retryRows.Close()
			return err
		}
		since[step] = at
	}
	retryRows.Close()
	if err = retryRows.Err(); err != nil {
		return err
	}
	sinceOf := func(steps ...string) time.Time {
		out := run.CreatedAt
		for _, st := range steps {
			if at, ok := since[st]; ok && at.After(out) {
				out = at
			}
		}
		return out
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
		tr, trErr := s.hooks.Transcribe(ctx, run.ClipID, sinceOf(StepTranscription), slices.Contains(run.ForcedSteps, StepTranscription))
		if trErr != nil {
			return trErr
		}
		states[StepTranscription] = transcriptionState(tr, sinceOf(StepTranscription))
	}
	// Indexing and voices are independent branches: each only needs the transcription.
	trState := states[StepTranscription]
	switch {
	case trState.status == StatusDone:
		if applicable(run.Scope, StepIndexing) {
			res, idxErr := s.hooks.Index(ctx, run.ClipID)
			if idxErr != nil {
				return idxErr
			}
			states[StepIndexing] = fromResult(res)
		}
		if applicable(run.Scope, StepDiarization) {
			if err = s.voiceStates(ctx, run, sinceOf(StepDiarization, StepFingerprints), slices.Contains(run.ForcedSteps, StepDiarization), states); err != nil {
				return err
			}
		}
	default:
		for _, step := range stepOrder[2:] {
			if !applicable(run.Scope, step) {
				continue
			}
			states[step] = dependentState(trState, "awaiting_transcription")
		}
	}
	for step, st := range states {
		if st.reason == "awaiting_worker" && st.jobID != nil {
			if st.reason, err = s.workerWaitReason(ctx, *st.jobID); err != nil {
				return err
			}
			states[step] = st
		}
	}
	failed, finished := false, true
	for step, st := range states {
		if _, err = s.pool.Exec(ctx, `UPDATE clip_analysis_steps SET status=$3,reason=$4,job_id=$5,attempt=$6,updated_at=now(),
 started_at=COALESCE(started_at,CASE WHEN $3 IN ('running','retry','done','failed') THEN now() END),
 finished_at=CASE WHEN $3 IN ('done','failed','not_applicable') THEN COALESCE(finished_at,now()) END
 WHERE run_id=$1 AND step=$2 AND (status,reason,job_id,attempt) IS DISTINCT FROM ($3::text,$4::text,$5::bigint,$6::integer)`, run.ID, step, st.status, st.reason, st.jobID, st.attempt); err != nil {
			return err
		}
		failed = failed || st.status == StatusFailed
		finished = finished && (st.status == StatusDone || st.status == StatusNotApplicable || st.status == StatusFailed || st.reason == reasonUpstreamFailed)
	}
	// The run ends when no branch can move any more; a failed branch makes it a partial failure.
	switch {
	case finished && failed:
		_, err = s.pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='failed',updated_at=now(),finished_at=now() WHERE id=$1 AND status='active'`, run.ID)
	case finished:
		_, err = s.pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='completed',updated_at=now(),finished_at=now() WHERE id=$1 AND status='active'`, run.ID)
	}
	return err
}

const reasonUpstreamFailed = "upstream_failed"

func fromResult(r StepResult) stepState {
	st := stepState{status: r.Status, reason: r.Reason}
	if r.Job != nil {
		id := r.Job.ID
		st.jobID, st.attempt = &id, r.Job.Attempts
	}
	return st
}

// dependentState is a step whose input step is in prev: it waits, unless prev failed.
func dependentState(prev stepState, waitReason string) stepState {
	if prev.status == StatusFailed || prev.reason == reasonUpstreamFailed {
		return stepState{status: StatusBlocked, reason: reasonUpstreamFailed}
	}
	return stepState{status: StatusWaiting, reason: waitReason}
}

// voiceStates fills the diarization, fingerprint and matching steps.
func (s *Service) voiceStates(ctx context.Context, run Run, since time.Time, force bool, states map[string]stepState) error {
	vs, err := s.hooks.Voice(ctx, run.ClipID, since, force)
	if err != nil {
		return err
	}
	states[StepDiarization] = fromResult(vs.Diarization)
	states[StepFingerprints] = fromResult(vs.Fingerprints)
	switch {
	case vs.Matching != nil:
		states[StepMatching] = fromResult(*vs.Matching)
		return nil
	case vs.Fingerprints.Status != StatusDone:
		states[StepMatching] = dependentState(states[StepFingerprints], "awaiting_fingerprints")
		return nil
	}
	// Matching runs in the database, not in a job: this run owns its outcome and retries it.
	var cur stepState
	if err = s.pool.QueryRow(ctx, `SELECT status,reason,attempt FROM clip_analysis_steps WHERE run_id=$1 AND step=$2`, run.ID, StepMatching).Scan(&cur.status, &cur.reason, &cur.attempt); err != nil {
		return err
	}
	if cur.status == StatusDone || cur.status == StatusFailed {
		states[StepMatching] = cur
		return nil
	}
	if matchErr := s.hooks.Match(ctx, run.ClipID); matchErr != nil {
		s.log.Warn("match analysis voices", "run", run.ID, "error", matchErr)
		attempt := cur.attempt + 1
		if attempt >= maxMatchingAttempts {
			states[StepMatching] = stepState{status: StatusFailed, reason: "matching_failed", attempt: attempt}
		} else {
			states[StepMatching] = stepState{status: StatusRetry, reason: "retry_scheduled", attempt: attempt}
		}
		return nil
	}
	states[StepMatching] = stepState{status: StatusDone, reason: "completed", attempt: cur.attempt + 1}
	return nil
}

func transcriptionState(tr Transcription, runStart time.Time) stepState {
	st := jobState(tr.Job, runStart)
	if st.status == StatusDone && tr.Kept {
		st.reason = "kept_for_corrections"
	}
	return st
}

// JobResult maps a job to a step result; a job older than the run is a reused result.
func JobResult(job jobs.Job, runStart time.Time) StepResult {
	st := jobState(job, runStart)
	return StepResult{Status: st.status, Reason: st.reason, Job: &job}
}

func jobState(job jobs.Job, runStart time.Time) stepState {
	id := job.ID
	st := stepState{jobID: &id, attempt: job.Attempts}
	switch job.Status {
	case jobs.Pending:
		st.status, st.reason = StatusWaiting, "awaiting_worker"
	case jobs.Leased:
		st.status, st.reason = StatusRunning, "running"
	case jobs.RetryWait:
		st.status, st.reason = StatusRetry, "retry_scheduled"
	case jobs.Done:
		st.status, st.reason = StatusDone, "completed"
		if job.CreatedAt.Before(runStart) {
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

// workerWaitReason says exactly why a pending job waits: no connected Worker supports its
// step (awaiting_compatible_worker), or one does and is busy (awaiting_worker).
func (s *Service) workerWaitReason(ctx context.Context, jobID int64) (string, error) {
	var compatible bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM jobs j, workers w, unnest(w.supported_steps) AS capability
 WHERE j.id=$1 AND w.revoked_at IS NULL AND w.last_seen_at >= now() - ($2 * interval '1 second')
 AND CASE WHEN right(capability,2)='.*' THEN left(j.step,length(capability)-1)=left(capability,length(capability)-1) ELSE j.step=capability END)`, jobID, int64(workers.ConnectedWindow.Seconds())).Scan(&compatible)
	if compatible {
		return "awaiting_worker", err
	}
	return "awaiting_compatible_worker", err
}

// Cancel stops the active run of a clip: no further step is planned, pending jobs of the
// run are cancelled, and a step already running may still finish with a valid result.
func (s *Service) Cancel(ctx context.Context, clipID int64) (*Run, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='cancelled',updated_at=now(),finished_at=now() WHERE clip_id=$1 AND status='active'`, clipID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotActive
	}
	if _, err = s.pool.Exec(ctx, `UPDATE jobs SET status='cancelled',updated_at=now() WHERE status IN ('pending','retry_wait') AND id IN (SELECT st.job_id FROM clip_analysis_steps st JOIN clip_analysis_runs r ON r.id=st.run_id WHERE r.clip_id=$1 AND r.status='cancelled' AND st.job_id IS NOT NULL AND st.status IN ('waiting','retry','blocked'))`, clipID); err != nil {
		return nil, err
	}
	if err = s.settleCancelled(ctx, clipID); err != nil {
		return nil, err
	}
	return s.Latest(ctx, clipID)
}

// settleCancelled brings the steps of cancelled runs to their final state: a step that was
// running keeps following its job until it ends, the others are marked cancelled.
func (s *Service) settleCancelled(ctx context.Context, clipID int64) error {
	_, err := s.pool.Exec(ctx, `WITH t AS (
 SELECT st.run_id,st.step,j.status AS job_status FROM clip_analysis_steps st
 JOIN clip_analysis_runs r ON r.id=st.run_id LEFT JOIN jobs j ON j.id=st.job_id
 WHERE r.status='cancelled' AND ($1=0 OR r.clip_id=$1) AND st.status IN ('waiting','running','retry','blocked')
 AND NOT (st.status='blocked' AND st.reason='run_cancelled') AND NOT (st.status='running' AND j.status='leased'))
UPDATE clip_analysis_steps st SET
 status=CASE WHEN t.job_status='done' THEN 'done' WHEN t.job_status='leased' THEN 'running' WHEN t.job_status IN ('dead','failed') THEN 'failed' ELSE 'blocked' END,
 reason=CASE WHEN t.job_status='done' THEN 'completed' WHEN t.job_status='leased' THEN 'running' WHEN t.job_status IN ('dead','failed') THEN 'job_failed' ELSE 'run_cancelled' END,
 finished_at=CASE WHEN t.job_status='leased' THEN NULL ELSE COALESCE(st.finished_at,now()) END,updated_at=now()
 FROM t WHERE st.run_id=t.run_id AND st.step=t.step`, clipID)
	return err
}

// Retry runs a failed step again after a definitive failure: the failed job of that step
// is ignored so it gets a fresh one, valid results stay, and the run is active again.
func (s *Service) Retry(ctx context.Context, clipID int64, step string) (*Run, error) {
	if !slices.Contains(stepOrder, step) {
		return nil, ErrInvalidStep
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	run, err := scanRun(tx.QueryRow(ctx, `SELECT `+runColumns+` FROM clip_analysis_runs WHERE clip_id=$1 ORDER BY (status='active') DESC,id DESC LIMIT 1 FOR UPDATE`, clipID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoRun
	}
	if err != nil {
		return nil, err
	}
	if run.Status != RunActive && run.Status != RunFailed {
		return nil, ErrNotRetryable
	}
	tag, err := tx.Exec(ctx, `UPDATE clip_analysis_steps SET status='waiting',reason='awaiting_previous_step',attempt=0,job_id=NULL,retry_since=now(),finished_at=NULL,updated_at=now() WHERE run_id=$1 AND step=$2 AND status='failed'`, run.ID, step)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrStepNotFailed
	}
	if _, err = tx.Exec(ctx, `UPDATE clip_analysis_runs SET status='active',updated_at=now(),finished_at=NULL WHERE id=$1`, run.ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	if step == StepIndexing {
		if err = s.hooks.RetryIndex(ctx, clipID); err != nil {
			return nil, err
		}
	}
	run.Status = RunActive
	if err = s.advance(ctx, run); err != nil {
		s.log.Warn("advance retried analysis run", "run", run.ID, "error", err)
	}
	return s.Latest(ctx, clipID)
}
