import React, { useEffect, useState } from "react";
import createClient from "openapi-fetch";
import type { paths, components } from "../api.gen";
import "./segment-history.css";

// Corrections of the transcriptions a retranscription replaced (phase 6): read-only,
// never applied again, never restorable, but still consultable.
const obsoleteApi = createClient<paths>();
type ObsoleteCorrection = components["schemas"]["ObsoleteCorrection"];

function formatDate(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString("fr-FR");
}

const kindLabels = {
  text: "Correction",
  speaker: "Locuteur",
  restore: "Restauration",
};

export function ObsoleteCorrections({
  clipId,
  jobId,
}: {
  clipId: number;
  jobId: number | undefined;
}): React.ReactElement | null {
  const [items, setItems] = useState<ObsoleteCorrection[]>([]);
  const [open, setOpen] = useState(false);

  // Reload when the visible transcription changes (a retranscription just completed).
  useEffect(() => {
    let cancelled = false;
    void obsoleteApi
      .GET("/api/clips/{id}/obsolete-corrections", {
        params: { path: { id: clipId } },
      })
      .then((result) => {
        if (!cancelled) setItems(result.data?.items ?? []);
      })
      .catch(() => {
        if (!cancelled) setItems([]);
      });
    return () => {
      cancelled = true;
    };
  }, [clipId, jobId]);

  if (items.length === 0) return null;
  return (
    <section
      className="obsolete-corrections"
      aria-label="Corrections obsolètes"
    >
      <button
        type="button"
        className="secondary"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
      >
        {open
          ? "Masquer les corrections obsolètes"
          : `Corrections obsolètes (${items.length})`}
      </button>
      {open && (
        <>
          <p className="segment-history-empty">
            Ces corrections appartiennent à une transcription remplacée par une
            retranscription : elles ne s’appliquent plus et ne se restaurent
            pas.
          </p>
          <ul className="segment-history-list">
            {items.map((item) => (
              <li key={`${item.transcription_job_id}-${item.index}`}>
                <strong>Segment {item.index + 1}</strong>{" "}
                <span className="segment-correction-badge">Obsolète</span>{" "}
                depuis le {formatDate(item.obsolete_at)}
                <span className="segment-history-text">
                  Texte d’origine : {item.original_text}
                </span>
                {item.text !== null && (
                  <span className="segment-history-text">
                    Dernier texte corrigé : {item.text}
                  </span>
                )}
                <ol className="segment-history-list">
                  {item.history.map((entry) => (
                    <li key={entry.id}>
                      {kindLabels[entry.kind]} version {entry.version_after} par{" "}
                      {entry.author.username} le {formatDate(entry.created_at)}
                      {entry.reason ? ` — motif : ${entry.reason}` : ""}
                      {entry.kind === "speaker" ? (
                        <span className="segment-history-text">
                          Locuteur : {entry.old_speaker_label || "déduit"} →{" "}
                          {entry.new_speaker_label || "déduit"}
                        </span>
                      ) : (
                        <span className="segment-history-text">
                          {entry.old_text} → {entry.new_text}
                        </span>
                      )}
                    </li>
                  ))}
                </ol>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
}
