-- +goose Up
-- Recalculation of the stored fingerprints on the net speech (#395, spec voix-breves section 6 step 3, ADR 0064).
-- Fingerprint jobs recompute the vectors from the diarizations already made: nothing is rediarized, and the old
-- fingerprint stays in service until the job's result replaces it. A run snapshots, at its start, the production
-- speaker fingerprints whose vector predates the net speech, with what each was then; its progress and its report
-- are read from the jobs and the fingerprints, never from counters, so a run resumes by itself after a restart.
-- At most one run is running. The author is a snapshot, without foreign key.
CREATE TABLE voice_recompute_runs (
    id bigserial PRIMARY KEY,
    status text NOT NULL CHECK (status IN ('running','completed','cancelled')),
    -- The number of fingerprints the administrator saw in the preview and confirmed.
    announced integer NOT NULL CHECK (announced >= 0),
    author_id bigint,
    author_username text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    -- The report frozen when the run ended; NULL while it runs.
    report jsonb,
    -- When the whole library was matched again after the run ended.
    rematched_at timestamptz,
    CHECK ((status = 'running') = (finished_at IS NULL))
);
CREATE UNIQUE INDEX voice_recompute_runs_one_running ON voice_recompute_runs((true)) WHERE status = 'running';

-- One row per original fingerprint of the run. kind is decided at the start: 'recomputable', 'missing_source' (the
-- database knows no audio source of the clip) or 'archived' (the voice has no current diarization: validated then
-- archived, without turns). Only a recomputable fingerprint is ever given a job. fingerprint_id and
-- local_speaker_id carry no foreign key: the original row is replaced when the job succeeds.
CREATE TABLE voice_recompute_items (
    run_id bigint NOT NULL REFERENCES voice_recompute_runs(id) ON DELETE CASCADE,
    fingerprint_id bigint NOT NULL,
    local_speaker_id bigint NOT NULL,
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('recomputable','missing_source','archived')),
    audio_stream_index integer NOT NULL,
    diarization_job_id bigint NOT NULL,
    -- The fingerprint step and input version of the job, for a recomputable fingerprint.
    step text,
    input_version text,
    before_usable boolean NOT NULL,
    before_contributive boolean NOT NULL,
    job_id bigint REFERENCES jobs(id) ON DELETE SET NULL,
    PRIMARY KEY (run_id, fingerprint_id),
    CHECK ((kind = 'recomputable') = (input_version IS NOT NULL AND step IS NOT NULL))
);
CREATE INDEX voice_recompute_items_job ON voice_recompute_items(job_id);

-- The jobs a run created, and only those: cancelling a run never touches another job.
CREATE TABLE voice_recompute_jobs (
    job_id bigint PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
    run_id bigint NOT NULL REFERENCES voice_recompute_runs(id) ON DELETE CASCADE
);
CREATE INDEX voice_recompute_jobs_run ON voice_recompute_jobs(run_id);

-- +goose Down
DROP TABLE voice_recompute_jobs;
DROP TABLE voice_recompute_items;
DROP TABLE voice_recompute_runs;
