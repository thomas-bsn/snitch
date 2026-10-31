-- +goose Up
-- Account ↔ Person link requests and their immutable audit trail (issue #96).
-- A Person is never an account (ADR 0011); this lot lets a signed-in user ask to
-- link its account to a Person, and an Administrator approve or refuse. The link
-- itself stays on persons.user_id, unique in both directions.

CREATE TABLE person_link_requests (
    id bigserial PRIMARY KEY,
    person_id bigint NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','refused')),
    message text NOT NULL DEFAULT '' CHECK (length(message) <= 500),
    created_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz,
    decided_by bigint REFERENCES users(id) ON DELETE SET NULL,
    decided_by_username text NOT NULL DEFAULT '',
    -- A decision always carries its author and date; a pending request carries neither.
    CHECK ((status = 'pending') = (decided_at IS NULL))
);
-- At most one pending request per account and per Person.
CREATE UNIQUE INDEX person_link_requests_pending_user ON person_link_requests(user_id) WHERE status='pending';
CREATE UNIQUE INDEX person_link_requests_pending_person ON person_link_requests(person_id) WHERE status='pending';
CREATE INDEX person_link_requests_person ON person_link_requests(person_id, id);

-- Append-only audit of every request and every link/unlink, without voice data.
-- Person, accounts and authors are kept by id and by name at the time, without
-- foreign keys, so the trail survives later Person or account deletions. Rows only
-- disappear through the cascade from their Person or account; direct UPDATE/DELETE
-- is refused.
CREATE TABLE person_link_events (
    id bigserial PRIMARY KEY,
    action text NOT NULL CHECK (action IN ('request','cancel','approve','refuse','link','unlink')),
    person_id bigint NOT NULL,
    person_name text NOT NULL DEFAULT '',
    user_id bigint,
    username text NOT NULL DEFAULT '',
    previous_user_id bigint,
    previous_username text NOT NULL DEFAULT '',
    actor_id bigint,
    actor_username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX person_link_events_person ON person_link_events(person_id, id);
CREATE TRIGGER person_link_events_immutable BEFORE UPDATE OR DELETE ON person_link_events
    FOR EACH ROW EXECUTE FUNCTION attribution_events_immutable();

-- +goose Down
DROP TRIGGER person_link_events_immutable ON person_link_events;
DROP TABLE person_link_events;
DROP TABLE person_link_requests;
