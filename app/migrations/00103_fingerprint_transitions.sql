-- +goose Up
-- Changing the fingerprint model makes the stored fingerprints incomparable (ADR 0044). The change is an explicit
-- transition: the references of the Persons are recomputed with the target model in a staging space (benchmark
-- fingerprints), then Snitch switches the active model and the two spaces at once. There is deliberately no
-- foreign key to the catalogue, as elsewhere. At most one transition is open.
CREATE TABLE voice_fingerprint_transitions (
    id bigserial PRIMARY KEY,
    from_model_id text NOT NULL CHECK (length(from_model_id) BETWEEN 1 AND 64),
    to_model_id text NOT NULL CHECK (length(to_model_id) BETWEEN 1 AND 64),
    -- preparing: no connected Worker holds the target ready yet; recomputing: the references are recomputed;
    -- switched: the target is the active model; cancelled: abandoned before the switch.
    phase text NOT NULL DEFAULT 'preparing' CHECK (phase IN ('preparing','recomputing','switched','cancelled')),
    -- The number of references the administrator saw and confirmed.
    announced_references integer NOT NULL CHECK (announced_references >= 0),
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,
    author_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    switched_at timestamptz,
    CHECK (from_model_id <> to_model_id)
);
CREATE UNIQUE INDEX voice_fingerprint_transitions_open ON voice_fingerprint_transitions((true)) WHERE phase IN ('preparing','recomputing');

-- +goose Down
DROP TABLE voice_fingerprint_transitions;
