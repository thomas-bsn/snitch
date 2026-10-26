package settings

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"snitch/internal/auth"
	"snitch/internal/database"
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

// Get returns the stored value, or "" when there is none. A value the current key cannot decrypt (key absent,
// different, or the ciphertext damaged) is reported as "" too: it counts as not configured and is to be entered
// again, see Unreadable. Only a database failure is an error.
func (s Store) Get(ctx context.Context, name string) (string, error) {
	var b []byte
	err := s.Pool.QueryRow(ctx, `SELECT value FROM settings WHERE key=$1`, name).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	value, err := auth.Open(s.Key, name, b)
	if err != nil {
		return "", nil
	}
	return value, nil
}

// Unreadable lists, in alphabetical order, the stored settings the current key cannot decrypt.
func (s Store) Unreadable(ctx context.Context) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT key,value FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		var value []byte
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		if _, err := auth.Open(s.Key, name, value); err != nil {
			names = append(names, name)
		}
	}
	return names, rows.Err()
}
