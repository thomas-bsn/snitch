package httpapi

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestDiscordStepFailedLogsOnlySafeFields(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		err    error
		want   []string
	}{
		{"token error code", 400, `{"error":"invalid_client","error_description":"leak-secret"}`, nil, []string{`"discord_error":"invalid_client"`, `"status":400`}},
		{"profile numeric code", 401, `{"code":0,"message":"401: Unauthorized leak-secret"}`, nil, []string{`"discord_error":"0"`, `"status":401`}},
		{"hostile code ignored", 400, `{"error":"leak-secret with spaces"}`, nil, []string{`"status":400`}},
		{"transport timeout", 0, ``, context.DeadlineExceeded, []string{`"error_kind":"timeout"`}},
		{"transport opaque", 0, ``, errors.New(`Post "https://discord.com/api/oauth2/token?code=leak-secret": boom`), []string{`"error_kind":"operation_error"`}},
	}
	for _, c := range cases {
		var logs bytes.Buffer
		s := &Server{Log: slog.New(slog.NewJSONHandler(&logs, nil))}
		s.discordStepFailed("token_exchange", "http_status", c.status, []byte(c.body), c.err)
		out := logs.String()
		for _, w := range append(c.want, `"step":"token_exchange"`) {
			if !strings.Contains(out, w) {
				t.Errorf("%s: %q missing in %s", c.name, w, out)
			}
		}
		if strings.Contains(out, "leak-secret") {
			t.Errorf("%s: leaked: %s", c.name, out)
		}
	}
}
