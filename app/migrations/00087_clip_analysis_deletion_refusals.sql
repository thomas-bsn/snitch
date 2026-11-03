-- +goose Up
-- The audit of an analysis deletion also records the refusals and the dispositions (ignored or
-- random) of the deleted voices: deleting the voices drops them (ADR 0031).
ALTER TABLE clip_analysis_deletions
 ADD COLUMN lost_refusals integer NOT NULL DEFAULT 0 CHECK (lost_refusals >= 0),
 ADD COLUMN lost_dispositions integer NOT NULL DEFAULT 0 CHECK (lost_dispositions >= 0);

-- +goose Down
ALTER TABLE clip_analysis_deletions
 DROP COLUMN lost_dispositions,
 DROP COLUMN lost_refusals;
