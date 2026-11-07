//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type analysisViewT struct {
	Run *struct {
		Status        string `json:"status"`
		PendingVoices int    `json:"pending_voices"`
		Steps         []struct {
			Step   string `json:"step"`
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"steps"`
	} `json:"run"`
}

// Every user who can open a clip sees its analysis read-only, live too; nothing technical
// leaks, a clip they cannot open reveals nothing, and the controls stay administrators'.
func testClipAnalysisReadOnlyView(t *testing.T, pool *pgxpool.Pool) {
	ctx := t.Context()
	h := newVoiceHarness(t, pool)
	h.useSingleAudioTrack()
	h.api.Events.Poll, h.api.Events.Heartbeat = 50*time.Millisecond, 300*time.Millisecond
	alice := h.newUser("analysis-viewer")
	path := fmt.Sprintf("/api/clips/%d/analysis", h.clipID)
	setVisibility := func(v string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE clips SET visibility=$2 WHERE id=$1`, h.clipID, v); err != nil {
			t.Fatal(err)
		}
	}
	view := func(u *voiceUser) (analysisViewT, string) {
		t.Helper()
		raw := h.userCall(u, http.MethodGet, path, nil, http.StatusOK)
		var out analysisViewT
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		return out, string(raw)
	}

	// A clip the user cannot open is not found, for the read and for the stream.
	setVisibility("private")
	h.userCall(alice, http.MethodGet, path, nil, http.StatusNotFound)
	h.userCall(alice, http.MethodGet, path+"/events", nil, http.StatusNotFound)
	if status, _ := (&http.Client{}).Get(h.ts.URL + path); status == nil || status.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous read: %+v", status)
	}
	setVisibility("shared")
	if out, _ := view(alice); out.Run != nil {
		t.Fatal("a clip never analyzed has no run")
	}

	// With a real job behind the transcription step, the view still carries no technical detail.
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusOK)
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	if _, err := pool.Exec(ctx, `UPDATE jobs SET last_error='open /srv/secret/path: token=hunter2-synthetic' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	out, raw := view(alice)
	if out.Run == nil || out.Run.Status != "active" || len(out.Run.Steps) != 6 {
		t.Fatalf("the member sees the run and its steps: %s", raw)
	}
	for _, leak := range []string{"job_id", "worker_id", "last_error", "error_code", "input_version", "hunter2", "/srv/secret", "started_by", "attempt"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("the read-only view leaks %q: %s", leak, raw)
		}
	}
	// A dead job with a recorded cause stays invisible to the member; the Administrator reads it.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='dead',last_error='whisper_model_missing' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,attempt,actor,error_code,step) SELECT id,'marked_dead',cycle,1,'worker','whisper_model_missing',step FROM jobs WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	_, dead := view(alice)
	for _, leak := range []string{"whisper", "recommended_action", "install_transcription_model", "marked_dead"} {
		if strings.Contains(dead, leak) {
			t.Fatalf("the read-only view of a failed step leaks %q: %s", leak, dead)
		}
	}
	reasons := map[string]string{}
	for _, st := range out.Run.Steps {
		reasons[st.Step] = st.Status + "/" + st.Reason
	}
	if reasons["transcription"] == "" || reasons["indexing"] != "not_applicable/outside_scope" {
		t.Fatalf("each step carries its status and stable reason: %+v", reasons)
	}

	// The partially identified state is the live count of voices to check, as in the queue.
	var job int64
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.2','view-pv','done') RETURNING id`, h.clipID).Scan(&job); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,2,$2,'SPEAKER_00',0,2,1)`, h.clipID, job); err != nil {
		t.Fatal(err)
	}
	if out, _ = view(alice); out.Run.PendingVoices != 1 {
		t.Fatalf("pending voices: %+v", out.Run)
	}

	// The controls remain administrators'.
	h.userCall(alice, http.MethodPost, "/api/admin/clips/"+fmt.Sprint(h.clipID)+"/analysis", map[string]any{}, http.StatusForbidden)
	h.userCall(alice, http.MethodPost, "/api/admin/clips/"+fmt.Sprint(h.clipID)+"/analysis/cancel", nil, http.StatusForbidden)
	h.userCall(alice, http.MethodGet, "/api/admin/clips/"+fmt.Sprint(h.clipID)+"/analysis", nil, http.StatusForbidden)
	h.userCall(alice, http.MethodGet, "/api/admin/clips/"+fmt.Sprint(h.clipID)+"/analysis/events", nil, http.StatusForbidden)

	// Live: the stream carries the same redacted view, and ends with a notice once access is lost.
	req, _ := http.NewRequest(http.MethodGet, h.ts.URL+path+"/events", nil)
	resp, err := alice.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("stream: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	frames := readFrames(resp)
	first := nextFrame(t, frames)
	if first.Event != "snapshot" || strings.Contains(first.Data, "job_id") || strings.Contains(first.Data, "hunter2") || !strings.Contains(first.Data, `"pending_voices":1`) {
		t.Fatalf("the live view is redacted too: %+v", first)
	}
	setVisibility("private")
	h.adminCall(http.MethodPost, h.analysisPath()+"/cancel", nil, http.StatusOK)
	for {
		f, ok := <-frames
		if !ok {
			t.Fatal("the stream ended without telling the viewer")
		}
		if strings.Contains(f.Data, "job_id") {
			t.Fatalf("a snapshot leaked: %+v", f)
		}
		if f.Event == "forbidden" {
			break
		}
		if f.Event == "snapshot" && strings.Contains(f.Data, `"cancelled"`) {
			t.Fatal("a clip no longer readable must not push its new state")
		}
	}
}
