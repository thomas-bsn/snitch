-- +goose Up
ALTER TABLE jobs ADD COLUMN cycle integer NOT NULL DEFAULT 0 CHECK (cycle >= 0);
ALTER TABLE jobs DROP CONSTRAINT jobs_status_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check CHECK (status IN ('pending','leased','done','failed','retry_wait','cancelled','dead'));
DROP INDEX jobs_active_identity;
CREATE UNIQUE INDEX jobs_active_identity ON jobs(clip_id,step,input_version) WHERE status IN ('pending','leased','retry_wait');

CREATE TABLE job_events (
 id bigserial PRIMARY KEY,
 job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
 event text NOT NULL CHECK (event IN ('attempt_started','retry_scheduled','marked_dead','admin_relaunched','cancelled','completed','lease_expired')),
 cycle integer NOT NULL CHECK (cycle >= 0),
 attempt integer CHECK (attempt IS NULL OR attempt > 0),
 actor text NOT NULL CHECK (actor IN ('admin','worker','system')),
 actor_id text NOT NULL DEFAULT '',
 error_code text NOT NULL DEFAULT '' CHECK (length(error_code) <= 64),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX job_events_history ON job_events(job_id,id);

-- +goose Down
DROP TABLE job_events;
DROP INDEX jobs_active_identity;
UPDATE jobs SET status=CASE WHEN status='retry_wait' THEN 'pending' WHEN status IN ('cancelled','dead') THEN 'failed' ELSE status END;
CREATE UNIQUE INDEX jobs_active_identity ON jobs(clip_id,step,input_version) WHERE status IN ('pending','leased');
ALTER TABLE jobs DROP CONSTRAINT jobs_status_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check CHECK (status IN ('pending','leased','done','failed'));
ALTER TABLE jobs DROP COLUMN cycle;
