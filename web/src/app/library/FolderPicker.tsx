import {
  ChevronRight,
  Folder,
  FolderPlus,
  HardDrive,
  Lock,
} from "lucide-react";
import { useCallback, useEffect, useId, useState } from "react";
import { buttonClass } from "../components/buttons";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import { isAdministrator } from "../roles";
import { useSession } from "../session";

type Entry = {
  path: string;
  directory: boolean;
  size_bytes: number;
  private?: boolean;
  locked?: boolean;
};

type FolderRequest = { parent: string; name: string; path: string };

const last = (path: string) => path.split("/").pop() ?? path;

/**
 * Picks the upload destination among the folders of the managed video root: a « Ranger dans » line with
 * the chosen folder and a « Changer… » action opening a dialog (path, folders, « Nouveau dossier »).
 * `value` is undefined until a folder was chosen on purpose; the empty string is the video root, chosen
 * explicitly. A member's new folder is a request an administrator decides (ADR 0033); an
 * administrator's exists at once and becomes the destination.
 */
export function FolderPicker({
  value,
  onChange,
}: {
  value: string | undefined;
  onChange: (path: string) => void;
}) {
  const { t, error } = useI18n();
  const { session, user } = useSession();
  const admin = user ? isAdministrator(user) : false;
  const csrf = session?.csrf_token ?? "";
  const valueId = useId();
  const [open, setOpen] = useState(false);
  const [directory, setDirectory] = useState("");
  const [entries, setEntries] = useState<Entry[]>([]);
  const [failure, setFailure] = useState("");
  // Folders waiting for a decision (ADR 0033) cannot receive uploads; a member creates them by request.
  const [waiting, setWaiting] = useState<FolderRequest[]>([]);
  const [name, setName] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [notice, setNotice] = useState("");
  const [filter, setFilter] = useState("");

  const loadWaiting = useCallback(async () => {
    try {
      const response = await fetch("/api/folder-requests", {
        headers: { "X-CSRF-Token": csrf },
      });
      if (!response.ok) return;
      const data = (await response.json()) as { items: FolderRequest[] };
      setWaiting(data.items);
    } catch {
      setWaiting([]);
    }
  }, [csrf]);

  const load = useCallback(
    async (path: string) => {
      try {
        const response = await fetch(
          `/api/admin/video-root/browse?path=${encodeURIComponent(path)}`,
          { headers: { "X-CSRF-Token": csrf } },
        );
        const data = await response.json();
        if (!response.ok) throw new Error(data.error ?? "unavailable");
        setEntries(data.items.filter((entry: Entry) => entry.directory));
        setFailure("");
      } catch (e) {
        setFailure(e instanceof Error ? e.message : "unavailable");
      }
    },
    [csrf],
  );

  useEffect(() => {
    if (open) void load(directory);
  }, [open, directory, load]);

  useEffect(() => {
    if (open) void loadWaiting();
  }, [open, loadWaiting]);

  function openChange(next: boolean) {
    setOpen(next);
    if (next) {
      setDirectory(value ?? "");
      setFormOpen(false);
      setFilter("");
      setName("");
      setNotice("");
    }
  }

  function go(path: string) {
    setDirectory(path);
    setFilter("");
  }

  function choose(path: string) {
    onChange(path);
    setOpen(false);
  }

  async function createFolder() {
    const folder = name.trim();
    if (!folder) return;
    setCreating(true);
    setNotice("");
    setFailure("");
    try {
      const response = await fetch("/api/folder-requests", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": csrf,
        },
        body: JSON.stringify({ parent: directory, name: folder }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error ?? "unavailable");
      setName("");
      setFormOpen(false);
      if (data.status === "created") {
        // An administrator's folder exists at once: it becomes the destination.
        choose(data.path as string);
        return;
      }
      setNotice(t("import.folder.requested"));
      await loadWaiting();
    } catch (e) {
      setFailure(e instanceof Error ? e.message : "unavailable");
    } finally {
      setCreating(false);
    }
  }

  const waitingPaths = waiting.map((item) => item.path);
  // A request made in this folder is listed with the folders, marked as waiting.
  const pending = waiting.filter(
    (item) =>
      item.parent === directory && !entries.some((e) => e.path === item.path),
  );
  // A filter only earns its place in a long list.
  const filtering = entries.length > 12;
  const shown = entries.filter(
    (entry) =>
      !filter.trim() ||
      last(entry.path).toLowerCase().includes(filter.trim().toLowerCase()),
  );
  const parts = directory ? directory.split("/") : [];
  const here = directory ? `/${directory}` : "/";

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <span className="text-mute" id={valueId}>
          {t("import.destination.into")}
        </span>
        {value === undefined ? (
          <span className="text-mute">{t("import.destination.unset")}</span>
        ) : (
          <code className="rounded bg-side px-2 py-1">
            {value === "" ? "/" : `/${value}`}
          </code>
        )}
        <button
          type="button"
          aria-describedby={valueId}
          className="inline-flex min-h-11 items-center text-accent-ink hover:underline md:min-h-0"
          onClick={() => openChange(true)}
        >
          {value === undefined
            ? t("import.destination.choose")
            : t("import.destination.change")}
        </button>
      </div>
      {value === undefined && (
        <p className="m-0 text-xs text-mute">{t("import.destination.none")}</p>
      )}
      <Modal
        open={open}
        onOpenChange={openChange}
        title={t("import.destination.label")}
        closeLabel={t("import.destination.close")}
        variant="wide"
      >
        <nav
          aria-label={t("import.destination.path")}
          className="flex flex-wrap items-center gap-1 text-sm"
        >
          <button
            type="button"
            className="inline-flex min-h-8 items-center gap-1 rounded px-1.5 py-0.5 hover:bg-side"
            onClick={() => go("")}
          >
            <HardDrive className="size-4" aria-hidden="true" />
            {t("import.destination.rootName")}
          </button>
          {parts.map((part, index) => (
            <span
              key={parts.slice(0, index + 1).join("/")}
              className="flex items-center gap-1"
            >
              <ChevronRight className="size-3.5 text-mute" aria-hidden="true" />
              <button
                type="button"
                className="min-h-8 rounded px-1.5 py-0.5 hover:bg-side"
                onClick={() => go(parts.slice(0, index + 1).join("/"))}
              >
                {part}
              </button>
            </span>
          ))}
        </nav>
        {filtering && (
          <input
            type="search"
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
            aria-label={t("import.destination.filter")}
            placeholder={t("import.destination.filter")}
            className="h-11 rounded-lg border border-line-strong bg-canvas px-3 md:h-9"
          />
        )}
        <ul className="m-0 flex max-h-[45vh] list-none flex-col overflow-y-auto rounded-lg border border-line p-1">
          {shown.map((entry) => (
            <li key={entry.path}>
              <button
                type="button"
                disabled={entry.locked || waitingPaths.includes(entry.path)}
                className="flex min-h-11 w-full items-center gap-2 rounded-md px-2 py-1.5 text-left hover:bg-side disabled:cursor-default disabled:text-mute md:min-h-9"
                onClick={() => go(entry.path)}
              >
                {entry.locked ? (
                  <Lock className="size-4 shrink-0" aria-hidden="true" />
                ) : (
                  <Folder
                    className="size-4 shrink-0 text-mute"
                    aria-hidden="true"
                  />
                )}
                <span className="min-w-0 flex-1 truncate">
                  {last(entry.path)}
                  {entry.locked && (
                    <span className="ml-2 text-xs">
                      {t("import.server.private")}
                    </span>
                  )}
                </span>
                {waitingPaths.includes(entry.path) ? (
                  <span className="rounded-full bg-warn-bg px-2 text-xs text-warn-fg">
                    {t("import.folder.pending")}
                  </span>
                ) : (
                  !entry.locked && (
                    <ChevronRight
                      className="size-4 text-mute"
                      aria-hidden="true"
                    />
                  )
                )}
              </button>
            </li>
          ))}
          {pending.map((item) => (
            <li
              key={item.path}
              className="flex min-h-9 items-center gap-2 rounded-md px-2 py-1.5 text-mute"
            >
              <Folder className="size-4" aria-hidden="true" />
              <span className="min-w-0 truncate">{item.name}</span>
              <span className="rounded-full bg-warn-bg px-2 text-xs text-warn-fg">
                {t("import.folder.pending")}
              </span>
            </li>
          ))}
          {shown.length === 0 && pending.length === 0 && (
            <li className="px-2 py-2 text-sm text-mute">
              {t("import.destination.empty")}
            </li>
          )}
        </ul>
        {formOpen ? (
          <form
            className="flex flex-col gap-2 rounded-lg border border-line p-3"
            onSubmit={(event) => {
              event.preventDefault();
              void createFolder();
            }}
          >
            <label className="flex flex-col gap-1 text-sm">
              <span className="font-medium">
                {t("import.folder.createIn", { path: here })}
              </span>
              <input
                value={name}
                onChange={(event) => setName(event.target.value)}
                maxLength={255}
                aria-label={t("import.folder.name")}
                className="h-11 rounded-lg border border-line-strong bg-surface px-3 md:h-9"
              />
            </label>
            {!admin && (
              <p className="m-0 text-xs text-mute">
                {t("import.folder.userHint")}
              </p>
            )}
            <div className="flex justify-end gap-2">
              <button
                type="button"
                className={buttonClass()}
                onClick={() => setFormOpen(false)}
              >
                {t("import.folder.cancel")}
              </button>
              <button
                type="submit"
                className={buttonClass("primary")}
                disabled={creating || !name.trim()}
              >
                {admin ? t("import.folder.create") : t("import.folder.send")}
              </button>
            </div>
          </form>
        ) : (
          <button
            type="button"
            className={`${buttonClass()} w-fit`}
            onClick={() => {
              setNotice("");
              setFormOpen(true);
            }}
          >
            <FolderPlus className="size-4" aria-hidden="true" />
            {t("import.folder.new")}
          </button>
        )}
        {notice && (
          <p role="status" className="m-0 text-sm text-mute">
            {notice}
          </p>
        )}
        {failure && (
          <p role="alert" className="m-0 text-sm text-bad-fg">
            {error(failure)}
          </p>
        )}
        <div className="flex items-center justify-end gap-2 border-t border-line pt-3">
          <span className="mr-auto text-sm text-mute">
            {t("import.destination.current", { path: here })}
          </span>
          <button
            type="button"
            className={buttonClass("primary")}
            onClick={() => choose(directory)}
          >
            {t("import.destination.useThis")}
          </button>
        </div>
      </Modal>
    </div>
  );
}
