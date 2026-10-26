// PROTOTYPE — throwaway variant switcher (arrow keys or buttons, shareable via ?variant=).
import { useEffect } from "react";

export const VARIANTS = [
  { key: "current", name: "UI actuelle" },
  { key: "atelier", name: "Atelier" },
];

export function currentVariant(): string {
  return new URLSearchParams(location.search).get("variant") ?? "current";
}

function go(delta: number) {
  const i = VARIANTS.findIndex((v) => v.key === currentVariant());
  const next = VARIANTS[(i + delta + VARIANTS.length) % VARIANTS.length];
  const q = new URLSearchParams(location.search);
  q.set("variant", next.key);
  location.search = q.toString();
}

export function PrototypeSwitcher() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement;
      if (t.closest("input,textarea,[contenteditable]")) return;
      if (e.key === "ArrowLeft") go(-1);
      if (e.key === "ArrowRight") go(1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  const v = VARIANTS.find((x) => x.key === currentVariant()) ?? VARIANTS[0];
  return (
    <div className="proto-bar">
      <button aria-label="Variante précédente" onClick={() => go(-1)}>
        ←
      </button>
      <span>
        {v.key} ({v.name})
      </span>
      <button aria-label="Variante suivante" onClick={() => go(1)}>
        →
      </button>
    </div>
  );
}
