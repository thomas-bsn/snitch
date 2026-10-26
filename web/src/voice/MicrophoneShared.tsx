import React from "react";
import {
  voiceApi,
  voiceChecked,
  type CsrfHeaders,
  type Person,
} from "./client";
import "./microphone.css";

export type MicrophoneTrackStatus =
  | "none"
  | "designated"
  | "disabled"
  | "track_not_found"
  | "ambiguous_track"
  | "conflicting_sources"
  | "blocked_by_validation"
  | "owner_profile_deleted";

export const microphoneStatusLabels: Record<MicrophoneTrackStatus, string> = {
  none: "Aucune piste micro désignée",
  designated: "Piste micro désignée",
  disabled: "Pas de piste micro pour ce clip",
  track_not_found:
    "La désignation ne correspond à aucune piste de ce clip : rien n’est attribué.",
  ambiguous_track:
    "Plusieurs pistes correspondent à la désignation : précisez le numéro de piste.",
  conflicting_sources:
    "Les règles de plusieurs sources se contredisent pour ce clip : rien n’est attribué.",
  blocked_by_validation:
    "La piste désignée porte des voix déjà validées par un humain : elle n’est pas remplacée.",
  owner_profile_deleted:
    "Le propriétaire désigné a supprimé son profil vocal : rien ne lui est attribué.",
};

export const microphoneMessages: Record<string, string> = {
  invalid_microphone_rule:
    "Choisissez un numéro de piste ou un titre, et une Personne propriétaire.",
  invalid_microphone_override:
    "Indiquez soit « pas de piste micro », soit une piste et une Personne.",
  audio_stream_not_found: "Cette piste n’existe pas dans ce clip.",
  person_not_found: "Cette Personne n’existe plus.",
  source_not_found: "Cette source n’existe plus.",
  clip_not_found: "Ce clip n’existe plus.",
  microphone_track_not_diarized:
    "Cette piste est la piste micro : elle est attribuée sans diarisation.",
};

export function checkedMicrophone<T>(result: {
  data?: T;
  error?: { error: string };
}): T {
  if (result.error)
    throw new Error(
      microphoneMessages[result.error.error] ??
        "La demande a échoué. Réessayez.",
    );
  if (!result.data) throw new Error("Réponse du serveur absente.");
  return result.data;
}

export async function loadPersons(): Promise<Person[]> {
  const result = await voiceApi.GET("/api/persons", { params: {} });
  return result.data?.items ?? [];
}

// Returns the owner id: the chosen Person, or a Person created from the typed name.
export async function resolveOwner(
  csrf: CsrfHeaders,
  personId: string,
  newName: string,
): Promise<number> {
  const name = newName.trim();
  if (name) {
    const created = voiceChecked(
      await voiceApi.POST("/api/persons", {
        params: { header: csrf },
        body: { display_name: name },
      }),
    );
    return created.id;
  }
  if (!personId)
    throw new Error("Choisissez ou créez la Personne propriétaire.");
  return Number(personId);
}

export function OwnerFields({
  persons,
  personId,
  newName,
  onPerson,
  onNewName,
}: {
  persons: Person[];
  personId: string;
  newName: string;
  onPerson: (value: string) => void;
  onNewName: (value: string) => void;
}): React.ReactElement {
  return (
    <>
      <label>
        Propriétaire du micro
        <select
          value={personId}
          onChange={(event) => onPerson(event.target.value)}
        >
          <option value="">Choisir une Personne</option>
          {persons.map((person) => (
            <option key={person.id} value={person.id}>
              {person.display_name}
            </option>
          ))}
        </select>
      </label>
      <label>
        Nouvelle Personne propriétaire
        <input
          value={newName}
          maxLength={100}
          onChange={(event) => onNewName(event.target.value)}
        />
      </label>
    </>
  );
}
