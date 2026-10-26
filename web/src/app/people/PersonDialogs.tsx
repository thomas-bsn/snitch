import { useId, useState, type FormEvent } from "react";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Field } from "../components/Field";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { usePersonActions, type Person } from "./peopleApi";

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

/** Chooses the Person that absorbs another one, then merges. */
export function MergeDialog({
  person,
  candidates,
  onClose,
  onMerged,
}: {
  person: Person | undefined;
  candidates: Person[];
  onClose: () => void;
  onMerged: (source: Person, target: Person, summary: string) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const selectId = useId();
  const [targetId, setTargetId] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const targets = candidates.filter(
    (candidate) =>
      candidate.id !== person?.id &&
      candidate.voice_profile_deleted_at === null,
  );

  function reset() {
    setTargetId("");
    setFailure(undefined);
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    const target = targets.find(
      (candidate) => String(candidate.id) === targetId,
    );
    if (!person || !target) return;
    setBusy(true);
    const result = await actions.merge(person.id, target.id);
    setBusy(false);
    if (!result.ok) {
      setFailure(result.code);
      return;
    }
    reset();
    onMerged(
      person,
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
      open={person !== undefined}
      onOpenChange={(open) => {
        if (!open) {
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
          <label htmlFor={selectId} className="font-medium">
            {t("people.merge.targetLabel")}
          </label>
          <select
            id={selectId}
            value={targetId}
            onChange={(event) => setTargetId(event.target.value)}
            className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
          >
            <option value="">{t("people.merge.choose")}</option>
            {targets.map((target) => (
              <option key={target.id} value={target.id}>
                {target.display_name}
              </option>
            ))}
          </select>
        </div>
        {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}
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
            disabled={busy || targetId === ""}
            className={buttonClass("primary")}
          >
            {t("people.merge.submit")}
          </button>
        </div>
      </form>
    </Modal>
  );
}
