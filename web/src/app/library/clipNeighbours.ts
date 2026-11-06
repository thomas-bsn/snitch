import { useCallback, useMemo } from "react";
import { useLoaded } from "../api/loaded";
import type { Result } from "../api/loaded";
import { fetchSearch } from "../search/searchApi";
import {
  asExplorerStage,
  asExplorerState,
  fetchExplorerAll,
  type ExplorerState,
} from "./explorerApi";

/** One clip of the list the fiche was opened from, in the order of that list. */
export type Neighbour = {
  clipId: number;
  title: string;
  /** Search only: the moment of the passage found, where the clip opens. */
  at?: number;
  /** Search only: how many passages of this clip were found. */
  passages?: number;
  excerpt?: string;
  durationSeconds?: number | null;
  recordedAt?: string | null;
  state?: ExplorerState;
};

/**
 * A search returns one result per passage, so the same clip can come four times.
 * The strip keeps one tile per clip, opened at its first passage, and counts them:
 * moving to the next clip must not walk through the same video's passages.
 */
function groupPassages(items: Neighbour[]): Neighbour[] {
  const byClip = new Map<number, Neighbour>();
  for (const item of items) {
    const seen = byClip.get(item.clipId);
    if (!seen) {
      byClip.set(item.clipId, { ...item, passages: 1 });
      continue;
    }
    seen.passages = (seen.passages ?? 1) + 1;
    if (item.at !== undefined && (seen.at === undefined || item.at < seen.at)) {
      seen.at = item.at;
      seen.excerpt = item.excerpt;
    }
  }
  return [...byClip.values()];
}

export type Sequence = {
  kind: "library" | "search";
  /** Where the list is: the folder, or the searched words. */
  label: string;
  items: Neighbour[];
};

function parse(from: string) {
  const [path, query = ""] = from.split("?");
  return { path, params: new URLSearchParams(query) };
}

/**
 * The list behind the origin address, read again with the same folder, filters and order.
 * Only videos that have a fiche count: a video not indexed yet has none to open.
 */
async function readSequence(from: string): Promise<Result<Sequence>> {
  const { path, params } = parse(from);
  if (path === "/search") {
    const page = Number(params.get("page"));
    const q = (params.get("q") ?? "").trim().slice(0, 200);
    const found = await fetchSearch({
      q,
      from: params.get("from") ?? "",
      to: params.get("to") ?? "",
      source: params.get("source") ?? "",
      visibility: params.get("visibility") ?? "",
      state: params.get("state") ?? "",
      person: params.get("person") ?? "",
      page: Number.isInteger(page) && page >= 1 ? page : 1,
    });
    if (!found.ok) return found;
    return {
      ok: true,
      value: {
        kind: "search",
        label: q,
        items: groupPassages(
          found.value.items
            .filter((r) => r.status === "ready")
            .map((r) => ({
              clipId: r.clip_id,
              title: r.clip_name,
              at: Math.floor(r.timestamp),
              excerpt: r.matched_text || r.context,
              recordedAt: r.recorded_at,
            })),
        ),
      },
    };
  }
  const folder = (params.get("folder") ?? "").replace(/^\/+|\/+$/g, "");
  const listing = await fetchExplorerAll({
    path: folder,
    q: params.get("filter") ?? "",
    state: asExplorerState(params.get("state")),
    stage: asExplorerStage(params.get("stage")),
    game: params.get("game") ?? "",
    from: params.get("from") ?? "",
    to: params.get("to") ?? "",
    visibility: params.get("visibility") ?? "",
    owner_id: params.get("owner") ?? "",
    person_id: params.get("person") ?? "",
    voices: params.get("voices") === "pending" ? "pending" : "",
  });
  if (!listing.ok) return listing;
  return {
    ok: true,
    value: {
      kind: "library",
      label: folder,
      items: listing.value.items.flatMap((entry) =>
        !entry.directory && entry.clip_id
          ? [
              {
                clipId: entry.clip_id,
                title: entry.name,
                durationSeconds: entry.duration_seconds,
                recordedAt: entry.recorded_at,
                state: entry.state,
              },
            ]
          : [],
      ),
    },
  };
}

/**
 * The clips around this one, read once per origin: moving from clip to clip keeps the
 * same origin, so the list is not read again at every step.
 */
const cache = new Map<string, Promise<Result<Sequence>>>();

export function useSequence(from: string | null) {
  const load = useCallback(() => {
    if (!from) {
      return Promise.resolve({ ok: false as const, code: "no_origin" });
    }
    let pending = cache.get(from);
    if (!pending) {
      pending = readSequence(from);
      cache.set(from, pending);
      // A failed read is not kept: the next fiche tries again.
      void pending.then((r) => {
        if (!r.ok) cache.delete(from);
      });
    }
    return pending;
  }, [from]);
  const loaded = useLoaded(load);
  return useMemo(
    () => (loaded.state.status === "ready" ? loaded.state.value : null),
    [loaded.state],
  );
}

const LAST_SEEN_KEY = "snitch.library.lastSeen";

/** The last clip seen from a list, so that coming back lands on it and not on the first one opened. */
export function rememberLastSeen(from: string, clipId: number) {
  try {
    const all = JSON.parse(
      window.sessionStorage.getItem(LAST_SEEN_KEY) ?? "{}",
    ) as Record<string, number>;
    all[from] = clipId;
    window.sessionStorage.setItem(LAST_SEEN_KEY, JSON.stringify(all));
  } catch {
    // Without storage, the list simply comes back where it was left.
  }
}

export function takeLastSeen(from: string): number | null {
  try {
    const all = JSON.parse(
      window.sessionStorage.getItem(LAST_SEEN_KEY) ?? "{}",
    ) as Record<string, number>;
    const value = all[from];
    delete all[from];
    window.sessionStorage.setItem(LAST_SEEN_KEY, JSON.stringify(all));
    return typeof value === "number" ? value : null;
  } catch {
    return null;
  }
}
