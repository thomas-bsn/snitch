-- +goose Up
-- Trace of every requalification of the stored fingerprints (#394, spec voix-breves section 6): who applied which
-- rule, when, and what it changed. A requalification is launched by saving the rule, changed or not; running it
-- again under the same rule changes no fingerprint but is traced as well. Append-only; the author is a snapshot,
-- without foreign key: accounts may disappear.
--
--   changed           production fingerprints (both scopes) whose eligibility moved
--   missing_sources   production speaker fingerprints requalified although the database knows no audio source of
--                     their clip (requalification reads no media; a recalculation would need one)
--   estimated         of which net speech is the fingerprint's duration (no measures, no stored turns)
--   original_vectors  speaker fingerprints whose vector predates the net speech (no Worker measures)
--   before, after     production speaker fingerprints by eligibility, and Persons with a Reference
CREATE TABLE voice_eligibility_requalifications (
    id bigserial PRIMARY KEY,
    rule_version integer NOT NULL REFERENCES voice_eligibility_rules(version),
    author_id bigint,
    author_username text,
    created_at timestamptz NOT NULL DEFAULT now(),
    changed integer NOT NULL CHECK (changed >= 0),
    missing_sources integer NOT NULL CHECK (missing_sources >= 0),
    estimated integer NOT NULL CHECK (estimated >= 0),
    original_vectors integer NOT NULL CHECK (original_vectors >= 0),
    before jsonb NOT NULL,
    after jsonb NOT NULL
);

-- +goose Down
DROP TABLE voice_eligibility_requalifications;
