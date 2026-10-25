import React, { useCallback, useEffect, useState } from "react";
import {
  errorCode,
  messageFor,
  voiceApi,
  type CsrfHeaders,
  type Person,
} from "./client";
import type { components } from "../api.gen";
import "./groups.css";

type VoiceGroup = components["schemas"]["VoiceGroup"];

// Groups of unknown voices: listen to the extracts, exclude the ones that do not
// belong, then name the whole group as a Person. A group is only a proposal; the
// counts come from the server and nothing is validated before "Nommer le groupe".
export function VoiceGroupsCard({
  csrf,
}: {
  csrf: CsrfHeaders;
}): React.ReactElement {
  const [groups, setGroups] = useState<VoiceGroup[]>([]);
  const [total, setTotal] = useState(0);
  const [proposedOnly, setProposedOnly] = useState(true);
  const [byMembers, setByMembers] = useState(false);
  const [persons, setPersons] = useState<Person[]>([]);
  const [message, setMessage] = useState("");

  const load = useCallback(async () => {
    const [list, people] = await Promise.all([
      voiceApi.GET("/api/voice-groups", {
        params: {
          query: {
            proposed: proposedOnly,
            sort: byMembers ? "members" : "clips",
            page_size: 20,
          },
        },
      }),
      voiceApi.GET("/api/persons"),
    ]);
    if (list.data) {
      setGroups(list.data.items);
      setTotal(list.data.total);
    }
    if (people.data) setPersons(people.data.items);
  }, [proposedOnly, byMembers]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <section className="card" id="voice-groups" aria-label="Groupes de voix">
      <h2>Voix inconnues à regrouper</h2>
      <p className="hint">
        Des voix proches sont proposées ensemble ; rien n’est confirmé tant que
        vous n’avez pas nommé le groupe.
      </p>
      <div className="group-filters">
        <label>
          <input
            type="checkbox"
            checked={proposedOnly}
            onChange={(event) => setProposedOnly(event.target.checked)}
          />{" "}
          Groupes proposés uniquement
        </label>
        <label>
          <input
            type="checkbox"
            checked={byMembers}
            onChange={(event) => setByMembers(event.target.checked)}
          />{" "}
          Trier par nombre de voix
        </label>
      </div>
      {message && (
        <p className="hint" aria-live="polite">
          {message}
        </p>
      )}
      {groups.length === 0 ? (
        <p className="hint">Aucun groupe de voix inconnues.</p>
      ) : (
        <p className="hint">
          {total} groupe(s), du plus fréquent au moins fréquent.
        </p>
      )}
      <ul className="voice-group-list" aria-label="Groupes de voix inconnues">
        {groups.map((group) => (
          <GroupItem
            key={group.id}
            group={group}
            persons={persons}
            csrf={csrf}
            onMessage={setMessage}
            onChanged={load}
          />
        ))}
      </ul>
    </section>
  );
}

function GroupItem({
  group,
  persons,
  csrf,
  onMessage,
  onChanged,
}: {
  group: VoiceGroup;
  persons: Person[];
  csrf: CsrfHeaders;
  onMessage: (text: string) => void;
  onChanged: () => Promise<void>;
}): React.ReactElement {
  const [personId, setPersonId] = useState("");
  const [newName, setNewName] = useState("");
  const [busy, setBusy] = useState(false);
  const members = group.members.filter((m) => !m.excluded);

  async function setExcluded(id: number, excluded: boolean) {
    setBusy(true);
    try {
      const result = excluded
        ? await voiceApi.PUT("/api/local-speakers/{id}/group-exclusion", {
            params: { path: { id }, header: csrf },
          })
        : await voiceApi.DELETE("/api/local-speakers/{id}/group-exclusion", {
            params: { path: { id }, header: csrf },
          });
      onMessage(
        result.error
          ? messageFor(errorCode(result.error))
          : excluded
            ? "Voix exclue du groupe."
            : "Voix réintégrée.",
      );
      await onChanged();
    } finally {
      setBusy(false);
    }
  }

  async function name() {
    setBusy(true);
    try {
      const person =
        newName.trim() !== ""
          ? { new_person_name: newName.trim() }
          : { person_id: Number(personId) };
      const result = await voiceApi.POST("/api/voice-groups/{id}/naming", {
        params: { path: { id: group.id }, header: csrf },
        body: {
          ...person,
          local_speaker_ids: members.map((m) => m.local_speaker_id),
        },
      });
      if (result.error) {
        onMessage(messageFor(errorCode(result.error)));
      } else {
        const { person: named, validated, skipped } = result.data;
        onMessage(
          `Groupe nommé ${named.display_name} : ${validated.length} voix validée(s)${skipped.length ? `, ${skipped.length} déjà validée(s) par quelqu’un d’autre` : ""}.`,
        );
      }
      await onChanged();
    } catch {
      onMessage("La demande a échoué. Réessayez.");
    } finally {
      setBusy(false);
    }
  }

  const singleExcluded =
    group.members.length === 1 && group.members[0].excluded;
  return (
    <li className="voice-group">
      <strong>
        {group.proposed
          ? "Groupe proposé"
          : singleExcluded
            ? "Voix exclue"
            : "Voix seule"}
      </strong>{" "}
      · {group.member_count} voix · {group.clip_count} clip
      {group.clip_count > 1 ? "s" : ""}
      <ul className="voice-group-members">
        {group.members.map((m) => (
          <li key={m.local_speaker_id}>
            {m.label} · {m.clip_name} · {m.speaking_seconds.toFixed(1)} s{" "}
            <audio
              controls
              preload="none"
              aria-label={`Extrait de ${m.label} (${m.clip_name})`}
              src={`/api/local-speakers/${m.local_speaker_id}/excerpt`}
            />{" "}
            {m.excluded ? (
              <button
                className="secondary"
                disabled={busy}
                onClick={() => void setExcluded(m.local_speaker_id, false)}
              >
                Réintégrer {m.label}
              </button>
            ) : (
              group.proposed && (
                <button
                  className="secondary"
                  disabled={busy}
                  onClick={() => void setExcluded(m.local_speaker_id, true)}
                >
                  Exclure {m.label} du groupe
                </button>
              )
            )}
          </li>
        ))}
      </ul>
      {members.length > 0 && (
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
                  {p.display_name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Nom du groupe
            <input
              value={newName}
              maxLength={100}
              onChange={(event) => setNewName(event.target.value)}
            />
          </label>
          <button
            disabled={busy || (newName.trim() === "" && personId === "")}
            onClick={() => void name()}
          >
            Nommer le groupe
          </button>
        </div>
      )}
    </li>
  );
}
