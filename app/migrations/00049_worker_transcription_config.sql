-- +goose Up
CREATE TABLE worker_transcription_config (
    worker_id text PRIMARY KEY REFERENCES workers(id) ON DELETE CASCADE,
    received_version text NOT NULL DEFAULT '',
    tested_version text NOT NULL DEFAULT '',
    active_version text NOT NULL DEFAULT '',
    error_code text NOT NULL DEFAULT '',
    test_requested_at timestamptz,
    tested_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    endpoint_override text NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE worker_transcription_config;
