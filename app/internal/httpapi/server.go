package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"
	"snitch/internal/auth"
	"snitch/internal/config"
	"snitch/internal/database"
	"snitch/internal/jobs"
	"snitch/internal/library"
	"snitch/internal/settings"
	"snitch/internal/setup"
	"snitch/internal/uploads"
	"snitch/internal/workers"
	"snitch/ui"
)

type Server struct {
	Pool            *pgxpool.Pool
	Config          config.Config
	Log             *slog.Logger
	Client          *http.Client
	Store           settings.Store
	Library         *library.Service
	Jobs            *jobs.Service
	Uploads         *uploads.Service
	Workers         *workers.Service
	rates           sync.Map
	embeddingCancel context.CancelFunc
	embeddingWG     sync.WaitGroup
	passwordSlots   chan struct{}
	dummyPassword   string
}
type session struct {
	User database.GetUserRow
	CSRF string
}
type sessionKey struct{}
type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type DiscordSettings struct {
	ClientID             *string         `json:"client_id"`
	ClientSecret         json.RawMessage `json:"client_secret,omitempty"`
	RegistrationApproval *bool           `json:"registration_approval"`
}

func New(p *pgxpool.Pool, c config.Config, logger *slog.Logger) *Server {
	return &Server{Pool: p, Config: c, Log: logger, Client: &http.Client{Timeout: 10 * time.Second}, Store: settings.Store{Pool: p, Key: c.MasterKey}, Library: library.New(p, library.Config{Roots: c.ClipRoots, ProbePath: c.ProbePath, Stability: c.StabilityWindow, ReconcileEvery: c.ReconcileInterval, Exclude: c.ClipExcludePatterns}, logger), Jobs: jobs.New(p), Uploads: uploads.New(p, uploads.Config{Root: c.UploadRoot, ProbePath: c.ProbePath, MaxBytes: c.UploadMaxBytes, ChunkSize: c.UploadChunkBytes, TTL: c.UploadSessionTTL}), Workers: workers.New(p), passwordSlots: make(chan struct{}, 2), dummyPassword: auth.Password(auth.Token())}
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.ensureEmbeddingProfile(ctx); err != nil {
		return fmt.Errorf("initialize embedding profile: %w", err)
	}
	if err := s.Uploads.Start(ctx); err != nil {
		return fmt.Errorf("start uploads: %w", err)
	}
	if err := s.Library.Start(ctx); err != nil {
		s.Uploads.Close()
		return fmt.Errorf("start library: %w", err)
	}
	if s.Config.EmbeddingsURL != "" {
		embeddingCtx, cancel := context.WithCancel(ctx)
		s.embeddingCancel = cancel
		for worker := 0; worker < 4; worker++ {
			s.embeddingWG.Add(1)
			go func() { defer s.embeddingWG.Done(); s.embeddingLoop(embeddingCtx) }()
		}
	}
	return nil
}
func (s *Server) Close() {
	if s.embeddingCancel != nil {
		s.embeddingCancel()
	}
	s.embeddingWG.Wait()
	s.Library.Close()
	s.Uploads.Close()
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health/live", func(w http.ResponseWriter, r *http.Request) { reply(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/health/ready", s.ready)
	mux.HandleFunc("GET /api/setup", s.setupStatus)
	mux.Handle("POST /api/setup", s.limited(http.HandlerFunc(s.bootstrap)))
	mux.Handle("POST /api/auth/login", s.limited(http.HandlerFunc(s.login)))
	mux.Handle("POST /api/auth/register", s.limited(http.HandlerFunc(s.register)))
	mux.Handle("GET /api/auth/me", s.protected(false, http.HandlerFunc(s.me)))
	mux.Handle("POST /api/auth/logout", s.protected(false, http.HandlerFunc(s.logout)))
	mux.Handle("POST /api/auth/discord/start", s.rateLimited("discord", http.HandlerFunc(s.discordStart)))
	mux.Handle("POST /api/auth/discord/link", s.protected(false, http.HandlerFunc(s.discordStart)))
	mux.Handle("GET /api/auth/discord/callback", s.rateLimited("discord", http.HandlerFunc(s.discordCallback)))
	mux.Handle("GET /api/admin/settings", s.protected(true, http.HandlerFunc(s.getSettings)))
	mux.Handle("GET /api/admin/vocabulary", s.protected(true, http.HandlerFunc(s.listVocabulary)))
	mux.Handle("GET /api/admin/transcription-pipeline", s.protected(true, http.HandlerFunc(s.getTranscriptionPipeline)))
	mux.Handle("PUT /api/admin/transcription-pipeline", s.protected(true, http.HandlerFunc(s.putTranscriptionPipeline)))
	mux.Handle("POST /api/admin/vocabulary", s.protected(true, http.HandlerFunc(s.createVocabulary)))
	mux.Handle("PUT /api/admin/vocabulary/{id}", s.protected(true, http.HandlerFunc(s.updateVocabulary)))
	mux.Handle("DELETE /api/admin/vocabulary/{id}", s.protected(true, http.HandlerFunc(s.deleteVocabulary)))
	mux.Handle("PUT /api/admin/settings", s.protected(true, http.HandlerFunc(s.putSettings)))
	mux.Handle("GET /api/admin/users", s.protected(true, http.HandlerFunc(s.pending)))
	mux.Handle("POST /api/admin/users/{id}/approve", s.protected(true, http.HandlerFunc(s.approve)))
	mux.Handle("GET /api/admin/jobs", s.protected(true, http.HandlerFunc(s.listJobs)))
	mux.Handle("POST /api/admin/jobs", s.protected(true, http.HandlerFunc(s.scheduleJob)))
	mux.Handle("GET /api/admin/jobs/{id}/history", s.protected(true, http.HandlerFunc(s.jobHistory)))
	mux.Handle("POST /api/admin/jobs/{id}/retry", s.protected(true, http.HandlerFunc(s.relaunchJob)))
	mux.Handle("DELETE /api/admin/jobs/{id}", s.protected(true, http.HandlerFunc(s.cancelJob)))
	mux.Handle("GET /api/admin/job-controls", s.protected(true, http.HandlerFunc(s.getJobControls)))
	mux.Handle("PUT /api/admin/job-controls", s.protected(true, http.HandlerFunc(s.updateJobControls)))
	mux.Handle("GET /api/admin/workers", s.protected(true, http.HandlerFunc(s.listWorkers)))
	mux.Handle("POST /api/admin/workers", s.protected(true, http.HandlerFunc(s.registerWorker)))
	mux.Handle("DELETE /api/admin/workers/{id}", s.protected(true, http.HandlerFunc(s.revokeWorker)))
	mux.Handle("POST /api/worker/pulse", s.workerOnly(http.HandlerFunc(s.workerPulse)))
	mux.Handle("POST /api/worker/jobs/claim", s.workerOnly(http.HandlerFunc(s.claimWorkerJob)))
	mux.Handle("GET /api/worker/jobs/{id}/input", s.workerOnly(http.HandlerFunc(s.workerJobInput)))
	mux.Handle("GET /api/worker/jobs/{id}/media", s.workerOnly(http.HandlerFunc(s.workerJobMedia)))
	mux.Handle("POST /api/worker/jobs/{id}/heartbeat", s.workerOnly(http.HandlerFunc(s.workerJobHeartbeat)))
	mux.Handle("POST /api/worker/jobs/{id}/complete", s.workerOnly(http.HandlerFunc(s.completeWorkerJob)))
	mux.Handle("POST /api/worker/jobs/{id}/fail", s.workerOnly(http.HandlerFunc(s.failWorkerJob)))
	mux.Handle("GET /api/admin/sources", s.protected(true, http.HandlerFunc(s.listSources)))
	mux.Handle("POST /api/admin/sources", s.protected(true, http.HandlerFunc(s.createSource)))
	mux.Handle("DELETE /api/admin/sources/{id}", s.protected(true, http.HandlerFunc(s.disableSource)))
	mux.Handle("GET /api/admin/sources/{id}/files", s.protected(true, http.HandlerFunc(s.sourceFiles)))
	mux.Handle("POST /api/admin/sources/{id}/scan", s.protected(true, http.HandlerFunc(s.scanSource)))
	mux.Handle("PUT /api/admin/sources/{id}/audio-track-rule", s.protected(true, http.HandlerFunc(s.putSourceAudioRule)))
	mux.Handle("PUT /api/admin/clips/{id}/audio-track", s.protected(true, http.HandlerFunc(s.putClipAudioTrack)))
	mux.Handle("DELETE /api/admin/clips/{id}/audio-track", s.protected(true, http.HandlerFunc(s.deleteClipAudioTrack)))
	mux.Handle("GET /api/uploads", s.protected(false, http.HandlerFunc(s.listUploads)))
	mux.Handle("POST /api/uploads", s.protected(false, http.HandlerFunc(s.createUpload)))
	mux.Handle("PUT /api/uploads/{id}/chunks", s.protected(false, http.HandlerFunc(s.uploadChunk)))
	mux.Handle("POST /api/uploads/{id}/complete", s.protected(false, http.HandlerFunc(s.completeUpload)))
	mux.Handle("DELETE /api/uploads/{id}", s.protected(false, http.HandlerFunc(s.cancelUpload)))
	mux.Handle("GET /api/clips", s.protected(false, http.HandlerFunc(s.listClips)))
	mux.Handle("GET /api/search", s.protected(false, http.HandlerFunc(s.search)))
	mux.Handle("GET /api/admin/embeddings", s.protected(true, http.HandlerFunc(s.embeddingStatus)))
	mux.Handle("POST /api/admin/embeddings/backfill", s.protected(true, http.HandlerFunc(s.backfillEmbeddings)))
	mux.Handle("POST /api/admin/embeddings/rebuild", s.protected(true, http.HandlerFunc(s.rebuildEmbeddings)))
	mux.Handle("GET /api/clips/{id}", s.protected(false, http.HandlerFunc(s.getClip)))
	mux.Handle("POST /api/admin/clips/{id}/transcription", s.protected(true, http.HandlerFunc(s.scheduleTranscription)))
	mux.Handle("GET /api/admin/transcriptions/batch-preview", s.protected(true, http.HandlerFunc(s.batchTranscriptionPreview)))
	mux.Handle("POST /api/admin/transcriptions/batch-schedule", s.protected(true, http.HandlerFunc(s.batchTranscriptionSchedule)))
	mux.Handle("GET /api/clips/{id}/media", s.protected(false, http.HandlerFunc(s.serveClipMedia)))
	s.registerVoiceRoutes(mux)
	embedded, _ := fs.Sub(ui.Files, "dist")
	files := http.FileServer(http.FS(embedded))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			fail(w, 404, "not_found")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			fail(w, 405, "method_not_allowed")
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(embedded, path); err != nil {
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		w.Header().Set("Cache-Control", "no-store")
		defer func() {
			if recover() != nil {
				s.Log.Error("request panic")
				fail(w, 500, "internal_error")
			}
		}()
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && !strings.HasPrefix(r.URL.Path, "/api/worker/") {
			// Browser mutations require the configured origin, including pre-authentication.
			if r.Header.Get("Origin") != s.Config.PublicURL {
				fail(w, 403, "origin_rejected")
				return
			}
		}
		limit := int64(16 * 1024)
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/uploads/") && strings.HasSuffix(r.URL.Path, "/chunks") {
			limit = s.Config.UploadChunkBytes
			if limit <= 0 {
				limit = 4 << 20
			}
		}
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/worker/jobs/") && strings.HasSuffix(r.URL.Path, "/complete") {
			limit = maxVoiceResultBytes + 4096 // voice fingerprints are larger than transcripts
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		mux.ServeHTTP(w, r)
	})
}
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code string) {
	reply(w, status, map[string]string{"error": code})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		fail(w, 415, "json_required")
		return false
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		fail(w, 400, "invalid_request")
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		fail(w, 400, "invalid_request")
		return false
	}
	return true
}
func valid(c credentials) bool {
	return validLengths(c) && strings.TrimSpace(c.Username) == c.Username && !strings.ContainsAny(c.Username, "\n\r\t")
}
func validLengths(c credentials) bool {
	u, p := utf8.RuneCountInString(c.Username), utf8.RuneCountInString(c.Password)
	return u >= 3 && u <= 64 && p >= 12 && p <= 256
}
func (s *Server) internal(w http.ResponseWriter, err error) {
	s.Log.Error("database operation failed", "kind", fmtError(err))
	fail(w, 503, "unavailable")
}
func fmtError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "operation_error"
}

