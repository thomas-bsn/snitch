-- +goose Up
-- The audio_visible_* views (00112) stack jsonb_array_elements lateral joins; their cost estimates reach 10^8..10^9
-- even for one clip, far above jit_above_cost, so PostgreSQL JIT-compiled every reader (search, segment text,
-- embeddings) for ~1 s per query (TestFoundation/transcription_pipeline: 5 s -> 27 s). Snitch is an OLTP workload
-- of short queries: JIT never pays. Disabled at database level; new connections pick it up.
-- +goose StatementBegin
DO $$ BEGIN EXECUTE format('ALTER DATABASE %I SET jit = off', current_database()); END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN EXECUTE format('ALTER DATABASE %I RESET jit', current_database()); END $$;
-- +goose StatementEnd
