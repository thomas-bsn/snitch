/** Throwaway UI exploration: three task-queue layouts on /admin/queue?variant=a|b|c. */
import { QueueFlow } from "./QueueFlow";
import { QueueHybrid } from "./QueueHybrid";
import { useEffect } from "react";
import { useSearchParams } from "react-router";
import { ArrowLeft, ArrowRight, CircleHelp, Clock3, Cpu, ListFilter, Pause, Play, RotateCcw, X } from "lucide-react";
import type { components } from "../../../api.gen";
import { useI18n } from "../../i18n";

type Job = Pick<components["schemas"]["Job"], "id" | "clip_id" | "step" | "status" | "last_error">;
type Controls = Pick<components["schemas"]["JobControls"], "paused" | "max_concurrency">;
export const sampleQueueJobs: Job[] = [
  { id: 240, clip_id: 18, step: "transcription.audio", status: "leased", last_error: "" },
  { id: 239, clip_id: 17, step: "voice.diarization", status: "leased", last_error: "" },
  { id: 238, clip_id: 16, step: "transcription.audio", status: "leased", last_error: "" },
  { id: 237, clip_id: 15, step: "transcription.audio", status: "leased", last_error: "" },
  { id: 236, clip_id: 14, step: "semantic.index", status: "pending", last_error: "" },
  { id: 235, clip_id: 13, step: "voice.diarization", status: "pending", last_error: "" },
  { id: 234, clip_id: 12, step: "transcription.audio", status: "dead", last_error: "Worker did not respond after three attempts" },
  { id: 233, clip_id: 11, step: "transcription.audio", status: "done", last_error: "" },
];
export const sampleQueueControls: Controls = { paused: false, max_concurrency: 4 };
const variants = [
  { key: "d", name: "Cartes détaillées" },
  { key: "e", name: "Parcours compact" },
  { key: "a", name: "Control room" },
  { key: "b", name: "Operations table" },
  { key: "c", name: "Work lanes" },
] as const;

function Help({ children }: { children: string }) {
  return (
    <details className="group relative inline-flex align-middle">
      <summary className="flex size-7 cursor-pointer list-none items-center justify-center rounded-full text-mute outline-none hover:bg-side focus-visible:ring-2 focus-visible:ring-accent" aria-label={children}>
        <CircleHelp size={16} aria-hidden="true" />
      </summary>
      <span className="absolute left-0 top-8 z-20 w-64 rounded-lg border border-line bg-surface p-3 text-sm font-normal text-ink shadow-lg group-open:block">{children}</span>
    </details>
  );
}

