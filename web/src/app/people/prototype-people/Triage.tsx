// PROTOTYPE — écran « À traiter » d'une vidéo : distinguer les voix une par une, avec la forme d'onde.
import { ArrowLeft, ArrowRight, Check, CheckCircle2, Dices, EyeOff, ExternalLink, Pause, Play, SkipForward, UserPlus, UserX, Users } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { buttonClass } from "../../components/buttons";
import type { Voice } from "../../library/prototype-fiche/data";
import { fmt } from "../../library/prototype-fiche/parts";
import { useFicheSim } from "../../library/prototype-fiche/useFicheSim";
import { VoiceDot, voiceColor, Waveform } from "../../library/prototype-fiche/waveform";
import { ficheVoices, people, todo, type TodoVideo } from "./data";

const CONFIDENCE: Record<string, number> = { "Julie Martin": 87, "Claire Dubois": 41, "Thomas Roux": 58, "Hugo Bernard": 55 };
const toResolve = (v: Voice) => ["suggested", "ambiguous", "unknown"].includes(v.state);

export function Triage({ video, onBack, onNext }: { video: TodoVideo; onBack: () => void; onNext: () => void }) {
  const sim = useFicheSim();
  const queue = useMemo(() => sim.voices.filter((v) => toResolve(v) || v.state === "validated" || v.state === "random" || v.state === "ignored"), [sim.voices]);
  const pending = sim.voices.filter(toResolve);
  const [currentId, setCurrentId] = useState<number | null>(() => sim.voices.find(toResolve)?.id ?? null);
  const [playhead, setPlayhead] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [resolved, setResolved] = useState<number[]>([]);
  const current = sim.voices.find((v) => v.id === currentId) ?? null;
  const total = resolved.length + pending.length;

  // Lecture factice : la tête avance tant qu'on « écoute ».
  useEffect(() => {
    if (!playing) return;
    const t = setInterval(() => setPlayhead((p) => p + 0.25), 250);
    return () => clearInterval(t);
  }, [playing]);

  const advance = (after: Voice) => {
    setResolved((r) => (r.includes(after.id) ? r : [...r, after.id]));
    const next = sim.voices.find((v) => v.id !== after.id && toResolve(v) && !resolved.includes(v.id));
    setCurrentId(next?.id ?? null);
    setPlaying(false);
  };
  const act = (fn: () => void, v: Voice) => { fn(); advance(v); };
  const listen = (s: number) => { setPlayhead(s); setPlaying(true); };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!current || (e.target as HTMLElement).closest("input, textarea")) return;
      const a = sim.voiceActions;
      if (e.key === "o" && current.person) act(() => a.validate(current, current.person!), current);
      if (e.key === "n" && current.person) a.reject(current);
      if (e.key === "r") act(() => a.random(current), current);
      if (e.key === "i") act(() => a.ignore(current), current);
      if (e.key === " ") { e.preventDefault(); setPlaying((p) => !p); }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const segs = current ? sim.segments.filter((s) => s.voiceId === current.id) : [];
  const done = current === null;

  return (
    <section className="flex flex-col gap-5 px-4 py-6 md:px-7">
      {/* En-tête */}
      <div className="flex flex-wrap items-center gap-3">
        <button type="button" onClick={onBack} className="inline-flex items-center gap-1 text-mute hover:text-ink"><ArrowLeft className="size-4" /> À traiter</button>
        <span className="h-4 w-px bg-line" />
        <div className="min-w-0 flex-1">
          <h1 className="m-0 truncate text-lg font-semibold">{video.title}</h1>
          <p className="m-0 text-xs text-mute">{video.folder} · {video.date} · {video.duration}</p>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <span className="text-mute">{resolved.length} / {total} voix</span>
          <div className="flex gap-1">
            {Array.from({ length: total }, (_, i) => <span key={i} className={`h-1.5 w-6 rounded-full ${i < resolved.length ? "bg-ok-fg" : "bg-line"}`} />)}
          </div>
        </div>
        <a href={ficheVoices(video.id)} className="inline-flex items-center gap-1 text-sm text-accent-ink hover:underline">Fiche complète <ExternalLink className="size-3.5" /></a>
      </div>

      {/* Forme d'onde de toute la conversation */}
      <div className="flex flex-col gap-3 rounded-2xl border border-line bg-side p-4">
        <div className="flex items-center gap-3">
          <button type="button" onClick={() => setPlaying((p) => !p)} className="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent-solid text-accent-fg" aria-label={playing ? "Pause" : "Lecture"}>
            {playing ? <Pause className="size-4" /> : <Play className="size-4 translate-x-px" />}
          </button>
          <span className="text-sm text-mute tabular-nums">{fmt(playhead)}</span>
          <div className="ml-auto flex flex-wrap gap-3 text-xs">
            {sim.voices.map((v) => (
              <button key={v.id} type="button" onClick={() => setCurrentId(v.id)} className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 ${currentId === v.id ? "bg-surface shadow-sm ring-1 ring-line" : ""}`}>
                <VoiceDot id={v.id} ignored={v.state === "ignored"} />
                {v.state === "validated" ? v.person : v.state === "random" ? "Random" : v.state === "ignored" ? "ignorée" : v.label.toLowerCase()}
              </button>
            ))}
          </div>
        </div>
        <Waveform segments={sim.segments} voices={sim.voices} focus={current?.id} playhead={playhead} onSeek={listen} lanes height={96} bars={240} />
      </div>

      {done ? (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-line p-10 text-center">
          <CheckCircle2 className="size-10 text-ok-fg" />
          <p className="m-0 text-lg font-semibold">Toutes les voix de cette vidéo sont traitées</p>
          <p className="m-0 text-mute">{todo.length - 1} vidéos restent à traiter.</p>
          <button type="button" className={buttonClass("primary")} onClick={onNext}>Vidéo suivante <ArrowRight className="size-4" /></button>
        </div>
      ) : (
        <div className="grid gap-5 lg:grid-cols-[260px_minmax(0,1fr)]">
          {/* File des voix */}
          <ol className="m-0 flex list-none flex-col gap-1 p-0">
            {queue.map((v) => {
              const isDone = resolved.includes(v.id) || !toResolve(v);
              return (
                <li key={v.id}>
                  <button type="button" onClick={() => setCurrentId(v.id)} className={`flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left ${currentId === v.id ? "bg-accent-soft" : "hover:bg-side"}`}>
                    <VoiceDot id={v.id} ignored={v.state === "ignored"} />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm font-medium">{v.state === "validated" ? v.person : v.label.toLowerCase()}</span>
                      <span className="block text-xs text-mute">{fmt(v.speakingS)} de parole</span>
                    </span>
                    {isDone ? <Check className="size-4 text-ok-fg" /> : <span className="size-2 rounded-full bg-warn-fg" />}
                  </button>
                </li>
              );
            })}
          </ol>

          {/* Voix en cours */}
          {current && (
            <div className="flex flex-col gap-5 rounded-2xl border border-line p-5" style={{ borderTop: `4px solid ${voiceColor(current.id)}` }}>
              <div className="flex flex-wrap items-baseline gap-3">
                <h2 className="m-0 text-xl font-semibold">Qui parle ici ?</h2>
                <span className="font-mono text-sm text-mute">{current.label.toLowerCase()}</span>
                <span className="text-sm text-mute">· {current.turns} interventions · {fmt(current.speakingS)}</span>
              </div>

              <ul className="m-0 flex list-none flex-col gap-2 p-0">
                {segs.slice(0, 4).map((s) => {
                  const on = playing && playhead >= s.start && playhead < s.end;
                  return (
                    <li key={s.id} className={`flex items-center gap-3 rounded-xl border px-3 py-2 ${on ? "border-accent bg-accent-soft" : "border-line"}`}>
                      <button type="button" onClick={() => (on ? setPlaying(false) : listen(s.start))} className="flex size-9 shrink-0 items-center justify-center rounded-full border border-line-strong hover:bg-side" aria-label="Écouter l'extrait">
                        {on ? <Pause className="size-4" /> : <Play className="size-4 translate-x-px" />}
                      </button>
                      <div className="w-28 shrink-0"><Waveform segments={sim.segments} voices={sim.voices} focus={current.id} from={s.start} to={s.end} height={28} bars={40} /></div>
                      <p className="m-0 min-w-0 flex-1 text-sm">« {s.text} »</p>
                      <span className="text-xs text-mute tabular-nums">{fmt(s.start)}</span>
                    </li>
                  );
                })}
              </ul>

              {current.person && current.state === "suggested" && (
                <div className="flex flex-wrap items-center gap-4 rounded-xl bg-accent-soft p-4">
                  <span className="flex size-12 items-center justify-center rounded-full bg-avatar-bg font-semibold text-accent-ink">{current.person.split(" ").map((p) => p[0]).join("")}</span>
                  <div className="min-w-0 flex-1">
                    <p className="m-0 text-xs text-accent-ink">Ça ressemble à</p>
                    <p className="m-0 text-lg font-semibold">{current.person} <span className="text-sm font-normal text-mute">· {CONFIDENCE[current.person] ?? 70} % de ressemblance</span></p>
                  </div>
                  <button type="button" className={buttonClass("primary")} onClick={() => act(() => sim.voiceActions.validate(current, current.person!), current)}><Check className="size-4" /> Oui <kbd className="ml-1 text-[10px] opacity-70">O</kbd></button>
                  <button type="button" className={buttonClass()} onClick={() => sim.voiceActions.reject(current)}><UserX className="size-4" /> Non <kbd className="ml-1 text-[10px] opacity-70">N</kbd></button>
                </div>
              )}

              {current.state !== "suggested" && (
                <div className="flex flex-col gap-2">
                  <p className="m-0 text-sm font-medium">{current.candidates?.length ? "Hésite entre" : "Choisir une personne"}</p>
                  <div className="flex flex-wrap gap-2">
                    {(current.candidates ?? []).map((c) => (
                      <button key={c} type="button" onClick={() => act(() => sim.voiceActions.validate(current, c), current)} className="inline-flex items-center gap-2 rounded-full border border-accent bg-accent-soft px-3 py-1.5 text-sm text-accent-ink hover:brightness-95">
                        {c} <span className="text-xs opacity-70">{CONFIDENCE[c] ?? 50} %</span>
                      </button>
                    ))}
                    {people.filter((p) => !current.candidates?.includes(p.name) && !current.refused.includes(p.name)).slice(0, 4).map((p) => (
                      <button key={p.id} type="button" onClick={() => act(() => sim.voiceActions.validate(current, p.name), current)} className="rounded-full border border-line px-3 py-1.5 text-sm hover:bg-side">{p.name}</button>
                    ))}
                    <button type="button" className="inline-flex items-center gap-1 rounded-full border border-dashed border-line-strong px-3 py-1.5 text-sm text-mute hover:text-ink"><UserPlus className="size-4" /> Nouvelle personne</button>
                  </div>
                  {current.refused.length > 0 && <p className="m-0 text-xs text-mute">Déjà refusé : {current.refused.join(", ")}</p>}
                </div>
              )}

              {current.group && (
                <p className="m-0 flex items-center gap-2 rounded-xl border border-line px-3 py-2 text-sm"><Users className="size-4 text-mute" /> Cette voix est aussi entendue ailleurs ({current.group}) : la nommer ici la nommera partout.</p>
              )}

              <div className="flex flex-wrap items-center gap-2 border-t border-line pt-4">
                <button type="button" className={buttonClass()} onClick={() => act(() => sim.voiceActions.random(current), current)} title="Vraie voix, volontairement non identifiée"><Dices className="size-4" /> Random <kbd className="text-[10px] opacity-60">R</kbd></button>
                <button type="button" className={buttonClass()} onClick={() => act(() => sim.voiceActions.ignore(current), current)} title="Fausse détection : ne pas compter"><EyeOff className="size-4" /> Fausse détection <kbd className="text-[10px] opacity-60">I</kbd></button>
                <button type="button" className="ml-auto inline-flex items-center gap-1 text-sm text-mute hover:text-ink" onClick={() => setCurrentId(sim.voices.find((v) => v.id !== current.id && toResolve(v))?.id ?? null)}>Passer <SkipForward className="size-4" /></button>
              </div>
              <p className="m-0 text-xs text-mute">Raccourcis : Espace lecture · O oui · N non · R random · I fausse détection</p>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
