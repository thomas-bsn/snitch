import { GitMerge, Split, Trash2 } from "lucide-react";
import { useId, useState } from "react";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { MergeDialog } from "./PersonDialogs";
import {
  usePersonActions,
  type Person,
  type PersonImpact,
  type PersonVoice,
} from "./peopleApi";

/**
 * A button that is never hidden: when the action is refused it stays visible and focusable,
 * does nothing, and its written reason is the description screen readers read with it.
 */
function ActionButton({
  label,
  shortLabel,
  icon: Icon,
  variant = "secondary",
  reason,
  reasonId,
  onClick,
}: {
  label: string;
  /** The visible text; absent for an icon-only button named by `label`. */
  shortLabel?: string;
  icon: typeof Trash2;
  variant?: "secondary" | "danger";
  reason: string | undefined;
  reasonId: string;
  onClick: () => void;
}) {
  const blocked = reason !== undefined;
  return (
    <button
      type="button"
      aria-label={label}
      aria-disabled={blocked}
      aria-describedby={blocked ? reasonId : undefined}
      onClick={() => {
        if (!blocked) onClick();
      }}
      className={`${buttonClass(variant)} aria-disabled:cursor-not-allowed aria-disabled:opacity-60`}
    >
      <Icon className="size-4" aria-hidden="true" />
      {shortLabel}
    </button>
  );
}

/** The consequences of an action, counted from the server's own figures. */
function Consequences({ lines }: { lines: string[] }) {
  return (
    <ul className="m-0 flex list-none flex-col gap-1 p-0 text-ink">
      {lines.map((line) => (
        <li key={line}>{line}</li>
      ))}
    </ul>
  );
}

/**
 * Merge, separate and delete a Person, from the header of its record only. Each action says
 * what it will do in figures before it is confirmed, and says in writing why it is refused.
 */
