-- +goose Up
-- Where the reference of a certified test clip comes from (#287, ADR 0061): the production pipeline, corrected by
-- humans (biased in favour of the production model), or an STM/RTTM file written by hand (the bias is lifted).
-- NULL for a clip with no reference (reference clips, uncertified test clips).
ALTER TABLE benchmark_dataset_clips ADD COLUMN reference_source text CHECK (reference_source IN ('production','import'));
UPDATE benchmark_dataset_clips SET reference_source='production' WHERE verified;

-- +goose Down
ALTER TABLE benchmark_dataset_clips DROP COLUMN reference_source;
