import { Link } from "react-router";
import { useFormatDate } from "../admin/format";
import { useI18n } from "../i18n";
import { formatTimecode } from "../library/clipFormat";
import { LevelBadge } from "../people/LevelBadge";
import { excerpt } from "./highlight";
import type { SearchResult } from "./searchApi";

/** One passage found: its timecode opens the clip at that moment. */
export function SearchResultItem({
  result,
  from,
}: {
  result: SearchResult;
  /** The address of the search, so that the clip can lead back to it. */
  from: string;
}) {
  const { t } = useI18n();
  const formatDate = useFormatDate();
  const { before, hit, after } = excerpt(result.context, result.matched_text);
  const seconds = Math.floor(result.timestamp);
  const timecode = formatTimecode(result.timestamp);
  const playable = result.status === "ready";

  return (
    <li className="flex gap-3 border-b border-line px-4 py-3.5 last:border-b-0 md:gap-4">
      <div className="w-14 shrink-0 pt-0.5 md:w-16">
        {playable ? (
          <Link
            to={`/clips/${result.clip_id}?t=${seconds}`}
            state={{ from }}
            aria-label={t("search.result.open", {
              clip: result.clip_name,
              time: timecode,
            })}
            className="inline-flex min-h-11 items-center font-mono text-[13px] font-medium text-accent-ink underline underline-offset-2 hover:text-accent md:min-h-7"
          >
            {timecode}
          </Link>
        ) : (
          <span className="font-mono text-[13px] text-mute">{timecode}</span>
        )}
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <p className="m-0 text-[15px] leading-relaxed [overflow-wrap:anywhere]">
          “{before}
          {hit && (
            <mark className="rounded-sm bg-hit px-0.5 text-ink">{hit}</mark>
          )}
          {after}”
        </p>
        <p className="m-0 flex flex-wrap items-center gap-x-2 gap-y-1 text-[13px] text-mute">
          <Link
            to={`/clips/${result.clip_id}`}
            state={{ from }}
            className="font-medium text-ink hover:text-accent-ink"
          >
            {result.clip_name}
          </Link>
          <span aria-hidden="true">·</span>
          <span>
            {result.person
              ? result.person.display_name
              : t("search.result.unknownSpeaker")}
          </span>
          <span aria-hidden="true">·</span>
          <span>{formatDate(result.recorded_at)}</span>
          <span aria-hidden="true">·</span>
          <span>{t(`search.matches.${result.match_type}`)}</span>
        </p>
        <p className="m-0 flex flex-wrap items-center gap-2">
          <LevelBadge level={result.decision_level} />
          {result.corrected && (
            <span className="rounded-full bg-side px-2.5 py-0.5 text-xs font-medium text-mute">
              {t("search.result.corrected")}
            </span>
          )}
          {result.visibility === "private" && (
            <span className="rounded-full bg-side px-2.5 py-0.5 text-xs font-medium text-mute">
              {t("search.result.private")}
            </span>
          )}
          {!playable && (
            <span className="rounded-full bg-warn-bg px-2.5 py-0.5 text-xs font-medium text-warn-fg">
              {t("search.result.unavailable")}
            </span>
          )}
        </p>
      </div>
    </li>
  );
}
