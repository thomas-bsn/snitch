import { useEffect, useState } from "react";
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
  personId,
  size = "md",
}: {
  name: string;
  personId?: number;
  size?: "xs" | "md" | "lg" | "record";
}) {
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [personId]);
  return (
    <span
      aria-hidden="true"
      className={`inline-flex shrink-0 items-center justify-center rounded-full bg-avatar-bg font-semibold text-accent-ink ${
        size === "xs"
          ? "size-6 text-[9px]"
          : size === "lg"
            ? "size-14 text-xs"
            : size === "record"
              ? "size-12 text-xs"
              : "size-10 text-xs"
      }`}
    >
      {personId && !failed ? (
        <img
          src={`/api/persons/${personId}/avatar`}
          alt=""
          className="size-full rounded-full object-cover"
          onError={() => setFailed(true)}
        />
      ) : (
        initials(name)
      )}
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

/** A short date such as "28 oct." (the year is left out: these lists are about recent clips). */
export function shortDate(iso: string, language: string): string {
  return new Intl.DateTimeFormat(language, {
    day: "numeric",
    month: "short",
  }).format(new Date(iso));
}

/** A duration as m:ss, or h:mm:ss from an hour. */
export function clock(seconds: number): string {
  const total = Math.max(0, Math.round(seconds));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = String(total % 60).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${s}` : `${m}:${s}`;
}

/** Speaking time as "6 h 52 min" or "41 min", without the trailing words. */
export function speechShort(seconds: number): string {
  const minutes = Math.round(seconds / 60);
  return minutes < 60
    ? `${minutes} min`
    : `${Math.floor(minutes / 60)} h ${minutes % 60} min`;
}
