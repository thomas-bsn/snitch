-- +goose Up
-- The 'incomplete' run status was never written: partial identification comes from the voices
-- the clip really has left to decide, not from the run status.
UPDATE clip_analysis_runs SET status = 'completed' WHERE status = 'incomplete';
ALTER TABLE clip_analysis_runs DROP CONSTRAINT clip_analysis_runs_status_check;
ALTER TABLE clip_analysis_runs ADD CONSTRAINT clip_analysis_runs_status_check CHECK (status IN ('active', 'completed', 'failed', 'cancelled'));
-- The instant up to which the human work at stake was shown and confirmed: a reanalysis only
-- makes obsolete what existed then, never a correction or a validation made afterwards.
ALTER TABLE clip_analysis_runs ADD COLUMN confirmed_stake_at timestamptz;
-- +goose Down
-- Lossy: the confirmed stake instants are dropped.
ALTER TABLE clip_analysis_runs DROP COLUMN confirmed_stake_at;
ALTER TABLE clip_analysis_runs DROP CONSTRAINT clip_analysis_runs_status_check;
ALTER TABLE clip_analysis_runs ADD CONSTRAINT clip_analysis_runs_status_check CHECK (status IN ('active', 'completed', 'incomplete', 'failed', 'cancelled'));
