-- +goose Up
-- Typed and bounded diagnostic of a worker failure; no free-form column, older rows keep NULL.
ALTER TABLE job_events
 ADD COLUMN step text CHECK (step IS NULL OR length(step) <= 64),
 ADD COLUMN provider text CHECK (provider IS NULL OR length(provider) <= 64),
 ADD COLUMN model text CHECK (model IS NULL OR length(model) <= 128),
 ADD COLUMN revision text CHECK (revision IS NULL OR length(revision) <= 128),
 ADD COLUMN http_status integer CHECK (http_status IS NULL OR http_status BETWEEN 100 AND 599),
 ADD COLUMN duration_ms integer CHECK (duration_ms IS NULL OR duration_ms >= 0);

-- +goose Down
ALTER TABLE job_events DROP COLUMN step, DROP COLUMN provider, DROP COLUMN model, DROP COLUMN revision, DROP COLUMN http_status, DROP COLUMN duration_ms;
