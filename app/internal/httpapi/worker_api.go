package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
	"snitch/internal/workers"
)

type workerIdentityKey struct{}

func (s *Server) workerOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") {
			fail(w, http.StatusUnauthorized, "worker_token_required")
			return
		}
		token := strings.TrimPrefix(authorization, "Bearer ")
		if token == "" || strings.ContainsAny(token, " \t\r\n") {
			fail(w, http.StatusUnauthorized, "worker_token_required")
			return
		}
		worker, err := s.Workers.Authenticate(r.Context(), token)
		if errors.Is(err, workers.ErrUnauthorized) {
			fail(w, http.StatusUnauthorized, "worker_token_invalid")
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
		if err = s.Workers.Touch(r.Context(), worker.ID, worker.ReportedConcurrency); err != nil {
			s.internal(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(withWorker(r, worker)))
	})
}

func withWorker(r *http.Request, worker workers.Worker) context.Context {
	return context.WithValue(r.Context(), workerIdentityKey{}, worker)
}

func currentWorker(r *http.Request) workers.Worker {
	return r.Context().Value(workerIdentityKey{}).(workers.Worker)
}

func (s *Server) listWorkers(w http.ResponseWriter, r *http.Request) {
	items, err := s.Workers.List(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) registerWorker(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name           string `json:"name"`
		MaxConcurrency int    `json:"max_concurrency"`
	}
	if !decode(w, r, &input) {
		return
	}
	registration, err := s.Workers.Register(r.Context(), workers.Input{Name: input.Name, MaxConcurrency: input.MaxConcurrency})
	if errors.Is(err, workers.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_worker")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusCreated, registration)
}

func (s *Server) revokeWorker(w http.ResponseWriter, r *http.Request) {
	err := s.Workers.Revoke(r.Context(), r.PathValue("id"))
	if errors.Is(err, workers.ErrNotFound) {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) workerPulse(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Capacity       int      `json:"capacity"`
		SupportedSteps []string `json:"supported_steps"`
	}
	if !decode(w, r, &input) {
		return
	}
	worker := currentWorker(r)
	if input.Capacity < 1 || input.Capacity > worker.MaxConcurrency {
		fail(w, http.StatusBadRequest, "invalid_worker_capacity")
		return
	}
	if err := s.Workers.Pulse(r.Context(), worker.ID, input.Capacity, input.SupportedSteps); err != nil {
		if errors.Is(err, workers.ErrUnauthorized) {
			fail(w, http.StatusUnauthorized, "worker_token_invalid")
		} else if errors.Is(err, workers.ErrInvalidInput) {
			fail(w, http.StatusBadRequest, "invalid_worker_capabilities")
		} else {
			s.internal(w, err)
		}
		return
	}
	worker.ReportedConcurrency = input.Capacity
	now := time.Now().UTC()
	worker.LastSeenAt = &now
	worker.Connected = true
	reply(w, http.StatusOK, worker)
}

func (s *Server) claimWorkerJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.Jobs.Claim(r.Context(), currentWorker(r).ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if job == nil {
		reply(w, http.StatusOK, map[string]any{"job": nil, "lease_token": ""})
		return
	}
	reply(w, http.StatusOK, job)
}

func workerLeaseToken(r *http.Request) string {
	return r.Header.Get("X-Worker-Lease-Token")
}

func workerJobID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_job_id")
		return 0, false
	}
	return id, true
}

func (s *Server) workerJobInput(w http.ResponseWriter, r *http.Request) {
	id, ok := workerJobID(w, r)
	if !ok {
		return
	}
	input, err := s.Jobs.Input(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r))
	if errors.Is(err, jobs.ErrLeaseNotOwned) {
		fail(w, http.StatusNotFound, "job_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, input)
}

func (s *Server) workerJobMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := workerJobID(w, r)
	if !ok {
		return
	}
	input, err := s.Jobs.Input(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r))
	if errors.Is(err, jobs.ErrLeaseNotOwned) {
		fail(w, http.StatusNotFound, "job_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	path, err := s.resolveClipMedia(r.Context(), input.Job.ClipID)
	if errors.Is(err, os.ErrNotExist) {
		fail(w, http.StatusNotFound, "media_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		fail(w, http.StatusNotFound, "media_not_found")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		fail(w, http.StatusNotFound, "media_not_found")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	name := filepath.Base(input.OriginalName)
	http.ServeContent(w, r, name, info.ModTime(), f)
}

func (s *Server) workerJobHeartbeat(w http.ResponseWriter, r *http.Request) {
	id, ok := workerJobID(w, r)
	if !ok {
		return
	}
	if err := s.Jobs.Heartbeat(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r)); errors.Is(err, jobs.ErrLeaseNotOwned) {
		fail(w, http.StatusConflict, "lease_not_owned")
	} else if err != nil {
		s.internal(w, err)
	} else {
		reply(w, http.StatusOK, map[string]bool{"renewed": true})
	}
}

func (s *Server) completeWorkerJob(w http.ResponseWriter, r *http.Request) {
	id, ok := workerJobID(w, r)
	if !ok {
		return
	}
	var input struct {
		InputVersion string          `json:"input_version"`
		Result       json.RawMessage `json:"result"`
	}
	if !decode(w, r, &input) {
		return
	}
	var step, status string
	stepErr := s.Pool.QueryRow(r.Context(), `SELECT step,status FROM jobs WHERE id=$1`, id).Scan(&step, &status)
	if stepErr != nil && !errors.Is(stepErr, pgx.ErrNoRows) {
		s.internal(w, stepErr)
		return
	}
	if stepErr == nil && strings.HasPrefix(step, "voice.") {
		s.completeVoiceJob(w, r, id, step, status, input.InputVersion, input.Result)
		return
	}
	if stepErr == nil && status != "done" && strings.HasPrefix(step, "transcription.audio.") {
		leased, leaseErr := s.Jobs.Input(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r))
		if errors.Is(leaseErr, jobs.ErrLeaseNotOwned) {
			fail(w, http.StatusConflict, "lease_not_owned")
			return
		}
		if leaseErr != nil {
			s.internal(w, leaseErr)
			return
		}
		if !validTranscriptionResult(input.Result, leased.Job.Step) {
			fail(w, http.StatusBadRequest, "invalid_transcription_result")
			return
		}
	}
	job, err := s.Jobs.Complete(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r), input.InputVersion, input.Result)
	if errors.Is(err, jobs.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_job_result")
	} else if errors.Is(err, jobs.ErrLeaseNotOwned) {
		fail(w, http.StatusConflict, "lease_not_owned")
	} else if err != nil {
		s.internal(w, err)
	} else {
		reply(w, http.StatusOK, job)
	}
}

func (s *Server) failWorkerJob(w http.ResponseWriter, r *http.Request) {
	id, ok := workerJobID(w, r)
	if !ok {
		return
	}
	var input struct {
		Error string `json:"error"`
	}
	if !decode(w, r, &input) {
		return
	}
	job, err := s.Jobs.Fail(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r), input.Error)
	if errors.Is(err, jobs.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_job_error")
	} else if errors.Is(err, jobs.ErrLeaseNotOwned) {
		fail(w, http.StatusConflict, "lease_not_owned")
	} else if err != nil {
		s.internal(w, err)
	} else {
		reply(w, http.StatusOK, job)
	}
}
