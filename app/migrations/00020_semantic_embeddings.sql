-- +goose Up
CREATE TABLE embedding_profiles (
 id bigserial PRIMARY KEY,
 model text NOT NULL,
 dimensions integer CHECK (dimensions IS NULL OR dimensions BETWEEN 1 AND 4000),
 state text NOT NULL CHECK (state IN ('active','building','ready','failed','retired')),
 error text,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE embedding_index_state (
 id integer PRIMARY KEY CHECK (id=1),
 active_profile_id bigint REFERENCES embedding_profiles(id),
 candidate_profile_id bigint REFERENCES embedding_profiles(id),
 updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO embedding_index_state(id) VALUES (1);
CREATE TABLE embedding_tasks (
 id bigserial PRIMARY KEY,
 profile_id bigint NOT NULL REFERENCES embedding_profiles(id) ON DELETE CASCADE,
 transcription_job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
 state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','leased','done','failed')),
 attempts integer NOT NULL DEFAULT 0,
 error text,
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(profile_id, transcription_job_id)
);
ALTER TABLE embedding_tasks ADD COLUMN lease_expires_at timestamptz;
CREATE INDEX embedding_tasks_pending ON embedding_tasks(profile_id,id) WHERE state IN ('pending','leased');
CREATE TABLE transcript_embeddings (
 profile_id bigint NOT NULL REFERENCES embedding_profiles(id) ON DELETE CASCADE,
 transcription_job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
 clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
 segment_index integer NOT NULL,
 segment_start double precision NOT NULL,
 segment_end double precision NOT NULL,
 text text NOT NULL,
 embedding vector NOT NULL,
 PRIMARY KEY(profile_id, transcription_job_id, segment_index)
);
-- +goose Down
DROP TABLE transcript_embeddings;
DROP TABLE embedding_tasks;
DROP TABLE embedding_index_state;
DROP TABLE embedding_profiles;
