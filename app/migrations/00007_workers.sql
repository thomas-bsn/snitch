-- +goose Up
CREATE TABLE workers (
 id text PRIMARY KEY CHECK (id ~ '^wk_[A-Za-z0-9_-]{20,60}$'),
 name text NOT NULL CHECK (length(name) BETWEEN 1 AND 64),
 token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
 max_concurrency integer NOT NULL DEFAULT 1 CHECK (max_concurrency BETWEEN 1 AND 4),
 reported_concurrency integer NOT NULL DEFAULT 0 CHECK (reported_concurrency BETWEEN 0 AND 4),
 last_seen_at timestamptz,
 revoked_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE jobs ADD COLUMN result jsonb CHECK (result IS NULL OR jsonb_typeof(result) = 'object');
ALTER TABLE jobs ADD CONSTRAINT jobs_lease_worker_fk FOREIGN KEY (lease_owner) REFERENCES workers(id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE jobs DROP CONSTRAINT jobs_lease_worker_fk;
ALTER TABLE jobs DROP COLUMN result;
DROP TABLE workers;