func sessionCookie(r *http.Request) (*http.Cookie, error) {
	if c, err := r.Cookie("snitch_session"); err == nil {
		return c, nil
	}
	return r.Cookie("receipts_session")
}
func oauthCookie(r *http.Request) (*http.Cookie, error) {
	if c, err := r.Cookie("snitch_oauth"); err == nil {
		return c, nil
	}
	return r.Cookie("receipts_oauth")
}

func (s *Server) cookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: s.Config.Secure, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, id int64) error {
	token, csrf := auth.Token(), auth.Token()
	_, err := s.Pool.Exec(r.Context(), `INSERT INTO sessions(token_hash,user_id,csrf,expires_at) VALUES($1,$2,$3,now()+interval '7 days')`, auth.Hash(token), id, csrf)
	if err != nil {
		return err
	}
	s.cookie(w, "snitch_session", token, 7*86400)
	return nil
}
func (s *Server) findSession(r *http.Request) (session, error) {
	var ss session
	c, err := sessionCookie(r)
	if err != nil {
		return ss, err
	}
	err = s.Pool.QueryRow(r.Context(), `SELECT u.id,u.username,u.role,u.approved,s.csrf FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.approved`, auth.Hash(c.Value)).Scan(&ss.User.ID, &ss.User.Username, &ss.User.Role, &ss.User.Approved, &ss.CSRF)
	return ss, err
}
func (s *Server) protected(admin bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ss, err := s.findSession(r)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) && !errors.Is(err, http.ErrNoCookie) {
				s.internal(w, err)
			} else {
				fail(w, 401, "unauthorized")
			}
			return
		}
		if admin && ss.User.Role != "admin" {
			fail(w, 403, "forbidden")
			return
		}
		if r.Method != "GET" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(ss.CSRF)) != 1 {
			fail(w, 403, "csrf_rejected")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, ss)))
	})
}

