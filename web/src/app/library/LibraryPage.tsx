import { VideoStorage } from "./VideoStorage";
import { Link, useLocation } from "react-router";
import { useCallback, useMemo } from "react";
import { Folder, ChevronRight } from "lucide-react";
import { useLoaded, type Result } from "../api/loaded";
import { useAddressParams } from "../api/useAddressParams";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { ClipCard } from "./ClipCard";
import { LibraryFilters } from "./LibraryFilters";
import { LibrarySearchTabs } from "./LibrarySearchTabs";
import {
  PAGE_SIZE,
  fetchClips,
  fetchFilterOptions,
  type LibraryFilters as Filters,
} from "./libraryApi";

const KEYS = [
  "game",
  "from",
  "to",
  "source",
  "visibility",
  "owner",
  "person",
  "state",
  "folder",
] as const;

function readFilters(params: URLSearchParams): Filters {
  const page = Number(params.get("page"));
  return {
    game: params.get("game") ?? "",
    from: params.get("from") ?? "",
    to: params.get("to") ?? "",
    source: params.get("source") ?? "",
    visibility: params.get("visibility") ?? "",
    owner: params.get("owner") ?? "",
    person: params.get("person") ?? "",
    state: params.get("state") ?? "",
    folder: params.get("folder") ?? "",
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  };
}

/** Build a nested folder tree from flat folder list. */
function buildFolderTree(
  folders: { source_id: number; source: string; name: string; path: string }[],
): FolderNode[] {
  const roots = new Map<string, FolderNode>();

  for (const f of folders) {
    const parts = f.path.split("/");
    let currentMap = roots;
    let currentPath = "";

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      currentPath = currentPath ? `${currentPath}/${part}` : part;

      if (!currentMap.has(part)) {
        const isLeaf = i === parts.length - 1;
        currentMap.set(part, {
          name: part,
          path: currentPath,
          sourceId: f.source_id,
          source: f.source,
          children: new Map(),
          isLeaf,
          clipCount: 0, // We don't have clip counts per folder from the API
        });
      } else if (i === parts.length - 1) {
        // Mark as leaf if this folder has clips directly in it
        currentMap.get(part)!.isLeaf = true;
      }

      currentMap = currentMap.get(part)!.children;
    }
  }

  return Array.from(roots.values());
}

type FolderNode = {
  name: string;
  path: string;
  sourceId: number;
  source: string;
  children: Map<string, FolderNode>;
  isLeaf: boolean;
  clipCount: number;
};

/** Flatten folder tree for rendering with indentation. */
function flattenFolders(
  nodes: FolderNode[],
  depth = 0,
): { node: FolderNode; depth: number }[] {
  const result: { node: FolderNode; depth: number }[] = [];
  for (const node of nodes) {
    result.push({ node, depth });
    if (node.children.size > 0) {
      result.push(
        ...flattenFolders(Array.from(node.children.values()), depth + 1),
      );
    }
  }
  return result;
}

