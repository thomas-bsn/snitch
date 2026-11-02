-- +goose Up
ALTER TABLE worker_voice_models DROP CONSTRAINT IF EXISTS worker_voice_models_cause_code_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_cause_code_check
 CHECK (cause_code IN ('','hf_token_missing','hf_access_denied','disk_full','runtime_incompatible','inference_failed','hf_auth_required','hf_not_found','hf_network','models_volume_unwritable','hf_cache_error'));

-- +goose Down
-- Lossy on purpose: the previous constraint lacks the five finer causes, so they are all folded
-- onto runtime_incompatible and the original cause is not recoverable after a rollback.
UPDATE worker_voice_models SET cause_code='runtime_incompatible' WHERE cause_code IN ('hf_auth_required','hf_not_found','hf_network','models_volume_unwritable','hf_cache_error');
ALTER TABLE worker_voice_models DROP CONSTRAINT IF EXISTS worker_voice_models_cause_code_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_cause_code_check
 CHECK (cause_code IN ('','hf_token_missing','hf_access_denied','disk_full','runtime_incompatible','inference_failed'));
