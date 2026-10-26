-- +goose Up
-- Database-side voice matching (phase 5): configurable thresholds, the
-- automation switch, the candidates of an ambiguous voice and the model that produced
-- a machine attribution. Matching never creates voice_samples.

-- Single-row settings. auto_enabled stays false until an administrator turns the
-- automation on (automation switch): while it is false every match is at most a suggestion.
-- Similarities are cosine similarities in [-1, 1]; the initial values are not final
-- choices and wait for the phase 7 benchmark.
CREATE TABLE voice_matching_settings (
    id boolean PRIMARY KEY DEFAULT true CHECK (id),
    auto_enabled boolean NOT NULL DEFAULT false,
    automatic_threshold double precision NOT NULL DEFAULT 0.90,
    suggestion_threshold double precision NOT NULL DEFAULT 0.75,
    ambiguity_margin double precision NOT NULL DEFAULT 0.05,
    max_candidates integer NOT NULL DEFAULT 5,
    -- No foreign key: the setting must outlive the account that changed it.
    updated_by bigint,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (suggestion_threshold >= 0 AND suggestion_threshold <= automatic_threshold AND automatic_threshold <= 1),
    CHECK (ambiguity_margin >= 0 AND ambiguity_margin <= 1),
    CHECK (max_candidates BETWEEN 2 AND 20)
);
INSERT INTO voice_matching_settings(id) VALUES (true);

-- Persons a machine attribution hesitated between. Filled for ambiguous voices (every
-- Person within the ambiguity margin of the best one) and kept for suggestions and
-- automatic matches (the next best Persons) so a reviewer can see the alternatives.
CREATE TABLE speaker_attribution_candidates (
    attribution_id bigint NOT NULL REFERENCES speaker_attributions(id) ON DELETE CASCADE,
    person_id bigint NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    rank integer NOT NULL CHECK (rank >= 1),
    score double precision NOT NULL,
    PRIMARY KEY (attribution_id, person_id)
);
CREATE INDEX speaker_attribution_candidates_person ON speaker_attribution_candidates(person_id);

-- Embedding model that produced a machine attribution; absent for human validations.
ALTER TABLE speaker_attributions
    ADD COLUMN match_model text,
    ADD COLUMN match_model_version text;

-- +goose Down
ALTER TABLE speaker_attributions DROP COLUMN match_model_version, DROP COLUMN match_model;
DROP TABLE speaker_attribution_candidates;
DROP TABLE voice_matching_settings;
