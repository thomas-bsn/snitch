-- +goose Up
-- Persons, attributions, validated voice samples and their immutable history
-- (phase 5, ticket 33). Later tickets extend these tables instead of replacing them.

-- A Person is an identity, never an account; user_id is an optional link.
CREATE TABLE persons (
    id bigserial PRIMARY KEY,
    display_name text NOT NULL CHECK (btrim(display_name) <> '' AND length(display_name) <= 100),
    user_id bigint UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    created_by bigint REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- The link between a local speaker and a Person. A local speaker holds at most one
-- human-validated row (the first validation stands, arbitrated by the partial unique
-- index below) and at most one machine row (written by matching in later tickets,
-- never an authority over the validated row). The effective attribution of a speaker
-- is the validated row when it exists, otherwise the machine row.
--
-- local_speakers rows are replaced by a re-diarization. The foreign key cascades so a
-- clip deletion erases derived data; a re-diarization over validated speakers is
-- refused instead (application guard + trigger below), so a human validation is never
-- lost through a cascade.
CREATE TABLE speaker_attributions (
    id bigserial PRIMARY KEY,
    local_speaker_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    person_id bigint REFERENCES persons(id),
    decision_level text NOT NULL CHECK (decision_level IN ('human','automatic','suggestion','ambiguous','unknown')),
    validation_state text NOT NULL CHECK (validation_state IN ('validated','unvalidated')),
    confidence double precision CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
    origin text NOT NULL CHECK (origin ~ '^[a-z][a-z_]{0,31}$'),
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_username text NOT NULL DEFAULT '',
    revision integer NOT NULL DEFAULT 1 CHECK (revision >= 1),
    decided_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((validation_state = 'validated') = (decision_level = 'human')),
    CHECK (decision_level IN ('unknown','ambiguous') OR person_id IS NOT NULL),
    CHECK (decision_level NOT IN ('unknown','ambiguous') OR person_id IS NULL)
);
CREATE UNIQUE INDEX speaker_attributions_validated ON speaker_attributions(local_speaker_id) WHERE validation_state = 'validated';
CREATE UNIQUE INDEX speaker_attributions_machine ON speaker_attributions(local_speaker_id) WHERE validation_state = 'unvalidated';
CREATE INDEX speaker_attributions_person ON speaker_attributions(person_id) WHERE person_id IS NOT NULL;

-- Validated voice samples are the references of a Person. A correction retires the
-- previous sample (retired_at) and creates a new one; automatic attributions never
-- create samples. Matching reads active samples only (retired_at IS NULL).
CREATE TABLE voice_samples (
    id bigserial PRIMARY KEY,
    person_id bigint NOT NULL REFERENCES persons(id),
    local_speaker_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    attribution_id bigint NOT NULL REFERENCES speaker_attributions(id) ON DELETE CASCADE,
    created_by bigint REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    retired_at timestamptz,
    retired_by bigint REFERENCES users(id) ON DELETE SET NULL,
    retired_reason text NOT NULL DEFAULT '' CHECK (retired_reason ~ '^[a-z_]{0,32}$'),
    CHECK ((retired_at IS NULL) = (retired_reason = ''))
);
CREATE UNIQUE INDEX voice_samples_active_speaker ON voice_samples(local_speaker_id) WHERE retired_at IS NULL;
CREATE INDEX voice_samples_active_person ON voice_samples(person_id) WHERE retired_at IS NULL;
CREATE INDEX voice_samples_clip ON voice_samples(clip_id);

-- Append-only history. Person and author are kept by id and by name at the time of
-- the change, without foreign keys, so the trail survives later person or account
-- deletions. No voice data is stored here. Rows only disappear through the cascade
-- from their local speaker (clip deletion); direct UPDATE/DELETE is refused.
CREATE TABLE attribution_events (
    id bigserial PRIMARY KEY,
    local_speaker_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    clip_id bigint NOT NULL,
    action text NOT NULL CHECK (action ~ '^[a-z][a-z_]{0,31}$'),
    old_person_id bigint,
    old_person_name text NOT NULL DEFAULT '',
    old_decision_level text NOT NULL DEFAULT '',
    old_sample_id bigint,
    new_person_id bigint,
    new_person_name text NOT NULL DEFAULT '',
    new_decision_level text NOT NULL DEFAULT '',
    new_sample_id bigint,
    author_id bigint,
    author_username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX attribution_events_speaker ON attribution_events(local_speaker_id, id);

-- +goose StatementBegin
CREATE FUNCTION attribution_events_immutable() RETURNS trigger AS $$
BEGIN
    -- Depth 1 is a direct statement; deeper is a foreign-key cascade.
    IF pg_trigger_depth() < 2 THEN
        RAISE EXCEPTION 'attribution_events is append-only' USING ERRCODE = '23001';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER attribution_events_immutable BEFORE UPDATE OR DELETE ON attribution_events
    FOR EACH ROW EXECUTE FUNCTION attribution_events_immutable();

-- Defense in depth: a local speaker with a human validation cannot be deleted while
-- its clip still exists (re-diarization, job deletion). When the clip itself is being
-- deleted the cascade is allowed.
-- +goose StatementBegin
CREATE FUNCTION local_speakers_keep_validated() RETURNS trigger AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM speaker_attributions WHERE local_speaker_id = OLD.id AND validation_state = 'validated')
       AND EXISTS (SELECT 1 FROM clips WHERE id = OLD.clip_id) THEN
        RAISE EXCEPTION 'local speaker % has a human validation', OLD.id USING ERRCODE = '23001';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER local_speakers_keep_validated BEFORE DELETE ON local_speakers
    FOR EACH ROW EXECUTE FUNCTION local_speakers_keep_validated();

-- +goose Down
DROP TRIGGER local_speakers_keep_validated ON local_speakers;
DROP FUNCTION local_speakers_keep_validated();
DROP TABLE attribution_events;
DROP FUNCTION attribution_events_immutable();
DROP TABLE voice_samples;
DROP TABLE speaker_attributions;
DROP TABLE persons;
