-- +goose Up
CREATE TABLE worker_supervisors (
  worker_id text PRIMARY KEY REFERENCES workers(id) ON DELETE CASCADE,
  token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash)=32),
  enrolled_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz,
  revoked_at timestamptz
);

CREATE TABLE worker_restart_commands (
  id text PRIMARY KEY CHECK (id ~ '^wr_[A-Za-z0-9_-]{20,60}$'),
  worker_id text NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
  idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 1 AND 128),
  actor_id text NOT NULL,
  reason text NOT NULL CHECK (length(reason) BETWEEN 1 AND 500),
  status text NOT NULL CHECK (status IN ('requested','accepted','stopping','starting','online','failed','expired')),
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  result text NOT NULL DEFAULT '',
  UNIQUE(worker_id,idempotency_key)
);
CREATE INDEX worker_restart_commands_pending ON worker_restart_commands(worker_id,created_at DESC) WHERE status IN ('requested','accepted','stopping','starting');

CREATE TABLE worker_restart_events (
  id bigserial PRIMARY KEY,
  command_id text NOT NULL REFERENCES worker_restart_commands(id) ON DELETE CASCADE,
  actor text NOT NULL,
  status text NOT NULL,
  occurred_at timestamptz NOT NULL DEFAULT now(),
  result text NOT NULL DEFAULT ''
);
CREATE INDEX worker_restart_events_command ON worker_restart_events(command_id,occurred_at);
CREATE FUNCTION prevent_worker_restart_event_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'worker restart events are immutable'; END;
$$;
CREATE TRIGGER worker_restart_events_immutable BEFORE UPDATE OR DELETE ON worker_restart_events FOR EACH ROW EXECUTE FUNCTION prevent_worker_restart_event_mutation();

-- +goose Down
DROP TABLE worker_restart_events;
DROP FUNCTION prevent_worker_restart_event_mutation();
DROP TABLE worker_restart_commands;
DROP TABLE worker_supervisors;
