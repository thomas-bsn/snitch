import type { components } from "../../api.gen";
import { api, errorCode } from "../api/client";
import type { Result } from "../api/loaded";

export type Clip = components["schemas"]["Clip"];
export type ClipPage = components["schemas"]["ClipPage"];
export type ClipFilterOptions = components["schemas"]["ClipFilterOptions"];
export type ClipVoice = components["schemas"]["ClipVoice"];
export type LocalSpeaker = components["schemas"]["LocalSpeaker"];
export type TranscriptSegment = components["schemas"]["TranscriptSegment"];
export type SegmentHistory = components["schemas"]["SegmentHistory"];
export type Person = components["schemas"]["Person"];

/** What the clip is called everywhere: the administrator's title, else the file name. */
export const clipTitle = (clip: {
  display_title?: string | null;
  original_name: string;
}) => clip.display_title || clip.original_name;

export const CLIP_STATES = [
  "ready",
  "waiting_stability",
  "unreadable",
  "missing",
] as const;
export type ClipState = (typeof CLIP_STATES)[number];

export const PAGE_SIZE = 12;

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

/** The filters of the library, as they live in the address. */
export type LibraryFilters = {
  game: string;
  from: string;
  to: string;
  source: string;
  visibility: string;
  owner: string;
  /** A Person's id, or "unknown" for the voices nobody identified. */
  person: string;
  state: string;
  /** Folder path to filter by (e.g., "League of Legends" or "League of Legends/2024") */
  folder: string;
  page: number;
};

const positive = (value: string) =>
  /^[1-9]\d*$/.test(value) ? Number(value) : undefined;

export async function fetchClips(
  filters: LibraryFilters,
): Promise<Result<ClipPage>> {
  const { data, error } = await api.GET("/api/clips", {
    params: {
      query: {
        game: filters.game || undefined,
        page: filters.page,
        page_size: PAGE_SIZE,
        from: filters.from || undefined,
        to: filters.to || undefined,
        source_id: positive(filters.source),
        visibility:
          filters.visibility === "shared" || filters.visibility === "private"
            ? filters.visibility
            : undefined,
        owner_id: positive(filters.owner),
        status: CLIP_STATES.find((state) => state === filters.state),
        person_id: positive(filters.person),
        person_unknown: filters.person === "unknown" ? true : undefined,
        folder: filters.folder || undefined,
      },
    },
  });
  return data ? { ok: true, value: data } : failure(error);
}

export async function fetchFilterOptions(): Promise<
  Result<{ options: ClipFilterOptions; persons: Person[] }>
> {
  const [options, persons] = await Promise.all([
    api.GET("/api/clip-filters"),
    api.GET("/api/persons"),
  ]);
  if (!options.data) return failure(options.error);
  return {
    ok: true,
    value: { options: options.data, persons: persons.data?.items ?? [] },
  };
}

export async function fetchClip(id: number): Promise<Result<Clip>> {
  const { data, error } = await api.GET("/api/clips/{id}", {
    params: { path: { id } },
  });
  return data ? { ok: true, value: data } : failure(error);
}

export async function fetchClipVoice(id: number): Promise<Result<ClipVoice>> {
  const { data, error } = await api.GET("/api/clips/{id}/voice", {
    params: { path: { id } },
  });
  return data ? { ok: true, value: data } : failure(error);
}

export async function fetchSegmentHistory(
  clipId: number,
  index: number,
): Promise<Result<SegmentHistory>> {
  const { data, error } = await api.GET(
    "/api/clips/{id}/segments/{index}/history",
    { params: { path: { id: clipId, index } } },
  );
  return data ? { ok: true, value: data } : failure(error);
}
