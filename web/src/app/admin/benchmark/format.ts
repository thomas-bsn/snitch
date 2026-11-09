import type { Language } from "../../preferences";

/** A rate between 0 and 1 as a percentage in the reader's language ("4,2 %" in French). */
export function percent(language: Language, value?: number | null): string {
  if (value === undefined || value === null) return "–";
  return new Intl.NumberFormat(language, {
    style: "percent",
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  }).format(value);
}

export function decimal(language: Language, value?: number | null): string {
  if (value === undefined || value === null) return "–";
  return new Intl.NumberFormat(language, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

/** Seconds as "3 min 12 s". */
export function elapsed(seconds?: number | null): string {
  if (seconds === undefined || seconds === null) return "–";
  const total = Math.round(seconds);
  const minutes = Math.floor(total / 60);
  return minutes > 0 ? `${minutes} min ${total % 60} s` : `${total} s`;
}

export function minutes(seconds: number): number {
  return Math.round(seconds / 60);
}

export function dateTime(language: Language, iso: string): string {
  return new Intl.DateTimeFormat(language, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(iso));
}
