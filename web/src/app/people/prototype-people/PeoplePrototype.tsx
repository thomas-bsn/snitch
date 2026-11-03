// PROTOTYPE — /prototype/people?variant=A|B|C. Hors session, sans serveur. Jamais fusionné dans main.
import {
  ArrowRight,
  ChevronLeft,
  ChevronRight,
  CircleHelp,
  GitMerge,
  Library,
  Play,
  Search,
  Split,
  Trash2,
  Upload,
  UserCheck,
  UserPlus,
  Users,
  UsersRound,
} from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import { useSearchParams } from "react-router";
import { buttonClass } from "../../components/buttons";
import { Brand } from "../../components/Logo";
import { Modal } from "../../components/Modal";
import { appearances, ficheVoices, groups, people, todo, total, type Group, type Person, type TodoVideo } from "./data";

const totalVoices = todo.reduce((n, v) => n + total(v), 0);
const fmtS = (s: number) => `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;

/* ---------- morceaux partagés ---------- */

function Counts({ v }: { v: TodoVideo }) {
  return (
    <span className="flex flex-wrap gap-1.5">
      {v.suggestions > 0 && <Badge tone="accent">{v.suggestions} à confirmer</Badge>}
      {v.ambiguous > 0 && <Badge tone="warn">{v.ambiguous} ambiguë{v.ambiguous > 1 ? "s" : ""}</Badge>}
      {v.unknown > 0 && <Badge tone="idle">{v.unknown} inconnue{v.unknown > 1 ? "s" : ""}</Badge>}
      {v.groups.length > 0 && <Badge tone="idle">groupe {v.groups.join(", ")}</Badge>}
    </span>
  );
}

function Badge({ tone, children }: { tone: "accent" | "warn" | "idle" | "ok"; children: ReactNode }) {
  const c = { accent: "bg-accent-soft text-accent-ink", warn: "bg-warn-bg text-warn-fg", idle: "bg-side text-mute", ok: "bg-ok-bg text-ok-fg" }[tone];
  return <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${c}`}>{children}</span>;
}

function Avatar({ name, size = "size-9" }: { name: string; size?: string }) {
  const initials = name.split(" ").map((p) => p[0]).join("").slice(0, 2);
  return <span className={`flex ${size} shrink-0 items-center justify-center rounded-full bg-avatar-bg text-xs font-semibold text-accent-ink`}>{initials}</span>;
}

