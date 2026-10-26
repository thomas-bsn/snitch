import React, { useEffect, useState } from "react";
import {
  voiceApi,
  voiceChecked,
  type CsrfHeaders,
  type VoicePipeline,
} from "./client";
import "./voice.css";

export function VoicePipelineCard({
  csrf,
}: {
  csrf: CsrfHeaders;
}): React.ReactElement {
  const [version, setVersion] = useState("");
  const [config, setConfig] = useState<VoicePipeline | null>(null);
  const [message, setMessage] = useState("");

  useEffect(() => {
    voiceApi
      .GET("/api/admin/voice-pipeline")
      .then((result) => {
        const state = voiceChecked(result);
        setVersion(state.version);
        setConfig(state.config);
      })
      .catch((cause) =>
        setMessage(cause instanceof Error ? cause.message : "Indisponible."),
      );
  }, []);

  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!config) return;
    try {
      const state = voiceChecked(
        await voiceApi.PUT("/api/admin/voice-pipeline", {
          params: { header: csrf },
          body: config,
        }),
      );
      setVersion(state.version);
      setConfig(state.config);
      setMessage("Configuration vocale enregistrée.");
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    }
  }

  const update = (patch: Partial<VoicePipeline>) =>
    setConfig((current) => (current ? { ...current, ...patch } : current));

  return (
    <section className="card" id="voice-pipeline">
      <h2>Pipeline vocal</h2>
      <p className="hint">
        Version active : {version}. Aucun modèle n’est téléchargé implicitement
        : confirmez l’accès (jeton et conditions) avant toute planification avec
        un vrai modèle.
      </p>
      {config && (
        <form className="voice-pipeline-form" onSubmit={save}>
          <label>
            Fournisseur de diarisation
            <select
              value={config.provider}
              onChange={(event) =>
                update({
                  provider: event.target.value as VoicePipeline["provider"],
                  fake:
                    event.target.value === "fake"
                      ? { speakers: 2, dimensions: 8 }
                      : undefined,
                })
              }
            >
              <option value="pyannote">pyannote</option>
              <option value="fake">Fournisseur factice (tests)</option>
            </select>
          </label>
          <label>
            Modèle de diarisation
            <input
              value={config.model}
              onChange={(event) => update({ model: event.target.value })}
            />
          </label>
          <label>
            Version du modèle
            <input
              value={config.model_version}
              onChange={(event) =>
                update({ model_version: event.target.value })
              }
            />
          </label>
          <label className="inline">
            <input
              type="checkbox"
              checked={config.access_confirmed}
              onChange={(event) =>
                update({ access_confirmed: event.target.checked })
              }
            />
            Accès au modèle confirmé
          </label>
          <label>
            Durée minimale d’une empreinte (s)
            <input
              type="number"
              step="0.1"
              min="0.1"
              max="60"
              value={config.min_fingerprint_seconds}
              onChange={(event) =>
                update({ min_fingerprint_seconds: Number(event.target.value) })
              }
            />
          </label>
          <label>
            Qualité minimale
            <input
              type="number"
              step="0.05"
              min="0"
              max="1"
              value={config.min_quality}
              onChange={(event) =>
                update({ min_quality: Number(event.target.value) })
              }
            />
          </label>
          {config.fake && (
            <label>
              Locuteurs factices
              <input
                type="number"
                min="1"
                max="8"
                value={config.fake.speakers}
                onChange={(event) =>
                  update({
                    fake: {
                      speakers: Number(event.target.value),
                      dimensions: config.fake?.dimensions ?? 8,
                    },
                  })
                }
              />
            </label>
          )}
          <button>Enregistrer et versionner</button>
        </form>
      )}
      {message && (
        <p className="hint" aria-live="polite">
          {message}
        </p>
      )}
    </section>
  );
}
