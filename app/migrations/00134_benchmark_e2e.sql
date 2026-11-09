-- +goose Up
-- End-to-end benchmark (#287, ADR 0060): a chain is a diarization model and a transcription model, measured together
-- by the speaker-attributed word error rate. It adds no job step: it composes the two stages' tasks.
ALTER TABLE benchmark_runs DROP CONSTRAINT benchmark_runs_stage_check;
ALTER TABLE benchmark_runs ADD CONSTRAINT benchmark_runs_stage_check CHECK (stage IN ('fingerprint','transcription','diarization','e2e'));

-- Who spoke each reference segment, frozen with it: the reference speaker whose turns overlap it the most. NULL when
-- no turn overlaps it.
ALTER TABLE benchmark_reference_segments ADD COLUMN speaker text;

-- +goose Down
ALTER TABLE benchmark_reference_segments DROP COLUMN speaker;
ALTER TABLE benchmark_runs DROP CONSTRAINT benchmark_runs_stage_check;
ALTER TABLE benchmark_runs ADD CONSTRAINT benchmark_runs_stage_check CHECK (stage IN ('fingerprint','transcription','diarization'));
