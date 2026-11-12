-- +goose Up
-- Discord webhook (#283, ADR 0067): the notifications stay the single source of truth; the webhook is one more
-- delivery channel. Its address is a secret, sealed in settings ('notification_webhook_url'); this row only holds
-- what the administration may read back.
CREATE TABLE notification_webhook (
 id boolean PRIMARY KEY DEFAULT true CHECK (id),
 enabled boolean NOT NULL DEFAULT false,
 categories text[] NOT NULL DEFAULT '{access_requests}' CHECK (categories <@ ARRAY['access_requests', 'human_actions', 'worker_offline', 'job_failures', 'job_blocked']::text[]),
 -- The address of Snitch seen when the webhook was saved, for the links of the messages ('' : no link).
 public_origin text NOT NULL DEFAULT '',
 -- The last four digits of the webhook id, to tell which one is configured without reading the secret.
 hint text NOT NULL DEFAULT '',
 last_status text NOT NULL DEFAULT 'never' CHECK (last_status IN ('never', 'sent', 'failed', 'rate_limited')),
 last_error text NOT NULL DEFAULT '',
 last_attempt_at timestamptz,
 last_success_at timestamptz,
 updated_at timestamptz NOT NULL DEFAULT now()
);

-- The category of a notification kind; NULL for a kind never relayed outside Snitch (a folder decision is private
-- to its requester).
-- +goose StatementBegin
CREATE FUNCTION notification_category(kind text) RETURNS text LANGUAGE sql IMMUTABLE AS $$
 SELECT CASE kind
  WHEN 'account_request' THEN 'access_requests'
  WHEN 'needs_speaker_review' THEN 'human_actions'
  WHEN 'worker_unavailable' THEN 'worker_offline'
  WHEN 'job_failed' THEN 'job_failures'
  WHEN 'provider_unavailable' THEN 'job_failures'
  WHEN 'model_missing' THEN 'job_failures'
  WHEN 'job_blocked' THEN 'job_blocked'
 END
$$;
-- +goose StatementEnd

-- One delivery per notification at most (the outbox): a retried or concurrent raise never posts twice.
CREATE TABLE notification_deliveries (
 id bigserial PRIMARY KEY,
 notification_id bigint NOT NULL UNIQUE REFERENCES notifications(id) ON DELETE CASCADE,
 category text NOT NULL,
 status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed', 'skipped')),
 attempts integer NOT NULL DEFAULT 0,
 next_attempt_at timestamptz NOT NULL DEFAULT now(),
 -- A code, never the raw error: a transport error spells out the address, and the address is the secret.
 last_error text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),
 sent_at timestamptz
);
CREATE INDEX notification_deliveries_due ON notification_deliveries (next_attempt_at, id) WHERE status = 'pending';

-- Enqueued in the transaction that raises the notification: no network call there, only a row.
-- +goose StatementBegin
CREATE FUNCTION notification_delivery_enqueue() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO notification_deliveries(notification_id, category)
 SELECT NEW.id, notification_category(NEW.kind) FROM notification_webhook w
 WHERE w.enabled AND NEW.state = 'active' AND notification_category(NEW.kind) = ANY (w.categories)
 ON CONFLICT DO NOTHING;
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER notifications_delivery_enqueue AFTER INSERT ON notifications FOR EACH ROW EXECUTE FUNCTION notification_delivery_enqueue();

-- Audit of the changes, immutable; the address is never written here.
CREATE TABLE notification_webhook_events (
 id bigserial PRIMARY KEY,
 action text NOT NULL CHECK (action IN ('created', 'rotated', 'updated', 'deleted', 'tested', 'auto_disabled')),
 enabled boolean NOT NULL DEFAULT false,
 categories text[] NOT NULL DEFAULT '{}',
 actor_id bigint,
 actor_username text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER notification_webhook_events_immutable BEFORE UPDATE OR DELETE ON notification_webhook_events
 FOR EACH ROW EXECUTE FUNCTION attribution_events_immutable();

-- +goose Down
DROP TRIGGER notification_webhook_events_immutable ON notification_webhook_events;
DROP TABLE notification_webhook_events;
DROP TRIGGER notifications_delivery_enqueue ON notifications;
DROP FUNCTION notification_delivery_enqueue();
DROP TABLE notification_deliveries;
DROP FUNCTION notification_category(text);
DROP TABLE notification_webhook;
DELETE FROM settings WHERE key = 'notification_webhook_url';
