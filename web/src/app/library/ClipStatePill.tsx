import {
  Clock,
  CircleCheck,
  FileX,
  TriangleAlert,
  type LucideIcon,
} from "lucide-react";
import { useI18n } from "../i18n";
import { CLIP_STATES, type ClipState } from "./libraryApi";

const looks: Record<ClipState, { icon: LucideIcon; className: string }> = {
  ready: { icon: CircleCheck, className: "bg-ok-bg text-ok-fg" },
  waiting_stability: { icon: Clock, className: "bg-warn-bg text-warn-fg" },
  unreadable: { icon: TriangleAlert, className: "bg-bad-bg text-bad-fg" },
  missing: { icon: FileX, className: "bg-bad-bg text-bad-fg" },
};

function isClipState(value: string | undefined): value is ClipState {
  return CLIP_STATES.some((state) => state === value);
}

/** The state of a clip's file: a label and a symbol, never the color alone. */
export function ClipStatePill({ status }: { status: string | undefined }) {
  const { t } = useI18n();
  if (!isClipState(status)) return null;
  const { icon: Icon, className } = looks[status];
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ${className}`}
    >
      <Icon className="size-3.5" />
      {t(`library.state.${status}`)}
    </span>
  );
}
