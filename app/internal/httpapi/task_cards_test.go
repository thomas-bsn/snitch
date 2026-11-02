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
	Steps        []struct {
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
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), UploadRoot: t.TempDir()}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
	if _, err = pool.Exec(ctx, `INSERT INTO workers(id,name,token_hash) VALUES('wk_syntheticworker0000001','Synthetic worker',decode(repeat('ab',32),'hex'))`); err != nil {
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
}