/** The library: the clips in a grid, filtered by what the address says, a page at a time. */
export function LibraryPage() {
  const { t, error } = useI18n();
  const location = useLocation();
  const { params, update } = useAddressParams();
  useDocumentTitle(t("pages.library.title"));

  const filters = useMemo(() => readFilters(params), [params]);
  const active = KEYS.some((key) => filters[key as keyof Filters] !== "");

  const clips = useLoaded(useCallback(() => fetchClips(filters), [filters]));
  const choices = useLoaded(fetchFilterOptions);
  const foldersData = useLoaded<
    {
      source_id: number;
      source: string;
      name: string;
      path: string;
    }[]
  >(async () => {
    const response = await fetch("/api/library/folders");
    if (!response.ok) throw new Error("unavailable");
    const data = await response.json();
    return { ok: true, value: data.items };
  });

  function change(patch: Partial<Filters>) {
    update((next) => {
      for (const [key, value] of Object.entries(patch)) {
        if (value === "" || value === 1) next.delete(key);
        else next.set(key, String(value));
      }
      if (!("page" in patch)) next.delete("page");
    });
  }

  function clearFolderFilter() {
    change({ folder: "", page: 1 });
  }

  function navigateToFolder(folderPath: string) {
    change({ folder: folderPath, page: 1 });
  }

  function navigateToParent() {
    if (!filters.folder) return;
    const parts = filters.folder.split("/");
    if (parts.length <= 1) {
      clearFolderFilter();
    } else {
      parts.pop();
      change({ folder: parts.join("/"), page: 1 });
    }
  }

  const page = clips.state.status === "ready" ? clips.state.value : null;
  const pages = page ? Math.max(1, Math.ceil(page.total / PAGE_SIZE)) : 1;
  const from = location.pathname + location.search;

  const foldersValue =
    foldersData.state.status === "ready" ? foldersData.state.value : [];
  const folderTree = useMemo(
    () => (foldersValue.length > 0 ? buildFolderTree(foldersValue) : []),
    [foldersValue],
  );

  const flatFolders = useMemo(() => flattenFolders(folderTree), [folderTree]);

  const currentFolderParts = filters.folder ? filters.folder.split("/") : [];

  return (
    <section className="flex flex-col gap-5 px-4 py-6 md:px-7">
      <LibrarySearchTabs />
      <div className="flex items-baseline gap-3">
        <h1 className="text-[22px] font-semibold">
          {t("pages.library.title")}
        </h1>
        {page && (
          <p className="text-mute" aria-live="polite">
            {page.total === 1
              ? t("library.countOne")
              : t("library.count", { count: page.total })}
          </p>
        )}
      </div>

      <VideoStorage />
      {foldersData.state.status === "ready" && flatFolders.length > 0 && (
        <nav
          aria-label={t("library.folderNavigation")}
          className="flex flex-col gap-2 rounded-xl border border-line p-4"
        >
          <div className="flex items-center gap-2">
            <h2 className="font-semibold">{t("library.folders")}</h2>
            {filters.folder && (
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={navigateToParent}
                aria-label={t("library.parentFolder")}
              >
                <ChevronRight size={16} className="rotate-180" />
              </button>
            )}
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              className={buttonClass(
                filters.folder === "" ? "primary" : "secondary",
              )}
              onClick={clearFolderFilter}
            >
              {t("library.allClips")}
            </button>
            {flatFolders.map(({ node, depth }) => {
              const isActive = filters.folder === node.path;
              const isParent =
                filters.folder.startsWith(node.path + "/") ||
                filters.folder === node.path;
              const hasChildren = node.children.size > 0;
              return (
                <button
                  key={`${node.sourceId}:${node.path}`}
                  className={buttonClass(
                    isActive ? "primary" : isParent ? "secondary" : "secondary",
                  )}
                  onClick={() => navigateToFolder(node.path)}
                  style={
                    {
                      paddingLeft: `${12 + depth * 16}px`,
                    } as React.CSSProperties
                  }
                  disabled={!node.isLeaf && !hasChildren}
                >
                  <Folder size={16} aria-hidden="true" />
                  {node.name}
                  {hasChildren && !node.isLeaf && (
                    <ChevronRight size={12} className="ml-1 opacity-50" />
                  )}
                </button>
              );
            })}
          </div>
        </nav>
      )}
      <LibraryFilters
        filters={filters}
        options={
          choices.state.status === "ready" ? choices.state.value.options : null
        }
        persons={
          choices.state.status === "ready" ? choices.state.value.persons : []
        }
        active={active}
        onChange={change}
        onClear={() =>
          update((next) => {
            for (const key of KEYS) next.delete(key);
            next.delete("page");
          })
        }
      />

      {clips.state.status === "failed" && (
        <div className="flex flex-col items-start gap-3">
          <ErrorAlert>{error(clips.state.code)}</ErrorAlert>
          <button
            type="button"
            className={buttonClass("secondary")}
            onClick={clips.reload}
          >
            {t("library.retry")}
          </button>
        </div>
      )}
      {clips.state.status === "loading" && (
        <p className="text-mute" role="status">
          {t("library.loading")}
        </p>
      )}

      {page && page.items.length === 0 && (
        <div className="flex flex-col items-center gap-2 rounded-xl border border-dashed border-line-strong px-6 py-12 text-center">
          <h2 className="text-base font-semibold">
            {active ? t("library.noMatch.title") : t("library.empty.title")}
          </h2>
          <p className="max-w-md text-mute">
            {active ? t("library.noMatch.body") : t("library.empty.body")}
          </p>
          {!active && (
            <Link to="/import" className={`${buttonClass("primary")} mt-2`}>
              {t("library.empty.action")}
            </Link>
          )}
        </div>
      )}

      {page && page.items.length > 0 && (
        <ul className="m-0 grid list-none grid-cols-1 gap-x-4 gap-y-6 p-0 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
          {page.items.map((clip) => (
            <ClipCard
              key={clip.id}
              clip={clip}
              from={from}
              onDeleted={clips.reload}
            />
          ))}
        </ul>
      )}

      {page && pages > 1 && (
        <nav
          aria-label={t("library.pagination.label")}
          className="flex items-center justify-center gap-3"
        >
          <button
            type="button"
            className={buttonClass("secondary")}
            disabled={filters.page <= 1}
            onClick={() => change({ page: filters.page - 1 })}
          >
            {t("library.pagination.previous")}
          </button>
          <span className="text-mute">
            {t("library.pagination.position", {
              page: filters.page,
              pages,
            })}
          </span>
          <button
            type="button"
            className={buttonClass("secondary")}
            disabled={filters.page >= pages}
            onClick={() => change({ page: filters.page + 1 })}
          >
            {t("library.pagination.next")}
          </button>
        </nav>
      )}
    </section>
  );
}
