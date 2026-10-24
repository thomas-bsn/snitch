-- +goose Up
CREATE TABLE jobs (
 id bigserial PRIMARY KEY,
 clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
 step text NOT NULL CHECK (step ~ '^[a-z][a-z0-9_.-]{0,63}$'),
 input_version text NOT NULL CHECK (length(input_version) BETWEEN 1 AND 128),
 status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','leased','done','failed')),
 priority integer NOT NULL DEFAULT 0 CHECK (priority BETWEEN -100 AND 100),
 attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
 max_attempts integer NOT NULL DEFAULT 3 CHECK (max_attempts > 0),
 lease_owner text,
 lease_expires_at timestamptz,
 next_attempt_at timestamptz NOT NULL DEFAULT now(),
 last_error text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK ((status = 'leased') = (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL))
);
CREATE UNIQUE INDEX jobs_active_identity ON jobs(clip_id,step,input_version) WHERE status IN ('pending','leased');
CREATE INDEX jobs_queue_order ON jobs(status,priority DESC,created_at,id);
CREATE INDEX jobs_clip_history ON jobs(clip_id,created_at DESC,id DESC);

-- +goose Down
DROP TABLE jobs;
