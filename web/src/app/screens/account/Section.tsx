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
      className={`flex max-w-2xl flex-col gap-3 rounded-xl border p-4 md:p-5 ${danger ? "border-bad-fg bg-bad-bg/40" : "border-line bg-surface"}`}
    >
      <h2
        id={id}
        className={`text-base font-semibold ${danger ? "text-bad-fg" : ""}`}
      >
        {title}
      </h2>
      {children}
    </section>
  );
}
