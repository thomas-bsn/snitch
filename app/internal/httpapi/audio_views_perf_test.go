//go:build integration

package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"snitch/migrations"
)

// TestAudioViewsPerformance loads a synthetic library (thousands of clips and segments, no real media) and
// bounds the cost of the readers built on audio_visible_* (#340): per-clip reads, the visible transcript
// CTE, and the semantic-search join. JIT compilation of the jsonb-heavy views once cost ~2 s per call even on
// one clip, so the bounds are generous but fail on that regression.
func TestAudioViewsPerformance(t *testing.T) {
	ctx := context.Background()
	pg, err := postgres.Run(ctx, pgImage, postgres.WithDatabase("receipts"), postgres.WithUsername("receipts"), postgres.WithPassword("synthetic-only"), postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	provider, err := goose.NewProvider(database.DialectPostgres, db, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	var jit string
	if err = pool.QueryRow(ctx, `SHOW jit`).Scan(&jit); err != nil || jit != "on" {
		t.Fatalf("Migration must preserve the default JIT policy (got %q, err %v)", jit, err)
	}
	const clips, segments = 3000, 12
	exec(`INSERT INTO users(username,role,approved) VALUES('perf','admin',true)`)
	exec(`INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) SELECT (SELECT id FROM users LIMIT 1),'c'||g,1,now(),now() FROM generate_series(1,$1) g`, clips)
	exec(`INSERT INTO jobs(clip_id,step,input_version,status,result) SELECT c.id,'transcription.audio.1','v'||c.id,'done',
 jsonb_build_object('audio_stream_index',1,'segments',(SELECT jsonb_agg(jsonb_build_object('start',s,'end',s+1,'text','mot '||s||' clip '||c.id,'words',jsonb_build_array(jsonb_build_object('word','mot','start',s,'end',s+0.5))) ORDER BY s) FROM generate_series(0,$1-1) s)) FROM clips c`, segments)
	exec(`ANALYZE`)
	timed := func(name, q string, budget time.Duration, args ...any) {
		t.Helper()
		start := time.Now()
		rows, err := pool.Query(ctx, q, args...)
		if err != nil {
			t.Fatal(name, err)
		}
		n := 0
		for rows.Next() {
			n++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatal(name, err)
		}
		d := time.Since(start)
		t.Logf("%-28s %6d rows %8.1f ms", name, n, float64(d.Microseconds())/1000)
		if d > budget {
			t.Errorf("%s took %s (budget %s)", name, d, budget)
		}
	}
	var someClip int64
	_ = pool.QueryRow(ctx, `SELECT id FROM clips ORDER BY id LIMIT 1 OFFSET 1500`).Scan(&someClip)
	// Warm once so plan caches and the first-use JIT do not hide behind the first measurement.
	timed("one clip segments", `SELECT seg FROM audio_visible_segments WHERE clip_id=$1`, 500*time.Millisecond, someClip)
	timed("one clip segments (again)", `SELECT seg FROM audio_visible_segments WHERE clip_id=$1`, 500*time.Millisecond, someClip)
	timed("visible transcript of a clip", `WITH `+visibleTranscriptsCTE("j.clip_id=$1")+` SELECT job_id FROM transcript`, 500*time.Millisecond, someClip)
	timed("effective segments of a clip", `WITH `+visibleTranscriptsCTE("j.clip_id=$1")+`, `+effectiveSegmentsCTE()+` SELECT idx FROM eff_segments`, 500*time.Millisecond, someClip)
	timed("job join (semantic search)", `SELECT v.seg_start FROM audio_visible_segments v JOIN jobs j ON j.id=v.job_id WHERE v.job_id=(SELECT id FROM jobs WHERE clip_id=$1 LIMIT 1) AND v.idx=3`, 800*time.Millisecond, someClip)
	timed("all visible segments", `SELECT count(*) FROM audio_visible_segments`, 15*time.Second)
	if testing.Verbose() {
		rows, err := pool.Query(ctx, `EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF) SELECT v.seg_start FROM audio_visible_segments v WHERE v.job_id=(SELECT id FROM jobs WHERE clip_id=$1 LIMIT 1) AND v.idx=3`, someClip)
		if err != nil {
			t.Fatal(err)
		}
		var plan []string
		for rows.Next() {
			var l string
			_ = rows.Scan(&l)
			plan = append(plan, l)
		}
		rows.Close()
		t.Log("\n" + strings.Join(plan, "\n"))
	}
}
