-- +goose Up
-- A durable analysis run of a clip: one active run per clip, steps persisted with a derived
-- status and a stable reason code. Jobs stay the unit of work; a step only points at its job.
CREATE TABLE clip_analysis_runs (
 id bigserial PRIMARY KEY,
 clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
 scope text NOT NULL,
 status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'failed', 'cancelled')),
 started_by bigint REFERENCES users(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 finished_at timestamptz
);
CREATE UNIQUE INDEX clip_analysis_runs_one_active ON clip_analysis_runs (clip_id) WHERE status = 'active';
CREATE INDEX clip_analysis_runs_clip ON clip_analysis_runs (clip_id, id DESC);
CREATE TABLE clip_analysis_steps (
 run_id bigint NOT NULL REFERENCES clip_analysis_runs(id) ON DELETE CASCADE,
 step text NOT NULL,
 status text NOT NULL CHECK (status IN ('waiting', 'running', 'retry', 'blocked', 'failed', 'done', 'not_applicable')),
 reason text NOT NULL DEFAULT '',
 job_id bigint REFERENCES jobs(id) ON DELETE SET NULL,
 attempt integer NOT NULL DEFAULT 0,
 started_at timestamptz,
 finished_at timestamptz,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (run_id, step)
);
-- +goose Down
DROP TABLE clip_analysis_steps;
DROP TABLE clip_analysis_runs;
