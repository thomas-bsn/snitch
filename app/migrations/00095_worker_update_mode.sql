-- +goose Up
-- Worker update strategy (ADR 0036, amended): disabled, pinned to a chosen version, or following the newest
-- published release. "version" stays the effective target in every mode: the administrator's choice when pinned,
-- the highest release seen so far when latest (it only ever grows). Existing choices become "pinned".
ALTER TABLE worker_update_target
    ADD COLUMN mode text NOT NULL DEFAULT 'disabled' CHECK (mode IN ('disabled','pinned','latest'));
UPDATE worker_update_target SET mode = 'pinned' WHERE version IS NOT NULL;
ALTER TABLE worker_update_target
    ADD CONSTRAINT worker_update_target_mode_version CHECK (
        (mode = 'disabled' AND version IS NULL) OR (mode = 'pinned' AND version IS NOT NULL) OR mode = 'latest');
-- +goose Down
ALTER TABLE worker_update_target DROP CONSTRAINT worker_update_target_mode_version;
UPDATE worker_update_target SET version = NULL WHERE mode = 'latest';
ALTER TABLE worker_update_target DROP COLUMN mode;
