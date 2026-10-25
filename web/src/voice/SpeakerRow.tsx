import React, { useState } from "react";
import {
  errorCode,
  levelGlyphs,
  levelLabels,
  messageFor,
  voiceApi,
  type Attribution,
  type AttributionCandidate,
  type AttributionEvent,
  type CsrfHeaders,
  type DecisionLevel,
  type LocalSpeaker,
  type Person,
} from "./client";

export function LevelBadge({
  level,
}: {
  level: DecisionLevel;
}): React.ReactElement {
  return (
    <span className={`level-badge level-${level}`}>
      <span aria-hidden="true">{levelGlyphs[level]}</span>
      {levelLabels[level]}
    </span>
  );
}

function when(iso: string): string {
  return new Date(iso).toLocaleString("fr-FR");
}

const actionLabels: Record<string, string> = {
  correct: "Correction",
  match: "Rapprochement",
  merge: "Fusion de Personnes",
  split: "Séparation de la Personne",
  delete_person: "Suppression de la Personne",
};

function actionLabel(action: string): string {
  return actionLabels[action] ?? "Validation";
}

function authorName(attribution: Attribution | null | undefined): string {
  return attribution?.author?.username ?? "un utilisateur supprimé";
}

function sampleText(count: number): string {
  return `${count} échantillon${count > 1 ? "s" : ""} validé${count > 1 ? "s" : ""}`;
}

