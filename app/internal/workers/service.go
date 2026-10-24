package workers

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"receipts/internal/auth"
)

var (
	ErrInvalidInput = errors.New("invalid worker input")
	ErrNotFound     = errors.New("worker not found")
	ErrUnauthorized = errors.New("worker unauthorized")
	namePattern     = regexp.MustCompile(`^[\pL\pN][\pL\pN _.-]{0,63}$`)
)

type Worker struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	MaxConcurrency      int        `json:"max_concurrency"`
	ReportedConcurrency int        `json:"reported_concurrency"`
	LastSeenAt          *time.Time `json:"last_seen_at"`
	CreatedAt           time.Time  `json:"created_at"`
	RevokedAt           *time.Time `json:"revoked_at"`
}

type Registration struct {
	Worker
	Token string `json:"token"`
}

type Input struct {
	Name           string
	MaxConcurrency int
}

type Service struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) Register(ctx context.Context, input Input) (Registration, error) {
	input.Name = strings.TrimSpace(input.Name)
	if !namePattern.MatchString(input.Name) || input.MaxConcurrency < 1 || input.MaxConcurrency > 4 {
		return Registration{}, ErrInvalidInput
	}
	id := "wk_" + auth.Token()[:28]
	token := "rw_" + auth.Token()
	var worker Worker
	err := s.pool.QueryRow(ctx, `INSERT INTO workers(id,name,token_hash,max_concurrency) VALUES($1,$2,$3,$4) RETURNING id,name,max_concurrency,reported_concurrency,last_seen_at,created_at,revoked_at`, id, input.Name, auth.Hash(token), input.MaxConcurrency).Scan(&worker.ID, &worker.Name, &worker.MaxConcurrency, &worker.ReportedConcurrency, &worker.LastSeenAt, &worker.CreatedAt, &worker.RevokedAt)
	if err != nil {
		return Registration{}, fmt.Errorf("register worker: %w", err)
	}
	return Registration{Worker: worker, Token: token}, nil
}

func (s *Service) List(ctx context.Context) ([]Worker, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,name,max_concurrency,reported_concurrency,last_seen_at,created_at,revoked_at FROM workers ORDER BY created_at,id`)
	if err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	defer rows.Close()
	items := []Worker{}
	for rows.Next() {
		var worker Worker
		if err = rows.Scan(&worker.ID, &worker.Name, &worker.MaxConcurrency, &worker.ReportedConcurrency, &worker.LastSeenAt, &worker.CreatedAt, &worker.RevokedAt); err != nil {
			return nil, fmt.Errorf("scan worker: %w", err)
		}
		items = append(items, worker)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read workers: %w", err)
	}
	return items, nil
}

func (s *Service) Revoke(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrNotFound
	}
	result, err := s.pool.Exec(ctx, `UPDATE workers SET revoked_at=COALESCE(revoked_at,now()) WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("revoke worker: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (Worker, error) {
	if !strings.HasPrefix(token, "rw_") || len(token) > 128 {
		return Worker{}, ErrUnauthorized
	}
	var worker Worker
	err := s.pool.QueryRow(ctx, `SELECT id,name,max_concurrency,reported_concurrency,last_seen_at,created_at,revoked_at FROM workers WHERE token_hash=$1 AND revoked_at IS NULL`, auth.Hash(token)).Scan(&worker.ID, &worker.Name, &worker.MaxConcurrency, &worker.ReportedConcurrency, &worker.LastSeenAt, &worker.CreatedAt, &worker.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Worker{}, ErrUnauthorized
	}
	if err != nil {
		return Worker{}, fmt.Errorf("authenticate worker: %w", err)
	}
	return worker, nil
}

func (s *Service) Touch(ctx context.Context, id string, capacity int) error {
	if capacity < 0 || capacity > 4 {
		return ErrInvalidInput
	}
	result, err := s.pool.Exec(ctx, `UPDATE workers SET last_seen_at=now(),reported_concurrency=$2 WHERE id=$1 AND revoked_at IS NULL AND max_concurrency >= $2`, id, capacity)
	if err != nil {
		return fmt.Errorf("update worker activity: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUnauthorized
	}
	return nil
}

func validID(id string) bool { return len(id) >= 23 && len(id) <= 63 && strings.HasPrefix(id, "wk_") }
