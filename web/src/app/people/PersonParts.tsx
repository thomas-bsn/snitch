import {
  CircleAlert,
  CircleCheck,
  CircleDashed,
  Video,
  type LucideIcon,
} from "lucide-react";
import { useEffect, useState } from "react";
import { useI18n } from "../i18n";
import type { PersonReference } from "./peopleApi";
import { videoCountText } from "./referenceFigures";

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

const levels: Record<
  PersonReference["level"],
  { className: string; icon: LucideIcon }
> = {
  good: { className: "bg-ok-bg text-ok-fg", icon: CircleCheck },
  weak: { className: "bg-warn-bg text-warn-fg", icon: CircleAlert },
  none: { className: "bg-warn-bg text-warn-fg", icon: CircleDashed },
};

/**
 * The Reference level decided by the server, compared with the other Persons: always the word and a symbol,
 * never the color alone, and why in the tooltip.
 */
export function ReferenceBadge({ reference }: { reference: PersonReference }) {
  const { t } = useI18n();
  const { className, icon: Icon } = levels[reference.level];
  return (
    <span
      title={t(`people.reference.basis.${reference.basis}`)}
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium whitespace-nowrap ${className}`}
    >
      <Icon className="size-3.5" aria-hidden="true" />
      {t("people.reference.levelLabel", {
        level: t(`people.reference.${reference.level}`),
      })}
    </span>
  );
}

/**
 * The number of videos of a card as a discreet pill: compact from 10,000, the exact number in the accessible
 * name and the tooltip.
 */
export function VideoCount({ count }: { count: number }) {
  const { language } = useI18n();
  const { short, exact } = videoCountText(count, language);
  return (
    <span
      title={exact}
      className="inline-flex items-center gap-1.5 rounded-full bg-side px-2.5 py-0.5 text-xs whitespace-nowrap text-mute tabular-nums"
    >
      <Video className="size-3.5" aria-hidden="true" />
      {short === exact ? (
        short
      ) : (
        <>
          <span aria-hidden="true">{short}</span>
          <span className="sr-only">{exact}</span>
        </>
      )}
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
