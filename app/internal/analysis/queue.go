package analysis

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"

	"snitch/internal/jobs"
)

// CompletedRuns is how many finished clips the queue keeps in view; the rest is history.
const CompletedRuns = 3

// NoDispositionVoice keeps to the local speakers (alias ls) that have no disposition (ignored
// or random, ADR 0031): the voices treated normally, the only ones verification and matching
// consider.
const NoDispositionVoice = `NOT EXISTS (SELECT 1 FROM local_speaker_dispositions d WHERE d.local_speaker_id=ls.id)`

// UnknownSpeakerPredicate selects (alias ls) the diarized voices no one has validated or
// automatically attributed, and that are not set aside as ignored or random (ADR 0031).
// PendingVoiceCondition narrows them to those that still await a decision (the Verification
// list and the speaker-review notification use the same one).
const UnknownSpeakerPredicate = `ls.kind='diarized' AND NOT EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND (a.validation_state='validated' OR a.decision_level='automatic')) AND ` + NoDispositionVoice

const PendingVoiceCondition = UnknownSpeakerPredicate + `
AND COALESCE((SELECT a.decision_level FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.validation_state='unvalidated'),'unknown') IN ('unknown','ambiguous')`

// HistoryPageSize is the page of the durable run history.
const HistoryPageSize = 10

// Location is where the original video of a clip lives in the storage Snitch indexes: the source
// and the full path under its root, as persisted, never a Worker work file. A file in a private
// folder names its source only; its path is not sent, since the queue is one snapshot for every
// administrator.
type Location struct {
	Source  string `json:"source"`
	Path    string `json:"path,omitempty"`
	Private bool   `json:"private,omitempty"`
}

type QueueRun struct {
	Run
	ClipName string `json:"clip_name"`
	// Locations lists every indexed file of the clip that is not missing; none invented as the main one.
	Locations []Location `json:"locations"`
}

