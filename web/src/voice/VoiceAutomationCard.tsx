import React, { useCallback, useEffect, useState } from "react";
import {
  voiceApi,
  voiceChecked,
  type CsrfHeaders,
  type VoiceAccuracy,
  type VoiceMatchingSettings,
} from "./client";
import "./voice.css";

const percent = (value: number | null | undefined) =>
  value === null || value === undefined
    ? "non mesurée"
    : `${Math.round(value * 100)} %`;

// Automation of the voice attribution. Every signed-in user sees whether it is
// active; only administrators see the measured accuracy and can change the
// criteria or the switch. No figure here is a guarantee of quality.
export function VoiceAutomationCard({
  isAdmin,
  csrf,
}: {
  isAdmin: boolean;
  csrf: CsrfHeaders;
}): React.ReactElement {
  const [settings, setSettings] = useState<VoiceMatchingSettings | null>(null);
  const [accuracy, setAccuracy] = useState<VoiceAccuracy | null>(null);
  const [message, setMessage] = useState("");
  const [forced, setForced] = useState(false);
  const [precision, setPrecision] = useState("");
  const [confirmations, setConfirmations] = useState("");

  const load = useCallback(async () => {
    try {
      const loaded = voiceChecked(await voiceApi.GET("/api/voice-matching"));
      setSettings(loaded);
      if (isAdmin) {
        const report = voiceChecked(
          await voiceApi.GET("/api/admin/voice-accuracy"),
        );
        setAccuracy(report);
        setPrecision(String(report.precision_threshold));
        setConfirmations(String(report.min_confirmations));
      }
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Indisponible.");
    }
  }, [isAdmin]);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveCriteria(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    try {
      voiceChecked(
        await voiceApi.PUT("/api/admin/voice-automation/criteria", {
          params: { header: csrf },
          body: {
            precision_threshold: Number(precision),
            min_confirmations: Number(confirmations),
          },
        }),
      );
      setMessage("Critères enregistrés.");
      await load();
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    }
  }

  async function switchTo(enabled: boolean) {
    try {
      const result = voiceChecked(
        await voiceApi.PUT("/api/admin/voice-automation", {
          params: { header: csrf },
          body: { enabled, confirm_unmet_criteria: forced },
        }),
      );
      setForced(false);
      setMessage(
        enabled
          ? `Attribution automatique activée : ${result.summary.automatic} voix attribuées automatiquement.`
          : "Attribution automatique désactivée : les attributions automatiques sont redevenues des suggestions.",
      );
      await load();
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    }
  }

  const evaluation = accuracy?.evaluation;
  const library = accuracy?.library;
  return (
    <section className="card" id="voice-automation">
      <h2>Attribution automatique des voix</h2>
      {settings && (
        <p>
          État :{" "}
          <strong>{settings.auto_enabled ? "activée" : "désactivée"}</strong>
          {settings.auto_enabled
            ? ". Les voix très proches d’une référence sont attribuées sans validation, et restent distinctes des validations humaines."
            : ". Les rapprochements restent des suggestions à confirmer."}
        </p>
      )}
      {!isAdmin && (
        <p className="hint">
          Seul un administrateur peut activer ou désactiver l’attribution
          automatique.
        </p>
      )}
      {isAdmin && accuracy && evaluation && library && (
        <>
          <p className="hint">
            Mesure sur des clips distincts de ceux des références : un
            échantillon validé est soit référence, soit test. Ces chiffres
            décrivent les échantillons disponibles, ils ne garantissent aucune
            qualité d’identification.
          </p>
          <ul className="automation-figures" aria-label="Mesure de précision">
            <li>Précision mesurée : {percent(evaluation.precision)}</li>
            <li>Confirmations : {evaluation.confirmed}</li>
            <li>
              Tests : {evaluation.test_samples} (références :{" "}
              {evaluation.reference_samples}, écartés :{" "}
              {evaluation.excluded_samples})
            </li>
            <li>Couverture simulée : {percent(evaluation.coverage)}</li>
          </ul>
          <ul className="automation-figures" aria-label="Bibliothèque">
            <li>Couverture automatique : {percent(library.coverage)}</li>
            <li>Voix automatiques ou éligibles : {library.automatic}</li>
            <li>Vérification restante : {library.to_verify}</li>
            <li>Validées par un humain : {library.validated}</li>
            <li>Pistes micro (à part) : {library.microphone_assigned}</li>
          </ul>
          {accuracy.persons.length > 0 && (
            <table aria-label="Précision par Personne">
              <thead>
                <tr>
                  <th>Personne</th>
                  <th>Références</th>
                  <th>Tests</th>
                  <th>Attributions automatiques simulées</th>
                  <th>Précision</th>
                </tr>
              </thead>
              <tbody>
                {accuracy.persons.map((row) => (
                  <tr key={row.person.id}>
                    <td>{row.person.display_name}</td>
                    <td>{row.reference_samples}</td>
                    <td>{row.test_samples}</td>
                    <td>{row.automatic_predictions}</td>
                    <td>{percent(row.precision)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          {accuracy.activation_proposed && (
            <p className="automation-proposal">
              Les critères sont atteints : Snitch vous propose d’activer
              l’attribution automatique. Rien ne change sans votre action.
            </p>
          )}
          <form className="voice-pipeline-form" onSubmit={saveCriteria}>
            <label>
              Précision minimale pour proposer l’activation
              <input
                type="number"
                step="any"
                min="0"
                max="1"
                value={precision}
                onChange={(event) => setPrecision(event.target.value)}
              />
            </label>
            <label>
              Confirmations minimales
              <input
                type="number"
                min="1"
                value={confirmations}
                onChange={(event) => setConfirmations(event.target.value)}
              />
            </label>
            <button>Enregistrer les critères d’activation</button>
          </form>
          {!accuracy.auto_enabled && !accuracy.criteria.met && (
            <label className="inline">
              <input
                type="checkbox"
                checked={forced}
                onChange={(event) => setForced(event.target.checked)}
              />
              Activer malgré des critères non atteints
            </label>
          )}
          {accuracy.auto_enabled ? (
            <button type="button" onClick={() => void switchTo(false)}>
              Désactiver l’attribution automatique
            </button>
          ) : (
            <button
              type="button"
              disabled={!accuracy.criteria.met && !forced}
              onClick={() => void switchTo(true)}
            >
              Activer l’attribution automatique
            </button>
          )}
          {accuracy.events.length > 0 && (
            <ul aria-label="Historique de l’automatisation">
              {accuracy.events.map((event) => (
                <li key={event.created_at + event.action}>
                  {event.action === "enable" ? "Activation" : "Désactivation"}{" "}
                  par {event.author}
                  {event.forced ? " (critères non atteints)" : ""} —{" "}
                  {new Date(event.created_at).toLocaleString("fr-FR")}
                </li>
              ))}
            </ul>
          )}
        </>
      )}
      {message && (
        <p className="hint" aria-live="polite">
          {message}
        </p>
      )}
    </section>
  );
}
