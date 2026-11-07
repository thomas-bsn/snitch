package analysis

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
	"snitch/internal/multitrack"
)

// AudioDiagnostics is the injectable production detector boundary. Intervals
// are track-relative; the Worker adds the stream offset exactly once.
type AudioDiagnostics interface {
	DiagnoseAudio(context.Context, audioplans.Plan, contract.AudioTrackInventory) (AudioObservation, error)
}

// WorkerAudioDiagnostics is the production detector boundary when the examination runs on a Worker (#338,
// ADR 0048): Reconcile queues one durable audio.diagnostic.N job per track (idempotent) and waits for the Worker's
// terminal observation, which the completion transaction records. It never examines audio in the server process,
// so DiagnoseAudio is not used. Wire the notification of a completed child to Reconcile like for transcription.
type WorkerAudioDiagnostics struct{}

func (WorkerAudioDiagnostics) DiagnoseAudio(context.Context, audioplans.Plan, contract.AudioTrackInventory) (AudioObservation, error) {
	return AudioObservation{}, errors.New("diagnostic_runs_on_worker")
}

type AudioObservation struct {
	Status    contract.AudioTrackDiagnosticStatus
	Cause     string
	Intervals []contract.AudioSpeechInterval
}

// AudioPublisher publishes the complete plan in the caller's transaction. It
// must use tx for all durable effects; a failure leaves the run resumable.
type AudioPublisher interface {
	PublishAudio(context.Context, pgx.Tx, multitrack.Result) error
}

type AudioService struct {
	pool        *pgxpool.Pool
	diagnostics AudioDiagnostics
	publisher   AudioPublisher
}

func NewAudio(pool *pgxpool.Pool, diagnostics AudioDiagnostics, publisher AudioPublisher) *AudioService {
	return &AudioService{pool, diagnostics, publisher}
}

var ErrActiveAudio = errors.New("a different analysis is active for this clip")

