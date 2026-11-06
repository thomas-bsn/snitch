import {
  CircleAlert,
  CircleCheck,
  CircleDashed,
  CircleDot,
  OctagonPause,
  CircleSlash,
  Hourglass,
  LoaderCircle,
  type LucideIcon,
} from "lucide-react";
import { useI18n } from "../i18n";
import type { Clip, ClipVoice } from "./libraryApi";
import { clipSteps, type LiveRun, type StepState } from "./progress";

const looks: Record<StepState, { icon: LucideIcon; className: string }> = {
  done: { icon: CircleCheck, className: "text-ok-fg" },
  running: { icon: LoaderCircle, className: "text-accent-ink" },
  blocked: { icon: OctagonPause, className: "text-bad-fg" },
  queued: { icon: Hourglass, className: "text-warn-fg" },
  partial: { icon: CircleDot, className: "text-warn-fg" },
  failed: { icon: CircleAlert, className: "text-bad-fg" },
  unavailable: { icon: CircleSlash, className: "text-mute" },
  none: { icon: CircleDashed, className: "text-mute" },
};

/** Where the clip stands: imported, transcribed, indexed, voices processed, voices identified. */
export function ProgressBanner({
  clip,
  voice,
  run = null,
  assistiveOnly = false,
}: {
  clip: Clip;
  voice: ClipVoice | null;
  /** The analysis in progress, when there is one: its steps are the live state. */
  run?: LiveRun | null;
  /**
   * On the page the steps are read by assistive technology only (the analysis panel shows
   * them); the panel's copy is then the visual one, hidden from it.
   */
  assistiveOnly?: boolean;
}) {
  const { t } = useI18n();
  return (
    <ol
      aria-label={t("clip.progress.label")}
      aria-hidden={assistiveOnly ? undefined : true}
      className={`${assistiveOnly ? "sr-only " : ""}m-0 grid list-none grid-cols-1 gap-y-2.5 rounded-xl border border-line bg-side p-3`}
    >
      {clipSteps(clip, voice, run).map((step) => {
        const { icon: Icon, className } = looks[step.state];
        const stateText = step.detail
          ? t("clip.progress.partial", step.detail)
          : t(`clip.progress.state.${step.state}`);
        return (
          <li
            key={step.key}
            data-state={step.state}
            className="flex items-start gap-2"
          >
            <Icon
              // Running is shown by a spin that stops under reduced motion; no percentage is invented.
              className={`mt-0.5 size-4 shrink-0 ${className}${step.state === "running" ? " motion-safe:animate-spin" : ""}`}
              aria-hidden="true"
            />
            <div className="min-w-0">
              <p className="m-0 text-sm font-medium">{t(step.label)}</p>
              <p className="m-0 text-xs text-mute">{stateText}</p>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
