-- +goose Up
-- The first real pulse, distinct from any authenticated call of the launcher (configuration, events).
ALTER TABLE workers ADD COLUMN first_pulse_at timestamptz;
UPDATE workers SET first_pulse_at=last_seen_at WHERE last_seen_at IS NOT NULL;
CREATE TABLE worker_install_events (
    id          bigserial PRIMARY KEY,
    worker_id   text        NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    phase       text        NOT NULL CHECK (phase IN ('prerequisites','registration','image','permissions','gpu','voice-models','voice-test','service')),
    status      text        NOT NULL CHECK (status IN ('started','succeeded','failed')),
    cause_code  text        NOT NULL DEFAULT '' CHECK (cause_code IN ('','unsupported_platform','docker_unavailable','registration_rejected','network_unreachable','image_pull_failed','volume_not_writable','hf_cache_not_writable','gpu_unavailable','service_start_failed','hf_access_denied','unknown')),
    occurred_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX worker_install_events_worker ON worker_install_events(worker_id, id DESC);
-- +goose Down
DROP TABLE worker_install_events;
ALTER TABLE workers DROP COLUMN first_pulse_at;
