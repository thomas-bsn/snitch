//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// reindexProvider is a deterministic embeddings server: the vector of a text only
// depends on the text, and the calls are recorded. It can be made unavailable.
type reindexProvider struct {
	mu    sync.Mutex
	down  bool
	calls [][]string
	srv   *httptest.Server
}

// vectorOf is "[len,1,first byte]"; the same text always gives the same vector.
func vectorOf(text string) []float32 {
	return []float32{float32(len(text)), 1, float32(text[0])}
}

func newReindexProvider(t *testing.T) *reindexProvider {
	p := &reindexProvider{}
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		p.mu.Lock()
		down := p.down
		if !down {
			p.calls = append(p.calls, in.Input)
		}
		p.mu.Unlock()
		if down {
			http.Error(w, "synthetic outage", http.StatusServiceUnavailable)
			return
		}
		data := make([]map[string]any, len(in.Input))
		for i, text := range in.Input {
			data[i] = map[string]any{"index": i, "embedding": vectorOf(text)}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(p.srv.Close)
	return p
}

func (p *reindexProvider) setDown(down bool) {
	p.mu.Lock()
	p.down = down
	p.mu.Unlock()
}

func (p *reindexProvider) reset() {
	p.mu.Lock()
	p.calls = nil
	p.mu.Unlock()
}

func (p *reindexProvider) seen() [][]string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([][]string(nil), p.calls...)
}

type embeddingRow struct {
	Text string
	Vec  string
	Xmin string
}

