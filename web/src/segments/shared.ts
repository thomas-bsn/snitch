// Types and messages shared by the phase 6 segment screens (text, history, speaker).

// Headers of a mutating request (the session's CSRF token and the origin).
export type CsrfHeaders = { "X-CSRF-Token": string; Origin: string };

export const maxSegmentTextLength = 4000;

// Message of each error code of the correction routes.
const segmentMessages: Record<string, string> = {
  invalid_text: "Le texte ne peut pas être vide.",
  text_too_long: `Le texte est trop long (${maxSegmentTextLength} caractères au plus).`,
  invalid_reason: "Le motif est trop long (500 caractères au plus).",
  text_unchanged: "Le texte n’a pas changé.",
  speaker_unchanged: "Le locuteur n’a pas changé.",
  invalid_local_speaker:
    "Ce locuteur n’appartient pas à ce clip. Actualisez la transcription.",
  event_not_found: "Cette entrée d’historique n’existe plus.",
  event_not_restorable: "Cette entrée ne contient pas de texte à restaurer.",
  segment_not_found: "Ce segment n’existe plus.",
  segment_conflict:
    "Ce segment a été modifié depuis votre lecture. Il a été actualisé.",
  clip_not_found: "Ce clip n’est plus visible.",
  csrf_rejected: "Session expirée. Rechargez la page.",
  transcription_changed:
    "Le clip a été retranscrit depuis votre lecture. Actualisez la transcription.",
};

export function segmentErrorMessage(code: string): string {
  return segmentMessages[code] ?? "La correction a échoué. Réessayez.";
}
