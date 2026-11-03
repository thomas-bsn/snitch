import { useEffect, useId, useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { ExcerptPlayer } from "../people/ExcerptPlayer";
import {
  fetchPersonPage,
  fetchVoiceGroupOf,
  type Person,
  type VoiceGroup,
} from "../people/peopleApi";
import { useSession } from "../session";

/** The voice as the decision needs it: who it is, how it is settled, and its disposition. */
export type DecidedVoice = components["schemas"]["LocalSpeaker"];

/** A disposition of a voice (ADR 0031), as the generated contract types it. */
export type Disposition =
  components["schemas"]["VoiceDispositionInput"]["disposition"];
/** The dispositions by name, checked against the generated enum. */
export const DISPOSITION = {
  ignored: "ignored",
  random: "random",
} as const satisfies { [D in Disposition]: D };

type Outcome = { data?: unknown; error?: unknown };

const SEARCH_DELAY_MS = 250;
const SEARCH_SIZE = 8;
const fieldClass =
  "h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10";

/**
 * Every decision on one voice (ADR 0031 and the human validation): "C'est X", "Ce n'est pas X",
 * Identifier, Random, Ignorer, Rétablir, "Annuler le refus" of each refused Person and the excerpt. Shared by the clip's Voices tab and the
 * voice processing screen; it writes through the existing calls and reports the change to
 * `onChanged`, which reloads whatever it shows.
 */
export function VoiceDecision({
  speaker,
  clipName,
  onChanged,
}: {
  speaker: DecidedVoice;
  /** Name of the clip the excerpt comes from. */
  clipName: string;
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const header = mutation(session?.csrf_token);
  const fieldId = useId();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [identifying, setIdentifying] = useState(false);
  const [query, setQuery] = useState("");
  const [found, setFound] = useState<Person[]>([]);
  const [newName, setNewName] = useState("");
  const [group, setGroup] = useState<VoiceGroup | null>(null);

  const person = speaker.person ?? null;
  const validated = speaker.attribution?.validation_state === "validated";
  const disposition = speaker.disposition ?? null;
  const label = speaker.label;
  const refused = speaker.refused_persons ?? [];

  // The voice's group of unknown voices, when it has one with other voices in it.
  useEffect(() => {
    let current = true;
    void fetchVoiceGroupOf(speaker.id).then((result) => {
      if (current && result.ok) setGroup(result.value);
    });
    return () => {
      current = false;
    };
  }, [speaker.id]);

  // The people search is done by the server, after a short pause in the typing.
  useEffect(() => {
    if (!identifying) return;
    let current = true;
    const timer = setTimeout(() => {
      void fetchPersonPage(query.trim(), 1, SEARCH_SIZE).then((result) => {
        if (current && result.ok) setFound(result.value.items);
      });
    }, SEARCH_DELAY_MS);
    return () => {
      current = false;
      clearTimeout(timer);
    };
  }, [query, identifying]);

  async function act(request: () => Promise<Outcome>) {
    setBusy(true);
    setFailure(undefined);
    try {
      const outcome = await request();
      if (outcome.error) setFailure(errorCode(outcome.error) ?? "unknown");
      else {
        setIdentifying(false);
        setQuery("");
        setNewName("");
        onChanged();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  /** Identifying a voice that is already validated corrects it; the expected Person guards the change. */
  function identify(choice: { personId: number } | { newName: string }) {
    const path = { path: { id: speaker.id }, header };
    if (validated && person) {
      void act(() =>
        api.POST("/api/local-speakers/{id}/correction", {
          params: path,
          body:
            "personId" in choice
              ? { person_id: choice.personId, expected_person_id: person.id }
              : {
                  new_person_name: choice.newName,
                  expected_person_id: person.id,
                },
        }),
      );
      return;
    }
    void act(() =>
      api.POST("/api/local-speakers/{id}/validation", {
        params: path,
        body:
          "personId" in choice
            ? { person_id: choice.personId }
            : { new_person_name: choice.newName },
      }),
    );
  }

  const shownPeople = found.filter((candidate) => candidate.id !== person?.id);
  const name = newName.trim();
  const groupId = `${fieldId}-group`;

  return (
    <div className="flex flex-col gap-2">
      <div
        role="group"
        aria-label={t("voiceDecision.decisionLabel", { label })}
        className="flex flex-wrap items-center gap-2"
      >
        {person && !validated && (
          <button
            type="button"
            disabled={busy}
            onClick={() => identify({ personId: person.id })}
            className={buttonClass("primary")}
          >
            {t("voiceDecision.yes", { person: person.display_name })}
          </button>
        )}
        {person && (
          <button
            type="button"
            disabled={busy}
            onClick={() =>
              void act(() =>
                api.POST("/api/local-speakers/{id}/refusal", {
                  params: { path: { id: speaker.id }, header },
                  body: { person_id: person.id },
                }),
              )
            }
            className={buttonClass()}
          >
            {t("voiceDecision.no", { person: person.display_name })}
          </button>
        )}
        <button
          type="button"
          aria-expanded={identifying}
          aria-controls={groupId}
          onClick={() => setIdentifying((open) => !open)}
          className={buttonClass()}
        >
          {t("voiceDecision.identify")}
        </button>
        {disposition !== DISPOSITION.random && (
          <button
            type="button"
            disabled={busy}
            onClick={() =>
              void act(() =>
                api.PUT("/api/local-speakers/{id}/disposition", {
                  params: { path: { id: speaker.id }, header },
                  body: { disposition: DISPOSITION.random },
                }),
              )
            }
            className={buttonClass()}
          >
            {t("voiceDecision.random")}
          </button>
        )}
        {disposition !== DISPOSITION.ignored && (
          <button
            type="button"
            disabled={busy}
            onClick={() =>
              void act(() =>
                api.PUT("/api/local-speakers/{id}/disposition", {
                  params: { path: { id: speaker.id }, header },
                  body: { disposition: DISPOSITION.ignored },
                }),
              )
            }
            className={buttonClass()}
          >
            {t("voiceDecision.ignore")}
          </button>
        )}
        {disposition && (
          <button
            type="button"
            disabled={busy}
            onClick={() =>
              void act(() =>
                api.DELETE("/api/local-speakers/{id}/disposition", {
                  params: { path: { id: speaker.id }, header },
                }),
              )
            }
            className={buttonClass()}
          >
            {t("voiceDecision.restore")}
          </button>
        )}
        <ExcerptPlayer
          localSpeakerId={speaker.id}
          label={label}
          clip={clipName}
        />
      </div>

      {identifying && (
        <div
          id={groupId}
          className="flex flex-col gap-2 rounded-lg border border-line p-3"
        >
          <label htmlFor={`${fieldId}-search`} className="font-medium">
            {t("voiceDecision.identifyLabel")}
          </label>
          <input
            id={`${fieldId}-search`}
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            aria-label={t("voiceDecision.searchLabel")}
            placeholder={t("voiceDecision.search")}
            className={fieldClass}
          />
          <div className="flex flex-wrap gap-2">
            {shownPeople.map((candidate) => (
              <button
                key={candidate.id}
                type="button"
                disabled={busy}
                onClick={() => identify({ personId: candidate.id })}
                className={buttonClass()}
              >
                {t("voiceDecision.asPerson", {
                  person: candidate.display_name,
                })}
              </button>
            ))}
            {query.trim() !== "" && shownPeople.length === 0 && (
              <p className="m-0 text-mute">{t("voiceDecision.noMatch")}</p>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <input
              type="text"
              value={newName}
              maxLength={100}
              onChange={(event) => setNewName(event.target.value)}
              aria-label={t("voiceDecision.createName")}
              autoComplete="off"
              className={`${fieldClass} min-w-0 flex-1`}
            />
            <button
              type="button"
              disabled={busy || name === ""}
              onClick={() => identify({ newName: name })}
              className={buttonClass()}
            >
              {t("voiceDecision.create")}
            </button>
          </div>
        </div>
      )}

      {refused.length > 0 && (
        <ul
          aria-label={t("voiceDecision.refusedLabel", { label })}
          className="m-0 flex list-none flex-wrap gap-2 p-0"
        >
          {refused.map((refusedPerson) => (
            <li
              key={refusedPerson.id}
              className="flex items-center gap-2 text-sm text-mute"
            >
              <span>
                {t("voiceDecision.refused", {
                  person: refusedPerson.display_name,
                })}
              </span>
              <button
                type="button"
                disabled={busy}
                aria-label={t("voiceDecision.cancelRefusalLabel", {
                  person: refusedPerson.display_name,
                })}
                onClick={() =>
                  void act(() =>
                    api.DELETE(
                      "/api/local-speakers/{id}/refusals/{person_id}",
                      {
                        params: {
                          path: { id: speaker.id, person_id: refusedPerson.id },
                          header,
                        },
                      },
                    ),
                  )
                }
                className={buttonClass()}
              >
                {t("voiceDecision.cancelRefusal")}
              </button>
            </li>
          ))}
        </ul>
      )}
      {validated && (
        <p className="m-0 text-xs text-mute">
          {t("voiceDecision.validatedHint")}
        </p>
      )}
      {group && group.member_count > 1 && (
        <p className="m-0 text-xs text-mute">
          {t("voiceDecision.group", { count: group.member_count })}
        </p>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
    </div>
  );
}
