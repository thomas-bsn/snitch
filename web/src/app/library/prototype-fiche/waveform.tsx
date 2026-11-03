// PROTOTYPE — forme d'onde colorée par voix (données factices déterministes).
import type { Segment, Voice } from "./data";

/** Une couleur par voix, lisible en clair comme en sombre. */
export const VOICE_COLORS = ["#5b5bd6", "#e5484d", "#12a594", "#f76b15", "#8e4ec6", "#0090ff", "#ad7f58"];
export const voiceColor = (id: number) => VOICE_COLORS[(id - 1) % VOICE_COLORS.length];

function seeded(seed: number) {
  let s = seed;
  return () => ((s = (s * 9301 + 49297) % 233280) / 233280);
}

/** Amplitudes factices : de la parole là où il y a un segment, du bruit ailleurs. */
function peaks(segments: Segment[], start: number, span: number, bars: number) {
  const r = seeded(7 + Math.round(start));
  return Array.from({ length: bars }, (_, i) => {
    const t = start + (i / bars) * span;
    const seg = segments.find((s) => t >= s.start && t < s.end);
    if (!seg) return { h: 0.06 + r() * 0.06, seg: undefined };
    const phrase = Math.sin(((t - seg.start) / (seg.end - seg.start)) * Math.PI);
    return { h: 0.2 + phrase * 0.55 * (0.5 + r() * 0.5) + r() * 0.15, seg };
  });
}

type Props = {
  segments: Segment[];
  voices: Voice[];
  playhead?: number;
  onSeek?: (s: number) => void;
  /** Ne colore que cette voix, les autres en gris. */
  focus?: number;
  height?: number;
  bars?: number;
  /** Fenêtre affichée ; par défaut toute la durée des segments. */
  from?: number;
  to?: number;
  lanes?: boolean;
};

export function Waveform({ segments, voices, playhead, onSeek, focus, height = 72, bars = 180, from, to, lanes = false }: Props) {
  const start = from ?? 0;
  const end = to ?? Math.max(1, ...segments.map((s) => s.end));
  const span = end - start;
  const inWindow = segments.filter((s) => s.end > start && s.start < end);
  const shown = peaks(segments, start, span, bars);
  const voiceOf = (id: number) => voices.find((v) => v.id === id);
  const colorFor = (seg?: Segment) => {
    if (!seg) return "var(--line-strong)";
    const v = voiceOf(seg.voiceId);
    if (!v || v.state === "ignored") return "var(--line-strong)";
    if (focus !== undefined && seg.voiceId !== focus) return "var(--line-strong)";
    return voiceColor(seg.voiceId);
  };
  const click = (e: React.MouseEvent<SVGSVGElement>) => {
    if (!onSeek) return;
    const box = e.currentTarget.getBoundingClientRect();
    onSeek(start + ((e.clientX - box.left) / box.width) * span);
  };
  const laneVoices = voices.filter((v) => inWindow.some((s) => s.voiceId === v.id));
  return (
    <div className="flex flex-col gap-1">
      <svg viewBox={`0 0 ${bars} 100`} preserveAspectRatio="none" style={{ height }} className={`w-full ${onSeek ? "cursor-pointer" : ""}`} onClick={click} role="img" aria-label="Forme d'onde, colorée par voix">
        {shown.map((b, i) => (
          <rect key={i} x={i + 0.15} width={0.7} y={50 - b.h * 48} height={b.h * 96} rx={0.35} fill={colorFor(b.seg)} opacity={b.seg && focus !== undefined && b.seg.voiceId === focus ? 1 : 0.9} />
        ))}
        {playhead !== undefined && playhead >= start && playhead <= end && (
          <line x1={((playhead - start) / span) * bars} x2={((playhead - start) / span) * bars} y1={0} y2={100} stroke="var(--ink)" strokeWidth={0.4} vectorEffect="non-scaling-stroke" />
        )}
      </svg>
      {lanes && (
        <div className="flex flex-col gap-0.5">
          {laneVoices.map((v) => (
            <div key={v.id} className="relative h-2 rounded-full bg-side" title={v.person ?? v.label}>
              {inWindow.filter((s) => s.voiceId === v.id).map((s) => (
                <span
                  key={s.id}
                  className="absolute top-0 h-2 rounded-full"
                  style={{
                    left: `${((Math.max(s.start, start) - start) / span) * 100}%`,
                    width: `${((Math.min(s.end, end) - Math.max(s.start, start)) / span) * 100}%`,
                    background: v.state === "ignored" ? "var(--line-strong)" : voiceColor(v.id),
                  }}
                />
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

/** Pastille de couleur de la voix, pour relier la liste à la forme d'onde. */
export function VoiceDot({ id, ignored = false }: { id: number; ignored?: boolean }) {
  return <span className="inline-block size-2.5 shrink-0 rounded-full" style={{ background: ignored ? "var(--line-strong)" : voiceColor(id) }} />;
}
