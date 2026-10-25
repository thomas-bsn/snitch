import createClient from "openapi-fetch";
import type { paths, components } from "../api.gen";

// Voice screens live in their own files; they share the generated contract but
// not main.tsx state, so later phase 5 tickets can add screens without conflicts.
export const voiceApi = createClient<paths>();
export type ClipVoice = components["schemas"]["ClipVoice"];
export type LocalSpeaker = components["schemas"]["LocalSpeaker"];
export type Person = components["schemas"]["Person"];
export type Attribution = components["schemas"]["Attribution"];
export type AttributionEvent = components["schemas"]["AttributionEvent"];
export type DecisionLevel = components["schemas"]["DecisionLevel"];
export type AttributionCandidate =
  components["schemas"]["AttributionCandidate"];
export type VoiceMatchingSettings =
  components["schemas"]["VoiceMatchingSettings"];
export type VoiceMatchingSummary =
  components["schemas"]["VoiceMatchingSummary"];
export type VoicePipeline = components["schemas"]["VoicePipeline"];
export type CsrfHeaders = { "X-CSRF-Token": string; Origin: string };

const voiceMessages: Record<string, string> = {
  transcription_required:
    "Transcrivez d’abord ce clip : le traitement vocal ne relance jamais la transcription.",
  voice_model_access_unconfirmed:
    "Confirmez l’accès au modèle de diarisation dans le pipeline vocal avant de planifier.",
  audio_selection_required: "Choisissez d’abord la piste audio du clip.",
  already_validated:
    "Ce Locuteur local est déjà validé : la première validation fait foi.",
  attribution_changed:
    "L’attribution a changé entre-temps. Vérifiez la valeur actuelle.",
  not_validated: "Ce Locuteur local n’est pas encore validé.",
  same_person: "Choisissez une Personne différente pour corriger.",
  person_not_found: "Cette Personne n’existe plus.",
  local_speaker_not_found: "Ce Locuteur local n’existe plus.",
  invalid_person: "Le nom de la Personne est invalide (1 à 100 caractères).",
  voice_locked_by_validation:
    "Des voix de ce clip sont validées : elles ne peuvent pas être rediarisées.",
  invalid_voice_pipeline: "Configuration vocale invalide.",
  invalid_voice_matching:
    "Seuils invalides : le seuil de suggestion ne peut pas dépasser le seuil automatique.",
  voice_profile_deleted:
    "Cette Personne a supprimé son profil vocal : aucune voix ne peut lui être attribuée.",
  user_not_found: "Aucun compte de ce nom.",
  account_already_linked: "Ce compte est déjà lié à une autre Personne.",
  invalid_account: "Indiquez le nom du compte.",
  forbidden: "Action réservée à la Personne concernée ou à un administrateur.",
  csrf_rejected: "Session expirée. Rechargez la page.",
};

export function voiceChecked<T>(result: {
  data?: T;
  error?: { error: string };
}): T {
  if (result.error)
    throw new Error(
      voiceMessages[result.error.error] ?? "La demande a échoué. Réessayez.",
    );
  if (!result.data) throw new Error("Réponse du serveur absente.");
  return result.data;
}

export const levelLabels: Record<DecisionLevel, string> = {
  human: "Validé par un humain",
  automatic: "Attribution automatique",
  suggestion: "Suggestion",
  ambiguous: "Ambigu",
  unknown: "Inconnu",
};

// Glyphs back the colors so the five states stay distinguishable without color.
export const levelGlyphs: Record<DecisionLevel, string> = {
  human: "✔",
  automatic: "⚙",
  suggestion: "≈",
  ambiguous: "⇄",
  unknown: "?",
};

export function errorCode(error: unknown): string {
  return typeof error === "object" && error !== null && "error" in error
    ? String((error as { error: unknown }).error)
    : "";
}

export function messageFor(code: string): string {
  return voiceMessages[code] ?? "La demande a échoué. Réessayez.";
}
