-- +goose Up
-- Deletion of analysis steps (ADR 0030). The guard against deleting a validated local speaker
-- stays, except inside the deletion transaction, which sets snitch.analysis_deletion locally.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION local_speakers_keep_validated() RETURNS trigger AS $$
BEGIN
    IF current_setting('snitch.analysis_deletion', true) = 'on' THEN
        RETURN OLD;
    END IF;
    IF EXISTS (SELECT 1 FROM speaker_attributions WHERE local_speaker_id = OLD.id AND validation_state = 'validated')
       AND EXISTS (SELECT 1 FROM clips WHERE id = OLD.clip_id) THEN
        RAISE EXCEPTION 'local speaker % has a human validation', OLD.id USING ERRCODE = '23001';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- Clip-level trace of an analysis deletion: author, steps and what the human work lost. It
-- survives the deletion of the steps and of the analysis history; only the clip removes it.
CREATE TABLE clip_analysis_deletions (
 id bigserial PRIMARY KEY,
 clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
 steps text[] NOT NULL CHECK (cardinality(steps) > 0),
 deleted_by bigint REFERENCES users(id) ON DELETE SET NULL,
 deleted_by_username text NOT NULL DEFAULT '',
 lost_segments integer NOT NULL DEFAULT 0 CHECK (lost_segments >= 0),
 lost_corrections integer NOT NULL DEFAULT 0 CHECK (lost_corrections >= 0),
 lost_voices integer NOT NULL DEFAULT 0 CHECK (lost_voices >= 0),
 lost_validated_voices integer NOT NULL DEFAULT 0 CHECK (lost_validated_voices >= 0),
 lost_persons integer NOT NULL DEFAULT 0 CHECK (lost_persons >= 0),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX clip_analysis_deletions_clip ON clip_analysis_deletions (clip_id, id DESC);

-- +goose Down
DROP TABLE clip_analysis_deletions;
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION local_speakers_keep_validated() RETURNS trigger AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM speaker_attributions WHERE local_speaker_id = OLD.id AND validation_state = 'validated')
       AND EXISTS (SELECT 1 FROM clips WHERE id = OLD.clip_id) THEN
        RAISE EXCEPTION 'local speaker % has a human validation', OLD.id USING ERRCODE = '23001';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
