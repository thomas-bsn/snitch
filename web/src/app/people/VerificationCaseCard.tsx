import { useEffect, useId, useRef, useState } from "react";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { ExcerptPlayer } from "./ExcerptPlayer";
import { tintOf } from "./ExcerptThumb";
import { useFailureText } from "./failureText";
import { LevelBadge } from "./LevelBadge";
import {
  fetchPersonPage,
  usePersonActions,
  type Choice,
  type Person,
  type VerificationCase,
} from "./peopleApi";

const NEW = "new";
const SEARCH_DELAY_MS = 250;
const SEARCH_SIZE = 8;
const optionClass =
  "flex min-h-11 cursor-pointer items-center gap-3 rounded-lg border border-line px-3 py-2 has-[:checked]:border-accent has-[:checked]:bg-accent-soft";
const fieldClass =
  "h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10";

/** A field where typing text is expected: the shortcuts leave it alone. */
function isTextField(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  if (target instanceof HTMLTextAreaElement) return true;
  if (target instanceof HTMLSelectElement) return true;
  return (
    target instanceof HTMLInputElement &&
    !["radio", "checkbox", "button", "submit"].includes(target.type)
  );
}

/**
 * The case to decide: the listenable excerpt, the closest people with their proximity, a
 * search in the existing people and the creation of a new one. "Confirm" stays off until
 * something is chosen. A proximity is a suggestion, never a confirmation.
 */
