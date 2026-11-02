-- +goose Up
-- Live notifications: every change that a recipient can see (a notification raised, refreshed,
-- resolved, read, unread or deleted for them) writes one event on the topic of that user.
-- The stream answers with a fresh snapshot, so the event carries no data.
-- +goose StatementBegin
CREATE FUNCTION notification_recipient_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO events(topic, kind) VALUES ('user:' || NEW.user_id, 'notifications');
 RETURN NULL;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION notification_shared_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO events(topic, kind) SELECT 'user:' || user_id, 'notifications' FROM notification_recipients WHERE notification_id = NEW.id AND deleted_at IS NULL;
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER notification_recipients_event AFTER INSERT OR UPDATE ON notification_recipients FOR EACH ROW EXECUTE FUNCTION notification_recipient_event();
CREATE TRIGGER notifications_event AFTER UPDATE ON notifications FOR EACH ROW EXECUTE FUNCTION notification_shared_event();
-- +goose Down
DROP TRIGGER notifications_event ON notifications;
DROP TRIGGER notification_recipients_event ON notification_recipients;
DROP FUNCTION notification_shared_event();
DROP FUNCTION notification_recipient_event();
