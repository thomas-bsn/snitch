-- +goose Up
-- A fingerprint result carries voice embeddings. They live in voice_fingerprints only,
-- so that deleting a voice profile removes every vector. The job keeps a summary
-- without vectors; result_digest (sha256 of the result as received) lets a worker
-- whose response was lost retry the identical result.
ALTER TABLE jobs ADD COLUMN result_digest text;

UPDATE jobs SET result = jsonb_build_object(
    'embedding_model', result->'embedding_model',
    'embedding_model_version', result->'embedding_model_version',
    'audio_stream_index', result->'audio_stream_index',
    'diarization_job_id', result->'diarization_job_id',
    'dimensions', result->'dimensions',
    'fingerprint_count', CASE WHEN jsonb_typeof(result->'fingerprints') = 'array' THEN jsonb_array_length(result->'fingerprints') ELSE 0 END)
WHERE step LIKE 'voice.fingerprint.%' AND result ? 'fingerprints';

-- +goose Down
ALTER TABLE jobs DROP COLUMN result_digest;
