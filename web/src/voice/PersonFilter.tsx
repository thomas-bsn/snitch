import { useEffect, useState } from "react";
import { LevelBadge } from "./SpeakerRow";
import { voiceApi, type DecisionLevel } from "./client";

// Person filter shared by transcript search and the library. The value is ""
// (no filter), "unknown" (voices without an identified Person) or a Person id.
export const unknownPerson = "unknown";

export function applyPersonFilter(query: URLSearchParams, value: string) {
  if (value === unknownPerson) query.set("person_unknown", "true");
  else if (value) query.set("person_id", value);
}

export function PersonFilterSelect({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}): React.ReactElement {
  const [persons, setPersons] = useState<
    { id: number; display_name: string }[]
  >([]);
  useEffect(() => {
    let live = true;
    void voiceApi.GET("/api/persons").then((result) => {
      if (live && result.data) setPersons(result.data.items);
    });
    return () => {
      live = false;
    };
  }, []);
  return (
    <label>
      Personne
      <select
        aria-label={label}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">Toutes</option>
        <option value={unknownPerson}>Voix inconnues</option>
        {persons.map((person) => (
          <option key={person.id} value={String(person.id)}>
            {person.display_name}
          </option>
        ))}
      </select>
    </label>
  );
}

// The Person (or the unknown state) of the voice speaking a search result.
export function ResultSpeaker({
  person,
  level,
}: {
  person?: { id: number; display_name: string } | null;
  level?: DecisionLevel;
}): React.ReactElement {
  return (
    <span className="result-speaker">
      Personne : {person ? <strong>{person.display_name}</strong> : "inconnue"}
      {level && level !== "unknown" && (
        <>
          {" "}
          <LevelBadge level={level} />
        </>
      )}
    </span>
  );
}

// Persons identified in a clip, shown on library cards and on the clip page.
export function ClipPersons({
  persons,
}: {
  persons?: {
    id: number;
    display_name: string;
    decision_level: DecisionLevel;
  }[];
}): React.ReactElement | null {
  if (!persons || persons.length === 0) return null;
  return (
    <ul className="clip-persons" aria-label="Personnes identifiées">
      {persons.map((person) => (
        <li key={person.id}>
          {person.display_name} <LevelBadge level={person.decision_level} />
        </li>
      ))}
    </ul>
  );
}
