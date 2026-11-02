// Package events is the generic server-sent event mechanism: an outbox table says when a
// topic changed, and a stream answers each change with a fresh snapshot of the state. A
// snapshot is always the full current state, so reconnecting, missing events or restarting
// the server all converge on the same view. Rights are rechecked before every snapshot.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrForbidden ends a stream: the viewer lost access to the topic.
var ErrForbidden = errors.New("access to the topic was lost")

type Service struct {
	pool *pgxpool.Pool
	log  *slog.Logger
	// Poll is how often a stream looks for new events; Heartbeat keeps idle streams and
	// proxies alive and rechecks access; Retention bounds the outbox.
	Poll, Heartbeat, Retention time.Duration
}

func New(pool *pgxpool.Pool, log *slog.Logger) *Service {
	return &Service{pool: pool, log: log, Poll: time.Second, Heartbeat: 15 * time.Second, Retention: 24 * time.Hour}
}

// Topic describes one stream. Authorize and Snapshot run at connection and before every
// snapshot; Snapshot returns the JSON-encodable state, already free of secrets.
type Topic struct {
	Name      string
	Authorize func(ctx context.Context) error
	Snapshot  func(ctx context.Context) (any, error)
}

func (s *Service) latest(ctx context.Context, topic string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(max(id),0) FROM events WHERE topic=$1`, topic).Scan(&id)
	return id, err
}

// Prune deletes events older than the retention.
func (s *Service) Prune(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM events WHERE created_at < now() - make_interval(secs => $1)`, s.Retention.Seconds())
	return err
}

// Serve prunes the outbox periodically until ctx ends.
func (s *Service) Serve(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		if err := s.Prune(ctx); err != nil && ctx.Err() == nil {
			s.log.Warn("prune events", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type frame struct {
	Seq      int64 `json:"seq"`
	Snapshot any   `json:"snapshot"`
}

// Stream serves a text/event-stream: a snapshot at once (its sequence is the last event
// seen before the state is read, so a change racing the read triggers another snapshot),
// then one snapshot per change. Last-Event-ID is not needed: the first snapshot is always
// complete, which makes reconnection safe without replaying events.
func (s *Service) Stream(w http.ResponseWriter, r *http.Request, t Topic) {
	ctx := r.Context()
	rc := http.NewResponseController(w)
	if err := t.Authorize(ctx); err != nil {
		status, code := http.StatusForbidden, "forbidden"
		if !errors.Is(err, ErrForbidden) {
			status, code = http.StatusInternalServerError, "internal_error"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprintf(w, `{"error":%q}`+"\n", code)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	// The server write timeout is meant for ordinary responses: each write gets its own deadline.
	write := func(format string, args ...any) error {
		_ = rc.SetWriteDeadline(time.Now().Add(s.Heartbeat + 15*time.Second))
		if _, err := fmt.Fprintf(w, format, args...); err != nil {
			return err
		}
		return rc.Flush()
	}
	if err := write("retry: 3000\n\n"); err != nil {
		return
	}
	var seen int64
	forbidden := func(err error) {
		if errors.Is(err, ErrForbidden) {
			_ = write("event: forbidden\ndata: {}\n\n")
		}
	}
	send := func() error {
		if err := t.Authorize(ctx); err != nil {
			forbidden(err)
			return err
		}
		seq, err := s.latest(ctx, t.Name)
		if err != nil {
			return err
		}
		snap, err := t.Snapshot(ctx)
		if err != nil {
			return err
		}
		body, err := json.Marshal(frame{Seq: seq, Snapshot: snap})
		if err != nil {
			return err
		}
		seen = seq
		return write("id: %d\nevent: snapshot\ndata: %s\n\n", seq, body)
	}
	if send() != nil {
		return
	}
	poll, beat := time.NewTicker(s.Poll), time.NewTicker(s.Heartbeat)
	defer poll.Stop()
	defer beat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-poll.C:
			latest, err := s.latest(ctx, t.Name)
			if err != nil {
				return
			}
			if latest > seen && send() != nil {
				return
			}
		case <-beat.C:
			// Access is rechecked even when nothing changed.
			if err := t.Authorize(ctx); err != nil {
				forbidden(err)
				return
			}
			if write(": ping\n\n") != nil {
				return
			}
		}
	}
}
