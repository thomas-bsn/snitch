-- +goose Up
-- Groups of unknown voices (phase 5, ticket 36). A group is never stored: it is the
-- connected set of unknown local speakers joined by proposed proximity links, computed
-- in SQL on every read, so member and clip counts always come from the database. Every
-- unknown voice is its own group until a link joins it to another one.

-- Proposed proximity between two unknown voices, rebuilt from the stored fingerprints
-- (nearest neighbors through the HNSW index of ticket 35). A link is only a proposal:
-- nothing in this table confirms an identity or creates an attribution.
CREATE TABLE voice_group_links (
    speaker_a_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    speaker_b_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    similarity double precision NOT NULL,
    model text NOT NULL,
    model_version text NOT NULL,
    dimensions integer NOT NULL,
    computed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (speaker_a_id, speaker_b_id),
    CHECK (speaker_a_id < speaker_b_id)
);
CREATE INDEX voice_group_links_b ON voice_group_links(speaker_b_id);

-- A member a human removed from its group. The exclusion belongs to the member alone:
-- links of the other members are untouched and a new proximity computation never
-- brings the member back. It is kept apart from the links so it survives their rebuild.
CREATE TABLE voice_group_exclusions (
    local_speaker_id bigint PRIMARY KEY REFERENCES local_speakers(id) ON DELETE CASCADE,
    excluded_by bigint REFERENCES users(id) ON DELETE SET NULL,
    excluded_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE voice_group_exclusions;
DROP TABLE voice_group_links;
