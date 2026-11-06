package httpapi

import (
	"context"
	"encoding/json"

	"snitch/internal/httpapi/contract"
)

// Joignabilité du serveur de transcription, testée par chaque Worker et transmise dans son pouls (#212).
const (
	maxReachabilityURL     = 2048
	maxReachabilityExcerpt = 200
	maxReachabilityLatency = 600000
)

// validReachability rejects a payload outside the contract bounds, before anything is stored.
func validReachability(r *contract.WorkerTranscriptionReachabilityInput) bool {
	if r.LatencyMs < 0 || r.LatencyMs > maxReachabilityLatency || r.CheckedAt.IsZero() {
		return false
	}
	if r.LastFailure == nil {
		return true
	}
	f := r.LastFailure
	if f.Method != contract.GET || !f.ErrorType.Valid() || f.FailedAt.IsZero() {
		return false
	}
	if f.Url == "" || len(f.Url) > maxReachabilityURL || len(f.BodyExcerpt) > maxReachabilityExcerpt {
		return false
	}
	if f.LatencyMs < 0 || f.LatencyMs > maxReachabilityLatency {
		return false
	}
	return f.StatusCode == nil || (*f.StatusCode >= 100 && *f.StatusCode <= 599)
}

// recordTranscriptionReachability stores the Worker's latest result. The pulse carries the full state for
// the current target, so an absent failure clears the stored one, and an absent result clears the test.
func (s *Server) recordTranscriptionReachability(ctx context.Context, workerID string, r *contract.WorkerTranscriptionReachabilityInput) error {
	if r == nil {
		_, err := s.Pool.Exec(ctx, `UPDATE worker_transcription_config SET reachable=false,reach_checked_at=NULL,reach_latency_ms=0,reach_failure=NULL,updated_at=now() WHERE worker_id=$1`, workerID)
		return err
	}
	var failure []byte
	if r.LastFailure != nil {
		raw, err := json.Marshal(r.LastFailure)
		if err != nil {
			return err
		}
		failure = raw
	}
	_, err := s.Pool.Exec(ctx, `INSERT INTO worker_transcription_config(worker_id,reachable,reach_checked_at,reach_latency_ms,reach_failure) VALUES($1,$2,$3,$4,$5) ON CONFLICT(worker_id) DO UPDATE SET reachable=EXCLUDED.reachable,reach_checked_at=EXCLUDED.reach_checked_at,reach_latency_ms=EXCLUDED.reach_latency_ms,reach_failure=EXCLUDED.reach_failure,updated_at=now()`, workerID, r.Reachable, r.CheckedAt.UTC(), r.LatencyMs, failure)
	return err
}
