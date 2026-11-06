import { useCallback, useEffect, useRef, useState } from "react";
import { ChevronRight, HardDrive } from "lucide-react";
import { FolderGlyph } from "./FolderGlyph";
import { useI18n } from "../i18n";
import { fetchExplorerAll, type ExplorerEntry } from "./explorerApi";

/** The folders of the Racine vidéo, loaded as they open; the current folder's branch opens with it. */
export function ExplorerTree({
  current,
  onOpen,
  refresh = 0,
}: {
  current: string;
  onOpen: (path: string) => void;
  /** Changes whenever the tree must be read again: folders added or removed on disk. */
  refresh?: number;
}) {
  const { t } = useI18n();
  const [children, setChildren] = useState<Record<string, ExplorerEntry[]>>({});
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set([""]));
  const [failed, setFailed] = useState<string | null>(null);

  const load = useCallback(async (path: string) => {
    // Every folder, page after page: a root can hold more folders than one page (100).
    const result = await fetchExplorerAll({ path });
    if (!result.ok) {
      setFailed(path);
      return;
    }
    setChildren((previous) => ({
      ...previous,
      [path]: result.value.items.filter((item) => item.directory),
    }));
  }, []);

  useEffect(() => {
    void load("");
  }, [load]);

  // Read again every folder already loaded, so a folder added or removed on disk shows.
  const loadedPaths = useRef<string[]>([]);
  loadedPaths.current = Object.keys(children);
  useEffect(() => {
    if (refresh === 0) return;
    for (const path of loadedPaths.current) void load(path);
  }, [refresh, load]);

  // Open every ancestor of the current folder so that it is visible in the tree.
  useEffect(() => {
    const parts = current ? current.split("/") : [];
    const ancestors = parts.map((_, i) => parts.slice(0, i + 1).join("/"));
    setExpanded((previous) => {
      const next = new Set(previous);
      next.add("");
      for (const path of ancestors) next.add(path);
      return next;
    });
    for (const path of ancestors) void load(path);
  }, [current, load]);

  function toggle(entry: ExplorerEntry) {
    const open = expanded.has(entry.path);
    setExpanded((previous) => {
      const next = new Set(previous);
      if (open) next.delete(entry.path);
      else next.add(entry.path);
      return next;
    });
    if (!open && children[entry.path] === undefined) void load(entry.path);
  }

  function renderLevel(path: string, depth: number): React.ReactNode {
    const items = children[path];
    if (items === undefined) {
      return failed === path ? null : (
        <li
          className="text-xs text-mute"
          style={{ paddingLeft: depth * 14 + 8 }}
        >
          {t("library.explorer.loadingFolders")}
        </li>
      );
    }
    return items.map((entry) => {
      const isOpen = expanded.has(entry.path);
      const isCurrent = current === entry.path;
      return (
        <li key={entry.path}>
          <div
            className={`group flex items-center rounded-md ${
              isCurrent ? "bg-accent-soft text-accent-ink" : "hover:bg-side"
            }`}
          >
            <button
              type="button"
              className="flex min-h-11 min-w-0 flex-1 items-center gap-2 rounded-md py-1.5 pr-1 text-left text-sm md:min-h-0"
              style={{ paddingLeft: 8 + depth * 14 }}
              aria-current={isCurrent ? "true" : undefined}
              disabled={entry.locked}
              title={
                entry.locked ? t("library.privateFolder.lockedHint") : undefined
              }
              onClick={() => onOpen(entry.path)}
            >
              <FolderGlyph
                name={entry.name}
                isPrivate={entry.private}
                size={16}
              />
              <span className="min-w-0 flex-1 truncate">{entry.name}</span>
              {entry.locked && (
                <span className="sr-only">
                  {t("library.privateFolder.label")}
                </span>
              )}
            </button>
            {!entry.locked && (
              <button
                type="button"
                className="grid size-6 shrink-0 place-items-center rounded text-mute opacity-0 hover:bg-line focus-visible:opacity-100 group-hover:opacity-100"
                aria-expanded={isOpen}
                aria-label={t(
                  isOpen
                    ? "library.explorer.collapse"
                    : "library.explorer.expand",
                  { name: entry.name },
                )}
                onClick={() => toggle(entry)}
              >
                <ChevronRight
                  size={14}
                  className={
                    isOpen
                      ? "rotate-90 transition-transform"
                      : "transition-transform"
                  }
                  aria-hidden="true"
                />
              </button>
            )}
          </div>
          {isOpen && !entry.locked && (
            <ul className="m-0 list-none p-0">
              {renderLevel(entry.path, depth + 1)}
            </ul>
          )}
        </li>
      );
    });
  }

  return (
    <nav
      aria-label={t("library.explorer.tree")}
      className="hidden min-w-0 border-r border-line p-2 md:block"
    >
      <button
        type="button"
        className={`mb-0.5 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm ${
          current === "" ? "bg-accent-soft text-accent-ink" : "hover:bg-side"
        }`}
        aria-current={current === "" ? "true" : undefined}
        onClick={() => onOpen("")}
      >
        <HardDrive size={16} aria-hidden="true" />
        {t("library.explorer.root")}
      </button>
      <ul className="m-0 list-none p-0">{renderLevel("", 1)}</ul>
      {failed === "" && (
        <p role="alert" className="text-sm text-bad-fg">
          {t("library.explorer.treeFailed")}
        </p>
      )}
    </nav>
  );
}
