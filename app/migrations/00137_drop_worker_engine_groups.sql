-- +goose Up
-- Worker groups were administration metadata only; scoping an order to a group is removed.
DROP TABLE worker_engine_groups;

-- +goose Down
CREATE TABLE worker_engine_groups (
    worker_id text PRIMARY KEY REFERENCES workers(id) ON DELETE CASCADE,
    group_name text NOT NULL CHECK (char_length(group_name) BETWEEN 1 AND 80)
);