export function VerificationCaseCard({
  item,
  total,
  canSkip,
  onSkip,
  onSettled,
  onTaken,
}: {
  item: VerificationCase;
  total: number;
  canSkip: boolean;
  onSkip: () => void;
  /** Called with the text to announce once the case is settled. */
  onSettled: (message: string) => void;
  /** Called when somebody else settled the case first; nothing was written. */
  onTaken: (message: string) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const group = useId();
  const [picked, setPicked] = useState<string>();
  // The Person chosen in the search stays listed even when the search changes.
  const [searched, setSearched] = useState<Person>();
  const [newName, setNewName] = useState("");
  const [query, setQuery] = useState("");
  const [found, setFound] = useState<Person[]>([]);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  // The search is done by the server, after a short pause in the typing.
  useEffect(() => {
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
  }, [query]);

  const candidates = item.candidates.slice(0, 9);
  const candidateIds = new Set(item.candidates.map((c) => c.person.id));
  const others = found.filter(
    (person) =>
      !candidateIds.has(person.id) &&
      person.id !== searched?.id &&
      person.voice_profile_deleted_at === null,
  );
  const name = newName.trim();
  const choice: Choice | undefined =
    picked === undefined
      ? undefined
      : picked === NEW
        ? name
          ? { newName: name }
          : undefined
        : { personId: Number(picked) };

  async function confirm() {
    if (!choice || busy) return;
    setBusy(true);
    const result = await actions.validate(item.local_speaker_id, choice);
    setBusy(false);
    if (!result.ok) {
      const taken = /^already_validated:(.*)$/.exec(result.code);
      if (taken) {
        onTaken(
          t("people.verification.alreadyValidated", { person: taken[1] }),
        );
        return;
      }
      setFailure(result.code);
      return;
    }
    onSettled(
      t("people.verification.saved", {
        label: item.label,
        person:
          result.value.personName ??
          ("newName" in choice
            ? choice.newName
            : (candidates.find((c) => c.person.id === choice.personId)?.person
                .display_name ??
              searched?.display_name ??
              "")),
      }),
    );
  }

  // 1 to 9 choose a candidate, Enter confirms, S skips; none of them inside a text field.
  const shortcut = useRef<(event: KeyboardEvent) => void>(() => {});
  shortcut.current = (event) => {
    if (
      event.defaultPrevented ||
      event.ctrlKey ||
      event.metaKey ||
      event.altKey
    )
      return;
    if (isTextField(event.target) || busy) return;
    if (/^[1-9]$/.test(event.key)) {
      const candidate = candidates[Number(event.key) - 1];
      if (candidate) setPicked(String(candidate.person.id));
    } else if (event.key === "Enter") {
      // On a button, Enter already presses it.
      if (event.target instanceof HTMLButtonElement) return;
      if (choice) {
        event.preventDefault();
        void confirm();
      }
    } else if (event.key === "s" || event.key === "S") {
      if (canSkip) onSkip();
    }
  };
  useEffect(() => {
    const listener = (event: KeyboardEvent) => shortcut.current(event);
    document.addEventListener("keydown", listener);
    return () => document.removeEventListener("keydown", listener);
  }, []);

  const radio = (
    value: string,
    label: string,
    extra?: string,
    key?: string,
  ) => (
    <label className={optionClass}>
      <input
        type="radio"
        name={group}
        value={value}
        checked={picked === value}
        aria-keyshortcuts={key}
        onChange={() => setPicked(value)}
        className="size-4 accent-accent-solid"
      />
      {key && (
        <kbd
          aria-hidden="true"
          className="rounded border border-line-strong px-1.5 text-xs text-mute"
        >
          {key}
        </kbd>
      )}
      <span className="flex-1">{label}</span>
      {extra && <span className="text-mute">{extra}</span>}
    </label>
  );

  return (
    <section
      aria-label={t("people.verification.caseLabel")}
      className="flex max-w-2xl flex-col gap-4 rounded-xl border border-line bg-surface p-4"
    >
      <div className="flex items-center gap-3">
        <div
          aria-hidden="true"
          className={`size-12 shrink-0 rounded-lg ${tintOf(item.local_speaker_id)}`}
        />
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="m-0 text-lg font-semibold">{item.label}</h2>
            <LevelBadge level={item.decision_level} />
          </div>
          <span className="truncate text-mute">
            {t("people.verification.clipLine", {
              clip: item.clip_name,
              speech: t("people.verification.speech", {
                seconds: item.speaking_seconds.toFixed(1),
              }),
            })}
          </span>
          <span className="text-mute">
            {item.group_voice_count > 1
              ? t("people.verification.heardIn", {
                  clips: item.group_clip_count,
                  voices: item.group_voice_count,
                })
              : t("people.verification.single")}
          </span>
        </div>
        <ExcerptPlayer
          localSpeakerId={item.local_speaker_id}
          label={item.label}
          clip={item.clip_name}
        />
      </div>
      <p className="m-0 text-mute">
        {t("people.verification.position", { total })}
      </p>

      {candidates.length > 0 && (
        <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
          <legend className="mb-1 font-medium">
            {t("people.decide.closest")}
          </legend>
          {candidates.map((candidate, index) => (
            <div key={candidate.person.id}>
              {radio(
                String(candidate.person.id),
                candidate.person.display_name,
                t("people.verification.similar", {
                  percent: Math.round(candidate.score * 100),
                }),
                String(index + 1),
              )}
            </div>
          ))}
          <p className="m-0 text-mute">{t("people.verification.suggestion")}</p>
        </fieldset>
      )}

      <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
        <legend className="mb-1 font-medium">
          {t(
            candidates.length > 0
              ? "people.decide.someoneElse"
              : "people.decide.existing",
          )}
        </legend>
        <input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          aria-label={t("people.decide.searchLabel")}
          placeholder={t("people.decide.searchPlaceholder")}
          className={fieldClass}
        />
        {searched && (
          <div>{radio(String(searched.id), searched.display_name)}</div>
        )}
        {others.map((person) => (
          <div key={person.id}>
            <label className={optionClass}>
              <input
                type="radio"
                name={group}
                value={String(person.id)}
                checked={picked === String(person.id)}
                onChange={() => {
                  setPicked(String(person.id));
                  setSearched(person);
                }}
                className="size-4 accent-accent-solid"
              />
              <span className="flex-1">{person.display_name}</span>
            </label>
          </div>
        ))}
        {others.length === 0 && !searched && (
          <p className="text-mute">{t("people.decide.noMatch")}</p>
        )}
      </fieldset>

      <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
        <legend className="mb-1 font-medium">
          {t("people.decide.someoneNew")}
        </legend>
        {radio(NEW, t("people.decide.createOption"))}
        {picked === NEW && (
          <input
            type="text"
            value={newName}
            maxLength={100}
            onChange={(event) => setNewName(event.target.value)}
            aria-label={t("people.decide.newName")}
            autoComplete="off"
            className={fieldClass}
          />
        )}
      </fieldset>

      {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-mute">{t("people.verification.shortcuts")}</span>
        <div className="flex gap-2">
          <button
            type="button"
            disabled={!canSkip}
            onClick={onSkip}
            aria-keyshortcuts="S"
            className={buttonClass("secondary")}
          >
            {t("people.decide.skip")}
          </button>
          <button
            type="button"
            disabled={!choice || busy}
            onClick={() => void confirm()}
            aria-keyshortcuts="Enter"
            className={buttonClass("primary")}
          >
            {t("people.decide.confirm")}
          </button>
        </div>
      </div>
    </section>
  );
}
