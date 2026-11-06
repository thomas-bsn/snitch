import { useMemo } from "react";
import { useI18n } from "../i18n";
import type { LocalSpeaker, TranscriptSegment } from "./libraryApi";

/** The colour of a voice, by its rank in the clip: the same in the waveform and beside the voice. */
const VOICE_COLORS = [
  "#5b5bd6",
  "#e5484d",
  "#12a594",
  "#f76b15",
  "#8e4ec6",
  "#0090ff",
  "#ad7f58",
];
export const voiceColor = (rank: number) =>
  VOICE_COLORS[rank % VOICE_COLORS.length];

/** Full height of a drawing; a peak is a level from 0 to this value, centred on the axis. */
const LEVELS = 255;

/** The bars of the peaks that fall inside `owned`, as one path (ADR 0034). */
function barsPath(peaks: number[], owned: Uint8Array, wanted: number) {
  let path = "";
  peaks.forEach((level, index) => {
    if (owned[index] !== wanted) return;
    const height = Math.max(4, Math.min(LEVELS, level));
    const top = (LEVELS - height) / 2;
    path += `M${index + 0.5} ${top}V${top + height}`;
  });
  return path;
}

/** The peaks a voice speaks over: its index plus one, 0 where nobody does. */
function owners(
  peaks: number[],
  tracks: { turns: TranscriptSegment[] }[],
  total: number,
) {
  const owned = new Uint8Array(peaks.length);
  tracks.forEach(({ turns }, rank) => {
    for (const turn of turns) {
      const from = Math.max(0, Math.floor((turn.start / total) * peaks.length));
      const to = Math.min(
        peaks.length,
        Math.ceil((turn.end / total) * peaks.length),
      );
      owned.fill(rank + 1, from, to);
    }
  });
  return owned;
}

/**
 * The audio drawn once, each bar in the colour of the voice that speaks there, with one thin
 * track per voice under it: where that voice speaks. A click places the reader; the keyboard
 * places it at the first turn of the voice. Without peaks nothing is drawn.
 */
export function VoiceWaveforms({
  speakers,
  segments,
  peaks,
  duration,
  playhead,
  onSeek,
  activeId,
}: {
  /** The voice being worked on: the other tracks are dimmed. */
  activeId?: number;
  speakers: LocalSpeaker[];
  segments: TranscriptSegment[];
  /** The peaks of the audio track the voices come from; absent before they were computed. */
  peaks?: number[];
  duration?: number | null;
  playhead: number;
  onSeek: (seconds: number) => void;
}) {
  const { t } = useI18n();
  const last = Math.max(0, ...segments.map((segment) => segment.end));
  const total =
    typeof duration === "number" && duration > 0
      ? Math.max(duration, last)
      : last;
  const drawn = !!peaks && peaks.length > 0 && total > 0;
  const tracks = useMemo(
    () =>
      speakers.map((speaker, rank) => ({
        speaker,
        turns: segments.filter(
          (segment) => segment.local_speaker_id === speaker.id,
        ),
        color: voiceColor(rank),
      })),
    [speakers, segments],
  );
  const bars = useMemo(() => {
    if (!peaks || peaks.length === 0 || total <= 0) return [];
    const owned = owners(peaks, tracks, total);
    return [
      { color: "var(--line-strong)", path: barsPath(peaks, owned, 0) },
      ...tracks.map(({ color }, rank) => ({
        color,
        path: barsPath(peaks, owned, rank + 1),
      })),
    ];
  }, [peaks, tracks, total]);
  const position = total > 0 ? Math.min(1, playhead / total) : 0;

  function seekIn(event: React.MouseEvent<HTMLElement>) {
    const box = event.currentTarget.getBoundingClientRect();
    const fraction = box.width > 0 ? (event.clientX - box.left) / box.width : 0;
    onSeek(Math.max(0, Math.min(1, fraction)) * total);
  }

  return (
    <div className="flex flex-col gap-1">
      {drawn && (
        <div
          aria-hidden="true"
          onClick={seekIn}
          className="relative h-16 w-full cursor-pointer"
        >
          <svg
            viewBox={`0 0 ${Math.max(1, peaks?.length ?? 1)} ${LEVELS}`}
            preserveAspectRatio="none"
            className="absolute inset-0 size-full"
          >
            {bars.map(
              ({ color, path }) =>
                path && (
                  <path
                    key={color}
                    d={path}
                    stroke={color}
                    strokeWidth={0.7}
                    strokeLinecap="round"
                    fill="none"
                  />
                ),
            )}
          </svg>
          <span
            className="absolute inset-y-0 w-px bg-ink"
            style={{ left: `${position * 100}%` }}
          />
        </div>
      )}
      <ul
        aria-label={t("clip.voices.tracksLabel")}
        className="m-0 flex list-none flex-col gap-0.5 p-0"
      >
        {tracks.map(({ speaker, turns, color }) => (
          <li
            key={speaker.id}
            data-active={activeId === speaker.id ? "true" : undefined}
          >
            <button
              type="button"
              aria-label={t("clip.voices.track", { label: speaker.label })}
              title={speaker.person?.display_name ?? speaker.label}
              disabled={total <= 0}
              onClick={(event) => {
                if (event.detail === 0) {
                  // Keyboard: the first turn of the voice.
                  const first = turns[0];
                  if (first) onSeek(first.start);
                  return;
                }
                seekIn(event);
              }}
              className={`block h-6 w-full min-w-0 cursor-pointer py-2 outline-none focus-visible:ring-2 focus-visible:ring-accent disabled:cursor-default ${
                activeId !== undefined && activeId !== speaker.id
                  ? "opacity-50"
                  : ""
              }`}
            >
              <span className="relative block h-2 w-full overflow-hidden rounded-full bg-side">
                {drawn && (
                  <svg
                    aria-hidden="true"
                    viewBox={`0 0 ${total} 1`}
                    preserveAspectRatio="none"
                    className="absolute inset-0 size-full"
                  >
                    <path
                      d={turns
                        .map(
                          (turn) =>
                            `M${turn.start} 0H${Math.min(turn.end, total)}V1H${turn.start}Z`,
                        )
                        .join("")}
                      fill={color}
                    />
                  </svg>
                )}
              </span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
