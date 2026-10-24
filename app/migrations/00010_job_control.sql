-- +goose Up
CREATE TABLE job_control (
 id boolean PRIMARY KEY DEFAULT true CHECK (id),
 paused boolean NOT NULL DEFAULT false,
 max_concurrency integer NOT NULL DEFAULT 1 CHECK (max_concurrency BETWEEN 1 AND 4),
 updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO job_control(id) VALUES (true);

-- +goose Down
DROP TABLE job_control;
