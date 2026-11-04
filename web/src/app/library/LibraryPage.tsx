import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";
import {
  ChevronLeft,
  ChevronRight,
  File,
  Film,
  Folder,
  FolderLock,
  HardDrive,
  Lock,
  LockOpen,
  MoreHorizontal,
  Play,
} from "lucide-react";
import { api, errorCode, mutation } from "../api/client";
import { useLoaded } from "../api/loaded";
import { useAddressParams } from "../api/useAddressParams";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { isAdministrator, isSuperAdministrator } from "../roles";
import { useSession } from "../session";
import { formatTimecode } from "./clipFormat";
import { ExplorerThumbnail } from "./ExplorerThumbnail";
import { ExplorerTree } from "./ExplorerTree";
import {
  EXPLORER_MAX_LIMIT,
  baseName,
  fetchExplorer,
  fetchExplorerAll,
  requestIndex,
  asExplorerState,
  type ExplorerEntry,
  type ExplorerState,
} from "./explorerApi";
import { LibraryFilters, type ExplorerFilters } from "./LibraryFilters";
import { LibrarySearchTabs } from "./LibrarySearchTabs";
import { fetchFilterOptions } from "./libraryApi";

type View = "list" | "icons";

// The Library filter lives in `filter`; `q` belongs to the transcript search.
const FILTER_KEYS = [
  "filter",
  "state",
  "game",
  "from",
  "to",
  "visibility",
  "owner",
  "person",
  "voices",
] as const;

/** Items per page when the address does not ask for another size. */
const DEFAULT_PAGE_SIZE = 50;
// How long the analysis of a folder waits for videos it asked to index: 60 polls, 2 s apart.
const INDEX_WAIT_ATTEMPTS = 60;
const INDEX_WAIT_MS = 2000;

function readNumber(value: string | null, fallback: number, max: number) {
  const n = Number(value);
  return Number.isInteger(n) && n >= 1 ? Math.min(n, max) : fallback;
}

function readFilters(params: URLSearchParams): ExplorerFilters {
  return {
    q: params.get("filter") ?? "",
    state: asExplorerState(params.get("state")),
    game: params.get("game") ?? "",
    from: params.get("from") ?? "",
    to: params.get("to") ?? "",
    visibility: params.get("visibility") ?? "",
    owner: params.get("owner") ?? "",
    person: params.get("person") ?? "",
    voices: params.get("voices") === "pending" ? "pending" : "",
  };
}

