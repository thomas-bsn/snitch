-- +goose Up
-- The administrator's confirmation of the losses a plan's publication will cause (#344, ADR 0054). guard is the
-- correction/validation history the preview saw and expected_revision_id the revision it would replace: the
-- publication is accepted only if both still hold, so any human edit or voice validation made after the preview
-- needs a new confirmation. One row per plan; a later confirmation replaces the previous one.
CREATE TABLE audio_plan_confirmations (
 plan_id bigint PRIMARY KEY REFERENCES audio_plans(id) ON DELETE CASCADE,
 expected_revision_id text NOT NULL,
 guard text NOT NULL,
 confirmed boolean NOT NULL,
 confirmed_by bigint,
 confirmed_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE audio_plan_confirmations;
