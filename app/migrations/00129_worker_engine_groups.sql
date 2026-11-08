-- +goose Up
-- Grouping is administration metadata, never an installation instruction.
CREATE TABLE worker_engine_groups (
    worker_id text PRIMARY KEY REFERENCES workers(id) ON DELETE CASCADE,
    group_name text NOT NULL CHECK (char_length(group_name) BETWEEN 1 AND 80)
);

-- +goose Down
DROP TABLE worker_engine_groups;
