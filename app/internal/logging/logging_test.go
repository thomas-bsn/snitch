package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

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
