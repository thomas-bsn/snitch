// PROTOTYPE — /prototype/fiche-video?variant=A|B|C. Hors session, sans serveur. Jamais fusionné dans main.
import { ChevronLeft, ChevronRight, Library, Upload, Users } from "lucide-react";
import { useEffect } from "react";
import { useSearchParams } from "react-router";
import { Brand } from "../../components/Logo";
import { useFicheSim } from "./useFicheSim";
import { VariantA, VariantB, VariantC } from "./variants";

const VARIANTS = [
  { key: "A", name: "Analyse en colonne", C: VariantA },
  { key: "B", name: "Bandeau de workflow", C: VariantB },
  { key: "C", name: "Analyse en panneau latéral", C: VariantC },
];

export function FichePrototype() {
  const [params, setParams] = useSearchParams();
  const sim = useFicheSim();
  const idx = Math.max(0, VARIANTS.findIndex((v) => v.key === (params.get("variant") ?? "A")));
  const go = (d: number) => setParams({ variant: VARIANTS[(idx + d + VARIANTS.length) % VARIANTS.length].key }, { replace: true });

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement;
      if (el.closest("input, textarea, [contenteditable]")) return;
      if (e.key === "ArrowLeft") go(-1);
      if (e.key === "ArrowRight") go(1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const V = VARIANTS[idx].C;
  return (
    <div className="flex min-h-dvh bg-canvas text-ink">
      <aside className="hidden w-[220px] shrink-0 flex-col gap-1 border-r border-line bg-side px-3 py-5 md:flex">
        <div className="px-2 pb-5"><Brand /></div>
        {[[Library, "Bibliothèque", true], [Users, "Personnes", false], [Upload, "Importer", false]].map(([Icon, label, cur]) => {
          const I = Icon as typeof Library;
          return (
            <span key={label as string} className={`flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium ${cur ? "bg-accent-soft text-accent-ink" : "text-nav"}`}>
              <I className="size-4" /> {label as string}
            </span>
          );
        })}
      </aside>
      <main className="min-w-0 flex-1 pb-28">
        <div className="border-b border-line px-4 py-3 text-sm text-mute md:px-7">‹ Bibliothèque / 2026 / novembre</div>
        <V sim={sim} />
      </main>

      {/* Barre du prototype : variantes + simulation. Ne fait pas partie du design évalué. */}
      <div className="fixed bottom-4 left-1/2 z-30 flex -translate-x-1/2 items-center whitespace-nowrap gap-3 rounded-full bg-black px-4 py-2 text-xs text-white shadow-2xl">
        <button type="button" onClick={() => go(-1)} aria-label="Variante précédente"><ChevronLeft className="size-4" /></button>
        <span className="font-semibold">{VARIANTS[idx].key} · {VARIANTS[idx].name}</span>
        <button type="button" onClick={() => go(1)} aria-label="Variante suivante"><ChevronRight className="size-4" /></button>
        <span className="h-4 w-px bg-white/30" />
        <label className="flex items-center gap-1">
          Erreur simulée
          <select value={sim.failNext} onChange={(e) => sim.setFailNext(e.target.value as typeof sim.failNext)} className="rounded bg-white/15 px-1 py-0.5">
            <option value="none">aucune</option>
            <option value="transcription">transcription (Whisper 503)</option>
            <option value="fingerprints">empreintes</option>
          </select>
        </label>
        <label className="flex items-center gap-1">
          <input type="checkbox" checked={sim.voiceWorkerMissing} onChange={(e) => sim.setVoiceWorkerMissing(e.target.checked)} /> pas de worker voix
        </label>
        <button type="button" onClick={sim.reset} className="rounded bg-white/15 px-2 py-0.5">Réinitialiser</button>
      </div>
    </div>
  );
}
