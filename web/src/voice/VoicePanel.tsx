import React, { useEffect, useState } from "react";
import {
  voiceApi,
  voiceChecked,
  type ClipVoice,
  type CsrfHeaders,
  type DecisionLevel,
  type Person,
  levelGlyphs,
  levelLabels,
} from "./client";
import { SpeakerRow } from "./SpeakerRow";
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
  personName,
  level,
}: {
  label: string | undefined;
  personName?: string;
  level?: DecisionLevel;
}): React.ReactElement | null {
  if (!label) return null;
  const effective: DecisionLevel = level ?? "unknown";
  return (
    <span
      className={`speaker-badge level-${effective}`}
      title={levelLabels[effective]}
    >
      <span aria-hidden="true">{levelGlyphs[effective]}</span> {label} ·{" "}
      {personName ?? "non identifié"}
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
  const [persons, setPersons] = useState<Person[]>([]);

  async function loadPersons() {
    const result = await voiceApi.GET("/api/persons", { params: {} });
    setPersons(result.data?.items ?? []);
  }

  async function reload() {
    await Promise.all([load(), loadPersons()]);
  }

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
    void reload();
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
          {voice.speakers.map((speaker) => (
            <li key={speaker.id}>
              <SpeakerRow
                speaker={speaker}
                persons={persons}
                csrf={csrf}
                onChanged={reload}
              />
            </li>
          ))}
        </ul>
      )}
      {voice && voice.status === "done" && voice.speakers.length === 0 && (
        <p className="hint">Aucune voix détectée.</p>
      )}
    </section>
  );
}
