-- +goose Up
-- Live logs and pause of a Worker (ADR 0063). A Worker only calls Snitch, never the other way round: the log lines
-- are pushed in batches, and the pause is a state the Worker reads, never a queue of commands.
--
-- worker_control holds the state an administrator wants (paused or not) and what the Worker says it applied.
CREATE TABLE worker_control (
    worker_id text PRIMARY KEY REFERENCES workers(id) ON DELETE CASCADE,
    paused boolean NOT NULL DEFAULT false,
    applied_paused boolean,
    requested_by bigint REFERENCES users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Lines are scrubbed by the Worker (tokens, credentialed URLs, paths) before they leave it. A session is one
-- process of the Worker: its lines are numbered from 0, which makes a batch sent twice harmless.
CREATE TABLE worker_log_lines (
    id bigserial PRIMARY KEY,
    worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    session_id text NOT NULL CHECK (session_id ~ '^[A-Za-z0-9_-]{8,64}$'),
    seq bigint NOT NULL CHECK (seq >= 0),
    line text NOT NULL CHECK (length(line) <= 4096),
    logged_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (worker_id, session_id, seq)
);
CREATE INDEX worker_log_lines_by_worker ON worker_log_lines (worker_id, id);

-- +goose Down
DROP TABLE worker_log_lines;
DROP TABLE worker_control;
