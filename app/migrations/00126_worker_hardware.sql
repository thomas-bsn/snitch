-- +goose Up
-- What a Worker reports of its own machine, so that an engine release can be checked against real constraints
-- before any order is given (python, platform, CUDA, RAM, VRAM, free disk). A Worker that never reported has no row:
-- the check then passes and the Worker still checks by itself before installing.
CREATE TABLE worker_hardware (
    worker_id text PRIMARY KEY REFERENCES workers(id) ON DELETE CASCADE,
    python text NOT NULL DEFAULT '' CHECK (python ~ '^([0-9]{1,2}\.[0-9]{1,2})?$'),
    platform text NOT NULL DEFAULT '' CHECK (platform ~ '^[a-z0-9_-]{0,32}$'),
    -- '' means that no usable CUDA device is present.
    cuda text NOT NULL DEFAULT '' CHECK (cuda ~ '^([0-9]{1,2}\.[0-9]{1,2})?$'),
    ram_bytes bigint NOT NULL DEFAULT 0 CHECK (ram_bytes >= 0),
    vram_bytes bigint NOT NULL DEFAULT 0 CHECK (vram_bytes >= 0),
    disk_free_bytes bigint NOT NULL DEFAULT 0 CHECK (disk_free_bytes >= 0),
    -- The Worker's own configuration: running a CUDA release on the CPU is only possible when it is enabled.
    cpu_fallback boolean NOT NULL DEFAULT false,
    reported_at timestamptz NOT NULL DEFAULT now()
);

-- Stable causes of a resource failure reported by a Worker.
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram'));
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_cause_code_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_cause_code_check CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing','platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram'));

-- +goose Down
UPDATE worker_engine_orders SET cause_code='runtime_incompatible' WHERE cause_code IN ('platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram');
UPDATE worker_engine_inventory SET cause_code='runtime_incompatible' WHERE cause_code IN ('platform_incompatible','cuda_incompatible','insufficient_ram','insufficient_vram');
ALTER TABLE worker_engine_inventory DROP CONSTRAINT worker_engine_inventory_cause_code_check;
ALTER TABLE worker_engine_inventory ADD CONSTRAINT worker_engine_inventory_cause_code_check CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing'));
ALTER TABLE worker_engine_orders DROP CONSTRAINT worker_engine_orders_cause_code_check;
ALTER TABLE worker_engine_orders ADD CONSTRAINT worker_engine_orders_cause_code_check CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing'));
DROP TABLE worker_hardware;
