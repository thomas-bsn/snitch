-- +goose Up
-- Explicit removal of one approved engine release on one Worker (#353). It is an order like the install one:
-- recorded, delivered on the Worker's pull once no model and no job still depends on the release, executed by the
-- Worker and reported. The inventory row stays as the trace of the result: absent from the Worker, not ordered again.
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_intent_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_intent_check CHECK (intent IN ('install','uninstall'));
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','runtime_in_use'));
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_state_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_state_check
    CHECK (state IN ('requested','downloading_runtime','installing','verifying','ready','failed','removing','removed'));

-- +goose Down
DELETE FROM worker_engine_events WHERE order_id IN (SELECT id FROM worker_engine_orders WHERE intent='uninstall');
UPDATE worker_engine_inventory SET state='ready',order_id=NULL WHERE state='removing';
DELETE FROM worker_engine_inventory WHERE state='removed';
DELETE FROM worker_engine_orders WHERE intent='uninstall';
UPDATE worker_engine_orders SET cause_code='runtime_incompatible' WHERE cause_code='runtime_in_use';
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_state_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_state_check
    CHECK (state IN ('requested','downloading_runtime','installing','verifying','ready','failed'));
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing'));
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_intent_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_intent_check CHECK (intent IN ('install'));
