//go:build integration

package httpapi

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"snitch/migrations"
)

// separationMigration is the version of the migration that lets a separation model be supported (#401).
const separationMigration = 150

// A catalogue that already listed separation references as unsupported (the only state possible before) keeps
// them: the candidate repository becomes the supported candidate with its limits, any other reference stays
// unsupported, and the migration can be reverted and replayed.
func TestSeparationMigrationKeepsExistingReferences(t *testing.T) {
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
	if _, err = provider.UpTo(ctx, separationMigration-1); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	const supported = `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision,supported) VALUES('synthetic-supported','separation','speechbrain-ecapa','synthetic/supported','d000000000000000000000000000000000000000',true)`
	if _, err = pool.Exec(ctx, supported); err == nil {
		t.Fatal("before the migration a supported separation reference is impossible")
	}
	// What the assisted addition recorded for these repositories until now: unsupported, under the id it derives.
	if _, err = pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,repo,revision,label,supported,unsupported_reason) VALUES
		('speechbrain-sepformer-whamr16k','separation','speechbrain/sepformer-whamr16k','0000000000000000000000000000000000000000','kept label',false,'no approved adapter runs separation yet'),
		('acme-other-separation','separation','acme/other-separation','0000000000000000000000000000000000000000','',false,'no approved adapter runs separation yet')`); err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	type row struct {
		engine, revision, label, family, reason string
		supported                               bool
		voices, rate                            int
	}
	read := func(id string) row {
		t.Helper()
		var r row
		if err := pool.QueryRow(ctx, `SELECT engine,revision,label,family,unsupported_reason,supported,separated_voices,sample_rate_hz FROM voice_model_catalog WHERE id=$1`, id).
			Scan(&r.engine, &r.revision, &r.label, &r.family, &r.reason, &r.supported, &r.voices, &r.rate); err != nil {
			t.Fatal(id, err)
		}
		return r
	}
	if r := read("speechbrain-sepformer-whamr16k"); r != (row{"speechbrain-sepformer", "21a5b500c6f52fddc387c5d9e5fb13ffd6f039c5", "kept label", "speechbrain", "", true, 2, 16000}) {
		t.Fatalf("the existing reference must become the candidate: %+v", r)
	}
	if r := read("speechbrain-sepformer-wsj03mix"); !r.supported || r.voices != 3 || r.rate != 8000 || r.family != "speechbrain" {
		t.Fatalf("the second candidate: %+v", r)
	}
	if r := read("acme-other-separation"); r.supported || r.reason == "" || r.voices != 0 || r.rate != 0 {
		t.Fatalf("another separation reference stays unsupported: %+v", r)
	}
	if _, err = pool.Exec(ctx, supported); err == nil {
		t.Fatal("a supported separation reference without its limits must be rejected")
	}
	var installed int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM worker_voice_models)+(SELECT count(*) FROM worker_model_orders)+(SELECT count(*) FROM voice_active_model WHERE kind='separation')`).Scan(&installed); err != nil || installed != 0 {
		t.Fatal("the migration installs, orders and activates nothing", installed, err)
	}

	// Reverted: separation cannot be supported again, and what the owner had added is still listed.
	if _, err = provider.DownTo(ctx, separationMigration-1); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, supported); err == nil {
		t.Fatal("after the rollback a supported separation reference is impossible again")
	}
	var others int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM voice_model_catalog WHERE kind='separation' AND NOT supported AND engine='' AND family=''`).Scan(&others); err != nil || others != 3 {
		t.Fatal("rollback lost a reference", others, err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if r := read("speechbrain-sepformer-whamr16k"); !r.supported || r.voices != 2 {
		t.Fatalf("replayed migration: %+v", r)
	}
}
