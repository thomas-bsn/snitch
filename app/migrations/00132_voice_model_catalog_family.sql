-- +goose Up
-- A catalogue model may ask for an on-demand engine family (ADR 0056). The column is empty for every existing
-- model, which keeps its contract: embedded adapters (pyannote, NeMo Sortformer, Silero) need no engine and the
-- signed adapters (nemo-titanet, speechbrain-ecapa) need an approved release. The engine column keeps naming the
-- adapter that loads the weights; a model whose family is not in the table of the server (the table lives in Go and grows without a migration)
-- is refused before any order.
ALTER TABLE voice_model_catalog ADD COLUMN family text NOT NULL DEFAULT '' CHECK (family ~ '^[a-z0-9-]{0,32}$');

-- +goose Down
ALTER TABLE voice_model_catalog DROP COLUMN family;
