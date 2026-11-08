-- +goose Up
-- The model waits for this exact approved release on its Worker. Keeping the initial engine order links the
-- administrator's two orders; readiness follows the inventory so a later explicit retry can unblock the model.
CREATE TABLE worker_model_engine_dependencies (
    model_order_id text PRIMARY KEY REFERENCES worker_model_orders(id) ON DELETE CASCADE,
    engine_order_id text NOT NULL REFERENCES worker_engine_orders(id) ON DELETE CASCADE
);
CREATE INDEX worker_model_engine_dependencies_engine ON worker_model_engine_dependencies(engine_order_id);

-- +goose Down
DROP TABLE worker_model_engine_dependencies;