func embeddingRows(t *testing.T, pool *pgxpool.Pool, profile, job int64) map[int]embeddingRow {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT segment_index,text,embedding::text,xmin::text FROM transcript_embeddings WHERE profile_id=$1 AND transcription_job_id=$2`, profile, job)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int]embeddingRow{}
	for rows.Next() {
		var i int
		var r embeddingRow
		if err := rows.Scan(&i, &r.Text, &r.Vec, &r.Xmin); err != nil {
			t.Fatal(err)
		}
		out[i] = r
	}
	return out
}

func testSegmentReindex(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.transcribe()
	provider := newReindexProvider(t)
	h.api.Config.EmbeddingsURL, h.api.Config.EmbeddingsModel = provider.srv.URL, "fixture-v1"
	if err := h.api.ensureEmbeddingProfile(ctx); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, "/api/admin/embeddings/backfill", map[string]any{}, http.StatusAccepted)
	if err := h.api.indexNext(ctx); err != nil {
		t.Fatal(err)
	}
	var active, job int64
	if err := pool.QueryRow(ctx, `SELECT active_profile_id FROM embedding_index_state WHERE id=1`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND status='done'`, h.clipID).Scan(&job); err != nil {
		t.Fatal(err)
	}
	before := embeddingRows(t, pool, active, job)
	if len(before) != 3 {
		t.Fatalf("the transcription must be indexed first: %+v", before)
	}
	status := func() (pending, failed int) {
		var s struct {
			Pending int `json:"reindex_pending"`
			Failed  int `json:"reindex_failed"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/embeddings", nil, http.StatusOK), &s); err != nil {
			t.Fatal(err)
		}
		return s.Pending, s.Failed
	}
	if p, _ := status(); p != 0 {
		t.Fatalf("no segment is waiting before any correction: %d", p)
	}
	correct := func(index int, text string) {
		h.adminCall(http.MethodPut, segmentTextPath(h.clipID, index), map[string]any{"text": text, "expected_version": 0}, http.StatusOK)
	}

	t.Run("an unavailable provider keeps the correction and the segment pending", func(t *testing.T) {
		provider.setDown(true)
		correct(1, "second corrigé")
		if p, f := status(); p != 1 || f != 0 {
			t.Fatalf("the corrected segment waits for reindexing: pending=%d failed=%d", p, f)
		}
		if err := h.api.reindexNext(ctx); err == nil {
			t.Fatal("an unavailable provider must be reported")
		}
		if got := h.clipTranscript(h.adminUser()).Transcription.Segments[1].Text; got != "second corrigé" {
			t.Fatalf("the correction stays recorded: %q", got)
		}
		var pending bool
		var attempts int
		var message string
		if err := pool.QueryRow(ctx, `SELECT reindex_pending,reindex_attempts,reindex_error FROM segment_corrections WHERE transcription_job_id=$1 AND segment_index=1`, job).Scan(&pending, &attempts, &message); err != nil || !pending || attempts != 1 || message == "" {
			t.Fatalf("pending=%v attempts=%d error=%q err=%v", pending, attempts, message, err)
		}
		// Backoff: the segment is not retried at once.
		provider.setDown(false)
		provider.reset()
		if err := h.api.reindexNext(ctx); err != nil || len(provider.seen()) != 0 {
			t.Fatalf("a failed segment waits before its next attempt: %v %v", err, provider.seen())
		}
	})

	t.Run("the segment is resumed once, without duplicate and without touching the others", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE segment_corrections SET reindex_lease_expires_at=now()-interval '1 second' WHERE transcription_job_id=$1`, job); err != nil {
			t.Fatal(err)
		}
		provider.reset()
		if err := h.api.reindexNext(ctx); err != nil {
			t.Fatal(err)
		}
		if calls := provider.seen(); len(calls) != 1 || len(calls[0]) != 1 || calls[0][0] != "second corrigé" {
			t.Fatalf("only the corrected text is embedded: %v", calls)
		}
		after := embeddingRows(t, pool, active, job)
		if len(after) != 3 || after[1].Text != "second corrigé" || after[1].Xmin == before[1].Xmin {
			t.Fatalf("the corrected vector replaces the old one: %+v", after)
		}
		if want := fmt.Sprintf("[%d,1,%d]", len("second corrigé"), 's'); after[1].Vec != want {
			t.Fatalf("vector %q, want %q", after[1].Vec, want)
		}
		for _, i := range []int{0, 2} {
			if after[i] != before[i] {
				t.Fatalf("segment %d must not be rewritten: %+v != %+v", i, after[i], before[i])
			}
		}
		if p, _ := status(); p != 0 {
			t.Fatalf("the segment is no longer pending: %d", p)
		}
		provider.reset()
		if err := h.api.reindexNext(ctx); err != nil || len(provider.seen()) != 0 {
			t.Fatalf("nothing is left to do: %v %v", err, provider.seen())
		}
	})

	t.Run("a live lease is respected, an expired one is taken over, attempts are bounded", func(t *testing.T) {
		correct(2, "ailleurs corrigé")
		if _, err := pool.Exec(ctx, `UPDATE segment_corrections SET reindex_lease_expires_at=now()+interval '1 minute',reindex_attempts=1 WHERE transcription_job_id=$1 AND segment_index=2`, job); err != nil {
			t.Fatal(err)
		}
		provider.reset()
		if err := h.api.reindexNext(ctx); err != nil || len(provider.seen()) != 0 {
			t.Fatalf("a live lease is respected: %v %v", err, provider.seen())
		}
		if _, err := pool.Exec(ctx, `UPDATE segment_corrections SET reindex_attempts=$2,reindex_lease_expires_at=now()-interval '1 second' WHERE transcription_job_id=$1 AND segment_index=2`, job, maxReindexAttempts); err != nil {
			t.Fatal(err)
		}
		if err := h.api.reindexNext(ctx); err != nil || len(provider.seen()) != 0 {
			t.Fatalf("exhausted attempts are not retried: %v %v", err, provider.seen())
		}
		if p, f := status(); p != 1 || f != 1 {
			t.Fatalf("an exhausted segment is reported: pending=%d failed=%d", p, f)
		}
		h.adminCall(http.MethodPost, "/api/admin/embeddings/backfill", map[string]any{}, http.StatusAccepted)
		if err := h.api.reindexNext(ctx); err != nil {
			t.Fatal(err)
		}
		if p, f := status(); p != 0 || f != 0 {
			t.Fatalf("a backfill gives exhausted segments a new chance: pending=%d failed=%d", p, f)
		}
		if rows := embeddingRows(t, pool, active, job); len(rows) != 3 || rows[2].Text != "ailleurs corrigé" {
			t.Fatalf("%+v", rows)
		}
	})

	t.Run("the candidate index receives the correction before the switch", func(t *testing.T) {
		h.adminCall(http.MethodPost, "/api/admin/embeddings/rebuild", map[string]string{"model": "fixture-v2"}, http.StatusAccepted)
		var candidate int64
		if err := pool.QueryRow(ctx, `SELECT candidate_profile_id FROM embedding_index_state WHERE id=1`).Scan(&candidate); err != nil {
			t.Fatal(err)
		}
		correct(0, "premier corrigé")
		if err := h.api.reindexNext(ctx); err != nil {
			t.Fatal(err)
		}
		if rows := embeddingRows(t, pool, active, job); rows[0].Text != "premier corrigé" {
			t.Fatalf("the active index is updated: %+v", rows)
		}
		if rows := embeddingRows(t, pool, candidate, job); len(rows) != 1 || rows[0].Text != "premier corrigé" {
			t.Fatalf("only the corrected segment is written to the candidate for now: %+v", rows)
		}
		for i := 0; i < 5; i++ {
			if err := h.api.indexNext(ctx); err != nil {
				t.Fatal(err)
			}
		}
		var model string
		if err := pool.QueryRow(ctx, `SELECT p.model FROM embedding_index_state s JOIN embedding_profiles p ON p.id=s.active_profile_id WHERE s.id=1`).Scan(&model); err != nil || model != "fixture-v2" {
			t.Fatalf("the candidate became active: %q %v", model, err)
		}
		rows := embeddingRows(t, pool, candidate, job)
		if len(rows) != 3 || rows[0].Text != "premier corrigé" || rows[1].Text != "second corrigé" || rows[2].Text != "ailleurs corrigé" {
			t.Fatalf("the switch must not bring the old texts back: %+v", rows)
		}
	})
}
