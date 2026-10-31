-- +goose Up
CREATE TABLE clip_metadata_settings (
 id boolean PRIMARY KEY DEFAULT true CHECK (id),
 country text NOT NULL DEFAULT '',
 timezone text NOT NULL DEFAULT ''
);
INSERT INTO clip_metadata_settings (id) VALUES (true);

ALTER TABLE setup_steps DROP CONSTRAINT setup_steps_step_check;
ALTER TABLE setup_steps ADD CONSTRAINT setup_steps_step_check CHECK (step IN ('discord','semantic_search','worker','import','location'));

ALTER TABLE source_files ADD COLUMN first_seen_at timestamptz NOT NULL DEFAULT now();
UPDATE source_files SET first_seen_at=stable_since;

ALTER TABLE clips ADD COLUMN game text;
ALTER TABLE clips ADD COLUMN game_origin text;
ALTER TABLE clips ADD COLUMN capture_time timestamptz;
ALTER TABLE clips ADD COLUMN capture_time_origin text;
ALTER TABLE clips ADD COLUMN game_edited boolean NOT NULL DEFAULT false;
ALTER TABLE clips ADD COLUMN capture_time_edited boolean NOT NULL DEFAULT false;
UPDATE clips SET capture_time=recorded_at, capture_time_origin='container' WHERE NOT recorded_at_estimated;

-- +goose Down
ALTER TABLE clips DROP COLUMN capture_time_edited, DROP COLUMN game_edited,
 DROP COLUMN capture_time_origin, DROP COLUMN capture_time, DROP COLUMN game_origin, DROP COLUMN game;
DELETE FROM setup_steps WHERE step='location';
ALTER TABLE setup_steps DROP CONSTRAINT setup_steps_step_check;
ALTER TABLE setup_steps ADD CONSTRAINT setup_steps_step_check CHECK (step IN ('discord','semantic_search','worker','import'));
ALTER TABLE source_files DROP COLUMN first_seen_at;
DROP TABLE clip_metadata_settings;
