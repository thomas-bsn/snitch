-- +goose Up
ALTER TABLE voice_model_catalog DROP CONSTRAINT voice_model_catalog_engine_check;
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_engine_check
    CHECK (engine IN ('','pyannote','nemo','silero-jit','nemo-titanet','speechbrain-ecapa'));
CREATE TABLE voice_engine_registry (
    digest text PRIMARY KEY CHECK (digest ~ '^[0-9a-f]{64}$'),
    engine text NOT NULL,
    version text NOT NULL,
    kind text NOT NULL,
    key_id text NOT NULL,
    signed_payload text NOT NULL,
    signature text NOT NULL,
    manifest jsonb NOT NULL,
    revoked boolean NOT NULL DEFAULT false,
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(engine,kind,version)
);
ALTER TABLE worker_model_orders ADD COLUMN runtime jsonb;

-- +goose Down
-- Preserve references created with the new adapters during rollback.
ALTER TABLE voice_model_catalog DROP CONSTRAINT voice_model_catalog_engine_check;
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_engine_check
    CHECK (engine IN ('','pyannote','nemo','silero-jit')) NOT VALID;
ALTER TABLE worker_model_orders DROP COLUMN runtime;
DROP TABLE voice_engine_registry;
