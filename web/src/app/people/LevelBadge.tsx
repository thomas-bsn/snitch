import { Popover } from "@base-ui/react/popover";
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

const order: DecisionLevel[] = [
  "human",
  "automatic",
  "suggestion",
  "ambiguous",
  "unknown",
];

const pill =
  "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium";

function Content({ level, count }: { level: DecisionLevel; count?: number }) {
  const { t } = useI18n();
  const { icon: Icon } = levels[level];
  return (
    <>
      <Icon className="size-3.5" aria-hidden="true" />
      {t(`people.levels.${level}`)}
      {count !== undefined && <b className="font-semibold">{count}</b>}
    </>
  );
}

/** The legend of the five levels, opened from a badge. */
function Legend({ level }: { level: DecisionLevel }) {
  const { t } = useI18n();
  return (
    <Popover.Positioner sideOffset={6} collisionPadding={8} className="z-50">
      <Popover.Popup className="flex w-80 max-w-[calc(100vw-1rem)] flex-col gap-2 rounded-xl border border-line bg-surface p-4 text-sm text-ink shadow-pop outline-none">
        <Popover.Title className="font-semibold">
          {t(`people.levels.${level}`)}
        </Popover.Title>
        <Popover.Description className="text-mute">
          {t(`people.legend.definition.${level}`)}
        </Popover.Description>
        <p className="m-0 font-medium">
          {t("people.legend.reference", {
            answer: t(
              level === "human" ? "people.legend.yes" : "people.legend.no",
            ),
          })}
        </p>
        <p className="m-0 text-mute">{t("people.legend.others")}</p>
        <ul className="m-0 flex list-none flex-col gap-1.5 p-0">
          {order
            .filter((other) => other !== level)
            .map((other) => (
              <li key={other} className="flex flex-col gap-0.5">
                <span className={`${pill} w-fit ${levels[other].className}`}>
                  <Content level={other} />
                </span>
                <span className="text-mute">
                  {t(`people.legend.definition.${other}`)}
                </span>
              </li>
            ))}
        </ul>
      </Popover.Popup>
    </Popover.Positioner>
  );
}

/**
 * The five states of a voice: always a label and a symbol, never the color alone. A badge is
 * a button that opens the legend (click, Enter or touch, never the hover alone). Inside
 * another button or link, pass `interactive={false}`: it is then plain text, so that no
 * button is nested in another.
 */
export function LevelBadge({
  level,
  count,
  interactive = true,
}: {
  level: DecisionLevel;
  count?: number;
  interactive?: boolean;
}) {
  const { className } = levels[level];
  if (!interactive) {
    return (
      <span className={`${pill} ${className}`}>
        <Content level={level} count={count} />
      </span>
    );
  }
  return (
    <Popover.Root>
      <Popover.Trigger
        className={`${pill} relative cursor-pointer after:absolute after:inset-x-0 after:-inset-y-2.5 after:content-[''] ${className}`}
      >
        <Content level={level} count={count} />
      </Popover.Trigger>
      <Popover.Portal>
        <Legend level={level} />
      </Popover.Portal>
    </Popover.Root>
  );
}
