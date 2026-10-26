import React from "react";
import type { components } from "../api.gen";
import "./segments.css";

// Protection of the human corrections when a clip is retranscribed (phase 6): the
// server refuses with 409 corrections_would_be_replaced until the administrator
// confirms; this panel tells what would be replaced.
export type CorrectionsToReplace =
  components["schemas"]["CorrectionsToReplace"];

export function replacedCorrections(
  error: unknown,
): CorrectionsToReplace | undefined {
  const body = error as
    | { error?: string; corrections?: CorrectionsToReplace }
    | undefined;
  return body?.error === "corrections_would_be_replaced"
    ? body.corrections
    : undefined;
}

export function describeCorrections(c: CorrectionsToReplace): string {
  const parts = [];
  if (c.text_segments > 0) parts.push(`${c.text_segments} texte(s) corrigé(s)`);
  if (c.speaker_segments > 0)
    parts.push(`${c.speaker_segments} locuteur(s) corrigé(s)`);
  const authors = c.authors.length ? ` par ${c.authors.join(", ")}` : "";
  const last = c.last_corrected_at
    ? `, dernière correction le ${new Date(c.last_corrected_at).toLocaleString("fr-FR")}`
    : "";
  return `${parts.join(" et ")}${authors}${last}`;
}

export function CorrectionsWarningPanel({
  corrections,
  busy,
  onConfirm,
  onCancel,
}: {
  corrections: CorrectionsToReplace;
  busy: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <section
      className="retranscription-warning"
      aria-label="Corrections à remplacer"
    >
      <p>
        <strong>
          Cette retranscription remplacerait des corrections humaines :
        </strong>{" "}
        {describeCorrections(corrections)}.
      </p>
      <p className="hint">
        Elles resteront dans l’historique, marquées obsolètes, mais ne seront
        jamais réappliquées au nouveau texte. Le texte actuel reste affiché tant
        que la retranscription n’a pas réussi.
      </p>
      <div className="segment-actions">
        <button disabled={busy} onClick={onConfirm}>
          Confirmer la retranscription et remplacer les corrections
        </button>
        <button className="secondary" disabled={busy} onClick={onCancel}>
          Annuler la retranscription
        </button>
      </div>
    </section>
  );
}
