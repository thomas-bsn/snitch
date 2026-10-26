-- +goose Up
-- Accuracy criteria and history of the automation switch (phase 5). The
-- switch itself (auto_enabled) was created with the matching settings and stays false
-- until an administrator turns it on. The initial criteria are conventional starting
-- points to be adjusted after the phase 7 benchmark, not guarantees of any kind.
ALTER TABLE voice_matching_settings
    ADD COLUMN precision_threshold double precision NOT NULL DEFAULT 0.95 CHECK (precision_threshold >= 0 AND precision_threshold <= 1),
    ADD COLUMN min_confirmations integer NOT NULL DEFAULT 50 CHECK (min_confirmations >= 1),
    -- Last change of the switch; no foreign key so the history outlives the account.
    ADD COLUMN auto_changed_at timestamptz,
    ADD COLUMN auto_changed_by bigint;

-- One row per effective activation or deactivation, with the measure the administrator
-- had in front of them (no voice data). Names are snapshots: accounts may disappear.
CREATE TABLE voice_automation_events (
    id bigserial PRIMARY KEY,
    action text NOT NULL CHECK (action IN ('enable', 'disable')),
    author_id bigint,
    author_username text NOT NULL DEFAULT '',
    criteria_met boolean NOT NULL,
    forced boolean NOT NULL DEFAULT false,
    measured_precision double precision,
    confirmations integer NOT NULL DEFAULT 0,
    precision_threshold double precision NOT NULL,
    min_confirmations integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE voice_automation_events;
ALTER TABLE voice_matching_settings
    DROP COLUMN auto_changed_by,
    DROP COLUMN auto_changed_at,
    DROP COLUMN min_confirmations,
    DROP COLUMN precision_threshold;
