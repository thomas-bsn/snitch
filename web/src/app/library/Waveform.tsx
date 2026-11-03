import { useMemo } from "react";

/** Full height of the drawing; a peak is a level from 0 to this value, centred on the axis. */
const LEVELS = 255;

/**
 * The amplitude peaks of one audio track, one bar per peak (ADR 0034). Peaks carry no
 * voice information; the drawing only shows where the sound is.
 */
export function Waveform({ peaks, label }: { peaks: number[]; label: string }) {
  const path = useMemo(
    () =>
      peaks
        .map((level, index) => {
          const height = Math.max(1, Math.min(LEVELS, level));
          const top = (LEVELS - height) / 2;
          return `M${index + 0.5} ${top}V${top + height}`;
        })
        .join(""),
    [peaks],
  );
  return (
    <svg
      role="img"
      aria-label={label}
      viewBox={`0 0 ${Math.max(1, peaks.length)} ${LEVELS}`}
      preserveAspectRatio="none"
      className="h-16 w-full text-accent"
    >
      <path d={path} stroke="currentColor" strokeWidth={0.8} fill="none" />
    </svg>
  );
}
