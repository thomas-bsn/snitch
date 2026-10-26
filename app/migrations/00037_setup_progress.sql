-- +goose Up
-- Progress of the guided installation, kept on the server so that an Administrator who closes the tab
-- finds the remaining steps again. A step has a row only once the person went through it ("done" or
-- "skipped"); what is really configured is read from the instance itself, never from these rows.
CREATE TABLE setup_steps (
 step text PRIMARY KEY CHECK (step IN ('discord','semantic_search','worker','import')),
 state text NOT NULL CHECK (state IN ('done','skipped')),
 updated_at timestamptz NOT NULL DEFAULT now()
);
-- Whether the "Finish the setup" banner was closed for good.
ALTER TABLE installation ADD COLUMN setup_banner_dismissed boolean NOT NULL DEFAULT false;
-- An installation that already exists was set up before the guided steps: it gets no banner.
UPDATE installation SET setup_banner_dismissed=true WHERE initialized;

-- +goose Down
ALTER TABLE installation DROP COLUMN setup_banner_dismissed;
DROP TABLE setup_steps;
