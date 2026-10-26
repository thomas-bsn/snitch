import React, { useCallback, useEffect, useState } from "react";
import createClient from "openapi-fetch";
import type { paths, components } from "../api.gen";
import { segmentErrorMessage, type CsrfHeaders } from "./shared";
import "./segment-history.css";

// History of one transcript segment and restoration of the original text or of a past
// version (phase 6). A restoration is itself a correction: it appends a history entry.
const historyApi = createClient<paths>();
type SegmentHistoryData = components["schemas"]["SegmentHistory"];
type HistoryEntry = components["schemas"]["SegmentHistoryEntry"];

const kindLabels: Record<HistoryEntry["kind"], string> = {
  text: "Correction",
  speaker: "Locuteur",
  restore: "Restauration",
};

function formatDate(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString("fr-FR");
}

export function SegmentHistory({
  clipId,
  jobId,
  index,
  version,
  csrf,
  onChanged,
}: {
  clipId: number;
  jobId: number | undefined;
  index: number | undefined;
  version: number;
  csrf: CsrfHeaders;
  onChanged: () => void | Promise<void>;
}): React.ReactElement | null {
  const [open, setOpen] = useState(false);
  const [data, setData] = useState<SegmentHistoryData | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const label = index === undefined ? "" : String(index + 1);

  const load = useCallback(async () => {
    if (index === undefined) return;
    try {
      const result = await historyApi.GET(
        "/api/clips/{id}/segments/{index}/history",
        { params: { path: { id: clipId, index } } },
      );
      if (result.data) setData(result.data);
      else
        setMessage(
          segmentErrorMessage((result.error as { error: string }).error),
        );
    } catch {
      setMessage("L’historique n’a pas pu être chargé.");
    }
  }, [clipId, index]);

  // Reload when opened and whenever the segment version moves (an edit made elsewhere).
  useEffect(() => {
    if (open) void load();
  }, [open, load, version]);

  if (index === undefined) return null;

  async function restore(
    target: { original: true } | { event_id: number },
  ): Promise<void> {
    if (index === undefined || !data) return;
    setBusy(true);
    setMessage("");
    try {
      const result = await historyApi.POST(
        "/api/clips/{id}/segments/{index}/restore",
        {
          params: { path: { id: clipId, index }, header: csrf },
          body: {
            ...target,
            expected_version: data.current.version,
            reason: reason.trim() || undefined,
            transcription_job_id: jobId,
          },
        },
      );
      if (result.error) {
        const code = (result.error as { error: string }).error;
        setMessage(segmentErrorMessage(code));
        await load();
        await onChanged();
        return;
      }
      setReason("");
      setMessage("Texte restauré.");
      await load();
      await onChanged();
    } catch {
      setMessage("La restauration a échoué. Réessayez.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <span className="segment-history">
      <button
        type="button"
        className="secondary segment-action"
        aria-expanded={open}
        aria-label={`Historique du segment ${label}`}
        onClick={() => setOpen(!open)}
      >
        {open ? "Masquer l’historique" : "Historique"}
      </button>
      {open && data && (
        <span className="segment-history-panel">
          {data.items.length === 0 ? (
            <span className="segment-history-empty">
              Aucune correction pour ce segment. Texte d’origine :{" "}
              {data.original_text}
            </span>
          ) : (
            <>
              <label className="segment-history-reason">
                Motif de la restauration du segment {label} (facultatif)
                <input
                  value={reason}
                  maxLength={500}
                  onChange={(event) => setReason(event.target.value)}
                />
              </label>
              {data.current.corrected && (
                <button
                  type="button"
                  className="secondary"
                  aria-label={`Restaurer le texte d’origine du segment ${label}`}
                  disabled={busy}
                  onClick={() => void restore({ original: true })}
                >
                  Restaurer l’original
                </button>
              )}
              <ol className="segment-history-list">
                {data.items.map((entry) => (
                  <li key={entry.id}>
                    <strong>{kindLabels[entry.kind]}</strong> version{" "}
                    {entry.version_after} par {entry.author.username} le{" "}
                    {formatDate(entry.created_at)}
                    {entry.reason ? ` — motif : ${entry.reason}` : ""}
                    {entry.kind === "speaker" ? (
                      <span className="segment-history-text">
                        Locuteur : {entry.old_speaker_label || "déduit"} →{" "}
                        {entry.new_speaker_label || "déduit"}
                      </span>
                    ) : (
                      <>
                        <span className="segment-history-text">
                          Avant : {entry.old_text}
                        </span>
                        <span className="segment-history-text">
                          Après :{" "}
                          {entry.new_corrected
                            ? entry.new_text
                            : `${entry.new_text} (texte d’origine)`}
                        </span>
                        {entry.new_text !== data.current.text && (
                          <button
                            type="button"
                            className="secondary"
                            aria-label={`Restaurer la version ${entry.version_after} du segment ${label}`}
                            disabled={busy}
                            onClick={() => void restore({ event_id: entry.id })}
                          >
                            Restaurer cette version
                          </button>
                        )}
                      </>
                    )}
                  </li>
                ))}
              </ol>
            </>
          )}
        </span>
      )}
      <span className="segment-message" aria-live="polite">
        {message}
      </span>
    </span>
  );
}
