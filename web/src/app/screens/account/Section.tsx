import { useId, type ReactNode } from "react";

/** A titled block of a settings screen; the title names the region. */
export function Section({
  title,
  children,
  danger = false,
}: {
  title: string;
  danger?: boolean;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <section
      aria-labelledby={id}
      className={`flex min-w-0 flex-col gap-5 rounded-2xl border p-5 sm:p-6 ${danger ? "border-line bg-surface" : "border-line bg-surface"}`}
    >
      <h2
        id={id}
        className={`text-sm font-semibold ${danger ? "text-bad-fg" : ""}`}
      >
        {title}
      </h2>
      {children}
    </section>
  );
}
