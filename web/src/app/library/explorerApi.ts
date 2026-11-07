import type { Result } from "../api/loaded";
import { errorCode } from "../api/client";

/** The states a video of the explorer can carry (ADR 0032). */
export const EXPLORER_STATES = [
  "not_indexed",
  "in_progress",
  "not_analyzed",
  "transcribed",
  "analyzed",
  "failed",
] as const;
export type ExplorerState = (typeof EXPLORER_STATES)[number];

/** The processing stages a filter can ask for (#282); each reads the real steps of the analysis. */
export const EXPLORER_STAGES = [
  "indexed",
  "transcribed",
  "voices",
  "complete",
] as const;
export type ExplorerStage = (typeof EXPLORER_STAGES)[number];

/** A stage from the address bar: anything else is ignored, never sent to the server. */
export function asExplorerStage(value: string | null): ExplorerStage | "" {
  return EXPLORER_STAGES.find((stage) => stage === value) ?? "";
}

/** A state from the address bar: anything else is ignored, never sent to the server. */
export function asExplorerState(value: string | null): ExplorerState | "" {
  return EXPLORER_STATES.find((state) => state === value) ?? "";
}

/** A folder or a video of the Racine vidéo, as the explorer lists it. */
export type ExplorerEntry = {
  name: string;
  /** Path relative to the Racine vidéo. */
  path: string;
  /** Parent folder of a video, relative to the Racine vidéo ("" at the root). */
  folder: string;
  directory: boolean;
  size_bytes: number;
  /** Top-level folders only: made private (ADR 0026). */
  private: boolean;
  /** Private and not openable by the viewer: only the name is known. */
  locked: boolean;
  state?: ExplorerState;
  clip_id: number | null;
  owner_id: number | null;
  owner: string;
  visibility?: string;
  duration_seconds: number | null;
  added_at: string | null;
  recorded_at: string | null;
  game: string;
  voices_pending: number;
  /** Folders of a filtered tree: the videos under it that match. */
  match_count?: number;
  /** Active custom logo of a folder: the glyph requests that version. */
  logo_version?: number;
};

export type ExplorerQuery = {
  path: string;
  recursive?: boolean;
  q?: string;
  state?: string;
  stage?: string;
  game?: string;
  from?: string;
  to?: string;
  visibility?: string;
  owner_id?: string;
  /** Videos with a voice attributed to this Person. */
  person_id?: string;
  /** "pending": videos that still have voices to process. */
  voices?: string;
  limit?: number;
  offset?: number;
};

export type ExplorerListing = {
  sourceId: number;
  /** False while no Racine vidéo is chosen: only the clips with no folder are listed. */
  rootConfigured: boolean;
  items: ExplorerEntry[];
  /** Size of the whole selection, beyond the page. */
  total: number;
};

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

async function errorOf(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return undefined;
  }
}

export async function fetchExplorer(
  query: ExplorerQuery,
): Promise<Result<ExplorerListing>> {
  const params = new URLSearchParams({ path: query.path });
  if (query.recursive) params.set("recursive", "1");
  for (const key of [
    "q",
    "state",
    "stage",
    "game",
    "from",
    "to",
    "visibility",
    "owner_id",
    "person_id",
    "voices",
  ] as const) {
    const value = query[key];
    if (value) params.set(key, value);
  }
  if (query.limit) params.set("limit", String(query.limit));
  if (query.offset) params.set("offset", String(query.offset));
  const response = await fetch(`/api/library/explorer?${params}`);
  const body = await response.json().catch(() => undefined);
  if (!response.ok || !body) return failure(await errorOf(response));
  return {
    ok: true,
    value: {
      sourceId: body.source_id,
      rootConfigured: body.root_configured !== false,
      items: body.items ?? [],
      total: body.total ?? (body.items ?? []).length,
    },
  };
}

/** Largest page the server answers: keep equal to explorerMaxLimit in app/internal/httpapi/explorer.go
 * (and to the maximum of the limit parameter in contracts/openapi.yaml). */
export const EXPLORER_MAX_LIMIT = 500;

/** Every item of a selection, page after page. */
export async function fetchExplorerAll(
  query: ExplorerQuery,
): Promise<Result<ExplorerListing>> {
  const limit = EXPLORER_MAX_LIMIT;
  let offset = 0;
  const items: ExplorerEntry[] = [];
  for (;;) {
    const page = await fetchExplorer({ ...query, limit, offset });
    if (!page.ok) return page;
    items.push(...page.value.items);
    offset += limit;
    if (offset >= page.value.total || page.value.items.length === 0) {
      return { ok: true, value: { ...page.value, items, total: items.length } };
    }
  }
}

/** Asks for the indexing of one video (no analysis). Answers with its clip when it already has one. */
export async function requestIndex(
  path: string,
  csrf: string,
): Promise<Result<{ state: string; clip_id?: number }>> {
  const response = await fetch("/api/library/explorer/index", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: JSON.stringify({ path }),
  });
  const body = await response.json().catch(() => undefined);
  if (!response.ok || !body) return failure(await errorOf(response));
  return { ok: true, value: body };
}

/** The file name, the last segment of a relative path. */
export const baseName = (path: string) => path.split("/").pop() ?? path;

/** A person who owns at least one readable clip of an entry. */
export type ExplorerOwner = { id: number; name: string; videos: number };

/** What an explorer entry carries beyond its name: size, durations, owners and dates. */
export type ExplorerProperties = {
  path: string;
  name: string;
  directory: boolean;
  size_bytes: number;
  video_count: number;
  folder_count: number;
  indexed_count: number;
  duration_seconds: number | null;
  added_at: string | null;
  recorded_at: string | null;
  game: string;
  state?: ExplorerState;
  owners: ExplorerOwner[];
};

export async function fetchExplorerProperties(
  path: string,
): Promise<Result<ExplorerProperties>> {
  const response = await fetch(
    `/api/library/explorer/properties?path=${encodeURIComponent(path)}`,
  );
  const body = await response.json().catch(() => undefined);
  if (!response.ok || !body) return failure(await errorOf(response));
  return { ok: true, value: body as ExplorerProperties };
}
