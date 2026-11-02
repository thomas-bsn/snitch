//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
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

// testTaskCardsIndexing checks the Search column against the real semantic
// indexing: its state comes from the index, it is an applicable step only when
// the indexing is active, and it keeps a clip on the board until it is done.
func testTaskCardsIndexing(t *testing.T, pool *pgxpool.Pool) {
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
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/setup", bytes.NewReader([]byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`)))
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("Content-Type", "application/json")
	if resp, e := admin.Do(req); e != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup administrator: %v", e)
	}
	board := func() taskBoardView {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/admin/task-cards", nil)
		req.Header.Set("Origin", ts.URL)
		resp, e := admin.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		var b taskBoardView
		if e = json.NewDecoder(resp.Body).Decode(&b); e != nil {
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
	stepState := func(c *taskCardView, key string) string {
		for _, s := range c.Steps {
			if s.Key == key {
				return s.State
			}
		}
		return "absent"
	}
	var ownerID, personID, profileID int64
	if err = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO persons(display_name) VALUES('Synthetic Microphone Owner') RETURNING id`).Scan(&personID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO embedding_profiles(model,dimensions,state) VALUES('synthetic-index',2,'active') RETURNING id`).Scan(&profileID); err != nil {
		t.Fatal(err)
	}
	// Each clip is a microphone track whose transcription is done: the index is
	// the only step left. An empty taskState means the index holds no row yet.
	clip := func(name, taskState string) int64 {
		t.Helper()
		var id, jobID int64
		if e := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at,tracks) VALUES($1,$2,12,now(),now(),'[{"index":1,"type":"audio","title":"","language":"fra","channels":2,"channel_layout":"stereo"}]'::jsonb) RETURNING id`, ownerID, name).Scan(&id); e != nil {
			t.Fatal(e)
		}
		if _, e := pool.Exec(ctx, `INSERT INTO clip_microphone_overrides(clip_id,audio_stream_index,person_id) VALUES($1,1,$2)`, id, personID); e != nil {
			t.Fatal(e)
		}
		if e := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'transcription.audio.1','synthetic-v1','done') RETURNING id`, id).Scan(&jobID); e != nil {
			t.Fatal(e)
		}
		if taskState != "" {
			if _, e := pool.Exec(ctx, `INSERT INTO embedding_tasks(profile_id,transcription_job_id,state,lease_expires_at) VALUES($1,$2,$3,CASE WHEN $3='leased' THEN now()+interval '2 minutes' END)`, profileID, jobID, taskState); e != nil {
				t.Fatal(e)
			}
		}
		return id
	}
	noRow := clip("index-no-row.mp4", "")
	pending := clip("index-pending.mp4", "pending")
	running := clip("index-running.mp4", "leased")
	failed := clip("index-failed.mp4", "failed")
	finished := clip("index-done.mp4", "done")

	// Indexing disabled (no embeddings URL): the step is not applicable and the
	// clips are finished, whatever the index holds.
	b := board()
	for _, id := range []int64{noRow, pending, running, failed, finished} {
		if find(b.Cards, id) != nil {
			t.Fatalf("clip %d must not be on the board while indexing is disabled: %+v", id, b.Cards)
		}
	}
	if len(b.Completed) != 3 {
		t.Fatalf("completed column must hold three clips: %+v", b.Completed)
	}
	if c := find(b.Completed, finished); c != nil && stepState(c, "indexing") != "not_applicable" {
		t.Fatalf("disabled indexing must be not applicable: %+v", c)
	}
	// The URL is set but no profile is active: still disabled.
	if err = server.Store.Set(ctx, "embeddings_url", "http://synthetic-embeddings.invalid"); err != nil {
		t.Fatal(err)
	}
	b = board()
	if find(b.Cards, pending) != nil || len(b.Completed) != 3 {
		t.Fatalf("no active profile means disabled indexing: %+v", b)
	}

	// Active indexing: the real index state places the card in Search.
	if _, err = pool.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=$1 WHERE id=1`, profileID); err != nil {
		t.Fatal(err)
	}
	b = board()
	for name, want := range map[string]struct {
		id    int64
		state string
	}{"no row": {noRow, "queued"}, "pending": {pending, "queued"}, "leased": {running, "running"}, "failed": {failed, "failed"}} {
		card := find(b.Cards, want.id)
		if card == nil || card.Column != "search" || card.State != want.state || card.CurrentStep != "indexing" || stepState(card, "indexing") != want.state {
			t.Fatalf("%s indexing card: %+v", name, card)
		}
	}
	if find(b.Cards, finished) != nil {
		t.Fatalf("a clip whose indexing is done must not be on the board: %+v", b.Cards)
	}
	done := find(b.Completed, finished)
	if done == nil || stepState(done, "indexing") != "done" {
		t.Fatalf("indexed clip should be completed: %+v", b.Completed)
	}
	if _, err = pool.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=NULL,candidate_profile_id=NULL WHERE id=1; DELETE FROM embedding_tasks; DELETE FROM embedding_profiles`); err != nil {
		t.Fatal(err)
	}
}
