//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

type historyEventView struct {
	Event             string `json:"event"`
	ErrorCode         string `json:"error_code"`
	Model             string `json:"model"`
	ActorID           string `json:"actor_id"`
	RecommendedAction string `json:"recommended_action"`
}

type historyStepView struct {
	JobID    int64              `json:"job_id"`
	Step     string             `json:"step"`
	Status   string             `json:"status"`
	Attempts int                `json:"attempts"`
	Events   []historyEventView `json:"events"`
}

type clipHistoryView struct {
	Items []struct {
		ClipID       int64             `json:"clip_id"`
		OriginalName string            `json:"original_name"`
		Steps        []historyStepView `json:"steps"`
	} `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type completedCardView struct {
	ClipID         int64 `json:"clip_id"`
	VoicesToReview bool  `json:"voices_to_review"`
}

func testClipHistoryAndCompleted(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{`DELETE FROM clips`, `DELETE FROM jobs`} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), UploadRoot: t.TempDir()}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewUnstartedServer(server.Handler())
	server.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	admin := &http.Client{Jar: jar}
	get := func(c *http.Client, path string, want int, into any) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		req.Header.Set("Origin", ts.URL)
		resp, e := c.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("GET %s: got %d want %d: %s", path, resp.StatusCode, want, data)
		}
		if into != nil {
			if e = json.Unmarshal(data, into); e != nil {
				t.Fatal(e)
			}
		}
	}
	board := func() (cards, completed []completedCardView) {
		t.Helper()
		var b struct {
			Cards     []completedCardView `json:"cards"`
			Completed []completedCardView `json:"completed"`
		}
		get(admin, "/api/admin/task-cards", http.StatusOK, &b)
		return b.Cards, b.Completed
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/setup", bytes.NewReader([]byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`)))
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("Content-Type", "application/json")
	if resp, e := admin.Do(req); e != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup administrator: %v", e)
	}
	var ownerID, personID int64
	if err = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO persons(display_name) VALUES('Synthetic Owner') RETURNING id`).Scan(&personID); err != nil {
		t.Fatal(err)
	}
	oneAudio := `[{"index":1,"type":"audio","title":"","language":"fra","channels":2,"channel_layout":"stereo"}]`
	// finished builds a clip on a microphone track whose only task is a finished
	// transcription: it is complete, minutesAgo minutes ago.
	finished := func(name string, minutesAgo int) int64 {
		t.Helper()
		var id int64
		if e := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at,tracks) VALUES($1,$2,12,now(),now(),$3::jsonb) RETURNING id`, ownerID, name, oneAudio).Scan(&id); e != nil {
			t.Fatal(e)
		}
		if _, e := pool.Exec(ctx, `INSERT INTO clip_microphone_overrides(clip_id,audio_stream_index,person_id) VALUES($1,1,$2)`, id, personID); e != nil {
			t.Fatal(e)
		}
		if _, e := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,updated_at) VALUES($1,'transcription.audio.1','v1','done',now()-make_interval(mins=>$2))`, id, minutesAgo); e != nil {
			t.Fatal(e)
		}
		return id
	}
	oldest, middle, newer, newest := finished("history-a.mp4", 40), finished("history-b.mp4", 30), finished("history-c.mp4", 20), finished("history-d.mp4", 10)
	// A failed and a pending clip, both updated more recently than any finished
	// clip, stay on the board and never take a place in Completed.
	failedClip := finished("history-failed.mp4", 5)
	if _, err = pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,last_error,updated_at) VALUES($1,'voice.diarization.1','v1','failed','provider_unavailable',now())`, failedClip); err != nil {
		t.Fatal(err)
	}
	pendingClip := finished("history-pending.mp4", 4)
	if _, err = pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.1','v1','pending')`, pendingClip); err != nil {
		t.Fatal(err)
	}
	cards, completed := board()
	if len(completed) != 3 || completed[0].ClipID != newest || completed[1].ClipID != newer || completed[2].ClipID != middle {
		t.Fatalf("completed must be the three most recently finished distinct clips, newest first: %+v", completed)
	}
	onBoard := map[int64]bool{}
	for _, c := range cards {
		onBoard[c.ClipID] = true
	}
	if !onBoard[failedClip] || !onBoard[pendingClip] {
		t.Fatalf("failed and pending clips must stay on the board: %+v", cards)
	}

	// Reanalysis: a new transcription task takes the clip out of Completed.
	if _, err = pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'transcription.audio.1','v2','pending')`, newest); err != nil {
		t.Fatal(err)
	}
	cards, completed = board()
	if len(completed) != 3 || completed[0].ClipID != newer || completed[2].ClipID != oldest {
		t.Fatalf("a reanalysed clip must leave completed: %+v", completed)
	}
	back := false
	for _, c := range cards {
		back = back || c.ClipID == newest
	}
	if !back {
		t.Fatalf("a reanalysed clip must return to the board: %+v", cards)
	}

	// Voices to review: a completed clip whose diarized voice is neither
	// validated nor attributed carries the flag; validating the voice clears it.
	var reviewed int64
	if err = pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at,tracks) VALUES($1,'history-voices.mp4',12,now(),now(),$2::jsonb) RETURNING id`, ownerID, oneAudio).Scan(&reviewed); err != nil {
		t.Fatal(err)
	}
	var diarizationJob int64
	for _, step := range []string{"transcription.audio.1", "voice.diarization.1", "voice.fingerprint.1"} {
		var id int64
		if err = pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,updated_at) VALUES($1,$2,'v1','done',now()+interval '1 minute') RETURNING id`, reviewed, step).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if step == "voice.diarization.1" {
			diarizationJob = id
		}
	}
	var speakerID int64
	if err = pool.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,1,$2,'SPEAKER_00',0,10,2) RETURNING id`, reviewed, diarizationJob).Scan(&speakerID); err != nil {
		t.Fatal(err)
	}
	flag := func(id int64) bool {
		t.Helper()
		_, completed := board()
		if len(completed) == 0 || completed[0].ClipID != id {
			t.Fatalf("clip %d should lead completed: %+v", id, completed)
		}
		return completed[0].VoicesToReview
	}
	if !flag(reviewed) {
		t.Fatal("an unattributed diarized voice must raise voices_to_review")
	}
	if _, err = pool.Exec(ctx, `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,origin) VALUES($1,$2,'human','validated','human')`, speakerID, personID); err != nil {
		t.Fatal(err)
	}
	if flag(reviewed) {
		t.Fatal("a validated voice must not raise voices_to_review")
	}

	// Clip history access: anonymous 401, ordinary user 403.
	get(http.DefaultClient, "/api/admin/clip-history", http.StatusUnauthorized, nil)
	var memberID int64
	if err = pool.QueryRow(ctx, `INSERT INTO users(username,role,approved) VALUES('ordinary','user',true) RETURNING id`).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	memberJar, _ := cookiejar.New(nil)
	member := &http.Client{Jar: memberJar}
	recorder := httptest.NewRecorder()
	if err = server.issueSession(recorder, httptest.NewRequest(http.MethodGet, ts.URL, nil), memberID); err != nil {
		t.Fatal(err)
	}
	base, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	memberJar.SetCookies(base.URL, recorder.Result().Cookies())
	get(member, "/api/admin/clip-history", http.StatusForbidden, nil)

	// Events of each step: model, worker, attempts and failure cause.
	var failedJob int64
	if err = pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step='voice.diarization.1'`, failedClip).Scan(&failedJob); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,attempt,actor,actor_id,error_code,step,model) VALUES($1,'attempt_started',0,1,'worker','wk_syntheticworker0000001','','voice.diarization.1','synthetic-model'),($1,'retry_scheduled',0,1,'worker','wk_syntheticworker0000001','provider_unavailable','voice.diarization.1','synthetic-model')`, failedJob); err != nil {
		t.Fatal(err)
	}
	var h clipHistoryView
	get(admin, "/api/admin/clip-history?q="+url.QueryEscape("history-failed"), http.StatusOK, &h)
	if h.Total != 1 || len(h.Items) != 1 || h.Items[0].ClipID != failedClip {
		t.Fatalf("search by name: %+v", h)
	}
	steps := h.Items[0].Steps
	if len(steps) != 2 {
		t.Fatalf("each step of the clip must appear: %+v", steps)
	}
	var diar *historyStepView
	for i := range steps {
		if steps[i].Step == "voice.diarization.1" {
			diar = &steps[i]
		}
	}
	if diar == nil || diar.Status != "failed" || len(diar.Events) != 2 || diar.Events[1].ErrorCode != "provider_unavailable" || diar.Events[1].Model != "synthetic-model" || diar.Events[1].ActorID != "wk_syntheticworker0000001" || diar.Events[1].RecommendedAction != "check_transcription_service" {
		t.Fatalf("diarization step history: %+v", diar)
	}
	get(admin, "/api/admin/clip-history?q="+fmt.Sprint(newer), http.StatusOK, &h)
	if h.Total != 1 || h.Items[0].ClipID != newer {
		t.Fatalf("search by clip id: %+v", h)
	}
	get(admin, "/api/admin/clip-history?q=no-such-clip-anywhere", http.StatusOK, &h)
	if h.Total != 0 || len(h.Items) != 0 {
		t.Fatalf("empty search: %+v", h)
	}

	// Pagination is by clip: 7 clips, pages of 3.
	get(admin, "/api/admin/clip-history?page_size=3&page=1", http.StatusOK, &h)
	if h.Total != 7 || len(h.Items) != 3 || h.Page != 1 || h.PageSize != 3 {
		t.Fatalf("first page: total %d items %d", h.Total, len(h.Items))
	}
	first := h.Items[0].ClipID
	get(admin, "/api/admin/clip-history?page_size=3&page=3", http.StatusOK, &h)
	if len(h.Items) != 1 || h.Items[0].ClipID == first {
		t.Fatalf("last page: %+v", h.Items)
	}
	get(admin, "/api/admin/clip-history?page=0", http.StatusBadRequest, nil)
	get(admin, "/api/admin/clip-history?page_size=1000", http.StatusBadRequest, nil)
}
