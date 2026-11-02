// Package logging builds the structured logger shared by the server's commands.
//
// The schema (timestamp, level, service, event, then context fields) is the same as the Worker's,
// so one job can be followed across both services. Keys that could carry a secret or private content
// are masked at every level, DEBUG included.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

const masked = "[masked]"

// forbidden lists key fragments (lowercase) whose value is never written.
var forbidden = []string{"authorization", "token", "cookie", "password", "secret", "signed_url", "transcript", "embedding", "fingerprint"}

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
					return a
				}
			}
			if sensitive(a.Key) {
				return slog.String(a.Key, masked)
			}
			return a
		},
	})
	return slog.New(handler).With("service", service)
}
