import React, { useEffect, useState } from "react";
import {
  voiceApi,
  voiceChecked,
  type ClipVoice,
  type CsrfHeaders,
} from "./client";
import "./voice.css";

const statusLabels: Record<ClipVoice["status"], string> = {
  none: "Voix non traitées",
  queued: "En file",
  running: "En cours",
  done: "Terminé",
  failed: "Échec",
};

const stepLabels: Record<string, string> = {
  "voice.diarization": "Diarisation",
  "voice.fingerprint": "Empreintes",
};

const jobStatusLabels: Record<string, string> = {
  pending: "en file",
  leased: "en cours",
  done: "terminé",
  failed: "échec",
  retry_wait: "nouvelle tentative",
  cancelled: "annulé",
  dead: "abandonné",
};

function stepLabel(step: string): string {
  const base = step.split(".").slice(0, 2).join(".");
  return stepLabels[base] ?? step;
}

export function SpeakerBadge({
  label,
}: {
  label: string | undefined;
}): React.ReactElement | null {
  if (!label) return null;
  return (
    <span
      className="speaker-badge"
      title="Locuteur local non identifié dans ce clip"
    >
      {label} · non identifié
    </span>
  );
}

export function VoicePanel({
  clipId,
  isAdmin,
  csrf,
  reloadToken,
}: {
  clipId: number;
  isAdmin: boolean;
  csrf: CsrfHeaders;
  reloadToken?: unknown;
}): React.ReactElement {
  const [voice, setVoice] = useState<ClipVoice | null>(null);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    try {
      setVoice(
        voiceChecked(
          await voiceApi.GET("/api/clips/{id}/voice", {
            params: { path: { id: clipId } },
          }),
        ),
      );
    } catch (cause) {
      setMessage(
        cause instanceof Error ? cause.message : "Voix indisponibles.",
      );
    }
  }
  useEffect(() => {
    void load();
  }, [clipId, reloadToken]);

  async function schedule() {
    setBusy(true);
    setMessage("");
    try {
      const response = voiceChecked(
        await voiceApi.POST("/api/admin/clips/{id}/voice", {
          params: { path: { id: clipId }, header: csrf },
          body: {},
        }),
      );
      setMessage(`Traitement vocal planifié (tâche ${response.job.id}).`);
      await load();
    } catch (cause) {
      setMessage(
        cause instanceof Error ? cause.message : "Planification impossible.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="clip-voice" aria-label="Voix du clip">
      <h3>Voix · {voice ? statusLabels[voice.status] : "chargement"}</h3>
      {isAdmin && (
        <button className="secondary" disabled={busy} onClick={schedule}>
          Diariser les voix
        </button>
      )}
      {message && (
        <p className="hint" aria-live="polite">
          {message}
        </p>
      )}
      {voice && voice.steps.length > 0 && (
        <ul className="voice-steps" aria-label="Étapes vocales">
          {voice.steps.map((step) => (
            <li key={step.job_id}>
              {stepLabel(step.step)} ·{" "}
              {jobStatusLabels[step.status] ?? step.status}
              {step.last_error && (
                <span className="voice-error">
                  {" "}
                  · erreur : {step.last_error}
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
      {voice && voice.speakers.length > 0 && (
        <ul className="voice-speakers" aria-label="Locuteurs locaux">
          {voice.speakers.map((speaker) => {
            const usable = speaker.fingerprints.filter((f) => f.usable).length;
            return (
              <li key={speaker.id}>
                <strong>{speaker.label}</strong> · non identifié ·{" "}
                {speaker.speaking_seconds.toFixed(1)} s de parole ·{" "}
                {speaker.fingerprints.length} empreinte(s), {usable}{" "}
                utilisable(s)
              </li>
            );
          })}
        </ul>
      )}
      {voice && voice.status === "done" && voice.speakers.length === 0 && (
        <p className="hint">Aucune voix détectée.</p>
      )}
    </section>
  );
}
