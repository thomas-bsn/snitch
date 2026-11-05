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
const requestedModelClaimGuard = `(j.requested_model_id IS NULL OR EXISTS (
		      SELECT 1 FROM worker_voice_models wm
		      WHERE wm.worker_id=$4::text AND wm.model_id=j.requested_model_id AND wm.state='ready' AND NOT wm.removal_requested))`

// checkRequestedModel accepts a request only for a fingerprint step and a supported, enabled catalogue model of
// that kind. Another kind could replace the production result of the clip (a diarization replaces the local
// speakers), which a benchmark must never do.
func (s *Service) checkRequestedModel(ctx context.Context, step, modelID string) error {
	if modelID == "" {
		return nil
	}
	if !strings.HasPrefix(step, "voice.fingerprint.") {
		return ErrInvalidInput
	}
	var kind string
	err := s.pool.QueryRow(ctx, `SELECT kind FROM voice_model_catalog WHERE id=$1 AND enabled AND supported`, modelID).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && kind != "fingerprint") {
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
