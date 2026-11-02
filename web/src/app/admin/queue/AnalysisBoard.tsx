import type { ReactNode } from "react";
import { Link } from "react-router";
import type { components } from "../../../api.gen";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";

export type Queue = components["schemas"]["AnalysisQueue"];
type QueueRun = components["schemas"]["AnalysisQueueRun"];
type QueueTask = components["schemas"]["AnalysisQueueTask"];
type Step = QueueRun["steps"][number];

/** Steps of a run, grouped the way they advance: a trunk, then two independent branches. */
const groups: { title: MessageKey; steps: Step["step"][] }[] = [
  { title: "queue.board.trunk", steps: ["preparation", "transcription"] },
  { title: "queue.board.branchSearch", steps: ["indexing"] },
  {
    title: "queue.board.branchVoices",
    steps: ["diarization", "fingerprints", "matching"],
  },
];

const needsAttention = (status: Step["status"]) =>
  status === "failed" || status === "blocked" || status === "retry";

function StepRow({ step }: { step: Step }) {
  const { t } = useI18n();
  return (
    <li
      className={`rounded-lg border p-2 ${needsAttention(step.status) ? "border-warn-fg bg-warn-bg" : "border-line"} ${step.status === "not_applicable" ? "opacity-70" : ""}`}
    >
      <p className="m-0 flex flex-wrap items-baseline justify-between gap-2 text-sm">
        <strong>{t(`clip.analysis.step.${step.step}`)}</strong>
        <span>{t(`clip.analysis.status.${step.status}`)}</span>
      </p>
      <p className="m-0 mt-1 text-xs text-mute">
        {t(`clip.analysis.reason.${step.reason}` as MessageKey)}
        {step.attempt > 1 &&
          ` · ${t("queue.board.attempt", { n: step.attempt })}`}
      </p>
    </li>
  );
}

/**
 * Voice outcome of a finished run, from the voices the clip really has left to decide (the
 * Verification list), never from the run status: a completed run with some left is only
 * partially identified.
 */
export function IdentificationBadge({
  run,
}: {
  run: {
    status: string;
    pending_voices: number;
    steps: { step: string; status: string }[];
  };
}) {
  const { t } = useI18n();
  if (run.status !== "completed") return null;
  const voices = run.steps.find((s) => s.step === "matching");
  if (run.pending_voices > 0)
    return (
      <p
        data-identification="partial"
        className="m-0 mt-1 text-xs font-medium text-accent-ink"
      >
        {t("queue.board.partiallyIdentified")} ·{" "}
        {t("queue.board.voicesToCheck", { n: run.pending_voices })}
      </p>
    );
  if (!voices || voices.status === "not_applicable") return null;
  return (
    <p
      data-identification="complete"
      className="m-0 mt-1 text-xs font-medium text-mute"
    >
      {t("queue.board.identified")}
    </p>
  );
}

function RunCard({ run }: { run: QueueRun }) {
  const { t } = useI18n();
  const title = run.clip_name || `Clip #${run.clip_id}`;
  const tone =
    run.status === "failed" || run.steps.some((s) => needsAttention(s.status))
      ? "border-warn-fg"
      : "border-line";
  return (
    <li>
      <article
        aria-label={title}
        data-run-card={run.clip_id}
        className={`flex h-full flex-col gap-3 rounded-xl border bg-surface p-3 shadow-sm ${tone}`}
      >
        <header className="min-w-0">
          <h4 className="m-0 break-words text-sm font-semibold">{title}</h4>
          <p className="m-0 mt-1 text-xs text-mute">
            #{run.clip_id} · {t(`queue.board.scope.${run.scope}`)} ·{" "}
            <span data-run-status>{t(`queue.board.run.${run.status}`)}</span>
          </p>
          <IdentificationBadge run={run} />
        </header>
        <div
          role="group"
          aria-label={t("queue.board.stepsOf", { clip: title })}
          className="flex flex-col gap-3"
        >
          {groups.map((group) => {
            const steps = group.steps
              .map((name) => run.steps.find((s) => s.step === name))
              .filter((s): s is Step => !!s);
            if (steps.length === 0) return null;
            return (
              <section key={group.title}>
                <h5 className="m-0 mb-1 text-xs font-medium uppercase tracking-wide text-mute">
                  {t(group.title)}
                </h5>
                <ul className="m-0 flex list-none flex-col gap-1 p-0">
                  {steps.map((step) => (
                    <StepRow key={step.step} step={step} />
                  ))}
                </ul>
              </section>
            );
          })}
        </div>
        <Link
          to={`/clips/${run.clip_id}`}
          className="mt-auto text-sm text-accent-ink underline"
        >
          {t("queue.board.openClip")}
        </Link>
      </article>
    </li>
  );
}

