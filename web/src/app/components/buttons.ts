const base =
  "inline-flex h-11 items-center justify-center gap-2 rounded-lg border px-4 text-sm font-medium transition-colors md:h-9 disabled:opacity-60";

const variants = {
  primary:
    "border-accent-solid bg-accent-solid text-accent-fg hover:brightness-110",
  secondary: "border-line-strong bg-surface text-ink hover:bg-side",
  danger: "border-bad-fg bg-bad-bg text-bad-fg hover:brightness-95",
} as const;

/** Class names of a button or a link that looks like one. */
export function buttonClass(variant: keyof typeof variants = "secondary") {
  return `${base} ${variants[variant]}`;
}
