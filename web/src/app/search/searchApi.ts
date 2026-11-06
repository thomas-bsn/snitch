import type { components } from "../../api.gen";
import { api, errorCode } from "../api/client";
import type { Result } from "../api/loaded";

export type SearchPage = components["schemas"]["SearchPage"];
export type SearchResult = components["schemas"]["SearchResult"];

export const PAGE_SIZE = 20;

/** What is searched, as it lives in the address. Search is always exact. */
export type SearchFilters = {
  q: string;
  from: string;
  to: string;
  source: string;
  visibility: string;
  state: string;
  /** A Person's id, or "unknown" for the voices nobody identified. */
  person: string;
  page: number;
};

const positive = (value: string) =>
  /^[1-9]\d*$/.test(value) ? Number(value) : undefined;

const STATES = ["ready", "waiting_stability", "unreadable", "missing"] as const;

export async function fetchSearch(
  filters: SearchFilters,
): Promise<Result<SearchPage>> {
  const { data, error } = await api.GET("/api/search", {
    params: {
      query: {
        q: filters.q,
        mode: "exact",
        page: filters.page,
        page_size: PAGE_SIZE,
        from: filters.from || undefined,
        to: filters.to || undefined,
        source_id: positive(filters.source),
        visibility:
          filters.visibility === "shared" || filters.visibility === "private"
            ? filters.visibility
            : undefined,
        status: STATES.find((state) => state === filters.state),
        person_id: positive(filters.person),
        person_unknown: filters.person === "unknown" ? true : undefined,
      },
    },
  });
  return data
    ? { ok: true, value: data }
    : { ok: false, code: errorCode(error) ?? "unknown" };
}
