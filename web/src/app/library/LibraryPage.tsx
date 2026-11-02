import { VideoStorage } from "./VideoStorage";
import { Link, useLocation } from "react-router";
import { useCallback, useMemo, useState } from "react";
import { Folder, ChevronRight, Lock, LockOpen } from "lucide-react";
import { api, errorCode, mutation } from "../api/client";
import type { components } from "../../api.gen";
import { useLoaded, type Result } from "../api/loaded";
import { useAddressParams } from "../api/useAddressParams";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { isSuperAdministrator } from "../roles";
import { useSession } from "../session";
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
function buildFolderTree(folders: LibraryFolder[]): FolderNode[] {
  const roots = new Map<string, FolderNode>();

  for (const f of folders) {
    const parts = f.path.split("/");
    let currentMap = roots;
    let currentPath = "";

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      currentPath = currentPath ? `${currentPath}/${part}` : part;
      // A top-level folder belongs to one source: its privacy is set per source.
      const key = i === 0 ? `${f.source_id}:${part}` : part;

      if (!currentMap.has(key)) {
        const isLeaf = i === parts.length - 1;
        currentMap.set(key, {
          name: part,
          path: currentPath,
          sourceId: f.source_id,
          source: f.source,
          children: new Map(),
          isLeaf,
          clipCount: 0, // We don't have clip counts per folder from the API
          isPrivate: i === 0 && f.private,
          locked: i === 0 && f.locked,
        });
      } else if (i === parts.length - 1) {
        // Mark as leaf if this folder has clips directly in it
        currentMap.get(key)!.isLeaf = true;
      }

      currentMap = currentMap.get(key)!.children;
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
  /** Made private by the Super-administrator (top-level folders only). */
  isPrivate: boolean;
  /** Private and not openable by the current user: only its name is shown. */
  locked: boolean;
};

type LibraryFolder = components["schemas"]["LibraryFolder"];

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
  const { user, session } = useSession();
  const canSetPrivacy = user ? isSuperAdministrator(user) : false;
  const foldersData = useLoaded<LibraryFolder[]>(async () => {
    const { data, error: failure } = await api.GET("/api/library/folders");
    if (!data) throw new Error(String(failure ?? "unavailable"));
    return { ok: true, value: data.items };
  });
  const [privacyTarget, setPrivacyTarget] = useState<FolderNode | null>(null);
  const [privacyBusy, setPrivacyBusy] = useState(false);
  const [privacyError, setPrivacyError] = useState<string | null>(null);

  async function applyPrivacy() {
    if (!privacyTarget) return;
    setPrivacyBusy(true);
    setPrivacyError(null);
    const { error: failure } = await api.PUT(
      "/api/admin/library/folders/privacy",
      {
        params: { header: mutation(session?.csrf_token) } as never,
        body: {
          source_id: privacyTarget.sourceId,
          folder: privacyTarget.name,
          private: !privacyTarget.isPrivate,
        },
      },
    );
    setPrivacyBusy(false);
    if (failure) {
      setPrivacyError(error(errorCode(failure) ?? "internal_error"));
      return;
    }
    setPrivacyTarget(null);
    foldersData.reload();
    clips.reload();
  }

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
              const folderButton = (
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
                  disabled={node.locked || (!node.isLeaf && !hasChildren)}
                  title={
                    node.locked
                      ? t("library.privateFolder.lockedHint")
                      : undefined
                  }
                >
                  {node.isPrivate ? (
                    <Lock size={16} aria-hidden="true" />
                  ) : (
                    <Folder size={16} aria-hidden="true" />
                  )}
                  {node.name}
                  {node.isPrivate && (
                    <span className="sr-only">
                      {t("library.privateFolder.label")}
                    </span>
                  )}
                  {hasChildren && !node.isLeaf && !node.locked && (
                    <ChevronRight size={12} className="ml-1 opacity-50" />
                  )}
                </button>
              );
              if (!canSetPrivacy || depth > 0) return folderButton;
              return (
                <span
                  key={`${node.sourceId}:${node.path}`}
                  className="inline-flex items-center gap-1"
                >
                  {folderButton}
                  <button
                    type="button"
                    className={buttonClass("secondary")}
                    onClick={() => {
                      setPrivacyError(null);
                      setPrivacyTarget(node);
                    }}
                    aria-label={t(
                      node.isPrivate
                        ? "library.privateFolder.makeVisible"
                        : "library.privateFolder.makePrivate",
                      { name: node.name },
                    )}
                    title={t(
                      node.isPrivate
                        ? "library.privateFolder.makeVisible"
                        : "library.privateFolder.makePrivate",
                      { name: node.name },
                    )}
                  >
                    {node.isPrivate ? (
                      <LockOpen size={14} aria-hidden="true" />
                    ) : (
                      <Lock size={14} aria-hidden="true" />
                    )}
                  </button>
                </span>
              );
            })}
          </div>
        </nav>
      )}
      {privacyTarget && (
        <ConfirmDialog
          open
          onOpenChange={(open) => {
            if (!open) setPrivacyTarget(null);
          }}
          title={t(
            privacyTarget.isPrivate
              ? "library.privateFolder.makeVisibleTitle"
              : "library.privateFolder.makePrivateTitle",
            { name: privacyTarget.name },
          )}
          cancelLabel={t("segmentEdit.cancel")}
          confirmLabel={t(
            privacyTarget.isPrivate
              ? "library.privateFolder.confirmVisible"
              : "library.privateFolder.confirmPrivate",
          )}
          busy={privacyBusy}
          onConfirm={applyPrivacy}
        >
          <p>
            {t(
              privacyTarget.isPrivate
                ? "library.privateFolder.makeVisibleBody"
                : "library.privateFolder.makePrivateBody",
            )}
          </p>
          <p>{t("library.privateFolder.distinctFromClip")}</p>
          {privacyError && <p role="alert">{privacyError}</p>}
        </ConfirmDialog>
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
