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
	"snitch/internal/events"
	"snitch/internal/httpapi/contract"
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
	page, err := s.workersPage(r)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, page)
}

// workersPage is the Workers list, also the snapshot of the live list (admin:workers).
func (s *Server) workersPage(r *http.Request) (map[string]any, error) {
	items, err := s.Workers.List(r.Context())
	if err != nil {
		return nil, err
	}
	if err = s.withInstallProgress(r, items, false); err != nil {
		return nil, err
	}
	var version string
	var raw []byte
	if err = s.Pool.QueryRow(r.Context(), `SELECT v.version,v.config FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&version, &raw); err != nil {
		return nil, err
	}
	var profile transcriptionPipeline
	if err = json.Unmarshal(raw, &profile); err != nil {
		return nil, err
	}
	var mode, endpoint, settingsVersion string
	err = s.Pool.QueryRow(r.Context(), `SELECT mode,endpoint,version FROM transcription_settings WHERE id=true`).Scan(&mode, &endpoint, &settingsVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		mode = "unconfigured"
	} else if err != nil {
		return nil, err
	}
	return map[string]any{
		"items":        items,
		"first_worker": len(items) == 0,
		"snitch_url":   s.publicOrigin(r),
		"transcription": map[string]string{
			"mode": mode, "endpoint": endpoint, "settings_version": settingsVersion,
			"model": profile.Model, "language": profile.Language, "version": version,
		},
	}, nil
}

var (
	transcriptionModelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:/+-]{0,99}$`)
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
	var mode, endpoint string
	err := s.Pool.QueryRow(r.Context(), `SELECT mode,endpoint FROM transcription_settings WHERE id=true`).Scan(&mode, &endpoint)
	if errors.Is(err, pgx.ErrNoRows) {
		mode, endpoint = "unconfigured", ""
	} else if err != nil {
		s.internal(w, err)
		return
	}
	// Legacy per-Worker fields only bootstrap an unconfigured Snitch; once the
	// Administrator has chosen settings they must never be overwritten or diverge silently.
	if mode != "unconfigured" && (input.TranscriptionURL != "" || input.Transcription != nil) {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	// Validate transcription config if provided (for bootstrap only)
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
		case !validTranscriptionLanguage(language):
			fail(w, http.StatusBadRequest, "invalid_transcription_language")
			return
		}
		profile = &transcriptionPipeline{Provider: "openai-compatible", Model: model, Language: language, Preparation: "wav-16khz-mono-v1"}
		profile.VAD.Mode, profile.VAD.Version = "off", "vad-off-v1"
	}
	// Determine the WhisperURL for the bundle: prefer centralized config, fallback to provided TranscriptionURL
	var bundleWhisperURL string
	var installMode, installEndpoint, installModel, installLanguage string
	registration, _, err := s.Workers.RegisterWith(r.Context(), workers.Input{Name: input.Name, MaxConcurrency: input.MaxConcurrency}, func(ctx context.Context, tx pgx.Tx, first bool) error {
		if !first {
			return nil
		}
		// For the first worker, initialize centralized config from registration input if provided
		if profile != nil {
			version, err := activatePipeline(ctx, tx, *profile)
			if err != nil {
				return err
			}
			// Initialize transcription_settings with the provided config
			_, err = tx.Exec(ctx, `INSERT INTO transcription_settings(mode,endpoint,version) VALUES('enabled',$1,$2) ON CONFLICT(id) DO UPDATE SET mode='enabled',endpoint=$1,version=$2,updated_at=now()`, input.TranscriptionURL, version)
			if err != nil {
				return err
			}
			bundleWhisperURL = input.TranscriptionURL
			installMode = "enabled"
			installEndpoint = input.TranscriptionURL
			installModel = profile.Model
			installLanguage = profile.Language
		}
		return nil
	})
	if errors.Is(err, workers.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_worker")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	// If not first worker or no profile provided, fetch centralized config for bundle
	if bundleWhisperURL == "" {
		var endpoint, version string
		err = s.Pool.QueryRow(r.Context(), `SELECT endpoint,version FROM transcription_settings WHERE id=true AND mode='enabled'`).Scan(&endpoint, &version)
		if err == nil && endpoint != "" {
			bundleWhisperURL = endpoint
		} else if input.TranscriptionURL != "" {
			bundleWhisperURL = input.TranscriptionURL
		}
	}
	// For install command, fetch the active transcription profile if not set by bootstrap
	if installMode == "" {
		installMode = mode
		installEndpoint = endpoint
		err = s.Pool.QueryRow(r.Context(), `SELECT v.config->>'model',v.config->>'language' FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&installModel, &installLanguage)
		if err != nil {
			s.internal(w, err)
			return
		}
	}
	registration.Bundle, err = s.workerBundle(r, registration.Token, bundleWhisperURL)
	if err != nil {
		s.internal(w, err)
		return
	}
	image := s.Config.WorkerImage
	if image == "" {
		image = config.DefaultWorkerImage
	}
	registration.InstallCommand, registration.InstallCommandUnavailable, err = s.createWorkerInstallCommand(r.Context(), registration.ID, registration.Token, image, s.publicOrigin(r), r.URL.Query().Get("allow_insecure_install") == "true", installMode, installEndpoint, installModel, installLanguage)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusCreated, registration)
}

var errProfileAlreadySet = errors.New("transcription profile already set")

func (s *Server) getWorker(w http.ResponseWriter, r *http.Request) {
	result, err := s.workerDetail(r, r.PathValue("id"))
	if errors.Is(err, workers.ErrNotFound) {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, result)
}

// workerDetail is the page of one Worker, also the snapshot of its live page (worker:<id>).
func (s *Server) workerDetail(r *http.Request, id string) (map[string]any, error) {
	detail, err := s.Workers.Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	one := []workers.Worker{detail.Worker}
	if err = s.withInstallProgress(r, one, true); err != nil {
		return nil, err
	}
	detail.Install = one[0].Install
	// The token is never readable: the deployment file carries a placeholder.
	bundle, err := s.workerBundle(r, workers.Placeholder, "")
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"id": detail.ID, "name": detail.Name, "max_concurrency": detail.MaxConcurrency, "reported_concurrency": detail.ReportedConcurrency,
		"last_seen_at": detail.LastSeenAt, "created_at": detail.CreatedAt, "revoked_at": detail.RevokedAt, "revoke_reason": detail.RevokeReason, "revocation_acknowledged_at": detail.RevocationAcknowledgedAt, "connected": detail.Connected,
		"models": detail.Models, "transcription_model": detail.TranscriptionModel, "machine": detail.Machine, "remote_address": detail.RemoteAddress, "install": detail.Install, "transcription_config_ready": detail.TranscriptionConfigReady, "recent_tasks": detail.RecentTasks, "bundle": bundle,
	}
	if detail.TranscriptionConfig != nil {
		result["transcription_config"] = detail.TranscriptionConfig
	}
	return result, nil
}

// workersEvents streams the Workers list: a snapshot, then one per change (admin:workers).
func (s *Server) workersEvents(w http.ResponseWriter, r *http.Request) {
	s.Events.Stream(w, r, events.Topic{
		Name:      "admin:workers",
		Authorize: s.adminAuthorizer(r),
		Snapshot: func(ctx context.Context) (any, error) {
			return s.workersPage(r.WithContext(ctx))
		},
	})
}

// workerEvents streams the page of one Worker (worker:<id>). A Worker that does not exist ends
// the stream without a snapshot; the page then falls back to GET and shows the error.
func (s *Server) workerEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.Events.Stream(w, r, events.Topic{
		Name:      "worker:" + id,
		Authorize: s.adminAuthorizer(r),
		Snapshot: func(ctx context.Context) (any, error) {
			return s.workerDetail(r.WithContext(ctx), id)
		},
	})
}

// serveWorkerPresence publishes the Offline changes that no write announces.
func (s *Server) serveWorkerPresence(ctx context.Context) {
	ticker := time.NewTicker(workers.PresenceCheckInterval)
	defer ticker.Stop()
	for {
		if err := s.Workers.PublishDisconnections(ctx); err != nil && ctx.Err() == nil {
			s.Log.Error("publish worker presence", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
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
	var mode, endpoint string
	err = s.Pool.QueryRow(r.Context(), `SELECT mode,endpoint FROM transcription_settings WHERE id=true`).Scan(&mode, &endpoint)
	if errors.Is(err, pgx.ErrNoRows) {
		mode, endpoint = "unconfigured", ""
	} else if err != nil {
		s.internal(w, err)
		return
	}
	var model, language string
	err = s.Pool.QueryRow(r.Context(), `SELECT v.config->>'model',v.config->>'language' FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&model, &language)
	if err != nil {
		s.internal(w, err)
		return
	}
	registration.Bundle, err = s.workerBundle(r, registration.Token, endpoint)
	if err != nil {
		s.internal(w, err)
		return
	}
	image := s.Config.WorkerImage
	if image == "" {
		image = config.DefaultWorkerImage
	}
	registration.InstallCommand, registration.InstallCommandUnavailable, err = s.createWorkerInstallCommand(r.Context(), registration.ID, registration.Token, image, s.publicOrigin(r), r.URL.Query().Get("allow_insecure_install") == "true", mode, endpoint, model, language)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusCreated, registration)
}

