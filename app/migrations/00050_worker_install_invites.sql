-- +goose Up
CREATE TABLE worker_install_invites (
 code_hash bytea PRIMARY KEY CHECK (octet_length(code_hash)=32),
 worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
 encrypted_token bytea NOT NULL,
 api_url text NOT NULL,
 image text NOT NULL,
 expires_at timestamptz NOT NULL
);
CREATE INDEX worker_install_invites_expiry ON worker_install_invites(expires_at);
-- +goose Down
DROP TABLE worker_install_invites;
