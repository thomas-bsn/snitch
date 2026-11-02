import { useState, type FormEvent } from "react";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import type { components } from "../../../api.gen";

type Queue = components["schemas"]["TaskQueueSummary"];

/**
 * Compact queue bar: running tasks over the ceiling, waiting clips (which use
 * no slot) and the global pause.
 */
export function QueueControlBar({
  queue,
  busy,
  onSave,
}: {
  queue: Queue;
  busy: boolean;
  onSave: (paused: boolean, maxConcurrency: number) => void;
}) {
  const { t } = useI18n();
  const [paused, setPaused] = useState<boolean>();
  const [concurrency, setConcurrency] = useState<string>();
  function submit(event: FormEvent) {
    event.preventDefault();
    onSave(
      paused ?? queue.paused,
      Number(concurrency ?? queue.max_concurrency),
    );
    setPaused(undefined);
    setConcurrency(undefined);
  }
  return (
    <section
      aria-label={t("queue.controlsTitle")}
      className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4"
    >
      <div className="flex flex-wrap items-end gap-x-8 gap-y-3">
        <p title={t("queue.runningHint")}>
          <span className="block text-xs text-mute">
            {t("queue.runningLabel")}
          </span>
          <strong className="text-lg">
            {t("queue.runningValue", {
              running: queue.running,
              max: queue.max_concurrency,
            })}
          </strong>
        </p>
        <p title={t("queue.waitingHint")}>
          <span className="block text-xs text-mute">
            {t("queue.waitingLabel")}
          </span>
          <strong className="text-lg" data-testid="waiting-count">
            {queue.waiting}
          </strong>
        </p>
        <form onSubmit={submit} className="flex flex-wrap items-end gap-4">
          <Field
            label={t("queue.concurrency")}
            type="number"
            min={1}
            max={4}
            required
            value={concurrency ?? String(queue.max_concurrency)}
            onChange={(e) => setConcurrency(e.target.value)}
          />
          <label className="flex min-h-11 items-center gap-2">
            <input
              type="checkbox"
              checked={paused ?? queue.paused}
              onChange={(e) => setPaused(e.target.checked)}
            />
            {t("queue.pause")}
          </label>
          <button disabled={busy} className={buttonClass("primary")}>
            {t("queue.save")}
          </button>
        </form>
      </div>
      <p className="text-xs text-mute">{t("queue.waitingHint")}</p>
      {queue.paused && (
        <p role="status" className="rounded-lg bg-warn-bg p-3 text-warn-fg">
          {t("queue.pausedBanner")}
        </p>
      )}
    </section>
  );
}
