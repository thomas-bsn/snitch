import { memo, useState, type ReactNode } from "react";
import { Link } from "react-router";
import type { components } from "../../../api.gen";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";
import { ClipThumbnail, ThumbnailSetting } from "./ClipThumbnail";
import {
  COLUMNS,
  columnOf,
  isKnownStep,
  needsAttention,
  placingStep,
  type Column,
} from "./columns";

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

/** Cause, code and advised action of a failure, or the remedy of a wait, in the reader's language. */
function Diagnosis({
  code,
  action,
  failed,
}: {
  code?: string;
  action?: string;
  failed: boolean;
}) {
  const { t } = useI18n();
  const known = (key: string) => {
    const text = t(key as MessageKey);
    return text === key ? undefined : text;
  };
  const advice = action
    ? (known(`queue.failure.actions.${action}`) ?? action)
    : undefined;
  if (!failed && !code)
    return advice ? (
      <p
        data-remedy={action}
        className="m-0 mt-1 text-xs font-medium text-accent-ink"
      >
        {advice}
      </p>
    ) : null;
  const cause = code
    ? (known(`queue.failure.causes.${code}`) ?? code)
    : t("queue.failure.unknownCause");
  return (
    <div
      data-failure={code ?? "unknown"}
      className="mt-2 flex flex-col gap-1 rounded-lg bg-warn-bg p-2 text-xs text-warn-fg"
    >
      <p className="m-0">
        <strong>{t("queue.failure.cause")}</strong> : {cause}
        {code && (
          <>
            {" "}
            (<code data-failure-code>{code}</code>)
          </>
        )}
      </p>
      <p className="m-0" data-failure-action={action}>
        <strong>{t("queue.failure.action")}</strong> :{" "}
        {advice ?? t("queue.failure.noAction")}
      </p>
    </div>
  );
}

