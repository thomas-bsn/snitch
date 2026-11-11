import type { components } from "../../api.gen";
import { api, mutation } from "../api/client";
import { failure, guard } from "../api/loaded";

export type RootScan = components["schemas"]["RootScan"];
export type RootScanPreview = components["schemas"]["RootScanPreview"];

/** What a scan of the whole Racine would queue and leave out, without the paths (#404). */
export const previewRootScan = () =>
  guard<RootScanPreview>(async () => {
    const { data, error } = await api.GET(
      "/api/admin/video-root/scan-runs/preview",
    );
    return data ? { ok: true, value: data } : failure(error);
  });

/** The latest scan of the whole Racine, or null when none was ever asked. */
export const fetchCurrentRootScan = () =>
  guard<RootScan | null>(async () => {
    const { data, error } = await api.GET(
      "/api/admin/video-root/scan-runs/current",
    );
    return data ? { ok: true, value: data.run ?? null } : failure(error);
  });

/** Starts the scan: the server answers at once and walks the Racine in the background. */
export const startRootScan = (csrf: string | undefined) =>
  guard<RootScan | null>(async () => {
    const { data, error } = await api.POST("/api/admin/video-root/scan-runs", {
      params: { header: mutation(csrf) },
    });
    return data ? { ok: true, value: data.run ?? null } : failure(error);
  });

export const cancelRootScan = (id: number, csrf: string | undefined) =>
  guard<RootScan | null>(async () => {
    const { data, error } = await api.POST(
      "/api/admin/video-root/scan-runs/{id}/cancel",
      { params: { path: { id }, header: mutation(csrf) } },
    );
    return data ? { ok: true, value: data.run ?? null } : failure(error);
  });
