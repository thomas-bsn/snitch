-- +goose Up
-- Voice matching runs in the background (#392). A change of references, fingerprints or settings records a
-- request; a background pass takes every request it finds at once and re-reads the latest committed state, so
-- several close decisions make one pass. A request is deleted only by the pass that committed after reading it:
-- a pass cut by a stop or a failure leaves its requests, and the next start (or the next retry) resumes them.
CREATE TABLE voice_matching_requests (
    id bigserial PRIMARY KEY,
    -- NULL: the whole library; otherwise only the voices of that clip (new fingerprints). No foreign key: a
    -- request for a deleted clip simply finds nothing to match.
    clip_id bigint CHECK (clip_id IS NULL OR clip_id > 0),
    requested_at timestamptz NOT NULL DEFAULT now()
);

-- Where matching stands, for the interface: is a pass running, when did the last one end, did it fail.
CREATE TABLE voice_matching_status (
    id boolean PRIMARY KEY DEFAULT true CHECK (id),
    running_since timestamptz,
    last_finished_at timestamptz,
    last_duration_ms bigint CHECK (last_duration_ms >= 0),
    last_requests integer CHECK (last_requests >= 0),
    last_failed_at timestamptz
);
INSERT INTO voice_matching_status(id) VALUES (true);

-- Live views follow requests and passes through the outbox (one event per statement).
-- +goose StatementBegin
CREATE FUNCTION voice_matching_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO events(topic, kind) VALUES ('voice_matching', 'status');
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER voice_matching_requests_event AFTER INSERT OR DELETE ON voice_matching_requests FOR EACH STATEMENT EXECUTE FUNCTION voice_matching_event();
CREATE TRIGGER voice_matching_status_event AFTER UPDATE ON voice_matching_status FOR EACH STATEMENT EXECUTE FUNCTION voice_matching_event();

-- +goose Down
DROP TRIGGER voice_matching_status_event ON voice_matching_status;
DROP TRIGGER voice_matching_requests_event ON voice_matching_requests;
DROP FUNCTION voice_matching_event();
DROP TABLE voice_matching_status;
DROP TABLE voice_matching_requests;
