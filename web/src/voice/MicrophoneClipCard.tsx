import React, { useEffect, useState } from "react";
import type { components } from "../api.gen";
import { voiceApi, type CsrfHeaders, type Person } from "./client";
import {
  OwnerFields,
  checkedMicrophone,
  loadPersons,
  microphoneStatusLabels,
  resolveOwner,
} from "./MicrophoneShared";
import "./microphone.css";

type ClipMicrophone = components["schemas"]["ClipMicrophoneTrack"];
type Snapshot = components["schemas"]["MicrophoneSnapshot"] | null | undefined;

const roleLabels: Record<string, string> = {
  microphone: "piste micro",
  candidate: "piste candidate",
  other: "traitée normalement",
};

const actionLabels: Record<string, string> = {
  set: "choix enregistré",
  disable: "pas de piste micro",
  remove: "choix retiré",
};

function describe(snapshot: Snapshot): string {
  if (!snapshot) return "aucun";
  if (snapshot.disabled) return "pas de piste micro";
  const parts = [
    snapshot.audio_stream_index !== undefined
      ? `piste ${snapshot.audio_stream_index}`
      : "",
    snapshot.title ? `titre « ${snapshot.title} »` : "",
    snapshot.person ? `→ ${snapshot.person.display_name}` : "",
  ];
  return parts.filter(Boolean).join(" ");
}

// Shows how the clip's audio tracks are treated and lets an administrator replace the
// source's microphone choice for this clip. Every change is listed in the history.
export function MicrophoneClipCard({
  clipId,
  isAdmin,
  csrf,
  onChanged,
}: {
  clipId: number;
  isAdmin: boolean;
  csrf: CsrfHeaders;
  onChanged?: () => void;
}): React.ReactElement {
  const [view, setView] = useState<ClipMicrophone | null>(null);
  const [persons, setPersons] = useState<Person[]>([]);
  const [track, setTrack] = useState("");
  const [personId, setPersonId] = useState("");
  const [newName, setNewName] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    const result = await voiceApi.GET("/api/clips/{id}/microphone-track", {
      params: { path: { id: clipId } },
    });
    if (result.data) setView(result.data);
    if (isAdmin) setPersons(await loadPersons());
  }
  useEffect(() => {
    void load();
  }, [clipId]);

  async function change(run: () => Promise<ClipMicrophone>, done: string) {
    setBusy(true);
    setMessage("");
    try {
      setView(await run());
      setMessage(done);
      if (isAdmin) setPersons(await loadPersons());
      onChanged?.();
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : "Échec.");
    } finally {
      setBusy(false);
    }
  }

  function replace(event: React.FormEvent) {
    event.preventDefault();
    if (!track) {
      setMessage("Choisissez une piste.");
      return;
    }
    void change(async () => {
      const owner = await resolveOwner(csrf, personId, newName);
      setNewName("");
      return checkedMicrophone(
        await voiceApi.PUT("/api/admin/clips/{id}/microphone-track", {
          params: { path: { id: clipId }, header: csrf },
          body: { audio_stream_index: Number(track), person_id: owner },
        }),
      );
    }, "Choix remplacé pour ce clip.");
  }

  const tracks = view?.tracks ?? [];
  return (
    <section className="microphone-card" aria-label="Piste micro du clip">
      <h3>Piste micro</h3>
      {view && (
        <p
          className={
            [
              "track_not_found",
              "ambiguous_track",
              "conflicting_sources",
              "blocked_by_validation",
              "owner_profile_deleted",
            ].includes(view.status)
              ? "microphone-warning"
              : undefined
          }
        >
          {microphoneStatusLabels[view.status]}
        </p>
      )}
      {view?.person && view.status === "designated" && (
        <p>
          Propriétaire : {view.person.display_name} (
          {view.selection_source === "clip_override"
            ? "choix propre à ce clip"
            : "règle de la source"}
          )
        </p>
      )}
      {tracks.length > 0 && (
        <ul className="microphone-tracks" aria-label="Pistes audio du clip">
          {tracks.map((item) => (
            <li key={item.index}>
              Piste {item.index} · {item.title || "sans titre"} ·{" "}
              <span className={`microphone-role role-${item.role}`}>
                {roleLabels[item.role]}
              </span>
            </li>
          ))}
        </ul>
      )}
      {isAdmin && (
        <form onSubmit={replace}>
          <label>
            Piste micro de ce clip
            <select
              value={track}
              onChange={(event) => setTrack(event.target.value)}
            >
              <option value="">Choisir une piste</option>
              {tracks.map((item) => (
                <option key={item.index} value={item.index}>
                  Piste {item.index} · {item.title || "sans titre"}
                </option>
              ))}
            </select>
          </label>
          <OwnerFields
            persons={persons}
            personId={personId}
            newName={newName}
            onPerson={setPersonId}
            onNewName={setNewName}
          />
          <button className="secondary" disabled={busy}>
            Remplacer pour ce clip
          </button>
          <button
            type="button"
            className="secondary"
            disabled={busy}
            onClick={() =>
              void change(
                async () =>
                  checkedMicrophone(
                    await voiceApi.PUT(
                      "/api/admin/clips/{id}/microphone-track",
                      {
                        params: { path: { id: clipId }, header: csrf },
                        body: { disabled: true },
                      },
                    ),
                  ),
                "Ce clip n’a pas de piste micro.",
              )
            }
          >
            Pas de piste micro pour ce clip
          </button>
          {view?.selection_source === "clip_override" && (
            <button
              type="button"
              className="secondary"
              disabled={busy}
              onClick={() =>
                void change(
                  async () =>
                    checkedMicrophone(
                      await voiceApi.DELETE(
                        "/api/admin/clips/{id}/microphone-track",
                        { params: { path: { id: clipId }, header: csrf } },
                      ),
                    ),
                  "Règle de la source rétablie.",
                )
              }
            >
              Revenir à la règle de la source
            </button>
          )}
        </form>
      )}
      {message && (
        <p className="hint" aria-live="polite">
          {message}
        </p>
      )}
      {view && view.history.length > 0 && (
        <>
          <h4>Historique de la piste micro</h4>
          <ul
            className="microphone-history"
            aria-label="Historique de la piste micro"
          >
            {view.history.map((event) => (
              <li key={event.id}>
                {new Date(event.created_at).toLocaleString("fr-FR")} ·{" "}
                {event.scope === "clip" ? "ce clip" : "source"} ·{" "}
                {actionLabels[event.action]} : {describe(event.old_rule)} →{" "}
                {describe(event.new_rule)}
                {event.author ? ` · par ${event.author.username}` : ""}
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
}