export function PersonActions({
  person,
  impact,
  voices,
  onChanged,
  onMerged,
  onDeleted,
  onRefused,
}: {
  person: Person;
  impact: PersonImpact;
  voices: PersonVoice[];
  /** A voice was separated: the message to announce. */
  onChanged: (message: string) => void;
  onMerged: (target: Person, message: string) => void;
  onDeleted: (message: string) => void;
  /** The server refused: the record is loaded again so that its reasons are up to date. */
  onRefused: () => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const ids = useId();
  const [merging, setMerging] = useState(false);
  const [separating, setSeparating] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [chosen, setChosen] = useState<number[]>([]);
  const name = person.display_name;
  const linked = person.user_id !== null;

  const mergeReason = linked
    ? t("people.actions.reasons.mergeLinked", { name })
    : impact.has_hidden_voices
      ? t("people.actions.reasons.mergeHidden", { name })
      : person.voice_profile_deleted_at !== null
        ? t("people.actions.reasons.mergeProfileDeleted", { name })
        : undefined;
  const deleteReason = linked
    ? t("people.actions.reasons.deleteLinked", { name })
    : impact.has_hidden_voices
      ? t("people.actions.reasons.deleteHidden", { name })
      : undefined;
  const separateReason =
    voices.length > 0
      ? undefined
      : person.local_speaker_count > 0
        ? t("people.actions.reasons.separateHidden", { name })
        : t("people.actions.reasons.separateNone", { name });
  const reasons = [
    { id: `${ids}-merge`, text: mergeReason },
    { id: `${ids}-separate`, text: separateReason },
    { id: `${ids}-delete`, text: deleteReason },
  ];

  function refused(code: string) {
    setFailure(code);
    onRefused();
  }

  async function remove() {
    setBusy(true);
    const result = await actions.remove(person.id);
    setBusy(false);
    setDeleting(false);
    if (!result.ok) {
      refused(result.code);
      return;
    }
    onDeleted(
      t("people.delete.done", {
        name,
        summary: t("people.summary", {
          attributions: result.value.attributions,
          samples: result.value.samples,
          rules: result.value.microphone_rules,
        }),
      }),
    );
  }

  const picked = voices.filter((voice) =>
    chosen.includes(voice.local_speaker_id),
  );
  const pickedSamples = picked.reduce((sum, v) => sum + v.sample_count, 0);

  async function separate() {
    setBusy(true);
    const result = await actions.separate(person.id, chosen);
    setBusy(false);
    setSeparating(false);
    setChosen([]);
    if (!result.ok) {
      refused(result.code);
      return;
    }
    setFailure(undefined);
    onChanged(
      t(
        result.value.attributions === 1
          ? "people.separate.doneOne"
          : "people.separate.done",
        { count: result.value.attributions, name },
      ),
    );
  }

  return (
    // `contents`: the buttons join the row of the record header; the written reasons and the
    // failure take a full row under it.
    <div className="contents">
      <div
        role="group"
        aria-label={t("people.actions.label", { name })}
        className="flex gap-2"
      >
        <ActionButton
          label={t("people.persons.merge", { name })}
          shortLabel={t("people.persons.mergeShort")}
          icon={GitMerge}
          reason={mergeReason}
          reasonId={reasons[0].id}
          onClick={() => setMerging(true)}
        />
        <ActionButton
          label={t("people.persons.separate", { name })}
          shortLabel={t("people.persons.separateShort")}
          icon={Split}
          reason={separateReason}
          reasonId={reasons[1].id}
          onClick={() => {
            setChosen([]);
            setSeparating(true);
          }}
        />
        <ActionButton
          label={t("people.persons.delete", { name })}
          icon={Trash2}
          variant="danger"
          reason={deleteReason}
          reasonId={reasons[2].id}
          onClick={() => setDeleting(true)}
        />
      </div>
      {reasons.some((reason) => reason.text !== undefined) && (
        <ul className="m-0 flex basis-full list-none flex-col gap-1 p-0 text-sm text-mute">
          {reasons.map(
            (reason) =>
              reason.text !== undefined && (
                <li key={reason.id} id={reason.id}>
                  {reason.text}
                </li>
              ),
          )}
        </ul>
      )}
      {failure && (
        <div className="basis-full">
          <ErrorAlert>
            {t("people.actions.notDone", { reason: failureText(failure) })}
          </ErrorAlert>
        </div>
      )}

      <MergeDialog
        person={merging ? person : undefined}
        impact={impact}
        onClose={() => setMerging(false)}
        onFailed={onRefused}
        onMerged={(target, summary) => {
          setMerging(false);
          onMerged(
            target,
            t("people.merge.done", {
              name,
              target: target.display_name,
              summary,
            }),
          );
        }}
      />

      <Modal
        open={separating}
        onOpenChange={(open) => !open && setSeparating(false)}
        title={t("people.separate.title", { name })}
        closeLabel={t("people.decide.close")}
      >
        <form
          onSubmit={(event) => {
            event.preventDefault();
            if (chosen.length > 0) void separate();
          }}
          className="flex flex-col gap-4"
        >
          <p className="text-mute">{t("people.separate.body", { name })}</p>
          <fieldset className="m-0 flex max-h-60 flex-col gap-1 overflow-y-auto border-0 p-0">
            <legend className="sr-only">
              {t("people.separate.legend", { name })}
            </legend>
            {voices.map((voice) => (
              <label
                key={voice.local_speaker_id}
                className="flex min-h-11 items-center gap-2 rounded-lg border border-line px-3"
              >
                <input
                  type="checkbox"
                  checked={chosen.includes(voice.local_speaker_id)}
                  onChange={(event) =>
                    setChosen(
                      event.target.checked
                        ? [...chosen, voice.local_speaker_id]
                        : chosen.filter((id) => id !== voice.local_speaker_id),
                    )
                  }
                />
                {t("people.separate.voice", {
                  label: voice.label,
                  clip: voice.clip_name,
                })}
              </label>
            ))}
          </fieldset>
          {picked.length === 0 ? (
            <p className="text-mute">{t("people.separate.choose")}</p>
          ) : (
            <div className="flex flex-col gap-2 text-mute">
              <Consequences
                lines={[
                  t("people.separate.voices", { count: picked.length }),
                  t("people.separate.samples", { count: pickedSamples }),
                ]}
              />
              <p>{t("people.separate.intact", { name })}</p>
            </div>
          )}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setSeparating(false)}
              className={buttonClass("secondary")}
            >
              {t("people.separate.cancel")}
            </button>
            <button
              type="submit"
              disabled={busy || picked.length === 0}
              className={buttonClass("primary")}
            >
              {t("people.separate.submit")}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={deleting}
        onOpenChange={(open) => !open && setDeleting(false)}
        title={t("people.delete.title", { name })}
        cancelLabel={t("people.delete.cancel")}
        confirmLabel={t("people.delete.submit")}
        danger
        busy={busy}
        onConfirm={() => void remove()}
      >
        <Consequences
          lines={[
            t("people.delete.voices", {
              count: person.local_speaker_count,
              name,
            }),
            t("people.delete.samples", {
              count: person.validated_sample_count,
            }),
            t("people.delete.rules", { count: impact.microphone_rules, name }),
          ]}
        />
        <p>{t("people.delete.intact")}</p>
      </ConfirmDialog>
    </div>
  );
}
