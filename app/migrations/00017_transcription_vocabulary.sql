-- +goose Up
CREATE TABLE transcription_vocabulary_terms (
    id bigserial PRIMARY KEY,
    term text NOT NULL,
    variants text[] NOT NULL DEFAULT '{}',
    preferred_spelling text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX transcription_vocabulary_term_unique ON transcription_vocabulary_terms (lower(term));
CREATE TABLE transcription_vocabulary_versions (
    version text PRIMARY KEY,
    terms jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE transcription_vocabulary_versions;
DROP TABLE transcription_vocabulary_terms;