func (s *Server) revokeWorker(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason string `json:"reason"`
	}
	if r.ContentLength != 0 && !decode(w, r, &input) {
		return
	}
	err := s.Workers.Revoke(r.Context(), r.PathValue("id"), input.Reason)
	if errors.Is(err, workers.ErrNotFound) {
		fail(w, http.StatusNotFound, "worker_not_found")
		return
	}
	if errors.Is(err, workers.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_revoke_reason")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	s.Log.Info("worker_revoked", "worker_id", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

// forgetWorker removes a revoked Worker, whose machine confirmed the revocation, from the list.
func (s *Server) forgetWorker(w http.ResponseWriter, r *http.Request) {
	switch err := s.Workers.Forget(r.Context(), r.PathValue("id")); {
	case errors.Is(err, workers.ErrNotFound):
		fail(w, http.StatusNotFound, "worker_not_found")
	case errors.Is(err, workers.ErrNotRevoked):
		fail(w, http.StatusConflict, "worker_not_revoked")
	case errors.Is(err, workers.ErrCleanupPending):
		fail(w, http.StatusConflict, "worker_cleanup_pending")
	case err != nil:
		s.internal(w, err)
	default:
		s.Log.Info("worker_removed", "worker_id", r.PathValue("id"))
		w.WriteHeader(http.StatusNoContent)
	}
}

// workerRevocation lets the local launcher learn that its identity is revoked. It is deliberately
// outside workerOnly: a revoked token is accepted here and nowhere else, and the answer carries
// no configuration and enables no action.
func (s *Server) workerRevocation(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authorization, "Bearer ")
	if !strings.HasPrefix(authorization, "Bearer ") || token == "" || strings.ContainsAny(token, " \t\r\n") {
		fail(w, http.StatusUnauthorized, "worker_token_required")
		return
	}
	revoked, at, err := s.Workers.RevocationStatus(r.Context(), token)
	if errors.Is(err, workers.ErrUnauthorized) {
		fail(w, http.StatusUnauthorized, "worker_token_invalid")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if !revoked {
		reply(w, http.StatusOK, map[string]any{"revoked": false})
		return
	}
	reply(w, http.StatusOK, map[string]any{"revoked": true, "revoked_at": at})
}

func (s *Server) workerPulse(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Capacity       int             `json:"capacity"`
		SupportedSteps []string        `json:"supported_steps"`
		Models         workers.Models  `json:"models"`
		Machine        workers.Machine `json:"machine"`
		// Joignabilité du serveur de transcription (#212) ; absente tant que le Worker n'a rien testé.
		TranscriptionReachability *contract.WorkerTranscriptionReachabilityInput `json:"transcription_reachability"`
	}
	if !decode(w, r, &input) {
		return
	}
	worker := currentWorker(r)
	if input.Capacity < 1 || input.Capacity > worker.MaxConcurrency {
		fail(w, http.StatusBadRequest, "invalid_worker_capacity")
		return
	}
	if input.TranscriptionReachability != nil && !validReachability(input.TranscriptionReachability) {
		fail(w, http.StatusBadRequest, "invalid_transcription_reachability")
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
	if err := s.recordTranscriptionReachability(r.Context(), worker.ID, input.TranscriptionReachability); err != nil {
		s.internal(w, err)
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
	defer s.syncIncidentsSoon(r.Context())
	job, err := s.Jobs.Claim(r.Context(), currentWorker(r).ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if job == nil {
		s.Log.Debug("claim_empty", "worker_id", currentWorker(r).ID)
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
		s.Log.Debug("job_heartbeat", "job_id", id, "worker_id", currentWorker(r).ID)
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
	var options jobs.CompleteOptions
	if stepErr == nil && strings.HasPrefix(step, "transcription.audio.") {
		options = jobs.CompleteOptions{MaxResultBytes: maxTranscriptionResultBytes, Summarize: withoutPeaks, Project: waveformProjection(step, input.Result)}
	}
	job, err := s.Jobs.CompleteWith(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r), input.InputVersion, input.Result, options)
	if errors.Is(err, jobs.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_job_result")
	} else if errors.Is(err, jobs.ErrLeaseNotOwned) {
		fail(w, http.StatusConflict, "lease_not_owned")
	} else if err != nil {
		s.internal(w, err)
	} else {
		s.reconcileAnalysis(r.Context(), job.ClipID)
		s.syncIncidents(r.Context())
		reply(w, http.StatusOK, job)
	}
}

func (s *Server) failWorkerJob(w http.ResponseWriter, r *http.Request) {
	id, ok := workerJobID(w, r)
	if !ok {
		return
	}
	var input struct {
		Error      string  `json:"error"`
		Provider   *string `json:"provider"`
		Model      *string `json:"model"`
		Revision   *string `json:"revision"`
		HTTPStatus *int    `json:"http_status"`
		DurationMS *int    `json:"duration_ms"`
	}
	if !decode(w, r, &input) {
		return
	}
	job, err := s.Jobs.Fail(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r), input.Error, jobs.Diagnostic{Provider: input.Provider, Model: input.Model, Revision: input.Revision, HTTPStatus: input.HTTPStatus, DurationMS: input.DurationMS})
	if errors.Is(err, jobs.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_job_error")
	} else if errors.Is(err, jobs.ErrLeaseNotOwned) {
		fail(w, http.StatusConflict, "lease_not_owned")
	} else if err != nil {
		s.internal(w, err)
	} else {
		s.reconcileAnalysis(r.Context(), job.ClipID)
		s.syncIncidents(r.Context())
		reply(w, http.StatusOK, job)
	}
}
