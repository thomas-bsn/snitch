import { ArrowRight, Play } from "lucide-react";
import { useId, useMemo } from "react";
import { Link } from "react-router";
import { useI18n } from "../i18n";
import { ExplorerThumbnail } from "../library/ExplorerThumbnail";
import type { SearchResult } from "./searchApi";
import { SearchResultItem } from "./SearchResultItem";
import { SpeakerAvatar } from "./SpeakerAvatar";

export type ClipGroup = {
  clipId: number;
  clipName: string;
  recordedAt: string | null;
  items: SearchResult[];
};

/** Groups the page's passages by clip, keeping the API order. */
export function groupResults(items: SearchResult[]): ClipGroup[] {
  const groups = new Map<number, ClipGroup>();
  for (const item of items) {
    const group = groups.get(item.clip_id);
    if (group) group.items.push(item);
    else
      groups.set(item.clip_id, {
        clipId: item.clip_id,
        clipName: item.clip_name,
        recordedAt: item.recorded_at,
        items: [item],
      });
  }
  return Array.from(groups.values());
}

/** One clip's passages: the clip, the voices heard in it, then each quote. */
export function SearchClipGroup({
  group,
  from,
}: {
  group: ClipGroup;
  from: string;
}) {
  const { t, language } = useI18n();
  const titleId = useId();
  const day = useMemo(() => {
    const format = new Intl.DateTimeFormat(language, { dateStyle: "medium" });
    return (iso: string) => format.format(new Date(iso));
  }, [language]);
  const voices = useMemo(() => {
    const seen = new Map<string, SearchResult["person"]>();
    for (const item of group.items) {
      const key = item.person ? `person-${item.person.id}` : "unknown";
      if (!seen.has(key)) seen.set(key, item.person);
    }
    return Array.from(seen.entries()).map(([key, person]) => ({
      key,
      person,
      name: person?.display_name ?? t("search.result.unknownSpeaker"),
    }));
  }, [group.items, t]);
  const passages =
    group.items.length === 1
      ? t("search.passageOne")
      : t("search.passages", { count: group.items.length });
  const voicesLabel =
    voices.length === 1
      ? t("search.voiceOne")
      : t("search.voices", { count: voices.length });
  const sub = [
    group.recordedAt ? day(group.recordedAt) : "",
    passages,
    voicesLabel,
  ]
    .filter(Boolean)
    .join(" · ");
  const names = voices.map((voice) => voice.name).join(", ");
  const visibleVoices = voices.slice(0, 4);

  return (
    <section
      aria-labelledby={titleId}
      className="overflow-hidden rounded-xl border border-line bg-surface"
    >
      <header className="grid grid-cols-[60px_minmax(0,1fr)_auto] items-center gap-x-3 gap-y-2 border-b border-line bg-side px-3.5 py-2.5 lg:flex lg:flex-wrap">
        <span className="relative col-start-1 row-start-1 flex h-9 w-[60px] shrink-0 items-center justify-center overflow-hidden rounded-md bg-[#1b1b22] text-white/60 lg:h-11 lg:w-[76px]">
          <Play className="size-4" aria-hidden="true" />
          <ExplorerThumbnail clipId={group.clipId} />
        </span>
        <div className="col-span-2 col-start-2 row-start-1 min-w-0 lg:flex-1">
          <h2 id={titleId} className="m-0 text-sm font-semibold lg:truncate">
            <Link
              to={`/clips/${group.clipId}`}
              state={{ from }}
              className="hover:text-accent-ink"
            >
              {group.clipName}
            </Link>
          </h2>
          <p className="m-0 mt-0.5 text-xs text-mute">{sub}</p>
        </div>
        <div className="col-span-2 col-start-1 row-start-2 flex min-w-0 items-center gap-2 lg:order-none lg:ml-auto lg:w-auto">
          <span className="flex shrink-0 -space-x-1.5" aria-hidden="true">
            {visibleVoices.map((voice) => (
              <span
                key={voice.key}
                className="inline-flex rounded-full ring-2 ring-side"
              >
                <SpeakerAvatar person={voice.person} size="xs" />
              </span>
            ))}
            {voices.length > visibleVoices.length && (
              <span className="inline-flex size-6 items-center justify-center rounded-full bg-side text-[9px] font-semibold text-mute ring-2 ring-side">
                {t("search.group.moreVoices", {
                  count: voices.length - visibleVoices.length,
                })}
              </span>
            )}
          </span>
          <span className="min-w-0 truncate text-xs text-mute" title={names}>
            {names}
          </span>
        </div>
        <Link
          to={`/clips/${group.clipId}`}
          state={{ from }}
          className="col-start-3 row-start-2 inline-flex min-h-11 items-center justify-self-end gap-1.5 rounded-md border border-line-strong bg-surface px-2.5 text-xs font-medium text-mute hover:text-ink lg:order-none lg:ml-2 lg:min-h-7"
        >
          {t("search.group.openClip")}
          <ArrowRight className="size-3.5" aria-hidden="true" />
        </Link>
      </header>
      <ol
        aria-label={t("search.resultsInClip", { clip: group.clipName })}
        className="m-0 list-none p-0"
      >
        {group.items.map((item, index) => (
          <SearchResultItem
            key={`${item.clip_id}-${item.timestamp}-${index}`}
            result={item}
            from={from}
          />
        ))}
      </ol>
    </section>
  );
}
