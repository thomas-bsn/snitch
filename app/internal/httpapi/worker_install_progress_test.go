//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type installView struct {
	State       string `json:"state"`
	FailedPhase string `json:"failed_phase"`
	CauseCode   string `json:"cause_code"`
	Voice       struct {
		State     string `json:"state"`
		CauseCode string `json:"cause_code"`
	} `json:"voice"`
	LastEvent *struct {
		Phase         string `json:"phase"`
		Status        string `json:"status"`
		ReceivedLater bool   `json:"received_later"`
	} `json:"last_event"`
	NoNews bool `json:"no_news"`
	Events []struct {
		Phase         string `json:"phase"`
		ReceivedLater bool   `json:"received_later"`
	} `json:"events"`
}

// install reads the admin view of one Worker, from the detail and from the list, which must agree.
func (h *installHarness) install(id string) installView {
	h.t.Helper()
	var detail struct {
		Install installView `json:"install"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/workers/"+id, nil, http.StatusOK), &detail); err != nil {
		h.t.Fatal(err)
	}
	var list struct {
		Items []struct {
			ID      string      `json:"id"`
			Install installView `json:"install"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/workers", nil, http.StatusOK), &list); err != nil {
		h.t.Fatal(err)
	}
	for _, it := range list.Items {
		if it.ID == id && it.Install.State != detail.Install.State {
			h.t.Fatalf("list and detail disagree: %s vs %s", it.Install.State, detail.Install.State)
		}
	}
	return detail.Install
}

func (h *installHarness) events(token string, want int, events ...map[string]any) []byte {
	h.t.Helper()
	return h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/install-events", mustJSON(h.t, map[string]any{"events": events}), "", token, false, want)
}

func ev(phase, status, cause string, at time.Time) map[string]any {
	m := map[string]any{"phase": phase, "status": status, "occurred_at": at.UTC().Format(time.RFC3339Nano)}
	if cause != "" {
		m["cause_code"] = cause
	}
	return m
}

func testWorkerInstallProgress(t *testing.T, pool *pgxpool.Pool) {
	h := newInstallHarness(t, pool, true)
	id, token, _, _ := h.register("")
	now := time.Now()

	// Fresh Worker, transcription unconfigured: installing, not failed.
	v := h.install(id)
	if v.State != "installing" || v.LastEvent != nil || v.Voice.State != "pending" {
		t.Fatalf("fresh worker: %+v", v)
	}

	// Event contract: closed vocabulary, no free text, cause only on failure, authenticated.
	h.events(token, http.StatusBadRequest, ev("nope", "started", "", now))
	h.events(token, http.StatusBadRequest, ev("image", "failed", "", now))
	h.events(token, http.StatusBadRequest, ev("image", "failed", "free text here", now))
	h.events(token, http.StatusBadRequest, ev("image", "succeeded", "unknown", now))
	h.events(token, http.StatusBadRequest, ev("image", "started", "", now.Add(time.Hour)))
	bad := ev("image", "started", "", now)
	bad["message"] = "secret hf_abc"
	h.events(token, http.StatusBadRequest, bad)
	h.events("", http.StatusUnauthorized, ev("image", "started", "", now))
	if v = h.install(id); v.LastEvent != nil {
		t.Fatal("a rejected batch must store nothing")
	}

	// Failure with phase and cause, visible to the admin.
	h.events(token, http.StatusNoContent, ev("prerequisites", "succeeded", "", now.Add(-time.Minute)), ev("image", "failed", "image_pull_failed", now.Add(-30*time.Second)))
	v = h.install(id)
	if v.State != "failed" || v.FailedPhase != "image" || v.CauseCode != "image_pull_failed" || v.LastEvent == nil || v.LastEvent.Phase != "image" || v.LastEvent.ReceivedLater {
		t.Fatalf("failure: %+v", v)
	}

	// Recovery: the phase succeeds on rerun.
	h.events(token, http.StatusNoContent, ev("image", "succeeded", "", now))
	if v = h.install(id); v.State != "installing" {
		t.Fatalf("after recovery: %+v", v)
	}

	// Queued offline: original timestamp kept, marked as received later.
	h.events(token, http.StatusNoContent, ev("service", "started", "", now.Add(-3*time.Hour)))
	v = h.install(id)
	var late bool
	for _, e := range v.Events {
		if e.Phase == "service" && e.ReceivedLater {
			late = true
		}
	}
	if !late {
		t.Fatalf("offline event not marked: %+v", v.Events)
	}

	// First pulse: connected. Voice is independent: not ready without a synthetic inference.
	h.do(h.anon, http.MethodPost, h.ts.URL+"/api/worker/pulse", []byte(`{"capacity":1,"supported_steps":[]}`), "", token, false, http.StatusOK)
	h.events(token, http.StatusNoContent, ev("voice-models", "succeeded", "", now))
	if v = h.install(id); v.State != "connected" || v.Voice.State != "pending" {
		t.Fatalf("connected: %+v", v)
	}

	// Voice refused by Hugging Face: the Worker stays connected, voice unavailable with its cause.
	h.events(token, http.StatusNoContent, ev("voice-models", "failed", "hf_access_denied", now))
	if v = h.install(id); v.State != "connected" || v.Voice.State != "unavailable" || v.Voice.CauseCode != "hf_access_denied" {
		t.Fatalf("voice refused: %+v", v)
	}

	// Voice ready only after a successful synthetic inference.
	h.events(token, http.StatusNoContent, ev("voice-models", "succeeded", "", now), ev("voice-test", "succeeded", "", now.Add(time.Second)))
	if v = h.install(id); v.State != "voice_ready" || v.Voice.State != "ready" {
		t.Fatalf("voice ready: %+v", v)
	}

	// Stored rows only hold the closed vocabulary.
	var dump string
	if err := pool.QueryRow(context.Background(), `SELECT string_agg(phase||status||cause_code,',') FROM worker_install_events`).Scan(&dump); err != nil || strings.Contains(dump, token) {
		t.Fatalf("event rows: %v %s", err, dump)
	}
}

func testWorkerInstallNeverConnectedAndSilence(t *testing.T, pool *pgxpool.Pool) {
	h := newInstallHarness(t, pool, true)
	id, token, _, _ := h.register("")
	now := time.Now()

	// Last event stays visible while the absence of news is reported separately.
	h.events(token, http.StatusNoContent, ev("image", "started", "", now))
	if v := h.install(id); v.NoNews || v.LastEvent == nil {
		t.Fatalf("recent: %+v", v)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE worker_install_events SET received_at=now()-interval '2 hours', occurred_at=now()-interval '2 hours'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE workers SET created_at=now()-interval '3 hours', last_seen_at=now()-interval '2 hours'`); err != nil {
		t.Fatal(err)
	}
	if v := h.install(id); v.State != "installing" || !v.NoNews || v.LastEvent == nil || v.LastEvent.Phase != "image" {
		t.Fatalf("silence: %+v", v)
	}

	// No pulse after 24 hours: never connected, last event still there.
	if _, err := pool.Exec(context.Background(), `UPDATE workers SET created_at=now()-interval '25 hours'`); err != nil {
		t.Fatal(err)
	}
	if v := h.install(id); v.State != "never_connected" || v.LastEvent == nil {
		t.Fatalf("never connected: %+v", v)
	}

	// Revoked Worker: events are refused.
	h.adminCall(http.MethodDelete, "/api/admin/workers/"+id, nil, http.StatusNoContent)
	h.events(token, http.StatusUnauthorized, ev("image", "started", "", now))
}
