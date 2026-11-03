// PROTOTYPE — morceaux partagés par les variantes de la fiche vidéo.
import {
  AlertTriangle,
  Ban,
  Check,
  CheckCircle2,
  ChevronDown,
  Circle,
  CircleDashed,
  Clock,
  Dices,
  EyeOff,
  Loader2,
  Pause,
  Pencil,
  Play,
  RotateCcw,
  ScrollText,
  Trash2,
  UserX,
  Volume2,
  XCircle,
} from "lucide-react";
import { useState, type ReactNode } from "react";
import { buttonClass } from "../../components/buttons";
import { Modal } from "../../components/Modal";
import {
  cascade,
  clipMeta,
  STEP_LABELS,
  STEP_ORDER,
  type Segment,
  type Step,
  type StepKey,
  type Voice,
} from "./data";
import { launchPlan, type Sim } from "./useFicheSim";
import { VoiceDot, Waveform } from "./waveform";

export function fmt(s: number) {
  const m = Math.floor(s / 60);
  const r = Math.floor(s % 60);
  return `${m}:${String(r).padStart(2, "0")}`;
}

/* ---------- Lecteur factice ---------- */

export function FakePlayer({ playhead, playing, onToggle }: { playhead: number; playing: boolean; onToggle: () => void }) {
  return (
    <div className="relative aspect-video w-full overflow-hidden rounded-xl bg-[#0b0b0e]">
      <div className="absolute inset-0 flex items-center justify-center">
        <button
          type="button"
          onClick={onToggle}
          className="flex size-16 items-center justify-center rounded-full bg-white/15 text-white backdrop-blur hover:bg-white/25"
          aria-label={playing ? "Pause" : "Lecture"}
        >
          {playing ? <Pause className="size-7" /> : <Play className="size-7 translate-x-0.5" />}
        </button>
      </div>
      <div className="absolute inset-x-0 bottom-0 flex items-center gap-3 bg-gradient-to-t from-black/70 to-transparent px-4 pt-8 pb-3 text-xs text-white/90">
        <span className="tabular-nums">{fmt(playhead)}</span>
        <div className="h-1 flex-1 rounded-full bg-white/25">
          <div className="h-1 rounded-full bg-white" style={{ width: `${(playhead / clipMeta.durationS) * 100}%` }} />
        </div>
        <span className="tabular-nums">{clipMeta.duration}</span>
        <Volume2 className="size-4" />
      </div>
      <span className="absolute top-3 left-3 rounded bg-black/50 px-2 py-0.5 text-[11px] text-white/80">PROTOTYPE · lecteur factice</span>
    </div>
  );
}

/* ---------- Métadonnées ---------- */

