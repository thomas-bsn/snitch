-- +goose Up
-- Updating the runtime of a model that is already Ready, and going back (#352). An update or a rollback is a model
-- order like an install (ADR 0038), flagged by its intent: the weights already installed and validated are
-- reused, and the model stays Ready on its current binding until the Worker proves the new one with an inference.
ALTER TABLE worker_model_orders ADD COLUMN runtime_intent text NOT NULL DEFAULT 'install' CHECK (runtime_intent IN ('install','update','rollback'));

-- The runtime a model is bound to on a Worker, as Snitch published it, and the one it was bound to before.
CREATE TABLE worker_model_runtime_bindings (
    worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    model_id text NOT NULL REFERENCES voice_model_catalog(id) ON DELETE CASCADE,
    digest text NOT NULL REFERENCES voice_engine_registry(digest),
    previous_digest text REFERENCES voice_engine_registry(digest),
    order_id text REFERENCES worker_model_orders(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (worker_id, model_id)
);

-- Append-only history of runtime updates and rollbacks, late reports included (ignored = true, with the reason).
CREATE TABLE worker_model_runtime_events (
    id bigserial PRIMARY KEY,
    worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    model_id text NOT NULL REFERENCES voice_model_catalog(id) ON DELETE CASCADE,
    order_id text REFERENCES worker_model_orders(id) ON DELETE SET NULL,
    intent text NOT NULL CHECK (intent IN ('update','rollback')),
    digest text NOT NULL REFERENCES voice_engine_registry(digest),
    state text NOT NULL CHECK (state IN ('requested','downloading','verifying','published','failed')),
    cause_code text NOT NULL DEFAULT '',
    ignored boolean NOT NULL DEFAULT false,
    reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX worker_model_runtime_events_model ON worker_model_runtime_events (worker_id, model_id, id DESC);

-- +goose Down
DROP TABLE worker_model_runtime_events;
DROP TABLE worker_model_runtime_bindings;
DELETE FROM worker_model_orders WHERE runtime_intent <> 'install';
ALTER TABLE worker_model_orders DROP COLUMN runtime_intent;
