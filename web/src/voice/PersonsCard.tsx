import React, { useCallback, useEffect, useState } from "react";
import {
  voiceApi,
  voiceChecked,
  type CsrfHeaders,
  type Person,
  type PersonCorrectionResult,
  type PersonEvent,
} from "./client";
import "./voice.css";

const actionLabels: Record<PersonEvent["action"], string> = {
  merge: "Fusion",
  split: "Séparation",
  delete: "Suppression",
};

function sampleText(count: number): string {
  return `${count} échantillon${count > 1 ? "s" : ""} validé${count > 1 ? "s" : ""}`;
}

function summaryText(result: PersonCorrectionResult): string {
  return `${result.attributions} attribution(s), ${result.samples} échantillon(s), ${result.microphone_rules} désignation(s) de piste micro concernés ; les voix sont rapprochées de nouveau.`;
}

type Pending = { kind: "merge" | "delete"; person: Person };

// Corrections of Persons created by mistake: merge into another, or delete. Each
// action is confirmed before it runs, answers with what it changed and is kept in an
// immutable history. Separating one voice from its Person is done on the voice itself.
export function PersonsCard({
  csrf,
}: {
  csrf: CsrfHeaders;
}): React.ReactElement {
  const [persons, setPersons] = useState<Person[]>([]);
  const [pending, setPending] = useState<Pending | null>(null);
  const [targetId, setTargetId] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [events, setEvents] = useState<PersonEvent[] | null>(null);

  const load = useCallback(async () => {
    const result = voiceChecked(await voiceApi.GET("/api/persons", {}));
    setPersons(result.items);
  }, []);

  const loadEvents = useCallback(async () => {
    const result = voiceChecked(await voiceApi.GET("/api/person-events", {}));
    setEvents(result.items);
  }, []);

  useEffect(() => {
    load().catch((cause) =>
      setMessage(cause instanceof Error ? cause.message : "Indisponible."),
    );
  }, [load]);

  function fail(code: string): string {
    switch (code) {
      case "person_linked_account":
        return "Cette Personne est liée à un compte : elle ne peut pas être absorbée ni supprimée ici.";
      case "voice_profile_deleted":
        return "Cette Personne a supprimé son profil vocal : elle ne peut être ni fusionnée ni cible d’une fusion.";
      case "same_person":
        return "Choisissez une autre Personne.";
      case "person_not_found":
        return "Cette Personne n’existe plus.";
      default:
        return "La demande a échoué. Réessayez.";
    }
  }

  async function confirm() {
    if (!pending) return;
    setBusy(true);
    setMessage("");
    try {
      const params = {
        path: { id: pending.person.id },
        header: csrf,
      };
      const result =
        pending.kind === "merge"
          ? await voiceApi.POST("/api/persons/{id}/merge", {
              params,
              body: { target_person_id: Number(targetId) },
            })
          : await voiceApi.DELETE("/api/persons/{id}", { params });
      if (result.error) {
        setMessage(fail((result.error as { error: string }).error));
      } else if (result.data) {
        setMessage(
          `${pending.kind === "merge" ? "Fusion" : "Suppression"} de ${pending.person.display_name} faite : ${summaryText(result.data)}`,
        );
        setPending(null);
        setTargetId("");
        await load();
        if (events) await loadEvents();
      }
    } catch {
      setMessage("La demande a échoué. Réessayez.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="card" id="persons">
      <h2>Personnes</h2>
      <p className="hint">
        Corrigez une Personne créée par erreur : la fusion rattache ses voix
        validées et ses échantillons à une autre Personne ; la suppression remet
        ses voix à « inconnu » et efface ses échantillons des références. Les
        clips ne sont jamais supprimés et chaque correction reste dans
        l’historique.
      </p>
      <button
        type="button"
        className="secondary"
        onClick={() => void load()}
        disabled={busy}
      >
        Actualiser les Personnes
      </button>
      {persons.length === 0 && <p className="hint">Aucune Personne.</p>}
      <ul aria-label="Liste des Personnes" className="persons-list">
        {persons.map((person) => (
          <li key={person.id}>
            <strong>{person.display_name}</strong> ·{" "}
            {sampleText(person.validated_sample_count)} ·{" "}
            {person.local_speaker_count} voix validée(s)
            {person.user_id !== null && " · liée à un compte"}
            {person.voice_profile_deleted_at !== null &&
              " · profil vocal supprimé"}{" "}
            <button
              type="button"
              className="secondary"
              disabled={busy}
              onClick={() => {
                setPending({ kind: "merge", person });
                setTargetId("");
              }}
            >
              Fusionner {person.display_name}
            </button>{" "}
            <button
              type="button"
              className="secondary"
              disabled={busy}
              onClick={() => setPending({ kind: "delete", person })}
            >
              Supprimer {person.display_name}
            </button>
          </li>
        ))}
      </ul>
      {pending && (
        <div className="person-correction" role="group" aria-label="Correction">
          {pending.kind === "merge" ? (
            <>
              <label>
                Fusionner {pending.person.display_name} dans
                <select
                  value={targetId}
                  onChange={(event) => setTargetId(event.target.value)}
                >
                  <option value="">Choisir…</option>
                  {persons
                    .filter(
                      (p) =>
                        p.id !== pending.person.id &&
                        p.voice_profile_deleted_at === null,
                    )
                    .map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.display_name}
                      </option>
                    ))}
                </select>
              </label>
              <button
                type="button"
                disabled={busy || targetId === ""}
                onClick={() => void confirm()}
              >
                Confirmer la fusion
              </button>
            </>
          ) : (
            <>
              <p>
                Supprimer {pending.person.display_name} remet ses voix à «
                inconnu » et efface ses échantillons des références. Cette
                action ne peut pas être annulée.
              </p>
              <button
                type="button"
                disabled={busy}
                onClick={() => void confirm()}
              >
                Confirmer la suppression
              </button>
            </>
          )}{" "}
          <button
            type="button"
            className="secondary"
            onClick={() => setPending(null)}
          >
            Annuler la correction
          </button>
        </div>
      )}
      {message && (
        <p className="hint" aria-live="polite">
          {message}
        </p>
      )}
      <button
        type="button"
        className="secondary"
        onClick={() => (events ? setEvents(null) : void loadEvents())}
      >
        {events
          ? "Masquer le journal des corrections"
          : "Journal des corrections"}
      </button>
      {events && (
        <ul aria-label="Historique des corrections de Personnes">
          {events.length === 0 && <li>Aucune correction enregistrée.</li>}
          {events.map((event) => (
            <li key={event.id}>
              {actionLabels[event.action]} : {event.person.display_name}
              {event.target ? ` → ${event.target.display_name}` : ""} ·{" "}
              {event.attribution_count} attribution(s) ·{" "}
              {event.author?.username ?? "utilisateur supprimé"} le{" "}
              {new Date(event.created_at).toLocaleString("fr-FR")}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
