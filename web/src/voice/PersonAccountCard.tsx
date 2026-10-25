import React, { useEffect, useState } from "react";
import {
  voiceApi,
  voiceChecked,
  type CsrfHeaders,
  type Person,
} from "./client";
import "./voice.css";

// Administrator card: link a Person to an account (by username) or remove the link.
export function PersonAccountCard({
  csrf,
}: {
  csrf: CsrfHeaders;
}): React.ReactElement {
  const [persons, setPersons] = useState<Person[]>([]);
  const [personId, setPersonId] = useState("");
  const [username, setUsername] = useState("");
  const [message, setMessage] = useState("");

  async function load() {
    const result = await voiceApi.GET("/api/persons");
    if (result.data) setPersons(result.data.items);
  }
  useEffect(() => {
    void load();
  }, []);

  async function link(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!personId) return;
    try {
      const linked = voiceChecked(
        await voiceApi.PUT("/api/admin/persons/{id}/account", {
          params: { header: csrf, path: { id: Number(personId) } },
          body: { username },
        }),
      );
      setMessage(`${linked.display_name} est liée au compte ${username}.`);
      await load();
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    }
  }

  async function unlink() {
    if (!personId) return;
    try {
      voiceChecked(
        await voiceApi.DELETE("/api/admin/persons/{id}/account", {
          params: { header: csrf, path: { id: Number(personId) } },
        }),
      );
      setMessage("Lien supprimé.");
      await load();
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    }
  }

  return (
    <section className="card" id="person-accounts">
      <h2>Comptes liés aux Personnes</h2>
      <p className="hint">
        Une Personne liée à un compte retrouve ses clips et peut supprimer son
        profil vocal. Une Personne n’est jamais un compte.
      </p>
      <form className="voice-pipeline-form" onSubmit={link}>
        <label>
          Personne à lier
          <select
            value={personId}
            onChange={(event) => setPersonId(event.target.value)}
          >
            <option value="">Choisir</option>
            {persons.map((person) => (
              <option key={person.id} value={String(person.id)}>
                {person.display_name}
                {person.user_id ? " (liée)" : ""}
              </option>
            ))}
          </select>
        </label>
        <label>
          Nom du compte
          <input
            value={username}
            onChange={(event) => setUsername(event.target.value)}
          />
        </label>
        <button type="button" className="secondary" onClick={() => void load()}>
          Recharger les Personnes
        </button>{" "}
        <button disabled={!personId || !username}>Lier au compte</button>{" "}
        <button type="button" className="secondary" onClick={unlink}>
          Délier le compte
        </button>
      </form>
      <p aria-live="polite">{message}</p>
    </section>
  );
}
