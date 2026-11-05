-- +goose Up
-- A voice job may ask for one precise catalogue model (ADR 0038, benchmark). Without it the active model of the
-- job's kind applies, as in production. There is deliberately no foreign key to the catalogue, as elsewhere.
ALTER TABLE jobs ADD COLUMN requested_model_id text CHECK (requested_model_id IS NULL OR length(requested_model_id) BETWEEN 1 AND 64);

-- A benchmark job and the production job of the same clip, step and version are two different works.
DROP INDEX jobs_active_identity;
CREATE UNIQUE INDEX jobs_active_identity ON jobs(clip_id,step,input_version,(COALESCE(requested_model_id,''))) WHERE status IN ('pending','leased','retry_wait');

-- The fingerprints computed for a requested model form a benchmark group: the production never compares them
-- with its references and no identification, attribution or matching of the library reads them.
ALTER TABLE voice_fingerprints ADD COLUMN benchmark boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE voice_fingerprints DROP COLUMN benchmark;
DROP INDEX jobs_active_identity;
-- Lossy on purpose: the requested model is forgotten, so duplicates of one identity cannot all stay active.
UPDATE jobs SET status='cancelled' WHERE requested_model_id IS NOT NULL AND status IN ('pending','leased','retry_wait');
CREATE UNIQUE INDEX jobs_active_identity ON jobs(clip_id,step,input_version) WHERE status IN ('pending','leased','retry_wait');
ALTER TABLE jobs DROP COLUMN requested_model_id;
