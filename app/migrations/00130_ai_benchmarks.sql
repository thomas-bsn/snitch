-- +goose Up
-- Benchmarks of the AI models of one step (#287, ADR 0056). Everything here is experimental data: no table of
-- this migration is read by identification, attribution, matching or the library, and a run only schedules jobs
-- that ask for a precise catalogue model (ADR 0038), whose fingerprints are flagged benchmark.

-- A dataset is an annotated set of local speakers. The ground truth is a snapshot of the human validations at
-- creation time: a later correction of a Person never changes the dataset, so a run stays reproducible.
CREATE TABLE benchmark_datasets (
    id bigserial PRIMARY KEY,
    name text NOT NULL CHECK (btrim(name) <> '' AND length(name) <= 100),
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX benchmark_datasets_name ON benchmark_datasets(lower(name));

-- split separates the references (enrolment of each Person) from the tests (voices to identify): a test voice
-- is never its own reference. A re-diarization or a clip deletion removes the speaker and so its item; the run
-- results already computed keep their own counts.
CREATE TABLE benchmark_dataset_items (
    dataset_id bigint NOT NULL REFERENCES benchmark_datasets(id) ON DELETE CASCADE,
    local_speaker_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    person_id bigint NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    split text NOT NULL CHECK (split IN ('reference','test')),
    PRIMARY KEY (dataset_id, local_speaker_id)
);
CREATE INDEX benchmark_dataset_items_speaker ON benchmark_dataset_items(local_speaker_id);

-- One active run at a time: two runs over the same clip and model would share one job.
CREATE TABLE benchmark_runs (
    id bigserial PRIMARY KEY,
    dataset_id bigint NOT NULL REFERENCES benchmark_datasets(id) ON DELETE RESTRICT,
    dataset_name text NOT NULL,
    stage text NOT NULL CHECK (stage = 'fingerprint'),
    status text NOT NULL CHECK (status IN ('running','completed','failed','cancelled')),
    model_ids text[] NOT NULL CHECK (cardinality(model_ids) BETWEEN 1 AND 4),
    params jsonb NOT NULL DEFAULT '{}',
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz
);
CREATE UNIQUE INDEX benchmark_runs_one_active ON benchmark_runs((true)) WHERE status = 'running';

CREATE TABLE benchmark_run_jobs (
    run_id bigint NOT NULL REFERENCES benchmark_runs(id) ON DELETE CASCADE,
    job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    model_id text NOT NULL,
    PRIMARY KEY (run_id, job_id, model_id)
);
CREATE INDEX benchmark_run_jobs_job ON benchmark_run_jobs(job_id);

-- The model and its revision are copied at the end of the run: the catalogue may change afterwards. Scores of
-- different models are never put side by side as one scale: each model carries its own threshold.
CREATE TABLE benchmark_results (
    run_id bigint NOT NULL REFERENCES benchmark_runs(id) ON DELETE CASCADE,
    model_id text NOT NULL,
    model text NOT NULL,
    model_version text NOT NULL DEFAULT '',
    dimensions integer NOT NULL DEFAULT 0,
    state text NOT NULL CHECK (state IN ('complete','partial','failed')),
    items_total integer NOT NULL,
    items_scored integer NOT NULL,
    elapsed_seconds double precision,
    metrics jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, model_id)
);

-- +goose Down
DROP TABLE benchmark_results;
DROP TABLE benchmark_run_jobs;
DROP TABLE benchmark_runs;
DROP TABLE benchmark_dataset_items;
DROP TABLE benchmark_datasets;
