import { useState, type ReactNode } from "react";

/** A folded block opened by a button (not a details element: the advanced settings already use one). */
export function Disclosure({
  id,
  label,
  children,
}: {
  id: string;
  label: string;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div>
      <button
        type="button"
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen(!open)}
        className="min-h-11 py-2 text-left underline-offset-2 hover:underline"
      >
        <span aria-hidden="true">{open ? "▾" : "▸"}</span> {label}
      </button>
      <div id={id} hidden={!open}>
        {children}
      </div>
    </div>
  );
}
