import createClient from "openapi-fetch";
import type { paths, components } from "../api.gen";

// Voice screens live in their own files; they share the generated contract but
// not main.tsx state, so later phase 5 tickets can add screens without conflicts.
export const voiceApi = createClient<paths>();
export type ClipVoice = components["schemas"]["ClipVoice"];
export type LocalSpeaker = components["schemas"]["LocalSpeaker"];
export type VoicePipeline = components["schemas"]["VoicePipeline"];
export type CsrfHeaders = { "X-CSRF-Token": string; Origin: string };

const voiceMessages: Record<string, string> = {
  transcription_required:
    "Transcrivez d’abord ce clip : le traitement vocal ne relance jamais la transcription.",
  voice_model_access_unconfirmed:
    "Confirmez l’accès au modèle de diarisation dans le pipeline vocal avant de planifier.",
  audio_selection_required: "Choisissez d’abord la piste audio du clip.",
  invalid_voice_pipeline: "Configuration vocale invalide.",
  forbidden: "Accès réservé aux administrateurs.",
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