// Start is an explicit opt-in seam. Legacy starts never create audio plans.
// The clip lock and partial unique index also exclude legacy concurrent starts.
func (s *AudioService) Start(ctx context.Context, clipID int64, input audioplans.Input) (audioplans.Plan, error) {
	canonical, err := audioplans.CanonicalInput(input)
	if err != nil {
		return audioplans.Plan{}, err
	}
	input = canonical.Input
	if input.Mode == "auto" {
		for _, track := range input.Inventory {
			if !slices.Contains(input.TrackIndices, track.Index) {
				return audioplans.Plan{}, errors.New("auto requires the entire audio inventory")
			}
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return audioplans.Plan{}, err
	}
	defer tx.Rollback(ctx)
	var clip int64
	if err = tx.QueryRow(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, clipID).Scan(&clip); err != nil {
		return audioplans.Plan{}, err
	}
	var runID int64
	err = tx.QueryRow(ctx, `SELECT id FROM clip_analysis_runs WHERE clip_id=$1 AND status='active'`, clipID).Scan(&runID)
	if err == nil {
		var id int64
		err = tx.QueryRow(ctx, `SELECT id FROM audio_plans WHERE run_id=$1 AND input_version=$2`, runID, canonical.Version).Scan(&id)
		if err != nil {
			return audioplans.Plan{}, ErrActiveAudio
		}
		state, e := audioplans.New(tx).GetPlan(ctx, id)
		return state.Plan, e
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return audioplans.Plan{}, err
	}
	if err = tx.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope) VALUES($1,'transcription') RETURNING id`, clipID).Scan(&runID); err != nil {
		return audioplans.Plan{}, err
	}
	plan, err := audioplans.New(tx).CreatePlan(ctx, runID, input)
	if err != nil {
		return plan, err
	}
	for _, step := range stepOrder {
		status := "not_applicable"
		if step == StepPreparation || step == StepTranscription {
			status = "waiting"
		}
		if _, err = tx.Exec(ctx, `INSERT INTO clip_analysis_steps(run_id,step,status,reason) VALUES($1,$2,$3,'audio_plan')`, runID, step, status); err != nil {
			return plan, err
		}
	}
	return plan, tx.Commit(ctx)
}

type AudioTrackProgress struct {
	TrackIndex int                   `json:"track_index"`
	Status     string                `json:"status"`
	Cause      string                `json:"cause"`
	JobID      int64                 `json:"job_id,omitempty"`
	Attempt    int                   `json:"attempt"`
	Diagnostic audioplans.Diagnostic `json:"diagnostic"`
}
type AudioProgress struct {
	State  audioplans.State     `json:"state"`
	Status string               `json:"status"`
	Cause  string               `json:"cause"`
	Tracks []AudioTrackProgress `json:"tracks"`
}

func (s *AudioService) State(ctx context.Context, id int64) (AudioProgress, error) {
	state, err := audioplans.New(s.pool).GetPlan(ctx, id)
	out := AudioProgress{State: state, Tracks: []AudioTrackProgress{}}
	if err != nil {
		return out, err
	}
	err = s.pool.QueryRow(ctx, `SELECT st.status,st.reason FROM clip_analysis_steps st WHERE run_id=$1 AND step='transcription'`, state.Plan.RunId).Scan(&out.Status, &out.Cause)
	if err != nil {
		return out, err
	}
	for _, d := range state.Diagnostics {
		track := AudioTrackProgress{TrackIndex: d.TrackIndex, Status: string(d.Status), Cause: d.Cause, Diagnostic: d}
		for _, c := range state.Children {
			if c.TrackIndex == d.TrackIndex && c.Stage == "transcription" {
				track.JobID = c.JobId
				track.Status = string(c.Status)
				track.Attempt = c.Attempt
				if err = s.pool.QueryRow(ctx, `SELECT last_error FROM jobs WHERE id=$1`, c.JobId).Scan(&track.Cause); err != nil {
					return out, err
				}
			}
		}
		out.Tracks = append(out.Tracks, track)
	}
	return out, nil
}

// Reconcile serializes on the run. A restart, duplicate notification or concurrent
// caller follows the same immutable plan and per-stage child bindings.
func (s *AudioService) Reconcile(ctx context.Context, id int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	if err = tx.QueryRow(ctx, `SELECT r.status FROM clip_analysis_runs r JOIN audio_plans p ON p.run_id=r.id WHERE p.id=$1 FOR UPDATE OF r`, id).Scan(&status); err != nil {
		return err
	}
	if status != "active" {
		return nil
	}
	store := audioplans.New(tx)
	state, err := store.GetPlan(ctx, id)
	if err != nil {
		return err
	}
	plan := state.Plan
	set := func(step, status, cause string) error {
		_, e := tx.Exec(ctx, `UPDATE clip_analysis_steps SET status=$3,reason=$4,updated_at=now() WHERE run_id=$1 AND step=$2`, plan.RunId, step, status, cause)
		return e
	}
	waitingOnWorker := false
	for _, d := range state.Diagnostics {
		// A diagnostic runs when never examined, or when an explicit Retry put its
		// durable job back in the queue (the job keeps its identity, attempts grow).
		var retryJob int64
		var retryAttempt int
		retrying := false
		for _, child := range state.Children {
			if child.TrackIndex == d.TrackIndex && child.Stage == "diagnostic" && child.Status == "queued" {
				retrying, retryJob, retryAttempt = true, child.JobId, child.Attempt
			}
		}
		if d.Status != "unexamined" && !retrying {
			continue
		}
		if _, onWorker := s.diagnostics.(WorkerAudioDiagnostics); onWorker {
			// Asynchronous: queue once, then wait. A job that died without any observation is a visible failure.
			waitingOnWorker = true
			var child *contract.AudioPlanChild
			for i := range state.Children {
				if state.Children[i].TrackIndex == d.TrackIndex && state.Children[i].Stage == "diagnostic" {
					child = &state.Children[i]
				}
			}
			if child == nil {
				if _, err = store.QueueDiagnostics(ctx, id); err != nil {
					return err
				}
			} else if child.Status == "dead" && d.Status == "unexamined" {
				var cause string
				if err = tx.QueryRow(ctx, `SELECT last_error FROM jobs WHERE id=$1`, child.JobId).Scan(&cause); err != nil {
					return err
				}
				if cause == "" {
					cause = "worker_job_failed"
				}
				d.Status, d.Cause, d.Intervals, d.JobId, d.Attempt = "failed", cause, nil, &child.JobId, child.Attempt
				if err = store.RecordDiagnostic(ctx, d); err != nil {
					return err
				}
				waitingOnWorker = false
			}
			continue
		}
		if s.diagnostics == nil {
			if err = set(StepTranscription, "blocked", "diagnostics_unavailable"); err != nil {
				return err
			}
			return tx.Commit(ctx)
		}
		jobID, attempt := retryJob, retryAttempt
		if !retrying {
			attempt = 1
			if err = tx.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,attempts) VALUES($1,$2,$3,1) RETURNING id`, plan.ClipId, fmt.Sprintf("audio.diagnostic.%d", d.TrackIndex), multitrack.Prefix+plan.InputVersion).Scan(&jobID); err != nil {
				return err
			}
			if err = store.BindChild(ctx, audioplans.Child{PlanId: id, TrackIndex: d.TrackIndex, Stage: "diagnostic", JobId: jobID, InputVersion: multitrack.Prefix + plan.InputVersion}); err != nil {
				return err
			}
		}
		var track contract.AudioTrackInventory
		for _, t := range plan.Input.Inventory {
			if t.Index == d.TrackIndex {
				track = t
			}
		}
		observation, diagnosticErr := s.diagnostics.DiagnoseAudio(ctx, plan, track)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if diagnosticErr != nil {
			observation = AudioObservation{Status: "failed", Cause: diagnosticErr.Error()}
		}
		d.Status = observation.Status
		d.Cause = observation.Cause
		d.Intervals = observation.Intervals
		d.JobId = &jobID
		d.Attempt = attempt
		jobStatus := "done"
		if d.Status == "failed" {
			jobStatus = "dead"
		}
		if _, err = tx.Exec(ctx, `UPDATE jobs SET status=$2,last_error=$3 WHERE id=$1`, jobID, jobStatus, d.Cause); err != nil {
			return err
		}
		if err = store.RecordDiagnostic(ctx, d); err != nil {
			return err
		}
	}
	state, err = store.GetPlan(ctx, id)
	if err != nil {
		return err
	}
	if waitingOnWorker {
		// Every track must be examined before any decision or transcription: nothing is inferred meanwhile.
		if err = set(StepPreparation, "running", "diagnostics_running"); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	retained := []int{}
	for _, d := range state.Diagnostics {
		forced := false
		for _, decision := range plan.Input.Decisions {
			if decision.TrackIndex == d.TrackIndex && decision.Action == "include" && decision.Forced {
				forced = true
			}
		}
		if d.Status == "failed" || d.Status == "ambiguous" || (plan.Input.Mode == "manual" && d.Status == "no_speech" && !forced) {
			if !forced {
				if err = set(StepTranscription, "blocked", "diagnostic_"+string(d.Status)); err != nil {
					return err
				}
				return tx.Commit(ctx)
			}
		}
		if d.Status == "speech" || forced {
			retained = append(retained, d.TrackIndex)
		}
	}
	if err = set(StepPreparation, "done", "diagnostics_complete"); err != nil {
		return err
	}
	for _, track := range retained {
		found := false
		for _, child := range state.Children {
			if child.TrackIndex == track && child.Stage == "transcription" {
				found = true
			}
		}
		if found {
			continue
		}
		var jobID int64
		version := fmt.Sprintf("%s%d:%s", multitrack.Prefix, id, plan.InputVersion)
		if err = tx.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version) VALUES($1,$2,$3) RETURNING id`, plan.ClipId, fmt.Sprintf("transcription.audio.%d", track), version).Scan(&jobID); err != nil {
			return err
		}
		if err = store.BindChild(ctx, audioplans.Child{PlanId: id, TrackIndex: track, Stage: "transcription", JobId: jobID, InputVersion: version}); err != nil {
			return err
		}
	}
	state, err = store.GetPlan(ctx, id)
	if err != nil {
		return err
	}
	result := multitrack.Result{Plan: plan, Tracks: []multitrack.TrackResult{}, Empty: len(retained) == 0}
	aggregateStatus, aggregateCause := "waiting", "audio_children"
	complete := true
	for _, child := range state.Children {
		if child.Stage != "transcription" {
			continue
		}
		if child.Status != "done" {
			complete = false
			switch child.Status {
			case "dead", "cancelled":
				aggregateStatus = "failed"
				aggregateCause = "audio_child_failed"
			case "running":
				if aggregateStatus != "failed" {
					aggregateStatus = "running"
				}
			case "retry":
				if aggregateStatus == "waiting" {
					aggregateStatus = "retry"
				}
			}
			continue
		}
		var output []byte
		if err = tx.QueryRow(ctx, `SELECT result FROM jobs WHERE id=$1`, child.JobId).Scan(&output); err != nil {
			return err
		}
		if err = multitrack.ValidateOutput(output, plan, child.TrackIndex); err != nil {
			complete = false
			aggregateStatus = "blocked"
			aggregateCause = "invalid_audio_result"
			continue
		}
		result.Tracks = append(result.Tracks, multitrack.TrackResult{TrackIndex: child.TrackIndex, JobID: child.JobId, InputVersion: child.InputVersion, Output: output})
	}
	if complete {
		if s.publisher == nil {
			aggregateStatus = "blocked"
			aggregateCause = "publication_unavailable"
		} else {
			if err = s.publisher.PublishAudio(ctx, tx, result); err != nil {
				return err
			}
			aggregateStatus = "done"
			aggregateCause = "audio_plan_published"
			if _, err = tx.Exec(ctx, `UPDATE clip_analysis_runs SET status='completed',finished_at=now(),updated_at=now() WHERE id=$1`, plan.RunId); err != nil {
				return err
			}
		}
	}
	if err = set(StepTranscription, aggregateStatus, aggregateCause); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Cancel stops the plan's own children and prevents any later planning. Jobs of
// other plans or legacy runs of the clip are untouched. It is idempotent.
func (s *AudioService) Cancel(ctx context.Context, id int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var runID int64
	var status string
	if err = tx.QueryRow(ctx, `SELECT r.id,r.status FROM clip_analysis_runs r JOIN audio_plans p ON p.run_id=r.id WHERE p.id=$1 FOR UPDATE OF r`, id).Scan(&runID, &status); err != nil {
		return err
	}
	if status != "active" {
		return nil
	}
	if _, err = tx.Exec(ctx, `UPDATE clip_analysis_runs SET status='cancelled',updated_at=now(),finished_at=now() WHERE id=$1`, runID); err != nil {
		return err
	}
	if err = cancelAudioChildren(ctx, tx, runID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// cancelAudioChildren cancels the unfinished jobs bound to the plans of runID,
// including leased ones (the Worker loses its lease and stops), and settles the
// run's steps. The caller already marked the run cancelled.
func cancelAudioChildren(ctx context.Context, q execer, runID int64) error {
	if _, err := q.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,attempt,actor,error_code)
 SELECT j.id,'cancelled',j.cycle,CASE WHEN j.status='leased' AND j.attempts>0 THEN j.attempts END,'system','audio_plan_cancelled'
 FROM jobs j JOIN audio_plan_children c ON c.job_id=j.id JOIN audio_plans p ON p.id=c.plan_id
 WHERE p.run_id=$1 AND j.status IN ('pending','retry_wait','leased')`, runID); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `UPDATE jobs SET status='cancelled',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,updated_at=now()
 WHERE status IN ('pending','retry_wait','leased') AND id IN (SELECT c.job_id FROM audio_plan_children c JOIN audio_plans p ON p.id=c.plan_id WHERE p.run_id=$1)`, runID); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `UPDATE clip_analysis_steps SET status='blocked',reason='run_cancelled',finished_at=COALESCE(finished_at,now()),updated_at=now()
 WHERE run_id=$1 AND status IN ('waiting','running','retry','blocked','failed') AND EXISTS(SELECT 1 FROM audio_plans WHERE run_id=$1)`, runID)
	return err
}

