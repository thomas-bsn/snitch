package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// The Discord webhook (#283, ADR 0067) relays a redacted summary of some notifications to a Discord channel. The
// notifications stay the single source of truth: a database trigger enqueues one delivery per notification raised
// in an allowed category, and the loop below posts the due deliveries, outside any request or transaction of the
// workflow that raised them. Discord being down, slow or limiting never blocks Snitch: deliveries are retried a
// bounded number of times, then marked failed, and the administration shows both.

const (
	webhookURLKey       = "notification_webhook_url"
	webhookMaxAttempts  = 6
	webhookBatch        = 10
	webhookLease        = 2 * time.Minute
	webhookMaxRetryWait = 30 * time.Minute
)

var webhookCategories = []string{"access_requests", "human_actions", "worker_offline", "job_failures", "job_blocked"}

// The only addresses accepted: an execute-webhook URL of Discord itself. Anything else (another host, a port,
// credentials, a query) is refused before it is stored, and nothing is ever fetched from an address a user typed
// but this one (no redirect is followed).
var (
	webhookHosts = []string{"discord.com", "discordapp.com", "canary.discord.com", "ptb.discord.com"}
	webhookPath  = regexp.MustCompile(`^/api/webhooks/([0-9]{15,25})/[A-Za-z0-9_-]{20,120}$`)
)

// validateWebhookURL returns the address to store and the hint shown in its place (the end of the webhook id).
func validateWebhookURL(raw string) (string, string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return "", "", false
	}
	host := strings.ToLower(u.Hostname())
	m := webhookPath.FindStringSubmatch(u.EscapedPath())
	if !slices.Contains(webhookHosts, host) || m == nil {
		return "", "", false
	}
	return "https://" + host + u.EscapedPath(), m[1][len(m[1])-4:], true
}

func (s *Server) registerNotificationWebhookRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/notification-webhook", s.protected(true, http.HandlerFunc(s.getNotificationWebhook)))
	mux.Handle("PUT /api/admin/notification-webhook", s.protectedFor(isSuperAdministrator, http.HandlerFunc(s.putNotificationWebhook)))
	mux.Handle("DELETE /api/admin/notification-webhook", s.protectedFor(isSuperAdministrator, http.HandlerFunc(s.deleteNotificationWebhook)))
	mux.Handle("POST /api/admin/notification-webhook/test", s.protectedFor(isSuperAdministrator, http.HandlerFunc(s.testNotificationWebhook)))
}

type webhookDeliveryView struct {
	ID            int64      `json:"id"`
	Category      string     `json:"category"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"last_error"`
	CreatedAt     time.Time  `json:"created_at"`
	NextAttemptAt time.Time  `json:"next_attempt_at"`
	SentAt        *time.Time `json:"sent_at"`
}

type webhookAuditView struct {
	Action        string    `json:"action"`
	Enabled       bool      `json:"enabled"`
	Categories    []string  `json:"categories"`
	ActorUsername string    `json:"actor_username"`
	CreatedAt     time.Time `json:"created_at"`
}

type webhookView struct {
	// Configured tells whether an address is stored; the address itself is never sent back.
	Configured    bool                  `json:"configured"`
	Enabled       bool                  `json:"enabled"`
	Categories    []string              `json:"categories"`
	Hint          string                `json:"hint"`
	LastStatus    string                `json:"last_status"`
	LastError     string                `json:"last_error"`
	LastAttemptAt *time.Time            `json:"last_attempt_at"`
	LastSuccessAt *time.Time            `json:"last_success_at"`
	Pending       int                   `json:"pending"`
	Failed        int                   `json:"failed"`
	Deliveries    []webhookDeliveryView `json:"deliveries"`
	Audit         []webhookAuditView    `json:"audit"`
}

