import { useCallback, useEffect, useRef, useState } from "react";
import { Folder, FileVideo, ArrowLeft } from "lucide-react";
import { useI18n } from "../i18n";
import { useSession } from "../session";
import { isAdministrator } from "../roles";
import { buttonClass } from "../components/buttons";
import { RootScanCounts, RootScanProgress } from "./RootScanReport";
import {
  cancelRootScan,
  fetchCurrentRootScan,
  previewRootScan,
  startRootScan,
  type RootScan,
  type RootScanPreview,
} from "./rootScanApi";

type Root = {
  id: number;
  path: string;
  health?: string;
  last_scan_at?: string;
};
type Entry = { path: string; directory: boolean; size_bytes: number };
type Queued = {
  relative_path: string;
  status: string;
  error_code?: string;
  scan_requested?: boolean;
};

/** The queue lists this many files at most: a scan of the whole Racine can request thousands. */
const QUEUE_SHOWN = 200;

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

/** One managed video root: choose it, browse it, request a scan, create subfolders. */
export function VideoStorage({ settings = false }: { settings?: boolean }) {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const csrf = session?.csrf_token ?? "";
  const [root, setRoot] = useState<Root | null>(null);
  const [worker, setWorker] = useState(false);
  const [path, setPath] = useState("");
  const [open, setOpen] = useState(settings);
  const [directory, setDirectory] = useState("");
  const [entries, setEntries] = useState<Entry[]>([]);
  const [filter, setFilter] = useState("");
  const [selected, setSelected] = useState<string[]>([]);
  const [preview, setPreview] = useState<string[] | null>(null);
  const [previewSelection, setPreviewSelection] = useState<string[]>([]);
  const [queue, setQueue] = useState<Queued[]>([]);
  const [rootScan, setRootScan] = useState<RootScan | null>(null);
  const [rootPreview, setRootPreview] = useState<RootScanPreview | null>(null);
  const [folder, setFolder] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [notice, setNotice] = useState("");
  // The periodic refresh must not overwrite a path being edited.
  const loadedRoot = useRef("");
  const admin = !!user && isAdministrator(user);
  const visibleEntries = entries.filter((entry) =>
    (entry.path.split("/").pop() ?? "")
      .toLocaleLowerCase()
      .includes(filter.trim().toLocaleLowerCase()),
  );

  const load = useCallback(async () => {
    const data = await request("/api/admin/video-root", csrf);
    setWorker(!!data.worker_available);
    setRoot(data.root);
    if (data.root) {
      setPath((current) =>
        loadedRoot.current === current ? data.root.path : current,
      );
      loadedRoot.current = data.root.path;
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
      const current = await fetchCurrentRootScan();
      if (current.ok) setRootScan(current.value);
    } else {
      setEntries([]);
      setQueue([]);
      setRootScan(null);
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
      setRootPreview(null);
      setPreview(result.paths);
      setPreviewSelection(paths);
    });
  }

  // « Scanner toute la racine » (#404): counts first, then a background scan on the server.
  async function previewRoot() {
    await action(async () => {
      const result = await previewRootScan();
      if (!result.ok) throw new Error(result.code);
      setPreview(null);
      setRootPreview(result.value);
    });
  }

  async function startRoot() {
    await action(async () => {
      const result = await startRootScan(csrf);
      if (!result.ok) throw new Error(result.code);
      setRootPreview(null);
      setRootScan(result.value);
      setNotice(t("videoStorage.rootScan.started"));
    });
  }

  function cancelRoot() {
    if (!rootScan || !window.confirm(t("videoStorage.rootScan.confirmCancel")))
      return;
    void action(async () => {
      const result = await cancelRootScan(rootScan.id, csrf);
      if (!result.ok) throw new Error(result.code);
      setRootScan(result.value);
    });
  }

  function saveRoot() {
    if (
      root &&
      path !== root.path &&
      !window.confirm(t("videoStorage.confirmRootChange"))
    ) {
      return;
    }
    void action(async () => {
      const changed = root !== null && path !== root.path;
      await request("/api/admin/video-root", csrf, { path }, "PUT");
      setDirectory("");
      setNotice(
        t(changed ? "videoStorage.rootChanged" : "videoStorage.rootSaved"),
      );
    });
  }

  function queueStatus(item: Queued) {
    if (item.status === "missing") return t("videoStorage.status.missing");
    if (item.error_code) return t("videoStorage.status.attention");
    // A ready file may still wait for a Worker or for the transcription profile.
    if (item.scan_requested)
      return worker
        ? t("videoStorage.status.pending")
        : t("videoStorage.status.waiting");
    if (item.status === "ready") return t("videoStorage.status.indexed");
    if (item.status === "waiting_stability")
      return t("videoStorage.status.waiting");
    return t("videoStorage.status.attention");
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
            {t("videoStorage.open")}
          </button>
        </div>
      )}
      {open && (
        <>
          <h2 className="text-lg font-semibold">{t("videoStorage.title")}</h2>
          {(settings || !root) && (
            <form
              className="flex max-w-xl flex-col gap-2"
              onSubmit={(e) => {
                e.preventDefault();
                saveRoot();
              }}
            >
              <label htmlFor="video-root-path">
                {t("videoStorage.rootLabel")}
              </label>
              <input
                id="video-root-path"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder="/videos"
                required
                className="h-11 rounded-lg border border-line-strong bg-surface px-3"
              />
              <p className="text-sm text-mute">{t("videoStorage.rootHint")}</p>
              <div>
                <button className={buttonClass("primary")} disabled={busy}>
                  {t("videoStorage.saveRoot")}
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
              <p className="text-sm" role="status">
                {worker
                  ? t("videoStorage.workerOnline")
                  : t("videoStorage.workerOffline")}
              </p>
              <p className="text-sm text-mute">
                {t("videoStorage.lastScan", {
                  date: root.last_scan_at
                    ? new Date(root.last_scan_at).toLocaleString()
                    : t("videoStorage.never"),
                })}
              </p>
              <div className="flex flex-wrap gap-2">
                {directory && (
                  <button
                    className={buttonClass("secondary")}
                    onClick={() => {
                      setDirectory(directory.split("/").slice(0, -1).join("/"));
                      setSelected([]);
                      setFilter("");
                    }}
                  >
                    <ArrowLeft size={16} />
                    {t("videoStorage.parent")}
                  </button>
                )}
                <button
                  disabled={busy}
                  className={buttonClass("secondary")}
                  onClick={() => void previewRoot()}
                >
                  {t("videoStorage.scanRoot")}
                </button>
                <button
                  disabled={busy || !selected.length}
                  className={buttonClass("primary")}
                  onClick={() => void scan(selected)}
                >
                  {t("videoStorage.scanSelection", { count: selected.length })}
                </button>
              </div>
              <p className="text-sm text-mute">{t("videoStorage.scanHint")}</p>
              {entries.length > 0 && (
                <label className="flex max-w-md flex-col gap-1 text-sm">
                  {t("videoStorage.filterLabel")}
                  <input
                    type="search"
                    value={filter}
                    onChange={(e) => setFilter(e.target.value)}
                    className="h-11 rounded-lg border border-line-strong bg-surface px-3"
                  />
                </label>
              )}
              <div
                role="region"
                aria-label={t("videoStorage.entriesLabel")}
                tabIndex={0}
                className="max-h-80 overflow-y-auto rounded-lg border border-line focus-visible:outline-2"
              >
                <ul className="divide-y divide-line">
                  {visibleEntries.map((entry) => (
                    <li
                      key={entry.path}
                      className="flex min-h-11 items-center gap-3 px-3"
                    >
                      <input
                        type="checkbox"
                        aria-label={t("videoStorage.select", {
                          path: entry.path,
                        })}
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
                            setFilter("");
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
                    <li className="p-3 text-mute">{t("videoStorage.empty")}</li>
                  )}
                  {entries.length > 0 && visibleEntries.length === 0 && (
                    <li className="p-3 text-mute">
                      {t("videoStorage.noMatches")}
                    </li>
                  )}
                </ul>
              </div>
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
                    setNotice(t("videoStorage.folderCreated"));
                  });
                }}
              >
                <label className="flex flex-col gap-1">
                  {t("videoStorage.newFolder")}
                  <input
                    value={folder}
                    onChange={(e) => setFolder(e.target.value)}
                    required
                    maxLength={255}
                    className="h-11 rounded-lg border border-line-strong bg-surface px-3"
                  />
                </label>
                <button className={buttonClass("secondary")} disabled={busy}>
                  {t("videoStorage.createFolder")}
                </button>
              </form>
              {rootPreview && (
                <div className="flex flex-col gap-3 rounded-lg border border-line p-4">
                  <h3 className="font-semibold">
                    {t("videoStorage.rootScan.confirmTitle")}
                  </h3>
                  <p className="text-sm">
                    {t("videoStorage.rootScan.confirmBody")}
                  </p>
                  <RootScanCounts
                    counts={{ ...rootPreview, queued: rootPreview.to_queue }}
                  />
                  <div className="flex gap-2">
                    <button
                      disabled={busy || rootPreview.to_queue === 0}
                      className={buttonClass("primary")}
                      onClick={() => void startRoot()}
                    >
                      {t("videoStorage.rootScan.confirm", {
                        count: rootPreview.to_queue,
                      })}
                    </button>
                    <button
                      className={buttonClass("secondary")}
                      onClick={() => setRootPreview(null)}
                    >
                      {t("videoStorage.cancel")}
                    </button>
                  </div>
                </div>
              )}
              {rootScan && (
                <RootScanProgress
                  run={rootScan}
                  busy={busy}
                  onCancel={cancelRoot}
                />
              )}
              {preview && (
                <div className="flex flex-col gap-3 rounded-lg border border-line p-4">
                  <h3 className="font-semibold">
                    {t("videoStorage.confirmTitle", { count: preview.length })}
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
                          setNotice(t("videoStorage.queued"));
                        })
                      }
                    >
                      {t("videoStorage.addToQueue")}
                    </button>
                    <button
                      className={buttonClass("secondary")}
                      onClick={() => setPreview(null)}
                    >
                      {t("videoStorage.cancel")}
                    </button>
                  </div>
                </div>
              )}
              {queue.length > 0 && (
                <>
                  <h3 className="font-semibold">
                    {t("videoStorage.queueTitle")}
                  </h3>
                  <ul className="max-h-72 overflow-auto divide-y divide-line text-sm">
                    {queue.slice(0, QUEUE_SHOWN).map((item) => (
                      <li
                        key={item.relative_path}
                        className="flex flex-wrap justify-between gap-2 py-2"
                      >
                        <span className="break-all">{item.relative_path}</span>
                        <span className="flex flex-col items-end">
                          <span>{queueStatus(item)}</span>
                          {item.error_code && (
                            <span className="text-xs text-danger">
                              {error(item.error_code)}
                            </span>
                          )}
                        </span>
                      </li>
                    ))}
                  </ul>
                  {queue.length > QUEUE_SHOWN && (
                    <p className="text-sm text-mute">
                      {t("videoStorage.queueMore", {
                        count: queue.length - QUEUE_SHOWN,
                      })}
                    </p>
                  )}
                </>
              )}
            </>
          )}
        </>
      )}
    </section>
  );
}
