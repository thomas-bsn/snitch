import { useCallback, useMemo } from "react";
import { useLocation } from "react-router";
import { useLoaded } from "../api/loaded";
import { useAddressParams } from "../api/useAddressParams";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { fetchFilterOptions } from "../library/libraryApi";
import { LibrarySearchTabs } from "../library/LibrarySearchTabs";
import { SearchForm } from "./SearchForm";
import { SearchResultItem } from "./SearchResultItem";
import {
  PAGE_SIZE,
  SEARCH_MODES,
  fetchSearch,
  type SearchFilters,
} from "./searchApi";

const FILTER_KEYS = [
  "from",
  "to",
  "source",
  "visibility",
  "state",
  "person",
] as const;

function readFilters(params: URLSearchParams): SearchFilters {
  const page = Number(params.get("page"));
  const mode = SEARCH_MODES.find((value) => value === params.get("mode"));
  return {
    q: (params.get("q") ?? "").trim().slice(0, 200),
    mode: mode ?? "full_text",
    from: params.get("from") ?? "",
    to: params.get("to") ?? "",
    source: params.get("source") ?? "",
    visibility: params.get("visibility") ?? "",
    state: params.get("state") ?? "",
    person: params.get("person") ?? "",
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  };
}

/** The search screen: the passages found in the transcripts, each opening its clip. */
export function SearchPage() {
  const { t, error } = useI18n();
  const location = useLocation();
  const { params, update } = useAddressParams();
  useDocumentTitle(t("pages.search.title"));

  const filters = useMemo(() => readFilters(params), [params]);
  const active = FILTER_KEYS.some((key) => filters[key] !== "");
  const choices = useLoaded(fetchFilterOptions);
  const search = useLoaded(
    useCallback(
      () =>
        filters.q === ""
          ? Promise.resolve({
              ok: true as const,
              value: { items: [], page: 1, page_size: PAGE_SIZE, total: 0 },
            })
          : fetchSearch(filters),
      [filters],
    ),
  );

  function change(patch: Partial<SearchFilters>) {
    update((next) => {
      for (const [key, value] of Object.entries(patch)) {
        if (
          value === "" ||
          value === 1 ||
          (key === "mode" && value === "full_text")
        )
          next.delete(key);
        else next.set(key, String(value));
      }
      if (!("page" in patch)) next.delete("page");
    });
  }

  const result =
    filters.q !== "" && search.state.status === "ready"
      ? search.state.value
      : null;
  const pages = result
    ? Math.max(1, Math.ceil(result.total / (result.page_size || PAGE_SIZE)))
    : 1;
  const here = location.pathname + location.search;

  return (
    <section className="flex flex-col gap-5 px-4 py-6 md:px-7">
      <LibrarySearchTabs />
      <h1 className="text-[22px] font-semibold">{t("pages.search.title")}</h1>

      <SearchForm
        filters={filters}
        options={
          choices.state.status === "ready" ? choices.state.value.options : null
        }
        persons={
          choices.state.status === "ready" ? choices.state.value.persons : []
        }
        active={active}
        onSearch={(q) => change({ q })}
        onChange={change}
        onClear={() =>
          update((next) => {
            for (const key of FILTER_KEYS) next.delete(key);
            next.delete("page");
          })
        }
      />

      {filters.q === "" && (
        <p className="max-w-xl text-mute">{t("search.prompt")}</p>
      )}

      {filters.q !== "" && search.state.status === "failed" && (
        <div className="flex flex-col items-start gap-3">
          <ErrorAlert>{error(search.state.code)}</ErrorAlert>
          <button
            type="button"
            className={buttonClass("secondary")}
            onClick={search.reload}
          >
            {t("library.retry")}
          </button>
        </div>
      )}
      {filters.q !== "" && search.state.status === "loading" && (
        <p className="text-mute" role="status">
          {t("search.loading")}
        </p>
      )}

      {result?.degraded && <StatusNotice>{t("search.degraded")}</StatusNotice>}

      {result && (
        <>
          <p className="m-0 text-mute" aria-live="polite">
            {result.total === 1
              ? t("search.countOne")
              : t("search.count", { count: result.total })}
            {" · "}
            {t("search.modeUsed", {
              mode: t(
                `search.modes.${result.degraded ? "full_text" : filters.mode}.name`,
              ),
            })}
          </p>

          {result.items.length === 0 ? (
            <div className="flex flex-col items-center gap-2 rounded-xl border border-dashed border-line-strong px-6 py-12 text-center">
              <h2 className="text-base font-semibold">
                {t("search.none.title")}
              </h2>
              <p className="max-w-md text-mute">{t("search.none.body")}</p>
            </div>
          ) : (
            <ol
              aria-label={t("search.results")}
              className="m-0 list-none rounded-xl border border-line p-0"
            >
              {result.items.map((item, index) => (
                <SearchResultItem
                  key={`${item.clip_id}-${item.timestamp}-${index}`}
                  result={item}
                  from={here}
                />
              ))}
            </ol>
          )}

          {pages > 1 && (
            <nav
              aria-label={t("search.pagination.label")}
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
        </>
      )}
    </section>
  );
}