func (s *Server) readNotificationWebhook(ctx context.Context) (webhookView, error) {
	v := webhookView{Categories: []string{"access_requests"}, LastStatus: "never", Deliveries: []webhookDeliveryView{}, Audit: []webhookAuditView{}}
	address, err := s.Store.Get(ctx, webhookURLKey)
	if err != nil {
		return v, err
	}
	v.Configured = address != ""
	err = s.Pool.QueryRow(ctx, `SELECT enabled,categories,hint,last_status,last_error,last_attempt_at,last_success_at FROM notification_webhook WHERE id`).Scan(&v.Enabled, &v.Categories, &v.Hint, &v.LastStatus, &v.LastError, &v.LastAttemptAt, &v.LastSuccessAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return v, err
	}
	// An address the current master key cannot read is not configured: nothing can be sent.
	v.Enabled = v.Enabled && v.Configured
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status='pending')::int,count(*) FILTER (WHERE status='failed' AND created_at>now()-interval '7 days')::int FROM notification_deliveries`).Scan(&v.Pending, &v.Failed); err != nil {
		return v, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT id,category,status,attempts,last_error,created_at,next_attempt_at,sent_at FROM notification_deliveries ORDER BY id DESC LIMIT 20`)
	if err != nil {
		return v, err
	}
	for rows.Next() {
		var d webhookDeliveryView
		if err = rows.Scan(&d.ID, &d.Category, &d.Status, &d.Attempts, &d.LastError, &d.CreatedAt, &d.NextAttemptAt, &d.SentAt); err != nil {
			rows.Close()
			return v, err
		}
		v.Deliveries = append(v.Deliveries, d)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return v, err
	}
	rows, err = s.Pool.Query(ctx, `SELECT action,enabled,categories,actor_username,created_at FROM notification_webhook_events ORDER BY id DESC LIMIT 20`)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		var a webhookAuditView
		if err = rows.Scan(&a.Action, &a.Enabled, &a.Categories, &a.ActorUsername, &a.CreatedAt); err != nil {
			return v, err
		}
		v.Audit = append(v.Audit, a)
	}
	return v, rows.Err()
}

func (s *Server) getNotificationWebhook(w http.ResponseWriter, r *http.Request) {
	v, err := s.readNotificationWebhook(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, v)
}

func auditWebhook(ctx context.Context, tx pgx.Tx, action string, actor *session) error {
	var id *int64
	name := ""
	if actor != nil {
		id, name = &actor.User.ID, actor.User.Username
	}
	_, err := tx.Exec(ctx, `INSERT INTO notification_webhook_events(action,enabled,categories,actor_id,actor_username)
SELECT $1,COALESCE((SELECT enabled FROM notification_webhook WHERE id),false),COALESCE((SELECT categories FROM notification_webhook WHERE id),'{}'),$2,$3`, action, id, name)
	return err
}

// skipStaleDeliveries drops the pending deliveries the current settings no longer allow.
const skipStaleDeliveries = `UPDATE notification_deliveries d SET status='skipped',last_error='webhook_disabled' WHERE status='pending'
AND NOT COALESCE((SELECT enabled AND d.category=ANY(categories) FROM notification_webhook WHERE id),false)`