// One local speaker: listen to an extract, then validate or correct its Person.
// Human validation and unknown state are told apart by label, glyph and style.
export function SpeakerRow({
  speaker,
  persons,
  csrf,
  onChanged,
}: {
  speaker: LocalSpeaker;
  persons: Person[];
  csrf: CsrfHeaders;
  onChanged: () => Promise<void>;
}): React.ReactElement {
  const [editing, setEditing] = useState(false);
  const [personId, setPersonId] = useState("");
  const [newName, setNewName] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [history, setHistory] = useState<AttributionEvent[] | null>(null);
  const [separating, setSeparating] = useState(false);

  const attribution = speaker.attribution;
  const validated = attribution?.validation_state === "validated";
  const usable = speaker.fingerprints.filter((f) => f.usable).length;
  const person = speaker.person
    ? persons.find((p) => p.id === speaker.person?.id)
    : undefined;
  const canSubmit = newName.trim() !== "" || personId !== "";

  async function loadHistory() {
    const result = await voiceApi.GET("/api/local-speakers/{id}/history", {
      params: { path: { id: speaker.id } },
    });
    setHistory(result.data?.items ?? []);
  }

  // A candidate proposed by matching is confirmed like any other validation:
  // only the human click creates the validated sample.
  async function confirmCandidate(candidate: AttributionCandidate) {
    await submit({ person_id: candidate.person.id });
  }

  async function submit(chosen?: { person_id: number }) {
    setBusy(true);
    setMessage("");
    const person =
      chosen ??
      (newName.trim() !== ""
        ? { new_person_name: newName.trim() }
        : { person_id: Number(personId) });
    try {
      if (validated && attribution?.person) {
        const result = await voiceApi.POST(
          "/api/local-speakers/{id}/correction",
          {
            params: { path: { id: speaker.id }, header: csrf },
            body: { ...person, expected_person_id: attribution.person.id },
          },
        );
        if (result.error) {
          setMessage(messageFor(errorCode(result.error)));
        } else {
          setMessage("Correction enregistrée.");
        }
      } else {
        const result = await voiceApi.POST(
          "/api/local-speakers/{id}/validation",
          {
            params: { path: { id: speaker.id }, header: csrf },
            body: person,
          },
        );
        if (result.error) {
          const conflict = result.error as {
            error: string;
            attribution?: Attribution;
          };
          setMessage(
            conflict.error === "already_validated" && conflict.attribution
              ? `Déjà validé par ${authorName(conflict.attribution)} : ${conflict.attribution.person?.display_name ?? "?"}. La première validation fait foi.`
              : messageFor(conflict.error),
          );
        } else {
          setMessage("Validation enregistrée.");
        }
      }
      setEditing(false);
      setPersonId("");
      setNewName("");
      if (history) await loadHistory();
      await onChanged();
    } catch {
      setMessage("La demande a échoué. Réessayez.");
    } finally {
      setBusy(false);
    }
  }

  // Withdraws the human validation of this voice: it is unknown again.
  async function separate() {
    if (!attribution?.person) return;
    setBusy(true);
    setMessage("");
    try {
      const result = await voiceApi.POST("/api/persons/{id}/separation", {
        params: { path: { id: attribution.person.id }, header: csrf },
        body: { local_speaker_ids: [speaker.id] },
      });
      setMessage(
        result.error
          ? messageFor(errorCode(result.error))
          : "Voix séparée de la Personne : elle est de nouveau inconnue.",
      );
      setSeparating(false);
      if (history) await loadHistory();
      await onChanged();
    } catch {
      setMessage("La demande a échoué. Réessayez.");
    } finally {
      setBusy(false);
    }
  }

  const formVisible = editing || !validated;
  return (
    <div className="speaker-row">
      <div>
        <strong>{speaker.label}</strong> ·{" "}
        {speaker.person ? (
          <>
            {speaker.person.display_name}{" "}
            {person && <>({sampleText(person.validated_sample_count)}) </>}
          </>
        ) : (
          <>non identifié </>
        )}
        <LevelBadge level={speaker.decision_level} />
        {" · "}
        {speaker.speaking_seconds.toFixed(1)} s de parole ·{" "}
        {speaker.fingerprints.length} empreinte(s), {usable} utilisable(s)
      </div>
      {validated && attribution && (
        <div className="hint">
          Validé par {authorName(attribution)} le {when(attribution.decided_at)}
          {attribution.revision > 1
            ? ` (révision ${attribution.revision})`
            : ""}
        </div>
      )}
      {!validated && attribution && attribution.candidates.length > 0 && (
        <div className="speaker-candidates">
          <span className="hint">
            {attribution.decision_level === "ambiguous"
              ? "Voix proche de plusieurs Personnes : à trancher."
              : "Proposé par le rapprochement vocal (similarité, pas une garantie)."}
          </span>
          <ul aria-label={`Personnes proches de ${speaker.label}`}>
            {attribution.candidates.map((candidate) => (
              <li key={candidate.person.id}>
                {candidate.person.display_name} · similarité{" "}
                {Math.round(candidate.score * 100)} %{" "}
                <button
                  className="secondary"
                  disabled={busy}
                  onClick={() => void confirmCandidate(candidate)}
                >
                  Valider comme {candidate.person.display_name}
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
      <audio
        controls
        preload="none"
        aria-label={`Extrait de ${speaker.label}`}
        src={`/api/local-speakers/${speaker.id}/excerpt`}
      />
      {formVisible ? (
        <div className="speaker-validation">
          <label>
            Personne existante
            <select
              value={personId}
              disabled={newName.trim() !== ""}
              onChange={(event) => setPersonId(event.target.value)}
            >
              <option value="">Choisir…</option>
              {persons.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.display_name} · {sampleText(p.validated_sample_count)}
                </option>
              ))}
            </select>
          </label>
          <label>
            Nouvelle Personne
            <input
              value={newName}
              maxLength={100}
              onChange={(event) => setNewName(event.target.value)}
            />
          </label>
          <button disabled={busy || !canSubmit} onClick={() => void submit()}>
            {validated ? "Enregistrer la correction" : "Valider"}
          </button>
          {validated && (
            <button className="secondary" onClick={() => setEditing(false)}>
              Annuler
            </button>
          )}
        </div>
      ) : (
        <div className="speaker-validation">
          <button className="secondary" onClick={() => setEditing(true)}>
            Corriger
          </button>
          {separating ? (
            <>
              <button disabled={busy} onClick={() => void separate()}>
                Confirmer la séparation
              </button>
              <button
                className="secondary"
                onClick={() => setSeparating(false)}
              >
                Annuler la séparation
              </button>
            </>
          ) : (
            <button className="secondary" onClick={() => setSeparating(true)}>
              Séparer de cette Personne
            </button>
          )}
        </div>
      )}
      <div className="speaker-validation">
        <button
          className="secondary"
          onClick={() => (history ? setHistory(null) : void loadHistory())}
        >
          {history ? "Masquer l’historique" : "Historique"}
        </button>
      </div>
      {message && (
        <p className="hint" aria-live="polite">
          {message}
        </p>
      )}
      {history && (
        <ul
          className="speaker-history"
          aria-label={`Historique ${speaker.label}`}
        >
          {history.length === 0 && <li>Aucun changement enregistré.</li>}
          {history.map((event) => (
            <li key={event.id}>
              {actionLabel(event.action)} :{" "}
              {event.old_person?.display_name ??
                levelLabels[event.old_decision_level as DecisionLevel] ??
                "Inconnu"}{" "}
              → {event.new_person?.display_name ?? "Inconnu"}{" "}
              {event.action === "match"
                ? "par le rapprochement vocal"
                : `par ${event.author?.username ?? "un utilisateur supprimé"}`}{" "}
              le {when(event.created_at)}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
