package workers

import (
	"testing"
	"time"
)

func TestRestartTransitionsAreMonotonic(t *testing.T) {
	allowed := [][2]string{{"requested", "accepted"}, {"accepted", "stopping"}, {"stopping", "starting"}, {"starting", "online"}, {"starting", "failed"}, {"requested", "expired"}}
	for _, pair := range allowed {
		if !RestartTransitionAllowed(pair[0], pair[1]) {
			t.Errorf("transition %s -> %s should be allowed", pair[0], pair[1])
		}
	}
	for _, pair := range [][2]string{{"online", "starting"}, {"failed", "accepted"}, {"requested", "online"}, {"accepted", "online"}, {"starting", "stopping"}} {
		if RestartTransitionAllowed(pair[0], pair[1]) {
			t.Errorf("transition %s -> %s should be rejected", pair[0], pair[1])
		}
	}
}

func TestRestartExpiryBoundary(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if RestartExpired(now, now.Add(time.Second)) || !RestartExpired(now, now) || !RestartExpired(now, now.Add(-time.Second)) {
		t.Fatal("expiry boundary is incorrect")
	}
}
