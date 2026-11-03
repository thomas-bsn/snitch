// PROTOTYPE — /prototype/library?variant=A|B|C et /prototype/import?variant=A|B|C. Jamais fusionné dans main.
import {
  ChevronLeft,
  ChevronRight,
  File,
  Film,
  Folder,
  FolderLock,
  FolderPlus,
  HardDrive,
  Library,
  Loader2,
  Play,
  Upload,
  Users,
} from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import { useSearchParams } from "react-router";
import { buttonClass } from "../../components/buttons";
import { Brand } from "../../components/Logo";
import { Modal } from "../../components/Modal";
import { at, countVideos, ficheFor, root, STATE_LABEL, STATE_TONE, type Node, type VideoState } from "./libraryData";

type Folder = Extract<Node, { kind: "folder" }>;
type Video = Extract<Node, { kind: "video" }>;

function StateTag({ s }: { s: VideoState }) {
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ${STATE_TONE[s]}`}>
      {s === "running" && <Loader2 className="size-3 animate-spin" />}
      {STATE_LABEL[s]}
    </span>
  );
}

/** Clic sur une vidéo : indexation à la demande si besoin, puis la fiche. */
function useOpenVideo() {
  const [indexing, setIndexing] = useState<string | null>(null);
  const open = (n: Video) => {
    if (n.state !== "unindexed") return void (window.location.href = ficheFor(n));
    setIndexing(n.name);
    setTimeout(() => (window.location.href = ficheFor(n)), 1400);
  };
  return { indexing, open };
}

function IndexingToast({ name }: { name: string | null }) {
  if (!name) return null;
  return (
    <div className="fixed top-4 right-4 z-40 flex items-center gap-2 rounded-lg border border-line bg-surface px-4 py-3 shadow-pop">
      <Loader2 className="size-4 animate-spin text-accent" /> Indexation de « {name} » (lecture et empreinte du fichier, sans transcription)…
    </div>
  );
}

function Crumbs({ path, onGo }: { path: string[]; onGo: (p: string[]) => void }) {
  return (
    <nav className="flex flex-wrap items-center gap-1 text-sm">
      <button type="button" onClick={() => onGo([])} className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 hover:bg-side"><HardDrive className="size-4" /> videos</button>
      {path.map((p, i) => (
        <span key={i} className="flex items-center gap-1">
          <ChevronRight className="size-3.5 text-mute" />
          <button type="button" onClick={() => onGo(path.slice(0, i + 1))} className="rounded px-1.5 py-0.5 hover:bg-side">{p}</button>
        </span>
      ))}
    </nav>
  );
}

const FILTERS: [string, (n: Video) => boolean][] = [
  ["Toutes", () => true],
  ["Non indexées", (n) => n.state === "unindexed"],
  ["Non analysées", (n) => n.state === "indexed"],
  ["Analysées", (n) => n.state === "analysed" || n.state === "transcribed"],
  ["Échec / en cours", (n) => n.state === "failed" || n.state === "running"],
];

function FolderIcon({ f }: { f: Folder }) {
  return f.private ? <FolderLock className="size-4 text-mute" /> : <Folder className="size-4 text-accent" />;
}

/* A — Arbre à gauche, contenu du dossier en tableau à droite. */
function LibA() {
  const [path, setPath] = useState<string[]>(["2026", "octobre"]);
  const [filter, setFilter] = useState(0);
  const { indexing, open } = useOpenVideo();
  const cur = at(path);
  const folders = cur.children.filter((c): c is Folder => c.kind === "folder");
  const videos = cur.children.filter((c): c is Video => c.kind === "video").filter(FILTERS[filter][1]);
  function Tree({ node, p, depth }: { node: Folder; p: string[]; depth: number }) {
    const selected = p.join("/") === path.join("/");
    const openNode = path.join("/").startsWith(p.join("/"));
    return (
      <li>
        <button type="button" onClick={() => setPath(p)} style={{ paddingLeft: 8 + depth * 14 }} className={`flex w-full items-center gap-2 rounded-md py-1.5 pr-2 text-left text-sm ${selected ? "bg-accent-soft text-accent-ink" : "hover:bg-side"}`}>
          {depth === 0 ? <HardDrive className="size-4" /> : <FolderIcon f={node} />}
          <span className="min-w-0 flex-1 truncate">{node.name}</span>
          <span className="text-xs text-mute">{countVideos(node).total}</span>
        </button>
        {openNode && (
          <ul className="m-0 list-none p-0">
            {node.children.filter((c): c is Folder => c.kind === "folder").map((c) => <Tree key={c.name} node={c} p={[...p, c.name]} depth={depth + 1} />)}
          </ul>
        )}
      </li>
    );
  }
  return (
    <section className="grid min-h-[80vh] gap-0 md:grid-cols-[260px_minmax(0,1fr)]">
      <aside className="border-r border-line p-3"><ul className="m-0 list-none p-0"><Tree node={root as Folder} p={[]} depth={0} /></ul></aside>
      <div className="flex min-w-0 flex-col gap-3 p-4 md:px-6">
        <div className="flex flex-wrap items-center gap-3">
          <Crumbs path={path} onGo={setPath} />
          <span className="ml-auto text-sm text-mute">{countVideos(cur).total} vidéos · {countVideos(cur).unindexed} jamais indexées</span>
        </div>
        <div className="flex flex-wrap gap-2">
          {FILTERS.map(([l], i) => <button key={l} type="button" onClick={() => setFilter(i)} className={`rounded-full border px-3 py-1 text-sm ${filter === i ? "border-accent bg-accent-soft text-accent-ink" : "border-line text-mute hover:text-ink"}`}>{l}</button>)}
        </div>
        <table className="w-full border-collapse text-sm">
          <thead><tr className="border-b border-line text-left text-xs text-mute"><th className="py-2 font-medium">Nom</th><th className="font-medium">État</th><th className="font-medium">Durée</th><th className="font-medium">Ajoutée</th><th className="font-medium">Taille</th></tr></thead>
          <tbody>
            {folders.map((f) => (
              <tr key={f.name} onClick={() => setPath([...path, f.name])} className="cursor-pointer border-b border-line hover:bg-side">
                <td className="py-2"><span className="flex items-center gap-2 font-medium"><FolderIcon f={f} /> {f.name}{f.private && <span className="text-xs text-mute">privé</span>}</span></td>
                <td colSpan={4} className="text-xs text-mute">{countVideos(f).total} vidéos · {countVideos(f).done} analysées</td>
              </tr>
            ))}
            {videos.map((n) => (
              <tr key={n.name} onClick={() => open(n)} className="cursor-pointer border-b border-line hover:bg-side">
                <td className="py-2"><span className="flex items-center gap-2"><Film className="size-4 text-mute" /><span className="min-w-0"><span className="block">{n.title ?? n.name}</span>{n.title && <span className="block text-xs text-mute">{n.name}</span>}</span>{n.todo ? <span className="rounded-full bg-warn-bg px-1.5 text-xs text-warn-fg">{n.todo} voix à traiter</span> : null}</span></td>
                <td><StateTag s={n.state} /></td>
                <td className="text-mute tabular-nums">{n.duration ?? "—"}</td>
                <td className="text-mute">{n.date}</td>
                <td className="text-mute">{n.size}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <IndexingToast name={indexing} />
    </section>
  );
}

/* B — Colonnes à la Finder : chaque niveau ouvre une colonne, la dernière montre l'aperçu. */
function LibB() {
  const [path, setPath] = useState<string[]>(["2026", "novembre"]);
  const [sel, setSel] = useState<Video | null>(null);
  const { indexing, open } = useOpenVideo();
  const cols: { folder: Folder; p: string[] }[] = [{ folder: root as Folder, p: [] }];
  path.forEach((_, i) => cols.push({ folder: at(path.slice(0, i + 1)), p: path.slice(0, i + 1) }));
  return (
    <section className="flex flex-col gap-3 p-4 md:px-6">
      <Crumbs path={path} onGo={(p) => { setPath(p); setSel(null); }} />
      <div className="flex min-h-[70vh] overflow-x-auto rounded-xl border border-line">
        {cols.map(({ folder, p }, ci) => (
          <ul key={ci} className="m-0 w-64 shrink-0 list-none overflow-y-auto border-r border-line p-1">
            {folder.children.map((c) => {
              const active = c.kind === "folder" ? path[ci] === c.name : sel?.name === c.name;
              return (
                <li key={c.name}>
                  <button
                    type="button"
                    onClick={() => (c.kind === "folder" ? (setPath([...p, c.name]), setSel(null)) : (setPath(p), setSel(c)))}
                    className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm ${active ? "bg-accent-soft text-accent-ink" : "hover:bg-side"}`}
                  >
                    {c.kind === "folder" ? <FolderIcon f={c} /> : <Film className="size-4 text-mute" />}
                    <span className="min-w-0 flex-1 truncate">{c.kind === "video" ? (c.title ?? c.name) : c.name}</span>
                    {c.kind === "folder" ? <ChevronRight className="size-3.5 text-mute" /> : <span className={`size-2 rounded-full ${c.state === "unindexed" ? "border border-mute" : c.state === "failed" ? "bg-bad-fg" : c.state === "running" ? "bg-accent" : c.state === "indexed" ? "bg-line-strong" : "bg-ok-fg"}`} />}
                  </button>
                </li>
              );
            })}
          </ul>
        ))}
        {sel && (
          <div className="flex w-80 shrink-0 flex-col gap-3 p-4">
            <div className="flex aspect-video items-center justify-center rounded-lg bg-[#0b0b0e] text-white/60"><Play className="size-6" /></div>
            <p className="m-0 font-semibold break-words">{sel.title ?? sel.name}</p>
            <StateTag s={sel.state} />
            <dl className="m-0 grid grid-cols-[max-content_1fr] gap-x-3 gap-y-1 text-sm">
              <dt className="text-mute">Fichier</dt><dd className="m-0 break-all">{sel.name}</dd>
              <dt className="text-mute">Durée</dt><dd className="m-0">{sel.duration ?? "inconnue (non indexée)"}</dd>
              <dt className="text-mute">Taille</dt><dd className="m-0">{sel.size}</dd>
              <dt className="text-mute">Ajoutée</dt><dd className="m-0">{sel.date}</dd>
            </dl>
            <button type="button" className={buttonClass("primary")} onClick={() => open(sel)}>Ouvrir la fiche</button>
          </div>
        )}
      </div>
      <IndexingToast name={indexing} />
    </section>
  );
}