type rateEntry struct {
	limiter *rate.Limiter
	seen    time.Time
}

func (s *Server) rateLimited(bucket string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Forwarded headers are deliberately ignored until a trusted proxy policy exists.
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		now := time.Now()
		s.rates.Range(func(k, v any) bool {
			if now.Sub(v.(rateEntry).seen) > time.Minute*10 {
				s.rates.Delete(k)
			}
			return true
		})
		entry, _ := s.rates.LoadOrStore(bucket+host, rateEntry{rate.NewLimiter(rate.Every(6*time.Second), 10), now})
		e := entry.(rateEntry)
		e.seen = now
		s.rates.Store(bucket+host, e)
		if !e.limiter.Allow() {
			w.Header().Set("Retry-After", "6")
			fail(w, 429, "rate_limited")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) limited(next http.Handler) http.Handler {
	return s.rateLimited("password", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case s.passwordSlots <- struct{}{}:
			defer func() { <-s.passwordSlots }()
			next.ServeHTTP(w, r)
		default:
			fail(w, 429, "busy")
		}
	}))
}
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	var initialized bool
	if err := s.Pool.QueryRow(ctx, `SELECT initialized FROM installation WHERE id=1`).Scan(&initialized); err != nil {
		fail(w, 503, "database_unavailable")
		return
	}
	status := "degraded"
	// No model is installed implicitly. The probe checks the provider, never indexes media.
	model := s.Config.EmbeddingsModel
	var activeModel string
	if err := s.Pool.QueryRow(ctx, `SELECT p.model FROM embedding_index_state i JOIN embedding_profiles p ON p.id=i.active_profile_id WHERE i.id=1`).Scan(&activeModel); err == nil && activeModel != "" {
		model = activeModel
	}
	if s.Config.EmbeddingsURL != "" && model != "" {
		body, _ := json.Marshal(embeddingBatch{Model: model, Input: []string{"health"}})
		requestBody := strings.NewReader(string(body))
		req, err := http.NewRequestWithContext(ctx, "POST", s.Config.EmbeddingsURL, requestBody)
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			resp, e := s.Client.Do(req)
			if e == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					status = "ok"
				}
			}
		}
	}
	reply(w, 200, map[string]string{"status": status})
}
func (s *Server) setupStatus(w http.ResponseWriter, r *http.Request) {
	var initialized bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT initialized FROM installation WHERE id=1`).Scan(&initialized); err != nil {
		s.internal(w, err)
		return
	}
	id, err := s.Store.Get(r.Context(), "discord_client_id")
	if err != nil {
		s.internal(w, err)
		return
	}
	secret, err := s.Store.Get(r.Context(), "discord_client_secret")
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]any{"initialized": initialized, "display_name": s.Config.DisplayName, "discord_enabled": id != "" && secret != "", "phase": 0, "deferred_steps": []string{"Dossiers : phase 1", "Worker : phase 2", "Embeddings : phase 4", "Hugging Face : phase 5"}})
}
func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	var c struct {
		credentials
		Code *string `json:"code"`
	}
	if !decode(w, r, &c) {
		return
	}
	if c.Code == nil || !valid(c.credentials) {
		fail(w, 400, "invalid_credentials_format")
		return
	}
	id, err := setup.Complete(r.Context(), s.Pool, *c.Code, c.Username, c.Password)
	if errors.Is(err, setup.ErrClosed) {
		fail(w, 409, "setup_closed_or_invalid_code")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if err := s.issueSession(w, r, id); err != nil {
		s.internal(w, err)
	} else {
		reply(w, 201, map[string]string{"status": "created"})
	}
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !decode(w, r, &c) {
		return
	}
	if !validLengths(c) {
		fail(w, 400, "invalid_request")
		return
	}
	var id int64
	var hash *string
	var approved bool
	err := s.Pool.QueryRow(r.Context(), `SELECT id,password_hash,approved FROM users WHERE lower(username)=lower($1)`, c.Username).Scan(&id, &hash, &approved)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	candidate := s.dummyPassword
	if hash != nil {
		candidate = *hash
	}
	matches := auth.Verify(candidate, c.Password)
	if err != nil || hash == nil || !matches || !approved {
		fail(w, 401, "invalid_credentials_or_pending")
		return
	}
	if err := s.issueSession(w, r, id); err != nil {
		s.internal(w, err)
	} else {
		reply(w, 200, map[string]string{"status": "ok"})
	}
}
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if !decode(w, r, &c) {
		return
	}
	if !valid(c) {
		fail(w, 400, "invalid_credentials_format")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var initialized, approval bool
	if err = tx.QueryRow(r.Context(), `SELECT initialized,registration_approval FROM installation WHERE id=1 FOR SHARE`).Scan(&initialized, &approval); err != nil {
		s.internal(w, err)
		return
	}
	if !initialized {
		fail(w, 409, "setup_required")
		return
	}
	tag, err := tx.Exec(r.Context(), `INSERT INTO users(username,password_hash,role,approved) VALUES($1,$2,'user',$3) ON CONFLICT DO NOTHING`, c.Username, auth.Password(c.Password), !approval)
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 409, "username_unavailable")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	status := "active"
	if approval {
		status = "pending"
	}
	reply(w, 201, map[string]string{"status": status})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	reply(w, 200, map[string]any{"user": ss.User, "csrf_token": ss.CSRF})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	c, _ := sessionCookie(r)
	if _, err := s.Pool.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash=$1`, auth.Hash(c.Value)); err != nil {
		s.internal(w, err)
		return
	}
	s.cookie(w, "snitch_session", "", -1)
	reply(w, 200, map[string]string{"status": "ok"})
}
func (s *Server) pending(w http.ResponseWriter, r *http.Request) {
	users, err := database.New(s.Pool).ListPending(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	if users == nil {
		users = []database.ListPendingRow{}
	}
	reply(w, 200, users)
}
func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_id")
		return
	}
	n, err := database.New(s.Pool).ApproveUser(r.Context(), id)
	if err != nil {
		s.internal(w, err)
		return
	}
	if n == 0 {
		fail(w, 404, "not_pending")
		return
	}
	reply(w, 200, map[string]string{"status": "ok"})
}
func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	id, err := s.Store.Get(r.Context(), "discord_client_id")
	if err != nil {
		s.internal(w, err)
		return
	}
	secret, err := s.Store.Get(r.Context(), "discord_client_secret")
	if err != nil {
		s.internal(w, err)
		return
	}
	var approval bool
	if err = s.Pool.QueryRow(r.Context(), `SELECT registration_approval FROM installation WHERE id=1`).Scan(&approval); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]any{"client_id": id, "secret_configured": secret != "", "registration_approval": approval})
}
func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	var c DiscordSettings
	if !decode(w, r, &c) {
		return
	}
	secret := ""
	if c.ClientSecret != nil {
		var value *string
		if err := json.Unmarshal(c.ClientSecret, &value); err != nil || value == nil {
			fail(w, 400, "invalid_request")
			return
		}
		secret = *value
	}
	if c.RegistrationApproval == nil || c.ClientID == nil || utf8.RuneCountInString(*c.ClientID) > 128 || utf8.RuneCountInString(secret) > 2048 {
		fail(w, 400, "invalid_request")
		return
	}
	if (*c.ClientID != "" && strings.TrimSpace(*c.ClientID) == "") || (secret != "" && strings.TrimSpace(secret) == "") {
		fail(w, 400, "discord_credentials_invalid")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	// Serialize settings changes before deciding whether the stored secret can be retained.
	if _, err = tx.Exec(r.Context(), `SELECT id FROM installation WHERE id=1 FOR UPDATE`); err != nil {
		s.internal(w, err)
		return
	}
	store := s.Store.WithTx(tx)
	if *c.ClientID != "" && secret == "" {
		storedSecret, err := store.Get(r.Context(), "discord_client_secret")
		if err != nil {
			s.internal(w, err)
			return
		}
		if storedSecret == "" {
			fail(w, 400, "discord_secret_required")
			return
		}
	}
	values := map[string]string{"discord_client_id": *c.ClientID}
	if secret != "" {
		values["discord_client_secret"] = secret
	}
	if *c.ClientID == "" {
		values["discord_client_secret"] = ""
	}
	for name, value := range values {
		if err := store.Set(r.Context(), name, value); err != nil {
			s.internal(w, err)
			return
		}
	}
	if _, err = tx.Exec(r.Context(), `UPDATE installation SET registration_approval=$1 WHERE id=1`, *c.RegistrationApproval); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]string{"status": "ok"})
}
func (s *Server) discordStart(w http.ResponseWriter, r *http.Request) {
	var initialized bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT initialized FROM installation WHERE id=1`).Scan(&initialized); err != nil {
		s.internal(w, err)
		return
	}
	if !initialized {
		fail(w, 409, "setup_required")
		return
	}
	id, err := s.Store.Get(r.Context(), "discord_client_id")
	if err != nil {
		s.internal(w, err)
		return
	}
	secret, err := s.Store.Get(r.Context(), "discord_client_secret")
	if err != nil {
		s.internal(w, err)
		return
	}
	if id == "" || secret == "" {
		fail(w, 409, "discord_not_configured")
		return
	}
	var userID *int64
	var sessionHash []byte
	if r.URL.Path == "/api/auth/discord/link" {
		ss := r.Context().Value(sessionKey{}).(session)
		userID = &ss.User.ID
		c, _ := sessionCookie(r)
		sessionHash = auth.Hash(c.Value)
	}
	state, browser := auth.Token(), auth.Token()
	_, err = s.Pool.Exec(r.Context(), `INSERT INTO oauth_states(state_hash,browser_hash,user_id,session_hash,expires_at) VALUES($1,$2,$3,$4,now()+interval '10 minutes')`, auth.Hash(state), auth.Hash(browser), userID, sessionHash)
	if err != nil {
		s.internal(w, err)
		return
	}
	s.cookie(w, "snitch_oauth", browser, 600)
	q := url.Values{"client_id": {id}, "response_type": {"code"}, "scope": {"identify"}, "state": {state}, "redirect_uri": {s.Config.PublicURL + "/api/auth/discord/callback"}}
	reply(w, 200, map[string]string{"url": s.Config.DiscordAuthorizeURL + "?" + q.Encode()})
}
func (s *Server) discordCallback(w http.ResponseWriter, r *http.Request) {
	state, code := r.URL.Query().Get("state"), r.URL.Query().Get("code")
	providerError := r.URL.Query().Get("error")
	browser, err := oauthCookie(r)
	if err != nil || state == "" || (code == "" && providerError == "") || (code != "" && providerError != "") {
		s.oauthFailure(w, r, 400, "invalid_oauth_state")
		return
	}
	var userID *int64
	var sessionHash []byte
	err = s.Pool.QueryRow(r.Context(), `DELETE FROM oauth_states WHERE state_hash=$1 AND browser_hash=$2 AND expires_at>now() RETURNING user_id,session_hash`, auth.Hash(state), auth.Hash(browser.Value)).Scan(&userID, &sessionHash)
	if err != nil {
		s.oauthFailure(w, r, 400, "invalid_oauth_state")
		return
	}
	s.cookie(w, "snitch_oauth", "", -1)
	if userID != nil {
		ss, e := s.findSession(r)
		c, _ := sessionCookie(r)
		if e != nil || ss.User.ID != *userID || c == nil || subtle.ConstantTimeCompare(sessionHash, auth.Hash(c.Value)) != 1 {
			s.oauthFailure(w, r, 403, "link_session_required")
			return
		}
	}
	if providerError != "" {
		if providerError == "access_denied" {
			http.Redirect(w, r, "/?notice=discord_denied", http.StatusSeeOther)
		} else {
			s.oauthFailure(w, r, 502, "discord_unavailable")
		}
		return
	}
	id, err := s.Store.Get(r.Context(), "discord_client_id")
	if err != nil {
		s.oauthInternal(w, r, err)
		return
	}
	secret, err := s.Store.Get(r.Context(), "discord_client_secret")
	if err != nil {
		s.oauthInternal(w, r, err)
		return
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {id}, "client_secret": {secret}, "redirect_uri": {s.Config.PublicURL + "/api/auth/discord/callback"}}
	req, err := http.NewRequestWithContext(r.Context(), "POST", s.Config.DiscordTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		s.oauthFailure(w, r, 502, "discord_unavailable")
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.Client.Do(req)
	if err != nil {
		s.oauthFailure(w, r, 502, "discord_unavailable")
		return
	}
	defer resp.Body.Close()
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if resp.StatusCode != 200 || json.NewDecoder(io.LimitReader(resp.Body, 16384)).Decode(&token) != nil || token.AccessToken == "" {
		s.oauthFailure(w, r, 502, "discord_unavailable")
		return
	}
	req, err = http.NewRequestWithContext(r.Context(), "GET", s.Config.DiscordUserURL, nil)
	if err != nil {
		s.oauthFailure(w, r, 502, "discord_unavailable")
		return
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err = s.Client.Do(req)
	if err != nil {
		s.oauthFailure(w, r, 502, "discord_unavailable")
		return
	}
	defer resp.Body.Close()
	var identity struct {
		ID string `json:"id"`
	}
	if resp.StatusCode != 200 || json.NewDecoder(io.LimitReader(resp.Body, 16384)).Decode(&identity) != nil || identity.ID == "" || len(identity.ID) > 64 {
		s.oauthFailure(w, r, 502, "discord_unavailable")
		return
	}
	if userID != nil {
		// Unique identity constraint rejects linking an identity already owned by another account.
		tag, e := s.Pool.Exec(r.Context(), `UPDATE users SET discord_id=$1 WHERE id=$2 AND (discord_id IS NULL OR discord_id=$1) AND EXISTS (SELECT 1 FROM sessions WHERE token_hash=$3 AND user_id=$2 AND expires_at>now())`, identity.ID, *userID, sessionHash)
		if e == nil && tag.RowsAffected() == 0 {
			if _, err := s.findSession(r); err != nil {
				s.oauthFailure(w, r, 403, "link_session_required")
				return
			}
		}
		if e != nil || tag.RowsAffected() != 1 {
			s.oauthFailure(w, r, 409, "discord_already_linked")
			return
		}
	} else {
		tx, e := s.Pool.Begin(r.Context())
		if e != nil {
			s.oauthInternal(w, r, e)
			return
		}
		defer tx.Rollback(r.Context())
		var approval bool
		if e = tx.QueryRow(r.Context(), `SELECT registration_approval FROM installation WHERE id=1 FOR SHARE`).Scan(&approval); e != nil {
			s.oauthInternal(w, r, e)
			return
		}
		var uid int64
		var approved bool
		e = tx.QueryRow(r.Context(), `INSERT INTO users(username,role,approved,discord_id) VALUES($1,'user',$2,$3) ON CONFLICT DO NOTHING RETURNING id,approved`, "discord_"+identity.ID, !approval, identity.ID).Scan(&uid, &approved)
		if errors.Is(e, pgx.ErrNoRows) {
			e = tx.QueryRow(r.Context(), `SELECT id,approved FROM users WHERE discord_id=$1`, identity.ID).Scan(&uid, &approved)
		}
		if errors.Is(e, pgx.ErrNoRows) {
			// A local username collision is never an identity match. Allocate a distinct name.
			e = tx.QueryRow(r.Context(), `INSERT INTO users(username,role,approved,discord_id) VALUES($1,'user',$2,$3) ON CONFLICT (discord_id) DO UPDATE SET discord_id=EXCLUDED.discord_id RETURNING id,approved`, "discord_"+auth.Token(), !approval, identity.ID).Scan(&uid, &approved)
		}
		if e != nil {
			s.oauthInternal(w, r, e)
			return
		}
		if e = tx.Commit(r.Context()); e != nil {
			s.oauthInternal(w, r, e)
			return
		}
		if !approved {
			http.Redirect(w, r, "/?notice=pending", http.StatusSeeOther)
			return
		}
		if err := s.issueSession(w, r, uid); err != nil {
			s.oauthInternal(w, r, err)
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Browser callbacks return to the application, while API callers retain structured status errors.
func (s *Server) oauthFailure(w http.ResponseWriter, r *http.Request, status int, code string) {
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.Redirect(w, r, "/?notice="+url.QueryEscape(code), http.StatusSeeOther)
		return
	}
	fail(w, status, code)
}

func (s *Server) oauthInternal(w http.ResponseWriter, r *http.Request, err error) {
	s.Log.Error("database operation failed", "kind", fmtError(err))
	s.oauthFailure(w, r, 503, "unavailable")
}
