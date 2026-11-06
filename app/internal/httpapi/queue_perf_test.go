//go:build integration

package httpapi

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
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
	spent map[string]time.Duration
}

type queryStart struct{}

func (q *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.total++
	kind := strings.Join(strings.Fields(data.SQL), " ")
	if len(kind) > 60 {
		kind = kind[:60]
	}
	return context.WithValue(context.WithValue(ctx, queryStart{}, time.Now()), q, kind)
}

func (q *queryCounter) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	started, _ := ctx.Value(queryStart{}).(time.Time)
	kind, _ := ctx.Value(q).(string)
	q.mu.Lock()
	defer q.mu.Unlock()
	q.spent[kind] += time.Since(started)
}

func (q *queryCounter) reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.total, q.spent = 0, map[string]time.Duration{}
}

// slowest lists the statements by the time the database spent on them.
func (q *queryCounter) slowest() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	var out []string
	for kind, d := range q.spent {
		out = append(out, d.Round(time.Millisecond).String()+" "+kind)
	}
	sort.Slice(out, func(a, b int) bool {
		return len(out[a]) < len(out[b]) || (len(out[a]) == len(out[b]) && out[a] > out[b])
	})
	return strings.Join(out, " | ")
}

func (q *queryCounter) count() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.total
}

// perfRuns is the number of synthetic active analysis runs of the measurement.
const perfRuns = 3000

