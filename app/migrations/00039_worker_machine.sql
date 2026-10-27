-- +goose Up
ALTER TABLE workers ADD COLUMN machine jsonb NOT NULL DEFAULT '{}', ADD COLUMN remote_address text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE workers DROP COLUMN machine, DROP COLUMN remote_address;
