-- +goose Up
-- On-demand engines (#370, ADR 0056) are rows of the same registry as signed releases, so that orders, inventory,
-- events, bindings and dispatch keep joining on the digest. Their identity is a deterministic hash of family,
-- version and kind and carries no signature: the signature columns become nullable for them only.
ALTER TABLE voice_engine_registry ADD COLUMN source text NOT NULL DEFAULT 'signed' CHECK (source IN ('signed','on_demand'));
ALTER TABLE voice_engine_registry ALTER COLUMN key_id DROP NOT NULL;
ALTER TABLE voice_engine_registry ALTER COLUMN signed_payload DROP NOT NULL;
ALTER TABLE voice_engine_registry ALTER COLUMN signature DROP NOT NULL;
ALTER TABLE voice_engine_registry ADD CONSTRAINT voice_engine_registry_source_signature_check CHECK (
    (source = 'signed' AND key_id IS NOT NULL AND signed_payload IS NOT NULL AND signature IS NOT NULL)
    OR (source = 'on_demand' AND key_id IS NULL AND signed_payload IS NULL AND signature IS NULL));

-- Install failure causes of a package resolved by pip, on top of the union carried by 00127.
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled','runtime_in_use','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram','package_not_found','resolution_failed','build_failed'));
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_cause_code_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled','runtime_in_use','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram','package_not_found','resolution_failed','build_failed'));

-- +goose Down
UPDATE worker_engine_orders SET cause_code='runtime_incompatible' WHERE cause_code IN ('package_not_found','resolution_failed','build_failed');
UPDATE worker_engine_inventory SET cause_code='runtime_incompatible' WHERE cause_code IN ('package_not_found','resolution_failed','build_failed');
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_cause_code_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled','runtime_in_use','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram'));
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check
    CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','cancelled','runtime_in_use','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram'));
-- Rows that reference an on-demand engine cannot outlive its removal from the registry.
DELETE FROM worker_model_runtime_events WHERE digest IN (SELECT digest FROM voice_engine_registry WHERE source='on_demand');
DELETE FROM worker_model_runtime_bindings WHERE digest IN (SELECT digest FROM voice_engine_registry WHERE source='on_demand') OR previous_digest IN (SELECT digest FROM voice_engine_registry WHERE source='on_demand');
DELETE FROM worker_engine_events WHERE digest IN (SELECT digest FROM voice_engine_registry WHERE source='on_demand');
DELETE FROM worker_engine_inventory WHERE digest IN (SELECT digest FROM voice_engine_registry WHERE source='on_demand');
DELETE FROM worker_model_engine_dependencies WHERE engine_order_id IN (SELECT id FROM worker_engine_orders WHERE digest IN (SELECT digest FROM voice_engine_registry WHERE source='on_demand'));
DELETE FROM worker_engine_orders WHERE digest IN (SELECT digest FROM voice_engine_registry WHERE source='on_demand');
DELETE FROM voice_engine_registry WHERE source='on_demand';
ALTER TABLE voice_engine_registry DROP CONSTRAINT voice_engine_registry_source_signature_check;
ALTER TABLE voice_engine_registry ALTER COLUMN signature SET NOT NULL;
ALTER TABLE voice_engine_registry ALTER COLUMN signed_payload SET NOT NULL;
ALTER TABLE voice_engine_registry ALTER COLUMN key_id SET NOT NULL;
ALTER TABLE voice_engine_registry DROP COLUMN source;
