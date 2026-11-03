// PROTOTYPE — trois mises en page de la fiche vidéo, structurellement différentes.
import { ChevronRight, ScrollText } from "lucide-react";
import { useState } from "react";
import { Modal } from "../../components/Modal";
import type { StepKey } from "./data";
import {
  AnalysisActions,
  ErrorBox,
  FakePlayer,
  LogViewer,
  MetaGrid,
  MetaHeader,
  OverallBadge,
  StepIcon,
  StepList,
  statusText,
  tabBtn,
  TranscriptPanel,
  usePlayer,
  VoicesPanel,
} from "./parts";
import type { Sim } from "./useFicheSim";

type Tab = "transcript" | "voices";

function WorkTabs({ sim, player, className = "" }: { sim: Sim; player: ReturnType<typeof usePlayer>; className?: string }) {
  const [tab, setTab] = useState<Tab>(() =>
    new URLSearchParams(window.location.search).get("tab") === "voices" ? "voices" : "transcript",
  );
  const todo = sim.voices.filter((v) => ["suggested", "ambiguous", "unknown"].includes(v.state)).length;
  return (
    <div className={`flex min-w-0 flex-col rounded-xl border border-line ${className}`}>
      <div className="flex gap-1 border-b border-line px-3" role="tablist">
        <button type="button" role="tab" aria-selected={tab === "transcript"} className={tabBtn(tab === "transcript")} onClick={() => setTab("transcript")}>
          Transcript <span className="text-xs text-mute">{sim.segments.length}</span>
        </button>
        <button type="button" role="tab" aria-selected={tab === "voices"} className={tabBtn(tab === "voices")} onClick={() => setTab("voices")}>
          Voices <span className="text-xs text-mute">{sim.voices.length}</span>
          {todo > 0 && <span className="rounded-full bg-warn-bg px-1.5 text-xs text-warn-fg">{todo} à traiter</span>}
        </button>
      </div>
      <div className="max-h-[70vh] overflow-y-auto">
        {tab === "transcript" ? <TranscriptPanel sim={sim} playhead={player.playhead} onSeek={player.seek} /> : <VoicesPanel sim={sim} onSeek={player.seek} playhead={player.playhead} />}
      </div>
    </div>
  );
}

/* A — Deux colonnes : la vidéo et ses métadonnées à gauche, l'analyse en panneau fixe à droite, le travail en dessous sur toute la largeur. */
export function VariantA({ sim }: { sim: Sim }) {
  const player = usePlayer();
  return (
    <section className="flex flex-col gap-5 px-4 py-4 md:px-7 md:py-5">
      <div className="grid gap-5 lg:grid-cols-[minmax(0,1.5fr)_minmax(320px,1fr)]">
        <div className="flex min-w-0 flex-col gap-4">
          <FakePlayer {...player} />
          <MetaHeader />
        </div>
        <aside className="flex flex-col gap-3 self-start rounded-xl border border-line p-4 lg:sticky lg:top-4">
          <div className="flex items-center justify-between gap-2">
            <h2 className="m-0 text-base font-semibold">Analyse</h2>
            <OverallBadge sim={sim} />
          </div>
          <StepList sim={sim} />
          <AnalysisActions sim={sim} />
        </aside>
      </div>
      <WorkTabs sim={sim} player={player} />
    </section>
  );
}