// putNotificationWebhook saves the settings. An omitted url keeps the stored one; a new one replaces it (rotation).
func (s *Server) putNotificationWebhook(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL        *string  `json:"url"`
		Enabled    *bool    `json:"enabled"`
		Categories []string `json:"categories"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Enabled == nil || in.Categories == nil {
		fail(w, 400, "invalid_request")
		return
	}
	categories := []string{}
	for _, c := range in.Categories {
		if !slices.Contains(webhookCategories, c) {
			fail(w, 400, "invalid_webhook_category")
			return
		}
		if !slices.Contains(categories, c) {
			categories = append(categories, c)
		}
	}
	address, hint := "", ""
	if in.URL != nil {
		var ok bool
		if address, hint, ok = validateWebhookURL(*in.URL); !ok {
			fail(w, 400, "invalid_discord_webhook")
			return
		}
	}
	ss := r.Context().Value(sessionKey{}).(session)
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), `INSERT INTO notification_webhook(id) VALUES(true) ON CONFLICT DO NOTHING`); err != nil {
		s.internal(w, err)
		return
	}
	var storedHint string
	if err = tx.QueryRow(r.Context(), `SELECT hint FROM notification_webhook WHERE id FOR UPDATE`).Scan(&storedHint); err != nil {
		s.internal(w, err)
		return
	}
	store := s.Store.WithTx(tx)
	stored, err := store.Get(r.Context(), webhookURLKey)
	if err != nil {
		s.internal(w, err)
		return
	}
	action := "updated"
	switch {
	case address != "" && stored == "":
		action = "created"
	case address != "" && address != stored:
		action = "rotated"
	case address == "":
		address, hint = stored, storedHint
	}
	if *in.Enabled && address == "" {
		fail(w, 400, "discord_webhook_required")
		return
	}
	if address != stored {
		if err = store.Set(r.Context(), webhookURLKey, address); err != nil {
			s.internal(w, err)
			return
		}
	}
	if _, err = tx.Exec(r.Context(), `UPDATE notification_webhook SET enabled=$1,categories=$2,hint=$3,public_origin=$4,updated_at=now(),
last_status=CASE WHEN $5 THEN 'never' ELSE last_status END,last_error=CASE WHEN $5 THEN '' ELSE last_error END WHERE id`, *in.Enabled, categories, hint, s.publicOrigin(r), action != "updated"); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), skipStaleDeliveries); err != nil {
		s.internal(w, err)
		return
	}
	if err = auditWebhook(r.Context(), tx, action, &ss); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	s.Log.Info("notification webhook "+action, "user_id", ss.User.ID)
	s.getNotificationWebhook(w, r)
}

// deleteNotificationWebhook forgets the address and stops every pending delivery.
func (s *Server) deleteNotificationWebhook(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err = s.Store.WithTx(tx).Set(r.Context(), webhookURLKey, ""); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE notification_webhook SET enabled=false,hint='',last_status='never',last_error='',updated_at=now() WHERE id`); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), skipStaleDeliveries); err != nil {
		s.internal(w, err)
		return
	}
	if err = auditWebhook(r.Context(), tx, "deleted", &ss); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	s.Log.Info("notification webhook deleted", "user_id", ss.User.ID)
	w.WriteHeader(http.StatusNoContent)
}

// testNotificationWebhook posts an explicit test message now and records the outcome as the last delivery status.
func (s *Server) testNotificationWebhook(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	address, err := s.Store.Get(r.Context(), webhookURLKey)
	if err != nil {
		s.internal(w, err)
		return
	}
	if address == "" {
		fail(w, 409, "discord_webhook_not_configured")
		return
	}
	result := s.postWebhook(r.Context(), address, "Snitch : message de test du webhook de notifications.")
	err = pgx.BeginFunc(r.Context(), s.Pool, func(tx pgx.Tx) error {
		if err := recordWebhookAttempt(r.Context(), tx, result); err != nil {
			return err
		}
		return auditWebhook(r.Context(), tx, "tested", &ss)
	})
	if err != nil {
		s.internal(w, err)
		return
	}
	switch result.outcome {
	case webhookSent:
		s.getNotificationWebhook(w, r)
	case webhookRateLimited:
		fail(w, 429, "discord_rate_limited")
	case webhookRejected:
		fail(w, 400, "discord_webhook_rejected")
	default:
		fail(w, 502, "discord_unavailable")
	}
}

type webhookOutcome int

const (
	webhookSent webhookOutcome = iota
	webhookRateLimited
	// The webhook no longer exists or the address is wrong: retrying is useless.
	webhookRejected
	// Discord refused this message (4xx other than the above): retrying is useless too.
	webhookRefused
	webhookTransient
)

type webhookResult struct {
	outcome    webhookOutcome
	code       string
	retryAfter time.Duration
}

// postWebhook posts one message. Mentions are disabled: a summary never pings anybody. The error kept is a code:
// a transport error spells out the address, and the address is the secret.
func (s *Server) postWebhook(ctx context.Context, address, content string) webhookResult {
	if _, _, ok := validateWebhookURL(address); !ok {
		return webhookResult{outcome: webhookRejected, code: "webhook_rejected"}
	}
	body, _ := json.Marshal(map[string]any{"content": content, "username": "Snitch", "allowed_mentions": map[string]any{"parse": []string{}}})
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return webhookResult{outcome: webhookRejected, code: "webhook_rejected"}
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: s.webhookTransport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			return webhookResult{outcome: webhookTransient, code: "timeout"}
		}
		return webhookResult{outcome: webhookTransient, code: "network"}
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return webhookResult{outcome: webhookSent}
	case resp.StatusCode == http.StatusTooManyRequests:
		return webhookResult{outcome: webhookRateLimited, code: "rate_limited", retryAfter: retryAfter(resp.Header, data)}
	case resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 404:
		return webhookResult{outcome: webhookRejected, code: "webhook_rejected"}
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return webhookResult{outcome: webhookRefused, code: "http_" + strconv.Itoa(resp.StatusCode)}
	}
	return webhookResult{outcome: webhookTransient, code: "http_" + strconv.Itoa(resp.StatusCode)}
}

