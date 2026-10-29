import { useCallback, useEffect, useState } from "react";
import { Folder, FileVideo, ArrowLeft } from "lucide-react";
import { useI18n } from "../i18n";
import { useSession } from "../session";
import { isAdministrator } from "../roles";
import { buttonClass } from "../components/buttons";

type Root = { id: number; path: string };
type Entry = { path: string; directory: boolean; size_bytes: number };
type Queued = { relative_path: string; status: string; error_code?: string };
async function request(
  path: string,
  csrf: string,
  body?: unknown,
  method = "POST",
) {
  const response = await fetch(
    path,
    body === undefined
      ? undefined
      : {
          method,
          headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
          body: JSON.stringify(body),
        },
  );
  const data = await response.json();
  if (!response.ok) throw new Error(data.error ?? "unavailable");
  return data;
}
export function VideoStorage({ settings = false }: { settings?: boolean }) {
  const { language, error } = useI18n();
  const fr = language === "fr";
  const { user, session } = useSession();
  const csrf = session?.csrf_token ?? "";
  const [root, setRoot] = useState<Root | null>(null);
  const [path, setPath] = useState("");
  const [open, setOpen] = useState(settings);
  const [directory, setDirectory] = useState("");
  const [entries, setEntries] = useState<Entry[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [preview, setPreview] = useState<string[] | null>(null);
  const [previewSelection, setPreviewSelection] = useState<string[]>([]);
  const [queue, setQueue] = useState<Queued[]>([]);
  const [folder, setFolder] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [notice, setNotice] = useState("");
  const admin = !!user && isAdministrator(user);
  const load = useCallback(async () => {
    const data = await request("/api/admin/video-root", csrf);
    setRoot(data.root);
    if (data.root) {
      setPath(data.root.path);
      const browsing = await request(
        `/api/admin/video-root/browse?path=${encodeURIComponent(directory)}`,
        csrf,
      );
      setEntries(browsing.items);
      const files = await request(
        `/api/admin/sources/${data.root.id}/files`,
        csrf,
      );
      setQueue(files.items);
    }
  }, [csrf, directory]);
  useEffect(() => {
    if (!admin || !open) return;
    void load().catch((e) => setFailure(String(e.message)));
    const timer = setInterval(() => void load().catch(() => {}), 5000);
    return () => clearInterval(timer);
  }, [load, admin, open]);
  async function action(fn: () => Promise<void>) {
    setBusy(true);
    setFailure("");
    setNotice("");
    try {
      await fn();
      await load();
    } catch (e) {
      setFailure(e instanceof Error ? e.message : "unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function scan(paths: string[]) {
    await action(async () => {
      const result = await request("/api/admin/video-root/scan", csrf, {
        paths,
        confirm: false,
      });
      setPreview(result.paths);
      setPreviewSelection(paths);
    });
  }
  if (!admin) return null;
  return (
    <section className="flex flex-col gap-3">
      {!settings && (
        <div>
          <button
            className={buttonClass("secondary")}
            onClick={() => setOpen(!open)}
            aria-expanded={open}
          >
            {fr
              ? "Scanner des vidéos et gérer les dossiers"
              : "Scan videos and manage folders"}
          </button>
        </div>
      )}
      {open && (
        <>
          <h2 className="text-lg font-semibold">
            {fr ? "Dossier des vidéos" : "Video folder"}
          </h2>
          {(settings || !root) && (
            <form
              className="flex max-w-xl flex-col gap-2"
              onSubmit={(e) => {
                e.preventDefault();
                void action(async () => {
                  await request("/api/admin/video-root", csrf, { path }, "PUT");
                  setNotice(
                    fr
                      ? "Dossier enregistré. Aucun scan lancé."
                      : "Folder saved. No scan started.",
                  );
                });
              }}
            >
              <label htmlFor="video-root-path">
                {fr
                  ? "Dossier racine accessible à Snitch"
                  : "Root folder accessible to Snitch"}
              </label>
              <input
                id="video-root-path"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder="/videos"
                required
                className="h-11 rounded-lg border border-line-strong bg-surface px-3"
              />
              <p className="text-sm text-mute">
                {fr
                  ? "Le dossier doit déjà être monté dans le conteneur et accessible en lecture et écriture. Enregistrer ce chemin ne lance aucun scan."
                  : "The folder must already be mounted in the container with read and write access. Saving this path does not start a scan."}
              </p>
              <div>
                <button className={buttonClass("primary")} disabled={busy}>
                  {fr ? "Enregistrer le dossier" : "Save folder"}
                </button>
              </div>
            </form>
          )}
          {failure && (
            <p role="alert" className="text-danger">
              {error(failure)}
            </p>
          )}
          {notice && <p aria-live="polite">{notice}</p>}
          {root && (
            <>
              <p className="break-all text-sm text-mute">
                {root.path}
                {directory ? `/${directory}` : ""}
              </p>
              <div className="flex flex-wrap gap-2">
                {directory && (
                  <button
                    className={buttonClass("secondary")}
                    onClick={() => {
                      setDirectory(directory.split("/").slice(0, -1).join("/"));
                      setSelected([]);
                    }}
                  >
                    <ArrowLeft size={16} />
                    {fr ? "Dossier parent" : "Parent folder"}
                  </button>
                )}
                <button
                  disabled={busy}
                  className={buttonClass("secondary")}
                  onClick={() => void scan([""])}
                >
                  {fr ? "Scanner toute la racine" : "Scan entire root"}
                </button>
                <button
                  disabled={busy || !selected.length}
                  className={buttonClass("primary")}
                  onClick={() => void scan(selected)}
                >
                  {fr
                    ? "Préparer le scan de la sélection"
                    : "Prepare selection scan"}{" "}
                  ({selected.length})
                </button>
              </div>
              <p className="text-sm text-mute">
                {fr
                  ? "Sélectionner un dossier inclut ses sous-dossiers. Rien n’est analysé avant confirmation. Sans Worker disponible, les fichiers restent en attente."
                  : "Selecting a folder includes its subfolders. Nothing is analysed before confirmation. Files wait when no Worker is available."}
              </p>
              <ul className="divide-y divide-line rounded-lg border border-line">
                {entries.map((entry) => (
                  <li
                    key={entry.path}
                    className="flex min-h-11 items-center gap-3 px-3"
                  >
                    <input
                      type="checkbox"
                      aria-label={`${fr ? "Sélectionner" : "Select"} ${entry.path}`}
                      checked={selected.includes(entry.path)}
                      onChange={(e) =>
                        setSelected(
                          e.target.checked
                            ? [...selected, entry.path]
                            : selected.filter((p) => p !== entry.path),
                        )
                      }
                    />
                    {entry.directory ? (
                      <Folder size={18} />
                    ) : (
                      <FileVideo size={18} />
                    )}
                    {entry.directory ? (
                      <button
                        className="min-h-11 text-accent-ink"
                        onClick={() => {
                          setDirectory(entry.path);
                          setSelected([]);
                        }}
                      >
                        {entry.path.split("/").pop()}
                      </button>
                    ) : (
                      <span className="break-all">
                        {entry.path.split("/").pop()}
                      </span>
                    )}
                  </li>
                ))}
                {!entries.length && (
                  <li className="p-3 text-mute">
                    {fr
                      ? "Aucun dossier ou fichier vidéo ici."
                      : "No folders or video files here."}
                  </li>
                )}
              </ul>
              <form
                className="flex flex-wrap items-end gap-2"
                onSubmit={(e) => {
                  e.preventDefault();
                  void action(async () => {
                    await request("/api/admin/video-root/folders", csrf, {
                      parent: directory,
                      name: folder,
                    });
                    setFolder("");
                  });
                }}
              >
                <label className="flex flex-col gap-1">
                  {fr ? "Nouveau sous-dossier" : "New subfolder"}
                  <input
                    value={folder}
                    onChange={(e) => setFolder(e.target.value)}
                    required
                    maxLength={255}
                    className="h-11 rounded-lg border border-line-strong bg-surface px-3"
                  />
                </label>
                <button className={buttonClass("secondary")} disabled={busy}>
                  {fr ? "Créer le dossier" : "Create folder"}
                </button>
              </form>
              {preview && (
                <div className="flex flex-col gap-3 rounded-lg border border-line p-4">
                  <h3 className="font-semibold">
                    {fr
                      ? "Confirmer les fichiers à scanner"
                      : "Confirm files to scan"}{" "}
                    ({preview.length})
                  </h3>
                  <ul className="max-h-64 overflow-auto text-sm">
                    {preview.map((p) => (
                      <li key={p} className="break-all py-1">
                        {p}
                      </li>
                    ))}
                  </ul>
                  <div className="flex gap-2">
                    <button
                      disabled={busy || !preview.length}
                      className={buttonClass("primary")}
                      onClick={() =>
                        void action(async () => {
                          await request("/api/admin/video-root/scan", csrf, {
                            paths: previewSelection,
                            confirm: true,
                          });
                          setPreview(null);
                          setSelected([]);
                          setNotice(
                            fr
                              ? "Fichiers ajoutés à la file."
                              : "Files added to the queue.",
                          );
                        })
                      }
                    >
                      {fr ? "Ajouter à la file" : "Add to queue"}
                    </button>
                    <button
                      className={buttonClass("secondary")}
                      onClick={() => setPreview(null)}
                    >
                      {fr ? "Annuler" : "Cancel"}
                    </button>
                  </div>
                </div>
              )}
              {queue.length > 0 && (
                <>
                  <h3 className="font-semibold">
                    {fr ? "File des fichiers demandés" : "Requested files"}
                  </h3>
                  <ul className="max-h-72 overflow-auto divide-y divide-line text-sm">
                    {queue.map((item) => (
                      <li
                        key={item.relative_path}
                        className="flex flex-wrap justify-between gap-2 py-2"
                      >
                        <span className="break-all">{item.relative_path}</span>
                        <span>
                          {item.status === "ready"
                            ? fr
                              ? "Indexé"
                              : "Indexed"
                            : item.status === "waiting_stability"
                              ? fr
                                ? "En attente"
                                : "Waiting"
                              : fr
                                ? "À vérifier"
                                : "Needs attention"}
                        </span>
                      </li>
                    ))}
                  </ul>
                </>
              )}
            </>
          )}
        </>
      )}
    </section>
  );
}
