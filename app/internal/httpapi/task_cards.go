package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"snitch/internal/jobs"
	"snitch/internal/workers"
)

// Task cards (ADR 0024): one card per clip, placed by a single server-side rule.
// Applicable steps place a card and decide completion; they never make a clip
// enter the board, which only holds clips with at least one unfinished task.

const (
	columnTranscription = "transcription"
	columnSearch        = "search"
	columnVoices        = "voices"
	columnRecognition   = "recognition"
	columnCompleted     = "completed"

	stepTranscription = "transcription"
	stepDiarization   = "diarization"
	stepFingerprint   = "fingerprint"
	stepRecognition   = "recognition"

	cardQueued        = "queued"
	cardWaitingWorker = "waiting_worker"
	cardRunning       = "running"
	cardRetry         = "retry"
	cardFailed        = "failed"
	cardDead          = "dead"
	cardCancelled     = "cancelled"
	cardReady         = "ready"
	cardWaiting       = "waiting"
	cardDone          = "done"
	cardNotApplicable = "not_applicable"

	actionStartWorker = "start_worker"
	actionReschedule  = "reschedule"
	actionRelaunch    = "relaunch_job"

	completedCardLimit = 3
)

type taskCardStep struct {
	Key       string  `json:"key"`
	State     string  `json:"state"`
	JobID     *int64  `json:"job_id"`
	JobStatus *string `json:"job_status"`
}

type taskCard struct {
	ClipID        int64          `json:"clip_id"`
	OriginalName  string         `json:"original_name"`
	Column        string         `json:"column"`
	State         string         `json:"state"`
	CurrentStep   string         `json:"current_step"`
	Steps         []taskCardStep `json:"steps"`
	UpdatedAt     time.Time      `json:"updated_at"`
	CompletedAt   *time.Time     `json:"completed_at"`
	Worker        *string        `json:"worker"`
	JobID         *int64         `json:"job_id"`
	JobStatus     *string        `json:"job_status"`
	Attempt       int            `json:"attempt"`
	MaxAttempts   int            `json:"max_attempts"`
	NextAttemptAt *time.Time     `json:"next_attempt_at"`
	LastError     string         `json:"last_error"`
	// FailureCode and RecommendedAction come from the persisted job events (#122).
	FailureCode       *string         `json:"failure_code"`
	RecommendedAction *string         `json:"recommended_action"`
	UnknownStep       *unknownStepRef `json:"unknown_step,omitempty"`
}

// unknownStepRef is the unfinished task of an unknown step attached to a card.
type unknownStepRef struct {
	Step      string `json:"step"`
	State     string `json:"state"`
	JobID     int64  `json:"job_id"`
	LastError string `json:"last_error"`
}

type taskCardBoard struct {
	Cards     []taskCard `json:"cards"`
	Completed []taskCard `json:"completed"`
}

type cardJob struct {
	id          int64
	clipID      int64
	step        string
	status      string
	attempts    int
	maxAttempts int
	leaseOwner  *string
	nextAttempt time.Time
	lastError   string
	updatedAt   time.Time
	noWorker    bool
}

func (j cardJob) state() string {
	switch j.status {
	case "pending":
		if j.noWorker {
			return cardWaitingWorker
		}
		return cardQueued
	case "leased":
		return cardRunning
	case "retry_wait":
		return cardRetry
	case "failed":
		return cardFailed
	case "dead":
		return cardDead
	case "cancelled":
		return cardCancelled
	}
	return cardDone
}

// attention orders unfinished states: the job that carries a stage's state.
var cardAttention = map[string]int{cardRunning: 0, cardRetry: 1, cardQueued: 2, cardWaitingWorker: 2, cardFailed: 3, cardDead: 4, cardCancelled: 5}

type cardStage struct {
	key   string
	state string
	job   *cardJob
	done  []cardJob
}

