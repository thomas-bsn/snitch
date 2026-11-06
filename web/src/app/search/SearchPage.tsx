import { Check, Search as SearchIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useLocation } from "react-router";
import { useLoaded } from "../api/loaded";
import { useAddressParams } from "../api/useAddressParams";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { fetchFilterOptions } from "../library/libraryApi";
import { SearchClipGroup, groupResults } from "./SearchClipGroup";
import { SearchForm } from "./SearchForm";
import { PAGE_SIZE, fetchSearch, type SearchFilters } from "./searchApi";

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
  return {
    q: (params.get("q") ?? "").trim().slice(0, 200),
    from: params.get("from") ?? "",
    to: params.get("to") ?? "",
    source: params.get("source") ?? "",
    visibility: params.get("visibility") ?? "",
    state: params.get("state") ?? "",
    person: params.get("person") ?? "",
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  };
}

/** The search screen: the passages found, grouped by clip, with the voice that speaks. */
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
        if (value === "" || value === 1) next.delete(key);
        else next.set(key, String(value));
      }
      if (!("page" in patch)) next.delete("page");
    });
  }

  const result =
    filters.q !== "" && search.state.status === "ready"
      ? search.state.value
      : null;
  const groups = useMemo(() => groupResults(result?.items ?? []), [result]);
  const pages = result
    ? Math.max(1, Math.ceil(result.total / (result.page_size || PAGE_SIZE)))
    : 1;
  const here = location.pathname + location.search;

  const passages =
    result && result.total > 0
      ? result.total === 1
        ? t("search.passageOne")
        : t("search.passages", { count: result.total })
      : "";
  const clips =
    result && result.total > 0
      ? groups.length === 1
        ? pages > 1
          ? t("search.clipPageOne")
          : t("search.clipOne")
        : pages > 1
          ? t("search.clipPage", { count: groups.length })
          : t("search.clips", { count: groups.length })
      : "";

  return (
    <section className="flex flex-col gap-5 px-4 py-6 md:px-7">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div className="min-w-0">
          <h1 className="text-[22px] font-semibold">
            {t("pages.search.title")}
          </h1>
          {filters.q !== "" && result && result.total > 0 && (
            <p className="mt-1.5 text-mute" aria-live="polite">
              {t("search.summary", {
                passages,
                clips,
                query: filters.q,
              })}
            </p>
          )}
          {filters.q !== "" && result && result.total === 0 && (
            <p className="mt-1.5 text-mute">
              {t("search.none.lead", { query: filters.q })}
            </p>
          )}
        </div>
        <span className="inline-flex w-fit shrink-0 items-center gap-1.5 rounded-full bg-accent-soft px-2.5 py-1 text-xs font-medium text-accent-ink">
          <Check className="size-3.5" aria-hidden="true" />
          {t("search.exactNote")}
        </span>
      </div>

      {filters.q === "" ? (
        <div className="flex max-w-2xl flex-col items-start gap-3 rounded-xl border border-dashed border-line-strong p-6">
          <span className="grid size-11 place-items-center rounded-xl bg-side text-mute">
            <SearchIcon className="size-5" aria-hidden="true" />
          </span>
          <div>
            <h2 className="text-base font-semibold">
              {t("search.prompt.title")}
            </h2>
            <p className="mt-1 text-mute">{t("search.prompt.body")}</p>
          </div>
        </div>
      ) : (
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_288px] lg:items-start">
          <div className="flex min-w-0 flex-col gap-4">
            {search.state.status === "failed" && (
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
            {search.state.status === "loading" && (
              <p className="text-mute" role="status">
                {t("search.loading")}
              </p>
            )}

            {result && result.items.length === 0 && (
              <div className="flex flex-col items-center gap-2 rounded-xl border border-dashed border-line-strong px-6 py-12 text-center">
                <span className="grid size-11 place-items-center rounded-xl bg-side text-mute">
                  <SearchIcon className="size-5" aria-hidden="true" />
                </span>
                <h2 className="text-base font-semibold">
                  {t("search.none.title")}
                </h2>
                <p className="max-w-md text-mute">{t("search.none.body")}</p>
                <button
                  type="button"
                  className={buttonClass("secondary")}
                  onClick={() => change({ q: "" })}
                >
                  {t("search.none.clear")}
                </button>
              </div>
            )}

            {result && result.items.length > 0 && (
              <>
                <p className="m-0 text-[10.5px] font-semibold uppercase tracking-[0.08em] text-mute">
                  {t("search.groupedByClip")}
                </p>
                <div className="flex flex-col gap-3.5">
                  {groups.map((group) => (
                    <SearchClipGroup
                      key={group.clipId}
                      group={group}
                      from={here}
                    />
                  ))}
                </div>

                {pages > 1 && (
                  <nav
                    aria-label={t("search.pagination.label")}
                    className="flex flex-wrap items-center justify-center gap-3"
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
          </div>

          <SearchForm
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
            active={active}
            onChange={change}
            onClear={() =>
              update((next) => {
                for (const key of FILTER_KEYS) next.delete(key);
                next.delete("page");
              })
            }
          />
        </div>
      )}
    </section>
  );
}
