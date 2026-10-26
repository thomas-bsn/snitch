import React, { useState } from "react";
import createClient from "openapi-fetch";
import type { paths, components } from "../api.gen";
import "./segments.css";

// Text of one transcript segment with its correction (phase 6). It lives in its own
// file so later phase 6 screens (history, speaker) extend it without touching main.tsx.
const segmentApi = createClient<paths>();
type TranscriptSegment = components["schemas"]["TranscriptSegment"];
type CsrfHeaders = { "X-CSRF-Token": string; Origin: string };

export const maxSegmentTextLength = 4000;

const segmentMessages: Record<string, string> = {
  invalid_text: "Le texte ne peut pas être vide.",
  text_too_long: `Le texte est trop long (${maxSegmentTextLength} caractères au plus).`,
  invalid_reason: "Le motif est trop long (500 caractères au plus).",
  text_unchanged: "Le texte n’a pas changé.",
  segment_not_found: "Ce segment n’existe plus.",
  clip_not_found: "Ce clip n’est plus visible.",
  csrf_rejected: "Session expirée. Rechargez la page.",
  transcription_changed:
    "Le clip a été retranscrit depuis votre lecture. Actualisez la transcription.",
};

export function segmentErrorMessage(code: string): string {
  return segmentMessages[code] ?? "La correction a échoué. Réessayez.";
}

function formatDate(value: string | undefined): string {
  if (!value) return "";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString("fr-FR");
}

export function SegmentText({
  clipId,
  jobId,
  segment,
  csrf,
  onChanged,
}: {
  clipId: number;
  jobId: number | undefined;
  segment: TranscriptSegment;
  csrf: CsrfHeaders;
  onChanged: () => void | Promise<void>;
}): React.ReactElement {
  const index = segment.index;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [showOriginal, setShowOriginal] = useState(false);
  const [message, setMessage] = useState("");
  const text = segment.text;
  const label = index === undefined ? "" : String(index + 1);

  async function save() {
    if (index === undefined) return;
    setBusy(true);
    setMessage("");
    try {
      const result = await segmentApi.PUT(
        "/api/clips/{id}/segments/{index}/text",
        {
          params: { path: { id: clipId, index }, header: csrf },
          body: {
            text: draft,
            expected_version: segment.correction_version ?? 0,
            reason: reason.trim() || undefined,
            transcription_job_id: jobId,
          },
        },
      );
      if (result.error) {
        const failure = result.error as {
          error: string;
          current?: { text: string; author?: { username: string } | null };
        };
        setMessage(
          failure.error === "segment_conflict" && failure.current
            ? `Ce segment a été modifié par ${failure.current.author?.username ?? "un autre utilisateur"} depuis votre lecture. Version actuelle : « ${failure.current.text} ». Actualisez la transcription avant de recommencer.`
            : segmentErrorMessage(failure.error),
        );
        if (failure.error === "segment_conflict") await onChanged();
        return;
      }
      setEditing(false);
      setReason("");
      setMessage("Correction enregistrée.");
      await onChanged();
    } catch {
      setMessage("La correction a échoué. Réessayez.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <span className="segment-text">
      <span className="segment-body">{text}</span>
      {segment.corrected && (
        <span className="segment-corrected">
          {" "}
          <span className="segment-correction-badge">Corrigé</span> par{" "}
          {segment.corrected_by} le {formatDate(segment.corrected_at)}
        </span>
      )}
      {segment.corrected && (
        <button
          type="button"
          className="secondary segment-action"
          aria-expanded={showOriginal}
          aria-label={`Texte d’origine du segment ${label}`}
          onClick={() => setShowOriginal(!showOriginal)}
        >
          {showOriginal ? "Masquer l’original" : "Voir l’original"}
        </button>
      )}
      {segment.corrected && showOriginal && (
        <span className="segment-original">
          Texte d’origine : {segment.original_text}
        </span>
      )}
      {!editing && index !== undefined && (
        <button
          type="button"
          className="secondary segment-action"
          aria-label={`Modifier le texte du segment ${label}`}
          onClick={() => {
            setDraft(text);
            setMessage("");
            setEditing(true);
          }}
        >
          Modifier
        </button>
      )}
      {editing && (
        <form
          className="segment-edit"
          onSubmit={(event) => {
            event.preventDefault();
            void save();
          }}
        >
          <label>
            Texte du segment {label}
            <textarea
              value={draft}
              rows={3}
              onChange={(event) => setDraft(event.target.value)}
            />
          </label>
          <label>
            Motif du segment {label} (facultatif)
            <input
              value={reason}
              maxLength={500}
              onChange={(event) => setReason(event.target.value)}
            />
          </label>
          <span className="segment-edit-actions">
            <button
              type="submit"
              aria-label={`Enregistrer le texte du segment ${label}`}
              disabled={busy || draft.trim() === ""}
            >
              Enregistrer
            </button>
            <button
              type="button"
              className="secondary"
              aria-label={`Abandonner la modification du segment ${label}`}
              disabled={busy}
              onClick={() => {
                setEditing(false);
                setMessage("");
              }}
            >
              Annuler
            </button>
          </span>
        </form>
      )}
      <span className="segment-message" aria-live="polite">
        {message}
      </span>
    </span>
  );
}
