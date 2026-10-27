package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/config"
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
	var version string
	var raw []byte
	if err = s.Pool.QueryRow(r.Context(), `SELECT v.version,v.config FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&version, &raw); err != nil {
		s.internal(w, err)
		return
	}
	var profile transcriptionPipeline
	if err = json.Unmarshal(raw, &profile); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{
		"items":        items,
		"first_worker": len(items) == 0,
		"snitch_url":   s.publicOrigin(r),
		"transcription": map[string]string{
			"model": profile.Model, "language": profile.Language, "version": version,
		},
	})
}

var (
	transcriptionLanguagePattern = regexp.MustCompile(`^[a-z]{2,3}$`)
	transcriptionModelPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:/+-]{0,99}$`)
)

// validWhisperURL accepts an http(s) address without credentials, query or fragment.
func validWhisperURL(raw string) bool {
	if len(raw) > 300 || strings.TrimSpace(raw) != raw {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && !strings.ContainsAny(raw, " \t\r\n\"'`$\\")
}

func (s *Server) workerBundle(r *http.Request, token, whisper string) (workers.Bundle, error) {
	image := s.Config.WorkerImage
	if image == "" {
		image = config.DefaultWorkerImage
	}
	return workers.NewBundle(workers.BundleInput{APIURL: s.publicOrigin(r), Token: token, Image: image, WhisperURL: whisper})
}

func (s *Server) registerWorker(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name             string `json:"name"`
		MaxConcurrency   int    `json:"max_concurrency"`
		TranscriptionURL string `json:"transcription_url"`
		Transcription    *struct {
			Model    string `json:"model"`
			Language string `json:"language"`
		} `json:"transcription"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.TranscriptionURL != "" && !validWhisperURL(input.TranscriptionURL) {
		fail(w, http.StatusBadRequest, "invalid_transcription_url")
		return
	}
	var profile *transcriptionPipeline
	if input.Transcription != nil {
		model, language := strings.TrimSpace(input.Transcription.Model), strings.TrimSpace(input.Transcription.Language)
		switch {
		case model == "":
			fail(w, http.StatusBadRequest, "transcription_model_required")
			return
		case language == "":
			fail(w, http.StatusBadRequest, "transcription_language_required")
			return
		case !transcriptionModelPattern.MatchString(model):
			fail(w, http.StatusBadRequest, "invalid_transcription_model")
			return
		case !transcriptionLanguagePattern.MatchString(language):
			fail(w, http.StatusBadRequest, "invalid_transcription_language")
			return
		}
		profile = &transcriptionPipeline{Provider: "openai-compatible", Model: model, Language: language, Preparation: "wav-16khz-mono-v1"}
		profile.VAD.Mode, profile.VAD.Version = "off", "vad-off-v1"
	}
	registration, _, err := s.Workers.RegisterWith(r.Context(), workers.Input{Name: input.Name, MaxConcurrency: input.MaxConcurrency}, func(ctx context.Context, tx pgx.Tx, first bool) error {
		if profile == nil {
			return nil
		}
		if !first {
			return errProfileAlreadySet
		}
		_, err := activatePipeline(ctx, tx, *profile)
		return err
	})
	if errors.Is(err, errProfileAlreadySet) {
		fail(w, http.StatusConflict, "transcription_profile_already_set")
		return
	}
	if errors.Is(err, workers.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_worker")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	registration.Bundle, err = s.workerBundle(r, registration.Token, input.TranscriptionURL)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusCreated, registration)
}

var errProfileAlreadySet = errors.New("transcription profile already set")

func (s *Server) getWorker(w http.ResponseWriter, r *http.Request) {
	detail, err := s.Workers.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, workers.ErrNotFound) {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	// The token is never readable: the deployment file carries a placeholder.
	bundle, err := s.workerBundle(r, workers.Placeholder, "")
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{
		"id": detail.ID, "name": detail.Name, "max_concurrency": detail.MaxConcurrency, "reported_concurrency": detail.ReportedConcurrency,
		"last_seen_at": detail.LastSeenAt, "created_at": detail.CreatedAt, "revoked_at": detail.RevokedAt, "connected": detail.Connected,
		"models": detail.Models, "transcription_model": detail.TranscriptionModel, "machine": detail.Machine, "remote_address": detail.RemoteAddress, "recent_tasks": detail.RecentTasks, "bundle": bundle,
	})
}

func (s *Server) putWorkerTranscriptionModel(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Model string `json:"model"`
	}
	if !decode(w, r, &input) {
		return
	}
	input.Model = strings.TrimSpace(input.Model)
	if input.Model != "" && !transcriptionModelPattern.MatchString(input.Model) {
		fail(w, http.StatusBadRequest, "invalid_transcription_model")
		return
	}
	result, err := s.Pool.Exec(r.Context(), `UPDATE workers SET transcription_model=$2 WHERE id=$1`, r.PathValue("id"), input.Model)
	if err != nil {
		s.internal(w, err)
		return
	}
	if result.RowsAffected() == 0 {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	reply(w, http.StatusOK, map[string]string{"model": input.Model})
}

func (s *Server) renewWorkerToken(w http.ResponseWriter, r *http.Request) {
	registration, err := s.Workers.RenewToken(r.Context(), r.PathValue("id"))
	if errors.Is(err, workers.ErrNotFound) {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	registration.Bundle, err = s.workerBundle(r, registration.Token, "")
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
		Capacity       int             `json:"capacity"`
		SupportedSteps []string        `json:"supported_steps"`
		Models         workers.Models  `json:"models"`
		Machine        workers.Machine `json:"machine"`
	}
	if !decode(w, r, &input) {
		return
	}
	worker := currentWorker(r)
	if input.Capacity < 1 || input.Capacity > worker.MaxConcurrency {
		fail(w, http.StatusBadRequest, "invalid_worker_capacity")
		return
	}
	remote, _, _ := net.SplitHostPort(r.RemoteAddr)
	if net.ParseIP(remote) == nil {
		remote = ""
	}
	if err := s.Workers.Pulse(r.Context(), worker.ID, input.Capacity, input.SupportedSteps, input.Models, input.Machine, remote); err != nil {
		if errors.Is(err, workers.ErrUnauthorized) {
			fail(w, http.StatusUnauthorized, "worker_token_invalid")
		} else if errors.Is(err, workers.ErrInvalidInput) && !input.Machine.Valid() {
			fail(w, http.StatusBadRequest, "invalid_worker_machine")
		} else if errors.Is(err, workers.ErrInvalidInput) && !input.Models.Valid() {
			fail(w, http.StatusBadRequest, "invalid_worker_models")
		} else if errors.Is(err, workers.ErrInvalidInput) {
			fail(w, http.StatusBadRequest, "invalid_worker_capabilities")
		} else {
			s.internal(w, err)
		}
		return
	}
	worker.ReportedConcurrency = input.Capacity
	worker.Models = input.Models
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