// RunHistory is one page of the finished runs, newest first.
type RunHistory struct {
	Items    []QueueRun `json:"items"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

type QueueJob struct {
	jobs.Job
	ClipName string `json:"clip_name"`
	// FailureCode is the persisted cause of the last failure and RecommendedAction the advised
	// recovery (jobs.RecommendedAction); both are empty without a failure.
	FailureCode       string `json:"failure_code,omitempty"`
	RecommendedAction string `json:"recommended_action,omitempty"`
}

// ActionStartWorker is the remedy of a step that waits for a Worker.
const ActionStartWorker = "start_worker"

// QueueSummary is the load of the queue at a glance. RunningTasks counts the jobs that hold a
// concurrency slot, exactly as the scheduler does (leased, lease not expired); WaitingClips counts
// the clips with an active analysis and no slot, whatever the reason (waiting, retrying, suspended).
type QueueSummary struct {
	RunningTasks   int  `json:"running_tasks"`
	WaitingClips   int  `json:"waiting_clips"`
	MaxConcurrency int  `json:"max_concurrency"`
	Paused         bool `json:"paused"`
}

// failureCodeOf keeps only a catalogue code: the cause of the last failure event, else the code
// kept on the job; anything else is a generic worker_failed, never the raw text.
func failureCodeOf(eventCode, lastError string) string {
	code := eventCode
	if code == "" {
		code = lastError
	}
	switch {
	case code == "":
		return ""
	case jobs.RecommendedAction(code) != "":
		return code
	default:
		return "worker_failed"
	}
}

// latestFailureCode is the SQL for the cause of the last persisted failure event of job j.
const latestFailureCode = `COALESCE((SELECT e.error_code FROM job_events e WHERE e.job_id=j.id AND e.error_code<>'' ORDER BY e.id DESC LIMIT 1),'')`

// Queue is the state of the Task Queue: one entry per analysis run, never paginated, plus the
// technical tasks that belong to no run so they are never lost.
type Queue struct {
	// Active lists the runs that still need attention: running, waiting, blocked, failed or
	// cancelled (never counted as finished, planned again to leave), and not superseded by a newer run of the same clip.
	Active []QueueRun `json:"active"`
	// Completed lists the last clips whose automatic processing finished.
	Completed []QueueRun `json:"completed"`
	// Tasks lists the unfinished jobs that no listed run follows.
	Tasks []QueueJob `json:"tasks"`
	// Summary is the load of the queue: running tasks on the ceiling, waiting clips, pause.
	Summary QueueSummary `json:"summary"`
}

// queueRuns lists the latest run of each clip that matches where; a limit of 0 lists them all, as
// the Task Queue does for the active runs: none is ever left out, whatever the size of the queue.
// QueueDelta is what changed in the Task Queue since a previous snapshot: the order of the
// active runs, the full content of those that are new or changed, and the small parts (finished
// runs, loose tasks, load) whole. A client rebuilds the queue from the snapshot it holds.
type QueueDelta struct {
	// ActiveIDs lists every active run, in the order of the queue.
	ActiveIDs []int64 `json:"active_ids"`
	// Active holds the runs of ActiveIDs that are new or differ from the previous snapshot.
	Active    []QueueRun   `json:"active"`
	Completed []QueueRun   `json:"completed"`
	Tasks     []QueueJob   `json:"tasks"`
	Summary   QueueSummary `json:"summary"`
}

// DeltaFrom compares the queue with a previous snapshot.
func (q Queue) DeltaFrom(previous Queue) QueueDelta {
	before := make(map[int64]*QueueRun, len(previous.Active))
	for i := range previous.Active {
		before[previous.Active[i].ID] = &previous.Active[i]
	}
	d := QueueDelta{ActiveIDs: make([]int64, 0, len(q.Active)), Active: []QueueRun{}, Completed: q.Completed, Tasks: q.Tasks, Summary: q.Summary}
	for i := range q.Active {
		run := &q.Active[i]
		d.ActiveIDs = append(d.ActiveIDs, run.ID)
		if old, ok := before[run.ID]; !ok || !reflect.DeepEqual(*old, *run) {
			d.Active = append(d.Active, *run)
		}
	}
	return d
}

func (s *Service) queueRuns(ctx context.Context, where, order string, limit int) ([]QueueRun, error) {
	rows, err := s.pool.Query(ctx, `SELECT r.id,r.clip_id,r.scope,r.status,r.started_by,r.created_at,r.updated_at,r.finished_at,COALESCE(c.display_title,c.original_name)
FROM clip_analysis_runs r JOIN clips c ON c.id=r.clip_id
WHERE r.id=(SELECT max(id) FROM clip_analysis_runs WHERE clip_id=r.clip_id) AND (`+where+`) ORDER BY `+order+` LIMIT NULLIF($1,0)`, limit)
	if err != nil {
		return nil, fmt.Errorf("list queue runs: %w", err)
	}
	return s.scanQueueRuns(ctx, rows)
}

func (s *Service) scanQueueRuns(ctx context.Context, rows pgx.Rows) ([]QueueRun, error) {
	defer rows.Close()
	out := []QueueRun{}
	ids := []int64{}
	clips := []int64{}
	var err error
	for rows.Next() {
		var r QueueRun
		if err = rows.Scan(&r.ID, &r.ClipID, &r.Scope, &r.Status, &r.StartedBy, &r.CreatedAt, &r.UpdatedAt, &r.FinishedAt, &r.ClipName); err != nil {
			return nil, err
		}
		r.Steps = []Step{}
		r.Locations = []Location{}
		out = append(out, r)
		ids = append(ids, r.ID)
		clips = append(clips, r.ClipID)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	steps, err := s.pool.Query(ctx, `SELECT st.run_id,st.step,st.status,st.reason,st.job_id,st.attempt,st.started_at,st.finished_at,st.updated_at,
 COALESCE(j.status,''),COALESCE(j.next_attempt_at<=now(),false),COALESCE(j.last_error,''),`+latestFailureCode+`
FROM clip_analysis_steps st LEFT JOIN jobs j ON j.id=st.job_id WHERE st.run_id=ANY($1) ORDER BY st.run_id,array_position($2::text[],st.step)`, ids, stepOrder)
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
		var jobStatus, lastError, eventCode string
		var due bool
		if err = steps.Scan(&runID, &st.Step, &st.Status, &st.Reason, &st.JobID, &st.Attempt, &st.StartedAt, &st.FinishedAt, &st.UpdatedAt, &jobStatus, &due, &lastError, &eventCode); err != nil {
			return nil, err
		}
		switch {
		case st.Status == StatusFailed:
			st.FailureCode = failureCodeOf(eventCode, lastError)
			st.RecommendedAction = jobs.RecommendedAction(st.FailureCode)
		case (st.Reason == "awaiting_worker" || st.Reason == "awaiting_compatible_worker") && jobStatus == string(jobs.Pending) && due:
			// A job whose next attempt is still in the future is a deferred retry, not a Worker wait.
			st.RecommendedAction = ActionStartWorker
		}
		out[index[runID]].Steps = append(out[index[runID]].Steps, st)
	}
	if err = steps.Err(); err != nil {
		return nil, err
	}
	steps.Close()
	if err = s.attachLocations(ctx, out, clips); err != nil {
		return nil, err
	}
	counts, err := s.pendingVoices(ctx, clips)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].PendingVoices = counts[out[i].ClipID]
	}
	return out, nil
}

// attachLocations reads, in one query, the indexed files of the clips of the runs.
func (s *Service) attachLocations(ctx context.Context, runs []QueueRun, clips []int64) error {
	rows, err := s.pool.Query(ctx, `SELECT sf.clip_id,cs.name,cs.root_path,sf.relative_path,path_in_private_folder(sf.source_id,sf.relative_path)
FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id
WHERE sf.clip_id=ANY($1) AND sf.status<>'missing' ORDER BY sf.clip_id,cs.name,sf.relative_path`, clips)
	if err != nil {
		return fmt.Errorf("list clip locations: %w", err)
	}
	defer rows.Close()
	byClip := map[int64][]Location{}
	for rows.Next() {
		var clip int64
		var name, root, rel string
		var private bool
		if err = rows.Scan(&clip, &name, &root, &rel, &private); err != nil {
			return err
		}
		loc := Location{Source: name, Private: private}
		if !private {
			loc.Path = filepath.Join(root, rel)
		}
		byClip[clip] = append(byClip[clip], loc)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for i := range runs {
		if l := byClip[runs[i].ClipID]; l != nil {
			runs[i].Locations = l
		}
	}
	return nil
}

// pendingVoices counts, per clip, the voices still awaiting a decision (PendingVoiceCondition).
func (s *Service) pendingVoices(ctx context.Context, clips []int64) (map[int64]int, error) {
	rows, err := s.pool.Query(ctx, `SELECT ls.clip_id,count(*)::int FROM local_speakers ls WHERE ls.clip_id=ANY($1) AND `+PendingVoiceCondition+` GROUP BY ls.clip_id`, clips)
	if err != nil {
		return nil, fmt.Errorf("count pending voices: %w", err)
	}
	defer rows.Close()
	counts := map[int64]int{}
	for rows.Next() {
		var clip int64
		var n int
		if err = rows.Scan(&clip, &n); err != nil {
			return nil, err
		}
		counts[clip] = n
	}
	return counts, rows.Err()
}

// FinishedRuns lists the finished runs of every clip, newest first, optionally narrowed to the
// clips whose name contains q. Steps and failures are kept: this is the durable record.
func (s *Service) FinishedRuns(ctx context.Context, q string, page int) (RunHistory, error) {
	if page < 1 {
		page = 1
	}
	h := RunHistory{Page: page, PageSize: HistoryPageSize}
	like := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
	const where = `r.status<>'active' AND (c.original_name ILIKE $1 OR c.display_title ILIKE $1)`
	if err := s.pool.QueryRow(ctx, `SELECT count(*)::int FROM clip_analysis_runs r JOIN clips c ON c.id=r.clip_id WHERE `+where, like).Scan(&h.Total); err != nil {
		return h, fmt.Errorf("count run history: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT r.id,r.clip_id,r.scope,r.status,r.started_by,r.created_at,r.updated_at,r.finished_at,COALESCE(c.display_title,c.original_name)
FROM clip_analysis_runs r JOIN clips c ON c.id=r.clip_id WHERE `+where+` ORDER BY r.finished_at DESC NULLS LAST,r.id DESC LIMIT $2 OFFSET $3`, like, HistoryPageSize, (page-1)*HistoryPageSize)
	if err != nil {
		return h, fmt.Errorf("list run history: %w", err)
	}
	if h.Items, err = s.scanQueueRuns(ctx, rows); err != nil {
		return h, err
	}
	return h, s.attachStepEvents(ctx, h.Items)
}