export function MetaHeader({ compact = false }: { compact?: boolean }) {
  const [editing, setEditing] = useState(false);
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="m-0 text-[22px] font-semibold break-words">{clipMeta.title}</h1>
          <p className="mt-1 mb-0 text-xs text-mute">{clipMeta.file}</p>
        </div>
        <button type="button" className={buttonClass()} onClick={() => setEditing(true)}>
          <Pencil className="size-4" /> Modifier les métadonnées
        </button>
      </div>
      <p className="m-0 text-mute">
        {[clipMeta.owner, clipMeta.recordedAt, clipMeta.duration, clipMeta.visibility].join(" · ")}
      </p>
      {!compact && <MetaGrid />}
      <Modal open={editing} onOpenChange={setEditing} title="Modifier les métadonnées" closeLabel="Fermer" variant="side">
        <div className="flex flex-col gap-3">
          {[["Titre", clipMeta.title], ["Date d'enregistrement", clipMeta.recordedAt], ...clipMeta.fields].map(([k, v]) => (
            <label key={k} className="flex flex-col gap-1 text-sm">
              <span className="font-medium">{k}</span>
              <input defaultValue={v} className="h-9 rounded-lg border border-line-strong bg-surface px-3" />
            </label>
          ))}
          <div className="flex justify-end gap-2">
            <button type="button" className={buttonClass()} onClick={() => setEditing(false)}>Annuler</button>
            <button type="button" className={buttonClass("primary")} onClick={() => setEditing(false)}>Enregistrer</button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

export function MetaGrid() {
  const rows: [string, string][] = [
    ["Dossier", clipMeta.folder],
    ["Ajoutée le", clipMeta.addedAt],
    ["Taille", clipMeta.size],
    ["Vidéo", clipMeta.format],
    ["Audio", clipMeta.audio],
    ...clipMeta.fields,
  ];
  return (
    <dl className="m-0 grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1.5 text-sm">
      {rows.map(([k, v]) => (
        <div key={k} className="contents">
          <dt className="text-mute">{k}</dt>
          <dd className="m-0 min-w-0 break-words">{v}</dd>
        </div>
      ))}
    </dl>
  );
}

/* ---------- Libellés des voix ---------- */

export function speakerName(v: Voice | undefined): { text: string; tone: "person" | "guess" | "raw" | "random" | "none" } {
  if (!v) return { text: "Voix supprimée", tone: "none" };
  switch (v.state) {
    case "validated":
      return { text: v.person!, tone: "person" };
    case "suggested":
      return { text: `${v.person} ?`, tone: "guess" };
    case "random":
      return { text: "Random", tone: "random" };
    case "ignored":
      return { text: "", tone: "none" };
    default:
      return { text: v.label.toLowerCase().replace("speaker_", "speaker_"), tone: "raw" };
  }
}

const toneClass = {
  person: "bg-accent-soft text-accent-ink",
  guess: "border border-dashed border-accent text-accent-ink",
  raw: "bg-side text-mute",
  random: "bg-warn-bg text-warn-fg",
  none: "",
};

export function SpeakerChip({ v }: { v: Voice | undefined }) {
  const n = speakerName(v);
  if (n.tone === "none") return <span className="text-xs text-mute italic">voix ignorée</span>;
  return <span className={`inline-flex w-fit rounded-full px-2 py-0.5 text-xs font-medium ${toneClass[n.tone]}`}>{n.text}</span>;
}

/* ---------- Onglet Transcript ---------- */

export function TranscriptPanel({ sim, playhead, onSeek }: { sim: Sim; playhead: number; onSeek: (s: number) => void }) {
  const [editing, setEditing] = useState<number | null>(null);
  const [draft, setDraft] = useState("");
  if (sim.segments.length === 0)
    return <Empty>Pas de transcription. Lancez l'étape « Transcription » depuis l'analyse.</Empty>;
  return (
    <ol className="m-0 flex list-none flex-col p-0">
      {sim.segments.map((s: Segment) => {
        const v = sim.voices.find((x) => x.id === s.voiceId);
        const active = playhead >= s.start && playhead < s.end;
        const ignored = v?.state === "ignored";
        return (
          <li key={s.id} className={`group flex gap-3 border-b border-line px-4 py-3 ${active ? "bg-hit" : ""}`}>
            <button type="button" onClick={() => onSeek(s.start)} className="h-6 shrink-0 rounded px-1.5 text-xs text-accent-ink tabular-nums hover:bg-accent-soft" title="Écouter depuis ici">
              {fmt(s.start)}
            </button>
            <div className={`flex min-w-0 flex-1 flex-col gap-1 ${ignored ? "opacity-80" : ""}`}>
              <SpeakerChip v={v} />
              {editing === s.id ? (
                <div className="flex flex-col gap-2">
                  <textarea value={draft} onChange={(e) => setDraft(e.target.value)} className="min-h-16 rounded-lg border border-line-strong bg-surface p-2 text-sm" autoFocus />
                  <div className="flex gap-2">
                    <button type="button" className={buttonClass("primary")} onClick={() => { sim.editSegment(s.id, draft); setEditing(null); }}>Enregistrer</button>
                    <button type="button" className={buttonClass()} onClick={() => setEditing(null)}>Annuler</button>
                  </div>
                </div>
              ) : (
                <p className="m-0 leading-relaxed">
                  {s.text}
                  {s.edited && <span className="ml-2 text-xs text-mute">(corrigé)</span>}
                </p>
              )}
            </div>
            {editing !== s.id && (
              <div className="flex shrink-0 items-start gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100">
                <IconBtn label="Écouter" onClick={() => onSeek(s.start)}><Play className="size-4" /></IconBtn>
                <IconBtn label="Corriger le texte" onClick={() => { setEditing(s.id); setDraft(s.text); }}><Pencil className="size-4" /></IconBtn>
              </div>
            )}
          </li>
        );
      })}
    </ol>
  );
}

/* ---------- Onglet Voices ---------- */

const stateLabel: Record<Voice["state"], string> = {
  validated: "Identifiée",
  suggested: "Suggestion",
  ambiguous: "Ambiguë",
  unknown: "Inconnue",
  ignored: "Ignorée (fausse détection)",
  random: "Random",
};

export function VoicesPanel({ sim, onSeek, playhead }: { sim: Sim; onSeek: (s: number) => void; playhead?: number }) {
  if (sim.voices.length === 0)
    return <Empty>Pas de voix détectées. Lancez l'étape « Détection des voix » depuis l'analyse.</Empty>;
  const todo = sim.voices.filter((v) => ["suggested", "ambiguous", "unknown"].includes(v.state)).length;
  return (
    <div className="flex flex-col">
      {todo > 0 && (
        <p className="m-0 flex items-center gap-2 border-b border-line bg-warn-bg px-4 py-2 text-sm text-warn-fg">
          <AlertTriangle className="size-4" /> {todo} voix à traiter dans cette vidéo
        </p>
      )}
      <div className="border-b border-line px-4 py-3">
        <Waveform segments={sim.segments} voices={sim.voices} playhead={playhead} onSeek={onSeek} lanes height={64} />
      </div>
      <ul className="m-0 flex list-none flex-col p-0">
        {sim.voices.map((v) => (
          <VoiceRow key={v.id} v={v} sim={sim} onSeek={onSeek} />
        ))}
      </ul>
    </div>
  );
}

export function VoiceRow({ v, sim, onSeek }: { v: Voice; sim: Sim; onSeek: (s: number) => void }) {
  const a = sim.voiceActions;
  const first = sim.segments.find((s) => s.voiceId === v.id);
  const muted = v.state === "ignored" || v.state === "random";
  return (
    <li className={`flex flex-col gap-2 border-b border-line px-4 py-3 ${muted ? "bg-side/60" : ""}`}>
      <div className="flex flex-wrap items-center gap-2">
        <VoiceDot id={v.id} ignored={v.state === "ignored"} />
        <span className="font-mono text-xs text-mute">{v.label.toLowerCase()}</span>
        <SpeakerChip v={v} />
        <span className="text-xs text-mute">· {stateLabel[v.state]} · {v.turns} interventions · {fmt(v.speakingS)} de parole</span>
        {first && (
          <button type="button" onClick={() => onSeek(first.start)} className="ml-auto inline-flex items-center gap-1 text-xs text-accent-ink hover:underline">
            <Play className="size-3.5" /> Écouter un extrait
          </button>
        )}
      </div>
      {v.group && v.state === "unknown" && <p className="m-0 text-xs text-mute">Entendue aussi ailleurs : {v.group}</p>}
      {v.refused.length > 0 && <p className="m-0 text-xs text-mute">Refusé : {v.refused.join(", ")} (ne sera plus proposé)</p>}
      <div className="flex flex-wrap gap-2">
        {v.state === "validated" && (
          <button type="button" className={buttonClass()} onClick={() => a.reject(v)}>
            <UserX className="size-4" /> Ce n'est pas {v.person}
          </button>
        )}
        {v.state === "suggested" && (
          <>
            <button type="button" className={buttonClass("primary")} onClick={() => a.validate(v, v.person!)}>
              <Check className="size-4" /> C'est {v.person}
            </button>
            <button type="button" className={buttonClass()} onClick={() => a.reject(v)}>
              <UserX className="size-4" /> Ce n'est pas {v.person}
            </button>
          </>
        )}
        {(v.state === "ambiguous" || v.state === "unknown") && (
          <PersonPicker v={v} onPick={(p) => a.validate(v, p)} />
        )}
        {!muted && v.state !== "validated" && (
          <>
            <button type="button" className={buttonClass()} onClick={() => a.random(v)} title="Vraie voix, volontairement non identifiée">
              <Dices className="size-4" /> Random
            </button>
            <button type="button" className={buttonClass()} onClick={() => a.ignore(v)} title="Fausse détection : ne pas compter cette voix">
              <EyeOff className="size-4" /> Ignorer
            </button>
          </>
        )}
        {muted && (
          <button type="button" className={buttonClass()} onClick={() => a.restore(v)}>
            <RotateCcw className="size-4" /> Rétablir
          </button>
        )}
      </div>
    </li>
  );
}

function PersonPicker({ v, onPick }: { v: Voice; onPick: (p: string) => void }) {
  const [open, setOpen] = useState(false);
  const people = ["Marc Lefèvre", "Julie Martin", "Claire Dubois", "Thomas Roux", "Hugo Bernard"].filter((p) => !v.refused.includes(p));
  return (
    <div className="relative">
      <button type="button" className={buttonClass("primary")} onClick={() => setOpen((o) => !o)}>
        Identifier <ChevronDown className="size-4" />
      </button>
      {open && (
        <ul className="absolute z-10 mt-1 w-60 list-none rounded-lg border border-line bg-surface p-1 shadow-pop">
          {v.candidates?.map((c) => (
            <li key={c}>
              <button type="button" className="w-full rounded px-2 py-1.5 text-left hover:bg-side" onClick={() => onPick(c)}>
                {c} <span className="text-xs text-mute">· candidat</span>
              </button>
            </li>
          ))}
          {people.filter((p) => !v.candidates?.includes(p)).map((p) => (
            <li key={p}>
              <button type="button" className="w-full rounded px-2 py-1.5 text-left hover:bg-side" onClick={() => onPick(p)}>{p}</button>
            </li>
          ))}
          <li><button type="button" className="w-full rounded px-2 py-1.5 text-left text-accent-ink hover:bg-side">+ Nouvelle personne…</button></li>
        </ul>
      )}
    </div>
  );
}

/* ---------- Analyse : étapes en direct ---------- */

export function StepIcon({ s }: { s: Step }) {
  const c = "size-4 shrink-0";
  switch (s.status) {
    case "done": return <CheckCircle2 className={`${c} text-ok-fg`} />;
    case "running": return <Loader2 className={`${c} animate-spin text-accent`} />;
    case "pending": return <Circle className={`${c} text-mute`} />;
    case "waiting": return <Clock className={`${c} text-warn-fg`} />;
    case "failed": return <XCircle className={`${c} text-bad-fg`} />;
    case "cancelled": return <Ban className={`${c} text-mute`} />;
    default: return <CircleDashed className={`${c} text-mute`} />;
  }
}

export const statusText: Record<Step["status"], string> = {
  absent: "Pas faite",
  pending: "En file",
  waiting: "En attente",
  running: "En cours",
  done: "Terminée",
  failed: "Échec",
  cancelled: "Annulée",
};

export function overall(sim: Sim): { label: string; tone: "ok" | "run" | "bad" | "idle" | "warn" } {
  const st = sim.steps;
  if (st.some((s) => s.status === "running")) {
    const cur = st.find((s) => s.status === "running")!;
    return { label: `${cur.label} · ${cur.progress ?? 0} %`, tone: "run" };
  }
  if (st.some((s) => s.status === "waiting")) return { label: "En attente d'un worker voix", tone: "warn" };
  if (st.some((s) => s.status === "pending")) return { label: "En file", tone: "run" };
  if (st.some((s) => s.status === "failed")) return { label: `Échec : ${st.find((s) => s.status === "failed")!.label}`, tone: "bad" };
  if (st.every((s) => s.status === "done")) return { label: "Analyse complète", tone: "ok" };
  if (st.every((s) => s.status === "absent")) return { label: "Jamais analysée", tone: "idle" };
  return { label: "Analyse partielle", tone: "idle" };
}

export const toneBadge = {
  ok: "bg-ok-bg text-ok-fg",
  run: "bg-accent-soft text-accent-ink",
  bad: "bg-bad-bg text-bad-fg",
  warn: "bg-warn-bg text-warn-fg",
  idle: "bg-side text-mute",
};

export function OverallBadge({ sim }: { sim: Sim }) {
  const o = overall(sim);
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${toneBadge[o.tone]}`}>
      {o.tone === "run" && <Loader2 className="size-3.5 animate-spin" />}
      {o.tone === "bad" && <XCircle className="size-3.5" />}
      {o.tone === "ok" && <CheckCircle2 className="size-3.5" />}
      {o.tone === "warn" && <Clock className="size-3.5" />}
      {o.label}
    </span>
  );
}

/** Liste verticale des étapes, chaque échec déplie son erreur et ses logs. */
export function StepList({ sim, dense = false }: { sim: Sim; dense?: boolean }) {
  const [openLogs, setOpenLogs] = useState<StepKey | null>(null);
  return (
    <ol className="m-0 flex list-none flex-col p-0">
      {sim.steps.map((s) => (
        <li key={s.key} className={`flex flex-col gap-2 border-b border-line last:border-b-0 ${dense ? "py-2" : "py-3"}`}>
          <div className="flex items-center gap-2.5">
            <StepIcon s={s} />
            <span className={`font-medium ${s.status === "absent" ? "text-mute" : ""}`}>{s.label}</span>
            <span className="ml-auto text-xs text-mute">
              {statusText[s.status]}
              {s.durationS != null && s.status === "done" && ` · ${fmt(s.durationS)}`}
              {s.worker && s.status !== "absent" && ` · ${s.worker}`}
            </span>
          </div>
          {s.status === "running" && (
            <div className="ml-6.5 h-1.5 rounded-full bg-side">
              <div className="h-1.5 rounded-full bg-accent transition-all" style={{ width: `${s.progress ?? 0}%` }} />
            </div>
          )}
          {s.status === "waiting" && <p className="m-0 ml-6.5 text-xs text-warn-fg">{s.waitCause}</p>}
          {s.error && s.status === "failed" && <ErrorBox step={s} />}
          {(sim.logs[s.key]?.length ?? 0) > 0 && (s.status === "failed" || s.status === "running" || openLogs === s.key) && (
            <button type="button" onClick={() => setOpenLogs(openLogs === s.key ? null : s.key)} className="ml-6.5 inline-flex w-fit items-center gap-1 text-xs text-accent-ink hover:underline">
              <ScrollText className="size-3.5" /> {openLogs === s.key ? "Masquer les logs" : `Logs du worker (${sim.logs[s.key].length} dernières lignes)`}
            </button>
          )}
          {openLogs === s.key && <LogViewer lines={sim.logs[s.key] ?? []} />}
        </li>
      ))}
    </ol>
  );
}

export function ErrorBox({ step }: { step: Step }) {
  return (
    <div className="ml-6.5 flex flex-col gap-1 rounded-lg border border-bad-fg/30 bg-bad-bg px-3 py-2 text-sm text-bad-fg">
      <div className="flex items-center gap-2 font-medium">
        <code className="rounded bg-black/10 px-1.5 py-0.5 text-xs">{step.error!.code}</code>
        <span className="text-xs opacity-80">{step.error!.at}</span>
      </div>
      <p className="m-0">{step.error!.message}</p>
    </div>
  );
}

export function LogViewer({ lines }: { lines: string[] }) {
  return (
    <pre className="m-0 ml-6.5 max-h-72 overflow-auto rounded-lg bg-[#0b0b0e] p-3 font-mono text-[11px] leading-5 text-[#d6d6de]">
      {lines.map((l, i) => (
        <div key={i} className={l.includes("ERROR") ? "text-[#ff9b9b]" : l.includes("WARN") ? "text-[#f0c870]" : ""}>{l}</div>
      ))}
    </pre>
  );
}

/* ---------- Analyse : lancer / supprimer ---------- */

export function AnalysisActions({ sim, stacked = false }: { sim: Sim; stacked?: boolean }) {
  const [mode, setMode] = useState<"launch" | "delete" | null>(null);
  const failed = sim.steps.find((s) => s.status === "failed");
  return (
    <div className={`flex flex-wrap gap-2 ${stacked ? "flex-col" : ""}`}>
      {sim.running ? (
        <button type="button" className={buttonClass()} onClick={sim.cancel}>
          <Ban className="size-4" /> Annuler la suite
        </button>
      ) : (
        <>
          {failed && (
            <button type="button" className={buttonClass("primary")} onClick={() => sim.launch([failed.key])}>
              <RotateCcw className="size-4" /> Relancer « {failed.label} »
            </button>
          )}
          <button type="button" className={buttonClass(failed ? "secondary" : "primary")} onClick={() => setMode("launch")}>
            <Play className="size-4" /> Lancer l'analyse…
          </button>
        </>
      )}
      <button type="button" className={buttonClass("danger")} disabled={sim.running} onClick={() => setMode("delete")} title={sim.running ? "Annulez d'abord l'analyse en cours" : undefined}>
        <Trash2 className="size-4" /> Supprimer…
      </button>
      <StepDialog mode={mode} sim={sim} onClose={() => setMode(null)} />
    </div>
  );
}

function StepDialog({ mode, sim, onClose }: { mode: "launch" | "delete" | null; sim: Sim; onClose: () => void }) {
  const [sel, setSel] = useState<StepKey[]>([]);
  const isDelete = mode === "delete";
  const available = STEP_ORDER.filter((k) => {
    const st = sim.steps.find((s) => s.key === k)!.status;
    return isDelete ? st !== "absent" : true;
  });
  const toggle = (k: StepKey) => setSel((s) => (s.includes(k) ? s.filter((x) => x !== k) : [...s, k]));
  const plan = isDelete ? cascade(sel) : launchPlan(sim.steps, sel).plan;
  const extra = plan.filter((k) => !sel.includes(k));
  const validatedLost = isDelete && plan.includes("diarization") ? sim.voices.filter((v) => v.state === "validated") : [];
  const close = () => { setSel([]); onClose(); };
  return (
    <Modal open={mode !== null} onOpenChange={(o) => !o && close()} title={isDelete ? "Supprimer l'analyse" : "Lancer l'analyse"} closeLabel="Fermer">
      <div className="flex flex-col gap-3">
        <div className="flex gap-2">
          <button type="button" className={buttonClass()} onClick={() => setSel(available)}>{isDelete ? "Tout supprimer" : "Analyse complète"}</button>
          {!isDelete && (
            <button type="button" className={buttonClass()} onClick={() => setSel(STEP_ORDER.filter((k) => sim.steps.find((s) => s.key === k)!.status !== "done"))}>
              Seulement ce qui manque
            </button>
          )}
        </div>
        <ul className="m-0 flex list-none flex-col gap-1 p-0">
          {available.map((k) => {
            const st = sim.steps.find((s) => s.key === k)!;
            const implied = extra.includes(k);
            return (
              <li key={k}>
                <label className={`flex items-center gap-3 rounded-lg border px-3 py-2 ${sel.includes(k) || implied ? (isDelete ? "border-bad-fg/40 bg-bad-bg" : "border-accent bg-accent-soft") : "border-line"}`}>
                  <input type="checkbox" checked={sel.includes(k) || implied} disabled={implied} onChange={() => toggle(k)} />
                  <span className="font-medium">{STEP_LABELS[k]}</span>
                  <span className="ml-auto text-xs text-mute">{implied ? (isDelete ? "supprimée par cascade" : "ajoutée automatiquement") : statusText[st.status]}</span>
                </label>
              </li>
            );
          })}
        </ul>
        {isDelete && plan.length > 0 && (
          <div className="rounded-lg border border-bad-fg/30 bg-bad-bg p-3 text-sm text-bad-fg">
            <p className="m-0 font-medium">Sera supprimé définitivement :</p>
            <ul className="mt-1 mb-0 pl-5">
              {plan.includes("transcription") && <li>la transcription ({sim.segments.length} segments) et ses corrections</li>}
              {plan.includes("indexing") && <li>l'index de recherche de cette vidéo</li>}
              {plan.includes("diarization") && <li>les {sim.voices.length} voix détectées et leurs identifications</li>}
              {!plan.includes("diarization") && plan.includes("matching") && <li>les identifications des voix (les voix restent)</li>}
              {validatedLost.length > 0 && <li><strong>{validatedLost.length} voix validée(s)</strong> : {validatedLost.map((v) => v.person).join(", ")} perdent les échantillons de cette vidéo dans leur Référence</li>}
            </ul>
            <p className="mt-2 mb-0 text-xs">Les Personnes sont conservées. Réservé au Super-admin.</p>
          </div>
        )}
        <div className="flex justify-end gap-2">
          <button type="button" className={buttonClass()} onClick={close}>Annuler</button>
          <button
            type="button"
            disabled={plan.length === 0}
            className={buttonClass(isDelete ? "danger" : "primary")}
            onClick={() => { if (isDelete) sim.remove(plan); else sim.launch(plan); close(); }}
          >
            {isDelete ? `Supprimer ${plan.length} étape(s)` : `Lancer ${plan.length} étape(s)`}
          </button>
        </div>
      </div>
    </Modal>
  );
}

/* ---------- Divers ---------- */

export function Empty({ children }: { children: ReactNode }) {
  return <p className="m-0 p-6 text-center text-mute">{children}</p>;
}

function IconBtn({ label, onClick, children }: { label: string; onClick: () => void; children: ReactNode }) {
  return (
    <button type="button" aria-label={label} title={label} onClick={onClick} className="flex size-8 items-center justify-center rounded-md text-mute hover:bg-side hover:text-ink">
      {children}
    </button>
  );
}

export const tabBtn = (active: boolean) =>
  `-mb-px inline-flex h-10 shrink-0 items-center gap-2 border-b-2 px-3 font-medium whitespace-nowrap ${active ? "border-accent text-ink" : "border-transparent text-mute hover:text-ink"}`;

export function usePlayer() {
  const [playhead, setPlayhead] = useState(25);
  const [playing, setPlaying] = useState(false);
  return { playhead, playing, onToggle: () => setPlaying((p) => !p), seek: (s: number) => { setPlayhead(s); setPlaying(true); } };
}
