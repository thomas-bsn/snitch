-- +goose Up
-- Reanalysis of an already processed clip: the mode the administrator chose, the steps forced
-- to run again (a forced step ignores the valid results created before the run) and the
-- consequences confirmed before the run started (kept in the history).
ALTER TABLE clip_analysis_runs
 ADD COLUMN mode text NOT NULL DEFAULT 'missing' CHECK (mode IN ('missing','full','steps')),
 ADD COLUMN forced_steps text[] NOT NULL DEFAULT '{}',
 ADD COLUMN confirmed_replace_corrections boolean NOT NULL DEFAULT false,
 ADD COLUMN confirmed_replace_validations boolean NOT NULL DEFAULT false;
-- +goose Down
-- Lossy: the reanalysis choices and confirmations of past runs are dropped.
ALTER TABLE clip_analysis_runs DROP COLUMN confirmed_replace_validations, DROP COLUMN confirmed_replace_corrections, DROP COLUMN forced_steps, DROP COLUMN mode;
