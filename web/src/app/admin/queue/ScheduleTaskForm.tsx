import { useState, type FormEvent } from "react";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";

export type UnitStep = "transcription" | "voice";

type Worker = { revoked_at?: string | null; supported_steps?: string[] | null };

const transcribes = (w: Worker) =>
  !w.revoked_at &&
  !!w.supported_steps?.some((s) => s.startsWith("transcription."));
const analysesVoices = (w: Worker) =>
  !w.revoked_at &&
  !!w.supported_steps?.some(
    (s) => s === "voice.*" || s.startsWith("voice.diarization."),
  );

/**
 * Schedule task, at the top of the page. The single-step task stays available
 * as a collapsed advanced option.
 */
export function ScheduleTaskForm({
  workers,
  busy,
  onSchedule,
}: {
  workers: Worker[];
  busy: boolean;
  onSchedule: (clip: number, step: UnitStep) => void;
}) {
  const { t } = useI18n();
  const [step, setStep] = useState<UnitStep>();
  const canTranscribe = workers.some(transcribes);
  const canAnalyseVoices = workers.some(analysesVoices);
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const clip = Number(new FormData(event.currentTarget).get("clip"));
    if (step) onSchedule(clip, step);
  }
  const option = (value: UnitStep, title: string, hint: string) => (
    <label className="flex min-h-16 cursor-pointer items-start gap-2 rounded-lg border border-line p-3 has-[:checked]:border-accent has-[:checked]:bg-accent-soft">
      <input
        type="radio"
        name="step"
        value={value}
        checked={step === value}
        onChange={() => setStep(value)}
        className="sr-only"
      />
      <span>
        <strong className="block text-sm">{title}</strong>
        <span className="mt-1 block text-xs text-mute">{hint}</span>
      </span>
    </label>
  );
  return (
    <form
      aria-label={t("queue.schedule")}
      onSubmit={submit}
      className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4"
    >
      <div className="flex flex-wrap items-end gap-3">
        <div className="min-w-0">
          <h3 className="text-base font-semibold">{t("queue.schedule")}</h3>
          <p className="text-sm text-mute">{t("queue.scheduleHint")}</p>
        </div>
        <label className="flex flex-col gap-1 text-sm font-medium">
          {t("queue.clip")}
          <input
            name="clip"
            type="number"
            min={1}
            required
            placeholder="ex. 18"
            className="h-11 w-40 rounded-lg border border-line-strong bg-surface px-3 text-sm text-ink md:h-9"
          />
        </label>
        <button
          disabled={busy || !step}
          className={buttonClass("primary")}
          type="submit"
        >
          {t("queue.scheduleSubmit")}
        </button>
      </div>
      {!canTranscribe && !canAnalyseVoices && (
        <p role="status" className="rounded-lg bg-warn-bg p-3 text-warn-fg">
          {t("queue.noCompatibleWorker")}
        </p>
      )}
      <details>
        <summary className="cursor-pointer text-sm font-medium">
          {t("queue.advancedTitle")}
        </summary>
        <fieldset className="mt-3 grid gap-3 md:grid-cols-2">
          <legend className="mb-2 text-sm text-mute">
            {t("queue.advancedHint")}
          </legend>
          {canTranscribe &&
            option(
              "transcription",
              t("queue.stepTranscription"),
              t("queue.stepTranscriptionHint"),
            )}
          {canAnalyseVoices &&
            option("voice", t("queue.stepVoice"), t("queue.stepVoiceHint"))}
        </fieldset>
      </details>
      {!step && <p className="text-xs text-mute">{t("queue.stepRequired")}</p>}
    </form>
  );
}
