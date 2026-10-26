import { useId, type ReactNode } from "react";

/** A titled block of a settings screen; the title names the region. */
export function Section({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <section
      aria-labelledby={id}
      className="flex max-w-2xl flex-col gap-3 rounded-xl border border-line bg-surface p-4 md:p-5"
    >
      <h2 id={id} className="text-base font-semibold">
        {title}
      </h2>
      {children}
    </section>
  );
}
