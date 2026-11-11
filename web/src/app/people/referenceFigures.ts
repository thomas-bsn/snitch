import { en } from "../i18n/en";
import { fr } from "../i18n/fr";
import { translate, type Messages } from "../i18n/messages";
import type { Language } from "../preferences";

const catalogs: Record<Language, Messages> = { en, fr };

/** From this count on, a number of videos is written in compact notation ("12K", "12 k"). */
const COMPACT_FROM = 10_000;

/**
 * The number of videos of a directory card, agreed in number: `short` is what the card shows (compact from
 * 10,000), `exact` the full number for the accessible name and the tooltip.
 */
export function videoCountText(
  count: number,
  language: Language,
): { short: string; exact: string } {
  const plural =
    new Intl.PluralRules(language).select(count) === "one" ? "one" : "other";
  const key = `people.persons.videos.${plural}` as const;
  const exactNumber = new Intl.NumberFormat(language).format(count);
  const shortNumber =
    count >= COMPACT_FROM
      ? new Intl.NumberFormat(language, { notation: "compact" }).format(count)
      : exactNumber;
  return {
    short: translate(catalogs[language], key, { count: shortNumber }),
    exact: translate(catalogs[language], key, { count: exactNumber }),
  };
}

/**
 * A duration such as "48 s", "3 min 12 s" or "1 h 05": never rounded down to nothing, so a few seconds of
 * speech never read "0 min".
 */
export function durationText(seconds: number): string {
  if (!(seconds > 0)) return "0 s";
  if (seconds < 1) return "< 1 s";
  const total = Math.round(seconds);
  if (total < 60) return `${total} s`;
  if (total < 3600) {
    const m = Math.floor(total / 60);
    const s = total % 60;
    return s === 0 ? `${m} min` : `${m} min ${s} s`;
  }
  const totalMinutes = Math.round(total / 60);
  const h = Math.floor(totalMinutes / 60);
  return `${h} h ${String(totalMinutes % 60).padStart(2, "0")}`;
}