function StepRow({ step, current }: { step: Step; current: boolean }) {
  const { t } = useI18n();
  return (
    <li
      aria-current={current ? "step" : undefined}
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
      <Diagnosis
        code={step.failure_code}
        action={step.recommended_action}
        failed={step.status === "failed"}
      />
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

// A card is redrawn only when its own run changed: a delta leaves the other runs untouched.
const RunCard = memo(function RunCard({ run }: { run: QueueRun }) {
  const { t } = useI18n();
  const title = run.clip_name || `Clip #${run.clip_id}`;
  const tone =
    run.status === "failed" || run.steps.some((s) => needsAttention(s.status))
      ? "border-warn-fg"
      : "border-line";
  const placing = placingStep(run.steps);
  const unknown = run.steps.filter((s) => !isKnownStep(s.step));
  return (
    <li className="min-w-0">
      {/* Off-screen cards skip their layout and painting; the height they keep is an estimate. */}
      <article
        aria-label={title}
        data-run-card={run.clip_id}
        className={`flex h-full flex-col gap-3 rounded-xl border bg-surface p-3 shadow-sm [contain-intrinsic-size:auto_900px] [content-visibility:auto] ${tone}`}
      >
        <ClipThumbnail clipId={run.clip_id} />
        <header className="min-w-0">
          <h4 className="m-0 break-words text-sm font-semibold">{title}</h4>
          <p className="m-0 mt-1 text-xs text-mute">
            #{run.clip_id} · {t(`queue.board.scope.${run.scope}`)} ·{" "}
            <span data-run-status>{t(`queue.board.run.${run.status}`)}</span>
          </p>
          <IdentificationBadge run={run} />
        </header>
        {unknown.map((step) => (
          <p
            key={step.step}
            role="alert"
            className="m-0 break-words rounded-lg bg-warn-bg p-2 text-xs text-warn-fg"
          >
            {t("queue.board.unknownStep", { step: step.step })}
          </p>
        ))}
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
                    <StepRow
                      key={step.step}
                      step={step}
                      current={step === placing}
                    />
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
});

/** The load of the queue at a glance, as the scheduler counts it. */
export function QueueSummaryBar({ summary }: { summary: Queue["summary"] }) {
  const { t } = useI18n();
  return (
    <ul
      data-queue-summary
      aria-label={t("queue.board.summary.label")}
      className="m-0 flex list-none flex-wrap gap-x-4 gap-y-1 rounded-xl border border-line bg-surface px-4 py-2 text-sm"
    >
      <li data-summary-running>
        {t("queue.board.summary.running", {
          n: summary.running_tasks,
          max: summary.max_concurrency,
        })}
      </li>
      <li data-summary-waiting>
        {t("queue.board.summary.waiting", { n: summary.waiting_clips })}
      </li>
      <li
        data-summary-paused={summary.paused ? "true" : "false"}
        className={summary.paused ? "font-medium text-warn-fg" : undefined}
      >
        {summary.paused
          ? t("queue.board.summary.paused")
          : t("queue.board.summary.active")}
      </li>
    </ul>
  );
}

/** Cards a column draws at first, and how many more each request adds; the count above is never limited. */
export const CARDS_AT_FIRST = 50;
export const CARDS_PER_REQUEST = 100;

function BoardColumn({ id, runs }: { id: Column; runs: QueueRun[] }) {
  const { t } = useI18n();
  const [shown, setShown] = useState(CARDS_AT_FIRST);
  const visible = runs.slice(0, shown);
  const left = runs.length - visible.length;
  return (
    <section
      data-column={id}
      aria-labelledby={`queue-column-${id}`}
      className="flex min-w-0 flex-col gap-3 rounded-xl border border-dashed border-line p-2"
    >
      <h4
        id={`queue-column-${id}`}
        className="m-0 flex items-center justify-between gap-2 px-1 text-sm font-semibold"
      >
        {t(`queue.board.columns.${id}`)}
        <span
          data-column-count
          className="rounded-full bg-accent-soft px-2 text-xs text-accent-ink"
        >
          {runs.length}
        </span>
      </h4>
      {runs.length === 0 ? (
        <p className="m-0 px-1 text-xs text-mute">{t("queue.board.empty")}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col gap-3 p-0">
          {visible.map((run) => (
            <RunCard key={run.id} run={run} />
          ))}
        </ul>
      )}
      {left > 0 && (
        <div data-column-more className="flex flex-col gap-1 px-1">
          <p className="m-0 text-xs text-mute">
            {t("queue.board.hiddenCards", { n: left })}
          </p>
          <button
            type="button"
            onClick={() => setShown((n) => n + CARDS_PER_REQUEST)}
            className="rounded-lg border border-line px-2 py-1 text-sm text-accent-ink"
          >
            {t("queue.board.showMore", {
              n: Math.min(left, CARDS_PER_REQUEST),
              left,
            })}
          </button>
        </div>
      )}
    </section>
  );
}

/** Technical tasks that belong to no analysis are alerts when they failed or are not known. */
const knownTask = /^(transcription|voice)\./;

function TaskCard({ task, actions }: { task: QueueTask; actions: ReactNode }) {
  const { t } = useI18n();
  const unknown = !knownTask.test(task.step);
  const alert = unknown || ["dead", "failed"].includes(task.status);
  const failed = ["dead", "failed", "retry_wait"].includes(task.status);
  return (
    <li
      data-task-card={task.id}
      className={`rounded-xl border bg-surface p-3 shadow-sm ${alert || task.status === "retry_wait" ? "border-warn-fg" : "border-line"}`}
    >
      <p className="m-0 break-words text-sm font-semibold">
        {task.clip_name || `Clip #${task.clip_id}`}
      </p>
      <p className="m-0 mt-1 break-words text-xs text-mute">
        #{task.clip_id} · {task.step} · {t(`queue.status.${task.status}`)}
      </p>
      {alert && (
        <p
          role="alert"
          className="m-0 mt-2 break-words rounded-lg bg-warn-bg p-2 text-xs text-warn-fg"
        >
          {unknown
            ? t("queue.board.taskUnknown", { step: task.step })
            : t("queue.board.taskNotFollowed")}
        </p>
      )}
      {failed || task.failure_code ? (
        <Diagnosis
          code={task.failure_code ?? undefined}
          action={task.recommended_action}
          failed={failed}
        />
      ) : null}
      <div className="mt-2 flex flex-wrap gap-2">
        <Link
          to={`/clips/${task.clip_id}`}
          className="text-sm text-accent-ink underline"
        >
          {t("queue.board.openClip")}
        </Link>
        {actions}
      </div>
    </li>
  );
}

/**
 * One card per analysis run, placed in the column of the step it is at: every active run is
 * listed (no page to turn), a finished step never stands for a finished analysis, parallel
 * branches and the diagnosis of each step stay visible. Technical tasks that no run follows keep
 * their own list, with the usual actions, so none is lost.
 */
export function AnalysisBoard({
  queue,
  taskActions,
}: {
  queue: Queue;
  taskActions: (task: QueueTask) => ReactNode;
}) {
  const { t } = useI18n();
  const byColumn = (id: Column) =>
    id === "completed"
      ? queue.completed
      : queue.active.filter((run) => columnOf(run) === id);
  return (
    <div className="flex flex-col gap-6">
      <QueueSummaryBar summary={queue.summary} />
      <ThumbnailSetting />
      <section aria-labelledby="queue-active">
        <h3 id="queue-active" className="mb-3 text-base font-semibold">
          {t("queue.board.activeTitle")} · {queue.active.length}
        </h3>
        {queue.active.length === 0 && (
          <p className="m-0 mb-3 text-mute">{t("queue.board.activeEmpty")}</p>
        )}
        <div
          data-board
          className="grid items-start gap-3 md:grid-cols-2 xl:grid-cols-5"
        >
          {COLUMNS.map((id) => (
            <BoardColumn key={id} id={id} runs={byColumn(id)} />
          ))}
        </div>
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
              <TaskCard key={task.id} task={task} actions={taskActions(task)} />
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
