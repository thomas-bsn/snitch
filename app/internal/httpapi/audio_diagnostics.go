package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
	"snitch/internal/jobs"
)

// maxAudioDiagnosticResultBytes bounds the intervals of a very long, speech-dense stream.
const maxAudioDiagnosticResultBytes = 1 << 20

// queueAudioDiagnostics queues one diagnostic job per selected track of an immutable plan (ADR 0048). It is
// idempotent and has no effect on any transcript. Which Worker takes the jobs is decided by the claim: only a
// Worker that supports the step AND holds the pinned detector (explicitly installed) can lease one.
func (s *Server) queueAudioDiagnostics(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("plan_id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_plan_id")
		return
	}
	store := audioplans.New(s.Pool)
	state, err := store.GetPlan(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && state.Plan.ClipId != clip.ID {
		fail(w, 404, "audio_plan_not_found")
		return
	}
	if err == nil {
		_, err = store.QueueDiagnostics(r.Context(), id)
	}
	if err == nil {
		state, err = store.GetPlan(r.Context(), id)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, state)
}

type audioDiagnosticResult struct {
	Status                string                         `json:"status"`
	Cause                 string                         `json:"cause"`
	Intervals             []contract.AudioSpeechInterval `json:"intervals"`
	DetectorVersion       string                         `json:"detector_version"`
	DetectorConfigVersion string                         `json:"detector_config_version"`
}

// decodeAudioDiagnosticResult accepts only an observation, never a decision: unexamined is not reportable by a
// Worker, and a failure keeps its cause.
func decodeAudioDiagnosticResult(raw json.RawMessage) (audioDiagnosticResult, bool) {
	var result audioDiagnosticResult
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Cause) == 0 || len(result.Cause) > 64 || len(result.Intervals) > 20000 {
		return result, false
	}
	switch result.Status {
	case "speech", "no_speech", "ambiguous", "failed":
	default:
		return result, false
	}
	for _, interval := range result.Intervals {
		if math.IsNaN(interval.Start) || math.IsNaN(interval.End) || math.IsInf(interval.Start, 0) || math.IsInf(interval.End, 0) {
			return result, false
		}
	}
	if result.Intervals == nil {
		result.Intervals = []contract.AudioSpeechInterval{}
	}
	return result, true
}

// audioDiagnosticProjection records the immutable observation in the completion transaction.
func audioDiagnosticProjection(result audioDiagnosticResult) func(context.Context, pgx.Tx, jobs.Job) error {
	return func(ctx context.Context, tx pgx.Tx, job jobs.Job) error {
		var planID int64
		var track int
		var version string
		err := tx.QueryRow(ctx, `SELECT c.plan_id,c.track_index,p.input_version FROM audio_plan_children c JOIN audio_plans p ON p.id=c.plan_id WHERE c.job_id=$1 AND c.stage='diagnostic'`, job.ID).Scan(&planID, &track, &version)
		if err != nil {
			return audioplans.ErrConflict
		}
		jobID := job.ID
		return audioplans.New(tx).RecordDiagnostic(ctx, contract.AudioTrackDiagnostic{
			PlanId: planID, TrackIndex: track, InputVersion: version, Status: contract.AudioTrackDiagnosticStatus(result.Status), Cause: result.Cause,
			DetectorVersion: result.DetectorVersion, DetectorConfigVersion: result.DetectorConfigVersion, Intervals: result.Intervals, JobId: &jobID, Attempt: job.Attempts,
		})
	}
}
