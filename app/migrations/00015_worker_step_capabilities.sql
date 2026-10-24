-- +goose Up
ALTER TABLE workers ADD COLUMN supported_steps text[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE workers DROP COLUMN supported_steps;
