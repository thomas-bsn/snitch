package setup

import (
	"context"
	"crypto/subtle"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"receipts/internal/auth"
	"time"
)

var ErrClosed = errors.New("installation closed or code invalid")

// Issue only replaces an expired code, allowing concurrent API replicas safely.
func Issue(ctx context.Context, p *pgxpool.Pool) (string, error) {
	code := auth.Token()
	tag, err := p.Exec(ctx, `UPDATE installation SET bootstrap_hash=$1, bootstrap_expires=now()+interval '15 minutes' WHERE id=1 AND NOT initialized AND (bootstrap_expires IS NULL OR bootstrap_expires<=now())`, auth.Hash(code))
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", nil
	}
	return code, nil
}
func Complete(ctx context.Context, p *pgxpool.Pool, code, username, password string) (int64, error) {
	tx, err := p.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var initialized bool
	var h []byte
	var expires *time.Time
	if err = tx.QueryRow(ctx, `SELECT initialized,bootstrap_hash,bootstrap_expires FROM installation WHERE id=1 FOR UPDATE`).Scan(&initialized, &h, &expires); err != nil {
		return 0, err
	}
	if initialized || expires == nil || !expires.After(time.Now()) || subtle.ConstantTimeCompare(h, auth.Hash(code)) != 1 {
		return 0, ErrClosed
	}
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO users(username,password_hash,role,approved) VALUES($1,$2,'admin',true) RETURNING id`, username, auth.Password(password)).Scan(&id)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `UPDATE installation SET initialized=true,bootstrap_hash=NULL,bootstrap_expires=NULL WHERE id=1`)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit(ctx)
}