function formatSize(bytes: number, language: string) {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${new Intl.NumberFormat(language, { maximumFractionDigits: 1 }).format(value)} ${units[unit]}`;
}

/** The Bibliothèque as an explorer of the Racine vidéo (ADR 0032): a tree, a toolbar, a list or icons. */
export function LibraryPage() {
  const { t, error, language } = useI18n();
  const navigate = useNavigate();
  const { params, update } = useAddressParams();
  useDocumentTitle(t("pages.library.title"));
  const { user, session } = useSession();
  const csrf = session?.csrf_token ?? "";
  const admin = user ? isAdministrator(user) : false;
  const superAdmin = user ? isSuperAdministrator(user) : false;

  const folder = (params.get("folder") ?? "").replace(/^\/+|\/+$/g, "");
  const view: View = params.get("view") === "icons" ? "icons" : "list";
  const filters = useMemo(() => readFilters(params), [params]);
  const filtered = FILTER_KEYS.some((key) => {
    const value =
      key === "owner" ? filters.owner : filters[key as keyof ExplorerFilters];
    return value !== "";
  });
  const pageSize = readNumber(
    params.get("size"),
    DEFAULT_PAGE_SIZE,
    EXPLORER_MAX_LIMIT,
  );
  const pageNumber = readNumber(params.get("page"), 1, 1_000_000);
  // Folded by default; open when the address already carries filters.
  const [filtersOpen, setFiltersOpen] = useState(filtered);
  const [search, setSearch] = useState(filters.q);

  const listing = useLoaded(
    useCallback(
      () =>
        fetchExplorer({
          path: folder,
          recursive: filtered,
          q: filters.q,
          state: filters.state,
          game: filters.game,
          from: filters.from,
          to: filters.to,
          visibility: filters.visibility,
          owner_id: filters.owner,
          person_id: filters.person,
          voices: filters.voices,
          limit: pageSize,
          offset: (pageNumber - 1) * pageSize,
        }),
      [folder, filtered, filters, pageSize, pageNumber],
    ),
  );
  const choices = useLoaded(fetchFilterOptions);
  const items: ExplorerEntry[] =
    listing.state.status === "ready" ? listing.state.value.items : [];
  const total =
    listing.state.status === "ready" ? listing.state.value.total : 0;
  const pages = Math.max(1, Math.ceil(total / pageSize));
  const sourceId =
    listing.state.status === "ready" ? listing.state.value.sourceId : 0;
  const rootConfigured =
    listing.state.status === "ready"
      ? listing.state.value.rootConfigured
      : true;

  // A video opened before its indexing: poll the folder until its clip exists, then open it.
  const [opening, setOpening] = useState<string | null>(null);
  const [openError, setOpenError] = useState<string | null>(null);
  useEffect(() => {
    if (!opening) return;
    const timer = window.setInterval(() => listing.reload(), 2000);
    return () => window.clearInterval(timer);
  }, [opening, listing]);
  useEffect(() => {
    if (!opening) return;
    const entry = items.find((item) => item.path === opening);
    if (entry?.clip_id) {
      setOpening(null);
      navigate(`/clips/${entry.clip_id}`);
    }
  }, [items, opening, navigate]);

  useEffect(() => {
    setSearch(filters.q);
  }, [filters.q]);
  useEffect(() => {
    const timer = window.setTimeout(() => {
      if (search !== filters.q) change({ filter: search });
    }, 300);
    return () => window.clearTimeout(timer);
    // The debounce follows what is typed only; `change` is stable in use.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search]);

  function change(patch: Partial<Record<string, string | number>>) {
    update((next) => {
      // Any change of the selection starts again at the first page.
      if (!("page" in patch)) next.delete("page");
      for (const [key, value] of Object.entries(patch)) {
        if (value === "" || value === undefined) next.delete(key);
        else next.set(key, String(value));
      }
    });
  }

  function openFolder(path: string) {
    setMenuFor(null);
    setOpening(null);
    update((next) => {
      next.delete("page");
      if (path) next.set("folder", path);
      else next.delete("folder");
    });
  }

  function clearFilters() {
    setSearch("");
    update((next) => {
      next.delete("page");
      for (const key of FILTER_KEYS) next.delete(key);
    });
  }

  async function openVideo(entry: ExplorerEntry) {
    setOpenError(null);
    if (entry.clip_id) {
      navigate(`/clips/${entry.clip_id}`);
      return;
    }
    const result = await requestIndex(entry.path, csrf);
    if (!result.ok) {
      setOpenError(t("library.explorer.indexFailed"));
      return;
    }
    if (result.value.clip_id) {
      navigate(`/clips/${result.value.clip_id}`);
      return;
    }
    setOpening(entry.path);
  }

  // Folder menu: analysis of the folder's unanalysed videos (administrators), privacy (super-administrators).
  const [menuFor, setMenuFor] = useState<string | null>(null);
  const [privacyTarget, setPrivacyTarget] = useState<ExplorerEntry | null>(
    null,
  );
  const [privacyBusy, setPrivacyBusy] = useState(false);
  const [privacyError, setPrivacyError] = useState<string | null>(null);
  const [analysisTarget, setAnalysisTarget] = useState<{
    path: string;
    clips: number[];
    /** Videos not indexed yet: indexed first, then analysed. */
    unindexed: string[];
  } | null>(null);
  const [analysisBusy, setAnalysisBusy] = useState(false);
  const [analysisNotice, setAnalysisNotice] = useState<string | null>(null);

  async function applyPrivacy() {
    if (!privacyTarget) return;
    setPrivacyBusy(true);
    setPrivacyError(null);
    const { error: failure } = await api.PUT(
      "/api/admin/library/folders/privacy",
      {
        params: { header: mutation(csrf) } as never,
        body: {
          source_id: sourceId,
          folder: privacyTarget.name,
          private: !privacyTarget.private,
        },
      },
    );
    setPrivacyBusy(false);
    if (failure) {
      setPrivacyError(error(errorCode(failure) ?? "internal_error"));
      return;
    }
    setPrivacyTarget(null);
    listing.reload();
  }

  async function prepareAnalysis(entry: ExplorerEntry) {
    setMenuFor(null);
    setAnalysisNotice(null);
    const [indexed, unindexed, indexing] = await Promise.all([
      fetchExplorerAll({
        path: entry.path,
        recursive: true,
        state: "not_analyzed",
      }),
      fetchExplorerAll({
        path: entry.path,
        recursive: true,
        state: "not_indexed",
      }),
      // Videos whose indexing was asked for but has not finished: no clip yet, so a later click resumes them.
      fetchExplorerAll({
        path: entry.path,
        recursive: true,
        state: "in_progress",
      }),
    ]);
    if (!indexed.ok || !unindexed.ok || !indexing.ok) {
      setAnalysisNotice(t("library.explorer.analysisFailed"));
      return;
    }
    const clips = indexed.value.items
      .map((item) => item.clip_id)
      .filter((id): id is number => id !== null);
    setAnalysisTarget({
      path: entry.path,
      clips,
      unindexed: [
        ...unindexed.value.items,
        ...indexing.value.items.filter((item) => item.clip_id === null),
      ].map((item) => item.path),
    });
  }

  async function runAnalysis() {
    if (!analysisTarget) return;
    setAnalysisBusy(true);
    let failed = 0;
    const clips = [...analysisTarget.clips];
    // Videos not indexed yet are indexed first (no analysis), then analysed once their clip exists.
    const waiting = new Set(analysisTarget.unindexed);
    for (const path of analysisTarget.unindexed) {
      const indexed = await requestIndex(path, csrf);
      if (!indexed.ok) {
        waiting.delete(path);
        failed += 1;
      } else if (indexed.value.clip_id) {
        waiting.delete(path);
        clips.push(indexed.value.clip_id);
      }
    }
    for (
      let attempt = 0;
      waiting.size > 0 && attempt < INDEX_WAIT_ATTEMPTS;
      attempt += 1
    ) {
      await new Promise((resolve) => window.setTimeout(resolve, INDEX_WAIT_MS));
      const found = await fetchExplorerAll({
        path: analysisTarget.path,
        recursive: true,
      });
      if (!found.ok) continue;
      for (const item of found.value.items) {
        if (waiting.has(item.path) && item.clip_id) {
          waiting.delete(item.path);
          clips.push(item.clip_id);
        }
      }
    }
    // Still indexing after the delay: not a failure, the next click resumes them.
    const pending = waiting.size;
    for (const id of clips) {
      const { error: failure } = await api.POST(
        "/api/admin/clips/{id}/analysis",
        {
          params: {
            path: { id },
            header: mutation(csrf),
          } as never,
        },
      );
      if (failure) failed += 1;
    }
    setAnalysisBusy(false);
    setAnalysisNotice(
      pending > 0
        ? t("library.explorer.analysisPending", {
            count: clips.length - failed,
            pending,
          })
        : failed === 0
          ? t("library.explorer.analysisQueued", {
              count: clips.length,
            })
          : t("library.explorer.analysisPartial", {
              count: clips.length - failed,
              failed,
            }),
    );
    setAnalysisTarget(null);
  }

  const crumbs = folder ? folder.split("/") : [];
  const parentPath = crumbs.slice(0, -1).join("/");
  const flat = filtered;

  const rootLabel = t("library.explorer.root");
  const toggle =
    "rounded-md px-2.5 py-1 text-sm min-h-11 min-w-11 md:min-h-0 md:min-w-0";
  return (
    <section className="flex min-h-[85vh] flex-col">
      <h1 className="sr-only">{t("pages.library.title")}</h1>
      <div className="px-4 pt-3 md:px-6">
        <LibrarySearchTabs />
      </div>

      <div className="flex flex-wrap items-center gap-3 border-b border-line px-4 py-3 md:px-6">
        <button
          type="button"
          className="flex size-11 items-center justify-center rounded-lg border border-line-strong disabled:opacity-40 md:size-9"
          onClick={() => openFolder(parentPath)}
          disabled={!folder}
          aria-label={t("library.parentFolder")}
        >
          <ChevronLeft size={16} aria-hidden="true" />
        </button>
        <nav aria-label={t("library.explorer.breadcrumb")} className="min-w-0">
          <ol className="m-0 flex list-none flex-wrap items-center gap-1 p-0 text-sm">
            <li>
              <button
                type="button"
                className="inline-flex min-h-11 items-center gap-1 rounded px-1.5 py-0.5 hover:bg-side md:min-h-0"
                onClick={() => openFolder("")}
                aria-current={folder ? undefined : "page"}
              >
                <HardDrive size={16} aria-hidden="true" />
                {rootLabel}
              </button>
            </li>
            {crumbs.map((part, index) => {
              const path = crumbs.slice(0, index + 1).join("/");
              const last = index === crumbs.length - 1;
              return (
                <li key={path} className="flex items-center gap-1">
                  <ChevronRight
                    size={14}
                    aria-hidden="true"
                    className="text-mute"
                  />
                  <button
                    type="button"
                    className="inline-flex min-h-11 items-center rounded px-1.5 py-0.5 hover:bg-side md:min-h-0"
                    onClick={() => openFolder(path)}
                    aria-current={last ? "page" : undefined}
                  >
                    {part}
                  </button>
                </li>
              );
            })}
          </ol>
        </nav>
        <label className="flex h-11 w-full items-center gap-2 rounded-lg border border-line-strong px-3 sm:ml-auto sm:w-64 md:h-9">
          <span className="sr-only">{t("library.explorer.searchLabel")}</span>
          <input
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={t("library.explorer.searchIn", {
              folder: crumbs.at(-1) ?? rootLabel,
            })}
            className="h-full min-h-11 min-w-0 flex-1 bg-transparent text-sm outline-none"
          />
        </label>
        <button
          type="button"
          className={`${buttonClass("secondary")} ${filtersOpen || filtered ? "border-accent text-accent-ink" : ""}`}
          aria-expanded={filtersOpen}
          aria-controls="library-filters"
          onClick={() => setFiltersOpen((open) => !open)}
        >
          {t("library.explorer.filters")}
        </button>
        <div
          role="group"
          aria-label={t("library.explorer.view")}
          className="flex rounded-lg border border-line-strong p-0.5"
        >
          {(["list", "icons"] as const).map((mode) => (
            <button
              key={mode}
              type="button"
              aria-pressed={view === mode}
              className={`${toggle} ${view === mode ? "bg-accent-soft text-accent-ink" : "text-mute"}`}
              onClick={() => change({ view: mode === "list" ? "" : mode })}
            >
              {t(
                mode === "list"
                  ? "library.explorer.list"
                  : "library.explorer.icons",
              )}
            </button>
          ))}
        </div>
      </div>

      {filtersOpen && (
        <div
          id="library-filters"
          className="border-b border-line bg-side px-4 py-3 md:px-6"
        >
          <LibraryFilters
            filters={filters}
            options={
              choices.state.status === "ready"
                ? choices.state.value.options
                : null
            }
            persons={
              choices.state.status === "ready"
                ? choices.state.value.persons
                : []
            }
            active={filtered}
            onChange={(patch) => change(patch)}
            onClear={clearFilters}
          />
        </div>
      )}

      <div className="grid min-w-0 flex-1 grid-cols-1 md:grid-cols-[230px_minmax(0,1fr)]">
        <ExplorerTree current={folder} onOpen={openFolder} />

        <div className="flex min-w-0 flex-col gap-3 p-4 md:px-6">
          {listing.state.status === "ready" && (
            <p className="m-0 text-sm text-mute" aria-live="polite">
              {flat
                ? t("library.explorer.flatHint", {
                    folder: crumbs.at(-1) ?? rootLabel,
                  })
                : t("library.explorer.counts", {
                    folders: items.filter((item) => item.directory).length,
                    videos:
                      total - items.filter((item) => item.directory).length,
                  })}
            </p>
          )}
          {listing.state.status === "ready" && !rootConfigured && (
            <div className="flex flex-col gap-2 rounded-xl border border-line p-4">
              <p className="font-semibold">
                {t("library.explorer.noRoot.title")}
              </p>
              <p className="text-mute">
                {admin
                  ? t("library.explorer.noRoot.adminBody")
                  : t("library.explorer.noRoot.body")}
              </p>
              {admin && (
                <Link
                  to="/admin/folders"
                  className={`${buttonClass("secondary")} w-fit`}
                >
                  {t("library.explorer.noRoot.action")}
                </Link>
              )}
            </div>
          )}
          {listing.state.status === "failed" && (
            <div className="flex flex-col items-start gap-3">
              <ErrorAlert>{error(listing.state.code)}</ErrorAlert>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={listing.reload}
              >
                {t("library.retry")}
              </button>
            </div>
          )}
          {listing.state.status === "loading" && (
            <p className="text-mute" role="status">
              {t("library.loading")}
            </p>
          )}
          {openError && <ErrorAlert>{openError}</ErrorAlert>}
          {opening && (
            <p className="text-mute" role="status">
              {t("library.explorer.indexing", { name: baseName(opening) })}
            </p>
          )}
          {analysisNotice && (
            <p role="status" className="text-sm text-ink">
              {analysisNotice}
            </p>
          )}

          {listing.state.status === "ready" && items.length === 0 && (
            <div className="flex flex-col items-center gap-2 rounded-xl border border-dashed border-line-strong px-6 py-12 text-center">
              <h2 className="text-base font-semibold">
                {filtered
                  ? t("library.noMatch.title")
                  : rootConfigured
                    ? t("library.explorer.emptyFolder")
                    : t("library.empty.title")}
              </h2>
              <p className="max-w-md text-mute">
                {filtered
                  ? t("library.noMatch.body")
                  : rootConfigured
                    ? t("library.explorer.emptyFolderBody")
                    : t("library.empty.body")}
              </p>
              {!filtered && !rootConfigured && (
                <Link to="/import" className={`${buttonClass("primary")} mt-2`}>
                  {t("library.empty.action")}
                </Link>
              )}
            </div>
          )}

          {listing.state.status === "ready" &&
            items.length > 0 &&
            view === "list" && (
              <div className="relative overflow-x-auto">
                <table className="w-full min-w-[40rem] border-collapse text-sm">
                  <caption className="sr-only">
                    {t("library.explorer.listCaption")}
                  </caption>
                  <thead className="text-left text-xs text-mute">
                    <tr>
                      <th
                        scope="col"
                        className="border-b border-line px-2 py-2 font-medium"
                      >
                        {t("library.explorer.columns.name")}
                      </th>
                      {flat && (
                        <th
                          scope="col"
                          className="border-b border-line px-2 py-2 font-medium"
                        >
                          {t("library.explorer.columns.folder")}
                        </th>
                      )}
                      <th
                        scope="col"
                        className="border-b border-line px-2 py-2 font-medium"
                      >
                        {t("library.explorer.columns.state")}
                      </th>
                      <th
                        scope="col"
                        className="border-b border-line px-2 py-2 font-medium"
                      >
                        {t("library.explorer.columns.duration")}
                      </th>
                      <th
                        scope="col"
                        className="border-b border-line px-2 py-2 font-medium"
                      >
                        {t("library.explorer.columns.owner")}
                      </th>
                      <th
                        scope="col"
                        className="border-b border-line px-2 py-2 font-medium"
                      >
                        {t("library.explorer.columns.added")}
                      </th>
                      <th
                        scope="col"
                        className="border-b border-line px-2 py-2 font-medium"
                      >
                        {t("library.explorer.columns.size")}
                      </th>
                      <th
                        scope="col"
                        className="border-b border-line px-2 py-2"
                      >
                        <span className="sr-only">
                          {t("library.explorer.columns.actions")}
                        </span>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {items.map((entry) => (
                      <ExplorerRow
                        key={entry.path}
                        entry={entry}
                        flat={flat}
                        language={language}
                        admin={admin}
                        superAdmin={superAdmin}
                        menuOpen={menuFor === entry.path}
                        onMenu={(open) => setMenuFor(open ? entry.path : null)}
                        onOpenFolder={openFolder}
                        onOpenVideo={openVideo}
                        onPrivacy={() => {
                          setPrivacyError(null);
                          setPrivacyTarget(entry);
                        }}
                        onAnalysis={() => prepareAnalysis(entry)}
                        busyPath={opening}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            )}

          {listing.state.status === "ready" &&
            items.length > 0 &&
            view === "icons" && (
              <ul className="m-0 grid list-none grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-2 p-0">
                {items.map((entry) => (
                  <li key={entry.path}>
                    <ExplorerIcon
                      entry={entry}
                      flat={flat}
                      onOpenFolder={openFolder}
                      onOpenVideo={openVideo}
                    />
                  </li>
                ))}
              </ul>
            )}

          {listing.state.status === "ready" && total > pageSize && (
            <nav
              aria-label={t("library.pagination.label")}
              className="flex flex-wrap items-center gap-3"
            >
              <button
                type="button"
                className={buttonClass("secondary")}
                disabled={pageNumber <= 1}
                onClick={() => change({ page: pageNumber - 1 })}
              >
                {t("library.pagination.previous")}
              </button>
              <span aria-live="polite">
                {t("library.pagination.position", {
                  page: pageNumber,
                  pages,
                })}
              </span>
              <button
                type="button"
                className={buttonClass("secondary")}
                disabled={pageNumber >= pages}
                onClick={() => change({ page: pageNumber + 1 })}
              >
                {t("library.pagination.next")}
              </button>
            </nav>
          )}
        </div>
      </div>

      {privacyTarget && (
        <ConfirmDialog
          open
          onOpenChange={(open) => {
            if (!open) setPrivacyTarget(null);
          }}
          title={t(
            privacyTarget.private
              ? "library.privateFolder.makeVisibleTitle"
              : "library.privateFolder.makePrivateTitle",
            { name: privacyTarget.name },
          )}
          cancelLabel={t("segmentEdit.cancel")}
          confirmLabel={t(
            privacyTarget.private
              ? "library.privateFolder.confirmVisible"
              : "library.privateFolder.confirmPrivate",
          )}
          busy={privacyBusy}
          onConfirm={applyPrivacy}
        >
          <p>
            {t(
              privacyTarget.private
                ? "library.privateFolder.makeVisibleBody"
                : "library.privateFolder.makePrivateBody",
            )}
          </p>
          <p>{t("library.privateFolder.distinctFromClip")}</p>
          {privacyError && <p role="alert">{privacyError}</p>}
        </ConfirmDialog>
      )}

      {analysisTarget && (
        <ConfirmDialog
          open
          onOpenChange={(open) => {
            if (!open) setAnalysisTarget(null);
          }}
          title={t("library.explorer.analysisTitle", {
            name: baseName(analysisTarget.path),
          })}
          cancelLabel={t("segmentEdit.cancel")}
          confirmLabel={t("library.explorer.analysisConfirm", {
            count:
              analysisTarget.clips.length + analysisTarget.unindexed.length,
          })}
          busy={analysisBusy}
          confirmDisabled={
            analysisTarget.clips.length + analysisTarget.unindexed.length === 0
          }
          onConfirm={runAnalysis}
        >
          <p>
            {analysisTarget.clips.length + analysisTarget.unindexed.length === 0
              ? t("library.explorer.analysisNone")
              : t("library.explorer.analysisBody", {
                  count:
                    analysisTarget.clips.length +
                    analysisTarget.unindexed.length,
                })}
          </p>
          {analysisTarget.unindexed.length > 0 && (
            <p>
              {t("library.explorer.analysisIndexFirst", {
                count: analysisTarget.unindexed.length,
              })}
            </p>
          )}
        </ConfirmDialog>
      )}
    </section>
  );
}

type RowProps = {
  entry: ExplorerEntry;
  flat: boolean;
  language: string;
  admin: boolean;
  superAdmin: boolean;
  menuOpen: boolean;
  onMenu: (open: boolean) => void;
  onOpenFolder: (path: string) => void;
  onOpenVideo: (entry: ExplorerEntry) => void;
  onPrivacy: () => void;
  onAnalysis: () => void;
  busyPath: string | null;
};

/** One row of the list view: a folder opens, a video opens its clip (indexing it first when needed). */
function ExplorerRow({
  entry,
  flat,
  language,
  admin,
  superAdmin,
  menuOpen,
  onMenu,
  onOpenFolder,
  onOpenVideo,
  onPrivacy,
  onAnalysis,
  busyPath,
}: RowProps) {
  const { t } = useI18n();
  const folderFlat = flat ? entry.folder : "";
  const name = (
    <span className="flex min-w-0 items-center gap-2">
      {entry.directory ? (
        entry.private ? (
          <FolderLock size={16} aria-hidden="true" />
        ) : (
          <Folder size={16} aria-hidden="true" className="text-accent" />
        )
      ) : (
        <Film size={16} aria-hidden="true" className="text-mute" />
      )}
      <span className="truncate">{entry.name}</span>
      {entry.private && (
        <span className="text-xs text-mute">
          {t("library.privateFolder.label")}
        </span>
      )}
    </span>
  );
  const busy = busyPath === entry.path;
  return (
    <tr className="border-b border-line align-middle hover:bg-side">
      <td className="px-2 py-2">
        <div className="flex items-center gap-2">
          {entry.directory ? (
            <button
              type="button"
              className="inline-flex min-h-11 max-w-full items-center text-left font-medium text-ink hover:underline disabled:cursor-not-allowed disabled:no-underline md:min-h-0"
              disabled={entry.locked}
              title={
                entry.locked ? t("library.privateFolder.lockedHint") : undefined
              }
              onClick={() => onOpenFolder(entry.path)}
            >
              {name}
            </button>
          ) : entry.clip_id ? (
            <Link
              to={`/clips/${entry.clip_id}`}
              aria-label={t("library.card.open", { name: entry.name })}
              className="inline-flex min-h-11 max-w-full items-center font-medium text-ink hover:underline md:min-h-0"
            >
              {name}
            </Link>
          ) : (
            <button
              type="button"
              className="inline-flex min-h-11 max-w-full items-center text-left font-medium text-ink hover:underline disabled:opacity-60 md:min-h-0"
              disabled={busy}
              onClick={() => onOpenVideo(entry)}
            >
              {name}
            </button>
          )}
          {entry.voices_pending > 0 && (
            <span className="rounded-full bg-warn-bg px-1.5 text-xs whitespace-nowrap text-warn-fg">
              {t("library.explorer.voicesToProcess", {
                count: entry.voices_pending,
              })}
            </span>
          )}
        </div>
      </td>
      {flat && (
        <td className="px-2 py-2 text-mute">
          {folderFlat || t("library.explorer.root")}
        </td>
      )}
      <td className="px-2 py-2">
        {entry.directory ? null : (
          <ExplorerStatePill state={busy ? "in_progress" : entry.state} />
        )}
      </td>
      <td className="px-2 py-2 tabular-nums text-mute">
        {entry.duration_seconds != null
          ? formatTimecode(entry.duration_seconds)
          : "—"}
      </td>
      <td className="px-2 py-2 text-mute">{entry.owner || "—"}</td>
      <td className="px-2 py-2 text-mute">
        {entry.added_at
          ? new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
              new Date(entry.added_at),
            )
          : "—"}
      </td>
      <td className="px-2 py-2 tabular-nums text-mute">
        {entry.directory ? "—" : formatSize(entry.size_bytes, language)}
      </td>
      <td className="relative px-3 py-2 text-right">
        {entry.directory && (admin || superAdmin) && !entry.locked && (
          <>
            <button
              type="button"
              className="inline-flex min-h-11 min-w-11 items-center justify-center rounded px-1.5 text-mute hover:bg-line md:min-h-0 md:min-w-0"
              aria-haspopup="menu"
              aria-expanded={menuOpen}
              aria-label={t("library.explorer.folderMenu", {
                name: entry.name,
              })}
              onClick={() => onMenu(!menuOpen)}
            >
              <MoreHorizontal size={16} aria-hidden="true" />
            </button>
            {menuOpen && (
              <div
                role="menu"
                className="absolute right-3 z-10 mt-1 flex min-w-[14rem] flex-col rounded-lg border border-line bg-surface p-1 text-left shadow-md"
              >
                {admin && (
                  <button
                    role="menuitem"
                    type="button"
                    className="rounded px-3 py-2 text-left text-sm hover:bg-hover"
                    onClick={onAnalysis}
                  >
                    {t("library.explorer.analyseUnanalysed")}
                  </button>
                )}
                {superAdmin && !flat && !entry.path.includes("/") && (
                  <button
                    role="menuitem"
                    type="button"
                    className="rounded px-3 py-2 text-left text-sm hover:bg-hover"
                    onClick={onPrivacy}
                  >
                    {entry.private ? (
                      <LockOpen size={14} aria-hidden="true" />
                    ) : (
                      <Lock size={14} aria-hidden="true" />
                    )}{" "}
                    {t(
                      entry.private
                        ? "library.explorer.makeShared"
                        : "library.explorer.makePrivate",
                    )}
                  </button>
                )}
              </div>
            )}
          </>
        )}
      </td>
    </tr>
  );
}

/** One tile of the icons view: the same name and state as the list. */
function ExplorerIcon({
  entry,
  flat,
  onOpenFolder,
  onOpenVideo,
}: {
  entry: ExplorerEntry;
  flat: boolean;
  onOpenFolder: (path: string) => void;
  onOpenVideo: (entry: ExplorerEntry) => void;
}) {
  const { t } = useI18n();
  const unindexed = entry.state === "not_indexed";
  const body = (
    <>
      {entry.directory ? (
        entry.private ? (
          <FolderLock
            size={56}
            strokeWidth={1.2}
            aria-hidden="true"
            className="text-mute"
          />
        ) : (
          <Folder
            size={56}
            strokeWidth={1.2}
            aria-hidden="true"
            className="text-accent"
            fill="var(--accent-soft)"
          />
        )
      ) : (
        <span
          className={`relative flex h-14 w-20 items-center justify-center rounded-md ${
            unindexed
              ? "border-2 border-dashed border-line-strong text-mute"
              : "bg-[#1b1b22] text-white/60"
          }`}
        >
          {unindexed ? (
            <File size={24} aria-hidden="true" />
          ) : (
            <Play size={20} aria-hidden="true" />
          )}
          {entry.clip_id ? <ExplorerThumbnail clipId={entry.clip_id} /> : null}
          {entry.voices_pending > 0 && (
            <span className="absolute -top-1.5 -right-1.5 rounded-full bg-warn-bg px-1.5 text-[10px] text-warn-fg">
              {entry.voices_pending}
              <span className="sr-only">
                {" "}
                {t("library.explorer.voicesToProcess", {
                  count: entry.voices_pending,
                })}
              </span>
            </span>
          )}
        </span>
      )}
      <span className="w-full truncate text-sm font-medium">{entry.name}</span>
      {flat && entry.folder && (
        <span className="w-full truncate text-xs text-mute">
          {entry.folder}
        </span>
      )}
      {!entry.directory && <ExplorerStatePill state={entry.state} />}
      {entry.private && (
        <span className="text-xs text-mute">
          {t("library.privateFolder.label")}
        </span>
      )}
    </>
  );
  const tile =
    "flex w-full flex-col items-center gap-1 rounded-xl p-3 text-center hover:bg-side disabled:opacity-60";
  if (entry.directory) {
    return (
      <button
        type="button"
        className={tile}
        disabled={entry.locked}
        onClick={() => onOpenFolder(entry.path)}
      >
        {body}
      </button>
    );
  }
  if (entry.clip_id) {
    return (
      <Link to={`/clips/${entry.clip_id}`} className={tile}>
        {body}
      </Link>
    );
  }
  return (
    <button type="button" className={tile} onClick={() => onOpenVideo(entry)}>
      {body}
    </button>
  );
}

/** The state of a video: a label and a symbol, never the colour alone. */
function ExplorerStatePill({
  state,
}: {
  state?: ExplorerState | "in_progress";
}) {
  const { t } = useI18n();
  if (!state) return null;
  const tone: Record<ExplorerState, string> = {
    not_indexed: "border border-dashed border-line-strong text-mute",
    in_progress: "bg-accent-soft text-accent-ink",
    not_analyzed: "bg-side text-mute",
    transcribed: "bg-ok-bg text-ok-fg",
    analyzed: "bg-ok-bg text-ok-fg",
    failed: "bg-bad-bg text-bad-fg",
  };
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ${tone[state]}`}
    >
      {t(`library.explorer.state.${state}`)}
    </span>
  );
}
