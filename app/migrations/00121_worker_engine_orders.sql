-- +goose Up
-- Explicit orders to install an approved engine release on one Worker, independent of any model weights. They
-- follow the model orders (ADR 0038): Snitch records the order, the Worker pulls, acknowledges and reports.
-- The signed identity is the registry digest; the local identity is what the Worker computes on its volume.
CREATE TABLE worker_engine_orders (
    id text PRIMARY KEY CHECK (id ~ '^ord_[A-Za-z0-9_-]{16,64}$'),
    worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    digest text NOT NULL REFERENCES voice_engine_registry(digest),
    engine text NOT NULL,
    version text NOT NULL,
    intent text NOT NULL CHECK (intent IN ('install')),
    -- The manifest is frozen at order time: a later release or revocation changes nothing for a queued order.
    manifest jsonb NOT NULL,
    -- Per Worker and per engine, strictly increasing: a report of an older generation is a late report.
    generation bigint NOT NULL CHECK (generation > 0),
    idempotency_key text NOT NULL UNIQUE CHECK (length(idempotency_key) BETWEEN 8 AND 100),
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_name text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','acknowledged','done','failed','refused','cancelled')),
    cause_code text NOT NULL DEFAULT '' CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing')),
    created_at timestamptz NOT NULL DEFAULT now(),
    acked_at timestamptz,
    finished_at timestamptz,
    UNIQUE (worker_id, engine, generation)
);
CREATE INDEX worker_engine_orders_open ON worker_engine_orders (worker_id) WHERE status IN ('pending','acknowledged');

-- What a Worker really holds of each approved release, as it reported it.
CREATE TABLE worker_engine_inventory (
    worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    digest text NOT NULL REFERENCES voice_engine_registry(digest),
    engine text NOT NULL,
    version text NOT NULL,
    state text NOT NULL CHECK (state IN ('requested','downloading_runtime','installing','verifying','ready','failed')),
    cause_code text NOT NULL DEFAULT '' CHECK (cause_code IN ('','disk_full','runtime_incompatible','hf_network','invalid_order','runtime_missing')),
    bytes_received bigint NOT NULL DEFAULT 0 CHECK (bytes_received >= 0),
    bytes_total bigint NOT NULL DEFAULT 0 CHECK (bytes_total >= 0),
    order_id text REFERENCES worker_engine_orders(id) ON DELETE SET NULL,
    generation bigint NOT NULL CHECK (generation > 0),
    local_identity text NOT NULL DEFAULT '' CHECK (local_identity = '' OR local_identity ~ '^[0-9a-f]{64}$'),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (worker_id, digest)
);

-- Append-only history, late reports included (ignored = true, with the reason).
CREATE TABLE worker_engine_events (
    id bigserial PRIMARY KEY,
    worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    digest text NOT NULL REFERENCES voice_engine_registry(digest),
    order_id text REFERENCES worker_engine_orders(id) ON DELETE SET NULL,
    generation bigint NOT NULL DEFAULT 0,
    state text NOT NULL,
    cause_code text NOT NULL DEFAULT '',
    ignored boolean NOT NULL DEFAULT false,
    reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX worker_engine_events_worker ON worker_engine_events (worker_id, digest, id DESC);

-- +goose Down
DROP TABLE worker_engine_events;
DROP TABLE worker_engine_inventory;
DROP TABLE worker_engine_orders;
