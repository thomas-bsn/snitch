package workers

import (
	"testing"
	"time"
)

func TestInstallProgressStates(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	created := now.Add(-time.Hour)
	seen := now.Add(-time.Second)
	e := func(phase, status, cause string, ago time.Duration) InstallEvent {
		at := now.Add(-ago)
		return InstallEvent{Phase: phase, Status: status, CauseCode: cause, OccurredAt: at, ReceivedAt: at}
	}
	cases := []struct {
		name    string
		created time.Time
		seen    *time.Time
		events  []InstallEvent
		state   string
		voice   string
		noNews  bool
	}{
		{"fresh", now.Add(-time.Minute), nil, nil, "installing", "pending", false},
		{"fresh but silent", now.Add(-time.Hour), nil, []InstallEvent{e("image", "started", "", 30*time.Minute)}, "installing", "pending", true},
		{"failure", created, nil, []InstallEvent{e("image", "failed", "image_pull_failed", time.Minute)}, "failed", "pending", false},
		{"failure recovered", created, nil, []InstallEvent{e("image", "failed", "image_pull_failed", 2*time.Minute), e("image", "succeeded", "", time.Minute)}, "installing", "pending", false},
		{"connected", created, &seen, nil, "connected", "pending", false},
		{"voice model denied keeps connected", created, &seen, []InstallEvent{e("voice-models", "failed", "hf_access_denied", time.Minute)}, "connected", "unavailable", false},
		{"models downloaded is not voice ready", created, &seen, []InstallEvent{e("voice-models", "succeeded", "", time.Minute)}, "connected", "pending", false},
		{"voice ready after synthetic inference", created, &seen, []InstallEvent{e("voice-models", "succeeded", "", 2*time.Minute), e("voice-test", "succeeded", "", time.Minute)}, "voice_ready", "ready", false},
		{"never connected", now.Add(-25 * time.Hour), nil, []InstallEvent{e("image", "started", "", 24*time.Hour)}, "never_connected", "pending", false},
		{"pulse beats 24h", now.Add(-25 * time.Hour), &seen, nil, "connected", "pending", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := ComputeInstallProgress(c.created, c.seen, c.seen, c.events, now)
			if p.State != c.state || p.Voice.State != c.voice || p.NoNews != c.noNews {
				t.Fatalf("got state=%s voice=%s noNews=%v", p.State, p.Voice.State, p.NoNews)
			}
			if len(c.events) > 0 && (p.LastEvent == nil || p.LastEvent.Phase != c.events[len(c.events)-1].Phase) {
				t.Fatal("last event must stay visible whatever the state")
			}
		})
	}
}

func TestValidInstallEvent(t *testing.T) {
	now := time.Now()
	ok := InstallEventInput{Phase: "gpu", Status: "failed", CauseCode: "gpu_unavailable", OccurredAt: now}
	if !ValidInstallEvent(ok, now) {
		t.Fatal("valid event refused")
	}
	for name, e := range map[string]InstallEventInput{
		"phase":       {Phase: "x", Status: "started", OccurredAt: now},
		"free cause":  {Phase: "gpu", Status: "failed", CauseCode: "it broke", OccurredAt: now},
		"cause on ok": {Phase: "gpu", Status: "succeeded", CauseCode: "unknown", OccurredAt: now},
		"future":      {Phase: "gpu", Status: "started", OccurredAt: now.Add(time.Hour)},
		"zero time":   {Phase: "gpu", Status: "started"},
	} {
		if ValidInstallEvent(e, now) {
			t.Errorf("%s accepted", name)
		}
	}
}
