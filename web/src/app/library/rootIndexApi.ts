import type { components } from "../../api.gen";
import { api, mutation } from "../api/client";
import { failure, guard } from "../api/loaded";

export type IndexRun = components["schemas"]["IndexRun"];
export type IndexRunPreview = components["schemas"]["IndexRunPreview"];
export type IndexMode = "cards" | "full";

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
