//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"snitch/internal/httpapi/contract"
	"snitch/migrations"
)

func TestAudioPlansMigrationPreservesLegacyConsumers(t *testing.T) {
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
	if _, err = provider.UpTo(ctx, 110); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	h := newVoiceHarness(t, pool)
	if _, err = pool.Exec(ctx, `INSERT INTO clip_audio_track_overrides(clip_id,audio_stream_index) VALUES($1,2)`, h.clipID); err != nil {
		t.Fatal(err)
	}
	var job int64
	if err = pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,result) VALUES($1,'transcription.audio.2','synthetic-legacy','done','{"provider":"fixture","model":"v1","version":"v1","language":"fr","audio_stream_index":2,"segments":[{"start":0.25,"end":0.75,"text":"ancien","words":[{"start":0.25,"end":0.75,"word":"ancien"}]}]}') RETURNING id`, h.clipID).Scan(&job); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO segment_corrections(transcription_job_id,segment_index,clip_id,text,text_author_username,text_corrected_at,version) VALUES($1,0,$2,'synthétique corrigé','legacy-editor',now(),2)`, job, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	replay, err := provider.Up(ctx)
	if err != nil || len(replay) != 0 {
		t.Fatal("migration replay", err)
	}
	var clip struct {
		Transcription contract.Transcription `json:"transcription"`
	}
	raw := h.adminCall("GET", fmt.Sprintf("/api/clips/%d", h.clipID), nil, 200)
	if err = json.Unmarshal(raw, &clip); err != nil {
		t.Fatal(err)
	}
	assertAudioContract(t, "Clip", raw)
	rev := clip.Transcription.Revision
	if rev == nil || rev.Id != fmt.Sprintf("legacy:%d", job) || rev.LegacyJobId == nil || *rev.LegacyJobId != job || rev.TimeBase != "clip" || rev.Segments[0].Words[0].Sources[0].WordIndex == nil {
		t.Fatal("legacy provenance", string(raw))
	}
	admin := h.adminUser()
	before := h.clipTranscript(admin)
	if before.Transcription.Segments[0].Text != "synthétique corrigé" || before.Transcription.Segments[0].Version != 2 {
		t.Fatal("lost correction", before)
	}
	var search correctedSearchPage
	if err = json.Unmarshal(h.adminCall("GET", "/api/search?mode=exact&q=corrig%C3%A9", nil, 200), &search); err != nil || search.Total != 1 {
		t.Fatal("legacy search", search, err)
	}
	var page exportPage
	if err = json.Unmarshal(h.adminCall("GET", segmentExportPath, nil, 200), &page); err != nil || page.Total != 1 || page.Items[0].TranscriptionJobID != job {
		t.Fatal("legacy export", page, err)
	}
	conflict := h.adminCall("PUT", segmentTextPath(h.clipID, 0), map[string]any{"text": "new correction", "expected_version": 2, "expected_revision_id": "legacy:999999"}, 409)
	if !strings.Contains(string(conflict), "transcription_changed") {
		t.Fatal(string(conflict))
	}
	h.adminCall("PUT", segmentTextPath(h.clipID, 0), map[string]any{"text": "new correction", "expected_version": 2, "expected_revision_id": rev.Id}, 200)
	if got := h.clipTranscript(admin); got.Transcription.JobID != job || got.Transcription.Segments[0].Text != "new correction" {
		t.Fatal(got)
	}
	// Projection never schedules a new ASR job, and preserves immutable worker text.
	if h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1`, h.clipID) != 1 {
		t.Fatal("migration retranscribed")
	}
	if _, err = provider.DownTo(ctx, 110); err != nil {
		t.Fatal("rollback", err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal("reapply", err)
	}
	if got := h.clipTranscript(admin); got.Transcription.Segments[0].Version != 3 {
		t.Fatal("rollback lost correction", got)
	}
}
