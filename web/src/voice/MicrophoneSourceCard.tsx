import React, { useEffect, useState } from "react";
import type { components } from "../api.gen";
import { voiceApi, type CsrfHeaders, type Person } from "./client";
import {
  OwnerFields,
  checkedMicrophone,
  loadPersons,
  resolveOwner,
} from "./MicrophoneShared";
import "./microphone.css";

type SourceRule = components["schemas"]["MicrophoneSourceRule"];
type Criterion = "index" | "title" | "both";

const outcomeLabels: Record<string, string> = {
  designated: "clip désigné",
  track_not_found: "clip sans piste correspondante",
  ambiguous_track: "clip à piste ambiguë",
  conflicting_sources: "clip aux règles contradictoires",
  blocked_by_validation: "clip dont les voix sont déjà validées",
  owner_profile_deleted: "clip dont le propriétaire a supprimé son profil",
  disabled: "clip sans piste micro",
  none: "clip sans désignation",
};

function describeCounts(counts: Record<string, number>): string {
  const parts = Object.entries(counts).map(
    ([status, count]) => `${count} ${outcomeLabels[status] ?? status}`,
  );
  return parts.length ? parts.join(", ") : "aucun clip prêt dans cette source";
}

// Administrator screen: designates, from the tracks really detected on the source's
// clips, the dedicated microphone track and its owner. No track title is assumed.
export function MicrophoneSourceCard({
  sourceId,
  csrf,
}: {
  sourceId: number;
  csrf: CsrfHeaders;
}): React.ReactElement {
  const [state, setState] = useState<SourceRule | null>(null);
  const [persons, setPersons] = useState<Person[]>([]);
  const [track, setTrack] = useState("");
  const [criterion, setCriterion] = useState<Criterion>("index");
  const [personId, setPersonId] = useState("");
  const [newName, setNewName] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    const [rule, people] = await Promise.all([
      voiceApi.GET("/api/admin/sources/{id}/microphone-track", {
        params: { path: { id: sourceId } },
      }),
      loadPersons(),
    ]);
    setPersons(people);
    if (rule.data) setState(rule.data);
  }
  useEffect(() => {
    void load();
  }, [sourceId]);

  const detected = state?.detected_tracks ?? [];
  const chosen = detected.find(
    (item) => String(item.audio_stream_index) === track,
  );

  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (!chosen) {
      setMessage("Choisissez une piste détectée.");
      return;
    }
    setBusy(true);
    setMessage("");
    try {
      const owner = await resolveOwner(csrf, personId, newName);
      const body: components["schemas"]["MicrophoneTrackRuleInput"] = {
        person_id: owner,
      };
      if (criterion !== "title")
        body.audio_stream_index = chosen.audio_stream_index;
      if (criterion !== "index") {
        if (!chosen.title) throw new Error("Cette piste n’a pas de titre.");
        body.title = chosen.title;
      }
      const result = checkedMicrophone(
        await voiceApi.PUT("/api/admin/sources/{id}/microphone-track", {
          params: { path: { id: sourceId }, header: csrf },
          body,
        }),
      );
      setState(result);
      setPersons(await loadPersons());
      setNewName("");
      setMessage(
        `Piste micro enregistrée : ${describeCounts(result.status_counts)}.`,
      );
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    setBusy(true);
    setMessage("");
    try {
      const result = checkedMicrophone(
        await voiceApi.DELETE("/api/admin/sources/{id}/microphone-track", {
          params: { path: { id: sourceId }, header: csrf },
        }),
      );
      setState(result);
      setMessage(
        `Désignation retirée : ${describeCounts(result.status_counts)}.`,
      );
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    } finally {
      setBusy(false);
    }
  }

  const rule = state?.rule;
  return (
    <details className="source-microphone-track">
      <summary>Piste micro pour cette source</summary>
      <section
        className="microphone-card"
        aria-label="Piste micro de la source"
      >
        <p>
          {rule
            ? `Désignée : ${[
                rule.audio_stream_index !== null
                  ? `piste ${rule.audio_stream_index}`
                  : "",
                rule.title ? `titre « ${rule.title} »` : "",
              ]
                .filter(Boolean)
                .join(" · ")} → ${rule.person.display_name}`
            : "Aucune piste micro désignée."}
        </p>
        {detected.length === 0 ? (
          <p className="hint">
            Aucune piste audio détectée sur les clips prêts.
          </p>
        ) : (
          <ul className="microphone-tracks" aria-label="Pistes détectées">
            {detected.map((item) => (
              <li
                key={`${item.audio_stream_index}-${item.title}-${item.language}`}
              >
                Piste {item.audio_stream_index} · {item.title || "sans titre"} ·{" "}
                {item.language || "langue inconnue"} · {item.clip_count} clip
                {item.clip_count > 1 ? "s" : ""}
              </li>
            ))}
          </ul>
        )}
        {detected.length > 0 && (
          <form onSubmit={save}>
            <label>
              Piste micro détectée
              <select
                value={track}
                onChange={(event) => setTrack(event.target.value)}
              >
                <option value="">Choisir une piste</option>
                {Array.from(
                  new Map(
                    detected.map((item) => [item.audio_stream_index, item]),
                  ).values(),
                ).map((item) => (
                  <option
                    key={item.audio_stream_index}
                    value={item.audio_stream_index}
                  >
                    Piste {item.audio_stream_index} ·{" "}
                    {item.title || "sans titre"}
                  </option>
                ))}
              </select>
            </label>
            <fieldset>
              <legend>Reconnaître la piste par</legend>
              {(
                [
                  ["index", "son numéro"],
                  ["title", "son titre exact"],
                  ["both", "son numéro et son titre"],
                ] as [Criterion, string][]
              ).map(([value, label]) => (
                <label key={value}>
                  <input
                    type="radio"
                    name={`criterion-${sourceId}`}
                    checked={criterion === value}
                    onChange={() => setCriterion(value)}
                  />
                  {label}
                </label>
              ))}
            </fieldset>
            <OwnerFields
              persons={persons}
              personId={personId}
              newName={newName}
              onPerson={setPersonId}
              onNewName={setNewName}
            />
            <button className="secondary" disabled={busy}>
              Désigner la piste micro
            </button>
          </form>
        )}
        {rule && (
          <button className="secondary" disabled={busy} onClick={remove}>
            Retirer la désignation
          </button>
        )}
        {message && (
          <p className="hint" aria-live="polite">
            {message}
          </p>
        )}
      </section>
    </details>
  );
}
