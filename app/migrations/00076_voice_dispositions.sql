-- +goose Up
-- Disposition of a local speaker and refusal of an identification (ADR 0031). A disposition
-- is not a decision level: it says whether a voice is counted at all. A refusal says that a
-- Person is not this voice, so that matching never proposes it again.

-- One current disposition per local speaker, absent for a voice that is handled normally.
-- Re-diarization replaces local speakers, which drops their dispositions with them.
CREATE TABLE local_speaker_dispositions (
    local_speaker_id bigint PRIMARY KEY REFERENCES local_speakers(id) ON DELETE CASCADE,
    disposition text NOT NULL CHECK (disposition IN ('ignored','random')),
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_username text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Journal of every change of disposition, restoration included. Append-only, like the
-- attribution history; names are snapshots.
CREATE TABLE voice_disposition_events (
    id bigserial PRIMARY KEY,
    local_speaker_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    clip_id bigint NOT NULL,
    old_disposition text NOT NULL DEFAULT '' CHECK (old_disposition IN ('','ignored','random')),
    new_disposition text NOT NULL DEFAULT '' CHECK (new_disposition IN ('','ignored','random')),
    author_id bigint,
    author_username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (old_disposition <> new_disposition)
);
CREATE INDEX voice_disposition_events_speaker ON voice_disposition_events(local_speaker_id, id);

-- +goose StatementBegin
CREATE FUNCTION voice_disposition_events_immutable() RETURNS trigger AS $$
BEGIN
    -- Depth 1 is a direct statement; deeper is a foreign-key cascade.
    IF pg_trigger_depth() < 2 THEN
        RAISE EXCEPTION 'voice_disposition_events is append-only' USING ERRCODE = '23001';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER voice_disposition_events_immutable BEFORE UPDATE OR DELETE ON voice_disposition_events
    FOR EACH ROW EXECUTE FUNCTION voice_disposition_events_immutable();

-- A Person refused for a voice: matching never proposes it for that voice again. It is a
-- rule about the voice, kept apart from the attributions so that removing them keeps it.
CREATE TABLE voice_refusals (
    local_speaker_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    person_id bigint NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (local_speaker_id, person_id)
);
CREATE INDEX voice_refusals_person ON voice_refusals(person_id);

-- +goose Down
DROP TABLE voice_refusals;
DROP TRIGGER voice_disposition_events_immutable ON voice_disposition_events;
DROP FUNCTION voice_disposition_events_immutable();
DROP TABLE voice_disposition_events;
DROP TABLE local_speaker_dispositions;
