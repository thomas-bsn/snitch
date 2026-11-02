// Package logging builds the structured logger shared by the server's commands.
//
// The schema (timestamp, level, service, event, then context fields) is the same as the Worker's,
// so one job can be followed across both services. Keys that could carry a secret or private content
// are masked at every level, DEBUG included.
package logging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const masked = "[masked]"

// forbidden lists key fragments (lowercase) whose value is never written.
var forbidden = []string{"authorization", "token", "cookie", "password", "secret", "signed_url", "transcript", "embedding", "fingerprint", "api_key", "apikey", "bearer", "credential"}

// rawErrorKeys carry free error text when given a string: never written, use Failure instead.
var rawErrorKeys = map[string]bool{"error": true, "err": true, "reason": true, "exception": true, "detail": true}

// ParseLevel maps ERROR, WARN, INFO or DEBUG (any case) to a level; anything else is INFO.
func ParseLevel(value string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ERROR":
		return slog.LevelError
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "DEBUG":
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// levelNotice sits above ERROR so a notice passes every LOG_LEVEL.
const levelNotice = slog.Level(12)

// InstallationCode prints the one-time installation code whatever LOG_LEVEL is. It is the only
// value that is deliberately visible (ADR 0006): without it the first installation is impossible.
func InstallationCode(l *slog.Logger, address, code string) {
	l.Log(context.Background(), levelNotice, "one-time installation code: open "+address+" and enter "+code+" (valid 15 minutes, works once)",
		"code", code, "url", address, "expires_in_minutes", 15)
}

// Failure describes an error as a stable code and a category, never its text: SQL and provider
// messages can echo data, URLs or tokens.
func Failure(err error) []any {
	code, category := "operation_error", "internal"
	var pg *pgconn.PgError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.DeadlineExceeded):
		code, category = "timeout", "timeout"
	case errors.Is(err, context.Canceled):
		code, category = "canceled", "timeout"
	case errors.As(err, &pg):
		code, category = pg.Code, "database"
	case errors.Is(err, pgx.ErrNoRows):
		code, category = "no_rows", "database"
	}
	return []any{"error_code", code, "error_category", category}
}

func sensitive(key string) bool {
	key = strings.ToLower(key)
	for _, fragment := range forbidden {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

// New returns a JSON logger writing to w for the named service at the given LOG_LEVEL value.
func New(w io.Writer, service, level string) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: ParseLevel(level),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 {
				switch a.Key {
				case slog.TimeKey:
					a.Key = "timestamp"
					return a
				case slog.MessageKey:
					a.Key = "event"
					return a
				case slog.LevelKey:
					if a.Value.Any() == levelNotice {
						return slog.String(a.Key, "NOTICE")
					}
					return a
				}
			}
			if sensitive(a.Key) {
				return slog.String(a.Key, masked)
			}
			if a.Value.Kind() == slog.KindString && rawErrorKeys[strings.ToLower(a.Key)] {
				return slog.String(a.Key, masked)
			}
			// Safety net for a raw error passed as a value: only its code and category survive.
			if a.Value.Kind() == slog.KindAny {
				if err, ok := a.Value.Any().(error); ok {
					f := Failure(err)
					return slog.Group(a.Key, "code", f[1], "category", f[3])
				}
			}
			return a
		},
	})
	return slog.New(handler).With("service", service)
}
