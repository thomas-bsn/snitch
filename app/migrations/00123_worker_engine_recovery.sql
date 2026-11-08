-- +goose Up
-- Leaving a failed engine installation (#351): cooperative cancellation and explicit cleanup are durable requests
-- recorded on the order, delivered on the Worker's usual pull and acknowledged by its next report. A cancelled
-- install ends as a failed state with the stable cause 'cancelled'.
ALTER TABLE worker_engine_orders
    ADD COLUMN cancel_requested_at timestamptz,
    ADD COLUMN cleanup_requested_at timestamptz,
    ADD COLUMN cleaned_at timestamptz;

ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled'));
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_cause_code_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled'));

-- +goose Down
UPDATE worker_engine_orders SET cause_code='runtime_incompatible' WHERE cause_code='cancelled';
UPDATE worker_engine_inventory SET cause_code='runtime_incompatible' WHERE cause_code='cancelled';
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_cause_code_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing'));
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing'));
ALTER TABLE worker_engine_orders DROP COLUMN cleaned_at, DROP COLUMN cleanup_requested_at, DROP COLUMN cancel_requested_at;
