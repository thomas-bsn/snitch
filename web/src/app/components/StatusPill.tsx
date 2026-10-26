import { Ban, CircleCheck, CircleOff, type LucideIcon } from "lucide-react";

const tones = {
  ok: { icon: CircleCheck, className: "bg-ok-bg text-ok-fg" },
  idle: { icon: CircleOff, className: "bg-side text-mute" },
  bad: { icon: Ban, className: "bg-bad-bg text-bad-fg" },
} satisfies Record<string, { icon: LucideIcon; className: string }>;

export type PillTone = keyof typeof tones;

/** A state: always a label and a symbol, never the color alone. */
export function StatusPill({
  tone,
  children,
}: {
  tone: PillTone;
  children: string;
}) {
  const { icon: Icon, className } = tones[tone];
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${className}`}
    >
      <Icon className="size-3.5" />
      {children}
    </span>
  );
}
