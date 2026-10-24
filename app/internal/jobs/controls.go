package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrInvalidControls = errors.New("invalid queue controls")

type Controls struct {
	Paused         bool      `json:"paused"`
	MaxConcurrency int       `json:"max_concurrency"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ControlsInput struct {
	Paused         bool
	MaxConcurrency int
}

func (s *Service) GetControls(ctx context.Context) (Controls, error) {
	var controls Controls
	err := s.pool.QueryRow(ctx, `SELECT paused,max_concurrency,updated_at FROM job_control WHERE id=true`).Scan(&controls.Paused, &controls.MaxConcurrency, &controls.UpdatedAt)
	if err != nil {
		return Controls{}, fmt.Errorf("read job controls: %w", err)
	}
	return controls, nil
}

func (s *Service) UpdateControls(ctx context.Context, input ControlsInput) (Controls, error) {
	if input.MaxConcurrency < 1 || input.MaxConcurrency > 4 {
		return Controls{}, ErrInvalidControls
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Controls{}, fmt.Errorf("begin job controls update: %w", err)
	}
	defer tx.Rollback(ctx)
	var controls Controls
	err = tx.QueryRow(ctx, `UPDATE job_control SET paused=$1,max_concurrency=$2,updated_at=now() WHERE id=true RETURNING paused,max_concurrency,updated_at`, input.Paused, input.MaxConcurrency).Scan(&controls.Paused, &controls.MaxConcurrency, &controls.UpdatedAt)
	if err != nil {
		return Controls{}, fmt.Errorf("update job controls: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Controls{}, fmt.Errorf("commit job controls update: %w", err)
	}
	return controls, nil
}