/* B — Bandeau de workflow : les étapes en ligne en haut de page, comme un pipeline ; un clic sur une étape déplie son détail. Dessous, la fiche classique vidéo | onglets. */
export function VariantB({ sim }: { sim: Sim }) {
  const player = usePlayer();
  const failed = sim.steps.find((s) => s.status === "failed")?.key ?? null;
  const [open, setOpen] = useState<StepKey | null>(failed);
  const sel = sim.steps.find((s) => s.key === (open ?? failed));
  return (
    <section className="flex flex-col gap-4 px-4 py-4 md:px-7 md:py-5">
      <MetaHeader compact />
      <div className="flex flex-col gap-3 rounded-xl border border-line p-3">
        <div className="flex flex-wrap items-center gap-3">
          <OverallBadge sim={sim} />
          <div className="ml-auto"><AnalysisActions sim={sim} /></div>
        </div>
        <ol className="m-0 flex list-none items-stretch gap-1 overflow-x-auto p-0">
          {sim.steps.map((s, i) => (
            <li key={s.key} className="flex min-w-[150px] flex-1 items-center gap-1">
              <button
                type="button"
                onClick={() => setOpen(open === s.key ? null : s.key)}
                className={`flex w-full flex-col gap-1 rounded-lg border px-3 py-2 text-left ${open === s.key ? "border-accent bg-accent-soft" : s.status === "failed" ? "border-bad-fg/40 bg-bad-bg" : "border-line hover:bg-side"}`}
              >
                <span className="flex items-center gap-2 text-sm font-medium"><StepIcon s={s} />{s.label}</span>
                <span className="text-xs text-mute">{statusText[s.status]}{s.status === "running" ? ` · ${s.progress ?? 0} %` : ""}</span>
                {s.status === "running" && (
                  <span className="h-1 rounded-full bg-side"><span className="block h-1 rounded-full bg-accent" style={{ width: `${s.progress ?? 0}%` }} /></span>
                )}
              </button>
              {i < sim.steps.length - 1 && <ChevronRight className="size-4 shrink-0 text-mute" />}
            </li>
          ))}
        </ol>
        {sel && open && (
          <div className="flex flex-col gap-2 border-t border-line pt-3">
            <p className="m-0 text-sm"><strong>{sel.label}</strong> — {statusText[sel.status]}{sel.worker ? ` sur ${sel.worker}` : ""}</p>
            {sel.waitCause && <p className="m-0 text-sm text-warn-fg">{sel.waitCause}</p>}
            {sel.error && sel.status === "failed" && <ErrorBox step={sel} />}
            {(sim.logs[sel.key]?.length ?? 0) > 0 ? (
              <>
                <p className="m-0 ml-6.5 flex items-center gap-1 text-xs text-mute"><ScrollText className="size-3.5" /> {sim.logs[sel.key].length} dernières lignes du worker</p>
                <LogViewer lines={sim.logs[sel.key]} />
              </>
            ) : (
              <p className="m-0 ml-6.5 text-xs text-mute">Pas de logs pour cette étape.</p>
            )}
          </div>
        )}
      </div>
      <div className="grid gap-5 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)]">
        <div className="flex min-w-0 flex-col gap-4">
          <FakePlayer {...player} />
          <MetaGrid />
        </div>
        <WorkTabs sim={sim} player={player} />
      </div>
    </section>
  );
}

/* C — Lecteur et transcript côte à côte, rien d'autre à l'écran ; l'état de l'analyse est une pastille vivante dans l'en-tête qui ouvre un panneau latéral. Les métadonnées se replient. */
export function VariantC({ sim }: { sim: Sim }) {
  const player = usePlayer();
  const [panel, setPanel] = useState(false);
  const [meta, setMeta] = useState(false);
  return (
    <section className="flex flex-col gap-4 px-4 py-4 md:px-7 md:py-5">
      <div className="flex flex-wrap items-start gap-3">
        <div className="min-w-0 flex-1"><MetaHeader compact /></div>
        <button type="button" onClick={() => setPanel(true)} className="inline-flex items-center gap-2 rounded-lg border border-line-strong px-3 py-1.5 hover:bg-side">
          <span className="text-sm font-medium">Analyse</span>
          <OverallBadge sim={sim} />
          <ChevronRight className="size-4 text-mute" />
        </button>
      </div>
      <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <div className="flex min-w-0 flex-col gap-3 lg:sticky lg:top-4 lg:self-start">
          <FakePlayer {...player} />
          <button type="button" onClick={() => setMeta((m) => !m)} className="inline-flex w-fit items-center gap-1 text-sm text-accent-ink hover:underline">
            <ChevronRight className={`size-4 transition-transform ${meta ? "rotate-90" : ""}`} /> {meta ? "Masquer" : "Afficher"} toutes les métadonnées
          </button>
          {meta && <div className="rounded-xl border border-line p-4"><MetaGrid /></div>}
        </div>
        <WorkTabs sim={sim} player={player} />
      </div>
      <Modal open={panel} onOpenChange={setPanel} title="Analyse de la vidéo" closeLabel="Fermer" variant="side">
        <OverallBadge sim={sim} />
        <AnalysisActions sim={sim} />
        <StepList sim={sim} />
      </Modal>
    </section>
  );
}
