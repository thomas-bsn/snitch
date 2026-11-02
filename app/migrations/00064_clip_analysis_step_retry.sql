-- +goose Up
-- A manual retry of a failed step: jobs of that step that ended without success before this
-- instant are ignored, so the step gets a fresh job while valid results are reused.
ALTER TABLE clip_analysis_steps ADD COLUMN retry_since timestamptz;
-- +goose Down
-- Lossy: the retry instants are dropped.
ALTER TABLE clip_analysis_steps DROP COLUMN retry_since;
