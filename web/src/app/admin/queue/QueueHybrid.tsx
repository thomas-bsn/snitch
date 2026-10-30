import { Check, Film, Cpu, Clock3, TriangleAlert, LoaderCircle } from "lucide-react";

const steps = [
  ["Paroles en texte", "Snitch écoute le clip et écrit ce qui est dit."],
  ["Recherche par les paroles", "Le texte est préparé pour retrouver ce clip avec une phrase ou une idée."],
  ["Voix distinctes", "Snitch repère les différentes voix présentes dans le clip."],
  ["Personnes reconnues", "Les voix sont comparées aux personnes connues du site."],
];
const clips = [
  { id: 18, time: "21:04:47", duration: "0:25", phase: 0, lane: "En cours" },
  { id: 17, time: "21:02:13", duration: "1:12", phase: 2, lane: "En cours" },
  { id: 16, time: "20:58:02", duration: "0:42", phase: 0, lane: "En cours" },
  { id: 15, time: "20:41:36", duration: "0:38", phase: 1, lane: "En cours" },
  { id: 14, time: "20:32:10", duration: "0:30", phase: 0, lane: "En attente" },
  { id: 13, time: "20:10:05", duration: "0:56", phase: 0, lane: "En attente" },
  { id: 12, time: "19:54:22", duration: "0:24", phase: 2, lane: "À résoudre" },
];

export function QueueHybrid() {
  return <div className="flex flex-col gap-6 pb-24">
    <header><p className="text-sm text-mute">Administration / Traitement des clips</p><h2 className="mt-2 text-3xl font-semibold">Où en sont mes vidéos ?</h2><p className="mt-2 max-w-3xl text-mute">Chaque carte représente un clip. Suivez ce que Snitch est en train d’en extraire pour retrouver ses paroles et les personnes qui parlent.</p></header>
    <section className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-line bg-surface p-5"><div className="flex items-center gap-3"><Cpu aria-hidden="true" className="text-accent-ink"/><div><h3 className="font-semibold">THOMAS-GAMER traite 4 clips</h3><p className="text-sm text-mute">RTX 4070 Super · 12 Go · scénario de démonstration</p></div></div><p className="rounded-lg bg-accent-soft px-4 py-2 text-sm text-accent-ink">4 en cours · 2 en attente · 1 à résoudre</p></section>
    <details className="rounded-xl border border-line bg-surface p-5" open><summary className="cursor-pointer font-semibold">Que fait Snitch avec un clip ?</summary><p className="mt-2 text-sm text-mute">Après l’import, ces traitements rendent le clip exploitable. Les traitements des voix et de la recherche peuvent avancer séparément.</p><div className="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">{steps.map(([name, description], i) => <div key={name} className="rounded-lg bg-side p-4"><span className="text-xs font-semibold text-accent-ink">{i < 2 ? "PAROLES" : "VOIX"}</span><h4 className="mt-2 font-semibold">{name}</h4><p className="mt-1 text-sm text-mute">{description}</p></div>)}</div></details>
    <div className="grid items-start gap-5 xl:grid-cols-[2fr_1fr_1fr]">{["En cours", "En attente", "À résoudre"].map(lane => <section key={lane} className="min-w-0 rounded-xl bg-side p-3"><h3 className="mb-3 flex items-center justify-between px-1 font-semibold"><span>{lane}</span><span className="rounded-full bg-surface px-2.5 py-1 text-sm">{clips.filter(c => c.lane === lane).length}</span></h3><div className={`grid gap-3 ${lane === "En cours" ? "md:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2" : ""}`}>{clips.filter(c => c.lane === lane).map(clip => <article key={clip.id} className={`relative min-w-0 rounded-xl border bg-surface p-4 ${lane === "À résoudre" ? "border-warn-fg" : "border-line"}`}>
      {lane === "À résoudre" && <TriangleAlert className="absolute right-3 top-3 text-warn-fg" size={20} aria-label="Attention : traitement bloqué" role="img"/>}
      <div className={`flex items-center gap-3 ${lane === "À résoudre" ? "pr-7" : ""}`}><div className="flex size-12 shrink-0 items-center justify-center rounded-lg bg-accent-soft text-accent-ink"><Film aria-hidden="true"/></div><div className="min-w-0"><h4 className="font-semibold">Call of Duty Black Ops 3</h4><p className="text-sm text-mute">9 avril 2025 · {clip.time}</p></div></div>
      <p className="mt-3 text-xs text-mute">Clip #{clip.id} · {clip.duration} · Importé par Thomas</p>
      <div className={`mt-4 rounded-lg p-3 text-sm ${lane === "À résoudre" ? "bg-warn-bg text-warn-fg" : "bg-accent-soft text-accent-ink"}`}><div className="flex items-center gap-2 font-semibold">{lane === "En cours" ? <LoaderCircle size={16} aria-hidden="true"/> : lane === "En attente" ? <Clock3 size={16} aria-hidden="true"/> : <TriangleAlert size={16} aria-hidden="true"/>}{lane === "En cours" ? ["Écrit les paroles du clip", "Prépare la recherche", "Distingue les voix", "Recherche les personnes"][clip.phase] : lane === "En attente" ? "Attend une place sur le PC" : "Le service des voix ne répond pas"}</div><p className="mt-1">{lane === "En cours" ? "Traitement sur THOMAS-GAMER. Durée restante inconnue." : lane === "En attente" ? "Sera pris en charge quand un traitement se termine." : "Les paroles sont disponibles. Le traitement des voix reste bloqué."}</p></div>
      <ul className="mt-4 space-y-2 text-sm">{steps.map(([name], i) => { const done = lane !== "En attente" && i < clip.phase; const current = i === clip.phase && lane !== "En attente"; return <li key={name} className="flex items-start justify-between gap-2"><span>{name}</span><span className={`flex shrink-0 items-center gap-1 text-xs ${done ? "text-ok-fg" : current ? "text-accent-ink" : "text-mute"}`}>{done && <Check size={13} aria-hidden="true"/>}{done ? "Prêt" : current ? lane === "À résoudre" ? "Bloqué" : "En cours" : "À faire"}</span></li>; })}</ul>
      <details className="mt-4 border-t border-line pt-3 text-xs text-mute"><summary className="cursor-pointer py-2">Nom complet du clip</summary><p className="mt-2 break-all">Call of Duty Black Ops 3 2025.04.09 - {clip.time.replaceAll(":", ".")}.04.DVR.mp4</p></details>
    </article>)}</div></section>)}</div>
  </div>;
}
