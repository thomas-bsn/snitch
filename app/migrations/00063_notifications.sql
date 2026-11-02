-- +goose Up
-- Persistent notifications. A notification is one shared action (resolution is the same for
-- everybody); reading and deleting are individual and live on the recipient row. A
-- notification stays after resolution as history until its recipient deletes it.
CREATE TABLE notifications (
 id bigserial PRIMARY KEY,
 kind text NOT NULL CHECK (kind IN ('needs_speaker_review')),
 clip_id bigint REFERENCES clips(id) ON DELETE CASCADE,
 state text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'resolved')),
 pending_count integer NOT NULL DEFAULT 0 CHECK (pending_count >= 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 resolved_at timestamptz
);
-- At most one active notification per kind and clip: voices are grouped per clip.
CREATE UNIQUE INDEX notifications_one_active ON notifications (kind, clip_id) WHERE state = 'active';
CREATE INDEX notifications_clip ON notifications (clip_id);
CREATE TABLE notification_recipients (
 notification_id bigint NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 read_at timestamptz,
 deleted_at timestamptz,
 PRIMARY KEY (notification_id, user_id)
);
CREATE INDEX notification_recipients_user ON notification_recipients (user_id, notification_id DESC);
-- +goose Down
DROP TABLE notification_recipients;
DROP TABLE notifications;
