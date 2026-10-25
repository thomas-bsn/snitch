import React, { useEffect, useState } from "react";
import {
  voiceApi,
  voiceChecked,
  type CsrfHeaders,
  type VoiceMatchingSettings,
  type VoiceMatchingSummary,
} from "./client";
import "./voice.css";

type Form = {
  automatic: string;
  suggestion: string;
  margin: string;
  candidates: string;
};

function toForm(settings: VoiceMatchingSettings): Form {
  return {
    automatic: String(settings.automatic_threshold),
    suggestion: String(settings.suggestion_threshold),
    margin: String(settings.ambiguity_margin),
    candidates: String(settings.max_candidates),
  };
}

// Administrator card: thresholds between the decision levels and an explicit
// rematch. The activation of the automatic attribution is a separate decision and
// is only displayed here.
export function VoiceMatchingCard({
  csrf,
}: {
  csrf: CsrfHeaders;
}): React.ReactElement {
  const [settings, setSettings] = useState<VoiceMatchingSettings | null>(null);
  const [form, setForm] = useState<Form | null>(null);
  const [message, setMessage] = useState("");
  const [summary, setSummary] = useState<VoiceMatchingSummary | null>(null);

  useEffect(() => {
    voiceApi
      .GET("/api/voice-matching")
      .then((result) => {
        const loaded = voiceChecked(result);
        setSettings(loaded);
        setForm(toForm(loaded));
      })
      .catch((cause) =>
        setMessage(cause instanceof Error ? cause.message : "Indisponible."),
      );
  }, []);

  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form) return;
    try {
      const saved = voiceChecked(
        await voiceApi.PUT("/api/admin/voice-matching", {
          params: { header: csrf },
          body: {
            automatic_threshold: Number(form.automatic),
            suggestion_threshold: Number(form.suggestion),
            ambiguity_margin: Number(form.margin),
            max_candidates: Number(form.candidates),
          },
        }),
      );
      setSettings(saved);
      setForm(toForm(saved));
      setMessage(
        "Seuils enregistrés ; les voix ont été rapprochées de nouveau.",
      );
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    }
  }

  async function rematch() {
    try {
      setSummary(
        voiceChecked(
          await voiceApi.POST("/api/admin/voice-matching/rematch", {
            params: { header: csrf },
          }),
        ),
      );
      setMessage("");
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    }
  }

  const update = (patch: Partial<Form>) =>
    setForm((current) => (current ? { ...current, ...patch } : current));

  return (
    <section className="card" id="voice-matching">
      <h2>Rapprochement vocal</h2>
      <p className="hint">
        Les références d’une Personne sont ses seuls échantillons validés par un
        humain. Le rapprochement s’exécute en base, sans le PC ni les vidéos.
        Les valeurs initiales ne sont pas des choix définitifs : elles attendent
        le benchmark.
      </p>
      {settings && (
        <p>
          Attribution automatique :{" "}
          <strong>{settings.auto_enabled ? "activée" : "désactivée"}</strong>
          {settings.auto_enabled
            ? "."
            : " — les rapprochements restent des suggestions."}
        </p>
      )}
      {form && (
        <form className="voice-pipeline-form" onSubmit={save}>
          <label>
            Seuil d’attribution automatique
            <input
              type="number"
              step="any"
              min="0"
              max="1"
              value={form.automatic}
              onChange={(event) => update({ automatic: event.target.value })}
            />
          </label>
          <label>
            Seuil de suggestion
            <input
              type="number"
              step="any"
              min="0"
              max="1"
              value={form.suggestion}
              onChange={(event) => update({ suggestion: event.target.value })}
            />
          </label>
          <label>
            Marge d’ambiguïté
            <input
              type="number"
              step="any"
              min="0"
              max="1"
              value={form.margin}
              onChange={(event) => update({ margin: event.target.value })}
            />
          </label>
          <label>
            Candidates affichées au plus
            <input
              type="number"
              min="2"
              max="20"
              value={form.candidates}
              onChange={(event) => update({ candidates: event.target.value })}
            />
          </label>
          <button>Enregistrer les seuils</button>
          <button type="button" className="secondary" onClick={rematch}>
            Relancer le rapprochement
          </button>
        </form>
      )}
      {summary && (
        <p className="hint" aria-live="polite">
          Rapprochement terminé : {summary.evaluated} voix examinées (
          {summary.automatic} automatiques, {summary.suggestion} suggestions,{" "}
          {summary.ambiguous} ambiguës, {summary.unknown} inconnues).
        </p>
      )}
      {message && (
        <p className="hint" aria-live="polite">
          {message}
        </p>
      )}
    </section>
  );
}