// attachStepEvents loads, in one query, the events of every job the steps of the runs followed.
func (s *Service) attachStepEvents(ctx context.Context, runs []QueueRun) error {
	type position struct{ run, step int }
	where := map[int64][]position{}
	ids := []int64{}
	for i, r := range runs {
		for j, st := range r.Steps {
			if st.JobID != nil {
				if _, seen := where[*st.JobID]; !seen {
					ids = append(ids, *st.JobID)
				}
				where[*st.JobID] = append(where[*st.JobID], position{i, j})
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := s.pool.Query(ctx, `SELECT e.job_id,e.id,e.event,e.cycle,e.attempt,e.actor,e.actor_id,e.error_code,e.created_at,e.step,e.provider,e.model,e.revision,e.http_status,e.duration_ms,COALESCE(w.name,'')
FROM job_events e LEFT JOIN workers w ON e.actor='worker' AND w.id=e.actor_id WHERE e.job_id=ANY($1) ORDER BY e.job_id,e.id`, ids)
	if err != nil {
		return fmt.Errorf("list step events: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var job int64
		var ev StepEvent
		if err = rows.Scan(&job, &ev.ID, &ev.Event.Event, &ev.Cycle, &ev.Attempt, &ev.Actor, &ev.ActorID, &ev.ErrorCode, &ev.CreatedAt, &ev.Step, &ev.Provider, &ev.Model, &ev.Revision, &ev.HTTPStatus, &ev.DurationMS, &ev.WorkerName); err != nil {
			return err
		}
		if ev.Actor == "worker" {
			// Whatever a Worker reported is reduced to a catalogue code, never relayed.
			ev.ErrorCode = failureCodeOf(ev.ErrorCode, "")
		}
		ev.RecommendedAction = jobs.RecommendedAction(ev.ErrorCode)
		for _, at := range where[job] {
			runs[at.run].Steps[at.step].Events = append(runs[at.run].Steps[at.step].Events, ev)
		}
	}
	return rows.Err()
}

// Queue reads the whole queue at once.
func (s *Service) Queue(ctx context.Context) (Queue, error) {
	var q Queue
	var err error
	if q.Active, err = s.queueRuns(ctx, `r.status IN ('active','failed','cancelled')`, `r.created_at,r.id`, 0); err != nil {
		return q, err
	}
	if q.Completed, err = s.queueRuns(ctx, `r.status='completed'`, `r.finished_at DESC NULLS LAST,r.id DESC`, CompletedRuns); err != nil {
		return q, err
	}
	rows, err := s.pool.Query(ctx, `SELECT j.id,j.clip_id,j.step,j.input_version,j.status,j.priority,j.attempts,j.max_attempts,j.lease_owner,j.lease_expires_at,j.next_attempt_at,j.last_error,j.created_at,j.updated_at,j.cycle,COALESCE(c.display_title,c.original_name),`+latestFailureCode+`
FROM jobs j JOIN clips c ON c.id=j.clip_id
WHERE j.status IN ('pending','leased','retry_wait','dead','failed')
 AND j.id NOT IN (SELECT st.job_id FROM clip_analysis_steps st JOIN clip_analysis_runs r ON r.id=st.run_id WHERE st.job_id IS NOT NULL AND r.status IN ('active','failed','cancelled') AND r.id=(SELECT max(id) FROM clip_analysis_runs WHERE clip_id=r.clip_id))
 AND NOT EXISTS (SELECT 1 FROM jobs n WHERE n.clip_id=j.clip_id AND n.step=j.step AND n.id>j.id)
ORDER BY j.priority DESC,j.created_at,j.id`)
	if err != nil {
		return q, fmt.Errorf("list queue tasks: %w", err)
	}
	defer rows.Close()
	q.Tasks = []QueueJob{}
	for rows.Next() {
		var j QueueJob
		var eventCode string
		if err = rows.Scan(&j.ID, &j.ClipID, &j.Step, &j.InputVersion, &j.Status, &j.Priority, &j.Attempts, &j.MaxAttempts, &j.LeaseOwner, &j.LeaseExpiresAt, &j.NextAttemptAt, &j.LastError, &j.CreatedAt, &j.UpdatedAt, &j.Cycle, &j.ClipName, &eventCode); err != nil {
			return q, err
		}
		j.FailureCode = failureCodeOf(eventCode, j.LastError)
		j.RecommendedAction = jobs.RecommendedAction(j.FailureCode)
		j.LastError = j.FailureCode
		q.Tasks = append(q.Tasks, j)
	}
	if err = rows.Err(); err != nil {
		return q, err
	}
	rows.Close()
	q.Summary, err = s.queueSummary(ctx)
	return q, err
}

// queueSummary reads the load of the queue with the scheduler's own rule: only a leased job whose
// lease has not expired holds a slot.
func (s *Service) queueSummary(ctx context.Context) (QueueSummary, error) {
	var sum QueueSummary
	if err := s.pool.QueryRow(ctx, `SELECT paused,max_concurrency FROM job_control WHERE id=true`).Scan(&sum.Paused, &sum.MaxConcurrency); err != nil {
		return sum, fmt.Errorf("read queue controls: %w", err)
	}
	const live = `j.status='leased' AND j.lease_expires_at>now()`
	if err := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM jobs j WHERE `+live+`)::int,
 (SELECT count(*) FROM clip_analysis_runs r WHERE r.status='active' AND NOT EXISTS (SELECT 1 FROM jobs j WHERE j.clip_id=r.clip_id AND `+live+`))::int`).Scan(&sum.RunningTasks, &sum.WaitingClips); err != nil {
		return sum, fmt.Errorf("count queue load: %w", err)
	}
	return sum, nil
}
