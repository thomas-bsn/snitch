import { SegmentActions } from "./SegmentActions";
import type { components } from "../../api.gen";
import { useFormatDate } from "../admin/format";
import { Pencil } from "lucide-react";
import { useCallback } from "react";
import { useLoaded } from "../api/loaded";
import { useI18n } from "../i18n";
import { formatTimecode } from "./clipFormat";
import { fetchSegmentHistory, type TranscriptSegment } from "./libraryApi";
import { SPEAKER_LOOKS, speakerState } from "./speakerState";

/** Every track a passage was captured on; the grouped ones with the acoustic proof. */
function SourceProvenance({ segment }: { segment: TranscriptSegment }) {
  const { t } = useI18n();
  const sources = segment.sources ?? [];
  if (sources.length < 2) return null;
  const proofs = new Map(
    (segment.grouping ?? []).map((g) => [g.source.track_index, g]),
  );
  return (
    <details className="mt-1 text-xs text-mute">
      <summary className="cursor-pointer">
        {t("clip.transcript.sourcesSummary", { count: sources.length })}
      </summary>
      <ul className="m-0 mt-1 list-none p-0">
        {sources.map((source, position) => {
          const proof = proofs.get(source.track_index);
          return (
            <li key={`${source.job_id}-${source.segment_index}`}>
              <span className="font-medium">
                {t("clip.transcript.sourceTrack", {
                  track: source.track_index,
                })}
              </span>
              {" — "}
              {proof
                ? t(
                    proof.strategy === "gain"
                      ? "clip.transcript.sourceGain"
                      : "clip.transcript.sourceExact",
                    {
                      gain: (proof.gain ?? 1).toFixed(2),
                      from: formatTimecode(proof.matched_start),
                      to: formatTimecode(proof.matched_end),
                    },
                  )
                : position === 0 &&
                  t("clip.transcript.sourceRetained", {
                    track: source.track_index,
                  })}
            </li>
          );
        })}
      </ul>
    </details>
  );
}

function SpeakerCorrection({
  clipId,
  index,
  version,
}: {
  clipId: number;
  index: number;
  version: number;
}) {
  const { t } = useI18n();
  const format = useFormatDate();
  const { state } = useLoaded(
    useCallback(async () => {
      try {
        return await fetchSegmentHistory(clipId, index);
      } catch {
        return { ok: false as const, code: "unavailable" };
      }
    }, [clipId, index, version]),
  );
  if (state.status !== "ready") return null;
  const event = state.value.items.find((entry) => entry.kind === "speaker");
  return event ? (
    <p className="mt-1 text-xs text-mute">
      {t("segmentEdit.speakerBy", {
        author: event.author.username,
        date: format(event.created_at),
      })}
    </p>
  ) : null;
}

function Segment({
  segment,
  active,
  onSeek,
  clipId,
  jobId,
  speakers,
  onChanged,
}: {
  clipId: number;
  jobId?: number;
  speakers: NonNullable<
    components["schemas"]["Transcription"]["local_speakers"]
  >;
  onChanged: () => void;
  segment: TranscriptSegment;
  active: boolean;
  onSeek: (seconds: number) => void;
}) {
  const { t } = useI18n();
  const formatDate = useFormatDate();
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
      className={`group relative flex gap-3 border-b border-line px-4 py-3 pr-28 md:pr-[4.5rem] ${active ? "bg-hit" : ""}`}
    >
      <button
        type="button"
        onClick={() => onSeek(segment.start)}
        aria-label={t("clip.transcript.play", { time })}
        className="h-8 min-w-12 shrink-0 self-start rounded px-1.5 text-xs text-accent-ink tabular-nums hover:bg-accent-soft md:h-6"
      >
        {time}
      </button>
      <div className="min-w-0 flex-1">
        {look && (
          <div className="mb-1 flex flex-wrap items-center gap-2">
            <span
              className={`inline-flex w-fit rounded-full px-2 py-0.5 text-xs font-medium ${
                state === "suggestion"
                  ? "border border-dashed border-accent text-accent-ink"
                  : segment.local_speaker_person_name
                    ? "bg-accent-soft text-accent-ink"
                    : "bg-side text-mute"
              }`}
            >
              {state === "suggestion" ? `${name} ?` : name}
            </span>
            <span className="sr-only">{t(look.label)}</span>
          </div>
        )}
        <p className="m-0 leading-relaxed">{segment.text}</p>
        {segment.corrected && (
          <p className="mt-1 text-xs text-mute">
            {t("segmentEdit.original", { text: segment.original_text ?? "" })}
          </p>
        )}
        {segment.local_speaker_corrected && segment.index !== undefined && (
          <SpeakerCorrection
            clipId={clipId}
            index={segment.index}
            version={segment.correction_version ?? 0}
          />
        )}
        {segment.corrected && (
          <p className="mt-1 flex items-center gap-1 text-xs text-mute">
            <Pencil className="size-3" aria-hidden="true" />
            {segment.corrected_at && (
              <span>{formatDate(segment.corrected_at)} · </span>
            )}
            {segment.corrected_by
              ? t("clip.transcript.textCorrected", {
                  author: segment.corrected_by,
                })
              : t("clip.transcript.textCorrectedAnon")}
          </p>
        )}
        <SourceProvenance segment={segment} />
        <SegmentActions
          clipId={clipId}
          jobId={jobId}
          segment={segment}
          speakers={speakers}
          onChanged={onChanged}
        />
      </div>
    </li>
  );
}

/** The transcript, segment by segment, with the speaker and how settled it is. */
export function TranscriptTab({
  segments,
  playhead,
  onSeek,
  clipId,
  jobId,
  speakers,
  onChanged,
}: {
  clipId: number;
  jobId?: number;
  speakers: NonNullable<
    components["schemas"]["Transcription"]["local_speakers"]
  >;
  onChanged: () => void;
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
      className="m-0 flex list-none flex-col p-0"
    >
      {segments.map((segment, index) => (
        <Segment
          key={segment.index ?? index}
          segment={segment}
          clipId={clipId}
          jobId={jobId}
          speakers={speakers}
          onChanged={onChanged}
          active={index === activeIndex}
          onSeek={onSeek}
        />
      ))}
    </ol>
  );
}
