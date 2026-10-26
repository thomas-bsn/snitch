import { Pencil } from "lucide-react";
import { useI18n } from "../i18n";
import { formatTimecode } from "./clipFormat";
import type { TranscriptSegment } from "./libraryApi";
import { SPEAKER_LOOKS, speakerState } from "./speakerState";

function Segment({
  segment,
  active,
  onSeek,
}: {
  segment: TranscriptSegment;
  active: boolean;
  onSeek: (seconds: number) => void;
}) {
  const { t } = useI18n();
  const time = formatTimecode(segment.start);
  const state = speakerState(segment);
  const look = state ? SPEAKER_LOOKS[state] : null;
  const name =
    segment.local_speaker_person_name ??
    segment.local_speaker_label ??
    t("clip.transcript.unknownSpeaker");
  return (
    <li
      aria-current={active ? "true" : undefined}
      className={`flex gap-3 rounded-lg p-2.5 ${active ? "bg-accent-soft" : ""}`}
    >
      <button
        type="button"
        onClick={() => onSeek(segment.start)}
        aria-label={t("clip.transcript.play", { time })}
        className="h-8 min-w-12 shrink-0 self-start rounded-md px-1.5 text-xs font-medium text-accent-ink tabular-nums underline-offset-2 hover:underline md:h-6"
      >
        {time}
      </button>
      <div className="min-w-0 flex-1">
        {look && (
          <div className="mb-1 flex flex-wrap items-center gap-2">
            <span className="rounded-full bg-side px-2 py-px text-xs font-medium text-ink">
              {name}
            </span>
            <span
              className={`inline-flex items-center gap-1 rounded-full px-2 py-px text-[11px] font-medium ${look.className}`}
            >
              <look.icon className="size-3" />
              {t(look.label)}
            </span>
          </div>
        )}
        <p className="m-0 leading-relaxed">{segment.text}</p>
        {segment.corrected && (
          <p className="mt-1 flex items-center gap-1 text-xs text-mute">
            <Pencil className="size-3" aria-hidden="true" />
            {segment.corrected_by
              ? t("clip.transcript.textCorrected", {
                  author: segment.corrected_by,
                })
              : t("clip.transcript.textCorrectedAnon")}
          </p>
        )}
      </div>
    </li>
  );
}

/** The transcript, segment by segment, with the speaker and how settled it is. */
export function TranscriptTab({
  segments,
  playhead,
  onSeek,
}: {
  segments: TranscriptSegment[];
  playhead: number;
  onSeek: (seconds: number) => void;
}) {
  const { t } = useI18n();
  if (segments.length === 0) {
    return <p className="p-4 text-mute">{t("clip.transcript.empty")}</p>;
  }
  // The segment being played is the last one that has started.
  let activeIndex = -1;
  segments.forEach((segment, index) => {
    if (segment.start <= playhead) activeIndex = index;
  });
  return (
    <ol
      aria-label={t("clip.transcript.listLabel")}
      className="m-0 flex list-none flex-col gap-0.5 p-2"
    >
      {segments.map((segment, index) => (
        <Segment
          key={segment.index ?? index}
          segment={segment}
          active={index === activeIndex}
          onSeek={onSeek}
        />
      ))}
    </ol>
  );
}
