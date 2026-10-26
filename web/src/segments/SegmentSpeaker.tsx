import React, { useEffect, useState } from "react";
import createClient from "openapi-fetch";
import type { paths, components } from "../api.gen";
import { segmentErrorMessage } from "./SegmentText";
import "./segment-speaker.css";

// Speaker of one transcript segment (phase 6): the user designates which Local speaker
// of the clip speaks it, or returns to the speaker deduced from the speaker turns. The
// correction never touches turns, fingerprints, attributions or voice samples.
const speakerApi = createClient<paths>();
type TranscriptSegment = components["schemas"]["TranscriptSegment"];
type LocalSpeakerChoice = NonNullable<
  components["schemas"]["Transcription"]["local_speakers"]
>[number];
type CsrfHeaders = { "X-CSRF-Token": string; Origin: string };

const speakerMessages: Record<string, string> = {
  invalid_local_speaker:
    "Ce locuteur n’appartient pas à ce clip. Actualisez la transcription.",
  speaker_unchanged: "Le locuteur n’a pas changé.",
  segment_conflict:
    "Ce segment a été modifié depuis votre lecture. La transcription a été actualisée.",
};

export function SegmentSpeaker({
  clipId,
  jobId,
  segment,
  speakers,
  csrf,
  onChanged,
}: {
  clipId: number;
  jobId: number | undefined;
  segment: TranscriptSegment;
  speakers: LocalSpeakerChoice[];
  csrf: CsrfHeaders;
  onChanged: () => void | Promise<void>;
}): React.ReactElement | null {
  const index = segment.index;
  const stored = segment.local_speaker_corrected
    ? String(segment.local_speaker_id ?? "")
    : "";
  const [choice, setChoice] = useState(stored);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  // The stored speaker follows the reading of the clip (another save, a conflict).
  useEffect(() => setChoice(stored), [stored, segment.correction_version]);
  if (index === undefined || speakers.length === 0) return null;
  const label = String(index + 1);

  async function save() {
    if (index === undefined) return;
    setBusy(true);
    setMessage("");
    try {
      const result = await speakerApi.PUT(
        "/api/clips/{id}/segments/{index}/speaker",
        {
          params: { path: { id: clipId, index }, header: csrf },
          body: {
            local_speaker_id: choice === "" ? null : Number(choice),
            expected_version: segment.correction_version ?? 0,
            transcription_job_id: jobId,
          },
        },
      );
      if (result.error) {
        const failure = result.error as { error: string };
        setMessage(
          speakerMessages[failure.error] ?? segmentErrorMessage(failure.error),
        );
        if (failure.error === "segment_conflict") await onChanged();
        return;
      }
      setMessage("Locuteur enregistré.");
      await onChanged();
    } catch {
      setMessage("La correction a échoué. Réessayez.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <span className="segment-speaker">
      {segment.local_speaker_corrected && (
        <span className="segment-correction-badge">Locuteur corrigé</span>
      )}
      <select
        aria-label={`Locuteur du segment ${label}`}
        value={choice}
        disabled={busy}
        onChange={(event) => setChoice(event.target.value)}
      >
        <option value="">Locuteur déduit des tours de parole</option>
        {speakers.map((speaker) => (
          <option key={speaker.id} value={String(speaker.id)}>
            {speaker.label}
          </option>
        ))}
      </select>
      <button
        type="button"
        className="secondary segment-action"
        aria-label={`Enregistrer le locuteur du segment ${label}`}
        disabled={busy || choice === stored}
        onClick={() => void save()}
      >
        Enregistrer le locuteur
      </button>
      <span aria-live="polite" className="segment-speaker-message">
        {message}
      </span>
    </span>
  );
}
