import {
  CircleCheck,
  CircleOff,
  TriangleAlert,
  type LucideIcon,
} from "lucide-react";
import { useI18n } from "../i18n";
import { type ReferenceLevel, referenceLevel } from "./peopleApi";

/** Two letters from the first words of the name: the avatar when there is no photo. */
export function initials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  const letters = words.slice(0, 2).map((word) => Array.from(word)[0] ?? "");
  return letters.join("").toUpperCase() || "?";
}

export function Avatar({
  name,
  size = "md",
}: {
  name: string;
  size?: "md" | "lg";
}) {
  return (
    <span
      aria-hidden="true"
      className={`inline-flex shrink-0 items-center justify-center rounded-full bg-accent-soft font-semibold text-accent-ink ${
        size === "lg" ? "size-14 text-lg" : "size-10 text-sm"
      }`}
    >
      {initials(name)}
    </span>
  );
}

type Translate = ReturnType<typeof useI18n>["t"];

/** Speaking time as hours and minutes, or minutes alone under an hour. */
export function speechText(seconds: number, t: Translate): string {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return t("people.figures.minutes", { minutes });
  return t("people.figures.hours", {
    hours: Math.floor(minutes / 60),
    minutes: minutes % 60,
  });
}

const levels: Record<ReferenceLevel, { icon: LucideIcon; className: string }> =
  {
    good: { icon: CircleCheck, className: "bg-ok-bg text-ok-fg" },
    weak: { icon: TriangleAlert, className: "bg-warn-bg text-warn-fg" },
    none: { icon: CircleOff, className: "bg-bad-bg text-bad-fg" },
  };

/** The Reference quality as a pill: always the word and a symbol, never the color alone. */
export function ReferenceBadge({ samples }: { samples: number }) {
  const { t } = useI18n();
  const level = referenceLevel(samples);
  const { icon: Icon, className } = levels[level];
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${className}`}
    >
      <Icon className="size-3.5" aria-hidden="true" />
      {t("people.reference.levelLabel", {
        level: t(`people.reference.${level}`),
      })}
    </span>
  );
}
