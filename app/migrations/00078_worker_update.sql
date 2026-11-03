-- +goose Up
-- Automatic Worker updates (ADR 0036). The target is a released version tag, or NULL: no automatic update.
-- Singleton row, like transcription_settings.
CREATE TABLE worker_update_target (
    id boolean PRIMARY KEY DEFAULT true CHECK (id),
    version text CHECK (version IS NULL OR version ~ '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'),
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO worker_update_target(id, version) VALUES (true, NULL);
-- Last status reported by a Worker's local updater. Details are deleted with the Worker's revocation.
CREATE TABLE worker_update_status (
    worker_id text PRIMARY KEY REFERENCES workers(id) ON DELETE CASCADE,
    state text NOT NULL CHECK (state IN ('idle','pulling','restarting','updated','rolled_back','failed')),
    version text NOT NULL CHECK (version ~ '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'),
    digest text NOT NULL DEFAULT '' CHECK (digest = '' OR digest ~ '^sha256:[a-f0-9]{64}$'),
    cause_code text NOT NULL DEFAULT '' CHECK (cause_code ~ '^[a-z_]{0,40}$'),
    reported_at timestamptz NOT NULL DEFAULT now()
);
-- +goose Down
DROP TABLE worker_update_status;
DROP TABLE worker_update_target;
