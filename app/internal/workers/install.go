package workers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

// Installation events carry only a phase, a status, a stable cause code and the original timestamp.
// There is no free text and no secret by construction.
var (
	InstallPhases   = []string{"prerequisites", "registration", "image", "permissions", "gpu", "voice-models", "voice-test", "service"}
	InstallStatuses = []string{"started", "succeeded", "failed"}
	InstallCauses   = []string{"unsupported_platform", "docker_unavailable", "registration_rejected", "network_unreachable", "image_pull_failed", "volume_not_writable", "hf_cache_not_writable", "gpu_unavailable", "service_start_failed", "hf_access_denied", "unknown"}
)

const (
	// NeverConnectedAfter marks a Worker without any pulse since its creation.
	NeverConnectedAfter = 24 * time.Hour
	// ReceivedLaterAfter is the delay after which an event counts as queued offline.
	ReceivedLaterAfter = 2 * time.Minute
	// NoNewsAfter is the silence after which "no news" is shown next to the last event.
	NoNewsAfter       = 10 * time.Minute
	maxInstallBatch   = 50
	maxFutureSkew     = 5 * time.Minute
	keptInstallEvents = 200
)

type InstallEventInput struct {
	Phase      string    `json:"phase"`
	Status     string    `json:"status"`
	CauseCode  string    `json:"cause_code"`
	OccurredAt time.Time `json:"occurred_at"`
}

type InstallEvent struct {
	Phase         string    `json:"phase"`
	Status        string    `json:"status"`
	CauseCode     string    `json:"cause_code"`
	OccurredAt    time.Time `json:"occurred_at"`
	ReceivedAt    time.Time `json:"received_at"`
	ReceivedLater bool      `json:"received_later"`
}

// InstallVoice is the voice capability, independent of transcription.
type InstallVoice struct {
	// State: ready (synthetic inference passed), unavailable (with CauseCode), pending (nothing known yet).
	State     string `json:"state"`
	CauseCode string `json:"cause_code"`
}

type InstallProgress struct {
	// State: installing, connected, voice_ready, failed, never_connected.
	State string `json:"state"`
	// FailedPhase and CauseCode describe the failure when State is failed.
	FailedPhase string        `json:"failed_phase"`
	CauseCode   string        `json:"cause_code"`
	Voice       InstallVoice  `json:"voice"`
	LastEvent   *InstallEvent `json:"last_event"`
	// LastActivityAt is the latest event or pulse; NoNews is the absence of news, distinct from the last event.
	LastActivityAt *time.Time     `json:"last_activity_at"`
	NoNews         bool           `json:"no_news"`
	Events         []InstallEvent `json:"events,omitempty"`
}

var ErrInvalidInstallEvent = errors.New("invalid install event")

// ValidInstallEvent checks the closed vocabulary and a plausible original timestamp.
func ValidInstallEvent(e InstallEventInput, now time.Time) bool {
	if !slices.Contains(InstallPhases, e.Phase) || !slices.Contains(InstallStatuses, e.Status) || e.OccurredAt.IsZero() || e.OccurredAt.After(now.Add(maxFutureSkew)) {
		return false
	}
	if e.Status == "failed" {
		return slices.Contains(InstallCauses, e.CauseCode)
	}
	return e.CauseCode == ""
}