// seedQueueRuns writes n active analysis runs straight into the database, with six steps, the
// jobs behind them and the voices to check, in the proportions of a busy library: most clips wait
// for transcription, some run, fail or retry, the others are further along. Never real data.
func seedQueueRuns(t *testing.T, pool *pgxpool.Pool, adminID int64, n int) {
	t.Helper()
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO workers(id,name,token_hash,max_concurrency,reported_concurrency,last_seen_at,supported_steps) VALUES('wk_syntheticworker0000001','Synthetic worker',decode(repeat('ab',32),'hex'),2,2,now(),ARRAY['transcription.*'])`)
	exec(`INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at,tracks) SELECT $1,'synthetic-clip-'||g||'.mp4',12,now(),now(),'[{"index":1,"type":"audio","title":"","language":"fra","channels":2,"channel_layout":"stereo"}]'::jsonb FROM generate_series(1,$2) g`, adminID, n)
	// m is the profile of a clip (rank modulo 10); cur is the rank (1..6) of the step it is at.
	const profile = `SELECT c.id AS clip_id,(row_number() OVER (ORDER BY c.id))%10 AS m,
 CASE (row_number() OVER (ORDER BY c.id))%10 WHEN 6 THEN 4 WHEN 8 THEN 6 WHEN 9 THEN 3 ELSE 2 END AS cur FROM clips c`
	exec(`INSERT INTO clip_analysis_runs(clip_id,scope,status,started_by) SELECT clip_id,'full',CASE WHEN m=7 THEN 'failed' ELSE 'active' END,$1 FROM (`+profile+`) p`, adminID)
	exec(`CREATE TEMP TABLE perf_steps AS SELECT r.id AS run_id,p.clip_id,p.m,p.cur,s.name,s.idx,s.job FROM clip_analysis_runs r JOIN (` + profile + `) p ON p.clip_id=r.clip_id
 CROSS JOIN (VALUES (1,'preparation',''),(2,'transcription','transcription.whisper.1'),(3,'indexing','transcription.index'),(4,'diarization','voice.diarization.2'),(5,'fingerprints','voice.fingerprints'),(6,'matching','voice.matching')) s(idx,name,job)`)
	// A job for each step from transcription to the current one.
	exec(`INSERT INTO jobs(clip_id,step,input_version,status,lease_owner,lease_expires_at,lease_token_hash,last_error,attempts)
 SELECT clip_id,job,'perf-v1',
  CASE WHEN idx<cur THEN 'done' WHEN m=7 THEN 'failed' WHEN m IN (4,5,6) THEN 'leased' ELSE 'pending' END,
  CASE WHEN idx=cur AND m IN (4,5,6) THEN 'wk_syntheticworker0000001' END,
  CASE WHEN idx=cur AND m IN (4,5,6) THEN now()+interval '5 minutes' END,
  CASE WHEN idx=cur AND m IN (4,5,6) THEN decode(repeat('ab',32),'hex') END,
  CASE WHEN idx=cur AND m=7 THEN 'worker_failed' ELSE '' END,
  CASE WHEN idx<cur THEN 1 ELSE 0 END
 FROM perf_steps WHERE idx>=2 AND idx<=cur`)
	exec(`INSERT INTO job_events(job_id,event,cycle,actor,error_code) SELECT j.id,'attempt_started',0,'worker','' FROM jobs j`)
	exec(`INSERT INTO job_events(job_id,event,cycle,actor,error_code) SELECT j.id,'retry_scheduled',0,'worker','worker_failed' FROM jobs j WHERE j.status='failed'`)
	exec(`INSERT INTO clip_analysis_steps(run_id,step,status,reason,job_id,attempt,started_at,finished_at)
 SELECT s.run_id,s.name,
  CASE WHEN s.idx<s.cur THEN 'done' WHEN s.idx>s.cur THEN 'waiting' WHEN s.m=7 THEN 'failed' WHEN s.m IN (4,5,6) THEN 'running' WHEN s.m=9 THEN 'retry' ELSE 'waiting' END,
  CASE WHEN s.idx<s.cur THEN 'completed' WHEN s.idx>s.cur THEN 'awaiting_previous_step' WHEN s.m=7 THEN 'job_failed' WHEN s.m IN (4,5,6) THEN 'running_on_worker' WHEN s.m=9 THEN 'retry_scheduled' ELSE 'awaiting_worker' END,
  j.id,CASE WHEN s.idx<=s.cur THEN 1 ELSE 0 END,CASE WHEN s.idx<=s.cur THEN now() END,CASE WHEN s.idx<s.cur THEN now() END
 FROM perf_steps s LEFT JOIN jobs j ON j.clip_id=s.clip_id AND j.step=s.job AND s.idx>=2`)
	// Voices to check on the clips that finished diarization.
	exec(`INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count)
 SELECT j.clip_id,1,j.id,'SPEAKER_0'||o,o,3,2 FROM jobs j CROSS JOIN generate_series(0,1) o WHERE j.step='voice.diarization.2' AND j.status='done'`)
	// A few finished runs and loose tasks, as on a real queue.
	exec(`INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) SELECT $1,'synthetic-done-'||g||'.mp4',12,now(),now() FROM generate_series(1,5) g`, adminID)
	exec(`INSERT INTO clip_analysis_runs(clip_id,scope,status,finished_at) SELECT id,'full','completed',now() FROM clips WHERE original_name LIKE 'synthetic-done-%'`)
	exec(`INSERT INTO jobs(clip_id,step,input_version,status) SELECT id,'transcription.whisper.1','perf-loose','pending' FROM clips WHERE original_name LIKE 'synthetic-done-%' AND right(original_name,5)<'3.mp4'`)
	exec(`DROP TABLE perf_steps`)
}

func sseFrames(body io.Reader, frames chan<- int) {
	reader := bufio.NewReaderSize(body, 1<<20)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			close(frames)
			return
		}
		if strings.HasPrefix(line, "data: {\"seq\"") {
			frames <- len(line)
		}
	}
}

// testQueuePerformance measures the Task Queue on thousands of synthetic active runs. It is a
// measurement, not a regression test: it runs only with SNITCH_PERF=1 and logs its figures. With
// SNITCH_PERF_DUMP=<file> it also writes the response, which the browser measurement replays.
func testQueuePerformance(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if os.Getenv("SNITCH_PERF") != "1" {
		t.Skip("measurement only: set SNITCH_PERF=1")
	}
	ctx := context.Background()
	counter := &queryCounter{spent: map[string]time.Duration{}}
	cfg := pool.Config().Copy()
	cfg.ConnConfig.Tracer = counter
	traced, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer traced.Close()
	server := New(traced, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), UploadRoot: t.TempDir()}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	server.Events.Poll, server.Events.Heartbeat = 50*time.Millisecond, time.Minute
	ts := httptest.NewUnstartedServer(server.Handler())
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
	session := recorder.Result().Cookies()[0]
	if _, err = pool.Exec(ctx, `UPDATE installation SET initialized=true WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	seedQueueRuns(t, pool, adminID, perfRuns)
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM clip_analysis_runs; DELETE FROM job_events; DELETE FROM jobs; DELETE FROM clips`)
	}()

	var durations []time.Duration
	var size, gzipped, queries, active int
	for i := 0; i < 5; i++ {
		counter.reset()
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/admin/analysis/queue", nil)
		started := time.Now()
		resp, e := admin.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		durations = append(durations, time.Since(started))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d: %s", resp.StatusCode, data)
		}
		size, queries = len(data), counter.count()
		var packed bytes.Buffer
		zw := gzip.NewWriter(&packed)
		_, _ = zw.Write(data)
		_ = zw.Close()
		gzipped = packed.Len()
		active = strings.Count(string(data), `"scope"`)
		if dump := os.Getenv("SNITCH_PERF_DUMP"); dump != "" && i == 0 {
			if e = os.WriteFile(dump, data, 0o600); e != nil {
				t.Fatal(e)
			}
		}
	}
	t.Logf("PERF queue sql time of the last call: %s", counter.slowest())
	sort.Slice(durations, func(a, b int) bool { return durations[a] < durations[b] })
	t.Logf("PERF queue: runs=%d scope_occurrences=%d median=%v min=%v max=%v sql_queries=%d bytes=%d gzip_bytes=%d", perfRuns, active, durations[len(durations)/2].Round(time.Millisecond), durations[0].Round(time.Millisecond), durations[len(durations)-1].Round(time.Millisecond), queries, size, gzipped)

	// The stream, as it was (a whole snapshot at each change) and as the page asks for it
	// (delta=1): the first frame, then the frame that answers a change of one step.
	touch := func(offset int) {
		if _, e := pool.Exec(ctx, `UPDATE clip_analysis_steps SET updated_at=now() WHERE (run_id,step)=(SELECT run_id,step FROM clip_analysis_steps ORDER BY run_id LIMIT 1 OFFSET $1)`, offset); e != nil {
			t.Error(e)
		}
	}
	for _, mode := range []struct{ label, query string }{{"whole snapshot", ""}, {"delta", "?delta=1"}} {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/admin/analysis/queue/events"+mode.query, nil)
		stream, e := admin.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		frames := make(chan int, 64)
		go sseFrames(stream.Body, frames)
		select {
		case first := <-frames:
			t.Logf("PERF sse %s: first frame bytes=%d", mode.label, first)
		case <-time.After(30 * time.Second):
			t.Fatal("no first frame")
		}
		var changeBytes int
		var changeLatency time.Duration
		const changes = 5
		for i := 0; i < changes; i++ {
			started := time.Now()
			touch(i)
			select {
			case f := <-frames:
				changeBytes += f
				changeLatency += time.Since(started)
			case <-time.After(30 * time.Second):
				t.Fatal("no frame after a change")
			}
			time.Sleep(200 * time.Millisecond)
		}
		stream.Body.Close()
		t.Logf("PERF sse %s: change frame avg_bytes=%d avg_latency=%v", mode.label, changeBytes/changes, (changeLatency / changes).Round(time.Millisecond))
	}

	// While the page is measured, a step changes every half second, as on a busy queue.
	done := make(chan struct{})
	defer close(done)
	go func() {
		for i := 10; ; i++ {
			select {
			case <-done:
				return
			case <-time.After(500 * time.Millisecond):
				touch(i)
			}
		}
	}()
	measureQueuePage(t, ts.URL, session)
}

// measureQueuePage opens the Task Queue page of the server in a browser (SNITCH_PERF_PAGE=1) and
// logs what it costs: DOM nodes, memory and long tasks, at load and on an update.
func measureQueuePage(t *testing.T, serverURL string, session *http.Cookie) {
	t.Helper()
	if os.Getenv("SNITCH_PERF_PAGE") != "1" {
		return
	}
	cmd := exec.Command("npx", "playwright", "test", "queue-perf.spec.ts")
	cmd.Dir = "../../../web"
	cmd.Env = append(os.Environ(), "SNITCH_TEST_URL="+serverURL, "SNITCH_PERF_COOKIE="+session.Name+"="+session.Value)
	out, err := cmd.CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "PERF page") {
			t.Log(strings.TrimSpace(line))
		}
	}
	if err != nil {
		t.Fatalf("page measurement: %v\n%s", err, out)
	}
}
