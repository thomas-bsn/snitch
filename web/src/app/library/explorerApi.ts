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
};

export type ExplorerQuery = {
  path: string;
  recursive?: boolean;
  q?: string;
  state?: string;
  game?: string;
  from?: string;
  to?: string;
  visibility?: string;
  owner_id?: string;
};

export type ExplorerListing = {
  sourceId: number;
  /** False while no Racine vidéo is chosen: only the clips with no folder are listed. */
  rootConfigured: boolean;
  items: ExplorerEntry[];
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
    "game",
    "from",
    "to",
    "visibility",
    "owner_id",
  ] as const) {
    const value = query[key];
    if (value) params.set(key, value);
  }
  const response = await fetch(`/api/library/explorer?${params}`);
  const body = await response.json().catch(() => undefined);
  if (!response.ok || !body) return failure(await errorOf(response));
  return {
    ok: true,
    value: {
      sourceId: body.source_id,
      rootConfigured: body.root_configured !== false,
      items: body.items ?? [],
    },
  };
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
