import {
  Check,
  ChevronDown,
  Dices,
  EyeOff,
  RotateCcw,
  UserPlus,
  UserX,
  Users,
} from "lucide-react";
import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { ExcerptPlayer } from "../people/ExcerptPlayer";
import { useFailureText } from "../people/failureText";
import { initials } from "../people/PersonParts";
import {
  fetchPersonPage,
  fetchVoiceGroupOf,
  type Person,
  type VoiceGroup,
} from "../people/peopleApi";
import { useSession } from "../session";
import { isInteractive, isTextField } from "./shortcuts";

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

/** A key hint on a button of the processing screen; hidden from assistive technology (aria-keyshortcuts says it). */
function Key({ children }: { children: ReactNode }) {
  return (
    <kbd aria-hidden="true" className="ml-1 font-sans text-[10px] opacity-70">
      {children}
    </kbd>
  );
}

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
  shortcuts = false,
  showCandidates = false,
  triage = false,
  trailing,
  excerpt = true,
}: {
  /** Whether the excerpt player closes the decision; false when the screen places it itself. */
  excerpt?: boolean;
  /** The layout of the processing screen: suggestion card, key hints, Random and false detection on a row of their own. */
  triage?: boolean;
  /** What closes the row of Random and false detection (the "Skip" button). */
  trailing?: ReactNode;
  /** O, N, R, I and Space press the matching button, except inside a text field. */
  shortcuts?: boolean;
  /** Lists the closest people the server proposes, each with its proximity. */
  showCandidates?: boolean;
  speaker: DecidedVoice;
  /** Name of the clip the excerpt comes from. */
  clipName: string;
  onChanged: () => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const { session } = useSession();
  const header = mutation(session?.csrf_token);
  const fieldId = useId();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  // Persons that already bear the name typed for a new one (409 person_exists).
  const [duplicates, setDuplicates] = useState<
    { id: number; display_name: string }[]
  >([]);
  const [identifying, setIdentifying] = useState(false);
  const [query, setQuery] = useState("");
  const [found, setFound] = useState<Person[]>([]);
  const [newName, setNewName] = useState("");
  const [group, setGroup] = useState<VoiceGroup | null>(null);
  const root = useRef<HTMLDivElement>(null);

  const person = speaker.person ?? null;
  const validated = speaker.attribution?.validation_state === "validated";
  const disposition = speaker.disposition ?? null;
  const label = speaker.label;
  const refused = speaker.refused_persons ?? [];

  // Shortcuts press the buttons of this decision, so each rule stays where it is.
  useEffect(() => {
    if (!shortcuts) return;
    const listener = (event: KeyboardEvent) => {
      if (
        event.defaultPrevented ||
        event.ctrlKey ||
        event.metaKey ||
        event.altKey ||
        isTextField(event.target)
      )
        return;
      const key = event.key.length === 1 ? event.key.toLowerCase() : event.key;
      if (!["o", "n", "r", "i", " "].includes(key)) return;
      if (key === " " && isInteractive(event.target)) return;
      const button = root.current?.querySelector<HTMLButtonElement>(
        `[data-shortcut="${key}"]`,
      );
      if (!button || button.disabled) return;
      event.preventDefault();
      button.click();
    };
    document.addEventListener("keydown", listener);
    return () => document.removeEventListener("keydown", listener);
  }, [shortcuts]);

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
    setDuplicates([]);
    try {
      const outcome = await request();
      if (outcome.error) {
        // The first validation stands: when the server says who won, the message names them.
        const code = errorCode(outcome.error) ?? "unknown";
        const winner = (
          outcome.error as {
            attribution?: { person?: { display_name: string } };
          }
        ).attribution?.person?.display_name;
        if (code === "person_exists") {
          setDuplicates(
            (
              outcome.error as {
                candidates?: { id: number; display_name: string }[];
              }
            ).candidates ?? [],
          );
        }
        setFailure(
          code === "already_validated" && winner
            ? `already_validated:${winner}`
            : code,
        );
      } else {
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
  function identify(
    choice: { personId: number } | { newName: string },
    allowHomonym = false,
  ) {
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
                  allow_homonym: allowHomonym || undefined,
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
            : {
                new_person_name: choice.newName,
                allow_homonym: allowHomonym || undefined,
              },
      }),
    );
  }

  const shownPeople = found.filter((candidate) => candidate.id !== person?.id);
  const name = newName.trim();
  const groupId = `${fieldId}-group`;

  const proposed = showCandidates
    ? (speaker.attribution?.candidates ?? []).filter(
        (candidate) => candidate.person.id !== person?.id,
      )
    : [];

  const yesLabel = person
    ? t("voiceDecision.yes", { person: person.display_name })
    : "";
  const yesButton = person && !validated && (
    <button
      type="button"
      disabled={busy}
      onClick={() => identify({ personId: person.id })}
      data-shortcut="o"
      aria-keyshortcuts={shortcuts ? "O" : undefined}
      className={buttonClass("primary")}
    >
      {triage ? (
        <>
          <Check className="size-4" aria-hidden="true" />
          <span aria-hidden="true">{t("voiceDecision.yesShort")}</span>
          <span className="sr-only">{yesLabel}</span>
          <Key>O</Key>
        </>
      ) : (
        yesLabel
      )}
    </button>
  );
  const noLabel = person
    ? t("voiceDecision.no", { person: person.display_name })
    : "";
  const noButton = person && (
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
      data-shortcut="n"
      aria-keyshortcuts={shortcuts ? "N" : undefined}
      className={buttonClass()}
    >
      {triage ? (
        <>
          <UserX className="size-4" aria-hidden="true" />
          <span aria-hidden="true">{t("voiceDecision.noShort")}</span>
          <span className="sr-only">{noLabel}</span>
          <Key>N</Key>
        </>
      ) : (
        noLabel
      )}
    </button>
  );
  const identifyButton = (
    <button
      type="button"
      aria-expanded={identifying}
      aria-controls={groupId}
      onClick={() => setIdentifying((open) => !open)}
      className={
        triage
          ? "inline-flex min-h-11 items-center gap-1 rounded-full border border-dashed border-line-strong px-3 text-sm text-mute hover:text-ink md:min-h-9"
          : buttonClass(person ? "secondary" : "primary")
      }
    >
      {triage && <UserPlus className="size-4" aria-hidden="true" />}
      {t("voiceDecision.identify")}
      {!triage && <ChevronDown className="size-4" aria-hidden="true" />}
    </button>
  );
  const randomButton = disposition !== DISPOSITION.random && (
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
      data-shortcut="r"
      aria-keyshortcuts={shortcuts ? "R" : undefined}
      className={buttonClass()}
    >
      <Dices className="size-4" aria-hidden="true" />
      {t("voiceDecision.random")}
      {triage && <Key>R</Key>}
    </button>
  );
  const ignoreButton = disposition !== DISPOSITION.ignored && (
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
      data-shortcut="i"
      aria-keyshortcuts={shortcuts ? "I" : undefined}
      className={buttonClass()}
    >
      <EyeOff className="size-4" aria-hidden="true" />
      {t(triage ? "voiceDecision.falseDetection" : "voiceDecision.ignore")}
      {triage && <Key>I</Key>}
    </button>
  );
  const restoreButton = disposition && (
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
      <RotateCcw className="size-4" aria-hidden="true" />
      {t("voiceDecision.restore")}
    </button>
  );
  const player = (
    <ExcerptPlayer
      localSpeakerId={speaker.id}
      label={label}
      clip={clipName}
      shortcut={shortcuts ? " " : undefined}
    />
  );
  const confidence = speaker.attribution?.confidence;

  return (
    <div ref={root} className={`flex flex-col ${triage ? "gap-5" : "gap-2"}`}>
      {triage && person && !validated && (
        <div className="flex flex-wrap items-center gap-4 rounded-xl bg-accent-soft p-4">
          <span
            aria-hidden="true"
            className="flex size-12 shrink-0 items-center justify-center rounded-full bg-avatar-bg text-base font-semibold text-accent-ink"
          >
            {initials(person.display_name)}
          </span>
          <div className="min-w-0 flex-1">
            <p className="m-0 text-xs text-accent-ink">
              {t("voiceDecision.looksLike")}
            </p>
            <p className="m-0 text-lg font-semibold">
              {person.display_name}
              {typeof confidence === "number" && (
                <span className="ml-1 text-sm font-normal text-mute">
                  ·{" "}
                  {t("voiceDecision.similarity", {
                    percent: Math.round(confidence * 100),
                  })}
                </span>
              )}
            </p>
          </div>
          {yesButton}
          {noButton}
        </div>
      )}
      {proposed.length > 0 && (
        <ul
          aria-label={t("voiceDecision.candidates")}
          className={`m-0 flex list-none flex-wrap p-0 ${triage ? "gap-2" : "gap-2"}`}
        >
          {proposed.map((candidate) => (
            <li key={candidate.person.id} className="flex items-center gap-2">
              <button
                type="button"
                disabled={busy}
                onClick={() => identify({ personId: candidate.person.id })}
                className={
                  triage
                    ? "inline-flex min-h-11 items-center gap-2 rounded-full border border-accent bg-accent-soft px-3 text-sm text-accent-ink hover:brightness-95 md:min-h-9"
                    : buttonClass()
                }
              >
                {t("voiceDecision.yes", {
                  person: candidate.person.display_name,
                })}
                {triage && (
                  <span className="text-xs">
                    {t("voiceDecision.proximity", {
                      percent: Math.round(candidate.score * 100),
                    })}
                  </span>
                )}
              </button>
              {!triage && (
                <span className="text-xs text-mute">
                  {t("voiceDecision.proximity", {
                    percent: Math.round(candidate.score * 100),
                  })}
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
      <div
        role="group"
        aria-label={t("voiceDecision.decisionLabel", { label })}
        className="flex flex-wrap items-center gap-2"
      >
        {!triage && yesButton}
        {!triage && noButton}
        {identifyButton}
        {!triage && randomButton}
        {!triage && ignoreButton}
        {restoreButton}
        {excerpt && player}
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
        <p
          className={
            triage
              ? "m-0 flex items-center gap-2 rounded-xl border border-line px-3 py-2 text-sm"
              : "m-0 text-xs text-mute"
          }
        >
          {triage && (
            <Users className="size-4 shrink-0 text-mute" aria-hidden="true" />
          )}
          {t("voiceDecision.group", { count: group.member_count })}
        </p>
      )}
      {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}
      {duplicates.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {duplicates.map((duplicate) => (
            <button
              key={duplicate.id}
              type="button"
              disabled={busy}
              onClick={() => identify({ personId: duplicate.id })}
              className={buttonClass("primary")}
            >
              {t("voiceDecision.useExisting", {
                person: duplicate.display_name,
              })}
            </button>
          ))}
          <button
            type="button"
            disabled={busy}
            onClick={() => identify({ newName: name }, true)}
            className={buttonClass()}
          >
            {t("voiceDecision.createAnyway")}
          </button>
        </div>
      )}
      {triage && (
        <div className="flex flex-wrap items-center gap-2 border-t border-line pt-4">
          {randomButton}
          {ignoreButton}
          {trailing && <div className="ml-auto">{trailing}</div>}
        </div>
      )}
    </div>
  );
}
