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

// A library with an overlapped fingerprint migrates: 00144 marks it overlap_only, a reason the check it replaces
// did not allow. An empty database never exercised that update.
func TestEligibilityMigrationWithOverlappedFingerprint(t *testing.T) {
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
	if _, err = provider.UpTo(ctx, 143); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// The smallest diarized library: one clip, its diarization and one local speaker with an overlapped fingerprint.
	if _, err = pool.Exec(ctx, `INSERT INTO users(id,username,role) VALUES (1,'owner','admin');
INSERT INTO clips(id,owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES (1,1,'synthetic.mp4',1,now(),now());
INSERT INTO jobs(id,clip_id,step,input_version,status) VALUES (1,1,'voice.diarization.2','v1','done'),(2,1,'voice.fingerprint.2','v1','done');
INSERT INTO local_speakers(id,clip_id,diarization_job_id,audio_stream_index,label,ordinal,speaking_seconds,turn_count) VALUES (1,1,1,2,'SPEAKER_00',0,2,1);
INSERT INTO voice_fingerprints(id,local_speaker_id,clip_id,fingerprint_job_id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,model,model_version,dimensions,embedding,measures_version,net_speech_seconds,overlap_ratio,overlapped)
VALUES (1,1,1,2,'speaker',0,2,2,0.4,true,'synthetic','1',2,'[1,0]',1,0,1,true);`); err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal("a library with an overlapped fingerprint does not migrate:", err)
	}
	var usable, contributive bool
	var reason string
	if err = pool.QueryRow(ctx, `SELECT usable,contributive,unusable_reason FROM voice_fingerprints WHERE id=1`).Scan(&usable, &contributive, &reason); err != nil {
		t.Fatal(err)
	}
	if usable || contributive || reason != "overlap_only" {
		t.Fatalf("usable=%v contributive=%v reason=%q", usable, contributive, reason)
	}
}