function RunList({ runs, empty }: { runs: QueueRun[]; empty: string }) {
  if (runs.length === 0) return <p className="m-0 text-mute">{empty}</p>;
  return (
    <ul className="m-0 grid list-none gap-3 p-0 sm:grid-cols-2 xl:grid-cols-3">
      {runs.map((run) => (
        <RunCard key={run.id} run={run} />
      ))}
    </ul>
  );
}

/**
 * One card per analysis run, as the server describes them: every active run is listed (no
 * page to turn), a finished step never stands for a finished analysis, parallel branches and
 * the diagnosis of each step stay visible. Technical tasks that no run follows keep their own
 * list, with the usual actions, so none is lost.
 */
export function AnalysisBoard({
  queue,
  taskActions,
}: {
  queue: Queue;
  taskActions: (task: QueueTask) => ReactNode;
}) {
  const { t } = useI18n();
  return (
    <div className="flex flex-col gap-6">
      <section aria-labelledby="queue-active">
        <h3 id="queue-active" className="mb-3 text-base font-semibold">
          {t("queue.board.activeTitle")} · {queue.active.length}
        </h3>
        <RunList runs={queue.active} empty={t("queue.board.activeEmpty")} />
      </section>
      <section aria-labelledby="queue-completed">
        <h3 id="queue-completed" className="mb-3 text-base font-semibold">
          {t("queue.board.completedTitle")}
        </h3>
        <RunList
          runs={queue.completed}
          empty={t("queue.board.completedEmpty")}
        />
      </section>
      {queue.tasks.length > 0 && (
        <section aria-labelledby="queue-tasks">
          <h3 id="queue-tasks" className="mb-1 text-base font-semibold">
            {t("queue.board.tasksTitle")} · {queue.tasks.length}
          </h3>
          <p className="m-0 mb-3 text-sm text-mute">
            {t("queue.board.tasksHint")}
          </p>
          <ul className="m-0 grid list-none gap-3 p-0 sm:grid-cols-2 xl:grid-cols-3">
            {queue.tasks.map((task) => (
              <li
                key={task.id}
                data-task-card={task.id}
                className={`rounded-xl border bg-surface p-3 shadow-sm ${["retry_wait", "dead"].includes(task.status) ? "border-warn-fg" : "border-line"}`}
              >
                <p className="m-0 break-words text-sm font-semibold">
                  {task.clip_name || `Clip #${task.clip_id}`}
                </p>
                <p className="m-0 mt-1 text-xs text-mute">
                  #{task.clip_id} · {task.step} ·{" "}
                  {t(`queue.status.${task.status}`)}
                </p>
                {task.last_error && (
                  <p className="m-0 mt-2 rounded-lg bg-warn-bg p-2 text-xs text-warn-fg">
                    {task.last_error}
                  </p>
                )}
                <div className="mt-2 flex flex-wrap gap-2">
                  <Link
                    to={`/clips/${task.clip_id}`}
                    className="text-sm text-accent-ink underline"
                  >
                    {t("queue.board.openClip")}
                  </Link>
                  {taskActions(task)}
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