func stageOfStep(step string) string {
	switch {
	case strings.HasPrefix(step, "transcription."):
		return stepTranscription
	case strings.HasPrefix(step, "voice.diarization."):
		return stepDiarization
	case strings.HasPrefix(step, "voice.fingerprint."):
		return stepFingerprint
	}
	return ""
}

func (s *Server) listTaskCards(w http.ResponseWriter, r *http.Request) {
	board, err := s.taskCardBoard(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, board)
}

func (s *Server) taskCardBoard(ctx context.Context) (taskCardBoard, error) {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT ON (clip_id,step) id,clip_id,step,status,attempts,max_attempts,lease_owner,next_attempt_at,last_error,updated_at FROM jobs WHERE step NOT LIKE 'voicetrial.%' ORDER BY clip_id,step,id DESC`)
	if err != nil {
		return taskCardBoard{}, err
	}
	defer rows.Close()
	capable, err := s.capableWorkerSteps(ctx)
	if err != nil {
		return taskCardBoard{}, err
	}
	byClip := map[int64][]cardJob{}
	for rows.Next() {
		var j cardJob
		if err = rows.Scan(&j.id, &j.clipID, &j.step, &j.status, &j.attempts, &j.maxAttempts, &j.leaseOwner, &j.nextAttempt, &j.lastError, &j.updatedAt); err != nil {
			return taskCardBoard{}, err
		}
		j.noWorker = j.status == "pending" && !capable(j.step)
		byClip[j.clipID] = append(byClip[j.clipID], j)
	}
	if err = rows.Err(); err != nil {
		return taskCardBoard{}, err
	}
	var active, finished []int64
	latest := map[int64]time.Time{}
	for id, list := range byClip {
		unfinished := false
		for _, j := range list {
			if j.status != "done" {
				unfinished = true
			}
			if j.updatedAt.After(latest[id]) {
				latest[id] = j.updatedAt
			}
		}
		if unfinished {
			active = append(active, id)
		} else {
			finished = append(finished, id)
		}
	}
	sort.Slice(active, func(a, b int) bool { return active[a] < active[b] })
	sort.Slice(finished, func(a, b int) bool {
		if !latest[finished[a]].Equal(latest[finished[b]]) {
			return latest[finished[a]].After(latest[finished[b]])
		}
		return finished[a] > finished[b]
	})
	board := taskCardBoard{Cards: []taskCard{}, Completed: []taskCard{}}
	names := map[int64]string{}
	nameRows, err := s.Pool.Query(ctx, `SELECT c.id,c.original_name FROM clips c WHERE EXISTS (SELECT 1 FROM jobs j WHERE j.clip_id=c.id)`)
	if err != nil {
		return taskCardBoard{}, err
	}
	for nameRows.Next() {
		var id int64
		var name string
		if err = nameRows.Scan(&id, &name); err != nil {
			nameRows.Close()
			return taskCardBoard{}, err
		}
		names[id] = name
	}
	nameRows.Close()
	if err = nameRows.Err(); err != nil {
		return taskCardBoard{}, err
	}
	for _, id := range active {
		card, complete, err := s.buildTaskCard(ctx, id, names[id], byClip[id])
		if err != nil {
			return taskCardBoard{}, err
		}
		if !complete {
			board.Cards = append(board.Cards, card)
		}
	}
	for _, id := range finished {
		if len(board.Completed) == completedCardLimit {
			break
		}
		card, complete, err := s.buildTaskCard(ctx, id, names[id], byClip[id])
		if err != nil {
			return taskCardBoard{}, err
		}
		if complete {
			board.Completed = append(board.Completed, card)
		}
	}
	if err = s.explainFailures(ctx, &board); err != nil {
		return taskCardBoard{}, err
	}
	return board, nil
}

// capableWorkerSteps returns the claim rule over the Workers that are not
// revoked, were seen recently and can take work: a capability ending in ".*"
// is a prefix, any other is an exact step.
func (s *Server) capableWorkerSteps(ctx context.Context) (func(string) bool, error) {
	rows, err := s.Pool.Query(ctx, `SELECT supported_steps FROM workers WHERE revoked_at IS NULL AND last_seen_at >= now() - ($1 * interval '1 second') AND LEAST(max_concurrency,reported_concurrency) >= 1`, int64(workers.ConnectedWindow.Seconds()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var capabilities []string
	for rows.Next() {
		var steps []string
		if err = rows.Scan(&steps); err != nil {
			return nil, err
		}
		capabilities = append(capabilities, steps...)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return func(step string) bool {
		for _, capability := range capabilities {
			if prefix, ok := strings.CutSuffix(capability, ".*"); ok {
				if strings.HasPrefix(step, prefix+".") {
					return true
				}
			} else if step == capability {
				return true
			}
		}
		return false
	}, nil
}

// explainFailures fills the cause and the advised action of every card from the
// latest persisted failure event of its current task (one query for the board).
func (s *Server) explainFailures(ctx context.Context, board *taskCardBoard) error {
	var ids []int64
	for _, list := range [][]taskCard{board.Cards, board.Completed} {
		for _, card := range list {
			if card.JobID != nil {
				ids = append(ids, *card.JobID)
			}
		}
	}
	causes := map[int64]string{}
	if len(ids) > 0 {
		rows, err := s.Pool.Query(ctx, `SELECT DISTINCT ON (job_id) job_id,error_code FROM job_events WHERE job_id = ANY($1) AND error_code <> '' ORDER BY job_id,id DESC`, ids)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var code string
			if err = rows.Scan(&id, &code); err != nil {
				return err
			}
			causes[id] = code
		}
		if err = rows.Err(); err != nil {
			return err
		}
	}
	for i := range board.Cards {
		card := &board.Cards[i]
		switch card.State {
		case cardWaitingWorker:
			action := actionStartWorker
			card.RecommendedAction = &action
		case cardRetry, cardFailed, cardDead, cardCancelled:
			var action string
			if card.JobID != nil {
				if code, ok := causes[*card.JobID]; ok {
					c := code
					card.FailureCode = &c
					action = jobs.RecommendedAction(code)
				}
			}
			switch {
			case card.State == cardCancelled:
				action = actionReschedule
			case card.State == cardDead && action == "":
				action = actionRelaunch
			}
			if action != "" {
				card.RecommendedAction = &action
			}
		}
	}
	return nil
}

// buildTaskCard derives the stages of one clip. complete is true when every
// applicable stage is finished and no task is left unfinished.
func (s *Server) buildTaskCard(ctx context.Context, clipID int64, name string, jobs []cardJob) (taskCard, bool, error) {
	tracks, err := s.clipAudioTracks(ctx, clipID)
	if err != nil {
		tracks = nil
	}
	var selected *int
	if len(tracks) > 0 {
		selection, selErr := s.audioSelection(ctx, clipID, tracks)
		if selErr != nil {
			return taskCard{}, false, selErr
		}
		if selection.Status == "selected" {
			selected = selection.AudioStreamIndex
		}
	}
	microphoneTrack := false
	if selected != nil {
		designation, micErr := s.resolveMicrophone(ctx, s.Pool, clipID, tracks)
		if micErr != nil {
			return taskCard{}, false, micErr
		}
		microphoneTrack = designation.Status == microphoneDesigned && designation.Index != nil && *designation.Index == *selected
	}
	stages := map[string]*cardStage{
		stepTranscription: {key: stepTranscription},
		stepDiarization:   {key: stepDiarization},
		stepFingerprint:   {key: stepFingerprint},
	}
	// Steps of an unknown family stay attached to the furthest stage that has a task.
	var unknown []cardJob
	var newest time.Time
	for _, j := range jobs {
		if j.updatedAt.After(newest) {
			newest = j.updatedAt
		}
		if key := stageOfStep(j.step); key != "" {
			stages[key].add(j)
		} else {
			unknown = append(unknown, j)
		}
	}
	var unknownRef *unknownStepRef
	for _, j := range unknown {
		if j.status != "done" && (unknownRef == nil || cardAttention[j.state()] < cardAttention[unknownRef.State]) {
			unknownRef = &unknownStepRef{Step: j.step, State: j.state(), JobID: j.id, LastError: j.lastError}
		}
	}
	if len(unknown) > 0 {
		home := stages[stepTranscription]
		for _, key := range []string{stepDiarization, stepFingerprint} {
			if stages[key].job != nil || len(stages[key].done) > 0 {
				home = stages[key]
			}
		}
		for _, j := range unknown {
			home.add(j)
		}
	}
	transcription, diarization, fingerprint := stages[stepTranscription], stages[stepDiarization], stages[stepFingerprint]
	transcription.settle(true, cardReady)
	diarizationApplicable := selected != nil && !microphoneTrack && transcription.state == cardDone
	diarization.settle(diarizationApplicable, cardReady)
	speakers := 0
	if diarization.state == cardDone && selected != nil {
		if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM local_speakers WHERE clip_id=$1 AND audio_stream_index=$2 AND kind='diarized'`, clipID, *selected).Scan(&speakers); err != nil {
			return taskCard{}, false, err
		}
	}
	fingerprint.settle(diarization.state == cardDone && speakers > 0, cardReady)
	// Voice recognition is not a task: it runs right after a fingerprint finishes
	// and keeps no state of its own, so it is derived from the fingerprints.
	recognition := &cardStage{key: stepRecognition, state: cardNotApplicable}
	if fingerprint.state != cardNotApplicable {
		recognition.state = cardWaiting
		if fingerprint.state == cardDone {
			recognition.state = cardDone
		}
	}
	ordered := []*cardStage{transcription, diarization, fingerprint, recognition}
	card := taskCard{UnknownStep: unknownRef, ClipID: clipID, OriginalName: name, Steps: []taskCardStep{}, UpdatedAt: newest, CurrentStep: stepTranscription, Column: columnTranscription, State: cardDone}
	var current *cardStage
	for _, stage := range ordered {
		entry := taskCardStep{Key: stage.key, State: stage.state}
		if stage.job != nil {
			id, status := stage.job.id, stage.job.status
			entry.JobID, entry.JobStatus = &id, &status
		}
		card.Steps = append(card.Steps, entry)
		if current == nil && stage.state != cardDone && stage.state != cardNotApplicable {
			current = stage
		}
	}
	if current == nil {
		completed := newest
		card.Column, card.State, card.CurrentStep, card.CompletedAt = columnCompleted, cardDone, stepRecognition, &completed
		return card, true, nil
	}
	card.CurrentStep, card.State = current.key, current.state
	card.Column = columnTranscription
	if current.key == stepDiarization || current.key == stepFingerprint {
		card.Column = columnVoices
	} else if current.key == stepRecognition {
		card.Column = columnRecognition
	}
	if j := current.job; j != nil {
		id, status, owner, next := j.id, j.status, j.leaseOwner, j.nextAttempt
		card.JobID, card.JobStatus, card.Worker, card.Attempt, card.MaxAttempts, card.LastError = &id, &status, owner, j.attempts, j.maxAttempts, j.lastError
		if j.status == "retry_wait" || j.status == "pending" {
			card.NextAttemptAt = &next
		}
	}
	return card, false, nil
}

func (c *cardStage) add(j cardJob) {
	if j.status == "done" {
		c.done = append(c.done, j)
		return
	}
	if c.job == nil || cardAttention[j.state()] < cardAttention[c.job.state()] {
		job := j
		c.job = &job
	}
}

// settle fixes the stage state: an unfinished task always shows itself, even
// when the step no longer applies; without one, an applicable stage is ready
// and an inapplicable one is not applicable (or done when it has finished tasks).
func (c *cardStage) settle(applicable bool, idle string) {
	switch {
	case c.job != nil:
		c.state = c.job.state()
	case len(c.done) > 0:
		c.state = cardDone
	case applicable:
		c.state = idle
	default:
		c.state = cardNotApplicable
	}
}
