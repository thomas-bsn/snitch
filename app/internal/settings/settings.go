package settings

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"receipts/internal/auth"
	"receipts/internal/database"
)

type Store struct {
	Pool database.DBTX
	Key  []byte
}

// WithTx uses the caller's transaction; the caller retains commit/rollback ownership.
func (s Store) WithTx(tx pgx.Tx) Store {
	s.Pool = tx
	return s
}

func (s Store) Set(ctx context.Context, name, value string) error {
	encrypted, err := auth.Seal(s.Key, name, value)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO settings(key,value) VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, name, encrypted)
	return err
}
func (s Store) Get(ctx context.Context, name string) (string, error) {
	var b []byte
	err := s.Pool.QueryRow(ctx, `SELECT value FROM settings WHERE key=$1`, name).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return auth.Open(s.Key, name, b)
}
