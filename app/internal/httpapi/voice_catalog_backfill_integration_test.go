//go:build integration

package httpapi

import (
	"strings"
	"testing"

	"snitch/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migration 00133 gives a family to the existing rows whose adapter maps to exactly one, and to nothing else.
// The test runs its Up statement on rows shaped like the ones that existed before the on-demand engines.
func testVoiceCatalogFamilyBackfill(t *testing.T, pool *pgxpool.Pool) {
	ctx := t.Context()
	raw, err := migrations.Files.ReadFile("00133_voice_model_catalog_family_backfill.sql")
	if err != nil {
		t.Fatal(err)
	}
	up := strings.SplitN(strings.TrimPrefix(string(raw), "-- +goose Up"), "-- +goose Down", 2)[0]
	rev := "0123456789abcdef0123456789abcdef01234567"
	for _, row := range [][]string{
		{"backfill-titanet", "fingerprint", "nemo-titanet", "synthetic/b-titanet"},
		{"backfill-ecapa", "fingerprint", "speechbrain-ecapa", "synthetic/b-ecapa"},
		{"backfill-signed", "fingerprint", "nemo-titanet", "synthetic/b-signed"},
		{"backfill-nemotron", "diarization", "nemo", "synthetic/b-nemotron"},
		{"backfill-pyannote", "fingerprint", "pyannote", "synthetic/b-pyannote"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,$2,$3,$4,$5)`, row[0], row[1], row[2], row[3], rev); err != nil {
			t.Fatal(err)
		}
	}
	// A model already installed on a signed Release keeps its contract.
	if _, err := pool.Exec(ctx, `INSERT INTO worker_model_orders(id,worker_id,kind,action,model_id,revision,idempotency_key,author_name,status,runtime)
		VALUES('ord_backfill_signed_000001',$2,'fingerprint','install','backfill-signed',$1,'backfill-signed','synthetic','done','{"engine":"nemo-titanet"}'::jsonb)`, rev, newVoiceHarness(t, pool).fixtureWorkerID()); err != nil {
		t.Fatal(err)
	}
	for range 2 { // idempotent
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]string{
		"backfill-titanet": "nemo", "backfill-ecapa": "speechbrain", "backfill-signed": "", "backfill-nemotron": "", "backfill-pyannote": "",
		"nvidia-sortformer-4spk-v2.1": "", "pyannote-wespeaker-resnet34": "",
	}
	for id, family := range want {
		var got string
		if err := pool.QueryRow(ctx, `SELECT family FROM voice_model_catalog WHERE id=$1`, id).Scan(&got); err != nil || got != family {
			t.Fatalf("%s: family %q (%v), want %q", id, got, err, family)
		}
	}
}
