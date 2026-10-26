import React, { useState } from "react";
import createClient from "openapi-fetch";
import type { paths, components } from "../api.gen";
import { segmentErrorMessage, type CsrfHeaders } from "./shared";
import "./segments.css";

// Text of one transcript segment with its correction (phase 6).
const segmentApi = createClient<paths>();
type TranscriptSegment = components["schemas"]["TranscriptSegment"];
type SegmentConflictState = {
  version: number;
  text: string;
  author: string;
  correctedAt: string;
};

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
  // Set when another correction won the race: the draft is kept, the version it must
  // now replace is the one returned by the server.
  const [conflict, setConflict] = useState<SegmentConflictState | null>(null);
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
            expected_version:
              conflict?.version ?? segment.correction_version ?? 0,
            reason: reason.trim() || undefined,
            transcription_job_id: jobId,
          },
        },
      );
      if (result.error) {
        const failure = result.error as {
          error: string;
          current?: components["schemas"]["SegmentCorrection"];
        };
        if (failure.error === "segment_conflict" && failure.current) {
          setConflict({
            version: failure.current.version,
            text: failure.current.text,
            author: failure.current.author?.username ?? "un autre utilisateur",
            correctedAt: formatDate(failure.current.corrected_at ?? undefined),
          });
          await onChanged();
          return;
        }
        setMessage(segmentErrorMessage(failure.error));
        return;
      }
      setEditing(false);
      setConflict(null);
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
      <span className="segment-body">
        {segment.corrected || segment.words.length === 0
          ? text
          : segment.words.map((word, wordIndex) => (
              <span key={`${word.start}-${wordIndex}`}>{word.word} </span>
            ))}
      </span>
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
            setConflict(null);
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
          {conflict && (
            <span
              className="segment-conflict"
              role="group"
              aria-label={`Conflit sur le segment ${label}`}
            >
              <span>
                Ce segment a été modifié par {conflict.author}
                {conflict.correctedAt ? ` le ${conflict.correctedAt}` : ""}{" "}
                depuis votre lecture. Votre modification n’a pas été
                enregistrée.
              </span>
              <span>Version actuelle : « {conflict.text} »</span>
              <span className="segment-edit-actions">
                <button
                  type="button"
                  aria-label={`Reprendre ma modification du segment ${label}`}
                  disabled={busy}
                  onClick={() => setConflict(null)}
                >
                  Reprendre ma modification
                </button>
                <button
                  type="button"
                  className="secondary"
                  aria-label={`Abandonner ma modification du segment ${label}`}
                  disabled={busy}
                  onClick={() => {
                    setEditing(false);
                    setConflict(null);
                    setMessage("");
                  }}
                >
                  Abandonner ma modification
                </button>
              </span>
            </span>
          )}
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
                setConflict(null);
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
