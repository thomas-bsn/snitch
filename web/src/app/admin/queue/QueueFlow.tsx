import { Dialog } from "@base-ui/react/dialog";
import { useState } from "react";
import { ArrowRight, ChevronDown, Inbox, AudioLines, Search, Users, ScanFace, CheckCheck, Monitor, Play, TriangleAlert, RotateCcw } from "lucide-react";
const stageIcons = [Inbox, AudioLines, Search, Users, ScanFace, CheckCheck];
const stages = [
  { name: "À traiter", help: "Attend une place sur le PC." },
  { name: "Paroles en texte", help: "Écrire ce qui est dit dans la vidéo." },
  { name: "Recherche prête", help: "Retrouver le clip à partir de ses paroles." },
  { name: "Voix distinctes", help: "Distinguer les personnes qui parlent." },
  { name: "Personnes reconnues", help: "Comparer les voix aux personnes connues." },
  { name: "Terminé", help: "Les résultats sont disponibles." },
];
const initial = [
  ...Array.from({ length: 127 }, (_, i) => ({ id: 1000-i, time: `${String(19-Math.floor(i/60)).padStart(2,"0")}:${String(59-i%60).padStart(2,"0")}:00`, duration: "0:32", step: 5, completed: 100-i })),
  { id: 18, time: "21:04:47", duration: "0:25", step: 1 },
  { id: 17, time: "21:02:13", duration: "1:12", step: 3 },
  { id: 16, time: "20:58:02", duration: "0:42", step: 1 },
  { id: 15, time: "20:41:36", duration: "0:38", step: 2 },
  { id: 14, time: "20:32:10", duration: "0:30", step: 0 },
  { id: 13, time: "20:10:05", duration: "0:56", step: 0 },
  { id: 12, time: "19:54:22", duration: "0:24", step: 3, blocked: true },
].map(c => ({ ...c, blocked: "blocked" in c ? Boolean(c.blocked) : false, completed: "completed" in c ? c.completed : 0, title: `Call of Duty Black Ops 3 2025.04.09 - ${c.time.replaceAll(":", ".")}.04.DVR.mp4`, game: "Call of Duty Black Ops 3" }));
function ClipArtwork({ id }: { id: number }) {
  return <svg viewBox="0 0 80 64" className="h-12 w-14 shrink-0 rounded-md" aria-hidden="true">
    <rect width="80" height="64" fill={id % 2 ? "#172a38" : "#24233c"}/>
    <path d="M0 48 20 28 32 38 49 14 80 42V64H0Z" fill={id % 2 ? "#345769" : "#514b77"}/>
    <path d="M0 57 27 41 45 53 63 32 80 46V64H0Z" fill="#101827"/>
    <path d="M39 22v8m0 10v8M26 35h8m10 0h8" stroke="#e9e8f4" strokeWidth="1.5" opacity=".8"/>
    <rect x="5" y="5" width="15" height="3" rx="1.5" fill="#c4b5fd"/>
  </svg>;
}
export function QueueFlow() {
  const [clips, setClips] = useState(initial);
  const [history, setHistory] = useState(false);
  const [historyPage, setHistoryPage] = useState(1);
  const completed = clips.filter(c => c.step === 5).sort((a,b) => b.completed-a.completed);
  const pageCount = Math.max(1, Math.ceil(completed.length / 10));
  function openHistory() { setHistoryPage(1); setHistory(true); }
  const [tick, setTick] = useState(0);
  const running = clips.filter(c => c.step > 0 && c.step < 5 && !c.blocked).length;
  function advance() {
    setClips(old => { let slots = 4 - old.filter(c => c.step > 0 && c.step < 4 && !c.blocked).length; return old.map(c => c.blocked || c.step === 5 ? c : c.step === 0 ? slots-- > 0 ? { ...c, step: 1 } : c : { ...c, step: c.step + 1, completed: c.step === 4 ? 101 + tick : c.completed }); });
    setTick(t => t + 1);
  }
  return <div className="space-y-6 pb-24">
    <header><p className="text-sm text-mute">Administration / Traitement des clips</p><h2 className="mt-2 text-3xl font-semibold">Le parcours de vos clips</h2><p className="mt-2 text-mute">Repérez une vidéo dans le schéma, puis dépliez sa carte pour voir ses détails.</p></header>
    <div className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-line bg-surface p-4"><div className="flex items-center gap-4"><div className="grid size-12 place-items-center rounded-xl bg-accent-soft text-accent-ink"><Monitor aria-hidden="true"/></div><div><p className="font-semibold">THOMAS-GAMER · RTX 4070 Super</p><p className="text-sm text-mute" role="status">{running} clips en cours · {clips.filter(c => c.step === 0).length} en attente · 1 à résoudre</p></div></div><div className="flex flex-wrap gap-2"><button onClick={openHistory} className="min-h-11 rounded-lg border border-line-strong px-4 text-sm font-semibold">Historique ({completed.length})</button><button onClick={advance} className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-accent-solid px-4 font-semibold text-accent-fg">Simuler la suite <ArrowRight size={16} aria-hidden="true"/></button><button onClick={() => {setClips(initial); setTick(0);}} className="inline-flex min-h-11 items-center gap-2 rounded-lg border border-line-strong px-3 text-sm"><RotateCcw size={16} aria-hidden="true"/>Recommencer</button></div></div>
    <p className="max-w-4xl text-sm text-mute">Simulation visuelle : ici, les clips suivent un parcours simplifié de gauche à droite. Dans Snitch, la recherche et les traitements des voix peuvent avancer séparément.</p>
    <section aria-label="Parcours des clips" className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-5">{stages.map((stage, index) => index === 0 ? null : <div key={stage.name} className="relative min-w-0 rounded-2xl border border-line bg-side p-3"><div className="mb-4"><div className="flex items-center justify-between gap-2"><span className={`relative grid size-14 place-items-center rounded-2xl border border-line shadow-sm ${index === 5 ? "bg-ok-bg text-ok-fg" : "bg-surface text-accent-ink"}`}>{(() => { const Icon = stageIcons[index]; return <Icon size={26} strokeWidth={1.5} aria-hidden="true"/>; })()}<span className="absolute -bottom-1 -right-1 grid size-5 place-items-center rounded-full border border-line bg-surface text-xs text-mute">{index}</span></span>{index < 5 && <ArrowRight size={18} className="text-accent-ink" aria-hidden="true"/>}</div><h3 className="mt-3 font-semibold">{stage.name} <span className="text-mute">· {clips.filter(c => c.step === index || (index === 1 && c.step === 0)).length}</span></h3><p className="mt-1 min-h-10 text-xs text-mute">{stage.help}</p></div><div className="space-y-3">{clips.filter(c => c.step === index || (index === 1 && c.step === 0)).sort((a,b) => index === 5 ? b.completed - a.completed : 0).slice(0, index === 5 ? 3 : clips.length).map(clip => <details key={clip.id} className={`group rounded-xl border bg-surface shadow-sm transition-shadow hover:shadow-md motion-reduce:transition-none ${(clip.blocked || clip.step === 0) ? "border-warn-fg" : "border-line"}`}><summary className="relative flex min-h-20 cursor-pointer list-none items-start gap-2 p-3 focus-visible:outline-2 focus-visible:outline-accent"><ClipArtwork id={clip.id}/><div className="min-w-0 flex-1 pr-4"><p className="break-words text-sm font-semibold">{clip.title}</p>{clip.game && <p className="mt-1 text-xs text-accent-ink">{clip.game}</p>}<p className="text-xs text-mute">{clip.time} · #{clip.id}</p>{clip.step === 0 && <p className="mt-2 text-xs text-warn-fg">En attente de traitement</p>}{clip.blocked && <p className="mt-2 text-xs text-warn-fg">Service des voix indisponible</p>}</div>{(clip.blocked || clip.step === 0) ? <TriangleAlert size={16} className="absolute right-2 top-2 text-warn-fg" aria-label={clip.blocked ? "Traitement bloqué" : "En attente de traitement"}/> : null}<ChevronDown size={14} className="absolute bottom-3 right-2 text-mute transition-transform group-open:rotate-180 motion-reduce:transition-none" aria-hidden="true"/></summary><div className="space-y-3 border-t border-line p-3 text-xs"><p>9 avril 2025 · {clip.duration}<br/>Importé par Thomas</p><p className="break-all text-mute">Call of Duty Black Ops 3 2025.04.09 - {clip.time.replaceAll(":", ".")}.04.DVR.mp4</p><p className={clip.blocked ? "rounded-md bg-warn-bg p-2 text-warn-fg" : "text-mute"}>{clip.blocked ? "Les paroles sont disponibles. Vérifiez le service des voix pour reprendre ce traitement." : index === 5 ? "Tous les traitements de cette simulation sont terminés." : clip.step === 0 ? "Attend qu’une place se libère sur THOMAS-GAMER." : `THOMAS-GAMER : ${stage.help}`}</p></div></details>)}{!clips.some(c => c.step === index || (index === 1 && c.step === 0)) && <div className="rounded-lg border border-dashed border-line-strong p-4 text-center text-xs text-mute"><div className="mb-2 flex justify-center"><Play size={20} strokeWidth={1} aria-hidden="true"/></div>Les clips arriveront ici</div>}</div>{index === 5 && <button onClick={openHistory} aria-haspopup="dialog" className="mt-4 min-h-11 w-full rounded-lg border border-line-strong bg-surface px-3 text-sm font-semibold">{history ? "Masquer" : "Voir"} l’historique ({clips.filter(c => c.step === 5).length})</button>}</div>)}</section>
    <Dialog.Root open={history} onOpenChange={setHistory}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-[60] bg-black/50"/>
        <Dialog.Viewport className="fixed inset-0 z-[70] flex items-center justify-center p-3 sm:p-6">
          <Dialog.Popup className="flex max-h-[90dvh] w-full max-w-5xl flex-col overflow-hidden rounded-2xl border border-line bg-surface text-ink shadow-xl outline-none">
            <header className="flex items-start justify-between gap-4 border-b border-line p-5"><div><Dialog.Title className="text-2xl font-semibold">Historique des clips</Dialog.Title><Dialog.Description className="mt-1 text-sm text-mute">{completed.length} clips terminés · du plus récent au plus ancien</Dialog.Description></div><Dialog.Close className="min-h-11 rounded-lg border border-line-strong px-4 text-sm">Fermer</Dialog.Close></header>
            <div className="min-h-0 overflow-y-auto px-5"><ul className="divide-y divide-line">{completed.slice((historyPage-1)*10, historyPage*10).map(c => <li key={c.id} className="flex items-center gap-3 py-3"><ClipArtwork id={c.id}/><div className="min-w-0 flex-1"><p className="break-words text-sm font-semibold">{c.title}</p><p className="mt-1 text-xs text-mute">{c.game} · Clip #{c.id} · {c.duration}</p></div><span className="rounded-full bg-ok-bg px-3 py-1 text-xs text-ok-fg">Terminé</span></li>)}</ul></div>
            <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-line p-5"><p className="text-sm text-mute" role="status">{(historyPage-1)*10+1}–{Math.min(historyPage*10, completed.length)} sur {completed.length} · Page {historyPage} / {pageCount}</p><nav aria-label="Pages de l’historique" className="flex flex-wrap gap-2">{[{label: "Première", page: 1, disabled: historyPage === 1}, {label: "Précédente", page: historyPage-1, disabled: historyPage === 1}, {label: "Suivante", page: historyPage+1, disabled: historyPage === pageCount}, {label: "Dernière", page: pageCount, disabled: historyPage === pageCount}].map(b => <button key={b.label} disabled={b.disabled} onClick={() => setHistoryPage(b.page)} className="min-h-11 rounded-lg border border-line-strong px-3 text-sm disabled:cursor-not-allowed disabled:opacity-40">{b.label}</button>)}</nav></footer>
          </Dialog.Popup>
        </Dialog.Viewport>
      </Dialog.Portal>
    </Dialog.Root>
    <p className="text-xs text-mute">Données fictives · {tick === 0 ? "État initial" : `Simulation avancée ${tick} fois`} · Aucun traitement réel n’est lancé.</p>
  </div>;
}
