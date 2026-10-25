-- +goose Up
-- Deleting a voice profile (phase 5). The opt-out lives on the Person and
-- holds no voice data: while it is set, matching ignores the Person's references,
-- validations and corrections to that Person are refused, and a microphone-track
-- designation naming the Person is not applied. Deleting the Person is a separate
-- operation (Person corrections) and also removes the opt-out together with the Person.
ALTER TABLE persons ADD COLUMN voice_profile_deleted_at timestamptz;

-- Audit trail of the deletions. Person and account are kept by id and by name at the
-- time of the deletion, without foreign keys, so the trail survives later Person or
-- account deletions. Only counts are stored: no embedding, no sample, no audio.
CREATE TABLE voice_profile_deletions (
    id bigserial PRIMARY KEY,
    person_id bigint NOT NULL,
    person_name text NOT NULL,
    linked_user_id bigint,
    deleted_by bigint,
    deleted_by_username text NOT NULL DEFAULT '',
    sample_count integer NOT NULL CHECK (sample_count >= 0),
    fingerprint_count integer NOT NULL CHECK (fingerprint_count >= 0),
    attribution_count integer NOT NULL CHECK (attribution_count >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX voice_profile_deletions_person ON voice_profile_deletions(person_id, id);

-- +goose StatementBegin
CREATE FUNCTION voice_profile_deletions_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'voice_profile_deletions is append-only' USING ERRCODE = '23001';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER voice_profile_deletions_immutable BEFORE UPDATE OR DELETE ON voice_profile_deletions
    FOR EACH ROW EXECUTE FUNCTION voice_profile_deletions_immutable();

-- +goose Down
DROP TABLE voice_profile_deletions;
DROP FUNCTION voice_profile_deletions_immutable();
ALTER TABLE persons DROP COLUMN voice_profile_deleted_at;
