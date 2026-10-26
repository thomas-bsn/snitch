import { useId, useState } from "react";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { LevelBadge } from "./LevelBadge";
import {
  excerptUrl,
  usePersonActions,
  type Choice,
  type Person,
  type VerificationCase,
} from "./peopleApi";

const NEW = "new";
const optionClass =
  "flex min-h-11 cursor-pointer items-center gap-3 rounded-lg border border-line px-3 py-2 has-[:checked]:border-accent has-[:checked]:bg-accent-soft";

/**
 * The "Decide" panel of a case: the closest people with their proximity, a search in the
 * existing people, or a new person. "Confirm" stays off until something is chosen; "Skip"
 * closes the panel and changes nothing.
 */
export function DecidePanel({
  item,
  persons,
  onClose,
  onSettled,
}: {
  item: VerificationCase | undefined;
  persons: Person[];
  onClose: () => void;
  /** Called with the text to announce once the case is settled. */
  onSettled: (message: string) => void;
}) {
  const { t } = useI18n();
  return (
    <Modal
      open={item !== undefined}
      onOpenChange={(open) => !open && onClose()}
      title={t("people.decide.title", { label: item?.label ?? "" })}
      closeLabel={t("people.decide.close")}
      variant="side"
    >
      {/* Mounted per case, so a new case starts without a choice. */}
      {item && (
        <DecideForm
          key={item.local_speaker_id}
          item={item}
          persons={persons}
          onClose={onClose}
          onSettled={onSettled}
        />
      )}
    </Modal>
  );
}

function DecideForm({
  item,
  persons,
  onClose,
  onSettled,
}: {
  item: VerificationCase;
  persons: Person[];
  onClose: () => void;
  onSettled: (message: string) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const group = useId();
  const [picked, setPicked] = useState<string>();
  const [newName, setNewName] = useState("");
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  const candidateIds = new Set(item.candidates.map((c) => c.person.id));
  const others = persons.filter(
    (person) =>
      !candidateIds.has(person.id) &&
      person.voice_profile_deleted_at === null &&
      person.display_name.toLowerCase().includes(query.trim().toLowerCase()),
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
    if (!choice) return;
    setBusy(true);
    const result = await actions.validate(item.local_speaker_id, choice);
    setBusy(false);
    if (!result.ok) {
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
            : (persons.find((p) => p.id === choice.personId)?.display_name ??
              "")),
      }),
    );
  }

  const radio = (value: string, label: string, extra?: string) => (
    <label className={optionClass}>
      <input
        type="radio"
        name={group}
        value={value}
        checked={picked === value}
        onChange={() => setPicked(value)}
        className="size-4 accent-accent-solid"
      />
      <span className="flex-1">{label}</span>
      {extra && <span className="text-mute">{extra}</span>}
    </label>
  );

  return (
    <>
      <div className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <b className="truncate">{item.clip_name}</b>
          <LevelBadge level={item.decision_level} />
        </div>
        <span className="text-mute">
          {t("people.verification.speech", {
            seconds: item.speaking_seconds.toFixed(1),
          })}
        </span>
        <audio
          controls
          preload="none"
          aria-label={t("people.excerpt.audio", {
            label: item.label,
            clip: item.clip_name,
          })}
          src={excerptUrl(item.local_speaker_id)}
          className="w-full"
        />
      </div>

      {item.candidates.length > 0 && (
        <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
          <legend className="mb-1 font-medium">
            {t("people.decide.closest")}
          </legend>
          {item.candidates.map((candidate) => (
            <div key={candidate.person.id}>
              {radio(
                String(candidate.person.id),
                candidate.person.display_name,
                t("people.verification.similar", {
                  percent: Math.round(candidate.score * 100),
                }),
              )}
            </div>
          ))}
        </fieldset>
      )}

      <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
        <legend className="mb-1 font-medium">
          {t(
            item.candidates.length > 0
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
          className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
        />
        {others.map((person) => (
          <div key={person.id}>
            {radio(String(person.id), person.display_name)}
          </div>
        ))}
        {others.length === 0 && (
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
            autoFocus
            className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
          />
        )}
      </fieldset>

      {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}
      <div className="mt-auto flex justify-end gap-2 pt-2">
        <button
          type="button"
          onClick={onClose}
          className={buttonClass("secondary")}
        >
          {t("people.decide.skip")}
        </button>
        <button
          type="button"
          disabled={!choice || busy}
          onClick={() => void confirm()}
          className={buttonClass("primary")}
        >
          {t("people.decide.confirm")}
        </button>
      </div>
    </>
  );
}
