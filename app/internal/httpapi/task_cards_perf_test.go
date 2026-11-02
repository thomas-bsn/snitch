//go:build integration

package httpapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
)

// queryCounter counts the SQL statements the server sends while it answers.
type queryCounter struct {
	mu    sync.Mutex
	total int
	kinds map[string]int
}

func (q *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.total++
	kind := strings.Join(strings.Fields(data.SQL), " ")
	if len(kind) > 70 {
		kind = kind[:70]
	}
	q.kinds[kind]++
	return ctx
}

func (q *queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (q *queryCounter) reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.total, q.kinds = 0, map[string]int{}
}

// testTaskCardsPerformance measures GET /api/admin/task-cards on thousands of
// synthetic clips written straight into the database. It is a measurement, not
// a regression test: it only runs with SNITCH_PERF=1 and logs its figures. With
// SNITCH_PERF_DUMP=<file> it also writes the response of the 3000 active clips,
// which the browser measurement of the queue page replays.
func testTaskCardsPerformance(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if os.Getenv("SNITCH_PERF") != "1" {
		t.Skip("measurement only: set SNITCH_PERF=1")
	}
	ctx := context.Background()
	counter := &queryCounter{kinds: map[string]int{}}
	config2 := pool.Config().Copy()
	config2.ConnConfig.Tracer = counter
	traced, err := pgxpool.NewWithConfig(ctx, config2)
	if err != nil {
		t.Fatal(err)
	}
	defer traced.Close()
	server := New(traced, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), UploadRoot: t.TempDir()}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewUnstartedServer(server.Handler())
	server.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()

	var adminID int64
	if err = pool.QueryRow(ctx, `INSERT INTO users(username,role,approved) VALUES('perf-admin','admin',true) RETURNING id`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	admin := &http.Client{Jar: jar}
	recorder := httptest.NewRecorder()
	if err = server.issueSession(recorder, httptest.NewRequest(http.MethodGet, ts.URL, nil), adminID); err != nil {
		t.Fatal(err)
	}
	base, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	jar.SetCookies(base.URL, recorder.Result().Cookies())
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, e := pool.Exec(ctx, sql, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec(`INSERT INTO workers(id,name,token_hash,max_concurrency,reported_concurrency,last_seen_at,supported_steps) VALUES('wk_syntheticworker0000001','Synthetic worker',decode(repeat('ab',32),'hex'),2,2,now(),ARRAY['transcription.*'])`)
	oneAudio := `[{"index":1,"type":"audio","title":"","language":"fra","channels":2,"channel_layout":"stereo"}]`
	clips := func(prefix string, n int) {
		exec(`INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at,tracks) SELECT $1,$2||'-'||g||'.mp4',12,now(),now(),$3::jsonb FROM generate_series(1,$4) g`, adminID, prefix, oneAudio, n)
	}
	// jobs adds one task of the given state to the clips of a prefix, between two ranks.
	jobs := func(prefix, step, status string, from, to int) {
		lease := ""
		if status == "leased" {
			lease = ",lease_owner,lease_expires_at,lease_token_hash"
		}
		values := `'` + status + `'`
		if status == "leased" {
			values += `,'wk_syntheticworker0000001',now()+interval '5 minutes',decode(repeat('ab',32),'hex')`
		}
		exec(`INSERT INTO jobs(clip_id,step,input_version,status`+lease+`) SELECT id,$2,'perf-v1',`+values+` FROM (SELECT id,row_number() OVER (ORDER BY id) AS n FROM clips WHERE original_name LIKE $1||'-%') c WHERE n BETWEEN $3 AND $4`, prefix, step, from, to)
	}
	clear := func() {
		exec(`DELETE FROM job_events; DELETE FROM jobs; DELETE FROM clips`)
	}
	measure := func(label string, total int) {
		t.Helper()
		var durations []time.Duration
		var size, gzipped, queries int
		var cards, completed int
		for i := 0; i < 5; i++ {
			counter.reset()
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/admin/task-cards", nil)
			req.Header.Set("Origin", ts.URL)
			started := time.Now()
			resp, e := admin.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			durations = append(durations, time.Since(started))
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s: status %d: %s", label, resp.StatusCode, data)
			}
			size = len(data)
			var packed bytes.Buffer
			zw := gzip.NewWriter(&packed)
			_, _ = zw.Write(data)
			_ = zw.Close()
			gzipped = packed.Len()
			if dump := os.Getenv("SNITCH_PERF_DUMP"); dump != "" && strings.HasPrefix(label, "B") && i == 0 {
				if e = os.WriteFile(dump, data, 0o600); e != nil {
					t.Fatal(e)
				}
			}
			counter.mu.Lock()
			queries = counter.total
			counter.mu.Unlock()
			if i == 0 {
				cards = strings.Count(string(data), `"clip_id"`)
				completed = cards
			}
		}
		sort.Slice(durations, func(a, b int) bool { return durations[a] < durations[b] })
		counter.mu.Lock()
		type kv struct {
			sql string
			n   int
		}
		var top []kv
		for k, v := range counter.kinds {
			top = append(top, kv{k, v})
		}
		counter.mu.Unlock()
		sort.Slice(top, func(a, b int) bool { return top[a].n > top[b].n })
		t.Logf("PERF %s: clips=%d clip_id_occurrences=%d median=%v min=%v max=%v sql_queries=%d bytes=%d gzip_bytes=%d", label, total, completed, durations[len(durations)/2].Round(time.Millisecond), durations[0].Round(time.Millisecond), durations[len(durations)-1].Round(time.Millisecond), queries, size, gzipped)
		for i, entry := range top {
			if i == 5 {
				break
			}
			t.Logf("PERF   %dx %s", entry.n, entry.sql)
		}
	}

	// A library of transcribed clips nobody analysed further: none is on the
	// board and none is completed, yet every one is looked at.
	clips("transcribed", 3000)
	jobs("transcribed", "transcription.audio.1", "done", 1, 3000)
	measure("A transcribed only, nothing active", 3000)

	// A large import in flight: every state of the board, 3000 active clips.
	clear()
	clips("active", 3000)
	jobs("active", "transcription.audio.1", "pending", 1, 1500)
	jobs("active", "transcription.audio.1", "leased", 1501, 1800)
	jobs("active", "transcription.audio.1", "done", 1801, 2400)
	jobs("active", "voice.diarization.1", "pending", 1801, 2400)
	jobs("active", "transcription.audio.1", "dead", 2401, 2700)
	exec(`INSERT INTO job_events(job_id,event,cycle,actor,error_code) SELECT id,'marked_dead',0,'worker','invalid_input' FROM jobs WHERE status='dead'`)
	jobs("active", "transcription.audio.1", "cancelled", 2701, 3000)
	measure("B 3000 active clips", 3000)

	// The same import on top of the library above: 6000 clips with tasks.
	clips("transcribed", 3000)
	jobs("transcribed", "transcription.audio.1", "done", 1, 3000)
	measure("C 3000 active + 3000 transcribed only", 6000)
	clear()
}
