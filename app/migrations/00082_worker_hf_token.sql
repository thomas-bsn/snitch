-- +goose Up
-- Central Hugging Face token (#194, ADR 0027). The token itself lives sealed in the settings table under the key
-- hf_token_central; this table only records, per Worker, the version of that token it last acknowledged, where its
-- copy came from, and the stable cause of the last failed application.
CREATE TABLE worker_hf_token (
    worker_id text PRIMARY KEY REFERENCES workers(id) ON DELETE CASCADE,
    acked_version text NOT NULL DEFAULT '0',
    source text NOT NULL DEFAULT '' CHECK (source IN ('','central','local','none')),
    error_code text NOT NULL DEFAULT '' CHECK (error_code IN ('','hf_token_store_failed')),
    updated_at timestamptz NOT NULL DEFAULT now()
);
-- The optional wizard step that asks for the central token.
ALTER TABLE setup_steps DROP CONSTRAINT IF EXISTS setup_steps_step_check;
ALTER TABLE setup_steps ADD CONSTRAINT setup_steps_step_check CHECK (step IN ('discord','semantic_search','worker','import','location','huggingface'));

-- +goose Down
DELETE FROM setup_steps WHERE step='huggingface';
ALTER TABLE setup_steps DROP CONSTRAINT IF EXISTS setup_steps_step_check;
ALTER TABLE setup_steps ADD CONSTRAINT setup_steps_step_check CHECK (step IN ('discord','semantic_search','worker','import','location'));
DROP TABLE IF EXISTS worker_hf_token;
