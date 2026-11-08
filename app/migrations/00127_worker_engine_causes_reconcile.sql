-- +goose Up
-- 00123 (cancellation, #351) and 00125 (removal, #353) each rewrote the order cause check from the same base, so
-- the later one dropped 'cancelled'. A cancelled install and a refused removal must both be recordable.
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled','runtime_in_use'));

-- +goose Down
UPDATE worker_engine_orders SET cause_code='runtime_incompatible' WHERE cause_code='cancelled';
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','runtime_in_use'));
