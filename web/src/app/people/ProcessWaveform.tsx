import { Pause, Play } from "lucide-react";
import { useMemo } from "react";
import { useI18n } from "../i18n";
import type { LocalSpeaker, TranscriptSegment } from "../library/libraryApi";

/** One colour per voice of the clip, by rank: the same in the waveform, the queue and the card. */
const COLORS = [
  "#5b5bd6",
  "#e5484d",
  "#12a594",
  "#f76b15",
  "#8e4ec6",
  "#0090ff",
  "#ad7f58",
];
export const processColor = (rank: number) => COLORS[rank % COLORS.length];

/** Full height of a peak level. */
const LEVELS = 255;

/** The time as minutes and seconds. */
export const clock = (seconds: number) =>
  `${Math.floor(seconds / 60)}:${String(Math.floor(seconds % 60)).padStart(2, "0")}`;

type Bar = { height: number; owner?: number };

/**
 * Bars of a stretch of the audio, each one coloured by the voice speaking there; with a focus,
 * only that voice keeps its colour. Bars come from the real peaks of the audio track.
 */
export function PeakBars({
  peaks,
  segments,
  colors,
  total,
  from = 0,
  to,
  focusId,
  bars,
  height,
  playhead,
  onSeek,
  label,
}: {
  peaks?: number[];
  segments: TranscriptSegment[];
  /** The colour of each voice (by id); an ignored voice is absent and drawn neutral. */
  colors: Map<number, string>;
  total: number;
  from?: number;
  to?: number;
  focusId?: number;
  bars: number;
  height: number;
  playhead?: number;
  onSeek?: (seconds: number) => void;
  label?: string;
}) {
  const end = to ?? total;
  const span = Math.max(0.001, end - from);
  const shown = useMemo<Bar[]>(() => {
    return Array.from({ length: bars }, (_, index) => {
      const start = from + (index / bars) * span;
      const stop = from + ((index + 1) / bars) * span;
      const middle = (start + stop) / 2;
      const owner = segments.find(
        (segment) => middle >= segment.start && middle < segment.end,
      )?.local_speaker_id;
      let level = 0;
      if (peaks && peaks.length > 0 && total > 0) {
        const first = Math.floor((start / total) * peaks.length);
        const last = Math.max(
          first,
          Math.ceil((stop / total) * peaks.length) - 1,
        );
        for (let i = first; i <= last && i < peaks.length; i++)
          level = Math.max(level, peaks[i]);
      }
      return {
        height: Math.max(0.06, Math.min(1, level / LEVELS)),
        owner: owner ?? undefined,
      };
    });
  }, [peaks, segments, total, from, span, bars]);

  const neutral = "var(--line-strong)";
  const colorOf = (bar: Bar) => {
    if (bar.owner === undefined) return neutral;
    if (focusId !== undefined && bar.owner !== focusId) return neutral;
    return colors.get(bar.owner) ?? neutral;
  };
  const position =
    playhead !== undefined && playhead >= from && playhead <= end
      ? ((playhead - from) / span) * bars
      : undefined;

  return (
    <svg
      viewBox={`0 0 ${bars} 100`}
      preserveAspectRatio="none"
      style={{ height }}
      className={`w-full ${onSeek ? "cursor-pointer" : ""}`}
      role={label ? "img" : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      onClick={(event) => {
        if (!onSeek) return;
        const box = event.currentTarget.getBoundingClientRect();
        if (box.width <= 0) return;
        onSeek(from + ((event.clientX - box.left) / box.width) * span);
      }}
    >
      {shown.map((bar, index) => (
        <rect
          key={index}
          x={index + 0.15}
          width={0.7}
          y={50 - bar.height * 48}
          height={bar.height * 96}
          rx={0.35}
          fill={colorOf(bar)}
        />
      ))}
      {position !== undefined && (
        <line
          x1={position}
          x2={position}
          y1={0}
          y2={100}
          stroke="var(--ink)"
          strokeWidth={0.4}
          vectorEffect="non-scaling-stroke"
        />
      )}
    </svg>
  );
}

/**
 * The waveform of the whole conversation, coloured by voice with the voice in progress
 * highlighted, and one lane per voice under it where that voice speaks. A click in the
 * waveform or in a lane places the reader there; the keyboard places it at the voice's first turn.
 */
export function ProcessWaveform({
  speakers,
  segments,
  peaks,
  duration,
  playhead,
  playing,
  onToggle,
  onSeek,
  activeId,
}: {
  speakers: LocalSpeaker[];
  segments: TranscriptSegment[];
  peaks?: number[];
  duration?: number | null;
  playhead: number;
  playing: boolean;
  onToggle: () => void;
  onSeek: (seconds: number) => void;
  activeId?: number;
}) {
  const { t } = useI18n();
  const total =
    typeof duration === "number" && duration > 0
      ? duration
      : Math.max(0, ...segments.map((segment) => segment.end));
  const colors = useMemo(
    () =>
      new Map(
        speakers
          .map((speaker, rank) => [speaker.id, processColor(rank)] as const)
          .filter(
            ([id]) =>
              speakers.find((speaker) => speaker.id === id)?.disposition !==
              "ignored",
          ),
      ),
    [speakers],
  );
  const Icon = playing ? Pause : Play;
  return (
    <div className="flex flex-col gap-3 rounded-2xl border border-line bg-side p-4">
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          aria-pressed={playing}
          aria-label={t(
            playing ? "people.process.pause" : "people.process.play",
          )}
          onClick={onToggle}
          disabled={total <= 0}
          className="flex size-11 shrink-0 items-center justify-center rounded-full bg-accent-solid text-accent-fg md:size-10"
        >
          <Icon className="size-4" aria-hidden="true" />
        </button>
        <span className="text-sm text-mute tabular-nums">
          {clock(playhead)}
        </span>
        <ul
          aria-hidden="true"
          className="m-0 ml-auto flex list-none flex-wrap gap-3 p-0 text-xs"
        >
          {speakers.map((speaker, rank) => (
            <li
              key={speaker.id}
              className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 ${
                speaker.id === activeId ? "bg-surface ring-1 ring-line" : ""
              }`}
            >
              <span
                className="size-2.5 shrink-0 rounded-full"
                style={{
                  backgroundColor:
                    speaker.disposition === "ignored"
                      ? "var(--line-strong)"
                      : processColor(rank),
                }}
              />
              {speaker.person?.display_name ?? (
                <span className="lowercase">{speaker.label}</span>
              )}
            </li>
          ))}
        </ul>
      </div>
      <PeakBars
        peaks={peaks}
        segments={segments}
        colors={colors}
        total={total}
        focusId={activeId}
        bars={240}
        height={96}
        playhead={playhead}
        onSeek={onSeek}
        label={t("people.process.waveformLabel")}
      />
      <ul
        aria-label={t("clip.voices.tracksLabel")}
        className="m-0 flex list-none flex-col gap-0.5 p-0"
      >
        {speakers.map((speaker, rank) => {
          const turns = segments.filter(
            (segment) => segment.local_speaker_id === speaker.id,
          );
          return (
            <li
              key={speaker.id}
              data-active={speaker.id === activeId ? "true" : undefined}
            >
              <button
                type="button"
                aria-label={t("clip.voices.track", { label: speaker.label })}
                disabled={total <= 0}
                onClick={(event) => {
                  if (event.detail === 0) {
                    const first = turns[0];
                    if (first) onSeek(first.start);
                    return;
                  }
                  const box = event.currentTarget.getBoundingClientRect();
                  const fraction =
                    box.width > 0 ? (event.clientX - box.left) / box.width : 0;
                  onSeek(Math.max(0, Math.min(1, fraction)) * total);
                }}
                className="flex h-6 w-full cursor-pointer items-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-accent disabled:cursor-default"
              >
                <span className="relative block h-2 w-full rounded-full bg-side">
                  {total > 0 &&
                    turns.map((turn) => (
                      <span
                        key={turn.start}
                        className="absolute top-0 h-2 rounded-full"
                        style={{
                          left: `${(turn.start / total) * 100}%`,
                          width: `${((turn.end - turn.start) / total) * 100}%`,
                          background:
                            speaker.disposition === "ignored"
                              ? "var(--line-strong)"
                              : processColor(rank),
                        }}
                      />
                    ))}
                </span>
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
