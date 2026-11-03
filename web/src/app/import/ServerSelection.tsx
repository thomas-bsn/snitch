import { Folder, Lock, Video } from "lucide-react";
import { useCallback, useEffect, useId, useState } from "react";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
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
 * « Déjà sur le serveur » (#224): picks videos already in the Racine vidéo. A folder selects its videos
 * recursively; a Dossier privé shows a lock and cannot be selected (ADR 0026). The server counts the
 * selection before anything is queued, and queues it only once confirmed; indexing and analysis then
 * follow the settings of the library (Lot B).
 */
export function ServerSelection() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const csrf = session?.csrf_token ?? "";
  const headingId = useId();
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

  function toggle(path: string) {
    setCounted(undefined);
    setNotice(undefined);
    setSelected((list) =>
      list.includes(path) ? list.filter((p) => p !== path) : [...list, path],
    );
  }

  async function count() {
    setBusy(true);
    setFailure(undefined);
    try {
      const data = await request("/api/library/import/scan", {
        method: "POST",
        body: JSON.stringify({ paths: selected, confirm: false }),
      });
      setCounted((data.paths as string[]).length);
    } catch (e) {
      setFailure((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  async function confirm() {
    if (counted === undefined) return;
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
      setSelected([]);
      setCounted(undefined);
    } catch (e) {
      setFailure((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  const parts = directory ? directory.split("/") : [];

  return (
    <section aria-labelledby={headingId} className="flex flex-col gap-4">
      <h2 id={headingId} className="text-lg font-semibold">
        {t("import.sections.server")}
      </h2>
      <p className="max-w-2xl text-sm text-mute">{t("import.server.intro")}</p>
      {rooted === false && <p>{t("import.server.unconfigured")}</p>}
      {rooted && (
        <>
          <nav
            aria-label={t("import.server.path")}
            className="flex flex-wrap items-center gap-1 text-sm text-mute"
          >
            <button
              type="button"
              className="min-h-8 text-accent-ink underline underline-offset-2"
              onClick={() => setDirectory("")}
            >
              {t("import.server.rootName")}
            </button>
            {parts.map((part, index) => (
              <span key={parts.slice(0, index + 1).join("/")}>
                {" / "}
                <button
                  type="button"
                  className="min-h-8 text-accent-ink underline underline-offset-2"
                  onClick={() =>
                    setDirectory(parts.slice(0, index + 1).join("/"))
                  }
                >
                  {part}
                </button>
              </span>
            ))}
          </nav>
          {directory && (
            <button
              type="button"
              className={`${buttonClass("secondary")} self-start`}
              onClick={() => setDirectory(parts.slice(0, -1).join("/"))}
            >
              {t("import.server.up")}
            </button>
          )}
          <ul className="flex max-h-96 flex-col divide-y divide-line overflow-y-auto rounded-xl border border-line">
            {entries.map((entry) => {
              const checked = selected.includes(entry.path);
              return (
                <li key={entry.path} className="flex items-center gap-3 px-3">
                  <input
                    type="checkbox"
                    className="size-5 shrink-0"
                    checked={checked}
                    disabled={entry.locked || busy}
                    aria-label={t("import.server.select", {
                      name: last(entry.path),
                    })}
                    onChange={() => toggle(entry.path)}
                  />
                  {entry.directory ? (
                    entry.locked ? (
                      <span className="flex min-h-11 items-center gap-2 text-mute">
                        <Lock size={18} aria-hidden="true" />
                        {last(entry.path)}
                        <span className="text-xs">
                          {t("import.server.private")}
                        </span>
                      </span>
                    ) : (
                      <button
                        type="button"
                        className="flex min-h-11 flex-1 items-center gap-2 text-left text-accent-ink"
                        onClick={() => setDirectory(entry.path)}
                      >
                        <Folder size={18} aria-hidden="true" />
                        {last(entry.path)}
                      </button>
                    )
                  ) : (
                    <span className="flex min-h-11 flex-1 items-center gap-2">
                      <Video
                        size={18}
                        aria-hidden="true"
                        className="text-mute"
                      />
                      {last(entry.path)}
                    </span>
                  )}
                </li>
              );
            })}
            {rooted && entries.length === 0 && (
              <li className="px-3 py-3 text-sm text-mute">
                {t("import.server.empty")}
              </li>
            )}
          </ul>
          <p aria-live="polite" className="text-sm text-mute">
            {t("import.server.selected", { count: selected.length })}
          </p>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className={buttonClass("secondary")}
              disabled={busy || selected.length === 0}
              onClick={() => void count()}
            >
              {busy && counted === undefined
                ? t("import.server.counting")
                : t("import.server.check")}
            </button>
            {selected.length > 0 && (
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => {
                  setSelected([]);
                  setCounted(undefined);
                }}
              >
                {t("import.server.clear")}
              </button>
            )}
          </div>
          {counted !== undefined && (
            <div className="flex flex-col gap-3 rounded-xl border border-line-strong p-4">
              {counted === 0 ? (
                <p>{t("import.server.none")}</p>
              ) : (
                <>
                  <p className="font-medium">
                    {t("import.server.total", { count: counted })}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      className={buttonClass("primary")}
                      disabled={busy}
                      onClick={() => void confirm()}
                    >
                      {t("import.server.confirm", { count: counted })}
                    </button>
                    <button
                      type="button"
                      className={buttonClass("secondary")}
                      onClick={() => setCounted(undefined)}
                    >
                      {t("import.server.cancel")}
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </>
      )}
      {notice && (
        <p role="status" className="text-sm text-ok-fg">
          {notice}
        </p>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
    </section>
  );
}
