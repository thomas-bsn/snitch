-- +goose Up
-- Journal of the refusals of an identification (ADR 0031): a refusal, its cancellation
-- ("Annuler le refus"), and what a new diarization does with it (carried to the new voice
-- that covers most of the old one's speaking time, or dropped). Append-only, like the
-- disposition journal; names are snapshots. It outlives its voice and its Person: their
-- deletion only clears the reference (ON DELETE SET NULL, done by the cascade).
CREATE TABLE voice_refusal_events (
    id bigserial PRIMARY KEY,
    local_speaker_id bigint REFERENCES local_speakers(id) ON DELETE SET NULL,
    clip_id bigint NOT NULL,
    person_id bigint REFERENCES persons(id) ON DELETE SET NULL,
    person_name text NOT NULL DEFAULT '',
    action text NOT NULL CHECK (action IN ('refuse','cancel','carry','drop')),
    author_id bigint,
    author_username text NOT NULL DEFAULT '',
    reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX voice_refusal_events_speaker ON voice_refusal_events(local_speaker_id, id);

-- +goose StatementBegin
CREATE FUNCTION voice_refusal_events_immutable() RETURNS trigger AS $$
BEGIN
    -- Depth 1 is a direct statement; deeper is a foreign-key action (ON DELETE SET NULL),
    -- which is the only update the journal admits.
    IF pg_trigger_depth() < 2 THEN
        RAISE EXCEPTION 'voice_refusal_events is append-only' USING ERRCODE = '23001';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER voice_refusal_events_immutable BEFORE UPDATE OR DELETE ON voice_refusal_events
    FOR EACH ROW EXECUTE FUNCTION voice_refusal_events_immutable();

-- +goose Down
DROP TRIGGER voice_refusal_events_immutable ON voice_refusal_events;
DROP FUNCTION voice_refusal_events_immutable();
DROP TABLE voice_refusal_events;
