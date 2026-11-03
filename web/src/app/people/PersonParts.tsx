import { useI18n } from "../i18n";

/**
 * Validated samples from which a Reference counts as sufficient. The spec fixes no
 * threshold: below this count (but above zero) the Reference is "weak" and the sheet warns.
 */
export const GOOD_REFERENCE_MIN_SAMPLES = 3;

export type ReferenceLevel = "good" | "weak" | "none";

export function referenceLevel(samples: number): ReferenceLevel {
  if (samples <= 0) return "none";
  return samples < GOOD_REFERENCE_MIN_SAMPLES ? "weak" : "good";
}

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
      className={`inline-flex shrink-0 items-center justify-center rounded-full bg-avatar-bg font-semibold text-accent-ink ${
        size === "lg" ? "size-14 text-xs" : "size-10 text-xs"
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

const levels: Record<ReferenceLevel, { className: string }> = {
  good: { className: "bg-ok-bg text-ok-fg" },
  weak: { className: "bg-warn-bg text-warn-fg" },
  none: { className: "bg-warn-bg text-warn-fg" },
};

/** The Reference quality as a pill: always the word and a symbol, never the color alone. */
export function ReferenceBadge({ samples }: { samples: number }) {
  const { t } = useI18n();
  const level = referenceLevel(samples);
  const { className } = levels[level];
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${className}`}
    >
      {t("people.reference.levelLabel", {
        level: t(`people.reference.${level}`),
      })}
    </span>
  );
}
