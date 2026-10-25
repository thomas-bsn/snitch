-- +goose Up
-- Person corrections (phase 5, ticket 40): merge, separation and deletion. The
-- per-voice history stays in attribution_events (free-form action); this table keeps
-- the Person-level trail, including for a Person that no longer exists. Persons and
-- authors are stored by id and by name at the time, without foreign keys, and no voice
-- data is kept. Rows are append-only: any direct UPDATE or DELETE is refused.
CREATE TABLE person_events (
    id bigserial PRIMARY KEY,
    action text NOT NULL CHECK (action IN ('merge','split','delete')),
    person_id bigint NOT NULL,
    person_name text NOT NULL DEFAULT '',
    target_person_id bigint,
    target_person_name text NOT NULL DEFAULT '',
    attribution_count integer NOT NULL DEFAULT 0 CHECK (attribution_count >= 0),
    sample_count integer NOT NULL DEFAULT 0 CHECK (sample_count >= 0),
    microphone_rule_count integer NOT NULL DEFAULT 0 CHECK (microphone_rule_count >= 0),
    author_id bigint,
    author_username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX person_events_person ON person_events(person_id, id);
CREATE INDEX person_events_target ON person_events(target_person_id, id) WHERE target_person_id IS NOT NULL;
CREATE TRIGGER person_events_immutable BEFORE UPDATE OR DELETE ON person_events
    FOR EACH ROW EXECUTE FUNCTION attribution_events_immutable();

-- +goose Down
DROP TABLE person_events;
