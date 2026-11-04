import {
  ChevronRight,
  CircleAlert,
  CircleCheck,
  CircleDashed,
  CircleSlash,
  Clock,
  LoaderCircle,
  type LucideIcon,
} from "lucide-react";
import type { components } from "../../api.gen";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";

type Run =
  | components["schemas"]["ClipAnalysisRun"]
  | components["schemas"]["ClipAnalysisView"];
type Step = Run["steps"][number];
type StepName = Step["step"];

/** Where the analysis of a clip stands, read from the run the server sends. */
export type PillState =
  | { kind: "never" }
  | { kind: "busy" }
  | { kind: "running"; step: StepName }
  | { kind: "waiting"; reason: string }
  | { kind: "failed"; step?: StepName }
  | { kind: "cancelled" }
  | { kind: "complete" };

const FAILED: Step["status"][] = ["failed"];
const WAITING: Step["status"][] = ["waiting", "retry", "blocked"];

/**
 * The pill follows the run: a running step wins, then a failure, then the cause of a wait.
 * The steps are never turned into a percentage: only their stable states are read.
 */
export function analysisPillState(run: Run | null): PillState {
  if (!run) return { kind: "never" };
  if (run.status === "active") {
    const running = run.steps.find((s) => s.status === "running");
    if (running) return { kind: "running", step: running.step };
    const failed = run.steps.find((s) => FAILED.includes(s.status));
    if (failed) return { kind: "failed", step: failed.step };
    const waiting = run.steps.find((s) => WAITING.includes(s.status));
    if (waiting) return { kind: "waiting", reason: waiting.reason };
    return { kind: "busy" };
  }
  const failed = run.steps.find((s) => FAILED.includes(s.status));
  if (failed) return { kind: "failed", step: failed.step };
  if (run.status === "failed") return { kind: "failed" };
  if (run.status === "cancelled") return { kind: "cancelled" };
  return { kind: "complete" };
}

const looks: Record<
  PillState["kind"],
  { icon: LucideIcon; className: string }
> = {
  never: { icon: CircleDashed, className: "bg-side text-mute" },
  busy: { icon: LoaderCircle, className: "bg-accent-soft text-accent-ink" },
  running: {
    icon: LoaderCircle,
    className: "bg-accent-soft text-accent-ink",
  },
  waiting: { icon: Clock, className: "bg-warn-bg text-warn-fg" },
  failed: { icon: CircleAlert, className: "bg-bad-bg text-bad-fg" },
  cancelled: { icon: CircleSlash, className: "bg-side text-mute" },
  complete: { icon: CircleCheck, className: "bg-ok-bg text-ok-fg" },
};

/**
 * The analysis of a clip in the header: a live pill that opens the analysis panel. The
 * label is text and an icon, never the colour alone.
 */
export function AnalysisPill({
  run,
  legacy,
  onOpen,
}: {
  run: Run | null;
  /** What the steps say when the server has no run for the clip. */
  legacy?: PillState | null;
  onOpen: () => void;
}) {
  const { t } = useI18n();
  const state: PillState = !run && legacy ? legacy : analysisPillState(run);
  const { icon: Icon, className } = looks[state.kind];
  const label = (() => {
    switch (state.kind) {
      case "never":
        return t("clip.analysis.pill.never");
      case "busy":
        return t("clip.analysis.pill.busy");
      case "running":
        return t("clip.analysis.pill.running", {
          step: t(`clip.analysis.step.${state.step}` as MessageKey),
        });
      case "waiting":
        return t("clip.analysis.pill.waiting", {
          reason: t(`clip.analysis.reason.${state.reason}` as MessageKey),
        });
      case "failed":
        return state.step
          ? t("clip.analysis.pill.failed", {
              step: t(`clip.analysis.step.${state.step}` as MessageKey),
            })
          : t("clip.analysis.pill.failedNoStep");
      case "cancelled":
        return t("clip.analysis.pill.cancelled");
      case "complete":
        return t("clip.analysis.pill.complete");
    }
  })();
  return (
    <button
      type="button"
      aria-haspopup="dialog"
      onClick={onOpen}
      className="inline-flex min-h-11 max-w-full items-center gap-2 rounded-lg border border-line-strong px-3 py-1.5 hover:bg-side md:min-h-0"
    >
      <span className="sr-only">{t("clip.analysis.label")}: </span>
      <span aria-hidden="true" className="text-sm font-medium">
        {t("clip.analysis.short")}
      </span>
      <span
        className={`inline-flex min-w-0 items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${className}`}
      >
        <Icon
          className={`size-3.5 shrink-0 ${state.kind === "running" || state.kind === "busy" ? "motion-safe:animate-spin" : ""}`}
          aria-hidden="true"
        />
        <span aria-live="polite" className="min-w-0 truncate text-left">
          {label}
        </span>
      </span>
      <ChevronRight className="size-4 shrink-0 text-mute" aria-hidden="true" />
    </button>
  );
}
