import {
  ArrowLeftRight,
  Check,
  CircleHelp,
  Cog,
  Equal,
  type LucideIcon,
} from "lucide-react";
import { useI18n } from "../i18n";
import type { DecisionLevel } from "./peopleApi";

const levels: Record<DecisionLevel, { icon: LucideIcon; className: string }> = {
  human: { icon: Check, className: "bg-ok-bg text-ok-fg" },
  automatic: { icon: Cog, className: "bg-accent-soft text-accent-ink" },
  suggestion: { icon: Equal, className: "bg-accent-soft text-accent-ink" },
  ambiguous: { icon: ArrowLeftRight, className: "bg-warn-bg text-warn-fg" },
  unknown: { icon: CircleHelp, className: "bg-bad-bg text-bad-fg" },
};

/** The five states of a voice: always a label and a symbol, never the color alone. */
export function LevelBadge({
  level,
  count,
}: {
  level: DecisionLevel;
  count?: number;
}) {
  const { t } = useI18n();
  const { icon: Icon, className } = levels[level];
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${className}`}
    >
      <Icon className="size-3.5" aria-hidden="true" />
      {t(`people.levels.${level}`)}
      {count !== undefined && <b className="font-semibold">{count}</b>}
    </span>
  );
}
