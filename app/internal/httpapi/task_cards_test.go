//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

type taskCardView struct {
	ClipID       int64  `json:"clip_id"`
	OriginalName string `json:"original_name"`
	Column       string `json:"column"`
	State        string `json:"state"`
	CurrentStep  string `json:"current_step"`
	CompletedAt  string `json:"completed_at"`
	LastError    string `json:"last_error"`
	FailureCode  string `json:"failure_code"`
	Action       string `json:"recommended_action"`
	UnknownStep  *struct {
		Step      string `json:"step"`
		State     string `json:"state"`
		LastError string `json:"last_error"`
	} `json:"unknown_step"`
	Steps []struct {
		Key   string `json:"key"`
		State string `json:"state"`
	} `json:"steps"`
}

type taskBoardView struct {
	Cards     []taskCardView `json:"cards"`
	Completed []taskCardView `json:"completed"`
}

func testTaskCards(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	logs := &syncBuffer{}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), UploadRoot: t.TempDir()}, slog.New(slog.NewJSONHandler(logs, nil)))
	ts := httptest.NewUnstartedServer(server.Handler())
	server.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	admin := &http.Client{Jar: jar}
	get := func(c *http.Client, want int) []byte {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/admin/task-cards", nil)
		req.Header.Set("Origin", ts.URL)
		resp, e := c.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("GET task-cards: got %d want %d: %s", resp.StatusCode, want, data)
		}
		return data
	}
	board := func() taskBoardView {
		t.Helper()
		var b taskBoardView
		if e := json.Unmarshal(get(admin, http.StatusOK), &b); e != nil {
			t.Fatal(e)
		}
		return b
	}
	find := func(cards []taskCardView, id int64) *taskCardView {
		for i := range cards {
			if cards[i].ClipID == id {
				return &cards[i]
			}
		}
		return nil
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/setup", bytes.NewReader([]byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`)))
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("Content-Type", "application/json")
	if resp, e := admin.Do(req); e != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup administrator: %v", e)
	}
	var ownerID int64
	if err = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	oneAudio := `[{"index":1,"type":"audio","title":"","language":"fra","channels":2,"channel_layout":"stereo"}]`
	clip := func(name string) int64 {
		t.Helper()
		var id int64
		if e := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at,tracks) VALUES($1,$2,12,now(),now(),$3::jsonb) RETURNING id`, ownerID, name, oneAudio).Scan(&id); e != nil {
			t.Fatal(e)
		}
		return id
	}
	job := func(clipID int64, step, status string) {
		t.Helper()
		lease := ""
		if status == "leased" {
			lease = ",lease_owner='wk_syntheticworker0000001',lease_expires_at=now()+interval '5 minutes',lease_token_hash=decode(repeat('ab',32),'hex')"
		}
		if _, e := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,$2,'synthetic-v1','pending')`, clipID, step); e != nil {
			t.Fatal(e)
		}
		if _, e := pool.Exec(ctx, `UPDATE jobs SET status=$3`+lease+` WHERE clip_id=$1 AND step=$2`, clipID, step, status); e != nil {
			t.Fatal(e)
		}
	}
	if _, err = pool.Exec(ctx, `INSERT INTO workers(id,name,token_hash,max_concurrency,reported_concurrency,last_seen_at,supported_steps) VALUES('wk_syntheticworker0000001','Synthetic worker',decode(repeat('ab',32),'hex'),2,2,now(),ARRAY['transcription.*'])`); err != nil {
		t.Fatal(err)
	}
	var personID int64
	if err = pool.QueryRow(ctx, `INSERT INTO persons(display_name) VALUES('Synthetic Microphone Owner') RETURNING id`).Scan(&personID); err != nil {
		t.Fatal(err)
	}
	microphone := func(clipID int64) {
		t.Helper()
		if _, e := pool.Exec(ctx, `INSERT INTO clip_microphone_overrides(clip_id,audio_stream_index,person_id) VALUES($1,1,$2)`, clipID, personID); e != nil {
			t.Fatal(e)
		}
	}

	// Access: anonymous 401, ordinary user 403.
	get(http.DefaultClient, http.StatusUnauthorized)
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
	get(member, http.StatusForbidden)

	// A clip whose only task is a finished transcription is neither on the board
	// nor completed: its voice step is applicable but nobody scheduled it.
	transcribedOnly := clip("transcribed-only.mp4")
	job(transcribedOnly, "transcription.audio.1", "done")

	// A clip with a microphone track finishes without any diarization, and a
	// pending diarization trial changes neither its column nor its completion.
	micro := clip("microphone-track.mp4")
	microphone(micro)
	job(micro, "transcription.audio.1", "done")
	job(micro, "voicetrial.diarization.1", "pending")

	// A diarization trial alone never puts a clip on the board.
	trialOnly := clip("trial-only.mp4")
	job(trialOnly, "transcription.audio.1", "done")
	job(trialOnly, "voicetrial.diarization.1", "pending")

	// A clip whose transcription runs sits in the first column; one whose
	// diarization runs sits in distinct voices with the transcription done.
	transcribing := clip("transcribing.mp4")
	job(transcribing, "transcription.audio.1", "leased")
	diarizing := clip("diarizing.mp4")
	job(diarizing, "transcription.audio.1", "done")
	job(diarizing, "voice.diarization.1", "leased")
	// A cancelled transcription stays visible and never counts as finished.
	cancelled := clip("cancelled.mp4")
	job(cancelled, "transcription.audio.1", "cancelled")

	b := board()
	for name, id := range map[string]int64{"transcription only": transcribedOnly, "microphone": micro, "trial only": trialOnly} {
		if find(b.Cards, id) != nil {
			t.Fatalf("%s clip must not be on the board: %+v", name, b.Cards)
		}
	}
	if find(b.Completed, transcribedOnly) != nil || find(b.Completed, trialOnly) != nil {
		t.Fatalf("a clip with an applicable step left must not be completed: %+v", b.Completed)
	}
	done := find(b.Completed, micro)
	if done == nil || done.Column != "completed" || done.State != "done" || done.CompletedAt == "" || done.OriginalName != "microphone-track.mp4" {
		t.Fatalf("microphone clip should be completed without diarization: %+v", b.Completed)
	}
	if card := find(b.Cards, transcribing); card == nil || card.Column != "transcription" || card.State != "running" {
		t.Fatalf("transcribing card: %+v", card)
	}
	if card := find(b.Cards, diarizing); card == nil || card.Column != "voices" || card.State != "running" || card.CurrentStep != "diarization" {
		t.Fatalf("diarizing card: %+v", card)
	}
	if card := find(b.Cards, cancelled); card == nil || card.Column != "transcription" || card.State != "cancelled" {
		t.Fatalf("cancelled card: %+v", card)
	}

	// More than twenty active clips are all returned, and the completed column
	// never exceeds three distinct clips.
	for i := 0; i < 25; i++ {
		job(clip(fmt.Sprintf("active-%02d.mp4", i)), "transcription.audio.1", "pending")
	}
	for i := 0; i < 4; i++ {
		id := clip(fmt.Sprintf("finished-%d.mp4", i))
		microphone(id)
		job(id, "transcription.audio.1", "done")
	}
	b = board()
	queued := 0
	for _, card := range b.Cards {
		if card.Column == "transcription" && card.State == "queued" {
			queued++
		}
	}
	if queued != 25 {
		t.Fatalf("all active clips must be returned without pagination: %d queued of %d cards", queued, len(b.Cards))
	}
	if len(b.Completed) != 3 {
		t.Fatalf("completed column must hold exactly three clips: %d", len(b.Completed))
	}

	// A pending task no recently seen Worker can run is waiting for a Worker,
	// with its remedy; one a capable Worker can run stays queued.
	voiceWait := clip("voice-waits-for-worker.mp4")
	job(voiceWait, "transcription.audio.1", "done")
	job(voiceWait, "voice.diarization.1", "pending")
	b = board()
	if card := find(b.Cards, voiceWait); card == nil || card.State != "waiting_worker" || card.Column != "voices" || card.Action != "start_worker" {
		t.Fatalf("voice card without a capable Worker: %+v", card)
	}
	// A pending task whose retry is deferred cannot be claimed yet: it is not
	// waiting for a Worker, it stays queued.
	deferred := clip("voice-deferred.mp4")
	job(deferred, "transcription.audio.1", "done")
	job(deferred, "voice.diarization.1", "pending")
	if _, err = pool.Exec(ctx, `UPDATE jobs SET next_attempt_at=now()+interval '1 hour' WHERE clip_id=$1 AND step='voice.diarization.1'`, deferred); err != nil {
		t.Fatal(err)
	}
	b = board()
	if card := find(b.Cards, deferred); card == nil || card.State != "queued" || card.Action != "" {
		t.Fatalf("a deferred pending task must stay queued: %+v", card)
	}
	if _, err = pool.Exec(ctx, `UPDATE workers SET last_seen_at=now()-interval '10 minutes'`); err != nil {
		t.Fatal(err)
	}
	b = board()
	if card := find(b.Cards, transcribing); card == nil || card.State != "running" {
		t.Fatalf("a leased task never waits for a Worker: %+v", card)
	}
	var stale int64
	for _, card := range b.Cards {
		if card.State == "waiting_worker" {
			stale++
		}
	}
	if stale < 26 {
		t.Fatalf("with no Worker seen recently every pending task must wait for a Worker: %d", stale)
	}
	if _, err = pool.Exec(ctx, `UPDATE workers SET last_seen_at=now()`); err != nil {
		t.Fatal(err)
	}

	event := func(clipID int64, step, name, code string) {
		t.Helper()
		if _, e := pool.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,actor,error_code) SELECT id,$3,0,'worker',$4 FROM jobs WHERE clip_id=$1 AND step=$2`, clipID, step, name, code); e != nil {
			t.Fatal(e)
		}
	}
	// A failure shows the latest persisted cause with its advised action.
	failing := clip("failed.mp4")
	job(failing, "transcription.audio.1", "failed")
	event(failing, "transcription.audio.1", "retry_scheduled", "worker_failed")
	event(failing, "transcription.audio.1", "marked_dead", "whisper_model_missing")
	// A dead task stays in its column, never completed, with its cause.
	dead := clip("dead.mp4")
	job(dead, "transcription.audio.1", "dead")
	event(dead, "transcription.audio.1", "marked_dead", "max_attempts_exhausted")
	// A cancelled task advises to schedule the clip again.
	job(cancelled, "transcription.audio.1", "cancelled")
	event(cancelled, "transcription.audio.1", "cancelled", "cancelled_by_admin")
	// An unknown step stays attached to the card with an alert and its error.
	mystery := clip("mystery.mp4")
	job(mystery, "transcription.audio.1", "done")
	job(mystery, "mystery.step.1", "failed")
	if _, err = pool.Exec(ctx, `UPDATE jobs SET last_error='synthetic mystery error' WHERE clip_id=$1 AND step='mystery.step.1'`, mystery); err != nil {
		t.Fatal(err)
	}
	b = board()
	if card := find(b.Cards, failing); card == nil || card.State != "failed" || card.FailureCode != "whisper_model_missing" || card.Action != "install_transcription_model" {
		t.Fatalf("failed card: %+v", card)
	}
	if card := find(b.Cards, dead); card == nil || card.State != "dead" || card.Column != "transcription" || card.FailureCode != "max_attempts_exhausted" || card.Action != "relaunch_job" {
		t.Fatalf("dead card: %+v", card)
	}
	if find(b.Completed, dead) != nil || find(b.Completed, cancelled) != nil {
		t.Fatal("dead and cancelled clips must never be completed")
	}
	if card := find(b.Cards, cancelled); card == nil || card.State != "cancelled" || card.Action != "reschedule" {
		t.Fatalf("cancelled card: %+v", card)
	}
	if card := find(b.Cards, mystery); card == nil || card.UnknownStep == nil || card.UnknownStep.Step != "mystery.step.1" || card.UnknownStep.LastError != "synthetic mystery error" {
		t.Fatalf("unknown step card: %+v", card)
	}

	// A pending track selection keeps an applicable step: a clip with two audio
	// tracks and no choice, transcribed and diarized, still owes its voice
	// fingerprints and must not be completed.
	twoAudio := `[{"index":1,"type":"audio","title":"","language":"fra","channels":2,"channel_layout":"stereo"},{"index":2,"type":"audio","title":"","language":"fra","channels":2,"channel_layout":"stereo"}]`
	pendingSelection := clip("selection-pending.mp4")
	if _, err = pool.Exec(ctx, `UPDATE clips SET tracks=$2::jsonb WHERE id=$1`, pendingSelection, twoAudio); err != nil {
		t.Fatal(err)
	}
	job(pendingSelection, "transcription.audio.1", "done")
	job(pendingSelection, "voice.diarization.1", "done")
	if _, err = pool.Exec(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) SELECT $1,1,id,'SPEAKER_00',0,1,1 FROM jobs WHERE clip_id=$1 AND step='voice.diarization.1'`, pendingSelection); err != nil {
		t.Fatal(err)
	}
	b = board()
	if find(b.Completed, pendingSelection) != nil {
		t.Fatalf("a clip with a pending track selection and an applicable step left must not be completed: %+v", b.Completed)
	}

	// A clip whose derivation fails stays on the board with an explicit alert,
	// never completed and never dropped; the failure is logged with the clip id
	// and without the raw error text; the other cards are unaffected.
	broken, brokenFinished := clip("derivation-fails.mp4"), clip("derivation-fails-finished.mp4")
	job(broken, "transcription.audio.1", "pending")
	microphone(brokenFinished)
	job(brokenFinished, "transcription.audio.1", "done")
	server.cardFault = func(id int64) error {
		if id == broken || id == brokenFinished {
			return errors.New("synthetic secret derivation text")
		}
		return nil
	}
	defer func() { server.cardFault = nil }()
	b = board()
	for _, id := range []int64{broken, brokenFinished} {
		card := find(b.Cards, id)
		if card == nil || card.State != "failed" || card.UnknownStep == nil || card.UnknownStep.Step != "derivation_failed" || card.OriginalName == "" {
			t.Fatalf("a clip whose derivation fails must stay visible with an alert: %+v", card)
		}
		if find(b.Completed, id) != nil {
			t.Fatalf("a clip whose derivation fails must never be completed: %d", id)
		}
		if !strings.Contains(logs.String(), fmt.Sprintf(`"clip_id":%d`, id)) {
			t.Fatalf("the derivation failure must be logged with the clip id: %s", logs.String())
		}
	}
	if strings.Contains(logs.String(), "synthetic secret derivation text") {
		t.Fatalf("raw error text must not be logged: %s", logs.String())
	}
	if find(b.Cards, transcribing) == nil {
		t.Fatal("the other cards must stay on the board")
	}
}

// syncBuffer is a log sink safe for the server goroutines and the test.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
