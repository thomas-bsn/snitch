//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"snitch/migrations"
)

// Accounts that had a photo before the Discord sources existed keep it, as their explicit personal choice.
func TestAvatarMigrationKeepsExistingPhotos(t *testing.T) {
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
	if _, err = provider.UpTo(ctx, 93); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	photo := solid(colourOf(hashRed), 256, 256)
	var withPhoto, without int64
	if err = pool.QueryRow(ctx, `INSERT INTO users(username,role,approved) VALUES('old-with-photo','user',true) RETURNING id`).Scan(&withPhoto); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO users(username,role,approved) VALUES('old-without','user',true) RETURNING id`).Scan(&without); err != nil {
		t.Fatal(err)
	}
	var linked int64
	if err = pool.QueryRow(ctx, `INSERT INTO users(username,role,approved,discord_id) VALUES('old-linked','user',true,'4242') RETURNING id`).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO account_avatars(user_id,png) VALUES($1,$2)`, withPhoto, photo); err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	var source string
	var stored []byte
	var version int64
	if err = pool.QueryRow(ctx, `SELECT p.source,p.version,a.png FROM avatar_preferences p JOIN account_avatars a ON a.user_id=p.user_id AND a.source=p.source WHERE p.user_id=$1`, withPhoto).Scan(&source, &version, &stored); err != nil {
		t.Fatal("the existing photo is no longer the displayed one:", err)
	}
	if source != "custom" || !bytes.Equal(stored, photo) {
		t.Fatalf("source %q, photo kept=%v", source, bytes.Equal(stored, photo))
	}
	var rows int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM avatar_preferences WHERE user_id=$1`, without).Scan(&rows); err != nil || rows != 0 {
		t.Fatal("an account without a photo got a preference", err, rows)
	}
	if err = pool.QueryRow(ctx, `SELECT source FROM avatar_preferences WHERE user_id=$1`, linked).Scan(&source); err != nil || source != "discord" {
		t.Fatal("an account linked before the migration does not adopt its Discord photo:", source, err)
	}
	// The migration can be reverted: the personal photo is still there afterwards.
	if _, err = provider.DownTo(ctx, 93); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT png FROM account_avatars WHERE user_id=$1`, withPhoto).Scan(&stored); err != nil || !bytes.Equal(stored, photo) {
		t.Fatal("rollback lost the photo", err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
}
