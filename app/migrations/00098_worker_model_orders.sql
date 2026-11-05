-- +goose Up
-- Orders to install a catalogue model on one Worker (ADR 0038): Snitch records what it asks, the Worker pulls
-- it on its usual outgoing request, acknowledges it and reports the real state in worker_voice_models.
CREATE TABLE worker_model_orders (
    id text PRIMARY KEY CHECK (id ~ '^ord_[A-Za-z0-9_-]{16,64}$'),
    worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('diarization','fingerprint','separation')),
    action text NOT NULL CHECK (action IN ('install')),
    -- The identity of the order is what it was recorded with: the Worker checks it back at reception.
    model_id text NOT NULL CHECK (length(model_id) BETWEEN 1 AND 64),
    revision text NOT NULL CHECK (length(revision) BETWEEN 1 AND 64),
    -- A replayed order (network cut, double click) carries the same key and has no second effect.
    idempotency_key text NOT NULL UNIQUE CHECK (length(idempotency_key) BETWEEN 8 AND 100),
    -- The author is kept as written at the time, so the audit survives the deletion of the account.
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_name text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','acknowledged','done','failed','refused')),
    cause_code text NOT NULL DEFAULT '' CHECK (cause_code IN ('','hf_token_missing','hf_access_denied','disk_full','runtime_incompatible','inference_failed','hf_auth_required','hf_not_found','hf_network','models_volume_unwritable','hf_cache_error','hf_token_invalid','hf_gated_denied','hf_repo_invalid','invalid_order')),
    created_at timestamptz NOT NULL DEFAULT now(),
    acked_at timestamptz,
    finished_at timestamptz,
    CHECK ((status IN ('failed','refused')) = (cause_code <> ''))
);
CREATE INDEX worker_model_orders_open ON worker_model_orders (worker_id) WHERE status IN ('pending','acknowledged');

-- The real state of a model on a Worker gains the order step, the pinned revision and the disk it occupies.
ALTER TABLE worker_voice_models DROP CONSTRAINT worker_voice_models_state_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_state_check CHECK (state IN ('requested','downloading','verifying','ready','failed'));
ALTER TABLE worker_voice_models DROP CONSTRAINT worker_voice_models_cause_code_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_cause_code_check
 CHECK (cause_code IN ('','hf_token_missing','hf_access_denied','disk_full','runtime_incompatible','inference_failed','hf_auth_required','hf_not_found','hf_network','models_volume_unwritable','hf_cache_error','hf_token_invalid','hf_gated_denied','hf_repo_invalid','invalid_order'));
ALTER TABLE worker_voice_models ADD COLUMN revision text NOT NULL DEFAULT '' CHECK (length(revision) <= 64);
ALTER TABLE worker_voice_models ADD COLUMN occupied_bytes bigint NOT NULL DEFAULT 0 CHECK (occupied_bytes >= 0);
ALTER TABLE worker_voice_models ADD COLUMN order_id text REFERENCES worker_model_orders(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE worker_voice_models DROP COLUMN order_id;
ALTER TABLE worker_voice_models DROP COLUMN occupied_bytes;
ALTER TABLE worker_voice_models DROP COLUMN revision;
-- Lossy on purpose: a requested model has not started, and an invalid order folds onto the coarse failure.
DELETE FROM worker_voice_models WHERE state='requested';
UPDATE worker_voice_models SET cause_code='runtime_incompatible' WHERE cause_code='invalid_order';
ALTER TABLE worker_voice_models DROP CONSTRAINT worker_voice_models_cause_code_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_cause_code_check
 CHECK (cause_code IN ('','hf_token_missing','hf_access_denied','disk_full','runtime_incompatible','inference_failed','hf_auth_required','hf_not_found','hf_network','models_volume_unwritable','hf_cache_error','hf_token_invalid','hf_gated_denied','hf_repo_invalid'));
ALTER TABLE worker_voice_models DROP CONSTRAINT worker_voice_models_state_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_state_check CHECK (state IN ('downloading','verifying','ready','failed'));
DROP TABLE worker_model_orders;
