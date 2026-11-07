-- +goose Up
-- One row per plan whose voice analysis was started (#342). stake_at is the instant the
-- human work at stake was counted (and, when there was some, confirmed): a validation or
-- speaker correction made after it protects the voices of the tracks being replaced.
CREATE TABLE audio_plan_voice_starts (
 plan_id bigint PRIMARY KEY REFERENCES audio_plans(id) ON DELETE CASCADE,
 pipeline_version text NOT NULL,
 stake_at timestamptz NOT NULL,
 stake jsonb NOT NULL CHECK (jsonb_typeof(stake)='object'),
 confirmed boolean NOT NULL DEFAULT false,
 started_by bigint,
 started_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE audio_plan_voice_starts;
