// Messages for the stable error codes returned by the server (inventory taken from the
// HTTP handlers). An unknown code falls back to a generic message that shows the code.

export const errorsEn = {
  engine_trust_unconfigured:
    "The operator has not configured trusted engine signing keys.",
  engine_release_untrusted:
    "The release signature or manifest is not approved.",
  engine_version_exists: "This engine version already exists and is immutable.",
  invalid_engine_digest: "Invalid engine release identifier.",
  engine_not_found: "This engine release does not exist.",
  engine_release_unavailable: "This engine release is unavailable or revoked.",
  engine_not_installed:
    "Import an approved release before installing this model.",
  invalid_engine_order: "The engine order is not valid.",
  invalid_engine_status: "This engine status report is not valid.",
  invalid_engine_inventory: "This engine inventory report is not valid.",
  engine_order_in_progress:
    "An order for this engine is already running on this Worker.",
  engine_already_installed:
    "This engine release is already installed on this Worker.",
  invalid_worker_hardware: "This Worker hardware report is not valid.",
  platform_incompatible:
    "This engine does not support this Worker's Python version or platform.",
  cuda_incompatible:
    "This engine needs a CUDA version this Worker does not have. A CPU runtime is a separate approved release.",
  insufficient_ram:
    "This Worker does not have enough memory (RAM) for this engine.",
  insufficient_vram:
    "This Worker does not have enough video memory (VRAM) for this engine.",
  insufficient_disk:
    "This Worker does not have enough free disk, counting the orders already in progress.",

  invalid_plan_id: "This audio plan identifier is not valid.",
  audio_plan_required:
    "A clip with several audio tracks is transcribed by an analysis, not by a single task.",
  invalid_audio_dedup: "This audio comparison result is not valid.",
  invalid_audio_diagnostic: "This audio diagnostic result is not valid.",
  audio_plan_not_found: "This audio plan does not exist for this clip.",
  audio_plan_not_published:
    "This audio plan is not the published transcript of the clip: its voices cannot be analysed yet.",
  audio_plan_superseded:
    "Another audio plan has been published since: refresh to see the current one.",
  voice_step_in_progress:
    "A voice step of this track is already running outside this plan. Wait for it to finish.",
  invalid_profile:
    "Choose a display name (1–100 characters) and a username (3–64 characters, without leading or trailing spaces).",
  invalid_password_format: "Choose a password between 12 and 256 characters.",
  current_password_incorrect: "The current password is incorrect.",

  avatar_too_large: "The photo must be smaller than 5 MB.",
  invalid_discord_bot_token: "Discord refused this bot token.",
  discord_bot_not_configured: "The Discord bot is not connected.",
  discord_sync_running: "A synchronization is already running.",
  discord_rate_limited: "Discord is limiting requests. Try again in a moment.",
  avatar_sync_too_soon:
    "The picture was just refreshed. Try again in a moment.",
  invalid_avatar_source: "This photo source does not exist.",
  no_custom_avatar: "Import your own picture first.",
  discord_bot_rejected: "Discord refused the bot token. Connect the bot again.",
  discord_account_not_found: "Discord does not know this account any more.",
  invalid_avatar: "Choose a JPEG or PNG image, up to 16 megapixels.",
  video_root_unavailable:
    "This root cannot be used. Check that it is mounted and writable.",
  folder_logo_not_found: "No game logo for this folder.",
  invalid_folder_logo: "Choose a valid PNG or JPEG image.",
  folder_logo_too_large: "This image is larger than 2 MB.",
  folder_logo_version_not_found: "This logo version no longer exists.",
  video_root_unconfigured: "Configure the video root first.",
  video_root_busy:
    "Finish or cancel the uploads waiting to be published before changing the root.",
  video_root_permission:
    "Snitch cannot write to the video root. Check its permissions.",
  publication_failed: "The file could not be published to the video root.",
  destination_changed:
    "The destination changed during the transfer. Retry or cancel the upload.",
  video_path_unavailable: "This path is unavailable or outside the video root.",
  video_selection_too_large: "Select fewer files (10,000 maximum).",
  invalid_folder_name: "Choose a folder name without slashes or leading dots.",
  folder_request_not_found: "The folder request does not exist.",
  folder_exists: "A folder with this name already exists.",
  folder_request_pending:
    "This folder waits for an administrator's decision. Nothing can be uploaded into it yet.",
  folder_not_created: "The folder already exists or could not be created.",
  invalid_video_selection: "Select files or folders to scan.",
  video_scan_incomplete:
    "The selection could not be listed completely. Check permissions or select fewer files.",
  overwrite_confirmation_required:
    "An existing file needs your confirmation before replacement.",

  member_protected: "This member cannot be modified.",
  member_not_found: "The member was not found.",
  invalid_role: "This role is invalid.",

  invalid_worker_machine: "The Worker machine declaration is invalid.",
  invalid_log_batch: "The job log batch is invalid.",
  invalid_log_line: "A job log line is invalid.",
  invalid_transcription_reachability:
    "The transcription server reachability report is invalid.",
  account_already_linked: "This account is already linked to a person.",
  audio_selection_required: "Choose which audio track to use first.",
  audio_stream_not_found: "This audio track does not exist in the clip.",
  audio_track_selection_required: "Choose which audio track to use first.",
  batch_confirmation_required: "Confirm the batch before starting it.",
  busy: "The server is busy. Try again in a few seconds.",
  chunk_too_large: "This part of the file is too large.",
  clip_not_found: "This clip does not exist or is not visible to you.",
  clip_not_ready: "This clip is not ready yet.",
  corrections_would_be_replaced:
    "This action would replace corrections that were already made.",
  criteria_not_met: "The conditions to enable this are not met yet.",
  csrf_rejected: "Your session expired. Reload the page.",
  database_unavailable: "The database is unavailable. Try again in a moment.",
  diarization_locked: "Voices are locked for this clip.",
  discord_already_linked:
    "This Discord identity is already linked. The existing link is kept.",
  discord_credentials_invalid:
    "The Discord identifier and secret cannot be only spaces.",
  discord_denied:
    "Discord sign-in was cancelled. You can sign in with your local account.",
  discord_not_configured: "Discord sign-in is not configured.",
  discord_not_linked: "No Discord identity is linked to this account.",
  discord_local_login_required:
    "Set a local password before unlinking Discord so you can still sign in.",
  discord_unlink_unavailable: "Discord could not be unlinked. Try again.",
  discord_secret_required:
    "Enter your Discord application's secret to enable sign-in.",
  discord_unavailable:
    "Discord is unavailable. Try again or use your local account.",
  duplicate_vocabulary_term: "This term already exists.",
  embedding_rebuild_in_progress: "An index rebuild is already running.",
  invalid_embedding_settings: "Enter a valid service address and model.",
  embedding_validation_required:
    "Test this configuration first: only a tested configuration can be saved.",
  embedding_validation_expired:
    "The validation has expired. Test again before saving.",
  embedding_validation_mismatch:
    "The values changed since the test. Test again before saving.",
  embedding_provider_timeout:
    "The embeddings service did not answer in time. Check its address and load, then test again.",
  embedding_model_not_found:
    "The embeddings service does not know this model. Check the model name.",
  embedding_response_invalid:
    "The embeddings service answered, but not with vectors Snitch can use.",
  embedding_provider_unavailable:
    "The embeddings service could not be reached or returned an invalid response.",
  embeddings_unconfigured: "Semantic search is not configured.",
  empty_audio_track_rule: "The audio track rule is empty.",
  excerpt_not_found: "This excerpt does not exist.",
  excerpt_unavailable: "This excerpt cannot be produced right now.",
  forbidden: "You do not have access to this.",
  group_changed: "This group changed in the meantime. Reload and try again.",
  internal_error: "An unexpected error occurred on the server.",
  notification_active:
    "This action is still pending. It disappears once it is handled.",
  notification_not_found: "This notification no longer exists.",
  invalid_account: "This account is not valid for this action.",
  invalid_audio_stream: "This audio track is not valid.",
  invalid_audio_track_rule: "This audio track rule is not valid.",
  invalid_analysis_scope: "This analysis scope is not valid.",
  analysis_not_active: "No analysis is active on this clip.",
  analysis_active:
    "An analysis is running on this clip: wait for it to finish or cancel it before scheduling a single step.",
  analysis_not_found: "This clip has not been analyzed.",
  analysis_not_retryable:
    "This analysis is over: start a new analysis instead.",
  analysis_step_not_failed: "This step has not failed.",
  invalid_analysis_step: "This analysis step is not valid.",
  analysis_deletion_confirmation_required:
    "Deleting these steps loses human work or a transcript: confirm the losses shown.",
  audio_selection_stale:
    "The audio tracks or their diagnostics changed since you opened this choice. Review it again.",
  invalid_audio_selection:
    "This audio track choice is not valid: pick existing tracks and a valid analysis.",
  audio_decision_required:
    "Decide whether to include or exclude each ambiguous track.",
  audio_force_required:
    "A track without speech needs an explicit “use despite the diagnostic”.",
  audio_preview_expired:
    "This preview expired. Preview the effects again before starting.",
  audio_preview_changed:
    "The choice or the work it affects changed since the preview. Preview it again.",
  audio_confirmation_required:
    "Confirm the consequences shown before the analysis starts.",
  audio_selection_disabled:
    "Choosing audio tracks is not available for this clip yet.",
  analysis_already_active: "An analysis is already running on this clip.",
  invalid_reanalysis: "This reanalysis choice is not valid.",
  reanalysis_confirmation_required:
    "Confirm the consequences of this reanalysis before it starts.",
  analysis_scope_conflict:
    "Another analysis with a different scope is already active on this clip.",
  invalid_clip_id: "This clip identifier is not valid.",
  invalid_content_range: "This part of the file is not valid.",
  invalid_credentials_format:
    "A username of 3 to 64 characters and a password of 12 to 256 characters are required.",
  invalid_credentials_or_pending:
    "Incorrect credentials, or the sign-up is still waiting for approval.",
  invalid_clip_title:
    "The title must have 1 to 200 characters and no control character.",
  metadata_reset_unavailable:
    "Snitch did not keep a detected value to restore for this field.",
  invalid_metadata_request: "This metadata change is not valid.",
  invalid_embedding_model: "This embedding model is not valid.",
  invalid_filter: "One of the filters is not valid.",
  invalid_folder: "This folder name is not valid.",
  invalid_format: "This format is not valid.",
  invalid_id: "This identifier is not valid.",
  invalid_job: "This task is not valid.",
  invalid_job_controls: "These queue settings are not valid.",
  invalid_job_error: "This task error is not valid.",
  invalid_job_filter: "One of the task filters is not valid.",
  invalid_job_id: "This task identifier is not valid.",
  invalid_job_result: "This task result is not valid.",
  invalid_language: "This language is not supported.",
  invalid_microphone_override: "This microphone choice is not valid.",
  invalid_microphone_rule: "This microphone rule is not valid.",
  invalid_mode: "This search mode is not valid.",
  public_address_unknown:
    "Snitch cannot determine its public address. Open it through its usual address or check the reverse proxy.",
  invalid_oauth_state:
    "The Discord return is invalid or expired. Start the sign-in again.",
  invalid_page: "This page number is not valid.",
  invalid_pagination: "This pagination is not valid.",
  invalid_person: "This person is not valid.",
  invalid_person_id: "This person identifier is not valid.",
  invalid_query: "This search is not valid.",
  invalid_request: "The request is not valid.",
  invalid_sort: "This sort order is not valid.",
  invalid_source_id: "This folder identifier is not valid.",
  invalid_transcription_pipeline: "These transcription settings are not valid.",
  invalid_transcription_settings:
    "Enter a valid Whisper/Speaches URL and model, and choose a language from the list.",
  invalid_transcription_config_status:
    "The Worker configuration status is invalid.",
  stale_transcription_config:
    "The transcription configuration changed. Try again.",
  invalid_transcription_result: "This transcription result is not valid.",
  invalid_turn: "This speaker turn is not valid.",
  invalid_upload: "This upload is not valid.",
  invalid_video: "This file is not a readable video.",
  invalid_vocabulary_id: "This vocabulary identifier is not valid.",
  invalid_vocabulary_term: "This vocabulary term is not valid.",
  invalid_visibility: 'Visibility must be "shared" or "private".',
  invalid_disposition: "This disposition is not valid.",
  invalid_voice_automation: "These automation settings are not valid.",
  invalid_voice_criteria: "These criteria are not valid.",
  invalid_voice_matching: "These matching settings are not valid.",
  invalid_voice_pipeline: "These voice processing settings are not valid.",
  invalid_voice_result: "This voice result is not valid.",
  invalid_worker: "This worker is not valid.",
  invalid_worker_capabilities: "These worker capabilities are not valid.",
  invalid_worker_capacity: "This worker capacity is not valid.",
  invalid_worker_update_status: "This update report is not valid.",
  invalid_worker_update_mode: "This update strategy is not valid.",
  invalid_worker_version:
    "This worker version is not a released version (vX.Y.Z).",
  invalid_revoke_reason:
    "The reason must be 200 characters or fewer, on a single line.",
  invalid_worker_models: "These worker models are not valid.",
  invalid_restart_request: "The restart request is not valid.",
  invalid_restart_result: "The restart result is not valid.",
  invalid_restart_transition: "This restart state change is not valid.",
  managed_supervisor_unavailable:
    "Managed restarts are unavailable on this worker.",
  restart_already_pending: "A restart is already pending for this worker.",
  restart_command_expired: "This restart command has expired.",
  restart_command_not_found: "This restart command no longer exists.",
  supervisor_token_invalid: "The supervisor credential is not valid.",
  worker_pulse_not_confirmed: "The worker has not confirmed its restart yet.",
  clip_delete_failed:
    "The video could not be deleted completely. Check its files and permissions, then retry.",
  super_administrator_required: "Only the Super-administrator can do this.",
  invalid_hf_token:
    "The token must start with hf_ and contain only letters and digits.",
  invalid_hf_token_ack: "The Worker acknowledgement is invalid.",
  stale_hf_token_version:
    "The token changed in the meantime. The Worker will read it again.",
  hf_token_store_failed:
    "The Worker could not store the token in its models volume. Check that the volume is writable.",
  invalid_transcription_language:
    "Choose one of the languages Whisper supports, by its code (for example fr).",
  invalid_transcription_model: "This model name is not valid.",
  invalid_transcription_url:
    "This Whisper address is not valid. Use an http or https address without credentials.",
  invalid_transcription_endpoint:
    "This Whisper address is not valid. Use an http or https address without credentials.",
  transcription_language_required: "Choose the language of the library.",
  transcription_model_required: "Enter the model of the library.",
  transcription_profile_already_set:
    "The library already has a transcription model. Reload the page.",
  job_not_cancellable: "This task can no longer be cancelled.",
  job_not_dead: "Only dead tasks can be relaunched.",
  job_not_found: "This task does not exist.",
  json_required: "The request must be sent as JSON.",
  lease_not_owned: "This task is no longer assigned to this worker.",
  link_session_required:
    "The session that started the link expired or changed. Start again from your account.",
  local_speaker_not_found: "This voice does not exist.",
  media_not_found: "This media file is not available.",
  thumbnail_not_found: "This thumbnail is not available.",
  method_not_allowed: "This action is not allowed here.",
  microphone_track_not_diarized: "The microphone track has no voices yet.",
  no_linked_person: "No person is linked to your account.",
  no_link_request: "You have not requested a link.",
  no_pending_request: "You have no pending request.",
  not_attributed_to_person: "This voice is not attributed to a person.",
  network_failure: "The connection was interrupted.",
  not_found: "This does not exist.",
  not_pending: "This sign-up is not waiting for approval.",
  not_unknown: "This voice is no longer unknown.",
  validated_voice:
    "This voice is validated: its identification must be withdrawn before it can be set aside.",
  not_validated: "This voice has not been validated.",
  origin_rejected: "This request came from an unexpected address.",
  person_has_hidden_voices: "This person still has hidden voices.",
  person_linked_account: "This person is linked to an account.",
  person_not_found: "This person does not exist.",
  person_request_pending: "This person already has a pending request.",
  rate_limited: "Too many attempts. Wait a few seconds.",
  request_not_found: "This link request does not exist.",
  request_not_pending: "This request has already been decided.",
  request_pending: "Your account already has a pending request.",
  same_person: "These are already the same person.",
  segment_not_found: "This segment does not exist.",
  session_required: "Sign in to continue.",
  setup_closed_or_invalid_code:
    "Installation is already finished, or the code is invalid or expired.",
  setup_required: "Installation is not finished yet.",
  source_already_exists: "This folder is already watched.",
  source_not_found: "This folder does not exist.",
  source_path_outside_allowed_roots:
    "This folder is outside the allowed roots.",
  source_path_unavailable: "This folder is not readable.",
  source_roots_unconfigured: "No folder is allowed on this server.",
  stale_voice_result: "This voice result is out of date.",
  transcription_required: "This clip has to be transcribed first.",
  unauthorized: "Sign in to continue.",
  unavailable: "The service is temporarily unavailable. Try again in a moment.",
  upload_conflict: "This upload changed in the meantime.",
  upload_expired: "This upload expired. Start it again.",
  upload_too_large: "This file is too large.",
  user_not_found: "This user does not exist.",
  username_unavailable: "This username is already taken.",
  vocabulary_term_not_found: "This vocabulary term does not exist.",
  voice_group_not_found: "This voice group does not exist.",
  voice_rediarization_confirmation_required:
    "Relaunching the diarization must be confirmed.",
  voice_locked_by_validation: "This voice is locked by a validation.",
  voice_model_already_active: "This model is already the active model.",
  voice_model_in_use:
    "This model is active or a candidate, so it cannot be deleted.",
  voice_model_not_found: "This model is not in the catalogue.",
  invalid_hf_source:
    "Paste a Hugging Face repository as owner/name or as a huggingface.co address.",
  voice_model_hf_not_found:
    "Hugging Face does not know this repository. Check its name.",
  voice_model_hf_denied:
    "Hugging Face refuses access to this repository: it is private or its access conditions are not met.",
  voice_model_hf_unavailable:
    "Hugging Face did not answer correctly. Try again in a moment.",
  voice_model_kind_mismatch:
    "The repository declares another type than the one chosen.",
  voice_model_exists:
    "This repository is already in the catalogue for this type.",
  idempotency_key_reused:
    "This order key was already used for another Worker or model.",
  invalid_idempotency_key: "The order key is not valid.",
  voice_model_already_installed:
    "This model is already installed on this Worker.",
  voice_model_order_in_progress:
    "An installation of this model is already under way on this Worker.",
  voice_model_type_occupied:
    "This Worker already holds another model of this type: replacing it is a separate operation.",
  voice_replacement_kind_unsupported:
    "Only the diarization model can be replaced here: changing the fingerprint model needs a versioned transition of the stored fingerprints.",
  voice_replaced_model_not_ready:
    "The model to replace must be installed and ready on this Worker.",
  voice_replacement_in_progress:
    "A replacement is already under way for this type on this Worker, or a download or test is still running.",
  voice_replacement_switched:
    "The switch is already decided: the replacement can no longer be cancelled.",
  voice_transition_preview_stale:
    "The number of references changed since the preview: check the impact again before confirming.",
  voice_transition_in_progress:
    "A fingerprint model change is already under way: finish or cancel it first.",
  voice_transition_not_found: "No fingerprint model change exists.",
  voice_transition_switched:
    "The fingerprint model is already switched: the change can no longer be cancelled.",
  voice_transition_required:
    "Replacing the fingerprint model starts with a confirmed change that shows its impact.",
  invalid_voice_transition:
    "The request to change the fingerprint model is incomplete.",
  voice_replacement_not_found: "No replacement is under way for this type.",
  voice_model_not_failed: "The last attempt of this model did not fail.",
  voice_model_not_removable:
    "This Worker does not hold this model in a state that can be removed.",
  worker_required: "Choose the Worker on which to remove this model.",
  voice_order_not_found: "This order does not exist for this Worker.",
  invalid_voice_order_ack: "This acknowledgement of an order is not valid.",
  stale_voice_order: "This order is already finished.",
  voice_model_not_ready:
    "No Worker has this model ready: wait for its preparation to finish.",
  invalid_voice_model: "This model cannot be chosen.",
  invalid_voice_model_status: "This voice model status is not valid.",
  voice_trial_option_unsupported:
    "This engine cannot be given a number of speakers.",
  invalid_voice_trial_options:
    "The number of speakers must be positive and within the model's limit, and num_speakers cannot be combined with min/max.",
  voice_trial_running: "This trial is already running.",
  voice_trial_not_found: "This trial no longer exists.",
  invalid_voice_trial_id: "Invalid trial identifier.",
  no_voice_candidate: "There is no failure to retry for the candidate.",
  voice_profile_deleted: "This voice profile was deleted.",
  index_run_active:
    "An indexation of the library is already running or paused.",
  index_run_state: "This indexation can no longer take that action.",
  index_run_not_found: "This indexation no longer exists.",
  nothing_to_index:
    "Every video already has a clip card: there is nothing to index.",
  invalid_index_mode: "Choose cards only, or cards and full analysis.",
  person_exists:
    "A person with this name already exists. Pick them, or confirm it is someone else.",
  worker_not_found: "This worker does not exist.",
  worker_not_revoked:
    "Only a deleted (revoked) Worker can be removed from the list.",
  worker_cleanup_pending:
    "The Worker's machine has not confirmed the revocation yet: it still needs this entry to clean up after itself.",
  install_invite_used:
    "The installation invitation was already used. Generate a new installation command in Admin > Workers.",
  invalid_install_event: "This installation event is not valid.",
  worker_token_invalid: "This worker token is not valid.",
  worker_token_required: "A worker token is required.",
};
