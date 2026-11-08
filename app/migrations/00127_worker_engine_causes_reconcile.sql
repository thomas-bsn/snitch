-- +goose Up
-- 00123 (cancellation, #351), 00125 (removal, #353) and 00126 (resource checks, #354) each rewrote the cause check of
-- worker_engine_orders and worker_engine_inventory from the same base, so the last one applied dropped the values of
-- the others ('cancelled' for 00125; 'runtime_in_use' for 00126). This migration carries the union of every ticket:
-- a cancelled install, a refused removal and a resource refusal must all be recordable. The inventory states
-- ('removing', 'removed' from 00125) and the order intents ('install', 'uninstall') are set only by 00125 and are kept.
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled','runtime_in_use','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram'));
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_cause_code_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled','runtime_in_use','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram'));

-- +goose Down
UPDATE worker_engine_orders SET cause_code='runtime_incompatible' WHERE cause_code='runtime_in_use';
UPDATE worker_engine_inventory SET cause_code='runtime_incompatible' WHERE cause_code='runtime_in_use';
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_cause_code_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram'));
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram'));
