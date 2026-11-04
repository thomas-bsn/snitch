import { ChevronRight, Film, Folder, HardDrive, Lock } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { useSession } from "../session";

type Entry = {
  path: string;
  directory: boolean;
  size_bytes: number;
  private?: boolean;
  locked?: boolean;
};

const last = (path: string) => path.split("/").pop() ?? path;

/**
 * « Déjà sur le serveur » (#224): picks videos already in the Racine vidéo, in a dialog. A folder selects
 * its videos recursively; a Dossier privé shows a lock and cannot be selected (ADR 0026). The server
 * counts the selection as it changes, before anything is queued, and queues it only once the person
 * confirms « Analyser N vidéo(s) »; indexing and analysis then follow the settings of the library (Lot B).
 */
export function ServerSelection() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const csrf = session?.csrf_token ?? "";
  const [open, setOpen] = useState(true);
  const [rooted, setRooted] = useState<boolean>();
  const [directory, setDirectory] = useState("");
  const [entries, setEntries] = useState<Entry[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [counted, setCounted] = useState<number>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState<string>();

  const request = useCallback(
    async (url: string, init?: RequestInit) => {
      const response = await fetch(url, {
        ...init,
        headers: {
          ...(init?.body ? { "Content-Type": "application/json" } : {}),
          "X-CSRF-Token": csrf,
        },
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error ?? "unavailable");
      return data;
    },
    [csrf],
  );

  useEffect(() => {
    void request("/api/admin/video-root")
      .then((data) => setRooted(Boolean(data.root)))
      .catch(() => setRooted(false));
  }, [request]);

  useEffect(() => {
    if (!rooted) return;
    void request(
      `/api/admin/video-root/browse?path=${encodeURIComponent(directory)}`,
    )
      .then((data) => {
        setEntries(data.items as Entry[]);
        setFailure(undefined);
      })
      .catch((e: Error) => setFailure(e.message));
  }, [request, rooted, directory]);

  // The server counts the videos of the selection (folders recursively) whenever it changes.
  useEffect(() => {
    setCounted(undefined);
    if (selected.length === 0) return;
    let current = true;
    void request("/api/library/import/scan", {
      method: "POST",
      body: JSON.stringify({ paths: selected, confirm: false }),
    })
      .then((data) => {
        if (current) setCounted((data.paths as string[]).length);
      })
      .catch((e: Error) => {
        if (current) setFailure(e.message);
      });
    return () => {
      current = false;
    };
  }, [request, selected]);

  function toggle(path: string) {
    setNotice(undefined);
    setSelected((list) =>
      list.includes(path) ? list.filter((p) => p !== path) : [...list, path],
    );
  }

  function close() {
    setSelected([]);
    setOpen(false);
  }

  async function confirm() {
    if (!counted) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const data = await request("/api/library/import/scan", {
        method: "POST",
        body: JSON.stringify({ paths: selected, confirm: true }),
      });
      setNotice(
        t("import.server.queued", { count: (data.paths as string[]).length }),
      );
      close();
    } catch (e) {
      setFailure((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  const parts = directory ? directory.split("/") : [];

  return (
    <>
      <Modal
        open={open}
        onOpenChange={(next) => !next && close()}
        title={t("import.server.title")}
        closeLabel={t("import.server.close")}
        variant="wide"
      >
        {rooted === false && (
          <p className="m-0">{t("import.server.unconfigured")}</p>
        )}
        {rooted && (
          <>
            <nav
              aria-label={t("import.server.path")}
              className="flex flex-wrap items-center gap-1 text-sm"
            >
              <button
                type="button"
                className="inline-flex min-h-8 items-center gap-1 rounded px-1.5 py-0.5 hover:bg-side"
                onClick={() => setDirectory("")}
              >
                <HardDrive className="size-4" aria-hidden="true" />
                {t("import.server.rootName")}
              </button>
              {parts.map((part, index) => (
                <span
                  key={parts.slice(0, index + 1).join("/")}
                  className="flex items-center gap-1"
                >
                  <ChevronRight
                    className="size-3.5 text-mute"
                    aria-hidden="true"
                  />
                  <button
                    type="button"
                    className="min-h-8 rounded px-1.5 py-0.5 hover:bg-side"
                    onClick={() =>
                      setDirectory(parts.slice(0, index + 1).join("/"))
                    }
                  >
                    {part}
                  </button>
                </span>
              ))}
            </nav>
            <ul className="m-0 flex max-h-[45vh] list-none flex-col overflow-y-auto rounded-lg border border-line p-1">
              {entries.map((entry) => (
                <li
                  key={entry.path}
                  className="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-side"
                >
                  <input
                    type="checkbox"
                    className="size-5 shrink-0 md:size-4"
                    checked={selected.includes(entry.path)}
                    disabled={entry.locked || busy}
                    aria-label={t("import.server.select", {
                      name: last(entry.path),
                    })}
                    onChange={() => toggle(entry.path)}
                  />
                  {entry.directory ? (
                    entry.locked ? (
                      <span className="flex min-h-9 flex-1 items-center gap-2 text-mute">
                        <Lock className="size-4" aria-hidden="true" />
                        {last(entry.path)}
                        <span className="text-xs">
                          {t("import.server.private")}
                        </span>
                      </span>
                    ) : (
                      <>
                        <Folder
                          className="size-4 shrink-0 text-mute"
                          aria-hidden="true"
                        />
                        <button
                          type="button"
                          className="min-h-9 min-w-0 flex-1 truncate text-left"
                          onClick={() => setDirectory(entry.path)}
                        >
                          {last(entry.path)}
                        </button>
                        <ChevronRight
                          className="size-4 text-mute"
                          aria-hidden="true"
                        />
                      </>
                    )
                  ) : (
                    <>
                      <Film
                        className="size-4 shrink-0 text-mute"
                        aria-hidden="true"
                      />
                      <span className="min-w-0 flex-1 truncate">
                        {last(entry.path)}
                      </span>
                    </>
                  )}
                </li>
              ))}
              {entries.length === 0 && (
                <li className="px-2 py-2 text-sm text-mute">
                  {t("import.server.empty")}
                </li>
              )}
            </ul>
            {counted === 0 && (
              <p className="m-0 text-sm">{t("import.server.none")}</p>
            )}
            <div className="flex flex-wrap items-center justify-end gap-2 border-t border-line pt-3">
              <span aria-live="polite" className="mr-auto text-sm text-mute">
                {selected.length === 0
                  ? t("import.server.hint")
                  : counted === undefined
                    ? t("import.server.counting")
                    : t("import.server.summary", {
                        items: selected.length,
                        videos: counted,
                      })}
              </span>
              <button type="button" className={buttonClass()} onClick={close}>
                {t("import.server.cancel")}
              </button>
              <button
                type="button"
                className={buttonClass("primary")}
                disabled={busy || !counted}
                onClick={() => void confirm()}
              >
                {t("import.server.analyze", { count: counted ?? 0 })}
              </button>
            </div>
          </>
        )}
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      </Modal>
      {notice && (
        <p role="status" className="text-sm text-ok-fg">
          {notice}
        </p>
      )}
    </>
  );
}