// Retry puts one failed child of the plan back in the queue. Successful siblings
// keep their results; only the named track and stage are recomputed.
func (s *AudioService) Retry(ctx context.Context, id int64, trackIndex int, stage string) error {
	if stage != "transcription" && stage != "diagnostic" {
		return ErrNotRetryableAudio
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	if err = tx.QueryRow(ctx, `SELECT r.status FROM clip_analysis_runs r JOIN audio_plans p ON p.run_id=r.id WHERE p.id=$1 FOR UPDATE OF r`, id).Scan(&status); err != nil {
		return err
	}
	if status != "active" {
		return ErrNotRetryableAudio
	}
	var jobID int64
	var cycle int
	// A diagnostic a Worker examined and reported as failed completes its job: it is retryable too, and its next
	// attempt is the next claim (no attempt is skipped).
	var failedOnWorker bool
	err = tx.QueryRow(ctx, `SELECT j.id,j.cycle,j.status='done' FROM audio_plan_children c JOIN jobs j ON j.id=c.job_id WHERE c.plan_id=$1 AND c.track_index=$2 AND c.stage=$3
 AND (j.status='dead' OR ($3='diagnostic' AND j.status='done' AND (SELECT d.payload->>'status' FROM audio_track_diagnostics d WHERE d.plan_id=c.plan_id AND d.track_index=c.track_index ORDER BY d.attempt DESC LIMIT 1)='failed')) FOR UPDATE OF j`, id, trackIndex, stage).Scan(&jobID, &cycle, &failedOnWorker)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotRetryableAudio
	}
	if err != nil {
		return err
	}
	// A transcription job restarts its own attempt counter, like an operator
	// relaunch; a diagnostic keeps counting so every observation stays immutable.
	attempts := "0"
	if stage == "diagnostic" && !failedOnWorker {
		attempts = "attempts+1"
	} else if stage == "diagnostic" {
		attempts = "attempts"
	}
	if _, err = tx.Exec(ctx, `UPDATE jobs SET status='pending',attempts=`+attempts+`,cycle=cycle+1,next_attempt_at=now(),last_error='',updated_at=now() WHERE id=$1`, jobID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,actor,error_code) VALUES($1,'admin_relaunched',$2,'system','audio_plan_retry')`, jobID, cycle+1); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE clip_analysis_steps SET status='waiting',reason='audio_retry',finished_at=NULL,updated_at=now() WHERE run_id=(SELECT run_id FROM audio_plans WHERE id=$1) AND step='transcription'`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var ErrNotRetryableAudio = errors.New("audio child is not retryable")