function PersonRecord({ p, onBack }: { p: Person; onBack?: () => void }) {
  return (
    <div className="flex flex-col gap-4">
      {onBack && (
        <button type="button" onClick={onBack} className="inline-flex w-fit items-center gap-1 text-mute hover:text-ink">
          <ChevronLeft className="size-4" /> Personnes
        </button>
      )}
      <div className="flex flex-wrap items-center gap-3">
        <Avatar name={p.name} size="size-12" />
        <div className="min-w-0 flex-1">
          <h2 className="m-0 text-lg font-semibold">{p.name}</h2>
          <p className="m-0 text-sm text-mute">{p.account ? `Compte lié : @${p.account}` : "Aucun compte lié"} · entendue pour la dernière fois le {p.lastHeard}</p>
        </div>
        <div className="flex gap-2">
          <button type="button" className={buttonClass()}><GitMerge className="size-4" /> Fusionner</button>
          <button type="button" className={buttonClass()}><Split className="size-4" /> Séparer</button>
          <button type="button" className={buttonClass("danger")}><Trash2 className="size-4" /></button>
        </div>
      </div>
      <dl className="m-0 grid grid-cols-2 gap-3 sm:grid-cols-4">
        {[["Vidéos", String(p.videos)], ["Temps de parole", `${Math.floor(p.speakingMin / 60)} h ${p.speakingMin % 60} min`], ["Échantillons de Référence", String(p.samples)], ["Qualité de la Référence", p.referenceQuality]].map(([k, v]) => (
          <div key={k} className="rounded-lg border border-line p-3">
            <dt className="text-xs text-mute">{k}</dt>
            <dd className={`m-0 mt-1 text-base font-semibold ${v === "faible" || v === "aucune" ? "text-warn-fg" : ""}`}>{v}</dd>
          </div>
        ))}
      </dl>
      {p.referenceQuality !== "bonne" && (
        <p className="m-0 rounded-lg bg-warn-bg px-3 py-2 text-sm text-warn-fg">Référence {p.referenceQuality} : la reconnaissance automatique sera peu fiable. Validez cette personne dans 2 ou 3 vidéos de plus.</p>
      )}
      <section>
        <h3 className="m-0 mb-2 text-sm font-semibold">Entendue dans</h3>
        <ul className="m-0 flex list-none flex-col p-0">
          {appearances.map((a) => (
            <li key={a.title} className="flex items-center gap-3 border-b border-line py-2 last:border-b-0">
              <button type="button" aria-label="Écouter" className="flex size-8 items-center justify-center rounded-md text-mute hover:bg-side"><Play className="size-4" /></button>
              <a href={ficheVoices(40)} className="min-w-0 flex-1 truncate hover:underline">{a.title}</a>
              <span className="text-xs text-mute">{a.date} · {a.speaking}</span>
              <Badge tone={a.state === "validée" ? "ok" : "accent"}>{a.state}</Badge>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}

function GroupNaming({ g, onClose }: { g: Group | null; onClose: () => void }) {
  return (
    <Modal open={g !== null} onOpenChange={(o) => !o && onClose()} title={g ? `Voix inconnue ${g.id}` : ""} closeLabel="Fermer" variant="side">
      {g && (
        <div className="flex flex-col gap-3">
          <p className="m-0 text-mute">La même voix semble parler dans {g.videos.length} vidéos ({fmtS(g.speakingS)} au total). Écoutez, retirez celles qui ne collent pas, puis nommez-la.</p>
          <ul className="m-0 flex list-none flex-col gap-1 p-0">
            {g.videos.map((t) => (
              <li key={t} className="flex items-center gap-2 rounded-lg border border-line px-3 py-2">
                <button type="button" aria-label="Écouter" className="flex size-8 items-center justify-center rounded-md hover:bg-side"><Play className="size-4" /></button>
                <span className="min-w-0 flex-1 truncate">{t}</span>
                <button type="button" className="text-xs text-mute hover:text-ink">Ce n'est pas la même</button>
              </li>
            ))}
          </ul>
          {g.guess && <p className="m-0 rounded-lg bg-accent-soft px-3 py-2 text-sm text-accent-ink">Ressemble à {g.guess}.</p>}
          <label className="flex flex-col gap-1 text-sm">
            <span className="font-medium">Qui est-ce ?</span>
            <input defaultValue={g.guess ?? ""} placeholder="Rechercher une personne ou saisir un nouveau nom" className="h-9 rounded-lg border border-line-strong bg-surface px-3" />
          </label>
          <div className="flex flex-wrap justify-end gap-2">
            <button type="button" className={buttonClass()} onClick={onClose}>Random</button>
            <button type="button" className={buttonClass()} onClick={onClose}>Ignorer</button>
            <button type="button" className={buttonClass("primary")} onClick={onClose}><UserCheck className="size-4" /> Nommer dans {g.videos.length} vidéos</button>
          </div>
        </div>
      )}
    </Modal>
  );
}

function GroupList({ onOpen, compact = false }: { onOpen: (g: Group) => void; compact?: boolean }) {
  return (
    <ul className="m-0 flex list-none flex-col gap-2 p-0">
      {groups.map((g) => (
        <li key={g.id}>
          <button type="button" onClick={() => onOpen(g)} className="flex w-full items-center gap-3 rounded-lg border border-line px-3 py-2 text-left hover:bg-side">
            <span className="flex size-9 items-center justify-center rounded-full bg-side"><CircleHelp className="size-4 text-mute" /></span>
            <span className="min-w-0 flex-1">
              <span className="block font-medium">Voix inconnue {g.id}</span>
              <span className="block truncate text-xs text-mute">{compact ? `${g.videos.length} vidéos` : `Entendue dans ${g.videos.length} vidéos : ${g.videos.join(", ")}`}</span>
            </span>
            {g.guess && !compact && <Badge tone="accent">ressemble à {g.guess}</Badge>}
            <ChevronRight className="size-4 text-mute" />
          </button>
        </li>
      ))}
    </ul>
  );
}

function SearchBox() {
  return (
    <label className="flex h-9 items-center gap-2 rounded-lg border border-line-strong bg-surface px-3">
      <Search className="size-4 text-mute" />
      <input placeholder="Rechercher une personne" className="min-w-0 flex-1 bg-transparent outline-none" />
    </label>
  );
}

const tabBtn = (active: boolean) =>
  `-mb-px inline-flex h-10 shrink-0 items-center gap-2 border-b-2 px-3 font-medium whitespace-nowrap ${active ? "border-accent text-ink" : "border-transparent text-mute hover:text-ink"}`;

/* ---------- A : onglets, « À traiter » en premier ---------- */

function VariantA() {
  const [tab, setTab] = useState<"todo" | "persons" | "groups">("todo");
  const [filter, setFilter] = useState<"all" | "confirm" | "unknown" | "groups">("all");
  const [person, setPerson] = useState<Person>(people[0]);
  const [group, setGroup] = useState<Group | null>(null);
  const rows = todo.filter((v) => filter === "all" || (filter === "confirm" ? v.suggestions > 0 : filter === "unknown" ? v.unknown + v.ambiguous > 0 : v.groups.length > 0));
  return (
    <section className="flex flex-col gap-4 px-4 py-6 md:px-7">
      <h1 className="m-0 text-[22px] font-semibold">Personnes</h1>
      <div className="flex gap-1 border-b border-line">
        <button type="button" className={tabBtn(tab === "todo")} onClick={() => setTab("todo")}>À traiter <span className="rounded-full bg-warn-bg px-2 text-xs text-warn-fg">{totalVoices}</span></button>
        <button type="button" className={tabBtn(tab === "persons")} onClick={() => setTab("persons")}>Personnes <span className="text-xs text-mute">{people.length}</span></button>
        <button type="button" className={tabBtn(tab === "groups")} onClick={() => setTab("groups")}>Voix inconnues <span className="text-xs text-mute">{groups.length}</span></button>
      </div>
      {tab === "todo" && (
        <>
          <div className="flex flex-wrap items-center gap-2">
            {([["all", "Tout"], ["confirm", "À confirmer"], ["unknown", "Inconnues / ambiguës"], ["groups", "Avec un groupe"]] as const).map(([k, l]) => (
              <button key={k} type="button" onClick={() => setFilter(k)} className={`rounded-full border px-3 py-1 text-sm ${filter === k ? "border-accent bg-accent-soft text-accent-ink" : "border-line text-mute hover:text-ink"}`}>{l}</button>
            ))}
            <span className="ml-auto text-sm text-mute">{totalVoices} voix dans {todo.length} vidéos</span>
          </div>
          <ul className="m-0 flex list-none flex-col rounded-xl border border-line p-0">
            {rows.map((v) => (
              <li key={v.id} className="flex flex-wrap items-center gap-3 border-b border-line px-4 py-3 last:border-b-0">
                <span className="min-w-0 flex-1">
                  <a href={ficheVoices(v.id)} className="block font-medium hover:underline">{v.title}</a>
                  <span className="block text-xs text-mute">{v.folder} · {v.date} · {v.duration}{v.recentlyAnalysed ? " · analysée aujourd'hui" : ""}</span>
                </span>
                <Counts v={v} />
                <a href={ficheVoices(v.id)} className={buttonClass("primary")}>Traiter {total(v)} <ArrowRight className="size-4" /></a>
              </li>
            ))}
          </ul>
        </>
      )}
      {tab === "persons" && (
        <div className="grid gap-5 lg:grid-cols-[300px_minmax(0,1fr)]">
          <div className="flex flex-col gap-2">
            <SearchBox />
            <button type="button" className={buttonClass()}><UserPlus className="size-4" /> Nouvelle personne</button>
            <ul className="m-0 flex list-none flex-col p-0">
              {people.map((p) => (
                <li key={p.id}>
                  <button type="button" onClick={() => setPerson(p)} className={`flex w-full items-center gap-3 rounded-lg px-2 py-2 text-left ${person.id === p.id ? "bg-accent-soft" : "hover:bg-side"}`}>
                    <Avatar name={p.name} />
                    <span className="min-w-0 flex-1"><span className="block font-medium">{p.name}</span><span className="block text-xs text-mute">{p.videos} vidéos · {p.samples} échantillons</span></span>
                  </button>
                </li>
              ))}
            </ul>
          </div>
          <div className="rounded-xl border border-line p-4"><PersonRecord p={person} /></div>
        </div>
      )}
      {tab === "groups" && (
        <>
          <p className="m-0 text-mute">Voix entendues dans plusieurs vidéos sans qu'on sache qui c'est. Les nommer identifie la personne partout d'un coup.</p>
          <GroupList onOpen={setGroup} />
        </>
      )}
      <GroupNaming g={group} onClose={() => setGroup(null)} />
    </section>
  );
}

/* ---------- B : file de travail, une vidéo à la fois ---------- */

function VariantB() {
  const [i, setI] = useState(0);
  const [person, setPerson] = useState<Person | null>(null);
  const [group, setGroup] = useState<Group | null>(null);
  const v = todo[i % todo.length];
  return (
    <section className="flex flex-col gap-5 px-4 py-6 md:px-7">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <h1 className="m-0 text-[22px] font-semibold">Personnes</h1>
        <SearchBox />
      </div>
      <div className="flex flex-col gap-4 rounded-2xl border border-line bg-side p-5">
        <div className="flex flex-wrap items-center gap-3">
          <span className="text-sm text-mute">Il reste <strong className="text-ink">{totalVoices} voix</strong> à traiter dans <strong className="text-ink">{todo.length} vidéos</strong></span>
          <div className="h-1.5 min-w-40 flex-1 rounded-full bg-line"><div className="h-1.5 w-1/4 rounded-full bg-accent" /></div>
        </div>
        <div className="flex flex-wrap items-center gap-4 rounded-xl border border-line bg-surface p-4">
          <div className="flex aspect-video w-48 items-center justify-center rounded-lg bg-[#0b0b0e] text-white/70"><Play className="size-6" /></div>
          <div className="min-w-0 flex-1">
            <p className="m-0 text-xs font-medium tracking-wide text-mute uppercase">Prochaine vidéo</p>
            <p className="m-0 mt-1 text-lg font-semibold">{v.title}</p>
            <p className="m-0 text-sm text-mute">{v.folder} · {v.date} · {v.duration}</p>
            <div className="mt-2"><Counts v={v} /></div>
          </div>
          <div className="flex flex-col gap-2">
            <a href={ficheVoices(v.id)} className={buttonClass("primary")}>Commencer <ArrowRight className="size-4" /></a>
            <button type="button" className={buttonClass()} onClick={() => setI(i + 1)}>Passer</button>
          </div>
        </div>
        <details>
          <summary className="cursor-pointer text-sm text-accent-ink">Voir toute la file ({todo.length})</summary>
          <ul className="m-0 mt-2 flex list-none flex-col p-0">
            {todo.map((x) => (
              <li key={x.id} className="flex items-center gap-3 border-b border-line py-2 last:border-b-0">
                <a href={ficheVoices(x.id)} className="min-w-0 flex-1 truncate hover:underline">{x.title}</a>
                <span className="text-xs text-mute">{total(x)} voix</span>
              </li>
            ))}
          </ul>
        </details>
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        <section className="flex flex-col gap-2">
          <div className="flex items-center justify-between"><h2 className="m-0 text-base font-semibold">Personnes ({people.length})</h2><button type="button" className={buttonClass()}><UserPlus className="size-4" /> Nouvelle</button></div>
          <ul className="m-0 flex list-none flex-col rounded-xl border border-line p-0">
            {people.map((p) => (
              <li key={p.id}>
                <button type="button" onClick={() => setPerson(p)} className="flex w-full items-center gap-3 border-b border-line px-3 py-2 text-left last:border-b-0 hover:bg-side">
                  <Avatar name={p.name} />
                  <span className="min-w-0 flex-1 font-medium">{p.name}</span>
                  <span className="text-xs text-mute">{p.videos} vidéos</span>
                  {p.referenceQuality !== "bonne" && <Badge tone="warn">Référence {p.referenceQuality}</Badge>}
                </button>
              </li>
            ))}
          </ul>
        </section>
        <section className="flex flex-col gap-2">
          <h2 className="m-0 text-base font-semibold">Voix inconnues à nommer ({groups.length})</h2>
          <GroupList onOpen={setGroup} />
        </section>
      </div>
      <Modal open={person !== null} onOpenChange={(o) => !o && setPerson(null)} title="Fiche personne" closeLabel="Fermer" variant="wide">
        {person && <PersonRecord p={person} />}
      </Modal>
      <GroupNaming g={group} onClose={() => setGroup(null)} />
    </section>
  );
}

/* ---------- C : annuaire en grille, rail « À traiter » toujours visible ---------- */

function VariantC() {
  const [person, setPerson] = useState<Person | null>(null);
  const [group, setGroup] = useState<Group | null>(null);
  return (
    <section className="grid gap-6 px-4 py-6 md:px-7 xl:grid-cols-[minmax(0,1fr)_340px]">
      <div className="flex min-w-0 flex-col gap-4">
        {person ? (
          <PersonRecord p={person} onBack={() => setPerson(null)} />
        ) : (
          <>
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="m-0 flex-1 text-[22px] font-semibold">Personnes</h1>
              <SearchBox />
              <button type="button" className={buttonClass()}><UserPlus className="size-4" /> Nouvelle</button>
            </div>
            <ul className="m-0 grid list-none grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-3 p-0">
              {people.map((p) => (
                <li key={p.id}>
                  <button type="button" onClick={() => setPerson(p)} className="flex w-full flex-col items-center gap-2 rounded-xl border border-line p-4 text-center hover:border-accent">
                    <Avatar name={p.name} size="size-14" />
                    <span className="font-semibold">{p.name}</span>
                    <span className="text-xs text-mute">{p.videos} vidéos · {Math.round(p.speakingMin / 60)} h de parole</span>
                    {p.referenceQuality !== "bonne" ? <Badge tone="warn">Référence {p.referenceQuality}</Badge> : <Badge tone="ok">Référence bonne</Badge>}
                  </button>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
      <aside className="flex flex-col gap-4 self-start xl:sticky xl:top-4">
        <div className="flex flex-col gap-2 rounded-xl border border-line p-4">
          <h2 className="m-0 flex items-center gap-2 text-base font-semibold"><UsersRound className="size-4" /> À traiter <Badge tone="warn">{totalVoices} voix</Badge></h2>
          <ul className="m-0 flex list-none flex-col p-0">
            {todo.map((v) => (
              <li key={v.id}>
                <a href={ficheVoices(v.id)} className="flex items-center gap-2 border-b border-line py-2 last:border-b-0 hover:bg-side">
                  <span className="min-w-0 flex-1"><span className="block truncate text-sm font-medium">{v.title}</span><span className="block text-xs text-mute">{v.date}{v.groups.length ? ` · groupe ${v.groups.join(", ")}` : ""}</span></span>
                  <Badge tone="accent">{total(v)}</Badge>
                </a>
              </li>
            ))}
          </ul>
        </div>
        <div className="flex flex-col gap-2 rounded-xl border border-line p-4">
          <h2 className="m-0 text-base font-semibold">Voix inconnues récurrentes</h2>
          <GroupList onOpen={setGroup} compact />
        </div>
      </aside>
      <GroupNaming g={group} onClose={() => setGroup(null)} />
    </section>
  );
}

/* ---------- cadre + barre de variantes ---------- */

const VARIANTS = [
  { key: "A", name: "Onglets, À traiter d'abord", C: VariantA },
  { key: "B", name: "File de travail", C: VariantB },
  { key: "C", name: "Annuaire + rail À traiter", C: VariantC },
];

export function PeoplePrototype() {
  const [params, setParams] = useSearchParams();
  const idx = Math.max(0, VARIANTS.findIndex((v) => v.key === (params.get("variant") ?? "A")));
  const go = (d: number) => setParams({ variant: VARIANTS[(idx + d + VARIANTS.length) % VARIANTS.length].key }, { replace: true });
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.target as HTMLElement).closest("input, textarea, [contenteditable]")) return;
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
        {([[Library, "Bibliothèque", false], [Users, "Personnes", true], [Upload, "Importer", false]] as const).map(([I, label, cur]) => (
          <span key={label} className={`flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium ${cur ? "bg-accent-soft text-accent-ink" : "text-nav"}`}>
            <I className="size-4" /> {label}
            {label === "Personnes" && <span className="ml-auto rounded-full bg-warn-bg px-1.5 text-xs text-warn-fg">{totalVoices}</span>}
          </span>
        ))}
      </aside>
      <main className="min-w-0 flex-1 pb-24"><V /></main>
      <div className="fixed bottom-4 left-1/2 z-30 flex -translate-x-1/2 items-center gap-3 rounded-full bg-black px-4 py-2 text-xs whitespace-nowrap text-white shadow-2xl">
        <button type="button" onClick={() => go(-1)} aria-label="Variante précédente"><ChevronLeft className="size-4" /></button>
        <span className="font-semibold">{VARIANTS[idx].key} · {VARIANTS[idx].name}</span>
        <button type="button" onClick={() => go(1)} aria-label="Variante suivante"><ChevronRight className="size-4" /></button>
      </div>
    </div>
  );
}
