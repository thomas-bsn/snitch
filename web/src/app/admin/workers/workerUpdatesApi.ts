import type { paths } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import type { Result } from "../../api/loaded";
import { useSession } from "../../session";

/** Automatic Worker updates (ADR 0036): the target version, the catalogue and each Worker's last report. */
export type WorkerUpdates =
  paths["/api/admin/worker-updates"]["get"]["responses"][200]["content"]["application/json"];

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

export async function fetchWorkerUpdates(): Promise<Result<WorkerUpdates>> {
  const { data, error } = await api.GET("/api/admin/worker-updates");
  return data ? { ok: true, value: data } : failure(error);
}

/** Chooses the target version; null turns automatic updates off. */
export function useWorkerUpdateActions() {
  const csrf = useSession().session?.csrf_token;
  return {
    async setTarget(version: string | null): Promise<Result<null>> {
      const { error } = await api.PUT("/api/admin/worker-updates", {
        params: { header: mutation(csrf) },
        body: { version },
      });
      return error ? failure(error) : { ok: true, value: null };
    },
  };
}
