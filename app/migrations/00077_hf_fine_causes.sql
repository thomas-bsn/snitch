-- +goose Up
-- Fine Hugging Face causes of the voice-model download (#203): token refused, gated access refused,
-- repository or revision not found. Installation events and voice-model states accept them.
ALTER TABLE worker_install_events DROP CONSTRAINT IF EXISTS worker_install_events_cause_code_check;
ALTER TABLE worker_install_events ADD CONSTRAINT worker_install_events_cause_code_check
 CHECK (cause_code IN ('','unsupported_platform','docker_unavailable','registration_rejected','network_unreachable','image_pull_failed','volume_not_writable','hf_cache_not_writable','gpu_unavailable','service_start_failed','hf_access_denied','unknown','hf_token_invalid','hf_gated_denied','hf_repo_invalid'));
ALTER TABLE worker_voice_models DROP CONSTRAINT IF EXISTS worker_voice_models_cause_code_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_cause_code_check
 CHECK (cause_code IN ('','hf_token_missing','hf_access_denied','disk_full','runtime_incompatible','inference_failed','hf_auth_required','hf_not_found','hf_network','models_volume_unwritable','hf_cache_error','hf_token_invalid','hf_gated_denied','hf_repo_invalid'));

-- +goose Down
-- Lossy on purpose: the three fine causes fold onto the coarser codes they refine.
UPDATE worker_install_events SET cause_code='hf_access_denied' WHERE cause_code IN ('hf_token_invalid','hf_gated_denied');
UPDATE worker_install_events SET cause_code='unknown' WHERE cause_code='hf_repo_invalid';
ALTER TABLE worker_install_events DROP CONSTRAINT IF EXISTS worker_install_events_cause_code_check;
ALTER TABLE worker_install_events ADD CONSTRAINT worker_install_events_cause_code_check
 CHECK (cause_code IN ('','unsupported_platform','docker_unavailable','registration_rejected','network_unreachable','image_pull_failed','volume_not_writable','hf_cache_not_writable','gpu_unavailable','service_start_failed','hf_access_denied','unknown'));
UPDATE worker_voice_models SET cause_code='hf_access_denied' WHERE cause_code IN ('hf_token_invalid','hf_gated_denied');
UPDATE worker_voice_models SET cause_code='hf_not_found' WHERE cause_code='hf_repo_invalid';
ALTER TABLE worker_voice_models DROP CONSTRAINT IF EXISTS worker_voice_models_cause_code_check;
ALTER TABLE worker_voice_models ADD CONSTRAINT worker_voice_models_cause_code_check
 CHECK (cause_code IN ('','hf_token_missing','hf_access_denied','disk_full','runtime_incompatible','inference_failed','hf_auth_required','hf_not_found','hf_network','models_volume_unwritable','hf_cache_error'));
