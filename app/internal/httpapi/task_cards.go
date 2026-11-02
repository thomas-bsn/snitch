package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"
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

	cardQueued        = "queued"
	cardRunning       = "running"
	cardRetry         = "retry"
	cardFailed        = "failed"
	cardDead          = "dead"
	cardCancelled     = "cancelled"
	cardReady         = "ready"
	cardWaiting       = "waiting"
	cardDone          = "done"
	cardNotApplicable = "not_applicable"

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
	// VoicesToReview is only computed for a completed clip: a diarized voice with
	// no human validation, automatic attribution or suggestion remains to settle.
	VoicesToReview bool `json:"voices_to_review"`
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
}

func (j cardJob) state() string {
	switch j.status {
	case "pending":
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
var cardAttention = map[string]int{cardRunning: 0, cardRetry: 1, cardQueued: 2, cardFailed: 3, cardDead: 4, cardCancelled: 5}

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
	indexing, err := s.cardIndexing(ctx)
	if err != nil {
		return taskCardBoard{}, err
	}
	byClip := map[int64][]cardJob{}
	for rows.Next() {
		var j cardJob
		if err = rows.Scan(&j.id, &j.clipID, &j.step, &j.status, &j.attempts, &j.maxAttempts, &j.leaseOwner, &j.nextAttempt, &j.lastError, &j.updatedAt); err != nil {
			return taskCardBoard{}, err
		}
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
		// A waiting, running or failed indexing is an unfinished task too.
		if idx, ok := indexing[id]; ok {
			if idx.state != cardDone {
				unfinished = true
			}
			if idx.updatedAt.After(latest[id]) {
				latest[id] = idx.updatedAt
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
		card, complete, err := s.buildTaskCard(ctx, id, names[id], byClip[id], indexing[id])
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
		card, complete, err := s.buildTaskCard(ctx, id, names[id], byClip[id], indexing[id])
		if err != nil {
			return taskCardBoard{}, err
		}
		if complete {
			board.Completed = append(board.Completed, card)
		}
	}
	return board, nil
}

// buildTaskCard derives the stages of one clip. complete is true when every
// applicable stage is finished and no task is left unfinished.
func (s *Server) buildTaskCard(ctx context.Context, clipID int64, name string, jobs []cardJob, index cardIndex) (taskCard, bool, error) {
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
	card := taskCard{ClipID: clipID, OriginalName: name, Steps: []taskCardStep{}, UpdatedAt: newest, CurrentStep: stepTranscription, Column: columnTranscription, State: cardDone}
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
		if err = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM local_speakers ls WHERE ls.clip_id=$1 AND `+unknownSpeakerPredicate+` AND NOT EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.decision_level='suggestion'))`, clipID).Scan(&card.VoicesToReview); err != nil {
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
		default:
			idx.state = cardQueued
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
