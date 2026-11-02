package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// trapped is a synthetic scenario mixing forbidden keys, a SQL error and a provider error.
func trapped(l *slog.Logger) {
	sqlErr := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint \"users_email_key\"", Detail: "Key (email)=(ann@example.test) already exists."}
	providerErr := errors.New("provider said: invalid key sk-synthetic-123 for https://api.example.test/v1?sig=abc")
	for _, emit := range []func(string, ...any){l.Debug, l.Info, l.Warn, l.Error} {
		emit("trap", "hf_token", "hf_synthetic", "Authorization", "Bearer synthetic", "transcript", "private words",
			"error", sqlErr, "err", providerErr, "job_id", 9)
		emit("trap_failure", Failure(sqlErr)...)
		emit("trap_failure", Failure(providerErr)...)
	}
}

func TestTrappedScenarioLeaksNothingAtAnyLevel(t *testing.T) {
	for _, level := range []string{"DEBUG", "INFO", "WARN", "ERROR"} {
		var buf bytes.Buffer
		trapped(New(&buf, "snitch", level))
		out := buf.String()
		for _, secret := range []string{"hf_synthetic", "Bearer synthetic", "private words", "users_email_key", "ann@example.test", "sk-synthetic", "sig=abc", "provider said", "duplicate key"} {
			if strings.Contains(out, secret) {
				t.Fatalf("level %s: %q leaked: %s", level, secret, out)
			}
		}
		if !strings.Contains(out, `"job_id":9`) {
			t.Fatalf("level %s: allowed field lost: %s", level, out)
		}
	}
}

func TestFailureGivesCodeAndCategory(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "snitch", "INFO")
	l.Error("a", Failure(&pgconn.PgError{Code: "23505", Message: "x"})...)
	l.Error("b", Failure(context.DeadlineExceeded)...)
	l.Error("c", Failure(errors.New("whatever"))...)
	got := lines(t, &buf)
	want := [][2]string{{"23505", "database"}, {"timeout", "timeout"}, {"operation_error", "internal"}}
	for i, w := range want {
		if got[i]["error_code"] != w[0] || got[i]["error_category"] != w[1] {
			t.Fatalf("line %d: %v, want %v", i, got[i], w)
		}
	}
}

// ADR 0006: the one-time installation code is the only value shown whatever LOG_LEVEL is.
func TestInstallationCodeIsShownAtEveryLevel(t *testing.T) {
	for _, level := range []string{"ERROR", "WARN", "INFO", "DEBUG", "", "garbage"} {
		var buf bytes.Buffer
		InstallationCode(New(&buf, "snitch", level), "http://localhost:8080", "ABCD-1234")
		out := buf.String()
		if !strings.Contains(out, "ABCD-1234") || !strings.Contains(out, "http://localhost:8080") {
			t.Fatalf("level %q: installation code not shown: %q", level, out)
		}
	}
	var buf bytes.Buffer
	New(&buf, "snitch", "ERROR").Warn("other", "code", "ABCD-1234")
	if buf.Len() != 0 {
		t.Fatal("only the installation notice bypasses LOG_LEVEL")
	}
}

func lines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("not JSON: %q", line)
		}
		out = append(out, m)
	}
	return out
}

func TestSharedSchema(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, "snitch", "INFO").Info("job_claimed", "job_id", 7, "clip_id", 3)
	got := lines(t, &buf)
	if len(got) != 1 || got[0]["event"] != "job_claimed" || got[0]["service"] != "snitch" || got[0]["level"] != "INFO" || got[0]["timestamp"] == nil || got[0]["job_id"] != float64(7) {
		t.Fatalf("unexpected schema: %v", got)
	}
}

func TestLevelControlsVolume(t *testing.T) {
	for level, want := range map[string]int{"ERROR": 1, "WARN": 2, "INFO": 3, "DEBUG": 4, "": 3, "garbage": 3, "debug": 4} {
		var buf bytes.Buffer
		l := New(&buf, "snitch", level)
		l.Debug("d")
		l.Info("i")
		l.Warn("w")
		l.Error("e")
		if got := len(lines(t, &buf)); got != want {
			t.Fatalf("level %q: %d lines, want %d", level, got, want)
		}
	}
}

func TestForbiddenKeysAreMaskedAtEveryLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "snitch", "DEBUG")
	l.Debug("probe", "Authorization", "Bearer abc", "lease_token", "lt_secret", "hf_token", "hf_x", "cookie", "c=1",
		"transcript", "private words", "embedding", []float64{1}, "signed_url", "https://x/?sig=1", "group", slog.GroupValue(slog.String("token", "nested-secret")), "job_id", 4)
	out := buf.String()
	for _, secret := range []string{"Bearer abc", "lt_secret", "hf_x", "c=1", "private words", "sig=1", "nested-secret"} {
		if strings.Contains(out, secret) {
			t.Fatalf("secret %q leaked: %s", secret, out)
		}
	}
	if !strings.Contains(out, `"job_id":4`) {
		t.Fatalf("allowed field lost: %s", out)
	}
}
