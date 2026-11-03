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

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

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
