-- +goose Up
ALTER TABLE workers ADD COLUMN models jsonb NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE workers DROP COLUMN models;
