import { groupClipWork } from "./workflowModel";
import { Dialog } from "@base-ui/react/dialog";
import { useEffect, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { AudioLines, Search, Users, ScanFace, CheckCheck, TriangleAlert, Film } from "lucide-react";
import { api } from "../../api/client";
import { useI18n } from "../../i18n";
import type { components } from "../../../api.gen";
type Job = components["schemas"]["Job"];
const stages = [
  { key: "transcription", label: "Paroles en texte", icon: AudioLines },
  { key: "index", label: "Recherche", icon: Search },
  { key: "diarization", label: "Voix distinctes", icon: Users },
  { key: "voice", label: "Reconnaissance des voix", icon: ScanFace },
  { key: "done", label: "Traitements terminés", icon: CheckCheck },
  { key: "other", label: "Autres traitements", icon: Film },
];
function stage(job: Job) {
  if (job.status === "done") return "done";
  if (job.step.includes("transcription")) return "transcription";
  if (job.step.includes("index") || job.step.includes("embedding")) return "index";
  if (job.step.includes("diar")) return "diarization";
  if (job.step.includes("voice") || job.step.includes("fingerprint")) return "voice";
  return "other";
}
export function JobWorkflow({jobs, actions}: {jobs: Job[]; actions: (job: Job) => ReactNode}) {
  const { t } = useI18n();
  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyPage, setHistoryPage] = useState(1);
  const groups = groupClipWork(jobs);
  const completedClips = groups.filter(g => g.finished);
  const completed = { items: completedClips.slice((historyPage-1)*10, historyPage*10).map(g => g.current), total: completedClips.length };
  const historyError = false;
  const loadingHistory = false;
  const displayedJobs = groups.filter(g => !g.finished).map(g => g.current).concat(completedClips.slice(0,3).map(g => g.current));
  const [titles, setTitles] = useState<Record<number,string>>({});
  const ids = [...new Set([...jobs, ...(completed?.items ?? [])].map(j => j.clip_id))].sort((a,b)=>a-b).join(",");
  useEffect(() => {
    let disposed = false;
    const clipIds = ids ? ids.split(",").map(Number) : [];
    void Promise.all(clipIds.map(async id => {
      try { const result = await api.GET("/api/clips/{id}", { params: { path: { id } } }); return [id, result.data?.original_name ?? `Clip #${id}`] as const; }
      catch { return [id, `Clip #${id}`] as const; }
    })).then(entries => { if (!disposed) setTitles(Object.fromEntries(entries)); });
    return () => { disposed = true; };
  }, [ids]);
  return <><div className="mb-4 flex justify-end"><button onClick={() => {setHistoryPage(1); setHistoryOpen(true);}} className="min-h-11 rounded-lg border border-line-strong px-4 font-semibold">Historique des clips</button></div><div className="grid items-start gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-5">{stages.filter(s => s.key !== "other" || displayedJobs.some(j => stage(j) === "other")).map(s => <section key={s.key} className="min-w-0 rounded-2xl border border-line bg-side p-3"><s.icon className="mb-3 text-accent-ink" size={28} aria-hidden="true"/><h3 className="mb-4 font-semibold">{s.label} · {displayedJobs.filter(j=>stage(j)===s.key).length}</h3><div className="space-y-3">{displayedJobs.filter(j=>stage(j)===s.key).slice(0, s.key === "done" ? 3 : jobs.length).map(job => <details key={job.id} className={`rounded-xl border bg-surface ${["pending","retry_wait","dead","failed"].includes(job.status) ? "border-warn-fg" : "border-line"}`}><summary className="relative min-h-16 cursor-pointer p-3 pr-8"><span className="block break-words text-sm font-semibold">{titles[job.clip_id] ?? `Clip #${job.clip_id}`}</span><span className="mt-2 block text-xs text-mute">#{job.clip_id} · {t(`queue.status.${job.status}`)}</span>{["pending","retry_wait","dead","failed"].includes(job.status) && <TriangleAlert size={17} className="absolute right-2 top-2 text-warn-fg" aria-label={t(`queue.status.${job.status}`)}/>}</summary><div className="space-y-3 border-t border-line p-3 text-sm"><Link to={`/clips/${job.clip_id}`} className="text-accent-ink underline">Ouvrir le clip</Link><p className="break-words text-mute">{job.step}</p>{job.last_error && <p className="rounded-lg bg-warn-bg p-2 text-warn-fg">{job.last_error}</p>}{groups.find(g => g.clipId === job.clip_id)?.jobs.map(item => <div key={item.id} className="space-y-2 border-t border-line pt-2"><p>{item.step} · {t(`queue.status.${item.status}`)}</p>{actions(item)}</div>)}</div></details>)}</div></section>)}</div>
    <Dialog.Root open={historyOpen} onOpenChange={setHistoryOpen}><Dialog.Portal><Dialog.Backdrop className="fixed inset-0 z-[60] bg-black/50"/><Dialog.Viewport className="fixed inset-0 z-[70] flex items-center justify-center p-4"><Dialog.Popup className="flex max-h-[90dvh] w-full max-w-5xl flex-col overflow-hidden rounded-2xl border border-line bg-surface text-ink shadow-xl outline-none">
      <header className="flex items-center justify-between gap-4 border-b border-line p-5"><div><Dialog.Title className="text-xl font-semibold">Historique des clips terminés</Dialog.Title><Dialog.Description className="text-sm text-mute">Clips dont tous les traitements planifiés ont réussi, dix par page.</Dialog.Description></div><Dialog.Close className="min-h-11 rounded-lg border border-line-strong px-4">Fermer</Dialog.Close></header>
      <div className="min-h-0 overflow-y-auto p-5">{loadingHistory ? <p role="status">{t("app.loading")}</p> : historyError ? <p role="alert">Impossible de charger l’historique.</p> : <ul className="divide-y divide-line">{completed?.items.map(job => <li key={job.id} className="py-3"><Link to={`/clips/${job.clip_id}`} className="break-words font-semibold text-accent-ink underline">{titles[job.clip_id] ?? `Clip #${job.clip_id}`}</Link><p className="mt-1 text-sm text-mute">{job.step} · {t(`queue.status.${job.status}`)} · {job.updated_at}</p></li>)}</ul>}</div>
      <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-line p-5"><span>Page {historyPage} · {completed?.total ?? 0} clips</span><div className="flex gap-2"><button disabled={loadingHistory || historyPage === 1} onClick={() => setHistoryPage(p => p-1)} className="min-h-11 rounded-lg border border-line-strong px-4 disabled:opacity-40">{t("queue.previous")}</button><button disabled={loadingHistory || !completed || historyPage*10 >= completed.total} onClick={() => setHistoryPage(p => p+1)} className="min-h-11 rounded-lg border border-line-strong px-4 disabled:opacity-40">{t("queue.next")}</button></div></footer>
    </Dialog.Popup></Dialog.Viewport></Dialog.Portal></Dialog.Root></>;
}
