//go:build integration

package httpapi

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// logEvents returns the structured log lines whose event matches.
func logEvents(t *testing.T, raw, event string) []map[string]any {
	t.Helper()
	var found []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil && m["event"] == event {
			found = append(found, m)
		}
	}
	return found
}

func hasLog(entries []map[string]any, want map[string]any) bool {
	for _, entry := range entries {
		match := true
		for k, v := range want {
			if entry[k] != v {
				match = false
			}
		}
		if match {
			return true
		}
	}
	return false
}
