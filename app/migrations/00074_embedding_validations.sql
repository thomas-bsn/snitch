-- +goose Up
-- Short-lived, single-use proof that an embeddings configuration (the effective URL and model) was tested
-- successfully by an administrator. Saving the configuration consumes it; only a hash of the token is stored.
CREATE TABLE embedding_validations (
 token_hash bytea PRIMARY KEY,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 url text NOT NULL,
 model text NOT NULL,
 expires_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX embedding_validations_expires ON embedding_validations (expires_at);
-- +goose Down
DROP TABLE embedding_validations;
