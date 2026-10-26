import {
  CircleAlert,
  CircleCheck,
  CircleDashed,
  CircleDot,
  CircleSlash,
  Hourglass,
  LoaderCircle,
  type LucideIcon,
} from "lucide-react";
import { useI18n } from "../i18n";
import type { Clip, ClipVoice } from "./libraryApi";
import { clipSteps, type StepState } from "./progress";

const looks: Record<StepState, { icon: LucideIcon; className: string }> = {
  done: { icon: CircleCheck, className: "text-ok-fg" },
  running: { icon: LoaderCircle, className: "text-accent-ink" },
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
}: {
  clip: Clip;
  voice: ClipVoice | null;
}) {
  const { t } = useI18n();
  return (
    <ol
      aria-label={t("clip.progress.label")}
      className="m-0 grid list-none grid-cols-2 gap-x-4 gap-y-2 rounded-xl border border-line bg-side p-3sm:grid-cols-3 xl:grid-cols-5"
    >
      {clipSteps(clip, voice).map((step) => {
        const { icon: Icon, className } = looks[step.state];
        const stateText = step.detail
          ? t("clip.progress.partial", step.detail)
          : t(`clip.progress.state.${step.state}`);
        return (
          <li key={step.key} className="flex items-start gap-2">
            <Icon
              className={`mt-0.5 size-4 shrink-0 ${className}`}
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
