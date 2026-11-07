-- +goose Up
ALTER TABLE worker_install_invites ADD COLUMN encrypted_supervisor_token bytea;

-- +goose Down
ALTER TABLE worker_install_invites DROP COLUMN encrypted_supervisor_token;
