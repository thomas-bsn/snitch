-- +goose Up
-- Diarization benchmark (#287, ADR 0059). Same shape as the transcription benchmark: a step of its own
-- (benchmark.diarization.N) whose result stays in jobs.result, and a reference frozen in the corpus.
ALTER TABLE benchmark_runs DROP CONSTRAINT benchmark_runs_stage_check;
ALTER TABLE benchmark_runs ADD CONSTRAINT benchmark_runs_stage_check CHECK (stage IN ('fingerprint','transcription','diarization'));

-- The reference turns of a corpus, frozen at creation: the speaking turns of the certified clips, each speaker
-- being the validated Person (or, without validation, the local voice itself). Times are on the audio stream's own
-- clock, as a Worker measures them (the plan offset of a multitrack stream is taken off).
CREATE TABLE benchmark_reference_turns (
    dataset_id bigint NOT NULL REFERENCES benchmark_datasets(id) ON DELETE CASCADE,
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    audio_stream_index integer NOT NULL CHECK (audio_stream_index >= 0),
    speaker text NOT NULL CHECK (speaker <> ''),
    start_seconds double precision NOT NULL CHECK (start_seconds >= 0),
    end_seconds double precision NOT NULL,
    CHECK (end_seconds > start_seconds)
);
CREATE INDEX benchmark_reference_turns_dataset ON benchmark_reference_turns(dataset_id, clip_id, audio_stream_index);

-- +goose Down
DROP TABLE benchmark_reference_turns;
ALTER TABLE benchmark_runs DROP CONSTRAINT benchmark_runs_stage_check;
ALTER TABLE benchmark_runs ADD CONSTRAINT benchmark_runs_stage_check CHECK (stage IN ('fingerprint','transcription'));
