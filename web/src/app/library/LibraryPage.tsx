import { Link, useLocation, useSearchParams } from "react-router";
import { useCallback, useMemo } from "react";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { ClipCard } from "./ClipCard";
import { LibraryFilters } from "./LibraryFilters";
import {
  PAGE_SIZE,
  fetchClips,
  fetchFilterOptions,
  type LibraryFilters as Filters,
} from "./libraryApi";

const KEYS = ["from", "to", "source", "owner", "person", "state"] as const;

function readFilters(params: URLSearchParams): Filters {
  const page = Number(params.get("page"));
  return {
    from: params.get("from") ?? "",
    to: params.get("to") ?? "",
    source: params.get("source") ?? "",
    owner: params.get("owner") ?? "",
    person: params.get("person") ?? "",
    state: params.get("state") ?? "",
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  };
}

/** The library: the clips in a grid, filtered by what the address says, a page at a time. */
export function LibraryPage() {
  const { t, error } = useI18n();
  const location = useLocation();
  const [params, setParams] = useSearchParams();
  useDocumentTitle(t("pages.library.title"));

  const filters = useMemo(() => readFilters(params), [params]);
  const active = KEYS.some((key) => filters[key] !== "");

  const clips = useLoaded(useCallback(() => fetchClips(filters), [filters]));
  const choices = useLoaded(fetchFilterOptions);

  function change(patch: Partial<Filters>) {
    const next = new URLSearchParams(params);
    for (const [key, value] of Object.entries(patch)) {
      if (value === "" || value === 1) next.delete(key);
      else next.set(key, String(value));
    }
    if (!("page" in patch)) next.delete("page");
    setParams(next);
  }

  const page = clips.state.status === "ready" ? clips.state.value : null;
  const pages = page ? Math.max(1, Math.ceil(page.total / PAGE_SIZE)) : 1;
  const from = location.pathname + location.search;

  return (
    <section className="flex flex-col gap-5 px-4 py-6 md:px-7">
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
        onClear={() => setParams(new URLSearchParams())}
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
            <ClipCard key={clip.id} clip={clip} from={from} />
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
