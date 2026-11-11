// Messages des codes d’erreur stables renvoyés par le serveur. Les clés doivent être
// exactement celles de errors.en.ts (le compilateur le vérifie).

import type { errorsEn } from "./errors.en";

export const errorsFr: Record<keyof typeof errorsEn, string> = {
  engine_trust_unconfigured:
    "L’opérateur n’a pas configuré les clés de signature des moteurs.",
  engine_release_untrusted:
    "La signature ou le manifeste de cette version n’est pas approuvé.",
  engine_version_exists:
    "Cette version du moteur existe déjà et ne peut pas être modifiée.",
  invalid_engine_digest: "Identifiant de version du moteur invalide.",
  engine_not_found: "Cette version du moteur n’existe pas.",
  engine_release_unavailable:
    "Cette version du moteur est indisponible ou révoquée.",
  engine_not_installed:
    "Importez une version approuvée avant d’installer ce modèle.",
  invalid_engine_order: "Cet ordre d’engine n’est pas valide.",
  invalid_engine_status: "Ce rapport d’état d’engine n’est pas valide.",
  invalid_engine_inventory: "Cet inventaire d’engines n’est pas valide.",
  engine_order_in_progress:
    "Un ordre pour cet engine est déjà en cours sur ce Worker.",
  engine_in_use:
    "Un modèle de ce Worker utilise encore cette version du moteur : elle ne peut pas être retirée.",
  engine_already_installed:
    "Cette version de l’engine est déjà installée sur ce Worker.",
  runtime_order_in_progress:
    "Un ordre est déjà en cours pour ce modèle sur ce Worker.",
  engine_release_mismatch:
    "Cette version ne correspond pas à l’engine de ce modèle.",
  engine_already_bound: "Ce modèle utilise déjà cette version du runtime.",
  invalid_runtime_order: "Cet ordre de mise à jour n’est pas valide.",
  invalid_worker_logs: "Ce lot de logs du Worker n’est pas valide.",
  invalid_worker_control: "Cette consigne de contrôle n’est pas valide.",
  invalid_worker_hardware: "Ce rapport matériel de Worker n’est pas valide.",
  platform_incompatible:
    "Cet engine ne prend pas en charge la version de Python ou la plateforme de ce Worker.",
  cuda_incompatible:
    "Cet engine exige une version de CUDA absente de ce Worker. Un runtime CPU est une release approuvée distincte.",
  insufficient_ram:
    "Ce Worker n’a pas assez de mémoire vive (RAM) pour cet engine.",
  insufficient_vram:
    "Ce Worker n’a pas assez de mémoire vidéo (VRAM) pour cet engine.",
  unsupported_engine_family:
    "Cette famille d’engine n’est pas prise en charge par Snitch : rien n’a été ordonné.",
  insufficient_disk:
    "Ce Worker n’a pas assez de disque libre, ordres en cours compris.",
  engine_not_failed:
    "Cette installation d’engine n’est pas en échec : elle ne peut être ni relancée ni nettoyée.",
  engine_order_not_cancellable:
    "Cette installation d’engine est déjà terminée : elle ne peut plus être annulée.",
  engine_order_not_found:
    "Aucune installation de cette version n’est enregistrée sur ce Worker.",

  invalid_plan_id: "Cet identifiant de plan audio n’est pas valide.",
  audio_plan_required:
    "Un clip à plusieurs pistes audio est transcrit par une analyse, pas par une tâche isolée.",
  invalid_audio_dedup: "Ce résultat de comparaison audio n’est pas valide.",
  invalid_audio_diagnostic: "Ce résultat de diagnostic audio n’est pas valide.",
  audio_plan_not_found: "Ce plan audio n’existe pas pour ce clip.",
  audio_plan_not_published:
    "Ce plan audio n’est pas la transcription publiée du clip : ses voix ne peuvent pas encore être analysées.",
  audio_plan_superseded:
    "Un autre plan audio a été publié depuis : actualisez pour voir le plan courant.",
  voice_step_in_progress:
    "Une étape vocale de cette piste est déjà en cours hors de ce plan. Attendez qu’elle se termine.",
  invalid_profile:
    "Choisissez un pseudo affiché de 1 à 100 caractères et un identifiant de 3 à 64 caractères, sans espaces au début ni à la fin.",
  invalid_password_format: "Choisissez un mot de passe de 12 à 256 caractères.",
  current_password_incorrect: "Le mot de passe actuel est incorrect.",

  avatar_too_large: "La photo doit faire moins de 5 Mo.",
  invalid_discord_bot_token: "Discord a refusé ce token de bot.",
  discord_bot_not_configured: "Le bot Discord n’est pas connecté.",
  discord_sync_running: "Une synchronisation est déjà en cours.",
  discord_rate_limited:
    "Discord limite les requêtes. Réessayez dans un instant.",
  avatar_sync_too_soon:
    "La photo vient d’être actualisée. Réessayez dans un instant.",
  invalid_avatar_source: "Cette source de photo n’existe pas.",
  no_custom_avatar: "Importez d’abord votre propre photo.",
  discord_bot_rejected: "Discord a refusé le token du bot. Reconnectez le bot.",
  discord_account_not_found: "Discord ne connaît plus ce compte.",
  invalid_avatar: "Choisissez une image JPEG ou PNG de 16 mégapixels maximum.",
  video_root_unavailable:
    "Cette racine ne peut pas être utilisée. Vérifiez qu’elle est montée et accessible en écriture.",
  folder_logo_not_found: "Aucun logo de jeu pour ce dossier.",
  invalid_folder_logo: "Choisissez une image PNG ou JPEG valide.",
  folder_logo_too_large: "Cette image dépasse 2 Mo.",
  folder_logo_version_not_found: "Cette version du logo n’existe plus.",
  video_root_unconfigured: "Configurez d’abord le dossier racine des vidéos.",
  video_root_busy:
    "Terminez ou annulez les envois en attente de publication avant de changer la racine.",
  video_root_permission:
    "Snitch ne peut pas écrire dans la racine vidéo. Vérifiez ses permissions.",
  publication_failed: "Le fichier n’a pas pu être publié dans la racine vidéo.",
  destination_changed:
    "La destination a changé pendant le transfert. Réessayez ou annulez l’envoi.",
  video_path_unavailable:
    "Ce chemin est inaccessible ou hors du dossier racine.",
  video_selection_too_large: "Sélectionnez moins de fichiers (10 000 maximum).",
  invalid_folder_name:
    "Choisissez un nom de dossier sans barre oblique ni point initial.",
  folder_request_not_found: "Cette demande de dossier n’existe pas.",
  folder_exists: "Un dossier porte déjà ce nom.",
  folder_request_pending:
    "Ce dossier attend la décision d’un administrateur. Rien ne peut y être téléversé pour l’instant.",
  folder_not_created: "Le dossier existe déjà ou ne peut pas être créé.",
  invalid_video_selection: "Sélectionnez les fichiers ou dossiers à scanner.",
  video_scan_incomplete:
    "La sélection n’a pas pu être entièrement listée. Vérifiez les droits ou sélectionnez moins de fichiers.",
  overwrite_confirmation_required:
    "Un fichier existe déjà : son remplacement nécessite votre confirmation.",
  member_protected: "Ce membre ne peut pas être modifié.",
  member_not_found: "Ce membre est introuvable.",
  invalid_role: "Ce rôle est invalide.",

  invalid_worker_machine: "La déclaration de machine du Worker est invalide.",
  invalid_log_batch: "Le lot de lignes de log est invalide.",
  invalid_log_line: "Une ligne de log du job est invalide.",
  invalid_transcription_reachability:
    "Le rapport de joignabilité du serveur de transcription est invalide.",
  account_already_linked: "Ce compte est déjà lié à une personne.",
  audio_selection_required: "Choisissez d’abord la piste audio à utiliser.",
  audio_stream_not_found: "Cette piste audio n’existe pas dans le clip.",
  audio_track_selection_required:
    "Choisissez d’abord la piste audio à utiliser.",
  batch_confirmation_required: "Confirmez le lot avant de le lancer.",
  busy: "Le serveur est occupé. Réessayez dans quelques secondes.",
  chunk_too_large: "Cette partie du fichier est trop grande.",
  clip_not_found: "Ce clip n’existe pas ou ne vous est pas visible.",
  clip_not_ready: "Ce clip n’est pas encore prêt.",
  corrections_would_be_replaced:
    "Cette action remplacerait des corrections déjà faites.",
  criteria_not_met:
    "Les conditions pour l’activer ne sont pas encore remplies.",
  csrf_rejected: "Votre session a expiré. Rechargez la page.",
  database_unavailable:
    "La base de données est indisponible. Réessayez dans un instant.",
  diarization_locked: "Les voix de ce clip sont verrouillées.",
  discord_already_linked:
    "Cette identité Discord est déjà liée. L’association existante est conservée.",
  discord_credentials_invalid:
    "L’identifiant et le secret Discord ne peuvent pas contenir uniquement des espaces.",
  discord_denied:
    "Connexion Discord annulée. Vous pouvez vous connecter avec votre compte local.",
  discord_not_configured: "La connexion Discord n’est pas configurée.",
  discord_not_linked: "Aucune identité Discord n’est liée à ce compte.",
  discord_local_login_required:
    "Définissez un mot de passe local avant de délier Discord pour garder un moyen de connexion.",
  discord_unlink_unavailable: "Discord n’a pas pu être délié. Réessayez.",
  discord_secret_required:
    "Saisissez le secret de votre application Discord pour activer la connexion.",
  discord_unavailable:
    "Discord est indisponible. Réessayez ou utilisez votre compte local.",
  duplicate_vocabulary_term: "Ce terme existe déjà.",
  embedding_rebuild_in_progress:
    "Une reconstruction de l’index est déjà en cours.",
  invalid_embedding_settings:
    "Indiquez une adresse de service et un modèle valides.",
  embedding_validation_required:
    "Testez d’abord cette configuration : seule une configuration testée peut être enregistrée.",
  embedding_validation_expired:
    "La validation a expiré. Testez à nouveau avant d’enregistrer.",
  embedding_validation_mismatch:
    "Les valeurs ont changé depuis le test. Testez à nouveau avant d’enregistrer.",
  embedding_provider_timeout:
    "Le service d’embeddings n’a pas répondu à temps. Vérifiez son adresse et sa charge, puis testez à nouveau.",
  embedding_model_not_found:
    "Le service d’embeddings ne connaît pas ce modèle. Vérifiez le nom du modèle.",
  embedding_response_invalid:
    "Le service d’embeddings a répondu, mais pas avec des vecteurs utilisables par Snitch.",
  embedding_provider_unavailable:
    "Le service d’embeddings est inaccessible ou sa réponse est invalide.",
  embeddings_unconfigured: "La recherche sémantique n’est pas configurée.",
  empty_audio_track_rule: "La règle de piste audio est vide.",
  excerpt_not_found: "Cet extrait n’existe pas.",
  excerpt_unavailable: "Cet extrait ne peut pas être produit pour le moment.",
  forbidden: "Vous n’avez pas accès à cette fonction.",
  group_changed: "Ce groupe a changé entre-temps. Rechargez puis réessayez.",
  internal_error: "Une erreur inattendue est survenue sur le serveur.",
  notification_active:
    "Cette action est encore en attente. Elle disparaît une fois traitée.",
  notification_not_found: "Cette notification n’existe plus.",
  invalid_account: "Ce compte ne convient pas pour cette action.",
  invalid_audio_stream: "Cette piste audio n’est pas valide.",
  invalid_audio_track_rule: "Cette règle de piste audio n’est pas valide.",
  invalid_analysis_scope: "Ce périmètre d’analyse n’est pas valide.",
  analysis_not_active: "Aucune analyse n’est active sur ce clip.",
  analysis_active:
    "Une analyse est en cours sur ce clip : attendez sa fin ou annulez-la avant de planifier une étape isolée.",
  analysis_not_found: "Ce clip n’a pas été analysé.",
  analysis_not_retryable:
    "Cette analyse est terminée : lancez-en une nouvelle.",
  analysis_step_not_failed: "Cette étape n’a pas échoué.",
  invalid_analysis_step: "Cette étape d’analyse n’est pas valide.",
  analysis_deletion_confirmation_required:
    "La suppression fait perdre du travail humain ou une transcription : confirmez les pertes affichées.",
  audio_selection_stale:
    "Les pistes audio ou leurs diagnostics ont changé depuis l’ouverture de ce choix. Vérifiez-le à nouveau.",
  invalid_audio_selection:
    "Ce choix de pistes audio n’est pas valide : choisissez des pistes existantes et une analyse valide.",
  audio_decision_required:
    "Décidez d’inclure ou d’écarter chaque piste ambiguë.",
  audio_force_required:
    "Une piste sans parole demande un « utiliser malgré le diagnostic » explicite.",
  audio_preview_expired:
    "Cet aperçu a expiré. Refaites l’aperçu des effets avant de lancer.",
  audio_preview_changed:
    "Le choix ou le travail concerné a changé depuis l’aperçu. Refaites l’aperçu.",
  audio_confirmation_required:
    "Confirmez les conséquences affichées avant le démarrage de l’analyse.",
  audio_selection_disabled:
    "Le choix des pistes audio n’est pas encore disponible pour ce clip.",
  analysis_already_active: "Une analyse est déjà en cours sur ce clip.",
  invalid_reanalysis: "Ce choix de réanalyse n’est pas valide.",
  reanalysis_confirmation_required:
    "Confirmez les conséquences de cette réanalyse avant son démarrage.",
  analysis_scope_conflict:
    "Une analyse d’un autre périmètre est déjà active sur ce clip.",
  invalid_clip_id: "Cet identifiant de clip n’est pas valide.",
  invalid_content_range: "Cette partie du fichier n’est pas valide.",
  invalid_credentials_format:
    "Un nom de 3 à 64 caractères et un mot de passe de 12 à 256 caractères sont requis.",
  invalid_credentials_or_pending:
    "Identifiants incorrects ou inscription en attente d’approbation.",
  invalid_clip_title:
    "Le titre doit faire de 1 à 200 caractères, sans caractère de contrôle.",
  metadata_reset_unavailable:
    "Snitch n’a pas conservé de valeur détectée à restaurer pour ce champ.",
  invalid_metadata_request:
    "Cette modification de métadonnées n’est pas valide.",
  invalid_embedding_model: "Ce modèle d’embeddings n’est pas valide.",
  invalid_filter: "Un des filtres n’est pas valide.",
  invalid_folder: "Ce nom de dossier n’est pas valide.",
  invalid_format: "Ce format n’est pas valide.",
  invalid_id: "Cet identifiant n’est pas valide.",
  invalid_job: "Cette tâche n’est pas valide.",
  invalid_job_controls: "Ces réglages de la file ne sont pas valides.",
  invalid_job_error: "Cette erreur de tâche n’est pas valide.",
  invalid_job_filter: "Un des filtres de tâches n’est pas valide.",
  invalid_job_id: "Cet identifiant de tâche n’est pas valide.",
  invalid_job_result: "Ce résultat de tâche n’est pas valide.",
  invalid_language: "Cette langue n’est pas prise en charge.",
  invalid_microphone_override: "Ce choix de micro n’est pas valide.",
  invalid_microphone_rule: "Cette règle de micro n’est pas valide.",
  invalid_mode: "Ce mode de recherche n’est pas valide.",
  public_address_unknown:
    "Snitch ne peut pas déterminer son adresse publique. Ouvrez-le par son adresse habituelle ou vérifiez le mandataire.",
  invalid_oauth_state:
    "Le retour Discord est invalide ou expiré. Recommencez la connexion.",
  invalid_page: "Ce numéro de page n’est pas valide.",
  invalid_pagination: "Cette pagination n’est pas valide.",
  invalid_person: "Cette personne n’est pas valide.",
  invalid_person_id: "Cet identifiant de personne n’est pas valide.",
  invalid_query: "Cette recherche n’est pas valide.",
  invalid_request: "La demande n’est pas valide.",
  invalid_sort: "Cet ordre de tri n’est pas valide.",
  invalid_source_id: "Cet identifiant de dossier n’est pas valide.",
  invalid_transcription_pipeline:
    "Ces réglages de transcription ne sont pas valides.",
  invalid_transcription_settings:
    "Saisissez une URL Whisper/Speaches et un modèle valides, et choisissez une langue dans la liste.",
  invalid_transcription_config_status:
    "L'état de configuration du Worker est invalide.",
  stale_transcription_config:
    "La configuration de transcription a changé. Réessayez.",
  invalid_transcription_result:
    "Ce résultat de transcription n’est pas valide.",
  invalid_turn: "Ce tour de parole n’est pas valide.",
  invalid_upload: "Cet envoi n’est pas valide.",
  invalid_video: "Ce fichier n’est pas une vidéo lisible.",
  invalid_vocabulary_id: "Cet identifiant de vocabulaire n’est pas valide.",
  invalid_vocabulary_term: "Ce terme de vocabulaire n’est pas valide.",
  invalid_visibility: 'La visibilité doit être "shared" ou "private".',
  invalid_disposition: "Cette mise de côté n’est pas valide.",
  invalid_voice_automation:
    "Ces réglages d’automatisation ne sont pas valides.",
  invalid_voice_criteria: "Ces critères ne sont pas valides.",
  invalid_voice_matching: "Ces réglages de rapprochement ne sont pas valides.",
  voice_group_unknown:
    "Aucune empreinte de ce modèle n’est en service : le modèle d’empreinte a peut-être changé. Recharger la page.",
  voice_threshold_proposal_stale:
    "Ces valeurs ne sont pas la proposition actuelle de la mesure pour ce modèle et ce mode de score : la mesure a peut-être changé. Recharger la page.",
  invalid_voice_eligibility:
    "Cette règle d’éligibilité n’est pas valide : durées entre 0,5 et 60 secondes, durée contributive au plus égale à la durée comparable, part écrêtée entre 0 et 1.",
  invalid_voice_score_mode:
    "Ce mode de score n’est pas valide : choisir la Référence cumulative ou le meilleur échantillon.",
  invalid_voice_pipeline:
    "Ces réglages du traitement vocal ne sont pas valides.",
  invalid_voice_recompute:
    "Confirmez le nombre d’empreintes indiqué par l’aperçu pour lancer le recalcul.",
  invalid_voice_result: "Ce résultat vocal n’est pas valide.",
  invalid_worker: "Ce Worker n’est pas valide.",
  invalid_worker_capabilities: "Ces capacités de Worker ne sont pas valides.",
  invalid_worker_capacity: "Cette capacité de Worker n’est pas valide.",
  invalid_worker_update_status:
    "Ce compte rendu de mise à jour n’est pas valide.",
  invalid_worker_update_mode:
    "Cette stratégie de mise à jour n’est pas valide.",
  invalid_worker_version:
    "Cette version de Worker n’est pas une version publiée (vX.Y.Z).",
  invalid_worker_models: "Ces modèles de Worker ne sont pas valides.",
  invalid_restart_request: "La demande de redémarrage n’est pas valide.",
  invalid_restart_result: "Le résultat du redémarrage n’est pas valide.",
  invalid_restart_transition:
    "Ce changement d’état du redémarrage n’est pas valide.",
  managed_supervisor_unavailable:
    "Les redémarrages supervisés ne sont pas disponibles sur ce Worker.",
  restart_already_pending: "Un redémarrage est déjà en attente pour ce Worker.",
  restart_command_expired: "Cette commande de redémarrage a expiré.",
  restart_command_not_found: "Cette commande de redémarrage n’existe plus.",
  supervisor_token_invalid: "L’identifiant du superviseur n’est pas valide.",
  worker_pulse_not_confirmed:
    "Le Worker n’a pas encore confirmé son redémarrage.",
  clip_delete_failed:
    "La vidéo ne peut pas être supprimée complètement. Vérifiez les fichiers et les permissions, puis réessayez.",
  super_administrator_required: "Seul le Super-administrateur peut faire cela.",
  invalid_hf_token:
    "Le token doit commencer par hf_ et ne contenir que des lettres et des chiffres.",
  invalid_hf_token_ack: "L’accusé de réception du Worker est invalide.",
  stale_hf_token_version: "Le token a changé entre-temps. Le Worker le relira.",
  hf_token_store_failed:
    "Le Worker n’a pas pu ranger le token dans son volume de modèles. Vérifiez que le volume est accessible en écriture.",
  invalid_transcription_language:
    "Choisissez une des langues prises en charge par Whisper, par son code (par exemple fr).",
  invalid_transcription_model: "Ce nom de modèle n’est pas valide.",
  invalid_transcription_url:
    "Cette adresse Whisper n’est pas valide. Utilisez une adresse http ou https sans identifiants.",
  invalid_transcription_endpoint:
    "Cette adresse Whisper n’est pas valide. Utilisez une adresse http ou https sans identifiants.",
  transcription_language_required: "Choisissez la langue de la bibliothèque.",
  transcription_model_required: "Indiquez le modèle de la bibliothèque.",
  transcription_profile_already_set:
    "La bibliothèque a déjà un modèle de transcription. Rechargez la page.",
  job_not_cancellable: "Cette tâche ne peut plus être annulée.",
  job_not_dead: "Seules les tâches mortes peuvent être relancées.",
  job_not_found: "Cette tâche n’existe pas.",
  json_required: "La demande doit être envoyée en JSON.",
  lease_not_owned: "Cette tâche n’est plus attribuée à ce Worker.",
  link_session_required:
    "La session à l’origine de la liaison a expiré ou changé. Recommencez depuis votre compte.",
  local_speaker_not_found: "Cette voix n’existe pas.",
  media_not_found: "Ce fichier média n’est pas disponible.",
  thumbnail_not_found: "Cette miniature n’est pas disponible.",
  method_not_allowed: "Cette action n’est pas autorisée ici.",
  microphone_track_not_diarized: "La piste du micro n’a pas encore de voix.",
  no_linked_person: "Aucune personne n’est liée à votre compte.",
  no_link_request: "Vous n’avez fait aucune demande de liaison.",
  no_pending_request: "Vous n’avez aucune demande en attente.",
  not_attributed_to_person: "Cette voix n’est attribuée à aucune personne.",
  network_failure: "La connexion a été interrompue.",
  not_found: "Cet élément n’existe pas.",
  not_pending: "Cette inscription n’attend pas d’approbation.",
  not_unknown: "Cette voix n’est plus inconnue.",
  validated_voice:
    "Cette voix est validée : son identification doit être retirée avant d’être mise de côté.",
  not_validated: "Cette voix n’a pas été validée.",
  origin_rejected: "Cette requête vient d’une adresse inattendue.",
  person_has_hidden_voices: "Cette personne a encore des voix masquées.",
  person_linked_account: "Cette personne est liée à un compte.",
  person_not_found: "Cette personne n’existe pas.",
  person_request_pending: "Cette personne a déjà une demande en attente.",
  rate_limited: "Trop de tentatives. Patientez quelques secondes.",
  request_not_found: "Cette demande de liaison n’existe pas.",
  request_not_pending: "Cette demande a déjà été décidée.",
  request_pending: "Votre compte a déjà une demande en attente.",
  same_person: "Il s’agit déjà de la même personne.",
  segment_not_found: "Ce segment n’existe pas.",
  session_required: "Connectez-vous pour continuer.",
  setup_closed_or_invalid_code:
    "Installation déjà terminée, ou code invalide ou expiré.",
  setup_required: "L’installation n’est pas encore terminée.",
  source_already_exists: "Ce dossier est déjà surveillé.",
  source_not_found: "Ce dossier n’existe pas.",
  source_path_outside_allowed_roots:
    "Ce dossier est en dehors des racines autorisées.",
  source_path_unavailable: "Ce dossier n’est pas lisible.",
  source_roots_unconfigured: "Aucun dossier n’est autorisé sur ce serveur.",
  stale_voice_result: "Ce résultat vocal est périmé.",
  transcription_required: "Ce clip doit d’abord être transcrit.",
  unauthorized: "Connectez-vous pour continuer.",
  unavailable:
    "Le service est temporairement indisponible. Réessayez dans un instant.",
  upload_conflict: "Cet envoi a changé entre-temps.",
  upload_expired: "Cet envoi a expiré. Recommencez-le.",
  upload_too_large: "Ce fichier est trop volumineux.",
  user_not_found: "Cet utilisateur n’existe pas.",
  username_unavailable: "Ce nom est déjà utilisé.",
  vocabulary_term_not_found: "Ce terme de vocabulaire n’existe pas.",
  voice_group_not_found: "Ce groupe de voix n’existe pas.",
  voice_rediarization_confirmation_required:
    "La relance de la diarisation doit être confirmée.",
  voice_locked_by_validation: "Cette voix est verrouillée par une validation.",
  voice_model_already_active: "Ce modèle est déjà le modèle actif.",
  voice_model_in_use:
    "Ce modèle est actif ou candidat : il ne peut pas être supprimé.",
  voice_model_not_found: "Ce modèle n’existe pas dans le catalogue.",
  invalid_hf_source:
    "Collez un dépôt Hugging Face sous la forme propriétaire/nom ou une adresse huggingface.co.",
  voice_model_hf_not_found:
    "Hugging Face ne connaît pas ce dépôt. Vérifiez son nom.",
  voice_model_hf_denied:
    "Hugging Face refuse l’accès à ce dépôt : il est privé ou ses conditions d’accès ne sont pas remplies.",
  voice_model_hf_unavailable:
    "Hugging Face n’a pas répondu correctement. Réessayez dans un instant.",
  voice_model_kind_mismatch: "Le dépôt déclare un autre type que celui choisi.",
  voice_model_exists: "Ce dépôt est déjà dans le catalogue pour ce type.",
  idempotency_key_reused:
    "Cette clé d’ordre a déjà servi pour un autre Worker ou un autre modèle.",
  invalid_idempotency_key: "La clé d’ordre n’est pas valide.",
  voice_model_already_installed: "Ce modèle est déjà installé sur ce Worker.",
  voice_model_order_in_progress:
    "Une installation de ce modèle est déjà en cours sur ce Worker.",
  voice_model_type_occupied:
    "Ce Worker porte déjà un autre modèle de ce type : le remplacer est une opération distincte.",
  voice_replacement_kind_unsupported:
    "Seul le modèle de diarisation peut être remplacé ici : changer le modèle d’empreinte demande une transition versionnée des empreintes enregistrées.",
  voice_replaced_model_not_ready:
    "Le modèle à remplacer doit être installé et prêt sur ce Worker.",
  voice_replacement_in_progress:
    "Un remplacement est déjà en cours pour ce type sur ce Worker, ou un téléchargement ou un test tourne encore.",
  voice_replacement_switched:
    "La bascule est déjà décidée : le remplacement ne peut plus être annulé.",
  voice_recompute_preview_stale:
    "Le nombre d’empreintes à recalculer a changé depuis l’aperçu : vérifiez-le de nouveau avant de confirmer.",
  voice_recompute_in_progress:
    "Un recalcul des empreintes est en cours : attendez sa fin ou annulez-le d’abord.",
  voice_recompute_not_found: "Aucun recalcul des empreintes n’existe.",
  voice_recompute_not_running: "Aucun recalcul des empreintes n’est en cours.",
  voice_transition_preview_stale:
    "Le nombre de références a changé depuis l’aperçu : vérifiez de nouveau l’impact avant de confirmer.",
  voice_transition_in_progress:
    "Un changement du modèle d’empreinte est déjà en cours : terminez-le ou annulez-le d’abord.",
  voice_transition_not_found:
    "Aucun changement du modèle d’empreinte n’existe.",
  voice_transition_switched:
    "Le modèle d’empreinte est déjà basculé : le changement ne peut plus être annulé.",
  voice_transition_required:
    "Le remplacement du modèle d’empreinte commence par un changement confirmé qui montre son impact.",
  invalid_voice_transition:
    "La demande de changement du modèle d’empreinte est incomplète.",
  voice_replacement_not_found: "Aucun remplacement en cours pour ce type.",
  voice_model_not_failed: "La dernière tentative de ce modèle n’a pas échoué.",
  voice_model_not_removable:
    "Ce Worker ne détient pas ce modèle dans un état où il peut être retiré.",
  worker_required: "Choisissez le Worker sur lequel retirer ce modèle.",
  voice_order_not_found: "Cet ordre n’existe pas pour ce Worker.",
  invalid_voice_order_ack: "Cet accusé de réception d’ordre n’est pas valide.",
  stale_voice_order: "Cet ordre est déjà terminé.",
  voice_model_not_ready:
    "Aucun Worker n’a ce modèle prêt : attendez la fin de sa préparation.",
  invalid_voice_model: "Ce modèle ne peut pas être choisi.",
  invalid_voice_model_status: "Cet état de modèle vocal n’est pas valide.",
  voice_trial_option_unsupported:
    "Ce moteur ne permet pas d’imposer le nombre de locuteurs.",
  invalid_voice_trial_options:
    "Le nombre de locuteurs doit être positif, dans la limite du modèle, et num_speakers ne se combine pas avec min/max.",
  voice_trial_running: "Cet essai est déjà en cours.",
  voice_trial_not_found: "Cet essai n’existe plus.",
  invalid_voice_trial_id: "Identifiant d’essai invalide.",
  no_voice_candidate: "Aucun échec à relancer pour le candidat.",
  voice_profile_deleted: "Ce profil vocal a été supprimé.",
  index_run_active:
    "Une indexation de la bibliothèque est déjà en cours ou en pause.",
  index_run_state: "Cette indexation ne peut plus faire cette action.",
  index_run_not_found: "Cette indexation n’existe plus.",
  nothing_to_index:
    "Toutes les vidéos ont déjà une fiche clip : il n’y a rien à indexer.",
  invalid_index_mode:
    "Choisissez les fiches seules, ou les fiches avec l’analyse complète.",
  person_exists:
    "Une personne porte déjà ce nom. Choisissez-la, ou confirmez qu’il s’agit de quelqu’un d’autre.",
  worker_not_found: "Ce Worker n’existe pas.",
  install_invite_used:
    "Cette invitation d’installation a déjà été utilisée. Générez une nouvelle commande dans Admin > Workers.",
  invalid_install_event: "Cet événement d’installation n’est pas valide.",
  worker_token_invalid: "Ce jeton de Worker n’est pas valide.",
  worker_token_required: "Un jeton de Worker est requis.",
  benchmark_dataset_name_taken:
    "Un jeu de données de benchmark porte déjà ce nom.",
  benchmark_dataset_used:
    "Un benchmark a utilisé ce jeu de données : il ne peut plus être supprimé.",
  benchmark_dataset_stale:
    "Le jeu de données renvoie à des clips réanalysés : créez-en un nouveau.",
  benchmark_model_invalid:
    "Choisissez des modèles d’empreinte du catalogue, activés et pris en charge.",
  benchmark_run_active:
    "Un benchmark est déjà en cours : attendez sa fin ou annulez-le d’abord.",
  benchmark_not_running: "Ce benchmark n’est plus en cours.",
  benchmark_transition_open:
    "Un changement du modèle d’empreinte est en cours : terminez-le ou annulez-le avant de lancer un benchmark.",
  benchmark_clip_duplicate:
    "Un clip est un test ou une référence, une seule fois.",
  benchmark_clip_invalid: "Un clip choisi n’existe pas ou est privé.",
  benchmark_verified_not_test:
    "Seul un clip de test peut être certifié relu en entier.",
  benchmark_dataset_unusable:
    "Ce corpus ne permet de mesurer aucune étape : ajoutez des clips de test relus ou des voix validées.",
  benchmark_stage_unavailable: "Cette étape ne peut pas encore être mesurée.",
  benchmark_stage_unsupported_by_dataset:
    "Ce corpus ne permet pas de mesurer cette étape.",
  benchmark_verified_empty:
    "Un clip certifié demande une transcription ou une voix validée.",
  benchmark_import_format:
    "Le fichier importé n'est pas un STM ou un RTTM valide : vérifiez chaque ligne.",
  benchmark_import_times:
    "Le fichier importé a des temps incohérents : un début n'est pas négatif et une fin suit son début.",
  benchmark_import_speakers:
    "Le STM et le RTTM de ce clip ne nomment aucun locuteur en commun.",
  benchmark_import_size: "Les fichiers importés dépassent 2 Mo au total.",
  benchmark_import_clip:
    "Une référence importée vise un clip de test du corpus, une seule fois.",
  benchmark_import_track:
    "La piste audio de ce clip est inconnue ou ambiguë (il lui faut une seule piste transcrite).",
  benchmark_activation_unsupported:
    "Le modèle d’empreinte change par sa propre transition, et une chaîne n’est pas un modèle.",
  benchmark_activation_confirm_required:
    "Confirmez explicitement le changement de modèle.",
  benchmark_activation_stale:
    "Le modèle en production a changé depuis l’aperçu : relisez-le avant de confirmer.",
  benchmark_activation_run_active:
    "Un benchmark est en cours : attendez sa fin ou annulez-le avant de changer le modèle.",
  benchmark_activation_run_mismatch:
    "Ce benchmark n’a pas mesuré ce modèle pour cette étape.",
};
