package httpapi

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"snitch/internal/jobs"
	"snitch/internal/logging"
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
	stepIndexing      = "indexing"
	stepDiarization   = "diarization"
	stepFingerprint   = "fingerprint"
	stepRecognition   = "recognition"

	// stepDerivationFailed marks the alert of a card that could not be computed.
	stepDerivationFailed = "derivation_failed"

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
	JobID     *int64  `json:"job_id,omitempty"`
	JobStatus *string `json:"job_status,omitempty"`
}

type taskCard struct {
	ClipID        int64          `json:"clip_id"`
	OriginalName  string         `json:"original_name"`
	Column        string         `json:"column"`
	State         string         `json:"state"`
	CurrentStep   string         `json:"current_step"`
	Steps         []taskCardStep `json:"steps"`
	UpdatedAt     time.Time      `json:"updated_at"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
	Worker        *string        `json:"worker,omitempty"`
	JobID         *int64         `json:"job_id,omitempty"`
	JobStatus     *string        `json:"job_status,omitempty"`
	Attempt       int            `json:"attempt"`
	MaxAttempts   int            `json:"max_attempts"`
	NextAttemptAt *time.Time     `json:"next_attempt_at,omitempty"`
	LastError     string         `json:"last_error"`
	// VoicesToReview is only computed for a completed clip: a diarized voice with
	// no human validation, automatic attribution or suggestion remains to settle.
	VoicesToReview bool `json:"voices_to_review"`
	// FailureCode and RecommendedAction come from the persisted job events (#122).
	FailureCode       *string         `json:"failure_code,omitempty"`
	RecommendedAction *string         `json:"recommended_action,omitempty"`
	UnknownStep       *unknownStepRef `json:"unknown_step,omitempty"`
}

// unknownStepRef is the unfinished task of an unknown step attached to a card.
type unknownStepRef struct {
	Step      string `json:"step"`
	State     string `json:"state"`
	JobID     int64  `json:"job_id,omitempty"`
	LastError string `json:"last_error"`
}

type taskQueueSummary struct {
	Running        int  `json:"running"`
	Waiting        int  `json:"waiting"`
	MaxConcurrency int  `json:"max_concurrency"`
	Paused         bool `json:"paused"`
}

type taskCardBoard struct {
	Queue     taskQueueSummary `json:"queue"`
	Cards     []taskCard       `json:"cards"`
	Completed []taskCard       `json:"completed"`
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
	// The board is polled and large: compress it for clients that accept it.
	if acceptsGzip(r) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusOK)
		zw := gzip.NewWriter(w)
		_ = json.NewEncoder(zw).Encode(board)
		_ = zw.Close()
		return
	}
	reply(w, http.StatusOK, board)
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		if name, _, _ := strings.Cut(strings.TrimSpace(part), ";"); strings.EqualFold(name, "gzip") {
			return true
		}
	}
	return false
}

func (s *Server) taskCardBoard(ctx context.Context) (taskCardBoard, error) {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT ON (clip_id,step) id,clip_id,step,status,attempts,max_attempts,lease_owner,next_attempt_at,last_error,updated_at,next_attempt_at>now() FROM jobs WHERE step NOT LIKE 'voicetrial.%' ORDER BY clip_id,step,id DESC`)
	if err != nil {
		return taskCardBoard{}, err
	}
	defer rows.Close()
	indexing, err := s.cardIndexing(ctx)
	if err != nil {
		return taskCardBoard{}, err
	}
	capable, err := s.capableWorkerSteps(ctx)
	if err != nil {
		return taskCardBoard{}, err
	}
	byClip := map[int64][]cardJob{}
	for rows.Next() {
		var j cardJob
		var deferred bool
		if err = rows.Scan(&j.id, &j.clipID, &j.step, &j.status, &j.attempts, &j.maxAttempts, &j.leaseOwner, &j.nextAttempt, &j.lastError, &j.updatedAt, &deferred); err != nil {
			return taskCardBoard{}, err
		}
		// A deferred retry is not claimable yet, so it waits for time, not for a Worker.
		j.noWorker = j.status == "pending" && !deferred && !capable(j.step)
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
		}
		// A waiting, running or failed indexing is an unfinished task too.
		if idx, ok := indexing[id]; ok && idx.state != cardDone {
			unfinished = true
		}
		if unfinished {
			active = append(active, id)
		} else {
			finished = append(finished, id)
			latest[id] = completionTime(list, indexing[id])
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
	clipIDs := make([]int64, 0, len(byClip))
	for id := range byClip {
		clipIDs = append(clipIDs, id)
	}
	inputs, err := s.loadCardInputs(ctx, clipIDs)
	if err != nil {
		return taskCardBoard{}, err
	}
	for _, id := range active {
		card, complete, err := s.safeTaskCard(ctx, inputs, id, byClip[id], indexing[id])
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
		card, complete, err := s.safeTaskCard(ctx, inputs, id, byClip[id], indexing[id])
		if err != nil {
			return taskCardBoard{}, err
		}
		if complete {
			board.Completed = append(board.Completed, card)
		} else if card.UnknownStep != nil && card.UnknownStep.Step == stepDerivationFailed {
			// A finished clip whose card cannot be computed is never dropped silently.
			board.Cards = append(board.Cards, card)
		}
	}
	if err = s.explainFailures(ctx, &board); err != nil {
		return taskCardBoard{}, err
	}
	controls, err := s.Jobs.GetControls(ctx)
	if err != nil {
		return taskCardBoard{}, err
	}
	board.Queue.MaxConcurrency, board.Queue.Paused = controls.MaxConcurrency, controls.Paused
	// Same rule as the scheduler: only a leased task with a valid lease fills the ceiling.
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE status='leased' AND lease_expires_at>now()`).Scan(&board.Queue.Running); err != nil {
		return taskCardBoard{}, err
	}
	for _, card := range board.Cards {
		switch card.State {
		case cardQueued, cardWaitingWorker, cardRetry, cardReady, cardWaiting:
			board.Queue.Waiting++
		}
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

// safeTaskCard derives a card; when the derivation of that one clip fails, the
// board is not lost: the clip stays visible with an explicit alert, never
// completed, and the failure is logged with the clip id and no raw text.
func (s *Server) safeTaskCard(ctx context.Context, in *cardInputs, clipID int64, jobs []cardJob, index cardIndex) (taskCard, bool, error) {
	card, complete, err := s.buildTaskCard(ctx, in, clipID, jobs, index)
	if err == nil {
		return card, complete, nil
	}
	if ctx.Err() != nil {
		return taskCard{}, false, err
	}
	if s.Log != nil {
		s.Log.Error("task card derivation", append([]any{"clip_id", clipID}, logging.Failure(err)...)...)
	}
	var newest time.Time
	for _, j := range jobs {
		if j.updatedAt.After(newest) {
			newest = j.updatedAt
		}
	}
	return taskCard{
		ClipID: clipID, OriginalName: in.names[clipID], Column: columnTranscription, State: cardFailed,
		CurrentStep: stepTranscription, Steps: []taskCardStep{}, UpdatedAt: newest,
		UnknownStep: &unknownStepRef{Step: stepDerivationFailed, State: cardFailed},
	}, false, nil
}

// buildTaskCard derives the stages of one clip. complete is true when every
// applicable stage is finished and no task is left unfinished.
func (s *Server) buildTaskCard(ctx context.Context, in *cardInputs, clipID int64, jobs []cardJob, index cardIndex) (taskCard, bool, error) {
	if s.cardFault != nil {
		if err := s.cardFault(clipID); err != nil {
			return taskCard{}, false, err
		}
	}
	name := in.names[clipID]
	var selected *int
	// A pending selection (ADR 0011) does not make the voice steps not applicable:
	// once a track is chosen they apply, so the clip is not finished meanwhile.
	selectionPending := false
	if len(in.tracks[clipID]) > 0 {
		if selection := in.selection(clipID); selection.Status == "selected" {
			selected = selection.AudioStreamIndex
		} else {
			selectionPending = true
		}
	}
	microphoneTrack := false
	if selected != nil {
		designation, micErr := in.microphone(clipID)
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
	diarizationApplicable := (selected != nil || selectionPending) && !microphoneTrack && transcription.state == cardDone
	diarization.settle(diarizationApplicable, cardReady)
	speakers := 0
	if diarization.state == cardDone {
		if selected != nil {
			speakers = in.diarizedCount[[2]int64{clipID, int64(*selected)}]
		} else if selectionPending {
			for key, count := range in.diarizedCount {
				if key[0] == clipID {
					speakers += count
				}
			}
		}
	}
	// Indexing is not a task of the job queue: its state is the index state.
	indexingStage := &cardStage{key: stepIndexing, state: cardNotApplicable}
	if index.state != "" {
		indexingStage.state = index.state
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
	ordered := []*cardStage{transcription, indexingStage, diarization, fingerprint, recognition}
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
		completed := completionTime(jobs, index)
		card.Column, card.State, card.CurrentStep, card.CompletedAt = columnCompleted, cardDone, stepRecognition, &completed
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM local_speakers ls WHERE ls.clip_id=$1 AND `+unknownSpeakerPredicate+` AND NOT EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.decision_level='suggestion'))`, clipID).Scan(&card.VoicesToReview); err != nil {
			return taskCard{}, false, err
		}
		return card, true, nil
	}
	card.CurrentStep, card.State = current.key, current.state
	card.Column = columnTranscription
	if current.key == stepIndexing {
		card.Column = columnSearch
	} else if current.key == stepDiarization || current.key == stepFingerprint {
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

// completionTime is the end of the last finished step, indexing included. It
// is the one date both the order and the display of the completed column use,
// and it never comes from a failed or cancelled task.
func completionTime(list []cardJob, index cardIndex) time.Time {
	var at time.Time
	for _, j := range list {
		if j.status == "done" && j.updatedAt.After(at) {
			at = j.updatedAt
		}
	}
	if index.state == cardDone && index.updatedAt.After(at) {
		at = index.updatedAt
	}
	return at
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

// cardIndex is the semantic indexing of a clip's visible transcription. An empty
// state means the step does not apply: indexing disabled or no transcription.
type cardIndex struct {
	state     string
	updatedAt time.Time
}

// cardIndexing reads the index state of every clip with a visible transcription
// in one query, with the rule of clipProgress: disabled when the embeddings URL
// is empty or no profile is active; otherwise the state of the task of the
// active profile for the visible transcription, a missing task being queued.
func (s *Server) cardIndexing(ctx context.Context) (map[int64]cardIndex, error) {
	out := map[int64]cardIndex{}
	settings, err := s.embeddingSettings(ctx)
	if err != nil || settings.URL == "" {
		return out, err
	}
	var profileID *int64
	if err = s.Pool.QueryRow(ctx, `SELECT active_profile_id FROM embedding_index_state WHERE id=1`).Scan(&profileID); err != nil || profileID == nil {
		return out, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT v.clip_id,COALESCE(t.state,''),COALESCE(t.updated_at,v.updated_at),
		EXISTS(SELECT 1 FROM segment_corrections c WHERE c.transcription_job_id=v.id AND c.reindex_pending AND c.stale_at IS NULL)
		FROM (SELECT DISTINCT ON (clip_id) clip_id,id,updated_at FROM jobs WHERE step LIKE 'transcription.audio.%' AND status='done' ORDER BY clip_id,updated_at DESC,id DESC) v
		LEFT JOIN embedding_tasks t ON t.profile_id=$1 AND t.transcription_job_id=v.id`, *profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var clipID int64
		var state string
		var updated time.Time
		var reindex bool
		if err = rows.Scan(&clipID, &state, &updated, &reindex); err != nil {
			return nil, err
		}
		idx := cardIndex{updatedAt: updated}
		switch state {
		case "done":
			idx.state = cardDone
			if reindex {
				idx.state = cardQueued
			}
		case "leased":
			idx.state = cardRunning
		case "failed":
			idx.state = cardFailed
		case "dead":
			idx.state = cardDead
		case "cancelled":
			idx.state = cardCancelled
		case "pending", "":
			idx.state = cardQueued
		default:
			// An unknown state is an alert, never silently a queued task.
			idx.state = cardFailed
		}
		out[clipID] = idx
	}
	return out, rows.Err()
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
