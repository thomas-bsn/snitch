import type { ReactNode } from "react";

/** An error: announced at once by screen readers, with a label besides the color. */
export function ErrorAlert({ children }: { children: ReactNode }) {
  return (
    <p role="alert" className="rounded-lg bg-bad-bg px-3 py-2.5 text-bad-fg">
      {children}
    </p>
  );
}

/** A neutral message (confirmation, information). */
export function StatusNotice({ children }: { children: ReactNode }) {
  return (
    <p
      role="status"
      className="rounded-lg bg-accent-soft px-3 py-2.5 text-accent-ink"
    >
      {children}
    </p>
  );
}
