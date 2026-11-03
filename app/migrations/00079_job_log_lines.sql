-- +goose Up
-- Last lines of a job as the Worker logged them, already masked by the Worker. The server keeps the
-- 50 most recent per job; deleting the job deletes them. Administrators only (ADR 0028).
CREATE TABLE job_log_lines (
 id bigserial PRIMARY KEY,
 job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
 text text NOT NULL CHECK (length(text) BETWEEN 1 AND 2000),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX job_log_lines_job ON job_log_lines (job_id, id);

-- +goose Down
DROP TABLE job_log_lines;
