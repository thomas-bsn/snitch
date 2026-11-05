-- +goose Up
-- Replacing the model of a type on a Worker (ADR 0038): the new one is installed and verified while the old one
-- stays in service, Snitch decides the switch, and only then does it order the removal of the old one.
CREATE TABLE worker_model_replacements (
    worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('diarization','fingerprint','separation')),
    old_model_id text NOT NULL CHECK (length(old_model_id) BETWEEN 1 AND 64),
    new_model_id text NOT NULL CHECK (length(new_model_id) BETWEEN 1 AND 64),
    -- preparing: the new model is not ready yet; switching: it is ready, the switch is decided and the removal
    -- waits (a running job, no remove order yet); removing: the remove order of the old model is out.
    phase text NOT NULL DEFAULT 'preparing' CHECK (phase IN ('preparing','switching','removing')),
    -- Why the removal of the old model failed on the Worker: it is kept and Retry is offered.
    cause_code text NOT NULL DEFAULT '' CHECK (length(cause_code) <= 40),
    remove_order_id text REFERENCES worker_model_orders(id) ON DELETE SET NULL,
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (worker_id, kind),
    CHECK (old_model_id <> new_model_id)
);

ALTER TABLE worker_model_orders DROP CONSTRAINT worker_model_orders_action_check;
ALTER TABLE worker_model_orders ADD CONSTRAINT worker_model_orders_action_check CHECK (action IN ('install','remove'));
ALTER TABLE worker_model_orders DROP CONSTRAINT worker_model_orders_status_check;
ALTER TABLE worker_model_orders ADD CONSTRAINT worker_model_orders_status_check CHECK (status IN ('pending','acknowledged','done','failed','refused','cancelled'));

ALTER TABLE worker_voice_models DROP CONSTRAINT worker_voice_models_state_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_state_check CHECK (state IN ('requested','downloading','verifying','ready','switching','removing','failed'));

-- +goose Down
-- Lossy on purpose: an order to remove a model is forgotten, and a model being removed is ready again.
UPDATE worker_voice_models SET state='ready' WHERE state IN ('switching','removing');
ALTER TABLE worker_voice_models DROP CONSTRAINT worker_voice_models_state_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_state_check CHECK (state IN ('requested','downloading','verifying','ready','failed'));
DROP TABLE worker_model_replacements;
DELETE FROM worker_model_orders WHERE action='remove' OR status='cancelled';
ALTER TABLE worker_model_orders DROP CONSTRAINT worker_model_orders_status_check;
ALTER TABLE worker_model_orders ADD CONSTRAINT worker_model_orders_status_check CHECK (status IN ('pending','acknowledged','done','failed','refused'));
ALTER TABLE worker_model_orders DROP CONSTRAINT worker_model_orders_action_check;
ALTER TABLE worker_model_orders ADD CONSTRAINT worker_model_orders_action_check CHECK (action IN ('install'));
