/** The Snitch mark: an indigo rounded square with four white bars (a waveform). */
export function Logo({ className = "size-6" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <rect width="32" height="32" rx="8" fill="#5b5bd6" />
      <rect x="6" y="13" width="3.5" height="6" rx="1.75" fill="#fff" />
      <rect x="11.5" y="8" width="3.5" height="16" rx="1.75" fill="#fff" />
      <rect x="17" y="11" width="3.5" height="10" rx="1.75" fill="#fff" />
      <rect x="22.5" y="15" width="3.5" height="3" rx="1.5" fill="#fff" />
    </svg>
  );
}

export function Brand({ className = "" }: { className?: string }) {
  return (
    <span
      className={`flex items-center gap-2 text-base font-semibold tracking-tight ${className}`}
    >
      <Logo />
      Snitch
    </span>
  );
}
