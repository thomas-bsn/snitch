import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import type { Result } from "../../api/loaded";
import { useSession } from "../../session";

export type WorkerPage = components["schemas"]["WorkerPage"];
export type Worker = components["schemas"]["Worker"];
export type WorkerDetail = components["schemas"]["WorkerDetail"];
export type WorkerRegistration = components["schemas"]["WorkerRegistration"];
export type WorkerBundle = components["schemas"]["WorkerBundle"];

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

export async function fetchWorkers(): Promise<Result<WorkerPage>> {
  const { data, error } = await api.GET("/api/admin/workers");
  return data ? { ok: true, value: data } : failure(error);
}

export async function fetchWorker(id: string): Promise<Result<WorkerDetail>> {
  const { data, error } = await api.GET("/api/admin/workers/{id}", {
    params: { path: { id } },
  });
  return data ? { ok: true, value: data } : failure(error);
}

export type NewWorker = {
  name: string;
};

/** The actions of the Workers screens; each one carries the session's CSRF token. */
export function useWorkerActions() {
  const csrf = useSession().session?.csrf_token;
  return {
    async create(input: NewWorker): Promise<Result<WorkerRegistration>> {
      // Never put the Hugging Face token here: it stays in the browser.
      const { data, error } = await api.POST("/api/admin/workers", {
        params: { header: mutation(csrf) },
        body: {
          name: input.name,
          max_concurrency: 1,
        },
      });
      return data ? { ok: true, value: data } : failure(error);
    },
    async renewToken(id: string): Promise<Result<WorkerRegistration>> {
      const { data, error } = await api.POST("/api/admin/workers/{id}/token", {
        params: { path: { id }, header: mutation(csrf) },
      });
      return data ? { ok: true, value: data } : failure(error);
    },
    async revoke(id: string, reason: string): Promise<Result<null>> {
      const { error } = await api.DELETE("/api/admin/workers/{id}", {
        params: { path: { id }, header: mutation(csrf) },
        body: reason.trim() ? { reason: reason.trim() } : undefined,
      });
      return error ? failure(error) : { ok: true, value: null };
    },
  };
}
