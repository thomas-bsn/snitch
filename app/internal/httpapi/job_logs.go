package httpapi

import (
	"context"
	"errors"
	"net/http"

	"snitch/internal/events"
	"snitch/internal/jobs"
)

// appendWorkerJobLogs receives a batch of log lines from the Worker holding the job's lease.
func (s *Server) appendWorkerJobLogs(w http.ResponseWriter, r *http.Request) {
	id, ok := workerJobID(w, r)
	if !ok {
		return
	}
	var input struct {
		Lines []string `json:"lines"`
	}
	if !decode(w, r, &input) {
		return
	}
	if len(input.Lines) < 1 || len(input.Lines) > jobs.KeptLogLines {
		fail(w, http.StatusBadRequest, "invalid_log_batch")
		return
	}
	for _, line := range input.Lines {
		if len(line) < 1 || len(line) > 2000 {
			fail(w, http.StatusBadRequest, "invalid_log_line")
			return
		}
	}
	stored, err := s.Jobs.AppendLogs(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r), input.Lines)
	if errors.Is(err, jobs.ErrLeaseNotOwned) {
		fail(w, http.StatusConflict, "lease_not_owned")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]int{"stored": stored})
}

// jobLogsEvents streams the last log lines of a job to administrators: a snapshot, then one per
// change. The administrator role is rechecked before every snapshot.
func (s *Server) jobLogsEvents(w http.ResponseWriter, r *http.Request) {
	id, ok := workerJobID(w, r)
	if !ok {
		return
	}
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM jobs WHERE id=$1)`, id).Scan(&exists); err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, http.StatusNotFound, "job_not_found")
		return
	}
	s.Events.Stream(w, r, events.Topic{
		Name:      jobs.JobLogsTopic(id),
		Authorize: s.adminAuthorizer(r),
		Snapshot: func(ctx context.Context) (any, error) {
			lines, err := s.Jobs.LogSnapshot(ctx, id)
			return map[string]any{"job_id": id, "lines": lines}, err
		},
	})
}
