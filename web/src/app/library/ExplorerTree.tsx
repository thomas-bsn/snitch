import { useCallback, useEffect, useState } from "react";
import { ChevronRight, Folder, Lock } from "lucide-react";
import { useI18n } from "../i18n";
import { fetchExplorer, type ExplorerEntry } from "./explorerApi";

/** The folders of the Racine vidéo, loaded as they open; the current folder's branch opens with it. */
export function ExplorerTree({
  current,
  onOpen,
}: {
  current: string;
  onOpen: (path: string) => void;
}) {
  const { t } = useI18n();
  const [children, setChildren] = useState<Record<string, ExplorerEntry[]>>({});
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set([""]));
  const [failed, setFailed] = useState<string | null>(null);

  const load = useCallback(async (path: string) => {
    const result = await fetchExplorer({ path });
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
            className="flex items-center gap-1"
            style={{ paddingLeft: depth * 14 }}
          >
            {entry.locked ? (
              <span className="size-6 shrink-0" aria-hidden="true" />
            ) : (
              <button
                type="button"
                className="grid size-6 shrink-0 place-items-center rounded text-mute hover:bg-hover"
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
            <button
              type="button"
              className={`flex min-h-11 min-w-0 flex-1 items-center gap-2 rounded px-2 py-1.5 text-left text-sm hover:bg-hover md:min-h-0 ${
                isCurrent ? "bg-accent-soft font-semibold text-ink" : "text-ink"
              }`}
              aria-current={isCurrent ? "true" : undefined}
              disabled={entry.locked}
              title={
                entry.locked ? t("library.privateFolder.lockedHint") : undefined
              }
              onClick={() => onOpen(entry.path)}
            >
              {entry.locked ? (
                <Lock size={14} aria-hidden="true" />
              ) : (
                <Folder size={14} aria-hidden="true" />
              )}
              <span className="truncate">{entry.name}</span>
              {entry.locked && (
                <span className="sr-only">
                  {t("library.privateFolder.label")}
                </span>
              )}
            </button>
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
      className="min-w-0 rounded-xl border border-line p-2"
    >
      <button
        type="button"
        className={`mb-1 flex min-h-11 w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm font-semibold hover:bg-hover md:min-h-0 ${
          current === "" ? "bg-accent-soft text-ink" : "text-ink"
        }`}
        aria-current={current === "" ? "true" : undefined}
        onClick={() => onOpen("")}
      >
        {t("library.allClips")}
      </button>
      <ul className="m-0 list-none p-0">{renderLevel("", 0)}</ul>
      {failed === "" && (
        <p role="alert" className="text-sm text-bad-fg">
          {t("library.explorer.treeFailed")}
        </p>
      )}
    </nav>
  );
}