function State({ value }: { value: Job["status"] }) {
  const { t } = useI18n();
  const tone = value === "leased" ? "bg-accent-soft text-accent-ink" : value === "failed" || value === "dead" ? "bg-bad-bg text-bad-fg" : value === "done" ? "bg-ok-bg text-ok-fg" : "bg-side text-ink";
  return <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${tone}`}>{t(`queue.status.${value}`)}</span>;
}

function JobTitle({ job }: { job: Job }) {
  const { t } = useI18n();
  const stepNames: Record<string, string> = { "transcription.audio": "Transcribing speech", "voice.diarization": "Separating speakers", "semantic.index": "Preparing clip for search" };
  const clipNames: Record<number, string> = { 18: "Black Ops 3 · 21.04.47", 17: "Black Ops 3 · 21.02.13", 16: "Black Ops 3 · 20.58.02", 15: "Black Ops 3 · 20.41.36", 14: "Warzone · 19.32.10", 13: "Black Ops 3 · 19.10.05", 12: "Black Ops 3 · 18.54.22", 11: "Black Ops 3 · 18.30.14" };
  return <><strong className="font-semibold">{clipNames[job.clip_id] ?? `Clip ${job.clip_id}`}</strong><span className="text-mute">{stepNames[job.step] ?? job.step} · {t("queue.job", { id: job.id, clip: job.clip_id, status: t(`queue.status.${job.status}`) })}</span></>;
}

function VariantA({ jobs, controls }: { jobs: Job[]; controls: Controls }) {
  const { t } = useI18n();
  const active = jobs.filter((job) => job.status === "leased");
  const pending = jobs.filter((job) => job.status === "pending" || job.status === "retry_wait");
  const failed = jobs.filter((job) => job.status === "failed" || job.status === "dead");
  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-wrap items-end justify-between gap-3"><div><p className="text-sm text-mute">Administration / Task queue</p><h2 className="text-2xl font-semibold">Work in progress</h2></div><button className="h-11 rounded-lg bg-accent-solid px-4 font-semibold text-accent-fg">+ Schedule work</button></header>
      <section className="grid gap-3 sm:grid-cols-3" aria-label="Queue summary">
        {[{ name: "Running", count: active.length, icon: Play }, { name: "Waiting", count: pending.length, icon: Clock3 }, { name: "Needs attention", count: failed.length, icon: RotateCcw }].map(({ name, count, icon: Icon }) => <article key={name} className="rounded-xl border border-line bg-surface p-4"><div className="flex items-center justify-between text-sm text-mute"><span>{name}</span><Icon size={17} aria-hidden="true" /></div><p className="mt-2 text-3xl font-semibold">{count}</p></article>)}
      </section>
      <section className="rounded-xl border border-line bg-surface p-5"><div className="mb-4 flex items-center gap-2"><Cpu size={18} aria-hidden="true"/><h3 className="font-semibold">Capacity</h3><Help>Global concurrency is the maximum number of jobs all Workers may run at the same time. Keep it low when transcription and voice models share one GPU.</Help></div><p className="text-sm text-mute">{active.length} of {controls.max_concurrency} job(s) running · Local gaming PC worker · {controls.paused ? "New work paused" : "Queue active"}</p><div className="mt-3 h-2 overflow-hidden rounded-full bg-side"><div className="h-full rounded-full bg-accent-solid" style={{ width: `${Math.min(100, (active.length / Math.max(1, controls.max_concurrency)) * 100)}%` }} /></div><button className="mt-4 rounded-lg border border-line-strong px-3 py-2 text-sm">{controls.paused ? "Resume queue" : "Pause new work"}</button></section>
      <section><h3 className="mb-3 font-semibold">Running now</h3>{active.length ? <div className="grid gap-3 md:grid-cols-2">{active.map((job) => <article key={job.id} className="rounded-xl border border-accent bg-surface p-4"><div className="flex justify-between gap-2"><JobTitle job={job}/><State value={job.status}/></div><div className="mt-4 h-2 rounded-full bg-side"><div className="h-full w-2/3 rounded-full bg-accent-solid" /></div><p className="mt-2 text-xs text-mute">Worker is processing this clip</p></article>)}</div> : <p className="rounded-xl border border-dashed border-line-strong p-5 text-sm text-mute">No active jobs. Scheduled work will appear here.</p>}</section>
      <section><div className="mb-3 flex items-center justify-between"><h3 className="font-semibold">Recent work</h3><button className="inline-flex items-center gap-2 text-sm text-mute"><ListFilter size={16}/>Filters</button></div><div className="divide-y divide-line rounded-xl border border-line bg-surface">{jobs.slice(0, 6).map((job) => <div key={job.id} className="flex flex-wrap items-center justify-between gap-3 p-4"><div className="flex flex-col gap-1"><JobTitle job={job}/></div><State value={job.status}/></div>)}{!jobs.length && <p className="p-5 text-sm text-mute">No jobs yet. Schedule the first task from a clip.</p>}</div></section>
    </div>
  );
}

function VariantB({ jobs, controls }: { jobs: Job[]; controls: Controls }) {
  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_290px]">
      <main className="min-w-0"><header className="mb-6 flex flex-wrap items-end justify-between gap-3"><div><p className="text-sm text-mute">Administration / Task queue</p><h2 className="text-2xl font-semibold">Tasks</h2></div><button className="h-11 rounded-lg bg-accent-solid px-4 font-semibold text-accent-fg">Schedule task</button></header><div className="mb-3 flex items-center justify-between"><p className="text-sm text-mute">Sorted by most recent</p><button className="inline-flex items-center gap-2 rounded-lg border border-line-strong px-3 py-2 text-sm"><ListFilter size={16}/>Filter</button></div><div className="overflow-x-auto rounded-xl border border-line bg-surface"><table className="w-full min-w-[650px] text-left text-sm"><thead className="border-b border-line bg-side text-xs uppercase tracking-wide text-mute"><tr><th className="p-3">Task / clip</th><th className="p-3">Step</th><th className="p-3">Status</th><th className="p-3">Updated</th><th className="p-3">Actions</th></tr></thead><tbody className="divide-y divide-line">{jobs.map((job) => <tr key={job.id}><td className="p-3"><a href={`/clips/${job.clip_id}`} className="font-semibold text-accent-ink underline">Clip {job.clip_id}</a><span className="ml-2 text-mute">#{job.id}</span></td><td className="p-3">{job.step}</td><td className="p-3"><State value={job.status}/></td><td className="p-3 text-mute">Recent</td><td className="p-3"><button className="rounded-md px-2 py-1 hover:bg-side">Details</button></td></tr>)}{!jobs.length && <tr><td colSpan={5} className="p-8 text-center text-mute">No tasks match these filters.</td></tr>}</tbody></table></div></main>
      <aside className="flex flex-col gap-4"><section className="rounded-xl border border-line bg-surface p-4"><div className="flex items-center justify-between"><h3 className="font-semibold">Queue controls</h3><button aria-label="Save controls" className="rounded-md bg-accent-solid px-3 py-2 text-sm font-semibold text-accent-fg">Save</button></div><label className="mt-5 flex items-center justify-between gap-2 text-sm">Global concurrency <Help>Maximum jobs running across all Workers at the same time.</Help></label><div className="mt-2 flex items-center gap-3"><input aria-label="Global concurrency" type="range" min="1" max="4" defaultValue={controls.max_concurrency} className="w-full accent-accent-solid"/><span className="font-semibold">{controls.max_concurrency}</span></div><p className="mt-2 text-xs text-mute">Increasing this may make GPU memory run out sooner.</p><label className="mt-5 flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={controls.paused}/> Pause new acquisitions</label><p className="mt-4 rounded-lg bg-side p-3 text-xs text-mute">Workers still finish jobs they already acquired.</p></section><section className="rounded-xl border border-line bg-surface p-4"><h3 className="font-semibold">Quick schedule</h3><p className="mt-1 text-sm text-mute">Choose a clip and a task. Snitch selects a compatible Worker.</p><button className="mt-3 w-full rounded-lg border border-line-strong px-3 py-2 text-sm">Choose clip…</button></section></aside>
    </div>
  );
}

function VariantC({ jobs }: { jobs: Job[] }) {
  const lanes = [{ title: "Running", status: ["leased"] }, { title: "Waiting", status: ["pending", "retry_wait"] }, { title: "Needs attention", status: ["failed", "dead"] }, { title: "Completed", status: ["done", "cancelled"] }] as const;
  return (
    <div className="flex flex-col gap-6"><header className="flex flex-wrap items-end justify-between gap-3"><div><p className="text-sm text-mute">Administration / Task queue</p><h2 className="text-2xl font-semibold">Queue board</h2><p className="mt-1 text-sm text-mute">See what is moving, waiting, and blocked.</p></div><button className="h-11 rounded-lg bg-accent-solid px-4 font-semibold text-accent-fg">+ Add a task</button></header><div className="flex flex-wrap gap-3"><button className="rounded-full bg-ink px-4 py-2 text-sm text-surface">All tasks</button><button className="rounded-full border border-line-strong px-4 py-2 text-sm">Transcription</button><button className="rounded-full border border-line-strong px-4 py-2 text-sm">Voice</button><button className="rounded-full border border-line-strong px-4 py-2 text-sm">Other</button></div><div className="grid gap-4 xl:grid-cols-4">{lanes.map((lane) => { const items = jobs.filter((job) => (lane.status as readonly string[]).includes(job.status)); return <section key={lane.title} className="min-h-52 rounded-xl bg-side p-3"><div className="mb-3 flex items-center justify-between"><h3 className="font-semibold">{lane.title}</h3><span className="rounded-full bg-surface px-2 py-0.5 text-xs">{items.length}</span></div><div className="flex flex-col gap-2">{items.map((job) => <article key={job.id} className="rounded-lg border border-line bg-surface p-3 shadow-sm"><div className="flex items-center justify-between gap-2"><span className="text-xs text-mute">#{job.id} · Clip {job.clip_id}</span><State value={job.status}/></div><p className="mt-2 font-semibold">{job.step}</p><p className="mt-1 line-clamp-2 text-xs text-bad-fg">{job.last_error || ""}</p><div className="mt-3 flex gap-2"><button className="inline-flex min-h-9 items-center gap-1 rounded-md border border-line px-2 text-xs"><Pause size={13}/>Pause</button><button className="inline-flex min-h-9 items-center gap-1 rounded-md border border-line px-2 text-xs"><X size={13}/>Cancel</button></div></article>)}{!items.length && <p className="rounded-lg border border-dashed border-line-strong p-3 text-xs text-mute">Nothing here right now.</p>}</div></section>; })}</div><details className="rounded-xl border border-line bg-surface p-4"><summary className="cursor-pointer font-semibold">Queue-wide settings and help</summary><p className="mt-3 text-sm text-mute">Use these settings to control how quickly work is acquired and how many jobs share your PC resources.</p><button className="mt-3 rounded-lg border border-line-strong px-3 py-2 text-sm">Open queue controls</button></details></div>
  );
}

export function QueuePrototype({ jobs, controls }: { jobs: Job[]; controls: Controls }) {
  const [search, setSearch] = useSearchParams();
  const current = variants.find((variant) => variant.key === search.get("variant")) ?? variants[0];
  function setVariant(key: string) { const next = new URLSearchParams(search); next.set("variant", key); setSearch(next, { replace: true }); }
  useEffect(() => { if (!search.get("variant")) setVariant("a"); }, [search]);
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.target instanceof HTMLElement && (event.target.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(event.target.tagName))) return;
      if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
      event.preventDefault(); const index = variants.findIndex((variant) => variant.key === current.key);
      setVariant(variants[(index + (event.key === "ArrowRight" ? 1 : -1) + variants.length) % variants.length].key);
    }
    window.addEventListener("keydown", onKey); return () => window.removeEventListener("keydown", onKey);
  }, [current.key, search]);
  return <>
    <div className="mb-5 rounded-lg border border-warn-fg bg-warn-bg px-4 py-3 text-sm text-warn-fg"><strong>UI prototype · read-only.</strong> Sample layouts for issue #98. Controls do not change jobs or settings.</div>
    {current.key === "e" ? <QueueFlow/> : current.key === "d" ? <QueueHybrid/> : current.key === "a" ? <VariantA jobs={jobs} controls={controls}/> : current.key === "b" ? <VariantB jobs={jobs} controls={controls}/> : <VariantC jobs={jobs}/>}
    <nav aria-label="Queue prototype variants" className="fixed bottom-4 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-full border border-white/20 bg-slate-950 px-3 py-2 text-white shadow-2xl">
      <button type="button" aria-label="Previous layout" onClick={() => setVariant(variants[(variants.findIndex((variant) => variant.key === current.key) + variants.length - 1) % variants.length].key)} className="grid size-9 place-items-center rounded-full hover:bg-white/15 focus-visible:outline-2 focus-visible:outline-white"><ArrowLeft size={17}/></button>
      <span className="min-w-40 text-center text-sm"><strong>{current.key.toUpperCase()}</strong> · {current.name}</span>
      <button type="button" aria-label="Next layout" onClick={() => setVariant(variants[(variants.findIndex((variant) => variant.key === current.key) + 1) % variants.length].key)} className="grid size-9 place-items-center rounded-full hover:bg-white/15 focus-visible:outline-2 focus-visible:outline-white"><ArrowRight size={17}/></button>
    </nav>
  </>;
}
