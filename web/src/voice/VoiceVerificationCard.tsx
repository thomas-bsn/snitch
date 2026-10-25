import React, { useCallback, useEffect, useState } from "react";
import {
  errorCode,
  levelLabels,
  messageFor,
  voiceApi,
  type Attribution,
  type CsrfHeaders,
  type DecisionLevel,
  type Person,
} from "./client";
import { LevelBadge } from "./SpeakerRow";
import type { components } from "../api.gen";
import "./verification.css";

type Verification = components["schemas"]["VoiceVerification"];
type VerificationCase = components["schemas"]["VoiceVerificationCase"];
type Level = "all" | "unknown" | "ambiguous";

const PAGE_SIZE = 10;
const LEVEL_ORDER: DecisionLevel[] = [
  "human",
  "automatic",
  "suggestion",
  "ambiguous",
  "unknown",
];

// Review of unknown and ambiguous voices, most frequent first. Settling a case is the
// ordinary human validation; counters are read back from the server afterwards. The
// numbers describe how many voices are in each state, never how reliable they are.
export function VoiceVerificationCard({
  csrf,
}: {
  csrf: CsrfHeaders;
}): React.ReactElement {
  const [data, setData] = useState<Verification | null>(null);
  const [level, setLevel] = useState<Level>("all");
  const [page, setPage] = useState(1);
  const [persons, setPersons] = useState<Person[]>([]);
  const [message, setMessage] = useState("");

  const load = useCallback(async () => {
    const [list, people] = await Promise.all([
      voiceApi.GET("/api/voice-verification", {
        params: { query: { level, page, page_size: PAGE_SIZE } },
      }),
      voiceApi.GET("/api/persons"),
    ]);
    if (list.data) setData(list.data);
    if (people.data) setPersons(people.data.items);
  }, [level, page]);

  useEffect(() => {
    void load();
  }, [load]);

  const items = data?.items ?? [];
  const total = data?.total ?? 0;
  const lastPage = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <section
      className="card"
      id="voice-verification"
      aria-label="Vérification des voix"
    >
      <h2>Vérification des voix</h2>
      <p className="hint">
        Seules les voix inconnues ou ambiguës sont à trancher, les plus
        fréquentes d’abord. Trancher un cas crée une validation humaine.
      </p>
      {data && (
        <ul className="verification-counts" aria-label="Voix par état">
          {LEVEL_ORDER.map((state) => (
            <li key={state}>
              <LevelBadge level={state} /> <strong>{data.counts[state]}</strong>
            </li>
          ))}
        </ul>
      )}
      <div className="group-filters">
        <label>
          Cas affichés
          <select
            value={level}
            onChange={(event) => {
              setLevel(event.target.value as Level);
              setPage(1);
            }}
          >
            <option value="all">Inconnus et ambigus</option>
            <option value="unknown">Inconnus seulement</option>
            <option value="ambiguous">Ambigus seulement</option>
          </select>
        </label>
      </div>
      {message && (
        <p className="hint" aria-live="polite">
          {message}
        </p>
      )}
      <p className="hint">
        {total === 0
          ? "Aucune voix à vérifier."
          : `${total} cas à vérifier (page ${page} sur ${lastPage}).`}
      </p>
      <ul className="verification-list" aria-label="Cas à vérifier">
        {items.map((item) => (
          <CaseItem
            key={item.local_speaker_id}
            item={item}
            persons={persons}
            csrf={csrf}
            onMessage={setMessage}
            onChanged={load}
          />
        ))}
      </ul>
      {lastPage > 1 && (
        <div className="group-filters">
          <button
            className="secondary"
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
          >
            Cas plus fréquents
          </button>
          <button
            className="secondary"
            disabled={page >= lastPage}
            onClick={() => setPage(page + 1)}
          >
            Cas moins fréquents
          </button>
        </div>
      )}
    </section>
  );
}

