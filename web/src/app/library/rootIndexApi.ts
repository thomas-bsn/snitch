import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import type { Result } from "../api/loaded";

export type IndexRun = components["schemas"]["IndexRun"];
export type IndexRunPreview = components["schemas"]["IndexRunPreview"];
export type IndexMode = "cards" | "full";

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

async function guard<T>(call: () => Promise<Result<T>>): Promise<Result<T>> {
  try {
    return await call();
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

/** What an indexation of the whole Racine would do, without starting it. */
export const previewIndexRun = () =>
  guard<IndexRunPreview>(async () => {
    const { data, error } = await api.GET(
      "/api/admin/library/index-runs/preview",
    );
    return data ? { ok: true, value: data } : failure(error);
  });

/** The latest indexation of the Racine, or null when none was ever asked. */
export const fetchCurrentIndexRun = () =>
  guard<IndexRun | null>(async () => {
    const { data, error } = await api.GET(
      "/api/admin/library/index-runs/current",
    );
    return data ? { ok: true, value: data.run ?? null } : failure(error);
  });

export const startIndexRun = (mode: IndexMode, csrf: string | undefined) =>
  guard<IndexRun | null>(async () => {
    const { data, error } = await api.POST("/api/admin/library/index-runs", {
      params: { header: mutation(csrf) },
      body: { mode },
    });
    return data ? { ok: true, value: data.run ?? null } : failure(error);
  });

export type IndexAction = "pause" | "resume" | "cancel";

export const changeIndexRun = (
  id: number,
  action: IndexAction,
  csrf: string | undefined,
) =>
  guard<IndexRun | null>(async () => {
    const path = `/api/admin/library/index-runs/{id}/${action}` as const;
    const { data, error } = await api.POST(path, {
      params: { path: { id }, header: mutation(csrf) },
    });
    return data ? { ok: true, value: data.run ?? null } : failure(error);
  });