/* C — Vignettes : dossiers en tuiles, vidéos en cartes avec leur état en surimpression. */
function LibC() {
  const [path, setPath] = useState<string[]>(["2026", "octobre"]);
  const { indexing, open } = useOpenVideo();
  const cur = at(path);
  const folders = cur.children.filter((c): c is Folder => c.kind === "folder");
  const videos = cur.children.filter((c): c is Video => c.kind === "video");
  return (
    <section className="flex flex-col gap-4 p-4 md:px-7">
      <div className="flex flex-wrap items-center gap-3">
        {path.length > 0 && <button type="button" onClick={() => setPath(path.slice(0, -1))} className={buttonClass()}><ChevronLeft className="size-4" /></button>}
        <Crumbs path={path} onGo={setPath} />
      </div>
      {folders.length > 0 && (
        <ul className="m-0 grid list-none grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-2 p-0">
          {folders.map((f) => {
            const c = countVideos(f);
            return (
              <li key={f.name}>
                <button type="button" onClick={() => setPath([...path, f.name])} className="flex w-full items-center gap-3 rounded-xl border border-line px-3 py-3 text-left hover:border-accent">
                  <FolderIcon f={f} />
                  <span className="min-w-0 flex-1"><span className="block truncate font-medium">{f.name}</span><span className="block text-xs text-mute">{c.total} vidéos · {c.unindexed} non indexées</span></span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
      <ul className="m-0 grid list-none grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-4 p-0">
        {videos.map((n) => (
          <li key={n.name}>
            <button type="button" onClick={() => open(n)} className="flex w-full flex-col gap-2 text-left">
              <span className={`relative flex aspect-video items-center justify-center rounded-lg ${n.state === "unindexed" ? "border-2 border-dashed border-line-strong bg-side text-mute" : "bg-[#1b1b22] text-white/50"}`}>
                {n.state === "unindexed" ? <File className="size-6" /> : <Play className="size-6" />}
                <span className="absolute top-2 left-2"><StateTag s={n.state} /></span>
                {n.duration && <span className="absolute right-2 bottom-2 rounded bg-black/60 px-1.5 text-xs text-white">{n.duration}</span>}
                {n.todo ? <span className="absolute top-2 right-2 rounded-full bg-warn-bg px-1.5 text-xs text-warn-fg">{n.todo} voix</span> : null}
              </span>
              <span className="truncate font-medium">{n.title ?? n.name}</span>
              <span className="text-xs text-mute">{n.date} · {n.size}</span>
            </button>
          </li>
        ))}
      </ul>
      <IndexingToast name={indexing} />
    </section>
  );
}

/* ---------- Import ---------- */

type Role = "user" | "admin";

/** Sélecteur dans /videos : fichiers multiples ou dossier entier, ou choix d'un dossier de destination. */
function PickerDialog({ open, onClose, mode, role }: { open: boolean; onClose: () => void; mode: "sources" | "destination"; role: Role }) {
  const [path, setPath] = useState<string[]>(["2026", "novembre"]);
  const [picked, setPicked] = useState<string[]>([]);
  const [creating, setCreating] = useState(false);
  const [requested, setRequested] = useState<string | null>(null);
  const [name, setName] = useState("");
  const cur = at(path);
  const key = (n: string) => [...path, n].join("/");
  const toggle = (k: string) => setPicked((p) => (p.includes(k) ? p.filter((x) => x !== k) : [...p, k]));
  const count = picked.reduce((n, k) => {
    const parts = k.split("/");
    const node = at(parts.slice(0, -1)).children.find((c) => c.name === parts.at(-1))!;
    return n + countVideos(node).total;
  }, 0);
  const close = () => { setPicked([]); setCreating(false); setRequested(null); onClose(); };
  return (
    <Modal open={open} onOpenChange={(o) => !o && close()} title={mode === "sources" ? "Choisir dans la bibliothèque" : "Dossier de destination"} closeLabel="Fermer" variant="wide">
      <Crumbs path={path} onGo={setPath} />
      <ul className="m-0 flex max-h-[45vh] list-none flex-col overflow-y-auto rounded-lg border border-line p-1">
        {cur.children.filter((c) => mode === "sources" || c.kind === "folder").map((c) => (
          <li key={c.name} className="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-side">
            {mode === "sources" && <input type="checkbox" checked={picked.includes(key(c.name))} onChange={() => toggle(key(c.name))} aria-label={`Sélectionner ${c.name}`} />}
            {c.kind === "folder" ? <FolderIcon f={c} /> : <Film className="size-4 text-mute" />}
            <button type="button" disabled={c.kind === "video"} onClick={() => c.kind === "folder" && setPath([...path, c.name])} className="min-w-0 flex-1 truncate text-left disabled:cursor-default">
              {c.name}
              {c.kind === "folder" && <span className="ml-2 text-xs text-mute">{countVideos(c).total} vidéos</span>}
            </button>
            {c.kind === "video" && <StateTag s={c.state} />}
            {c.kind === "folder" && <ChevronRight className="size-4 text-mute" />}
          </li>
        ))}
        {requested && (
          <li className="flex items-center gap-2 rounded-md px-2 py-1.5 text-mute"><Folder className="size-4" /> {requested} <span className="rounded-full bg-warn-bg px-2 text-xs text-warn-fg">demande en attente</span></li>
        )}
      </ul>
      {creating ? (
        <div className="flex flex-col gap-2 rounded-lg border border-line p-3">
          <label className="flex flex-col gap-1 text-sm"><span className="font-medium">Nouveau dossier dans /{["videos", ...path].join("/")}</span><input value={name} onChange={(e) => setName(e.target.value)} className="h-9 rounded-lg border border-line-strong bg-surface px-3" autoFocus /></label>
          {role === "user" && <p className="m-0 text-xs text-mute">Un administrateur doit valider la création. Vous serez notifié ; en attendant, vous ne pouvez pas y téléverser.</p>}
          <div className="flex justify-end gap-2">
            <button type="button" className={buttonClass()} onClick={() => setCreating(false)}>Annuler</button>
            <button type="button" className={buttonClass("primary")} disabled={!name} onClick={() => { if (role === "admin") { cur.children.unshift({ kind: "folder", name, children: [] }); } else setRequested(name); setCreating(false); setName(""); }}>
              {role === "admin" ? "Créer" : "Envoyer la demande"}
            </button>
          </div>
        </div>
      ) : (
        <button type="button" className={`${buttonClass()} w-fit`} onClick={() => setCreating(true)}><FolderPlus className="size-4" /> Nouveau dossier</button>
      )}
      <div className="flex items-center justify-end gap-2 border-t border-line pt-3">
        {mode === "sources" ? (
          <>
            <span className="mr-auto text-sm text-mute">{picked.length ? `${picked.length} élément(s) · ${count} vidéo(s), sous-dossiers compris` : "Cochez des vidéos ou des dossiers"}</span>
            <button type="button" className={buttonClass()} onClick={close}>Annuler</button>
            <button type="button" className={buttonClass("primary")} disabled={!count} onClick={close}>Analyser {count} vidéo(s)</button>
          </>
        ) : (
          <>
            <span className="mr-auto text-sm text-mute">Destination : /{["videos", ...path].join("/")}</span>
            <button type="button" className={buttonClass("primary")} onClick={close}>Choisir ce dossier</button>
          </>
        )}
      </div>
    </Modal>
  );
}

function DropZone({ onPickDest, dest }: { onPickDest: () => void; dest: string }) {
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col items-center gap-2 rounded-xl border-2 border-dashed border-line-strong px-6 py-10 text-center">
        <Upload className="size-6 text-mute" />
        <p className="m-0 font-medium">Glissez des vidéos ici</p>
        <button type="button" className={buttonClass()}>Parcourir mon ordinateur</button>
      </div>
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <span className="text-mute">Ranger dans</span>
        <code className="rounded bg-side px-2 py-1">{dest}</code>
        <button type="button" className="text-accent-ink hover:underline" onClick={onPickDest}>Changer…</button>
      </div>
      <p className="m-0 text-xs text-mute">L'analyse complète démarre automatiquement après l'envoi (réglage d'administration).</p>
    </div>
  );
}

function ImpA({ role }: { role: Role }) {
  const [mode, setMode] = useState<null | "local" | "upload">(null);
  const [picker, setPicker] = useState<null | "sources" | "destination">(null);
  return (
    <section className="flex max-w-3xl flex-col gap-5 p-4 md:px-7 md:py-6">
      <h1 className="m-0 text-[22px] font-semibold">Importer</h1>
      {mode === null && (
        <div className="grid gap-4 sm:grid-cols-2">
          <Choice icon={<HardDrive className="size-6" />} title="Déjà sur le serveur" body="La vidéo est déjà dans /videos. Choisissez des fichiers ou un dossier entier à analyser." onClick={() => { setMode("local"); setPicker("sources"); }} />
          <Choice icon={<Upload className="size-6" />} title="Depuis mon ordinateur" body="Téléversez des vidéos et choisissez le dossier où les ranger." onClick={() => setMode("upload")} />
        </div>
      )}
      {mode === "upload" && <><button type="button" className="w-fit text-sm text-mute hover:text-ink" onClick={() => setMode(null)}>‹ Changer de mode</button><DropZone dest="/videos/2026/novembre" onPickDest={() => setPicker("destination")} /></>}
      {mode === "local" && <button type="button" className="w-fit text-sm text-mute hover:text-ink" onClick={() => setMode(null)}>‹ Changer de mode</button>}
      <PickerDialog open={picker !== null} mode={picker ?? "sources"} role={role} onClose={() => setPicker(null)} />
    </section>
  );
}

function Choice({ icon, title, body, onClick }: { icon: ReactNode; title: string; body: string; onClick: () => void }) {
  return (
    <button type="button" onClick={onClick} className="flex flex-col gap-2 rounded-xl border border-line p-5 text-left hover:border-accent hover:bg-accent-soft/40">
      <span className="text-accent">{icon}</span>
      <span className="text-base font-semibold">{title}</span>
      <span className="text-sm text-mute">{body}</span>
    </button>
  );
}

function ImpB({ role }: { role: Role }) {
  const [tab, setTab] = useState<"upload" | "local">("upload");
  const [picker, setPicker] = useState<null | "sources" | "destination">(null);
  const tb = (a: boolean) => `-mb-px inline-flex h-10 items-center gap-2 border-b-2 px-3 font-medium ${a ? "border-accent text-ink" : "border-transparent text-mute hover:text-ink"}`;
  return (
    <section className="flex max-w-3xl flex-col gap-4 p-4 md:px-7 md:py-6">
      <h1 className="m-0 text-[22px] font-semibold">Importer</h1>
      <div className="flex gap-1 border-b border-line">
        <button type="button" className={tb(tab === "upload")} onClick={() => setTab("upload")}><Upload className="size-4" /> Téléverser</button>
        <button type="button" className={tb(tab === "local")} onClick={() => setTab("local")}><HardDrive className="size-4" /> Depuis la bibliothèque</button>
      </div>
      {tab === "upload" ? (
        <DropZone dest="/videos/2026/novembre" onPickDest={() => setPicker("destination")} />
      ) : (
        <div className="flex flex-col gap-3 rounded-xl border border-line p-5">
          <p className="m-0">Analysez des vidéos déjà présentes dans <code>/videos</code> : des fichiers, plusieurs, ou un dossier entier avec ses sous-dossiers.</p>
          <p className="m-0 text-sm text-mute">8 vidéos de la bibliothèque n'ont jamais été indexées.</p>
          <button type="button" className={`${buttonClass("primary")} w-fit`} onClick={() => setPicker("sources")}>Choisir dans /videos…</button>
        </div>
      )}
      <PickerDialog open={picker !== null} mode={picker ?? "sources"} role={role} onClose={() => setPicker(null)} />
    </section>
  );
}

function ImpC({ role }: { role: Role }) {
  const [picker, setPicker] = useState<null | "sources" | "destination">(null);
  return (
    <section className="flex max-w-3xl flex-col gap-4 p-4 md:px-7 md:py-6">
      <h1 className="m-0 text-[22px] font-semibold">Importer</h1>
      <DropZone dest="/videos/2026/novembre" onPickDest={() => setPicker("destination")} />
      <div className="flex items-center gap-3 text-sm text-mute"><span className="h-px flex-1 bg-line" />ou<span className="h-px flex-1 bg-line" /></div>
      <button type="button" onClick={() => setPicker("sources")} className="flex items-center gap-3 rounded-xl border border-line p-4 text-left hover:border-accent">
        <HardDrive className="size-5 text-accent" />
        <span className="min-w-0 flex-1"><span className="block font-medium">La vidéo est déjà sur le serveur</span><span className="block text-sm text-mute">Choisir des fichiers ou un dossier dans /videos</span></span>
        <ChevronRight className="size-4 text-mute" />
      </button>
      <PickerDialog open={picker !== null} mode={picker ?? "sources"} role={role} onClose={() => setPicker(null)} />
    </section>
  );
}

/* ---------- cadre ---------- */

const SCREENS = {
  library: [
    { key: "A", name: "Arbre + tableau", C: LibA },
    { key: "B", name: "Colonnes (Finder)", C: LibB },
    { key: "C", name: "Vignettes", C: LibC },
  ],
  import: [
    { key: "A", name: "Deux choix", C: ImpA },
    { key: "B", name: "Onglets", C: ImpB },
    { key: "C", name: "Téléverser d'abord", C: ImpC },
  ],
};

export function LibraryPrototype({ screen }: { screen: "library" | "import" }) {
  const [params, setParams] = useSearchParams();
  const [role, setRole] = useState<Role>("user");
  const list = SCREENS[screen];
  const idx = Math.max(0, list.findIndex((v) => v.key === (params.get("variant") ?? "A")));
  const go = (d: number) => setParams({ variant: list[(idx + d + list.length) % list.length].key }, { replace: true });
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.target as HTMLElement).closest("input, textarea, [contenteditable]")) return;
      if (e.key === "ArrowLeft") go(-1);
      if (e.key === "ArrowRight") go(1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });
  const V = list[idx].C as (p: { role: Role }) => ReactNode;
  return (
    <div className="flex min-h-dvh bg-canvas text-ink">
      <aside className="hidden w-[220px] shrink-0 flex-col gap-1 border-r border-line bg-side px-3 py-5 md:flex">
        <div className="px-2 pb-5"><Brand /></div>
        {([[Library, "Bibliothèque", screen === "library"], [Users, "Personnes", false], [Upload, "Importer", screen === "import"]] as const).map(([I, label, cur]) => (
          <span key={label} className={`flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium ${cur ? "bg-accent-soft text-accent-ink" : "text-nav"}`}><I className="size-4" /> {label}</span>
        ))}
      </aside>
      <main className="min-w-0 flex-1 pb-24"><V role={role} /></main>
      <div className="fixed bottom-4 left-1/2 z-30 flex -translate-x-1/2 items-center gap-3 rounded-full bg-black px-4 py-2 text-xs whitespace-nowrap text-white shadow-2xl">
        <button type="button" onClick={() => go(-1)} aria-label="Variante précédente"><ChevronLeft className="size-4" /></button>
        <span className="font-semibold">{list[idx].key} · {list[idx].name}</span>
        <button type="button" onClick={() => go(1)} aria-label="Variante suivante"><ChevronRight className="size-4" /></button>
        {screen === "import" && (
          <>
            <span className="h-4 w-px bg-white/30" />
            <label className="flex items-center gap-1">Rôle
              <select value={role} onChange={(e) => setRole(e.target.value as Role)} className="rounded bg-white/15 px-1 py-0.5"><option value="user">utilisateur</option><option value="admin">admin</option></select>
            </label>
          </>
        )}
      </div>
    </div>
  );
}
