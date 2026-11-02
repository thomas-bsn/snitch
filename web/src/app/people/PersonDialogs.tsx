import { useCallback, useId, useState, type FormEvent } from "react";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Field } from "../components/Field";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import {
  fetchPersonPage,
  usePersonActions,
  type Person,
  type PersonImpact,
} from "./peopleApi";

/** Asks for a name and creates the Person. */
export function NewPersonDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (person: Person) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = String(
      new FormData(event.currentTarget).get("name") ?? "",
    ).trim();
    if (!name) return;
    setBusy(true);
    const result = await actions.create(name);
    setBusy(false);
    if (!result.ok) {
      setFailure(result.code);
      return;
    }
    setFailure(undefined);
    onCreated(result.value);
  }

  return (
    <Modal
      open={open}
      onOpenChange={(next) => {
        if (!next) setFailure(undefined);
        onOpenChange(next);
      }}
      title={t("people.create.title")}
      closeLabel={t("people.decide.close")}
    >
      <form onSubmit={submit} className="flex flex-col gap-4">
        <Field
          label={t("people.create.nameLabel")}
          hint={t("people.create.nameHint")}
          name="name"
          required
          maxLength={100}
          autoComplete="off"
        />
        {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={() => onOpenChange(false)}
            className={buttonClass("secondary")}
          >
            {t("people.create.cancel")}
          </button>
          <button
            type="submit"
            disabled={busy}
            className={buttonClass("primary")}
          >
            {t("people.create.submit")}
          </button>
        </div>
      </form>
    </Modal>
  );
}

/** Chooses, by a search done on the server, the Person that absorbs another one, then merges. */
export function MergeDialog({
  person,
  impact,
  onClose,
  onMerged,
  onFailed,
}: {
  person: Person | undefined;
  /** What the merge also touches (microphone track rules), counted by the server. */
  impact: PersonImpact;
  onClose: () => void;
  onMerged: (target: Person, summary: string) => void;
  /** The server refused: the caller loads the record again. */
  onFailed: () => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const searchId = useId();
  const [query, setQuery] = useState("");
  const [target, setTarget] = useState<Person>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const search = query.trim();
  const open = person !== undefined;
  const load = useCallback(
    () =>
      open
        ? fetchPersonPage(search, 1, 11)
        : Promise.resolve({
            ok: true as const,
            value: { items: [] as Person[], total: 0 },
          }),
    [open, search],
  );
  const { state } = useLoaded(load);
  const targets =
    state.status === "ready"
      ? state.value.items
          .filter(
            (candidate) =>
              candidate.id !== person?.id &&
              candidate.voice_profile_deleted_at === null,
          )
          .slice(0, 10)
      : [];

  function reset() {
    setQuery("");
    setTarget(undefined);
    setFailure(undefined);
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!person || !target) return;
    setBusy(true);
    const result = await actions.merge(person.id, target.id);
    setBusy(false);
    if (!result.ok) {
      setFailure(result.code);
      onFailed();
      return;
    }
    reset();
    onMerged(
      target,
      t("people.summary", {
        attributions: result.value.attributions,
        samples: result.value.samples,
        rules: result.value.microphone_rules,
      }),
    );
  }

  return (
    <Modal
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          reset();
          onClose();
        }
      }}
      title={t("people.merge.title", { name: person?.display_name ?? "" })}
      closeLabel={t("people.decide.close")}
    >
      <form onSubmit={submit} className="flex flex-col gap-4">
        <p className="text-mute">
          {t("people.merge.body", { name: person?.display_name ?? "" })}
        </p>
        <div className="flex flex-col gap-1.5">
          <label htmlFor={searchId} className="font-medium">
            {t("people.merge.targetLabel")}
          </label>
          <input
            id={searchId}
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("people.persons.searchPlaceholder")}
            autoComplete="off"
            className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
          />
        </div>
        <div
          role="radiogroup"
          aria-label={t("people.merge.choose")}
          className="flex max-h-60 flex-col gap-1 overflow-y-auto"
        >
          {targets.map((candidate) => (
            <label
              key={candidate.id}
              className="flex min-h-11 items-center gap-2 rounded-lg border border-line px-3"
            >
              <input
                type="radio"
                name="merge-target"
                checked={target?.id === candidate.id}
                onChange={() => setTarget(candidate)}
              />
              {candidate.display_name}
            </label>
          ))}
          {state.status === "ready" && targets.length === 0 && (
            <p className="text-mute">{t("people.persons.noMatch")}</p>
          )}
        </div>
        {person && target && (
          <ul className="m-0 flex list-none flex-col gap-1 p-0">
            <li>
              {t("people.merge.voices", {
                count: person.local_speaker_count,
                name: person.display_name,
                target: target.display_name,
              })}
            </li>
            <li>
              {t("people.merge.samples", {
                count: person.validated_sample_count,
                name: person.display_name,
                target: target.display_name,
              })}
            </li>
            <li>
              {t("people.merge.rules", {
                count: impact.microphone_rules,
                name: person.display_name,
                target: target.display_name,
              })}
            </li>
          </ul>
        )}
        {person && target && (
          <p className="text-mute">
            {t("people.merge.intact", {
              name: person.display_name,
              target: target.display_name,
            })}
          </p>
        )}
        {failure && (
          <ErrorAlert>
            {t("people.actions.notDone", { reason: failureText(failure) })}
          </ErrorAlert>
        )}
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={() => {
              reset();
              onClose();
            }}
            className={buttonClass("secondary")}
          >
            {t("people.merge.cancel")}
          </button>
          <button
            type="submit"
            disabled={busy || target === undefined}
            className={buttonClass("primary")}
          >
            {t("people.merge.submit")}
          </button>
        </div>
      </form>
    </Modal>
  );
}
