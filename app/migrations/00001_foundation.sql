-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE installation (
 id integer PRIMARY KEY CHECK (id = 1),
 initialized boolean NOT NULL DEFAULT false,
 bootstrap_hash bytea,
 bootstrap_expires timestamptz,
 registration_approval boolean NOT NULL DEFAULT true
);
INSERT INTO installation(id) VALUES (1);
CREATE TABLE users (
 id bigserial PRIMARY KEY,
 username text NOT NULL,
 password_hash text,
 role text NOT NULL CHECK (role IN ('admin','user')),
 approved boolean NOT NULL DEFAULT false,
 discord_id text UNIQUE,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_username_lower ON users(lower(username));
CREATE TABLE sessions (
 token_hash bytea PRIMARY KEY,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 csrf text NOT NULL,
 expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_expiry ON sessions(expires_at);
CREATE TABLE oauth_states (
 state_hash bytea PRIMARY KEY,
 browser_hash bytea NOT NULL,
 user_id bigint REFERENCES users(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL
);
CREATE TABLE settings (
 key text PRIMARY KEY,
 value bytea NOT NULL
);

-- +goose Down
DROP TABLE settings, oauth_states, sessions, users, installation;
DROP EXTENSION IF EXISTS vector;
