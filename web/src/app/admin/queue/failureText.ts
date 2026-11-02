/** Wording of the failure causes and advised actions of the closed catalogue (#122). */
type Pair = [fr: string, en: string];

const causes: Record<string, Pair> = {
  worker_failed: ["Échec du Worker", "Worker failure"],
  provider_unavailable: [
    "Service de transcription indisponible",
    "Transcription service unavailable",
  ],
  invalid_input: ["Média invalide", "Invalid media"],
  audio_preparation_failed: [
    "Préparation de l’audio impossible",
    "Audio preparation failed",
  ],
  lease_expired: ["Bail expiré", "Lease expired"],
  max_attempts_exhausted: ["Essais épuisés", "Attempts exhausted"],
  cancelled_by_admin: ["Annulé par un administrateur", "Cancelled by an admin"],
  input_version_superseded: [
    "Remplacé par une version plus récente",
    "Superseded by a newer version",
  ],
  worker_revoked: ["Worker révoqué", "Worker revoked"],
  whisper_network: ["Réseau Whisper injoignable", "Whisper unreachable"],
  whisper_timeout: ["Délai Whisper dépassé", "Whisper timed out"],
  whisper_http_error: ["Erreur HTTP de Whisper", "Whisper HTTP error"],
  whisper_model_missing: [
    "Modèle de transcription manquant",
    "Transcription model missing",
  ],
  whisper_invalid_response: [
    "Réponse Whisper invalide",
    "Invalid Whisper response",
  ],
  pyannote_model_missing: ["Modèle vocal manquant", "Voice model missing"],
  pyannote_access_denied: [
    "Accès au modèle vocal refusé",
    "Voice model access denied",
  ],
  gpu_unavailable: ["GPU indisponible", "GPU unavailable"],
  gpu_out_of_memory: ["GPU saturé", "GPU out of memory"],
  diarization_inference_failed: [
    "Échec de la diarisation",
    "Diarization failed",
  ],
  hf_auth_required: [
    "Authentification Hugging Face requise",
    "Hugging Face authentication required",
  ],
  hf_not_found: ["Modèle Hugging Face introuvable", "Hugging Face not found"],
  hf_network: ["Réseau Hugging Face injoignable", "Hugging Face unreachable"],
  models_volume_unwritable: [
    "Volume des modèles non inscriptible",
    "Models volume not writable",
  ],
  hf_cache_error: ["Cache de modèles en erreur", "Model cache error"],
};

const actions: Record<string, Pair> = {
  check_worker_logs: [
    "Consulter les logs du Worker.",
    "Check the Worker logs.",
  ],
  check_transcription_service: [
    "Vérifier le service de transcription et le réseau.",
    "Check the transcription service and the network.",
  ],
  check_clip_media: ["Vérifier le média du clip.", "Check the clip media."],
  check_worker_connection: [
    "Vérifier la connexion du Worker au serveur.",
    "Check the Worker connection to the server.",
  ],
  relaunch_job: [
    "Relancer la tâche avec le bouton de relance.",
    "Relaunch the task with the relaunch button.",
  ],
  none: [
    "Aucune action : une version plus récente la remplace.",
    "No action: a newer version replaces it.",
  ],
  install_transcription_model: [
    "Installer le modèle de transcription.",
    "Install the transcription model.",
  ],
  install_voice_model: [
    "Installer le modèle vocal sur le Worker.",
    "Install the voice model on the Worker.",
  ],
  check_huggingface_token: [
    "Vérifier le jeton Hugging Face et les conditions du modèle.",
    "Check the Hugging Face token and the model terms.",
  ],
  check_gpu: [
    "Vérifier le GPU (pilote, WSL2, mémoire).",
    "Check the GPU (driver, WSL2, memory).",
  ],
  check_voice_model_repository: [
    "Vérifier le dépôt du modèle.",
    "Check the model repository.",
  ],
  check_worker_network: [
    "Vérifier le réseau du Worker.",
    "Check the Worker network.",
  ],
  check_models_volume: [
    "Vérifier les droits et le contenu du volume des modèles.",
    "Check the permissions and content of the models volume.",
  ],
  start_worker: [
    "Aucun Worker récent ne peut prendre cette étape : installer ou démarrer un Worker.",
    "No recent Worker can take this step: install or start a Worker.",
  ],
  reschedule: [
    "Replanifier le clip depuis le formulaire de planification.",
    "Schedule the clip again from the scheduling form.",
  ],
};

const pick = (pair: Pair | undefined, fallback: string, language: string) =>
  pair ? (language === "fr" ? pair[0] : pair[1]) : fallback;

export const causeText = (code: string, language: string) =>
  pick(causes[code], code, language);
export const actionText = (code: string, language: string) =>
  pick(actions[code], code, language);
