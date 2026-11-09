-- +goose Up
-- Transcription benchmark (#287, ADR 0058). A run of this stage schedules benchmark.transcription.N jobs: a step of
-- their own, so a benchmark transcript is never read as the transcript of a clip, nor indexed, nor corrected.

-- The widest stage list a run may carry; each slice of #287 opens one more.
ALTER TABLE benchmark_runs DROP CONSTRAINT benchmark_runs_stage_check;
ALTER TABLE benchmark_runs ADD CONSTRAINT benchmark_runs_stage_check CHECK (stage IN ('fingerprint','transcription'));

-- The reference text of a corpus, frozen when it is created: the corrected text of the segments of every
-- certified clip (a human correction wins over the production text). A later correction never changes a run.
CREATE TABLE benchmark_reference_segments (
    dataset_id bigint NOT NULL REFERENCES benchmark_datasets(id) ON DELETE CASCADE,
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    audio_stream_index integer NOT NULL CHECK (audio_stream_index >= 0),
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    start_seconds double precision NOT NULL CHECK (start_seconds >= 0),
    end_seconds double precision NOT NULL,
    text text NOT NULL CHECK (btrim(text) <> ''),
    corrected boolean NOT NULL,
    PRIMARY KEY (dataset_id, clip_id, audio_stream_index, ordinal),
    CHECK (end_seconds >= start_seconds)
);

-- +goose Down
DROP TABLE benchmark_reference_segments;
ALTER TABLE benchmark_runs DROP CONSTRAINT benchmark_runs_stage_check;
ALTER TABLE benchmark_runs ADD CONSTRAINT benchmark_runs_stage_check CHECK (stage = 'fingerprint');
