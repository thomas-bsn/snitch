import { Link } from "react-router";
import { useI18n } from "../i18n";
import { formatTimecode } from "../library/clipFormat";
import { LevelBadge } from "../people/LevelBadge";
import { excerpt } from "./highlight";
import type { SearchResult } from "./searchApi";
import { SpeakerAvatar } from "./SpeakerAvatar";

const pill =
  "rounded-full bg-side px-2 py-0.5 text-[11px] font-medium text-mute";

/** One passage found: who speaks, the quote with the hit, and the timecode that opens it. */
export function SearchResultItem({
  result,
  from,
}: {
  result: SearchResult;
  /** The address of the search, so that the clip can lead back to it. */
  from: string;
}) {
  const { t } = useI18n();
  const { before, hit, after } = excerpt(result.context, result.matched_text);
  const seconds = Math.floor(result.timestamp);
  const timecode = formatTimecode(result.timestamp);
  const playable = result.status === "ready";
  const name = result.person
    ? result.person.display_name
    : t("search.result.unknownSpeaker");

  return (
    <li className="flex gap-3 border-b border-line px-3.5 py-3 last:border-b-0 hover:bg-side md:gap-4 md:px-4 md:py-3.5">
      <span className="mt-0.5 flex shrink-0 rounded-full ring-1 ring-line">
        <SpeakerAvatar person={result.person} />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span
            className={`text-[13px] font-semibold ${
              result.person ? "text-ink" : "italic text-mute"
            }`}
          >
            {name}
          </span>
          <LevelBadge level={result.decision_level} />
          {result.corrected && (
            <span className={pill}>{t("search.result.corrected")}</span>
          )}
          {result.visibility === "private" && (
            <span className={pill}>{t("search.result.private")}</span>
          )}
          {!playable && (
            <span className="rounded-full bg-warn-bg px-2 py-0.5 text-[11px] font-medium text-warn-fg">
              {t("search.result.unavailable")}
            </span>
          )}
          {playable ? (
            <Link
              to={`/clips/${result.clip_id}?t=${seconds}`}
              state={{ from }}
              aria-label={t("search.result.open", {
                clip: result.clip_name,
                time: timecode,
              })}
              className="ml-auto inline-flex min-h-11 items-center font-mono text-[12px] font-semibold text-accent-ink underline underline-offset-2 hover:text-accent md:min-h-7"
            >
              {timecode}
            </Link>
          ) : (
            <span className="ml-auto font-mono text-[12px] text-mute">
              {timecode}
            </span>
          )}
        </div>
        <p className="m-0 text-[14px] leading-relaxed [overflow-wrap:anywhere]">
          “{before}
          {hit && (
            <mark className="rounded-sm bg-hit px-0.5 text-ink">{hit}</mark>
          )}
          {after}”
        </p>
      </div>
    </li>
  );
}
