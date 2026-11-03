-- +goose Up
-- Demandes de création de dossier (ADR 0033, #225). A member asks for a folder of the Racine vidéo;
-- an Administrator approves (the folder is created) or refuses. The trail is immutable, as for the
-- Person link requests.
CREATE TABLE folder_requests (
    id bigserial PRIMARY KEY,
    -- Root-relative paths: '' is the Racine vidéo itself.
    parent text NOT NULL DEFAULT '' CHECK (parent NOT LIKE '/%' AND parent NOT LIKE '%/'),
    name text NOT NULL CHECK (length(name) BETWEEN 1 AND 255 AND name NOT LIKE '%/%' AND name NOT LIKE '.%'),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','refused')),
    requested_by bigint REFERENCES users(id) ON DELETE SET NULL,
    requester_username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz,
    decided_by bigint REFERENCES users(id) ON DELETE SET NULL,
    decided_by_username text NOT NULL DEFAULT '',
    -- A decision always carries its author and date; a pending request carries neither.
    CHECK ((status = 'pending') = (decided_at IS NULL))
);
-- At most one pending request per folder.
CREATE UNIQUE INDEX folder_requests_pending ON folder_requests(parent, name) WHERE status = 'pending';
CREATE INDEX folder_requests_requester ON folder_requests(requested_by, id);

CREATE TABLE folder_request_events (
    id bigserial PRIMARY KEY,
    action text NOT NULL CHECK (action IN ('request','approve','refuse')),
    parent text NOT NULL DEFAULT '',
    name text NOT NULL DEFAULT '',
    requester_id bigint,
    requester_username text NOT NULL DEFAULT '',
    actor_id bigint,
    actor_username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER folder_request_events_immutable BEFORE UPDATE OR DELETE ON folder_request_events
    FOR EACH ROW EXECUTE FUNCTION attribution_events_immutable();

-- The requester learns the decision through a notification with no clip. Its folder and outcome
-- travel with it; they are empty for every other kind.
ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN ('needs_speaker_review', 'worker_unavailable', 'provider_unavailable', 'model_missing', 'job_failed', 'job_blocked', 'folder_request_decided'));
ALTER TABLE notifications ADD COLUMN folder_path text NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN folder_outcome text NOT NULL DEFAULT '' CHECK (folder_outcome IN ('', 'approved', 'refused'));

-- +goose Down
ALTER TABLE notifications DROP COLUMN folder_outcome;
ALTER TABLE notifications DROP COLUMN folder_path;
DELETE FROM notification_recipients WHERE notification_id IN (SELECT id FROM notifications WHERE kind='folder_request_decided');
DELETE FROM notifications WHERE kind='folder_request_decided';
ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN ('needs_speaker_review', 'worker_unavailable', 'provider_unavailable', 'model_missing', 'job_failed', 'job_blocked'));
DROP TRIGGER folder_request_events_immutable ON folder_request_events;
DROP TABLE folder_request_events;
DROP TABLE folder_requests;
