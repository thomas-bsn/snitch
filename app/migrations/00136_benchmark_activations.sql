-- +goose Up
-- The model changes an administrator made from a benchmark result (#287, ADR 0062): who changed what, from which
-- result, so a change of production model can always be traced back to the measure that motivated it.
CREATE TABLE benchmark_activations (
    id bigserial PRIMARY KEY,
    stage text NOT NULL CHECK (stage IN ('transcription','diarization')),
    from_model text NOT NULL,
    to_model text NOT NULL,
    run_id bigint REFERENCES benchmark_runs(id) ON DELETE SET NULL,
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE benchmark_activations;