// RecordInstallEvents stores a batch atomically; one invalid event rejects the whole batch.
func (s *Service) RecordInstallEvents(ctx context.Context, workerID string, events []InstallEventInput) error {
	now := time.Now()
	if len(events) == 0 || len(events) > maxInstallBatch {
		return ErrInvalidInstallEvent
	}
	for _, e := range events {
		if !ValidInstallEvent(e, now) {
			return ErrInvalidInstallEvent
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin install events: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, e := range events {
		if _, err = tx.Exec(ctx, `INSERT INTO worker_install_events(worker_id,phase,status,cause_code,occurred_at) VALUES($1,$2,$3,$4,$5)`, workerID, e.Phase, e.Status, e.CauseCode, e.OccurredAt); err != nil {
			return fmt.Errorf("insert install event: %w", err)
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM worker_install_events WHERE worker_id=$1 AND id < (SELECT COALESCE(min(id),0) FROM (SELECT id FROM worker_install_events WHERE worker_id=$1 ORDER BY id DESC LIMIT $2) k)`, workerID, keptInstallEvents); err != nil {
		return fmt.Errorf("trim install events: %w", err)
	}
	return tx.Commit(ctx)
}

// ComputeInstallProgress derives the admin-visible state. events are ordered by occurrence, oldest first.
// firstPulse is the first real pulse; launcher calls made with the token (configuration, events) only count as activity.
func ComputeInstallProgress(created time.Time, firstPulse, lastSeen *time.Time, events []InstallEvent, now time.Time) InstallProgress {
	p := InstallProgress{Voice: InstallVoice{State: "pending"}}
	var lastFailure *InstallEvent
	for i := range events {
		e := events[i]
		p.LastEvent = &events[i]
		voice := e.Phase == "voice-test" || e.Phase == "voice-models"
		switch {
		case e.Phase == "voice-test" && e.Status == "succeeded":
			p.Voice = InstallVoice{State: "ready"}
		case voice && e.Status == "failed":
			p.Voice = InstallVoice{State: "unavailable", CauseCode: e.CauseCode}
		case e.Phase == "voice-models" && e.Status == "started":
			p.Voice = InstallVoice{State: "pending"}
		}
		if e.Status == "failed" && !voice {
			lastFailure = &events[i]
		} else if e.Status == "succeeded" && lastFailure != nil && e.Phase == lastFailure.Phase {
			lastFailure = nil
		}
		if p.LastActivityAt == nil || e.ReceivedAt.After(*p.LastActivityAt) {
			t := e.ReceivedAt
			p.LastActivityAt = &t
		}
	}
	if lastSeen != nil && (p.LastActivityAt == nil || lastSeen.After(*p.LastActivityAt)) {
		t := *lastSeen
		p.LastActivityAt = &t
	}
	switch {
	case firstPulse != nil && p.Voice.State == "ready":
		p.State = "voice_ready"
	case firstPulse != nil:
		p.State = "connected"
	case now.Sub(created) > NeverConnectedAfter:
		p.State = "never_connected"
	case lastFailure != nil:
		p.State = "failed"
	default:
		p.State = "installing"
	}
	if p.State == "failed" {
		p.FailedPhase, p.CauseCode = lastFailure.Phase, lastFailure.CauseCode
	}
	last := created
	if p.LastActivityAt != nil {
		last = *p.LastActivityAt
	}
	p.NoNews = (p.State == "installing" || p.State == "failed") && now.Sub(last) > NoNewsAfter
	return p
}

// InstallProgress reads events for the given Workers; withEvents adds the recent history.
func (s *Service) InstallProgress(ctx context.Context, ws []Worker, withEvents bool) (map[string]InstallProgress, error) {
	ids := make([]string, 0, len(ws))
	for _, w := range ws {
		ids = append(ids, w.ID)
	}
	rows, err := s.pool.Query(ctx, `SELECT worker_id,phase,status,cause_code,occurred_at,received_at FROM worker_install_events WHERE worker_id = ANY($1) ORDER BY occurred_at,id`, ids)
	if err != nil {
		return nil, fmt.Errorf("read install events: %w", err)
	}
	defer rows.Close()
	byWorker := map[string][]InstallEvent{}
	for rows.Next() {
		var id string
		var e InstallEvent
		if err = rows.Scan(&id, &e.Phase, &e.Status, &e.CauseCode, &e.OccurredAt, &e.ReceivedAt); err != nil {
			return nil, fmt.Errorf("scan install event: %w", err)
		}
		e.ReceivedLater = e.ReceivedAt.Sub(e.OccurredAt) > ReceivedLaterAfter
		byWorker[id] = append(byWorker[id], e)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read install event rows: %w", err)
	}
	pulses := map[string]*time.Time{}
	pr, err := s.pool.Query(ctx, `SELECT id,first_pulse_at FROM workers WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("read first pulses: %w", err)
	}
	defer pr.Close()
	for pr.Next() {
		var id string
		var at *time.Time
		if err = pr.Scan(&id, &at); err != nil {
			return nil, fmt.Errorf("scan first pulse: %w", err)
		}
		pulses[id] = at
	}
	if err = pr.Err(); err != nil {
		return nil, fmt.Errorf("read first pulse rows: %w", err)
	}
	now := time.Now()
	out := make(map[string]InstallProgress, len(ws))
	for _, w := range ws {
		p := ComputeInstallProgress(w.CreatedAt, pulses[w.ID], w.LastSeenAt, byWorker[w.ID], now)
		if withEvents {
			ev := byWorker[w.ID]
			if len(ev) > 30 {
				ev = ev[len(ev)-30:]
			}
			p.Events = ev
		}
		out[w.ID] = p
	}
	return out, nil
}
