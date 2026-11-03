import { useMemo } from "react";
import { useI18n } from "../i18n";
import type { LocalSpeaker, TranscriptSegment } from "./libraryApi";

/** The colour of a voice, by its rank in the clip: the same in the waveform and beside the voice. */
const VOICE_COLORS = [
  "#2563eb",
  "#c2410c",
  "#15803d",
  "#7e22ce",
  "#be185d",
  "#0e7490",
];
export const voiceColor = (rank: number) =>
  VOICE_COLORS[rank % VOICE_COLORS.length];

/** Full height of a drawing; a peak is a level from 0 to this value, centred on the axis. */
const LEVELS = 255;

/** The curve of the peaks where the voice speaks, one bar per peak (ADR 0034). */
function voicePath(peaks: number[], turns: TranscriptSegment[], total: number) {
  const owned = new Uint8Array(peaks.length);
  for (const turn of turns) {
    const from = Math.max(0, Math.floor((turn.start / total) * peaks.length));
    const to = Math.min(
      peaks.length,
      Math.ceil((turn.end / total) * peaks.length),
    );
    owned.fill(1, from, to);
  }
  let path = "";
  peaks.forEach((level, index) => {
    if (!owned[index]) return;
    const height = Math.max(1, Math.min(LEVELS, level));
    const top = (LEVELS - height) / 2;
    path += `M${index + 0.5} ${top}V${top + height}`;
  });
  return path;
}

/**
 * One track per voice of the clip: the audio drawn where that voice speaks, in the voice's
 * colour. A click places the reader in the track; the keyboard places it at the first turn.
 * Without peaks the tracks are shown with no curve.
 */
export function VoiceWaveforms({
  speakers,
  segments,
  peaks,
  duration,
  playhead,
  onSeek,
}: {
  speakers: LocalSpeaker[];
  segments: TranscriptSegment[];
  /** The peaks of the audio track the voices come from; absent before they were computed. */
  peaks?: number[];
  duration?: number | null;
  playhead: number;
  onSeek: (seconds: number) => void;
}) {
  const { t } = useI18n();
  const total =
    typeof duration === "number" && duration > 0
      ? duration
      : Math.max(0, ...segments.map((segment) => segment.end));
  const tracks = useMemo(
    () =>
      speakers.map((speaker, rank) => {
        const turns = segments.filter(
          (segment) => segment.local_speaker_id === speaker.id,
        );
        return {
          speaker,
          turns,
          color: voiceColor(rank),
          path:
            peaks && peaks.length > 0 && total > 0
              ? voicePath(peaks, turns, total)
              : "",
        };
      }),
    [speakers, segments, peaks, total],
  );
  const position = total > 0 ? Math.min(1, playhead / total) : 0;

  return (
    <ul
      aria-label={t("clip.voices.tracksLabel")}
      className="m-0 flex list-none flex-col gap-2 p-0"
    >
      {tracks.map(({ speaker, turns, color, path }) => (
        <li key={speaker.id} className="flex items-center gap-2">
          <span
            aria-hidden="true"
            className="size-2.5 shrink-0 rounded-full"
            style={{ backgroundColor: color }}
          />
          <button
            type="button"
            aria-label={t("clip.voices.track", { label: speaker.label })}
            disabled={total <= 0}
            onClick={(event) => {
              if (event.detail === 0) {
                // Keyboard: the first turn of the voice.
                const first = turns[0];
                if (first) onSeek(first.start);
                return;
              }
              const box = event.currentTarget.getBoundingClientRect();
              const fraction =
                box.width > 0 ? (event.clientX - box.left) / box.width : 0;
              onSeek(Math.max(0, Math.min(1, fraction)) * total);
            }}
            className="relative h-12 min-w-0 flex-1 cursor-pointer overflow-hidden rounded-md bg-side outline-none focus-visible:ring-2 focus-visible:ring-accent disabled:cursor-default"
          >
            {path && (
              <svg
                aria-hidden="true"
                viewBox={`0 0 ${Math.max(1, peaks?.length ?? 1)} ${LEVELS}`}
                preserveAspectRatio="none"
                className="absolute inset-0 size-full"
              >
                <path d={path} stroke={color} strokeWidth={0.8} fill="none" />
              </svg>
            )}
            <span
              aria-hidden="true"
              className="absolute inset-y-0 w-px bg-ink"
              style={{ left: `${position * 100}%` }}
            />
          </button>
          <span className="w-20 shrink-0 truncate text-xs text-mute">
            {speaker.label}
          </span>
        </li>
      ))}
    </ul>
  );
}