function CaseItem({
  item,
  persons,
  csrf,
  onMessage,
  onChanged,
}: {
  item: VerificationCase;
  persons: Person[];
  csrf: CsrfHeaders;
  onMessage: (text: string) => void;
  onChanged: () => Promise<void>;
}): React.ReactElement {
  const [personId, setPersonId] = useState("");
  const [newName, setNewName] = useState("");
  const [busy, setBusy] = useState(false);
  const level: DecisionLevel = item.decision_level;

  async function settle(
    person: { person_id: number } | { new_person_name: string },
  ) {
    setBusy(true);
    try {
      const result = await voiceApi.POST(
        "/api/local-speakers/{id}/validation",
        {
          params: { path: { id: item.local_speaker_id }, header: csrf },
          body: person,
        },
      );
      if (result.error) {
        const conflict = result.error as {
          error: string;
          attribution?: Attribution;
        };
        onMessage(
          conflict.error === "already_validated" && conflict.attribution
            ? `Déjà validé : ${conflict.attribution.person?.display_name ?? "?"}. La première validation fait foi.`
            : messageFor(conflict.error || errorCode(result.error)),
        );
      } else {
        onMessage(
          `${item.label} (${item.clip_name}) : cas tranché, validation humaine enregistrée.`,
        );
      }
      await onChanged();
    } catch {
      onMessage("La demande a échoué. Réessayez.");
    } finally {
      setBusy(false);
    }
  }

  const choice =
    newName.trim() !== ""
      ? { new_person_name: newName.trim() }
      : { person_id: Number(personId) };
  return (
    <li className={`verification-case level-case-${level}`}>
      <div>
        <LevelBadge level={level} /> <strong>{item.label}</strong> ·{" "}
        {item.clip_name} · {item.speaking_seconds.toFixed(1)} s de parole
      </div>
      <div className="hint">
        {item.group_voice_count > 1
          ? `Groupe proposé : ${item.group_voice_count} voix dans ${item.group_clip_count} clip${item.group_clip_count > 1 ? "s" : ""}.`
          : "Voix seule."}{" "}
        {levelLabels[level]} : à confirmer par vous.
      </div>
      {item.candidates.length > 0 && (
        <ul aria-label={`Candidates pour ${item.label}`}>
          {item.candidates.map((candidate) => (
            <li key={candidate.person.id}>
              {candidate.person.display_name} · similarité{" "}
              {Math.round(candidate.score * 100)} %{" "}
              <button
                className="secondary"
                disabled={busy}
                onClick={() => void settle({ person_id: candidate.person.id })}
              >
                Trancher {item.label} : {candidate.person.display_name}
              </button>
            </li>
          ))}
        </ul>
      )}
      <audio
        controls
        preload="none"
        aria-label={`Extrait à vérifier de ${item.label} (${item.clip_name})`}
        src={`/api/local-speakers/${item.local_speaker_id}/excerpt`}
      />
      <div className="speaker-validation">
        <label>
          Personne pour {item.label}
          <select
            value={personId}
            disabled={newName.trim() !== ""}
            onChange={(event) => setPersonId(event.target.value)}
          >
            <option value="">Choisir…</option>
            {persons.map((p) => (
              <option
                key={p.id}
                value={p.id}
                disabled={p.voice_profile_deleted_at !== null}
              >
                {p.display_name}
                {p.voice_profile_deleted_at !== null
                  ? " (profil vocal supprimé)"
                  : ""}
              </option>
            ))}
          </select>
        </label>
        <label>
          Nouvelle Personne pour {item.label}
          <input
            value={newName}
            maxLength={100}
            onChange={(event) => setNewName(event.target.value)}
          />
        </label>
        <button
          disabled={busy || (newName.trim() === "" && personId === "")}
          onClick={() => void settle(choice)}
        >
          Trancher {item.label}
        </button>
      </div>
    </li>
  );
}