// retryAfter reads Discord's wait (JSON retry_after in seconds, else the Retry-After header), between 1 s and 10 min.
func retryAfter(h http.Header, body []byte) time.Duration {
	var limited struct {
		RetryAfter float64 `json:"retry_after"`
	}
	wait := time.Duration(0)
	if json.Unmarshal(body, &limited) == nil && limited.RetryAfter > 0 {
		wait = time.Duration(limited.RetryAfter * float64(time.Second))
	} else if n, err := strconv.ParseFloat(h.Get("Retry-After"), 64); err == nil && n > 0 {
		wait = time.Duration(n * float64(time.Second))
	}
	return min(max(wait, time.Second), 10*time.Minute)
}

func recordWebhookAttempt(ctx context.Context, tx pgx.Tx, result webhookResult) error {
	status := map[webhookOutcome]string{webhookSent: "sent", webhookRateLimited: "rate_limited"}[result.outcome]
	if status == "" {
		status = "failed"
	}
	if _, err := tx.Exec(ctx, `INSERT INTO notification_webhook(id) VALUES(true) ON CONFLICT DO NOTHING`); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE notification_webhook SET last_status=$1,last_error=$2,last_attempt_at=now(),last_success_at=CASE WHEN $1='sent' THEN now() ELSE last_success_at END WHERE id`, status, result.code)
	return err
}

// webhookLine is the redacted summary of a notification: what kind of thing happened and where to look in Snitch.
// Never a name, a clip, a transcription, an error text or a media link.
func webhookLine(kind string, count int) (string, string) {
	switch kind {
	case kindAccountRequest:
		return "Nouvelle demande d'accès en attente de validation.", "/admin/users/signups"
	case kindSpeakerReview:
		return "Des voix attendent une vérification.", "/people/verification"
	case "worker_unavailable":
		return "Un Worker est hors ligne.", "/admin/workers"
	case "job_blocked":
		return fmt.Sprintf("Des traitements sont bloqués en attente d'une action (%d).", count), "/admin/workers"
	case "job_failed", "provider_unavailable", "model_missing":
		return fmt.Sprintf("Un traitement a échoué (%d).", count), "/queue"
	}
	// A kind added to notification_category() without its summary here: say only that something happened.
	return "Une notification attend dans Snitch.", "/notifications"
}

func webhookMessage(origin string, kinds []string, counts []int) string {
	var lines []string
	repeats := map[string]int{}
	for i, kind := range kinds {
		text, path := webhookLine(kind, counts[i])
		if origin != "" {
			text += " <" + origin + path + ">"
		}
		if repeats[text] == 0 {
			lines = append(lines, text)
		}
		repeats[text]++
	}
	// Identical summaries of one batch are posted once, with their number.
	for i, text := range lines {
		if n := repeats[text]; n > 1 {
			lines[i] = text + " (×" + strconv.Itoa(n) + ")"
		}
	}
	return strings.Join(lines, "\n")
}

// serveNotificationWebhook posts the due deliveries until ctx ends.
func (s *Server) serveNotificationWebhook(ctx context.Context) {
	tick := s.webhookTick
	if tick == 0 {
		tick = 5 * time.Second
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		// A bounded number of batches per tick: a failing batch is rescheduled, so it is never due twice in a row.
		for range 10 {
			n, err := s.deliverNotificationBatch(ctx)
			if err != nil {
				if ctx.Err() == nil {
					s.Log.Error("notification webhook delivery failed", "error", err)
				}
				break
			}
			if n == 0 {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// deliverNotificationBatch claims up to webhookBatch due deliveries, posts them as one message and records the
// outcome. The claim is a lease (next attempt pushed forward), committed before the call: no transaction is held
// open while Discord answers, and another instance never posts the same delivery.
func (s *Server) deliverNotificationBatch(ctx context.Context) (int, error) {
	address, err := s.Store.Get(ctx, webhookURLKey)
	if err != nil {
		return 0, err
	}
	var origin string
	var enabled bool
	err = s.Pool.QueryRow(ctx, `SELECT enabled,public_origin FROM notification_webhook WHERE id`).Scan(&enabled, &origin)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	if !enabled || address == "" {
		_, err = s.Pool.Exec(ctx, `UPDATE notification_deliveries SET status='skipped',last_error='webhook_disabled' WHERE status='pending'`)
		return 0, err
	}
	rows, err := s.Pool.Query(ctx, `UPDATE notification_deliveries d SET next_attempt_at=now()+$2*interval '1 microsecond',attempts=d.attempts+1
FROM notifications n WHERE n.id=d.notification_id AND d.id IN (SELECT id FROM notification_deliveries WHERE status='pending' AND next_attempt_at<=now() ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED)
RETURNING d.id,n.kind,n.pending_count`, webhookBatch, webhookLease.Microseconds())
	if err != nil {
		return 0, err
	}
	var ids []int64
	var kinds []string
	var counts []int
	for rows.Next() {
		var id int64
		var kind string
		var count int
		if err = rows.Scan(&id, &kind, &count); err != nil {
			rows.Close()
			return 0, err
		}
		ids, kinds, counts = append(ids, id), append(kinds, kind), append(counts, count)
	}
	rows.Close()
	if err = rows.Err(); err != nil || len(ids) == 0 {
		return 0, err
	}
	result := s.postWebhook(ctx, address, webhookMessage(origin, kinds, counts))
	base := s.webhookRetryBase
	if base == 0 {
		base = 30 * time.Second
	}
	// The outcome is recorded even when the server stops meanwhile: the call happened.
	ctx = context.WithoutCancel(ctx)
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		switch result.outcome {
		case webhookSent:
			_, err = tx.Exec(ctx, `UPDATE notification_deliveries SET status='sent',sent_at=now(),last_error='' WHERE id=ANY($1)`, ids)
		case webhookRateLimited:
			// Discord asked to wait: not a failed attempt. The whole queue waits, or the next batch would hit the
			// same limit at once.
			if _, err = tx.Exec(ctx, `UPDATE notification_deliveries SET attempts=attempts-1,last_error=$2 WHERE id=ANY($1)`, ids, result.code); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE notification_deliveries SET next_attempt_at=GREATEST(next_attempt_at,now()+$1*interval '1 microsecond') WHERE status='pending'`, result.retryAfter.Microseconds())
		case webhookTransient:
			// Exponential backoff from the base, bounded; failed after the last attempt.
			_, err = tx.Exec(ctx, `UPDATE notification_deliveries SET last_error=$2,status=CASE WHEN attempts>=$3 THEN 'failed' ELSE 'pending' END,
next_attempt_at=now()+LEAST($4*power(2,attempts-1),$5)*interval '1 microsecond' WHERE id=ANY($1)`, ids, result.code, webhookMaxAttempts, base.Microseconds(), webhookMaxRetryWait.Microseconds())
		case webhookRejected:
			// The webhook is gone (deleted on Discord's side) or wrong: stop instead of failing forever.
			if _, err = tx.Exec(ctx, `UPDATE notification_deliveries SET status='failed',last_error=$2 WHERE id=ANY($1)`, ids, result.code); err != nil {
				return err
			}
			tag, err := tx.Exec(ctx, `UPDATE notification_webhook SET enabled=false,updated_at=now() WHERE id AND enabled`)
			if err != nil {
				return err
			}
			// Audited once: another batch may have disabled it already.
			if tag.RowsAffected() == 1 {
				if err = auditWebhook(ctx, tx, "auto_disabled", nil); err != nil {
					return err
				}
			}
			_, err = tx.Exec(ctx, skipStaleDeliveries)
		default:
			_, err = tx.Exec(ctx, `UPDATE notification_deliveries SET status='failed',last_error=$2 WHERE id=ANY($1)`, ids, result.code)
		}
		if err != nil {
			return err
		}
		return recordWebhookAttempt(ctx, tx, result)
	})
	if result.outcome != webhookSent {
		s.Log.Warn("notification webhook not delivered", "code", result.code, "deliveries", len(ids))
	}
	return len(ids), err
}
