-- +goose Up
ALTER TABLE worker_install_invites ADD COLUMN used_at timestamptz;
-- +goose Down
ALTER TABLE worker_install_invites DROP COLUMN used_at;
