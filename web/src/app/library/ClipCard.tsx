import { Play } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";
import { useI18n } from "../i18n";
import { ClipStatePill } from "./ClipStatePill";
import { formatTimecode } from "./clipFormat";
import type { Clip } from "./libraryApi";

// The server makes no image of a clip: the tile is a quiet gradient picked from the clip's
// number, so the same clip always looks the same.
const tiles = [
  "from-accent-soft to-side",
  "from-ok-bg to-side",
  "from-warn-bg to-side",
  "from-avatar-bg to-accent-soft",
  "from-hit to-side",
  "from-bad-bg to-side",
];

/** A clip of the library grid: tile, name, owner, date and state, one link to its page. */
export function ClipCard({ clip, from }: { clip: Clip; from: string }) {
  const { t, language } = useI18n();
  const date = useMemo(
    () =>
      new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
        new Date(clip.recorded_at),
      ),
    [clip.recorded_at, language],
  );
  const tile = tiles[clip.id % tiles.length];
  return (
    <li className="min-w-0">
      <article className="relative flex flex-col gap-2">
        <div
          className={`relative flex aspect-video items-center justify-center rounded-lg border border-line bg-linear-to-br ${tile}`}
        >
          <Play className="size-7 text-mute" aria-hidden="true" />
          {typeof clip.duration_seconds === "number" && (
            <span className="absolute right-1.5 bottom-1.5 rounded bg-black/75 px-1.5 py-0.5 text-[11px] font-medium text-white">
              {formatTimecode(clip.duration_seconds)}
            </span>
          )}
        </div>
        <h2 className="truncate text-sm font-medium">
          <Link
            to={`/clips/${clip.id}`}
            state={{ from }}
            aria-label={t("library.card.open", { name: clip.original_name })}
            className="after:absolute after:inset-0 after:rounded-lg"
          >
            {clip.original_name}
          </Link>
        </h2>
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-mute">
          <span>
            {t("library.card.by", {
              owner: clip.owner_deleted
                ? t("library.deletedOwner")
                : clip.owner,
            })}
          </span>
          <span aria-hidden="true">·</span>
          <time dateTime={clip.recorded_at}>{date}</time>
          <span className="ml-auto">
            <ClipStatePill status={clip.status} />
          </span>
        </div>
      </article>
    </li>
  );
}
