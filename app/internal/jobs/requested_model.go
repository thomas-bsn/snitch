package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// requestedModelClaimGuard is the Claim condition for the requested model of a job (ADR 0038, benchmark). It is
// the Worker $4 that must hold the requested model, in the real state it reported (ready, not being removed),
// whatever it declares. A job without a requested model is not concerned here: the active model rule applies.
//
// Since #350 the model also needs its pinned revision and, for an engine that needs one, its approved runtime
// ready on that same Worker (RuntimeReadyClause).
var requestedModelClaimGuard = `(j.requested_model_id IS NULL OR j.step LIKE 'benchmark.transcription.%' OR EXISTS (
		      SELECT 1 FROM worker_voice_models wm JOIN voice_model_catalog rm ON rm.id=wm.model_id
		      WHERE wm.worker_id=$4::text AND wm.model_id=j.requested_model_id AND wm.state='ready' AND NOT wm.removal_requested
		        AND ` + RuntimeReadyClause("wm", "rm") + `))`

// checkRequestedModel accepts a request only for a fingerprint step and a supported, enabled catalogue model of
// that kind. Another kind could replace the production result of the clip (a diarization replaces the local
// speakers), which a benchmark must never do.
func (s *Service) checkRequestedModel(ctx context.Context, step, modelID string) error {
	if modelID == "" {
		return nil
	}
	if strings.HasPrefix(step, BenchmarkTranscriptionPrefix) {
		// A transcription model is a name known to the external service, not a catalogue entry (ADR 0058).
		return checkTranscriptionModelName(modelID)
	}
	// A diarization benchmark has a step of its own: the production diarization replaces the local speakers of a
	// clip, which a benchmark must never do (ADR 0059).
	wanted := "fingerprint"
	switch {
	case strings.HasPrefix(step, BenchmarkDiarizationPrefix):
		wanted = "diarization"
	case !strings.HasPrefix(step, "voice.fingerprint."):
		return ErrInvalidInput
	}
	var kind string
	err := s.pool.QueryRow(ctx, `SELECT kind FROM voice_model_catalog WHERE id=$1 AND enabled AND supported`, modelID).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && kind != wanted) {
		return ErrInvalidInput
	}
	if err != nil {
		return fmt.Errorf("read requested model: %w", err)
	}
	return nil
}

// RequestedModelRepo is the repository of the model a job asked for, or "" when it asked for none. A request
// whose catalogue entry has disappeared yields the raw id, which no result can match.
func RequestedModelRepo(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, jobID int64) (string, error) {
	var repo string
	err := q.QueryRow(ctx, `SELECT COALESCE(c.repo,j.requested_model_id,'') FROM jobs j LEFT JOIN voice_model_catalog c ON c.id=j.requested_model_id WHERE j.id=$1`, jobID).Scan(&repo)
	if err != nil {
		return "", fmt.Errorf("read requested model of job: %w", err)
	}
	return repo, nil
}

func withRequestedModel(job Job, modelID string) Job {
	if modelID != "" {
		job.RequestedModelID = &modelID
	}
	return job
}

// BenchmarkTranscriptionPrefix is the step of a transcription benchmark job. It is deliberately not
// "transcription.audio.": the production readers of a clip's transcript, the embeddings and the corrections
// never see it.
const BenchmarkTranscriptionPrefix = "benchmark.transcription."

// checkTranscriptionModelName accepts a model name as the transcription service lists it: printable, no space
// at either end, at most 64 characters (the column limit).
func checkTranscriptionModelName(name string) error {
	if name == "" || len(name) > 64 || strings.TrimSpace(name) != name {
		return ErrInvalidInput
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return ErrInvalidInput
		}
	}
	return nil
}

// ValidTranscriptionModelName tells whether a name may be requested for a transcription benchmark.
func ValidTranscriptionModelName(name string) bool { return checkTranscriptionModelName(name) == nil }

// BenchmarkDiarizationPrefix is the step of a diarization benchmark job (ADR 0059), never "voice.diarization.".
const BenchmarkDiarizationPrefix = "benchmark.diarization."
