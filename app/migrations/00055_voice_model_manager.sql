-- +goose Up
-- The voice model chosen to replace the active one: at most one, from the catalogue in the code.
CREATE TABLE voice_model_candidate (
    id boolean PRIMARY KEY DEFAULT true CHECK (id),
    model_id text NOT NULL CHECK (length(model_id) BETWEEN 1 AND 64),
    chosen_at timestamptz NOT NULL DEFAULT now()
);

-- What each Worker really reports about each catalogue model (absence of a row means not installed).
CREATE TABLE worker_voice_models (
    worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    model_id text NOT NULL CHECK (length(model_id) BETWEEN 1 AND 64),
    state text NOT NULL CHECK (state IN ('downloading','verifying','ready','failed')),
    cause_code text NOT NULL DEFAULT '' CHECK (cause_code IN ('','hf_token_missing','hf_access_denied','disk_full','runtime_incompatible','inference_failed')),
    bytes_received bigint NOT NULL DEFAULT 0 CHECK (bytes_received >= 0),
    bytes_total bigint NOT NULL DEFAULT 0 CHECK (bytes_total >= 0),
    retry_requested boolean NOT NULL DEFAULT false,
    removal_requested boolean NOT NULL DEFAULT false,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (worker_id, model_id),
    CHECK ((state = 'failed') = (cause_code <> ''))
);

-- +goose Down
DROP TABLE worker_voice_models;
DROP TABLE voice_model_candidate;
