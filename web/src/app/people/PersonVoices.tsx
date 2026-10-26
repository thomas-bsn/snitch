import { useCallback, useState } from "react";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import {
  excerptUrl,
  fetchPersonVoices,
  usePersonActions,
  type Person,
  type PersonVoice,
} from "./peopleApi";

/**
 * The voices confirmed for a Person, each with its excerpt to listen to and the action that
 * separates it (the voice becomes unknown again, the Person stays).
 */
export function PersonVoices({
  person,
  onSeparated,
}: {
  person: Person;
  /** Called with the text to announce, once a voice was separated. */
  onSeparated: (message: string) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const load = useCallback(() => fetchPersonVoices(person.id), [person.id]);
  const { state, reload } = useLoaded(load);
  const [pending, setPending] = useState<PersonVoice>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  async function separate(voice: PersonVoice) {
    setBusy(true);
    const result = await actions.separate(person.id, voice.local_speaker_id);
    setBusy(false);
    setPending(undefined);
    if (!result.ok) {
      setFailure(result.code);
      return;
    }
    setFailure(undefined);
    reload();
    onSeparated(
      t("people.voicesPanel.separated", {
        label: voice.label,
        name: person.display_name,
      }),
    );
  }

  return (
    <div
      role="group"
      aria-label={t("people.voicesPanel.title", { name: person.display_name })}
      className="flex flex-col gap-2"
    >
      {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}
      {state.status === "loading" && (
        <p className="text-mute">{t("people.loading")}</p>
      )}
      {state.status === "failed" && (
        <ErrorAlert>{failureText(state.code)}</ErrorAlert>
      )}
      {state.status === "ready" && state.value.length === 0 && (
        <p className="text-mute">{t("people.voicesPanel.empty")}</p>
      )}
      {state.status === "ready" && state.value.length > 0 && (
        <ul className="m-0 flex list-none flex-col gap-2 p-0">
          {state.value.map((voice) => (
            <li
              key={voice.local_speaker_id}
              className="flex flex-wrap items-center gap-3 rounded-lg border border-line bg-surface p-2.5"
            >
              <div className="min-w-0 flex-1">
                <b>{voice.label}</b>
                <span className="block truncate text-mute">
                  {voice.clip_name} ·{" "}
                  {t("people.voicesPanel.speech", {
                    seconds: voice.speaking_seconds.toFixed(1),
                  })}
                </span>
              </div>
              <audio
                controls
                preload="none"
                aria-label={t("people.voicesPanel.excerpt", {
                  label: voice.label,
                  clip: voice.clip_name,
                })}
                src={excerptUrl(voice.local_speaker_id)}
                className="h-10 max-w-full"
              />
              <button
                type="button"
                onClick={() => setPending(voice)}
                className={buttonClass("secondary")}
              >
                {t("people.voicesPanel.separate", {
                  label: voice.label,
                  name: person.display_name,
                })}
              </button>
            </li>
          ))}
        </ul>
      )}
      <ConfirmDialog
        open={pending !== undefined}
        onOpenChange={(open) => !open && setPending(undefined)}
        title={t("people.voicesPanel.separateTitle")}
        cancelLabel={t("people.voicesPanel.separateCancel")}
        confirmLabel={t("people.voicesPanel.separateSubmit")}
        busy={busy}
        onConfirm={() => pending && void separate(pending)}
      >
        {pending && (
          <p>
            {t("people.voicesPanel.separateBody", {
              label: pending.label,
              clip: pending.clip_name,
              name: person.display_name,
            })}
          </p>
        )}
      </ConfirmDialog>
    </div>
  );
}
